package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/internal/jsonx"
	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"

	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// BaseAgent Agent 基类
type BaseAgent struct {
	name             string
	sessionID        string
	config           *AgentConfig
	llm              llm.LLM
	tools            []toolspkg.Tool
	memory           Memory
	contextBuilder   *ContextBuilder
	toolRegistry     *toolspkg.ToolRegistry
	jsonParser       jsonx.JSONParser
	eventCh          chan<- model.BaseEvent // 事件输出通道（由 Flow 注入，nil 时静默）
	shellWatchMu     sync.Mutex
	shellWatchCancel context.CancelFunc
	shellWatchDone   <-chan struct{}
}

// NewBaseAgent 创建基础 Agent
func NewBaseAgent(name, sessionID string, config *AgentConfig, llm llm.LLM, tools []toolspkg.Tool) *BaseAgent {
	registry := toolspkg.NewToolRegistry()
	for _, tool := range tools {
		registry.Register(tool)
	}

	// 默认使用带修复功能的 JSON 解析器
	jsonParser := jsonx.NewRepairJSONParser()

	return &BaseAgent{
		name:           name,
		sessionID:      sessionID,
		config:         config,
		llm:            llm,
		tools:          tools,
		memory:         NewSimpleMemory(),
		contextBuilder: NewContextBuilder(ContextPolicy{}),
		toolRegistry:   registry,
		jsonParser:     jsonParser,
	}
}

// formatRepairTypes 将修复步骤列表格式化为逗号分隔的字符串
func formatRepairTypes(repairs []jsonx.Repair) string {
	if len(repairs) == 0 {
		return ""
	}
	types := make([]string, 0, len(repairs))
	for _, r := range repairs {
		types = append(types, r.Type)
	}
	return strings.Join(types, ",")
}

// NewBaseAgentWithParser 创建基础 Agent（带自定义 JSON 解析器）
func NewBaseAgentWithParser(name, sessionID string, config *AgentConfig, llm llm.LLM, tools []toolspkg.Tool, jsonParser jsonx.JSONParser) *BaseAgent {
	registry := toolspkg.NewToolRegistry()
	for _, tool := range tools {
		registry.Register(tool)
	}

	if jsonParser == nil {
		jsonParser = jsonx.NewRepairJSONParser()
	}

	return &BaseAgent{
		name:           name,
		sessionID:      sessionID,
		config:         config,
		llm:            llm,
		tools:          tools,
		memory:         NewSimpleMemory(),
		contextBuilder: NewContextBuilder(ContextPolicy{}),
		toolRegistry:   registry,
		jsonParser:     jsonParser,
	}
}

// SetEventCh 注入事件输出通道（Flow 创建事件流后调用）
func (a *BaseAgent) SetEventCh(ch chan<- model.BaseEvent) {
	a.eventCh = ch
}

// emitEvent 向事件通道发送事件，通道未注入时静默跳过。
// Flow 在关闭事件通道前停止所有后台 watcher，保证发送方生命周期不越界。
func (a *BaseAgent) emitEvent(ctx context.Context, ev model.BaseEvent) {
	if a.eventCh == nil {
		return
	}
	select {
	case a.eventCh <- ev:
	case <-ctx.Done():
	}
}

// Name 返回 Agent 名称
func (a *BaseAgent) Name() string {
	return a.name
}

// SessionID 返回会话 ID
func (a *BaseAgent) SessionID() string {
	return a.sessionID
}

// AddMemory 添加记忆
func (a *BaseAgent) AddMemory(ctx context.Context, msg llmcore.Message) error {
	return a.memory.Add(msg)
}

// buildConversationMessages 构建带记忆的完整 LLM 消息列表：system + 记忆原生消息 + 本次请求。
// 记忆以原生消息形态参与对话，tool 消息保留 tool_call_id 配对。
func (a *BaseAgent) buildConversationMessages(systemPrompt, query string) ([]llmcore.Message, error) {
	memoryMessages := a.memory.GetMessages()
	return a.contextBuilder.Build(systemPrompt, memoryMessages, query)
}

// mergeMemory 把本轮对话产生的新消息合并进记忆。
// msgs 必须只包含本轮新增的消息（不含 system 与历史记忆）。
func (a *BaseAgent) mergeMemory(ctx context.Context, msgs []llmcore.Message) {
	if len(msgs) == 0 {
		return
	}
	if err := a.memory.MergeMessages(msgs); err != nil {
		logger.ErrorContext(ctx, "记忆合并失败", logger.Err(err))
	}
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
func (a *BaseAgent) invokeWithEmptyRetry(ctx context.Context, req *llm.LLMRequest, maxRetries int) (*llmcore.LLMResponse, int, error) {
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
		resp, err := a.invokeLLM(ctx, &current, false)
		if err != nil {
			if ctx.Err() != nil {
				return nil, attempt, ctx.Err()
			}
			lastErr = err
			// LLM 错误：注入空 assistant + 重试提示，然后继续
			current.Messages = append(current.Messages,
				llmcore.Message{Role: model.RoleAssistant, ContentText: ""},
				llmcore.Message{Role: model.RoleUser, ContentText: "AI 无响应内容，请继续。"},
			)
			continue
		}
		if resp.Message.ContentText != "" {
			return resp, attempt, nil
		}
		lastErr = nil
		logger.WarnContext(ctx, "LLM 返回空内容，执行重试",
			logger.String("session_id", a.sessionID),
			logger.String("agent", a.name),
			logger.Int("attempt", attempt))
		current.Messages = append(current.Messages,
			llmcore.Message{Role: model.RoleAssistant, ContentText: ""},
			llmcore.Message{Role: model.RoleUser, ContentText: "AI 无响应内容，请继续。"},
		)
	}

	if lastErr != nil {
		return nil, maxRetries, lastErr
	}
	return nil, maxRetries, fmt.Errorf("LLM 连续 %d 次返回空内容", maxRetries)
}

// invokeLLM 优先消费 provider 的 token stream，并在事件通道中发布增量；
// 不支持流式的 mock/适配器继续走 Invoke，保证 Agent 接口保持兼容。
// 增量只携带文本，不暴露 reasoning，工具参数仍由聚合后的完整响应处理。
func (a *BaseAgent) invokeLLM(ctx context.Context, req *llm.LLMRequest, publishDeltas bool) (*llmcore.LLMResponse, error) {
	resp, _, err := a.invokeLLMWithEmission(ctx, req, publishDeltas)
	return resp, err
}

// invokeLLMWithEmission 与 invokeLLM 相同，但额外返回本次调用是否已经向事件流发布文本增量。
// 总结阶段据此避免同时发送 message_delta/message_done 和重复的 message 事件。
func (a *BaseAgent) invokeLLMWithEmission(ctx context.Context, req *llm.LLMRequest, publishDeltas bool) (*llmcore.LLMResponse, bool, error) {
	streaming, ok := a.llm.(llm.StreamingLLM)
	if !ok {
		resp, err := a.llm.Invoke(ctx, req)
		if err == nil && ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		return resp, false, err
	}

	deltas, err := streaming.Stream(ctx, req)
	if err != nil {
		return nil, false, err
	}
	if deltas == nil {
		return nil, false, fmt.Errorf("LLM 流式响应通道为空")
	}
	all := make([]llmcore.LLMDelta, 0, 16)
	messageID := uuid.NewString()
	sequence := 0
	hasText := false
	var streamErr string

streamLoop:
	for {
		// 某些 provider 在取消后不会及时关闭响应流；不能用 range 等待其关闭，
		// 否则 StopSession 会被上游连接生命周期拖住。
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		var delta llmcore.LLMDelta
		var ok bool
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case delta, ok = <-deltas:
			if !ok {
				break streamLoop
			}
		}
		if delta.Error != "" {
			if streamErr == "" {
				streamErr = delta.Error
			}
			continue
		}
		all = append(all, delta)
		if publishDeltas && delta.ContentText != "" {
			hasText = true
			sequence++
			a.emitEvent(ctx, model.NewMessageDeltaEvent(messageID, delta.ContentText, sequence))
		}
	}
	if streamErr != "" {
		return nil, false, fmt.Errorf("LLM 流式调用失败: %s", streamErr)
	}
	// ctx 取消/超时时流会被静默截断：此时 all 里只有半截内容，
	// 不能当成完整回复合并进记忆或发给调用方。
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	resp := llmcore.MergeDeltas(a.llm.ModelName(), all)
	if resp == nil {
		return nil, false, fmt.Errorf("LLM 流式响应为空")
	}
	if publishDeltas && hasText {
		a.emitEvent(ctx, model.NewMessageDoneEvent(messageID, resp.Message.ContentText, resp.FinishReason))
	}
	return resp, publishDeltas && hasText, nil
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
	return a.invoke(ctx, systemPrompt, query, true)
}

// InvokeWithoutStreaming 执行 ReAct 循环但不发布用户可见 token 增量。
// Planner/ReAct 的结构化 JSON 响应需要完整聚合后解析，不能把半截 JSON 推给前端。
func (a *BaseAgent) InvokeWithoutStreaming(ctx context.Context, systemPrompt, query string) (*InvokeResult, error) {
	return a.invoke(ctx, systemPrompt, query, false)
}

func (a *BaseAgent) invoke(ctx context.Context, systemPrompt, query string, publishDeltas bool) (*InvokeResult, error) {
	// 指标埋点：记录本轮 ReAct 实际迭代次数，用于对比上下文精简前后的工具调用收敛效果
	iteration := 0
	defer func() {
		logger.InfoContext(ctx, "metric.react_iterations",
			logger.String("session_id", a.sessionID),
			logger.Int("iterations", iteration))
	}()

	// 1. 构建初始消息：system + 记忆 + 本次用户消息
	messages, err := a.buildConversationMessages(systemPrompt, query)
	if err != nil {
		return nil, err
	}
	// mergeFrom 指向本次用户消息，之后的所有消息都是本轮新增，需要合并进记忆
	mergeFrom := len(messages) - 1

	// 2. 循环调用 LLM 直到达到最大迭代次数或 LLM 不再调用工具
	for iteration < a.config.MaxIterations {
		iteration++
		// 3. 调用 LLM
		llmReq := &llm.LLMRequest{
			Messages: messages,
			Tools:    a.GetToolsForLLM(),
		}
		resp, err := a.invokeLLM(ctx, llmReq, publishDeltas && shouldPublishDeltas(llmReq))
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// LLM 调用失败，尝试重试
			for retry := 0; retry < a.config.MaxRetries; retry++ {
				logger.WarnContext(ctx, "LLM 调用失败，执行重试",
					logger.Int("retry", retry+1),
					logger.Err(err))

				// 添加空回复到历史
				messages = append(messages,
					llmcore.Message{Role: model.RoleAssistant, ContentText: ""},
					llmcore.Message{Role: model.RoleUser, ContentText: "AI 无响应内容，请继续。"},
				)

				llmReq = &llm.LLMRequest{
					Messages: messages,
					Tools:    a.GetToolsForLLM(),
				}
				resp, err = a.invokeLLM(ctx, llmReq, publishDeltas && shouldPublishDeltas(llmReq))
				if err == nil {
					break
				}
				if ctx.Err() != nil {
					return nil, ctx.Err()
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
				Role:        model.RoleAssistant,
				ContentText: resp.Message.ContentText,
				ToolCalls:   resp.Message.ToolCalls,
				Reasoning:   resp.Message.Reasoning,
			}
			messages = append(messages, assistantMsg)

			// 限制只处理第一个工具调用（避免并发问题）
			toolCalls := resp.Message.ToolCalls
			if len(toolCalls) > 1 {
				logger.WarnContext(ctx, "LLM 返回多个工具调用，只处理第一个",
					logger.Int("total", len(toolCalls)))
				toolCalls = toolCalls[:1]
			}

			// 处理每个工具调用
			for _, tc := range toolCalls {
				result, err := a.handleToolCall(ctx, tc, messages)
				if err != nil {
					logger.ErrorContext(ctx, "工具调用失败",
						logger.String("function", tc.Function.Name),
						logger.String("tool_call_id", tc.ID),
						logger.Err(err))
					// 添加错误结果到历史，继续循环
					messages = append(messages, llmcore.Message{
						Role:        model.RoleTool,
						ToolCallID:  tc.ID,
						ContentText: fmt.Sprintf(`{"success": false, "message": "%s"}`, err.Error()),
					})
					continue
				}

				// 检测是否需要等待用户输入
				if result.WaitForUser {
					// 将工具结果添加到历史
					messages = append(messages, llmcore.Message{
						Role:        model.RoleTool,
						ToolCallID:  result.ToolCallID,
						ContentText: result.Result.LLMJSON(),
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
					Role:        model.RoleTool,
					ToolCallID:  result.ToolCallID,
					ContentText: result.Result.LLMJSON(),
				})
			}

			// 工具执行完成后，继续循环让 LLM 处理结果
			continue
		}

		// 5. 如果没有工具调用，检查是否有有效内容
		if resp.Message.ContentText == "" {
			logger.WarnContext(ctx, "LLM 返回空内容，执行重试")

			// 添加空回复到历史
			messages = append(messages,
				llmcore.Message{Role: model.RoleAssistant, ContentText: ""},
				llmcore.Message{Role: model.RoleUser, ContentText: "AI 无响应内容，请继续。"},
			)
			continue
		}

		// 6. 有有效内容，添加到最后并返回
		messages = append(messages, llmcore.Message{
			Role:        model.RoleAssistant,
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

// shouldPublishDeltas 对非结构化响应发布文本增量。
// 工具参数始终只在完整流聚合后解析；即使请求携带工具定义，模型返回的自然语言内容仍可即时展示。
func shouldPublishDeltas(req *llm.LLMRequest) bool {
	return req != nil && req.ResponseFormat == nil
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
		if repairParser, ok := a.jsonParser.(*jsonx.RepairJSONParser); ok {
			// 带修复埋点的解析路径
			repairs, err := repairParser.ParseWithRepairs(toolCall.Function.Arguments, &arguments)
			if err != nil && repairs != nil && repairs.WasRepaired {
				// 修复后仍失败，但有修复记录则埋点
				logger.WarnContext(ctx, "json.parse.repaired.failed",
					logger.String("session_id", a.sessionID),
					logger.String("function", functionName),
					logger.String("repair_types", formatRepairTypes(repairs.Repairs)))
				arguments = make(map[string]interface{})
			} else if err == nil && repairs != nil && repairs.WasRepaired {
				// 成功但经过了修复，记录 info 便于观测
				logger.InfoContext(ctx, "json.parse.repaired",
					logger.String("session_id", a.sessionID),
					logger.String("function", functionName),
					logger.String("repair_types", formatRepairTypes(repairs.Repairs)))
			}
		} else {
			// 非 RepairJSONParser，降级到普通解析
			if err := a.jsonParser.Parse(toolCall.Function.Arguments, &arguments); err != nil {
				if err := sonic.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
					arguments = make(map[string]interface{})
				}
			}
		}
	}

	// 获取工具
	tool, ok := a.toolRegistry.Get(functionName)
	if !ok {
		// 可能是 MCP 工具，格式为 mcp_serverName_toolName
		if strings.HasPrefix(functionName, toolspkg.MCPFunctionPrefix) {
			tool, ok = a.toolRegistry.Get(toolspkg.ToolNameMCP)
		}
		if !ok {
			return nil, fmt.Errorf("未知工具: %s", functionName)
		}
	}

	logger.InfoContext(ctx, "执行工具调用",
		logger.String("tool", tool.Name()),
		logger.String("function", functionName),
		logger.Any("arguments", arguments))

	// session_id 由 Agent 注入，不交给模型填写：沙箱会话与本次对话一一对应，
	// 模型自行编造 session_id 会在共享沙箱里串到别的会话。
	// 必须在发出 tool_calling 之前注入，前端才能用它关联 shell_output 增量。
	if functionName == toolspkg.ToolNameShell {
		arguments["session_id"] = a.sessionID
	}

	// 发出工具调用开始事件（tool_calling），前端 SSE 实时展示调用参数
	callingEvent := model.NewToolCallingEvent(toolCallID, functionName, arguments)
	callingEvent.Name = tool.Name()
	a.emitEvent(ctx, callingEvent)

	// 特殊处理 message_ask_user 工具
	if functionName == toolspkg.MessageFunctionAskUser {
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

	// 不重试工具调用：click/exec/write 这类操作没有幂等保证，
	// 超时后重放可能重复点击、重复执行带副作用的命令。
	// 失败交由模型根据错误结果自行决策是否换个方式重试。
	var result *model.ToolResult
	var err error
	if multiTool, ok := tool.(toolspkg.MultiFunctionTool); ok {
		result, err = multiTool.InvokeWithName(functionName, ctx, arguments)
	} else {
		result, err = tool.Invoke(ctx, arguments)
	}

	if err != nil {
		logger.WarnContext(ctx, "metric.tool_call",
			logger.String("session_id", a.sessionID),
			logger.String("function", functionName),
			logger.Bool("success", false),
			logger.Err(err))

		// 失败也发出 tool_called 事件，携带错误结果，前端可展示失败详情
		calledEvent := model.NewToolCalledEvent(toolCallID, functionName, arguments, model.NewToolError(err.Error()))
		calledEvent.Name = tool.Name()
		a.emitEvent(ctx, calledEvent)

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
	a.emitEvent(ctx, calledEvent)

	// 长命令输出实时推流：shell exec 返回 running（同步等待窗口内未结束）时，
	// 启动后台 watch 周期性推送控制台快照，前端据此刷新 shell 预览。
	if functionName == "shell" && result != nil && result.Success {
		if action, _ := arguments["action"].(string); action == "exec" {
			if sessionID, _ := arguments["session_id"].(string); sessionID != "" {
				if data, ok := result.Data.(map[string]interface{}); ok && data["status"] == "running" {
					if sh, ok := tool.(interface{ Sandbox() sandbox.Sandbox }); ok && sh.Sandbox() != nil {
						a.startShellWatch(ctx, sh.Sandbox(), sessionID)
					}
				}
			}
		}
	}

	logger.InfoContext(ctx, "metric.tool_call",
		logger.String("session_id", a.sessionID),
		logger.String("function", functionName),
		logger.Bool("success", result != nil && result.Success))

	return &ToolCallResult{
		ToolCallID:   toolCallID,
		ToolName:     tool.Name(),
		FunctionName: functionName,
		Arguments:    arguments,
		Result:       result,
	}, nil
}

// shellOutputWatchTimeout 单条长命令输出 watch 的最长时长。
const shellOutputWatchTimeout = 15 * time.Minute

// StopShellWatch 停止正在进行的 shell 输出 watch。
// flow 到达终态/任务收尾时必须调用：事件通道随即关闭，
// 存活的 watcher 向其发送会触发 send-on-closed-channel panic。
func (a *BaseAgent) StopShellWatch() {
	a.shellWatchMu.Lock()
	defer a.shellWatchMu.Unlock()
	a.stopShellWatchLocked()
}

func (a *BaseAgent) stopShellWatchLocked() {
	if a.shellWatchCancel != nil {
		a.shellWatchCancel()
	}
	if a.shellWatchDone != nil {
		<-a.shellWatchDone
	}
	a.shellWatchCancel = nil
	a.shellWatchDone = nil
}

// startShellWatch 在 shell exec 返回 running 后启动后台轮询：
// 周期性读取沙箱控制台记录并以 ShellOutputEvent 推送增量快照，
// 进程结束后推一次最终快照再退出。新 watch 会顶掉旧 watch。
func (a *BaseAgent) startShellWatch(ctx context.Context, sandbox sandbox.Sandbox, sessionID string) {
	a.shellWatchMu.Lock()
	defer a.shellWatchMu.Unlock()

	// 顶掉旧 watch：同一会话同一时刻只跟踪最新一条长命令
	a.stopShellWatchLocked()
	watchCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	a.shellWatchCancel = cancel
	a.shellWatchDone = done

	go func() {
		defer func() {
			cancel()
			close(done)
		}()
		deadline := time.Now().Add(shellOutputWatchTimeout)
		var lastSnap string
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-time.After(1500 * time.Millisecond):
			}
			if time.Now().After(deadline) {
				return
			}

			res, err := sandbox.ReadShellOutput(watchCtx, sessionID, true)
			if err != nil || res == nil || !res.Success {
				// 会话尚未产出记录等情况：下一轮重试
				continue
			}
			if data, ok := res.Data.(map[string]interface{}); ok {
				if console, ok := data["console_records"].([]interface{}); ok {
					snap := fmt.Sprintf("%v", console)
					if snap != lastSnap {
						lastSnap = snap
						records := make([]map[string]interface{}, 0, len(console))
						for _, rec := range console {
							if m, ok := rec.(map[string]interface{}); ok {
								records = append(records, m)
							}
						}
						a.emitEvent(watchCtx, model.NewShellOutputEvent(sessionID, records))
					}
				}
			}

			// 完成检测：WaitProcess 成功（非超时错误）说明进程已结束，
			// 推一次最终快照后退出。
			// 1s 等待窗口兼做轮询间隔：进程结束则收尾，超时（BadRequest）继续下一轮
			waitSecs := 1
			if wp, err := sandbox.WaitProcess(watchCtx, sessionID, &waitSecs); err == nil && wp != nil && wp.Success {
				if res, err := sandbox.ReadShellOutput(watchCtx, sessionID, true); err == nil && res != nil && res.Success {
					if data, ok := res.Data.(map[string]interface{}); ok {
						if console, ok := data["console_records"].([]interface{}); ok {
							records := make([]map[string]interface{}, 0, len(console))
							for _, rec := range console {
								if m, ok := rec.(map[string]interface{}); ok {
									records = append(records, m)
								}
							}
							a.emitEvent(watchCtx, model.NewShellOutputEvent(sessionID, records))
						}
					}
				}
				return
			}
		}
	}()
}
