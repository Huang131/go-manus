package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
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
	mcpTool      *MCPTool
	a2aTool      *A2ATool
	mq           external.MessageQueue

	// 运行中的任务
	runningTasks map[string]*AgentTaskRunner

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
		mcpTool:       mcpTool,
		a2aTool:       a2aTool,
		runningTasks:  make(map[string]*AgentTaskRunner),
		taskBySession: make(map[string]*RedisStreamTask),
		mq:            mq,
	}
}

// Chat 处理聊天消息
// 对齐 Python 版本的 RedisStreamTask 架构
func (s *AgentService) Chat(ctx context.Context, sessionID string, message *model.Message) (string, error) {
	// 获取会话
	session, err := s.sessionRep.GetByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("获取会话失败: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 更新最新消息
	if err := s.sessionRep.UpdateLatestMessage(ctx, sessionID, message.Message); err != nil {
		logger.Warn("更新最新消息失败", zap.String("session_id", sessionID), zap.Error(err))
	}

	// 添加用户消息事件到数据库
	userEventData, err := safeMarshal(map[string]interface{}{
		"role":    "user",
		"message": message.Message,
	})
	if err != nil {
		logger.Error("序列化用户消息事件失败",
			zap.String("session_id", sessionID),
			zap.Error(err))
		return "", fmt.Errorf("序列化用户消息事件失败: %w", err)
	}

	userEvent := &model.Event{
		Type: model.EventTypeMessage,
		Data: userEventData,
	}
	if err := s.sessionRep.AppendEvent(ctx, sessionID, userEvent); err != nil {
		logger.Warn("添加用户消息事件失败", zap.String("session_id", sessionID), zap.Error(err))
	}

	// 创建独立的 task context，不受 HTTP 请求 context 影响
	// 注意：这里不使用 defer cancel()，让 context 保持活跃直到 Agent 任务完成
	// 这样当 HTTP 请求返回后，Agent 任务仍能继续运行
	taskCtx, _ := context.WithCancel(context.Background())

	// 获取或创建 RedisStreamTask
	task, err := s.getOrCreateTask(ctx, session, s.getTools())
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}

	// 启动任务执行（后台 goroutine）
	if err := task.Invoke(taskCtx); err != nil {
		// 如果任务已经在运行，不是错误，只记录日志
		logger.Debug("任务已启动或已完成", zap.String("task_id", task.ID()), zap.Error(err))
	}

	// 将消息放入 input_stream
	msgEvent := &model.MessageEvent{
		Type:    model.EventTypeMessage,
		Role:    message.Role,
		Message: message.Message,
	}
	if _, err := task.PutInput(taskCtx, msgEvent); err != nil {
		logger.Error("放入消息失败", zap.String("session_id", sessionID), zap.Error(err))
		return task.ID(), fmt.Errorf("放入消息失败: %w", err)
	}

	logger.Info("Chat 处理消息",
		zap.String("session_id", sessionID),
		zap.String("task_id", task.ID()))

	return task.ID(), nil
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

	// 清理旧的 TaskRunner
	if _, ok := s.runningTasks[sessionID]; ok {
		delete(s.runningTasks, sessionID)
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

	s.runningTasks = make(map[string]*AgentTaskRunner)
	logger.Info("Agent 服务已关闭")
}

// getOrCreateTaskRunner 获取或创建任务运行器
// Deprecated: 此方法已废弃，请使用 getOrCreateTask() 代替
func (s *AgentService) getOrCreateTaskRunner(session *model.Session, tools []Tool) *AgentTaskRunner {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已有运行中的任务，直接返回
	if runner, ok := s.runningTasks[session.ID]; ok {
		return runner
	}

	// 创建新的任务运行器
	runner := NewAgentTaskRunner(&AgentTaskRunnerConfig{
		SessionID:   session.ID,
		AgentConfig: s.agentConfig,
		LLM:         s.llm,
		Tools:       tools,
		SessionRep:  s.sessionRep,
		FileRep:     s.fileRep,
		Sandbox:     s.sandbox,
	})

	s.runningTasks[session.ID] = runner
	return runner
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
		zap.Int("count", len(tools)),
		zap.Any("tools", s.getToolNames(tools)))

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
	})

	task := NewRedisStreamTask(s.mq, runner)
	s.taskBySession[session.ID] = task

	logger.Info("创建新的 RedisStreamTask",
		zap.String("session_id", session.ID),
		zap.String("task_id", task.ID()))

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
		logger.Warn("GetTaskEvents: task not found", zap.String("task_id", taskID))
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	logger.Debug("GetTaskEvents 获取事件",
		zap.String("task_id", taskID),
		zap.String("start_id", startID))

	events, err := task.GetOutput(ctx, startID)
	if err != nil {
		logger.Error("GetTaskEvents 获取事件失败",
			zap.String("task_id", taskID),
			zap.Error(err))
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

	logger.Debug("GetTaskEvents 返回事件",
		zap.String("task_id", taskID),
		zap.Int("event_count", len(result)))

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

// RegisterTool 注册自定义工具
func (s *AgentService) RegisterTool(tool Tool) {
	// 工具注册到所有活跃任务
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 注意: AgentTaskRunner 创建时已固定工具列表
	// 如需动态添加工具，需要修改设计
	logger.Info("注册工具", zap.String("name", tool.Name()))
}

// GetRunningTasks 获取运行中的任务数
func (s *AgentService) GetRunningTasks() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.runningTasks)
}
