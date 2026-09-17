package external

import "time"

const (
	openAIChatCompletionsPath = "/chat/completions"
	anthropicMessagesPath     = "/v1/messages"

	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicVersion = "2023-06-01"
	defaultA2AHTTPTimeout   = 10 * time.Minute
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

// 搜索 provider 协议值：Tavily / Bocha 的厂商枚举值。
const (
	tavilySearchDepthBasic = "basic"
	tavilyTopicGeneral     = "general"
	tavilyTimeRangeDay     = "day"
	tavilyTimeRangeWeek    = "week"
	tavilyTimeRangeMonth   = "month"
	tavilyTimeRangeYear    = "year"

	bochaFreshnessOneDay   = "oneDay"
	bochaFreshnessOneWeek  = "oneWeek"
	bochaFreshnessOneMonth = "oneMonth"
	bochaFreshnessOneYear  = "oneYear"
)

// MCP JSON-RPC 和初始化协议值。
const (
	mcpJSONRPCVersion   = "2.0"
	mcpProtocolVersion  = "2024-11-05"
	mcpMethodInitialize = "initialize"
	mcpMethodListTools  = "tools/list"
	mcpMethodCallTool   = "tools/call"
	mcpClientName       = "go-manus"
	mcpClientVersion    = "1.0.0"
	mcpContentTypeText  = "text"
)

// A2A JSON-RPC 协议值。
const (
	a2aAgentCardPath     = "/.well-known/agent-card.json"
	a2aJSONRPCVersion    = "2.0"
	a2aMethodMessageSend = "message/send"
	a2aRoleUser          = "user"
	a2aPartKindText      = "text"
)

// 常用 HTTP MIME 类型（Content-Type / Accept 头取值）。
const (
	ContentTypeJSON = "application/json"
	ContentTypeSSE  = "text/event-stream"
)

// 搜索客户端通用运行参数。
const (
	// defaultSearchHTTPTimeout 是搜索客户端缺省的 HTTP 请求超时。
	defaultSearchHTTPTimeout = 30 * time.Second
	// authBearerPrefix 是 Authorization 头 Bearer 认证方案的固定前缀。
	authBearerPrefix = "Bearer "
)
