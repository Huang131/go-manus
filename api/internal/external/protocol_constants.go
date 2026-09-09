package external

import "time"

const (
	openAIChatCompletionsPath = "/chat/completions"
	anthropicMessagesPath     = "/v1/messages"

	defaultAnthropicBaseURL    = "https://api.anthropic.com"
	defaultAnthropicVersion    = "2023-06-01"
	defaultExternalHTTPTimeout = 120 * time.Second
	defaultA2AHTTPTimeout      = 10 * time.Minute
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
