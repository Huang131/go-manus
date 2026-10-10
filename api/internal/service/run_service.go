package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/settings"
	"github.com/google/uuid"
)

// runStore 是 RunService 需要的最小持久化边界。
// 具体 Repository 接口可以包含更多查询能力，但服务只依赖这些状态和消息操作。
type runStore interface {
	CreateWithInitialMessage(ctx context.Context, run *model.Run, initial *model.RunMessage) (*model.Run, bool, error)
	GetByID(ctx context.Context, id string) (*model.Run, error)
	GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error)
	ListBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*model.Run, int, error)
	ListMessages(ctx context.Context, runID string) ([]*model.RunMessage, error)
	TransitionStatus(ctx context.Context, id string, from []model.RunStatus, to model.RunStatus) (bool, error)
	EnterWaitingInput(ctx context.Context, runID string, expectedRevision int, snapshot model.RunExecutionSnapshot, question *model.RunMessage) (bool, error)
	ResumeWaitingInput(ctx context.Context, runID string, expectedRevision int, answer *model.RunMessage) (bool, error)
	FinishTerminal(ctx context.Context, runID string, terminal repository.TerminalTransition, final *model.RunMessage) (bool, error)
}

// RunService 是 Run 聚合的唯一业务入口。
//
// 3C 只负责持久化用例和状态约束，不启动 Engine、不管理进程内 cancel 句柄，也不发布事件。
type RunService interface {
	Create(ctx context.Context, input CreateRunInput) (*model.Run, error)
	Get(ctx context.Context, id string) (*model.Run, error)
	GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error)
	ListBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*model.Run, int, error)
	ListMessages(ctx context.Context, runID string) ([]*model.RunMessage, error)
	Start(ctx context.Context, id string) (*model.Run, error)
	EnterWaitingInput(ctx context.Context, runID string, snapshot model.RunExecutionSnapshot, question *model.RunMessage) (*model.Run, error)
	SubmitInput(ctx context.Context, runID string, input SubmitInputRequest) (*RunResume, error)
	RequestCancel(ctx context.Context, id string) (*model.Run, error)
	ReconcileCancelling(ctx context.Context, id string) (*model.Run, error)
	SaveWaitingInput(ctx context.Context, runID string, snapshot model.RunExecutionSnapshot, question *model.RunMessage) error
	Finish(ctx context.Context, runID string, result RunExecutionResult) error
}

// CreateRunInput 是创建一次顶层用户请求所需的冻结输入。
type CreateRunInput struct {
	SessionID        string
	IdempotencyKey   string
	SettingsSnapshot settings.AgentSettings
	PromptHash       string
	Content          string
	Attachments      []string
}

// SubmitInputRequest 是 waiting_input Run 的一次用户恢复输入。
type SubmitInputRequest struct {
	IdempotencyKey   string
	ReplyToMessageID string
	Content          string
	Attachments      []string
}

// RunResume 包含恢复后的 Run 和确定性的 Engine 上下文。
type RunResume struct {
	Run            *model.Run
	Messages       []llmcore.Message
	StartExecution bool
	AlreadyApplied bool
}

type defaultRunService struct {
	store runStore
}

// NewRunService 创建纯 Run 用例服务。
func NewRunService(store runStore) RunService {
	return &defaultRunService{store: store}
}

// Create 原子创建 Run 和首条用户消息；幂等重试由 Repository 返回原聚合。
func (s *defaultRunService) Create(ctx context.Context, input CreateRunInput) (*model.Run, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	run := &model.Run{
		ID:               uuid.NewString(),
		SessionID:        input.SessionID,
		Status:           model.RunStatusPending,
		IdempotencyKey:   input.IdempotencyKey,
		SettingsSnapshot: input.SettingsSnapshot,
		PromptHash:       input.PromptHash,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	initial := &model.RunMessage{
		ID:             uuid.NewString(),
		SessionID:      run.SessionID,
		RunID:          run.ID,
		IdempotencyKey: run.IdempotencyKey,
		Role:           model.RoleUser,
		Content:        input.Content,
		Attachments:    append([]string(nil), input.Attachments...),
		CreatedAt:      now,
	}
	created, inserted, err := s.store.CreateWithInitialMessage(ctx, run, initial)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if !inserted {
		messages, err := s.store.ListMessages(ctx, created.ID)
		if err != nil {
			return nil, normalizeRunError(err)
		}
		storedInput := findMessageByIdempotency(messages, input.IdempotencyKey)
		if storedInput == nil || !sameInitialInput(storedInput, created, input) {
			return nil, apperr.Conflict("idempotency key is already bound to different input")
		}
	}
	return created, nil
}

func sameInitialInput(message *model.RunMessage, run *model.Run, input CreateRunInput) bool {
	return message.RunID == run.ID && message.SessionID == run.SessionID && message.Role == model.RoleUser &&
		message.ReplyToMessageID == "" && message.Content == input.Content && sameStrings(message.Attachments, input.Attachments)
}

func validateCreateInput(input CreateRunInput) error {
	if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.IdempotencyKey) == "" {
		return apperr.BadRequest("session id and idempotency key are required")
	}
	if strings.TrimSpace(input.PromptHash) == "" {
		return apperr.BadRequest("prompt hash is required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return apperr.BadRequest("content is required")
	}
	if err := input.SettingsSnapshot.Validate(); err != nil {
		return apperr.BadRequest(fmt.Sprintf("invalid settings snapshot: %v", err))
	}
	return nil
}

// Get 获取 Run；不存在统一返回 404 业务错误。
func (s *defaultRunService) Get(ctx context.Context, id string) (*model.Run, error) {
	if strings.TrimSpace(id) == "" {
		return nil, apperr.BadRequest("run id is required")
	}
	run, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if run == nil {
		return nil, apperr.NotFound("run not found")
	}
	return run, nil
}

// GetActiveBySessionID 查询会话当前活跃 Run，不把旧 Session 状态字段当作执行事实。
func (s *defaultRunService) GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, apperr.BadRequest("session id is required")
	}
	run, err := s.store.GetActiveBySessionID(ctx, sessionID)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	return run, nil
}

// ListBySessionID 读取会话历史 Run；分页边界由 Repository 统一执行。
func (s *defaultRunService) ListBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*model.Run, int, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, 0, apperr.BadRequest("session id is required")
	}
	runs, total, err := s.store.ListBySessionID(ctx, sessionID, limit, offset)
	if err != nil {
		return nil, 0, normalizeRunError(err)
	}
	return runs, total, nil
}

// ListMessages 暴露 Run 聚合的持久化消息读取，供应用层组装 UI 历史视图。
func (s *defaultRunService) ListMessages(ctx context.Context, runID string) ([]*model.RunMessage, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, apperr.BadRequest("run id is required")
	}
	messages, err := s.store.ListMessages(ctx, runID)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	return messages, nil
}

// Start 只允许 pending Run 进入 running。
func (s *defaultRunService) Start(ctx context.Context, id string) (*model.Run, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	changed, err := s.store.TransitionStatus(ctx, id, []model.RunStatus{model.RunStatusPending}, model.RunStatusRunning)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if !changed {
		return nil, s.statusConflict(ctx, id, "run cannot start")
	}
	return s.Get(ctx, id)
}

// EnterWaitingInput 将 running Run 原子保存为可恢复等待状态。
func (s *defaultRunService) EnterWaitingInput(ctx context.Context, runID string, snapshot model.RunExecutionSnapshot, question *model.RunMessage) (*model.Run, error) {
	if err := snapshot.ValidateWaitingInput(); err != nil {
		return nil, apperr.FailedPrecondition(fmt.Sprintf("invalid waiting input snapshot: %v", err))
	}
	run, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != model.RunStatusRunning {
		return nil, runStatusConflict(run, "run is not running")
	}
	if snapshot.SnapshotRevision != run.SnapshotRevision+1 {
		return nil, apperr.Conflict("waiting input snapshot revision is stale")
	}
	if question == nil || question.ID != snapshot.WaitingCheckpoint.QuestionMessageID ||
		question.RunID != run.ID || question.SessionID != run.SessionID || question.Role != model.RoleAssistant {
		return nil, apperr.BadRequest("waiting question must belong to this run and match the snapshot")
	}
	changed, err := s.store.EnterWaitingInput(ctx, runID, run.SnapshotRevision, snapshot, question)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if !changed {
		return nil, s.statusConflict(ctx, runID, "waiting input checkpoint is stale")
	}
	return s.Get(ctx, runID)
}

// SaveWaitingInput 适配 RunExecutor，并保留领域服务对等待快照的完整校验。
func (s *defaultRunService) SaveWaitingInput(ctx context.Context, runID string, snapshot model.RunExecutionSnapshot, question *model.RunMessage) error {
	_, err := s.EnterWaitingInput(ctx, runID, snapshot, question)
	return err
}

// Finish 原子持久化终态、执行快照和可选的最终助手消息。
func (s *defaultRunService) Finish(ctx context.Context, runID string, result RunExecutionResult) error {
	run, err := s.Get(ctx, runID)
	if err != nil {
		return err
	}
	var terminal repository.TerminalTransition
	switch result.Kind {
	case RunExecutionSucceeded:
		terminal = repository.TerminalTransition{Status: model.RunStatusSucceeded, Snapshot: result.Snapshot}
	case RunExecutionFailed:
		terminal = repository.TerminalTransition{Status: model.RunStatusFailed, Snapshot: result.Snapshot, ErrorCode: "execution_failed", ErrorMessage: errorMessage(result.Error)}
	case RunExecutionCancelled:
		terminal = repository.TerminalTransition{Status: model.RunStatusCancelled, Snapshot: result.Snapshot, ErrorCode: "cancelled", ErrorMessage: errorMessage(result.Error)}
	default:
		return apperr.BadRequest(fmt.Sprintf("unsupported terminal execution result %q", result.Kind))
	}
	if !run.Status.CanTransitionTo(terminal.Status) {
		return runStatusConflict(run, "run cannot finish")
	}

	final := result.Message
	if final == nil && result.Text != "" {
		final = &model.RunMessage{
			ID: uuid.NewString(), SessionID: run.SessionID, RunID: run.ID,
			Role: model.RoleAssistant, Content: result.Text, CreatedAt: time.Now().UTC(),
		}
	}
	if final != nil {
		final = cloneRunMessage(final)
		if final.ID == "" {
			final.ID = uuid.NewString()
		}
		if final.SessionID == "" {
			final.SessionID = run.SessionID
		}
		if final.RunID == "" {
			final.RunID = run.ID
		}
		if final.Role == "" {
			final.Role = model.RoleAssistant
		}
		if final.CreatedAt.IsZero() {
			final.CreatedAt = time.Now().UTC()
		}
		if final.SessionID != run.SessionID || final.RunID != run.ID || final.Role != model.RoleAssistant {
			return apperr.BadRequest("final message must be an assistant message for this run")
		}
	}
	changed, err := s.store.FinishTerminal(ctx, runID, terminal, final)
	if err != nil {
		return normalizeRunError(err)
	}
	if !changed {
		return s.statusConflict(ctx, runID, "run terminal transition is stale")
	}
	return nil
}

func cloneRunMessage(message *model.RunMessage) *model.RunMessage {
	if message == nil {
		return nil
	}
	cloned := *message
	cloned.Attachments = append([]string(nil), message.Attachments...)
	return &cloned
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// SubmitInput 幂等提交回答，并返回重建后的稳定上下文。
func (s *defaultRunService) SubmitInput(ctx context.Context, runID string, input SubmitInputRequest) (*RunResume, error) {
	if strings.TrimSpace(input.IdempotencyKey) == "" || strings.TrimSpace(input.ReplyToMessageID) == "" {
		return nil, apperr.BadRequest("idempotency key and reply message id are required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return nil, apperr.BadRequest("content is required")
	}
	run, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	messages, err := s.store.ListMessages(ctx, runID)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if existing := findMessageByIdempotency(messages, input.IdempotencyKey); existing != nil {
		if !sameInputMessage(existing, run, input) {
			return nil, apperr.Conflict("idempotency key is already bound to different input")
		}
		return &RunResume{Run: run, AlreadyApplied: true}, nil
	}
	if run.Status != model.RunStatusWaitingInput {
		return nil, runStatusConflict(run, "run is not waiting for input")
	}
	if run.WaitingMessageID != input.ReplyToMessageID {
		return nil, apperr.Conflict("reply does not match the current waiting question")
	}
	initial, question, err := findResumeMessages(messages, run.WaitingMessageID)
	if err != nil {
		return nil, apperr.FailedPrecondition(err.Error())
	}
	answer := &model.RunMessage{
		ID:               uuid.NewString(),
		SessionID:        run.SessionID,
		RunID:            run.ID,
		IdempotencyKey:   input.IdempotencyKey,
		ReplyToMessageID: input.ReplyToMessageID,
		Role:             model.RoleUser,
		Content:          input.Content,
		Attachments:      append([]string(nil), input.Attachments...),
		CreatedAt:        time.Now().UTC(),
	}
	contextMessages, err := BuildResumeMessages(*initial, run.ExecutionSnapshot, *question, *answer)
	if err != nil {
		return nil, apperr.FailedPrecondition(fmt.Sprintf("cannot rebuild run context: %v", err))
	}
	changed, err := s.store.ResumeWaitingInput(ctx, runID, run.SnapshotRevision, answer)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if !changed {
		return nil, s.statusConflict(ctx, runID, "waiting input revision is stale")
	}
	resumed, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	return &RunResume{Run: resumed, Messages: contextMessages, StartExecution: true}, nil
}

func findMessageByIdempotency(messages []*model.RunMessage, key string) *model.RunMessage {
	for _, message := range messages {
		if message != nil && message.IdempotencyKey == key {
			return message
		}
	}
	return nil
}

func sameInputMessage(existing *model.RunMessage, run *model.Run, input SubmitInputRequest) bool {
	return existing.RunID == run.ID && existing.SessionID == run.SessionID &&
		existing.Role == model.RoleUser && existing.ReplyToMessageID == input.ReplyToMessageID &&
		existing.Content == input.Content && sameStrings(existing.Attachments, input.Attachments)
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func findResumeMessages(messages []*model.RunMessage, questionID string) (*model.RunMessage, *model.RunMessage, error) {
	var initial, question *model.RunMessage
	for _, message := range messages {
		if message == nil {
			continue
		}
		if initial == nil && message.Role == model.RoleUser && message.ReplyToMessageID == "" {
			initial = message
		}
		if message.ID == questionID {
			question = message
		}
	}
	if initial == nil || question == nil || question.Role != model.RoleAssistant {
		return nil, nil, fmt.Errorf("waiting input messages are incomplete")
	}
	return initial, question, nil
}

// RequestCancel 将可取消状态先迁移为 cancelling；4A 才负责通知进程内执行句柄。
func (s *defaultRunService) RequestCancel(ctx context.Context, id string) (*model.Run, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	changed, err := s.store.TransitionStatus(ctx, id, []model.RunStatus{
		model.RunStatusPending, model.RunStatusRunning, model.RunStatusWaitingInput,
	}, model.RunStatusCancelling)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if changed {
		return s.Get(ctx, id)
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status == model.RunStatusCancelling || current.Status == model.RunStatusCancelled {
		return current, nil
	}
	return nil, runStatusConflict(current, "run cannot be cancelled")
}

// ReconcileCancelling 在没有执行句柄或进程重启后幂等收敛取消状态。
func (s *defaultRunService) ReconcileCancelling(ctx context.Context, id string) (*model.Run, error) {
	run, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run.Status == model.RunStatusCancelled {
		return run, nil
	}
	if run.Status != model.RunStatusCancelling {
		return nil, runStatusConflict(run, "run is not cancelling")
	}
	changed, err := s.store.TransitionStatus(ctx, id, []model.RunStatus{model.RunStatusCancelling}, model.RunStatusCancelled)
	if err != nil {
		return nil, normalizeRunError(err)
	}
	if !changed {
		current, getErr := s.Get(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		if current.Status == model.RunStatusCancelled {
			return current, nil
		}
		return nil, runStatusConflict(current, "cancelling reconciliation lost the state race")
	}
	return s.Get(ctx, id)
}

func (s *defaultRunService) statusConflict(ctx context.Context, id, message string) error {
	run, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	return runStatusConflict(run, message)
}

func runStatusConflict(run *model.Run, message string) *apperr.Error {
	return apperr.Conflict(fmt.Sprintf("%s: status=%s", message, run.Status))
}

func normalizeRunError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrActiveRunExists) {
		return apperr.Conflict("session already has an active run")
	}
	var appError *apperr.Error
	if errors.As(err, &appError) {
		return err
	}
	return apperr.ToInternal(err)
}
