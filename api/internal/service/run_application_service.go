package service

import (
	"context"
	"strings"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/settings"
)

// RunApplicationService 是 HTTP 与 Run 领域之间的应用层边界。
// 它负责组装一次请求所需的快照和外部上下文，不让 Handler 直接依赖仓储或对象存储。
type RunApplicationService interface {
	Create(ctx context.Context, input CreateApplicationRunInput) (*ApplicationRun, error)
	SubmitInput(ctx context.Context, runID string, input SubmitInputRequest) (*ApplicationRun, error)
	Get(ctx context.Context, runID string) (*model.Run, error)
	GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error)
	ListBySessionID(ctx context.Context, sessionID string, limit, offset int) (*RunHistoryPage, error)
	Cancel(ctx context.Context, runID string) (*model.Run, error)
}

// RunHistoryItem 是 UI 恢复会话时间线所需的持久化消息视图。
type RunHistoryItem struct {
	Run      *model.Run          `json:"run"`
	Messages []*model.RunMessage `json:"messages"`
}

type RunHistoryPage struct {
	Items  []RunHistoryItem `json:"items"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// CreateApplicationRunInput 是创建 Run 的 API 级输入。
type CreateApplicationRunInput struct {
	SessionID      string
	IdempotencyKey string
	Content        string
	AttachmentIDs  []string
}

// ApplicationRun 返回持久化 Run 和当前进程的可选控制句柄。
type ApplicationRun struct {
	Run    *model.Run
	Handle *RunExecutionHandle
}

type runApplicationStore interface {
	Create(ctx context.Context, input CreateRunInput) (*model.Run, error)
	SubmitInput(ctx context.Context, runID string, input SubmitInputRequest) (*RunResume, error)
	Get(ctx context.Context, runID string) (*model.Run, error)
	FindBySessionAndIdempotencyKey(ctx context.Context, sessionID, idempotencyKey string) (*model.Run, error)
	GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error)
	ListBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*model.Run, int, error)
	ListMessages(ctx context.Context, runID string) ([]*model.RunMessage, error)
	ListMessagesByRunIDs(ctx context.Context, runIDs []string) (map[string][]*model.RunMessage, error)
	RequestCancel(ctx context.Context, runID string) (*model.Run, error)
}

type runApplicationExecutor interface {
	Start(ctx context.Context, runID string, messages []llmcore.Message, contexts ...[]attachment.FileContext) (*RunExecutionHandle, error)
	Cancel(ctx context.Context, runID string) (*model.Run, error)
}

type sessionLookup interface {
	GetByID(ctx context.Context, id string) (*model.Session, error)
}

type fileLookup interface {
	GetBySessionAndID(ctx context.Context, sessionID, id string) (*model.File, error)
}

type agentSettingsSource interface {
	GetAgentSettings(ctx context.Context) (*settings.AgentSettings, error)
}

type attachmentLoader interface {
	Load(ctx context.Context, files []model.File, userMessage string) []attachment.FileContext
}

type defaultRunApplicationService struct {
	runs        runApplicationStore
	executor    runApplicationExecutor
	sessions    sessionLookup
	files       fileLookup
	settings    agentSettingsSource
	attachments attachmentLoader
	promptHash  string
}

// NewRunApplicationService 创建应用层编排服务。
func NewRunApplicationService(
	runs runApplicationStore,
	executor runApplicationExecutor,
	sessions sessionLookup,
	files fileLookup,
	settingsSource agentSettingsSource,
	attachments attachmentLoader,
	promptHash string,
) RunApplicationService {
	return &defaultRunApplicationService{
		runs: runs, executor: executor, sessions: sessions, files: files,
		settings: settingsSource, attachments: attachments, promptHash: strings.TrimSpace(promptHash),
	}
}

// Create 校验会话和附件归属，冻结 Settings/Prompt，并启动 pending Run。
func (s *defaultRunApplicationService) Create(ctx context.Context, input CreateApplicationRunInput) (*ApplicationRun, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return nil, apperr.BadRequest("session id is required")
	}
	if s.sessions == nil {
		return nil, apperr.FailedPrecondition("session lookup is unavailable")
	}
	session, err := s.sessions.GetByID(ctx, input.SessionID)
	if err != nil {
		return nil, apperr.ToInternal(err)
	}
	if session == nil {
		return nil, apperr.NotFound("session not found")
	}
	if s.runs == nil || s.executor == nil {
		return nil, apperr.FailedPrecondition("run execution is unavailable")
	}
	existing, err := s.findExistingCreateRun(ctx, input.SessionID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	var files []model.File
	var settingsSnapshot settings.AgentSettings
	promptHash := s.promptHash
	if existing != nil {
		settingsSnapshot = existing.SettingsSnapshot
		promptHash = existing.PromptHash
	}
	if existing == nil || existing.Status == model.RunStatusPending {
		files, err = s.resolveFiles(ctx, input.SessionID, input.AttachmentIDs)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			settingsSnapshot, err = s.loadSettings(ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	if strings.TrimSpace(promptHash) == "" {
		return nil, apperr.FailedPrecondition("prompt catalog is unavailable")
	}
	run, err := s.runs.Create(ctx, CreateRunInput{
		SessionID: input.SessionID, IdempotencyKey: input.IdempotencyKey,
		SettingsSnapshot: settingsSnapshot, PromptHash: promptHash,
		Content: input.Content, Attachments: append([]string(nil), input.AttachmentIDs...),
	})
	if err != nil {
		return nil, err
	}
	if run.Status != model.RunStatusPending {
		return &ApplicationRun{Run: run}, nil
	}
	contexts := s.loadAttachments(ctx, files, input.Content)
	executionCtx := context.WithoutCancel(ctx)
	handle, err := s.executor.Start(executionCtx, run.ID, []llmcore.Message{{
		Role: model.RoleUser, ContentText: input.Content, Attachments: append([]string(nil), input.AttachmentIDs...),
	}}, contexts)
	if err != nil {
		return nil, err
	}
	// Start 成功即表示执行器已同步完成 pending -> running；再次读取只会让
	// 已启动的请求因刷新失败误报失败，返回确定的状态迁移结果即可。
	started := *run
	started.Status = model.RunStatusRunning
	started.ExecutionSnapshot = run.ExecutionSnapshot.Clone()
	return &ApplicationRun{Run: &started, Handle: handle}, nil
}

// SubmitInput 持久化 waiting_input 回答后重新启动同一 Run 的执行上下文。
func (s *defaultRunApplicationService) SubmitInput(ctx context.Context, runID string, input SubmitInputRequest) (*ApplicationRun, error) {
	if s.runs == nil || s.executor == nil {
		return nil, apperr.FailedPrecondition("run execution is unavailable")
	}
	// 已持久化的回答是幂等重试：附件可能已过期或被清理，不能阻塞返回已应用结果。
	// RunService 仍会校验消息内容和回复目标，避免同一键绑定不同请求。
	applied, err := s.inputAlreadyApplied(ctx, runID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	var files []model.File
	if !applied {
		files, err = s.resolveFilesForRun(ctx, runID, input.Attachments)
		if err != nil {
			// 首次检查与附件读取之间，另一个相同请求可能已原子写入回答。
			// 仅在第二次确认相同幂等键已存在时才进入领域层：这样既能让
			// 重试跨过已清理的附件，也不会把无效附件持久化为一次新恢复。
			appliedAfterFailure, checkErr := s.inputAlreadyApplied(ctx, runID, input.IdempotencyKey)
			if checkErr != nil {
				return nil, checkErr
			}
			if !appliedAfterFailure {
				return nil, err
			}
			return s.submitAppliedInput(ctx, runID, input)
		}
	}
	resume, err := s.runs.SubmitInput(ctx, runID, input)
	if err != nil {
		return nil, err
	}
	if resume == nil {
		return nil, apperr.ToInternal(apperr.Internal("submit input returned empty result"))
	}
	if !resume.StartExecution {
		return &ApplicationRun{Run: resume.Run}, nil
	}
	contexts := s.loadAttachments(ctx, files, input.Content)
	handle, err := s.executor.Start(context.WithoutCancel(ctx), runID, resume.Messages, contexts)
	if err != nil {
		return nil, err
	}
	return &ApplicationRun{Run: resume.Run, Handle: handle}, nil
}

// submitAppliedInput 让领域层复核已持久化回答的完整契约，并返回其当前 Run 快照。
// 它只在应用层已确认同一幂等键存在时调用，不能作为附件校验失败的通用回退路径。
func (s *defaultRunApplicationService) submitAppliedInput(ctx context.Context, runID string, input SubmitInputRequest) (*ApplicationRun, error) {
	resume, err := s.runs.SubmitInput(ctx, runID, input)
	if err != nil {
		return nil, err
	}
	if resume == nil {
		return nil, apperr.ToInternal(apperr.Internal("submit input returned empty result"))
	}
	if resume.StartExecution {
		return nil, apperr.ToInternal(apperr.Internal("idempotent input unexpectedly requested execution start"))
	}
	return &ApplicationRun{Run: resume.Run}, nil
}

func (s *defaultRunApplicationService) inputAlreadyApplied(ctx context.Context, runID, idempotencyKey string) (bool, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return false, nil
	}
	messages, err := s.runs.ListMessages(ctx, runID)
	if err != nil {
		return false, apperr.ToInternal(err)
	}
	for _, message := range messages {
		if message != nil && message.IdempotencyKey == idempotencyKey {
			return true, nil
		}
	}
	return false, nil
}

func (s *defaultRunApplicationService) findExistingCreateRun(ctx context.Context, sessionID, idempotencyKey string) (*model.Run, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, nil
	}
	run, err := s.runs.FindBySessionAndIdempotencyKey(ctx, sessionID, idempotencyKey)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *defaultRunApplicationService) Get(ctx context.Context, runID string) (*model.Run, error) {
	if s.runs == nil {
		return nil, apperr.FailedPrecondition("run service is unavailable")
	}
	return s.runs.Get(ctx, runID)
}

// GetActiveBySessionID 查询 Session 当前唯一的活跃 Run，供临时 Session 路由适配使用。
func (s *defaultRunApplicationService) GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error) {
	if s.runs == nil {
		return nil, apperr.FailedPrecondition("run service is unavailable")
	}
	return s.runs.GetActiveBySessionID(ctx, sessionID)
}

// ListBySessionID 读取历史 Run 及其消息，避免 UI 继续依赖已停止写入的 Session.events。
func (s *defaultRunApplicationService) ListBySessionID(ctx context.Context, sessionID string, limit, offset int) (*RunHistoryPage, error) {
	if s.runs == nil {
		return nil, apperr.FailedPrecondition("run service is unavailable")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, apperr.BadRequest("session id is required")
	}
	runs, total, err := s.runs.ListBySessionID(ctx, sessionID, limit, offset)
	if err != nil {
		return nil, apperr.ToInternal(err)
	}
	runIDs := make([]string, 0, len(runs))
	for _, run := range runs {
		if run != nil {
			runIDs = append(runIDs, run.ID)
		}
	}
	messagesByRunID, err := s.runs.ListMessagesByRunIDs(ctx, runIDs)
	if err != nil {
		return nil, apperr.ToInternal(err)
	}
	page := &RunHistoryPage{Items: make([]RunHistoryItem, 0, len(runs)), Total: total, Limit: limit, Offset: offset}
	for _, run := range runs {
		if run == nil {
			continue
		}
		page.Items = append(page.Items, RunHistoryItem{Run: run, Messages: messagesByRunID[run.ID]})
	}
	return page, nil
}

func (s *defaultRunApplicationService) Cancel(ctx context.Context, runID string) (*model.Run, error) {
	if s.executor == nil {
		return nil, apperr.FailedPrecondition("run execution is unavailable")
	}
	return s.executor.Cancel(ctx, runID)
}

func (s *defaultRunApplicationService) loadSettings(ctx context.Context) (settings.AgentSettings, error) {
	if s.settings == nil {
		return settings.DefaultAgentSettings(), nil
	}
	persisted, err := s.settings.GetAgentSettings(ctx)
	if err != nil {
		return settings.AgentSettings{}, apperr.ToInternal(err)
	}
	if persisted == nil {
		return settings.DefaultAgentSettings(), nil
	}
	if err := persisted.Validate(); err != nil {
		return settings.AgentSettings{}, apperr.FailedPrecondition("invalid agent settings")
	}
	return *persisted, nil
}

func (s *defaultRunApplicationService) resolveFiles(ctx context.Context, sessionID string, ids []string) ([]model.File, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if s.files == nil {
		return nil, apperr.FailedPrecondition("file lookup is unavailable")
	}
	files := make([]model.File, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return nil, apperr.BadRequest("attachment id is required")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		file, err := s.files.GetBySessionAndID(ctx, sessionID, id)
		if err != nil {
			return nil, apperr.ToInternal(err)
		}
		if file == nil {
			return nil, apperr.NotFound("attachment not found")
		}
		files = append(files, *file)
	}
	return files, nil
}

func (s *defaultRunApplicationService) resolveFilesForRun(ctx context.Context, runID string, ids []string) ([]model.File, error) {
	run, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	return s.resolveFiles(ctx, run.SessionID, ids)
}

func (s *defaultRunApplicationService) loadAttachments(ctx context.Context, files []model.File, message string) []attachment.FileContext {
	if s.attachments == nil || len(files) == 0 {
		return nil
	}
	return s.attachments.Load(ctx, files, message)
}
