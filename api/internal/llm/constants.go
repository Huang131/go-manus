package llm

const (
	openAIChatCompletionsPath = "/chat/completions"
	anthropicMessagesPath     = "/v1/messages"

	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicVersion = "2023-06-01"
)

// Anthropic 内容块和思考模式协议值。
const (
	anthropicContentTypeText       = "text"
	anthropicContentTypeThinking   = "thinking"
	anthropicContentTypeToolUse    = "tool_use"
	anthropicContentTypeToolResult = "tool_result"
	anthropicContentTypeImage      = "image"
	anthropicImageSourceTypeURL    = "url"
	anthropicThinkingTypeDisabled  = "disabled"
	anthropicThinkingTypeEnabled   = "enabled"
)

// Anthropic stop_reason 原始值（响应体与 message_delta 事件共用）。
// 这些值只在 Adapter 层出现，出 Adapter 前会被归一化为 llmcore 的 canonical 值。
const (
	anthropicStopReasonEndTurn      = "end_turn"
	anthropicStopReasonStopSequence = "stop_sequence"
	anthropicStopReasonToolUse      = "tool_use"
	anthropicStopReasonMaxTokens    = "max_tokens"
)

// OpenAI 兼容协议的 finish_reason 原始值。
// stop/tool_calls/length/content_filter 与 llmcore canonical 值同名，
const (
	openAIFinishReasonFunctionCall = "function_call"
)

// OpenAI 兼容协议 SSE 流协议值。
const (
	openAISSEEndToken         = "[DONE]"
	openAIReasoningEffortNone = "none"
	openAIReasoningEffortLow  = "low"
	openAIReasoningEffortHigh = "high"
)

// Anthropic SSE 流事件类型（event: 行的取值）。
const (
	anthropicEventError             = "error"               // 错误事件
	anthropicEventMessageStart      = "message_start"       // 消息开始
	anthropicEventContentBlockStart = "content_block_start" // 新内容块出现（如 text/thinking/tool_use）
	anthropicEventContentBlockDelta = "content_block_delta" // 内容块增量
	anthropicEventMessageDelta      = "message_delta"       // 消息尾部增量
)

// Anthropic content_block_delta 的 delta.type 原始值。
const (
	anthropicDeltaTypeTextDelta     = "text_delta"       // 文本增量
	anthropicDeltaTypeThinkingDelta = "thinking_delta"   // 思考增量
	anthropicDeltaTypeInputJSON     = "input_json_delta" // JSON 参数增量
)

// === 通用超时与缓冲区配置 ===
const (
	// DefaultToolCallTimeout 工具调用默认超时（秒）
	// 用于快速识别不支持 tool calling 的模型（而非等待全局超时）
	DefaultToolCallTimeout = 15

	// ScannerBufferSize Scanner 初始 buffer 大小
	ScannerBufferSize = 4096

	// ScannerMaxBufferSize Scanner 最大 buffer（1MB）
	// 防止超长 SSE 行导致内存问题
	ScannerMaxBufferSize = 1024 * 1024
)

// === Anthropic 思考模式配置 ===
const (
	// ThinkingBudgetLow Low/Auto 模式思考预算（2K tokens）
	ThinkingBudgetLow = 2048

	// ThinkingBudgetHigh High 模式思考预算（8K tokens）
	ThinkingBudgetHigh = 8192

	// ThinkingMinBudget 思考模式最小启用预算
	// 小于此值时 Anthropic API 会拒绝请求
	ThinkingMinBudget = 1024
)

// === 路由权重配置 ===
const (
	// HealthRankUnknown 健康状态未知时的排名分值
	HealthRankUnknown = 3
)
