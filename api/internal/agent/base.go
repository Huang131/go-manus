package agent

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"strings"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/llmcore"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// BaseAgent Agent 基类
type BaseAgent struct {
	name         string
	sessionID    string
	config       *AgentConfig
	llm          external.LLM
	tools        []Tool
	memory       Memory
	toolRegistry *ToolRegistry
	jsonParser   external.JSONParser
	sessionRepo  repository.SessionRepository
}

// NewBaseAgent 创建基础 Agent
func NewBaseAgent(name, sessionID string, config *AgentConfig, llm external.LLM, tools []Tool) *BaseAgent {
	registry := NewToolRegistry()
	for _, tool := range tools {
		registry.Register(tool)
	}

	// 默认使用带修复功能的 JSON 解析器
	jsonParser := external.NewRepairJSONParser()

	return &BaseAgent{
		name:         name,
		sessionID:    sessionID,
		config:       config,
		llm:          llm,
		tools:        tools,
		memory:       NewSimpleMemory(config.MaxMemorySize),
		toolRegistry: registry,
		jsonParser:   jsonParser,
	}
}

// NewBaseAgentWithParser 创建基础 Agent（带自定义 JSON 解析器）
func NewBaseAgentWithParser(name, sessionID string, config *AgentConfig, llm external.LLM, tools []Tool, jsonParser external.JSONParser) *BaseAgent {
	registry := NewToolRegistry()
	for _, tool := range tools {
		registry.Register(tool)
	}

	if jsonParser == nil {
		jsonParser = external.NewRepairJSONParser()
	}

	return &BaseAgent{
		name:         name,
		sessionID:    sessionID,
		config:       config,
		llm:          llm,
		tools:        tools,
		memory:       NewSimpleMemory(config.MaxMemorySize),
		toolRegistry: registry,
		jsonParser:   jsonParser,
	}
}

// NewBaseAgentWithRepo 创建基础 Agent（带数据库仓储）
func NewBaseAgentWithRepo(name, sessionID string, config *AgentConfig, llm external.LLM, tools []Tool, sessionRepo repository.SessionRepository) *BaseAgent {
	agent := NewBaseAgent(name, sessionID, config, llm, tools)
	agent.sessionRepo = sessionRepo
	return agent
}

// NewBaseAgentWithRepoAndParser 创建基础 Agent（带数据库仓储和自定义 JSON 解析器）
func NewBaseAgentWithRepoAndParser(name, sessionID string, config *AgentConfig, llm external.LLM, tools []Tool, sessionRepo repository.SessionRepository, jsonParser external.JSONParser) *BaseAgent {
	agent := NewBaseAgentWithParser(name, sessionID, config, llm, tools, jsonParser)
	agent.sessionRepo = sessionRepo
	return agent
}

// SetSessionRepo 设置 Session 仓储
func (a *BaseAgent) SetSessionRepo(sessionRepo repository.SessionRepository) {
	a.sessionRepo = sessionRepo
}

// Name 返回 Agent 名称
func (a *BaseAgent) Name() string {
	return a.name
}

// SessionID 返回会话 ID
func (a *BaseAgent) SessionID() string {
	return a.sessionID
}

// AddMemory 添加记忆（带持久化）
func (a *BaseAgent) AddMemory(ctx context.Context, msg *model.Message) error {
	// 1. 添加到内存
	if err := a.memory.Add(msg); err != nil {
		return err
	}

	// 2. 持久化到数据库
	if a.sessionRepo != nil {
		memory := a.getModelMemory()
		if err := a.sessionRepo.SaveMemory(ctx, a.sessionID, a.name, memory); err != nil {
			logger.Error("记忆持久化失败",
				zap.String("session_id", a.sessionID),
				zap.String("agent_name", a.name),
				zap.Error(err))
			// 不返回错误，因为内存已成功添加
		}
	}

	return nil
}

// getModelMemory 获取模型层的 Memory
func (a *BaseAgent) getModelMemory() *model.Memory {
	messages := a.memory.GetMessages()
	modelMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		modelMessages = append(modelMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Message,
		})
	}
	return &model.Memory{
		Messages: modelMessages,
	}
}

// EnsureMemory 确保记忆已初始化（从数据库加载或创建新记忆）
func (a *BaseAgent) EnsureMemory(ctx context.Context) error {
	// 如果没有数据库仓储，跳过
	if a.sessionRepo == nil {
		return nil
	}

	// 从数据库加载记忆
	memory, err := a.sessionRepo.GetMemory(ctx, a.sessionID, a.name)
	if err != nil {
		logger.Warn("从数据库加载记忆失败，创建新记忆",
			zap.String("session_id", a.sessionID),
			zap.String("agent_name", a.name),
			zap.Error(err))
		return nil
	}

	// 如果记忆为空，创建新记忆
	if memory == nil || len(memory.Messages) == 0 {
		return nil
	}

	// 将数据库中的记忆加载到内存
	for _, msg := range memory.Messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		a.memory.Add(&model.Message{
			Role:    role,
			Message: content,
		})
	}

	logger.Info("从数据库加载记忆成功",
		zap.String("session_id", a.sessionID),
		zap.String("agent_name", a.name),
		zap.Int("message_count", len(memory.Messages)))

	return nil
}

// SaveMemory 手动保存记忆到数据库
func (a *BaseAgent) SaveMemory(ctx context.Context) error {
	if a.sessionRepo == nil {
		return nil
	}

	memory := a.getModelMemory()
	return a.sessionRepo.SaveMemory(ctx, a.sessionID, a.name, memory)
}

// GetMemory 获取记忆
func (a *BaseAgent) GetMemory() []*model.Message {
	return a.memory.GetMessages()
}

// CompactMemory 压缩记忆
func (a *BaseAgent) CompactMemory() error {
	// 保留最近的消息，确保最近的上下文不会丢失
	keepCount := 10
	return a.memory.Compact(keepCount)
}

// MemorySize 返回记忆大小
func (a *BaseAgent) MemorySize() int {
	return a.memory.Size()
}

// GetToolsForLLM 获取 LLM 可用的工具（阶段 1d：返回 llmcore.ToolSpec 强类型）
func (a *BaseAgent) GetToolsForLLM() []llmcore.ToolSpec {
	return a.toolRegistry.GetToolsForLLM()
}

// RollBack 回滚操作 (用于处理历史消息)
//
// 该方法用于确保 Agent 的消息列表状态正确，用于发送新消息、暂停/停止任务、通知用户
// 逻辑：
//   - 如果最后一条消息是工具调用 (assistant + tool_calls)
//   - 如果是 message_ask_user，添加用户回复作为工具响应
//   - 否则直接删除最后一条消息
//   - 如果最后一条消息不是工具调用，添加用户消息到记忆
func (a *BaseAgent) RollBack(ctx context.Context, msg *model.Message) error {
	// 1. 获取最后一条消息
	lastMessage := a.memory.GetLastMessage()
	if lastMessage == nil {
		// 如果没有消息，直接添加用户消息
		return a.AddMemory(ctx, msg)
	}

	// 2. 检查最后一条消息是否有 tool_calls
	hasToolCalls := len(lastMessage.ToolCalls) > 0

	if !hasToolCalls {
		// 3. 如果最后一条消息不是工具调用，直接添加用户消息
		return a.AddMemory(ctx, msg)
	}

	// 4. 如果是工具调用，获取工具信息
	toolCall := lastMessage.ToolCalls[0]
	toolCallID := toolCall.ID
	functionName := toolCall.Function.Name

	// 5. 判断是否是 message_ask_user 工具
	if functionName == "message_ask_user" {
		// 6. 特殊处理：添加用户回复作为工具响应
		userMsg := msg.Message
		content, _ := sonic.Marshal(userMsg)

		// 创建工具消息
		toolMsg := &model.Message{
			Role:    "tool",
			Message: string(content),
			ToolCalls: []llmcore.ToolCall{
				{
					ID:   toolCallID,
					Type: "function",
					Function: llmcore.ToolCallFunction{
						Name: functionName,
					},
				},
			},
		}
		return a.AddMemory(ctx, toolMsg)
	}

	// 7. 其他工具调用，直接删除最后一条消息
	if err := a.memory.RollbackLast(); err != nil {
		return err
	}

	// 8. 持久化到数据库
	if a.sessionRepo != nil {
		memory := a.getModelMemory()
		if err := a.sessionRepo.SaveMemory(ctx, a.sessionID, a.name, memory); err != nil {
			logger.Error("Rollback 后记忆持久化失败",
				zap.String("session_id", a.sessionID),
				zap.String("agent_name", a.name),
				zap.Error(err))
		}
	}

	return nil
}

// InvokeResult ReAct 循环的返回结果
type InvokeResult struct {
	// Content LLM 返回的文本内容
	Content string
	// ToolCall 是否调用了工具
	ToolCall bool
	// WaitForUser 是否需要等待用户输入
	WaitForUser bool
	// UserQuestion 如果需要等待用户输入，返回要展示给用户的问题
	UserQuestion string
	// Error 错误信息
	Error error
}

// invokeWithEmptyRetry 调用 LLM，当返回空 content 时自动注入"AI 无响应内容，请继续。"
// 并重试，对齐 mooc-manus 的 base.py retry-injection 逻辑。
// 返回最终响应与实际调用次数（成功即停止，不超过 maxRetries 次）。
//
// 参数 maxRetries 表示最多尝试次数（含首次）。例如 maxRetries=3 表示最多重试 2 次。
//
// 阶段 1d 改造点：messages 类型从 []map 改 []llmcore.Message。
// llmcore.Message 是值类型，深拷贝 = 元素拷贝即可（不再 map-by-map）。
func (a *BaseAgent) invokeWithEmptyRetry(ctx context.Context, req *external.LLMRequest, maxRetries int) (*external.LLMResponse, int, error) {
	if maxRetries < 1 {
		maxRetries = 1
	}

	// 深拷贝 messages，避免污染调用方的 slice
	messages := make([]llmcore.Message, len(req.Messages))
	copy(messages, req.Messages)

	current := *req
	current.Messages = messages

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := a.llm.Invoke(ctx, &current)
		if err != nil {
			lastErr = err
			// LLM 错误：注入空 assistant + 重试提示，然后继续
			current.Messages = append(current.Messages,
				llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""},
				llmcore.Message{Role: llmcore.RoleUser, ContentText: "AI 无响应内容，请继续。"},
			)
			continue
		}
		if resp.Content != "" {
			return resp, attempt, nil
		}
		lastErr = nil
		logger.Warn("LLM 返回空内容，执行重试",
			zap.String("session_id", a.sessionID),
			zap.String("agent", a.name),
			zap.Int("attempt", attempt))
		current.Messages = append(current.Messages,
			llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""},
			llmcore.Message{Role: llmcore.RoleUser, ContentText: "AI 无响应内容，请继续。"},
		)
	}

	if lastErr != nil {
		return nil, maxRetries, lastErr
	}
	return nil, maxRetries, fmt.Errorf("LLM 连续 %d 次返回空内容", maxRetries)
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	// ToolCallID 工具调用 ID
	ToolCallID string
	// ToolName 工具名称
	ToolName string
	// FunctionName 函数名称
	FunctionName string
	// Arguments 函数参数
	Arguments map[string]interface{}
	// Result 工具执行结果
	Result *model.ToolResult
	// WaitForUser 是否需要等待用户输入
	WaitForUser bool
}

// ErrWaitForUser 等待用户输入的错误
var ErrWaitForUser = fmt.Errorf("waiting for user input")

// retryInterval 重试间隔 (秒)
const retryInterval = 1.0

// Invoke 执行 ReAct 循环，调用 LLM 并处理工具调用
// 返回最终的消息内容，如果没有有效回复则返回错误
//
// 阶段 1d 改造点：messages 改 []llmcore.Message，handleToolCall 改 llmcore.ToolCall。
// assistantMsg / tool 消息改成 Message 结构体，不再用 map 拼字符串键。
func (a *BaseAgent) Invoke(ctx context.Context, query string) (*InvokeResult, error) {
	// 1. 构建初始消息
	messages := []llmcore.Message{
		{Role: llmcore.RoleUser, ContentText: query},
	}

	// 2. 循环调用 LLM 直到达到最大迭代次数或 LLM 不再调用工具
	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		// 3. 调用 LLM
		resp, err := a.llm.Invoke(ctx, &external.LLMRequest{
			Messages: messages,
			Tools:    a.GetToolsForLLM(),
		})
		if err != nil {
			// LLM 调用失败，尝试重试
			for retry := 0; retry < a.config.MaxRetries; retry++ {
				logger.Warn("LLM 调用失败，执行重试",
					zap.Int("retry", retry+1),
					zap.Error(err))

				// 添加空回复到历史
				messages = append(messages,
					llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""},
					llmcore.Message{Role: llmcore.RoleUser, ContentText: "AI 无响应内容，请继续。"},
				)

				resp, err = a.llm.Invoke(ctx, &external.LLMRequest{
					Messages: messages,
					Tools:    a.GetToolsForLLM(),
				})
				if err == nil {
					break
				}
			}

			if err != nil {
				return nil, fmt.Errorf("LLM 调用失败，已达到最大重试次数(%d): %w", a.config.MaxRetries, err)
			}
		}

		// 4. 处理工具调用
		if len(resp.ToolUse) > 0 {
			// 4a. 把 assistant + tool_calls 写回历史
			assistantMsg := llmcore.Message{
				Role:        llmcore.RoleAssistant,
				ContentText: resp.Content,
				ToolCalls:   resp.ToolUse,
				Reasoning:   resp.ReasoningContent,
			}
			messages = append(messages, assistantMsg)

			// 限制只处理第一个工具调用（避免并发问题）
			toolCalls := resp.ToolUse
			if len(toolCalls) > 1 {
				logger.Warn("LLM 返回多个工具调用，只处理第一个",
					zap.Int("total", len(toolCalls)))
				toolCalls = toolCalls[:1]
			}

			// 处理每个工具调用
			for _, tc := range toolCalls {
				result, err := a.handleToolCall(ctx, tc, messages)
				if err != nil {
					logger.Error("工具调用失败",
						zap.String("function", tc.Function.Name),
						zap.String("tool_call_id", tc.ID),
						zap.Error(err))
					// 添加错误结果到历史，继续循环
					messages = append(messages, llmcore.Message{
						Role:        llmcore.RoleTool,
						ToolCallID:  tc.ID,
						ContentText: fmt.Sprintf(`{"success": false, "message": "%s"}`, err.Error()),
					})
					continue
				}

				// 检测是否需要等待用户输入
				if result.WaitForUser {
					// 将工具结果添加到历史
					messages = append(messages, llmcore.Message{
						Role:        llmcore.RoleTool,
						ToolCallID:  result.ToolCallID,
						ContentText: result.Result.JSON(),
					})

					// 获取用户问题
					userQuestion := ""
					if text, ok := result.Arguments["text"].(string); ok {
						userQuestion = text
					}

					// 返回等待用户输入的特殊结果
					return &InvokeResult{
						Content:      "",
						ToolCall:     true,
						WaitForUser:  true,
						UserQuestion: userQuestion,
					}, nil
				}

				// 将工具结果添加到历史
				messages = append(messages, llmcore.Message{
					Role:        llmcore.RoleTool,
					ToolCallID:  result.ToolCallID,
					ContentText: result.Result.JSON(),
				})
			}

			// 工具执行完成后，继续循环让 LLM 处理结果
			continue
		}

		// 5. 如果没有工具调用，检查是否有有效内容
		if resp.Content == "" {
			logger.Warn("LLM 返回空内容，执行重试")

			// 添加空回复到历史
			messages = append(messages,
				llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""},
				llmcore.Message{Role: llmcore.RoleUser, ContentText: "AI 无响应内容，请继续。"},
			)
			continue
		}

		// 6. 有有效内容，添加到最后并返回
		messages = append(messages, llmcore.Message{
			Role:        llmcore.RoleAssistant,
			ContentText: resp.Content,
			Reasoning:   resp.ReasoningContent,
		})
		return &InvokeResult{
			Content:  resp.Content,
			ToolCall: false,
		}, nil
	}

	// 达到最大迭代次数
	return &InvokeResult{
		Content: "",
		Error:   fmt.Errorf("Agent 迭代超过最大次数: %d", a.config.MaxIterations),
	}, fmt.Errorf("Agent 迭代超过最大次数: %d", a.config.MaxIterations)
}

// handleToolCall 处理单个工具调用
//
// 阶段 1d 改造点：toolCall 从 map 改 llmcore.ToolCall；messages 同步改 []llmcore.Message。
// toolCall.Arguments 已经是 JSON 字符串，直接 sonic.Unmarshal / Parse 即可，不再 cast map。
func (a *BaseAgent) handleToolCall(ctx context.Context, toolCall llmcore.ToolCall, messages []llmcore.Message) (*ToolCallResult, error) {
	functionName := toolCall.Function.Name
	toolCallID := toolCall.ID

	// 解析参数（Arguments 是 JSON 字符串）
	var arguments map[string]interface{}
	if toolCall.Function.Arguments != "" {
		if err := a.jsonParser.Parse(toolCall.Function.Arguments, &arguments); err != nil {
			// 尝试直接解析
			if err := sonic.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
				arguments = make(map[string]interface{})
			}
		}
	}

	// 获取工具
	tool, ok := a.toolRegistry.Get(functionName)
	if !ok {
		// 可能是 MCP 工具，格式为 mcp_serverName_toolName
		if strings.HasPrefix(functionName, "mcp_") {
			tool, ok = a.toolRegistry.Get("mcp")
		}
		if !ok {
			return nil, fmt.Errorf("未知工具: %s", functionName)
		}
	}

	logger.Info("执行工具调用",
		zap.String("tool", tool.Name()),
		zap.String("function", functionName),
		zap.Any("arguments", arguments))

	// 特殊处理 message_ask_user 工具
	if functionName == "message_ask_user" {
		// 返回成功结果，并标记需要等待用户输入
		return &ToolCallResult{
			ToolCallID:   toolCallID,
			ToolName:     tool.Name(),
			FunctionName: functionName,
			Arguments:    arguments,
			Result:       model.NewToolResult(map[string]interface{}{"waiting_for_user": true}),
			WaitForUser:  true,
		}, nil
	}

	// 调用工具（带重试）
	var result *model.ToolResult
	var err error
	for retry := 0; retry < a.config.MaxRetries; retry++ {
		if multiTool, ok := tool.(MultiFunctionTool); ok {
			result, err = multiTool.InvokeWithName(functionName, ctx, arguments)
		} else {
			result, err = tool.Invoke(ctx, arguments)
		}
		if err == nil {
			break
		}
		logger.Warn("工具调用失败，执行重试",
			zap.String("function", functionName),
			zap.Int("retry", retry+1),
			zap.Error(err))
	}

	if err != nil {
		return &ToolCallResult{
			ToolCallID:   toolCallID,
			ToolName:     tool.Name(),
			FunctionName: functionName,
			Arguments:    arguments,
			Result:       model.NewToolError(err.Error()),
		}, err
	}

	return &ToolCallResult{
		ToolCallID:   toolCallID,
		ToolName:     tool.Name(),
		FunctionName: functionName,
		Arguments:    arguments,
		Result:       result,
	}, nil
}

// InvokeWithEvents 执行 ReAct 循环，返回事件流
//
// 阶段 1d 改造点：messages 改 []llmcore.Message，handleToolCall 改 llmcore.ToolCall。
// 取参数的方式从 tc["function"]["name"] 改 tc.Name，argumentsJSON cast 改 tc.Arguments。
func (a *BaseAgent) InvokeWithEvents(ctx context.Context, query string) <-chan model.BaseEvent {
	ch := make(chan model.BaseEvent, 100)

	go func() {
		defer close(ch)

		// 1. 构建初始消息
		messages := []llmcore.Message{
			{Role: llmcore.RoleUser, ContentText: query},
		}

		// 2. 循环调用 LLM
		for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
			// 发送工具调用开始事件
			ch <- model.NewMessageEvent("thinking", "思考中...")

			// 调用 LLM
			resp, err := a.llm.Invoke(ctx, &external.LLMRequest{
				Messages: messages,
				Tools:    a.GetToolsForLLM(),
			})
			if err != nil {
				ch <- model.NewErrorEvent(fmt.Sprintf("LLM 调用失败: %v", err))
				return
			}

			// 处理工具调用
			if len(resp.ToolUse) > 0 {
				// 构建助手消息
				assistantMsg := llmcore.Message{
					Role:        llmcore.RoleAssistant,
					ContentText: resp.Content,
					ToolCalls:   resp.ToolUse,
					Reasoning:   resp.ReasoningContent,
				}
				messages = append(messages, assistantMsg)

				// 限制只处理第一个工具调用
				toolCalls := resp.ToolUse
				if len(toolCalls) > 1 {
					logger.Warn("LLM 返回多个工具调用，只处理第一个",
						zap.Int("total", len(toolCalls)))
					toolCalls = toolCalls[:1]
				}

				for _, tc := range toolCalls {
					// 解析工具调用信息
					functionName := tc.Function.Name
					toolCallID := tc.ID

					// 解析参数
					var arguments map[string]interface{}
					if tc.Function.Arguments != "" {
						_ = a.jsonParser.Parse(tc.Function.Arguments, &arguments)
					}

					// 发送工具调用中事件
					ch <- model.NewToolCallingEvent(toolCallID, functionName, arguments)

					// 执行工具
					result, err := a.handleToolCall(ctx, tc, messages)
					if err != nil {
						ch <- model.NewToolCalledEvent(toolCallID, functionName, arguments, model.NewToolError(err.Error()))
					} else {
						ch <- model.NewToolCalledEvent(toolCallID, functionName, arguments, result.Result)
					}

					// 添加工具结果到历史
					messages = append(messages, llmcore.Message{
						Role:        llmcore.RoleTool,
						ToolCallID:  toolCallID,
						ContentText: result.Result.JSON(),
					})
				}
				continue
			}

			// 没有工具调用，检查内容
			if resp.Content != "" {
				messages = append(messages, llmcore.Message{
					Role:        llmcore.RoleAssistant,
					ContentText: resp.Content,
					Reasoning:   resp.ReasoningContent,
				})
				ch <- model.NewMessageEvent("assistant", resp.Content)
				return
			}

			// 空内容，继续循环
			messages = append(messages,
				llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""},
				llmcore.Message{Role: llmcore.RoleUser, ContentText: "AI 无响应内容，请继续。"},
			)
		}

		ch <- model.NewErrorEvent(fmt.Sprintf("Agent 迭代超过最大次数: %d", a.config.MaxIterations))
	}()

	return ch
}

// GetToolRegistry 获取工具注册表
func (a *BaseAgent) GetToolRegistry() *ToolRegistry {
	return a.toolRegistry
}

// GetLLM 获取 LLM
func (a *BaseAgent) GetLLM() external.LLM {
	return a.llm
}
