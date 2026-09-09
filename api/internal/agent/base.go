package agent

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"strings"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"

	"github.com/Huang131/go-manus/api/pkg/logger"
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
	eventCh      chan<- model.BaseEvent // 事件输出通道（由 Flow 注入，nil 时静默）
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

// SetEventCh 注入事件输出通道（Flow 创建事件流后调用）
func (a *BaseAgent) SetEventCh(ch chan<- model.BaseEvent) {
	a.eventCh = ch
}

// emitEvent 向事件通道发送事件，通道未注入时静默跳过。
// 由 Flow 保证事件通道的消费方（task_runner）持续消费，此处阻塞发送安全。
func (a *BaseAgent) emitEvent(ev model.BaseEvent) {
	if a.eventCh == nil {
		return
	}
	a.eventCh <- ev
}

// Name 返回 Agent 名称
func (a *BaseAgent) Name() string {
	return a.name
}

// SessionID 返回会话 ID
func (a *BaseAgent) SessionID() string {
	return a.sessionID
}

// LoadMemory 从数据库恢复记忆（当前未启用数据库持久化，保留为空操作）
func (a *BaseAgent) LoadMemory(ctx context.Context) error {
	return nil
}

// AddMemory 添加记忆
func (a *BaseAgent) AddMemory(ctx context.Context, msg llmcore.Message) error {
	return a.memory.Add(msg)
}

// buildConversationMessages 构建带记忆的完整 LLM 消息列表：system + 记忆原生消息 + 本次请求。
// 记忆以原生消息形态参与对话，tool 消息保留 tool_call_id 配对。
func (a *BaseAgent) buildConversationMessages(systemPrompt, query string) []llmcore.Message {
	memoryMessages := a.memory.GetMessages()
	messages := make([]llmcore.Message, 0, len(memoryMessages)+2)
	if systemPrompt != "" {
		messages = append(messages, llmcore.Message{Role: llmcore.RoleSystem, ContentText: systemPrompt})
	}
	messages = append(messages, memoryMessages...)
	messages = append(messages, llmcore.Message{Role: llmcore.RoleUser, ContentText: query})
	return messages
}

// mergeMemory 把本轮对话产生的新消息合并进记忆。
// msgs 必须只包含本轮新增的消息（不含 system 与历史记忆）。
func (a *BaseAgent) mergeMemory(ctx context.Context, msgs []llmcore.Message) {
	if len(msgs) == 0 {
		return
	}
	if err := a.memory.MergeMessages(msgs); err != nil {
		logger.Error("记忆合并失败", logger.Err(err))
	}
}

// GetMemory 获取记忆
func (a *BaseAgent) GetMemory() []llmcore.Message {
	return a.memory.GetMessages()
}

// CompactMemory 压缩记忆
func (a *BaseAgent) CompactMemory() error {
	// 保留最近的消息，确保最近的上下文不会丢失
	keepCount := 10
	return a.memory.Compact(keepCount)
}

// GetToolsForLLM 获取 LLM 可用的工具（阶段 1d：返回 llmcore.ToolSpec 强类型）
func (a *BaseAgent) GetToolsForLLM() []llmcore.ToolSpec {
	return a.toolRegistry.GetToolsForLLM()
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
func (a *BaseAgent) invokeWithEmptyRetry(ctx context.Context, req *external.LLMRequest, maxRetries int) (*llmcore.LLMResponse, int, error) {
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
		if resp.Message.ContentText != "" {
			return resp, attempt, nil
		}
		lastErr = nil
		logger.Warn("LLM 返回空内容，执行重试",
			logger.String("session_id", a.sessionID),
			logger.String("agent", a.name),
			logger.Int("attempt", attempt))
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
// 返回最终的消息内容，如果没有有效回复则返回错误。
//
// 消息组装：system + 记忆原生消息 + 本次 query（不再拼接字符串上下文）。
// 循环产生的 assistant/tool 消息在出口处合并进记忆并持久化。
func (a *BaseAgent) Invoke(ctx context.Context, systemPrompt, query string) (*InvokeResult, error) {
	// 1. 构建初始消息：system + 记忆 + 本次用户消息
	messages := a.buildConversationMessages(systemPrompt, query)
	// mergeFrom 指向本次用户消息，之后的所有消息都是本轮新增，需要合并进记忆
	mergeFrom := len(messages) - 1

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
					logger.Int("retry", retry+1),
					logger.Err(err))

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
				// 硬失败：本轮对话被重试注入污染，不合并进记忆
				return nil, fmt.Errorf("LLM 调用失败，已达到最大重试次数(%d): %w", a.config.MaxRetries, err)
			}
		}

		// 4. 处理工具调用
		if len(resp.Message.ToolCalls) > 0 {
			// 4a. 把 assistant + tool_calls 写回历史
			assistantMsg := llmcore.Message{
				Role:        llmcore.RoleAssistant,
				ContentText: resp.Message.ContentText,
				ToolCalls:   resp.Message.ToolCalls,
				Reasoning:   resp.Message.Reasoning,
			}
			messages = append(messages, assistantMsg)

			// 限制只处理第一个工具调用（避免并发问题）
			toolCalls := resp.Message.ToolCalls
			if len(toolCalls) > 1 {
				logger.Warn("LLM 返回多个工具调用，只处理第一个",
					logger.Int("total", len(toolCalls)))
				toolCalls = toolCalls[:1]
			}

			// 处理每个工具调用
			for _, tc := range toolCalls {
				result, err := a.handleToolCall(ctx, tc, messages)
				if err != nil {
					logger.Error("工具调用失败",
						logger.String("function", tc.Function.Name),
						logger.String("tool_call_id", tc.ID),
						logger.Err(err))
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

					// 合并本轮对话进记忆（assistant+tool_calls 与 tool 结果保持配对）
					a.mergeMemory(ctx, messages[mergeFrom:])

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
		if resp.Message.ContentText == "" {
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
			ContentText: resp.Message.ContentText,
			Reasoning:   resp.Message.Reasoning,
		})

		// 合并本轮完整对话进记忆并持久化
		a.mergeMemory(ctx, messages[mergeFrom:])

		return &InvokeResult{
			Content:  resp.Message.ContentText,
			ToolCall: false,
		}, nil
	}

	// 达到最大迭代次数：合并已发生的对话（工具调用历史对后续轮次有价值）
	a.mergeMemory(ctx, messages[mergeFrom:])

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
		if strings.HasPrefix(functionName, MCPFunctionPrefix) {
			tool, ok = a.toolRegistry.Get(ToolNameMCP)
		}
		if !ok {
			return nil, fmt.Errorf("未知工具: %s", functionName)
		}
	}

	logger.Info("执行工具调用",
		logger.String("tool", tool.Name()),
		logger.String("function", functionName),
		logger.Any("arguments", arguments))

	// 发出工具调用开始事件（tool_calling），前端 SSE 实时展示调用参数
	callingEvent := model.NewToolCallingEvent(toolCallID, functionName, arguments)
	callingEvent.Name = tool.Name()
	a.emitEvent(callingEvent)

	// 特殊处理 message_ask_user 工具
	if functionName == MessageFunctionAskUser {
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
			logger.String("function", functionName),
			logger.Int("retry", retry+1),
			logger.Err(err))
	}

	if err != nil {
		// 失败也发出 tool_called 事件，携带错误结果，前端可展示失败详情
		calledEvent := model.NewToolCalledEvent(toolCallID, functionName, arguments, model.NewToolError(err.Error()))
		calledEvent.Name = tool.Name()
		a.emitEvent(calledEvent)

		return &ToolCallResult{
			ToolCallID:   toolCallID,
			ToolName:     tool.Name(),
			FunctionName: functionName,
			Arguments:    arguments,
			Result:       model.NewToolError(err.Error()),
		}, err
	}

	// 发出工具调用完成事件（tool_called）
	calledEvent := model.NewToolCalledEvent(toolCallID, functionName, arguments, result)
	calledEvent.Name = tool.Name()
	a.emitEvent(calledEvent)

	return &ToolCallResult{
		ToolCallID:   toolCallID,
		ToolName:     tool.Name(),
		FunctionName: functionName,
		Arguments:    arguments,
		Result:       result,
	}, nil
}

// GetToolRegistry 获取工具注册表
func (a *BaseAgent) GetToolRegistry() *ToolRegistry {
	return a.toolRegistry
}

// GetLLM 获取 LLM
func (a *BaseAgent) GetLLM() external.LLM {
	return a.llm
}
