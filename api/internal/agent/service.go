package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// AgentService Agent 服务
type AgentService struct {
	mu           sync.RWMutex
	sessionRep   repository.SessionRepository
	fileRep      repository.FileRepository
	configRep    repository.AppConfigRepository
	llm          external.LLM
	sandbox      external.Sandbox
	agentConfig  *AgentConfig
	mcpConfig    *MCPConfig
	a2aConfig    *A2AConfig
	browser      external.Browser
	searchEngine external.SearchEngine
	fileStorage  COSFileStorage
	mcpTool      *MCPTool
	a2aTool      *A2ATool
	mq           external.MessageQueue

	// Session 与 Task 的映射（用于对接 Task 架构）
	taskBySession map[string]*RedisStreamTask
}

// NewAgentService 创建 Agent 服务
func NewAgentService(
	sessionRep repository.SessionRepository,
	fileRep repository.FileRepository,
	configRep repository.AppConfigRepository,
	llm external.LLM,
	sandbox external.Sandbox,
	agentConfig *AgentConfig,
	mcpConfig *MCPConfig,
	a2aConfig *A2AConfig,
	browser external.Browser,
	searchEngine external.SearchEngine,
	mq external.MessageQueue,
	fileStorage COSFileStorage,
) *AgentService {
	// 初始化 MCP 工具
	mcpTool := NewMCPTool()
	if mcpConfig != nil {
		_ = mcpTool.Initialize(mcpConfig)
	}

	// 初始化 A2A 工具
	a2aTool := NewA2ATool()
	if a2aConfig != nil {
		_ = a2aTool.Initialize(a2aConfig)
	}

	return &AgentService{
		sessionRep:    sessionRep,
		fileRep:       fileRep,
		configRep:     configRep,
		llm:           llm,
		sandbox:       sandbox,
		agentConfig:   agentConfig,
		mcpConfig:     mcpConfig,
		a2aConfig:     a2aConfig,
		browser:       browser,
		searchEngine:  searchEngine,
		fileStorage:   fileStorage,
		mcpTool:       mcpTool,
		a2aTool:       a2aTool,
		taskBySession: make(map[string]*RedisStreamTask),
		mq:            mq,
	}
}

// Chat 处理聊天消息
// 对齐 Python 版本的 RedisStreamTask 架构
func (s *AgentService) Chat(ctx context.Context, sessionID string, message *llmcore.Message) (string, error) {
	// 获取会话
	session, err := s.sessionRep.GetByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("获取会话失败: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 更新最新消息
	if err := s.sessionRep.UpdateLatestMessage(ctx, sessionID, message.ContentText); err != nil {
		logger.WarnContext(ctx, "更新最新消息失败", logger.String("session_id", sessionID), logger.Err(err))
	}

	// 创建独立的 task context，不受 HTTP 请求取消影响，但保留请求中的 trace 等 values。
	// 任务真正的取消由 RedisStreamTask.Invoke 创建并管理，避免这里遗留未释放的 cancel 函数。
	taskCtx := context.WithoutCancel(ctx)

	// 构建用户消息事件（DB 事件与 Redis 输入共用同一份，避免两种形状重复序列化）
	msgEvent := &model.MessageEvent{
		Type:    model.EventTypeMessage,
		Role:    string(message.Role),
		Message: message.ContentText,
	}
	if len(message.Attachments) > 0 {
		msgEvent.Attachments = s.resolveMessageAttachments(taskCtx, sessionID, message.Attachments)
	}

	// 添加用户消息事件到数据库
	userEventData, err := sonic.Marshal(msgEvent)
	if err != nil {
		logger.ErrorContext(ctx, "序列化用户消息事件失败",
			logger.String("session_id", sessionID),
			logger.Err(err))
		return "", fmt.Errorf("序列化用户消息事件失败: %w", err)
	}

	userEvent := &model.Event{
		Type: model.EventTypeMessage,
		Data: userEventData,
	}
	if err := s.sessionRep.AppendEvent(ctx, sessionID, userEvent); err != nil {
		logger.WarnContext(ctx, "添加用户消息事件失败", logger.String("session_id", sessionID), logger.Err(err))
	}

	// 获取或创建 RedisStreamTask
	task, err := s.getOrCreateTask(ctx, session, s.getTools())
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}

	// 启动任务执行（后台 goroutine）
	if err := task.Invoke(taskCtx); err != nil {
		// 如果任务已经在运行，不是错误，只记录日志
		logger.DebugContext(ctx, "任务已启动或已完成", logger.String("task_id", task.ID()), logger.Err(err))
	}

	// 将消息放入 input_stream
	if _, err := task.PutInput(taskCtx, msgEvent); err != nil {
		logger.ErrorContext(ctx, "放入消息失败", logger.String("session_id", sessionID), logger.Err(err))
		return task.ID(), fmt.Errorf("放入消息失败: %w", err)
	}

	logger.InfoContext(ctx, "Chat 处理消息",
		logger.String("session_id", sessionID),
		logger.String("task_id", task.ID()))

	return task.ID(), nil
}

// resolveMessageAttachments 将 API 传入的文件 ID 解析为文件元数据。
// 输入流携带完整文件对象，runner 才能在异步执行阶段下载并同步到沙箱。
func (s *AgentService) resolveMessageAttachments(ctx context.Context, sessionID string, fileIDs []string) []model.File {
	if s.fileRep == nil {
		return nil
	}

	attachments := make([]model.File, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		fileID = strings.TrimSpace(fileID)
		if fileID == "" {
			continue
		}

		file, err := s.fileRep.GetByID(ctx, fileID)
		if err != nil {
			logger.WarnContext(ctx, "获取聊天附件失败",
				logger.String("session_id", sessionID),
				logger.String("file_id", fileID),
				logger.Err(err))
			continue
		}
		if file != nil {
			attachments = append(attachments, *file)
		}
	}
	return attachments
}

// StopSession 停止会话
func (s *AgentService) StopSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 清理 Task
	if task, ok := s.taskBySession[sessionID]; ok {
		task.Cancel()
		delete(s.taskBySession, sessionID)
	}

	// 更新会话状态
	_ = s.sessionRep.UpdateStatus(ctx, sessionID, model.SessionStatusCompleted)

	return nil
}

// Shutdown 关闭服务
func (s *AgentService) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Info("Agent 服务关闭中...")

	// 取消所有 Task
	for sessionID, task := range s.taskBySession {
		task.Cancel()
		delete(s.taskBySession, sessionID)
	}

	logger.Info("Agent 服务已关闭")
}

// getTools 获取工具列表
func (s *AgentService) getTools() []Tool {
	tools := make([]Tool, 0)

	// 1. Shell 工具 (依赖 sandbox)
	if s.sandbox != nil {
		tools = append(tools, NewShellTool(s.sandbox))
	}

	// 2. File 工具 (依赖 sandbox)
	if s.sandbox != nil {
		tools = append(tools, NewFileTool(s.sandbox))
	}

	// 3. Browser 工具 (依赖 browser)
	if s.browser != nil {
		tools = append(tools, NewBrowserTool(s.browser))
	}

	// 4. Search 工具 (依赖 searchEngine)
	if s.searchEngine != nil {
		tools = append(tools, NewSearchTool(s.searchEngine))
	}

	// 5. Message 工具 (无需外部依赖)
	tools = append(tools, NewMessageTool())

	// 6. MCP 工具 (可选)
	if s.mcpTool != nil {
		tools = append(tools, s.mcpTool)
	}

	// 7. A2A 工具 (可选)
	if s.a2aTool != nil {
		tools = append(tools, s.a2aTool)
	}

	logger.Info("注册工具列表",
		logger.Int("count", len(tools)),
		logger.Any("tools", s.getToolNames(tools)))

	return tools
}

// getOrCreateTask 获取或创建 RedisStreamTask
// 对齐 Python: task = await RedisStreamTask.create(task_runner)
func (s *AgentService) getOrCreateTask(ctx context.Context, session *model.Session, tools []Tool) (*RedisStreamTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已有 task
	if task, ok := s.taskBySession[session.ID]; ok && !task.Done() {
		return task, nil
	}

	// 创建新的 task
	runner := NewAgentTaskRunner(&AgentTaskRunnerConfig{
		SessionID:   session.ID,
		AgentConfig: s.agentConfig,
		LLM:         s.llm,
		Tools:       tools,
		SessionRep:  s.sessionRep,
		FileRep:     s.fileRep,
		Sandbox:     s.sandbox,
		FileStorage: s.fileStorage,
	})

	task := NewRedisStreamTask(s.mq, runner)
	s.taskBySession[session.ID] = task

	logger.Info("创建新的 RedisStreamTask",
		logger.String("session_id", session.ID),
		logger.String("task_id", task.ID()))

	return task, nil
}

// GetTaskEvents 获取任务事件（供前端轮询 output_stream）
// 对齐 Python: 前端轮询 task.output_stream
//
// 修复点：原实现再次把 event.Data 当 model.Event 反序列化，导致
//  1. event.ID（Redis Stream ID）在传递给 handler 之前丢失
//  2. handler 收到 ID 为空的事件，前端 lastEventIdRef 永远拿不到值
//  3. 后续 startEmptyStream 用空游标从头读，巧合性能跑通但语义错位
//
// 现在直接透传底层解析好的 *model.Event，并保留 Redis Stream ID 作为 event_id 暴露给前端。
func (s *AgentService) GetTaskEvents(ctx context.Context, taskID string, startID string) ([]*model.Event, error) {
	task := defaultTaskRegistry.Get(taskID)
	if task == nil {
		logger.WarnContext(ctx, "GetTaskEvents: task not found", logger.String("task_id", taskID))
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	logger.DebugContext(ctx, "GetTaskEvents 获取事件",
		logger.String("task_id", taskID),
		logger.String("start_id", startID))

	events, err := task.GetOutput(ctx, startID)
	if err != nil {
		logger.ErrorContext(ctx, "GetTaskEvents 获取事件失败",
			logger.String("task_id", taskID),
			logger.Err(err))
		return nil, fmt.Errorf("get task events failed: %w", err)
	}

	// 透传事件：task.GetOutput 已将 Data 解析为业务 payload，Type / ID 已正确填充。
	result := make([]*model.Event, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		result = append(result, event)
	}

	logger.DebugContext(ctx, "GetTaskEvents 返回事件",
		logger.String("task_id", taskID),
		logger.Int("event_count", len(result)))

	return result, nil
}

// getToolNames 获取工具名称列表
func (s *AgentService) getToolNames(tools []Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}

// GetActiveTaskID 根据 sessionID 获取当前活跃的 task ID（用于空流续读）
// 对齐原 mooc-manus Python 版本的 _get_task 逻辑：返回该 session 最近关联的 task。
// 如果 task 已结束（不在 taskBySession 中），返回空字符串，调用方应关闭 SSE 流。
func (s *AgentService) GetActiveTaskID(ctx context.Context, sessionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.taskBySession[sessionID]
	if !ok || task == nil {
		return "", nil
	}
	return task.ID(), nil
}
