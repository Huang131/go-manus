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
// 仅旧版 function_call 需要翻译。
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
	anthropicEventError             = "error"
	anthropicEventMessageStart      = "message_start"
	anthropicEventContentBlockStart = "content_block_start"
	anthropicEventContentBlockDelta = "content_block_delta"
	anthropicEventMessageDelta      = "message_delta"
)

// Anthropic content_block_delta 的 delta.type 原始值。
const (
	anthropicDeltaTypeTextDelta     = "text_delta"
	anthropicDeltaTypeThinkingDelta = "thinking_delta"
	anthropicDeltaTypeInputJSON     = "input_json_delta"
)
