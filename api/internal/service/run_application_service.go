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
	Cancel(ctx context.Context, runID string) (*model.Run, error)
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
	GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error)
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
	files, err := s.resolveFiles(ctx, input.SessionID, input.AttachmentIDs)
	if err != nil {
		return nil, err
	}
	settingsSnapshot, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	if s.runs == nil || s.executor == nil {
		return nil, apperr.FailedPrecondition("run execution is unavailable")
	}
	if s.promptHash == "" {
		return nil, apperr.FailedPrecondition("prompt catalog is unavailable")
	}
	run, err := s.runs.Create(ctx, CreateRunInput{
		SessionID: input.SessionID, IdempotencyKey: input.IdempotencyKey,
		SettingsSnapshot: settingsSnapshot, PromptHash: s.promptHash,
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
	// Start 已同步写入 running；读取最新事实，避免创建响应返回过时的 pending 状态。
	started, err := s.runs.Get(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	return &ApplicationRun{Run: started, Handle: handle}, nil
}

// SubmitInput 持久化 waiting_input 回答后重新启动同一 Run 的执行上下文。
func (s *defaultRunApplicationService) SubmitInput(ctx context.Context, runID string, input SubmitInputRequest) (*ApplicationRun, error) {
	if s.runs == nil || s.executor == nil {
		return nil, apperr.FailedPrecondition("run execution is unavailable")
	}
	files, err := s.resolveFilesForRun(ctx, runID, input.Attachments)
	if err != nil {
		return nil, err
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
