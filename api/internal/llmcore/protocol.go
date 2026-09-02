// Package llmcore 是 LLM 调用协议的"内核层"。
//
// 设计原则（来自 MULTI_LLM_ADAPTER_DESIGN.md）：
//   - 业务层（agent、orchestrator）只看到这套类型，不接触任何厂商协议
//   - Adapter（openai/anthropic）把厂商响应归一化到这套类型
//   - 类型设计"足够完整但不过度抽象"：先满足当前业务能落地，未来扩展以新增字段而非推倒重来
//
// 与 external 包的关系：
//   - external.LLM interface 仍存在；它的内部实现改用 llmcore.LLMRequest / LLMResponse
//   - 业务侧看到的是 external 包，但底下的 wire format 是 llmcore
package llmcore

// ContentPart 消息内容片段
// 多模态场景下 messages[i].content 是 []ContentPart；纯文本场景下是 string
// 为简化，V1 让 messages 元素用统一 Message.ContentText / ContentParts 两种字段
// 由 Adapter 在转厂商格式时选择合适的形状
type ContentPart struct {
	Type     string      `json:"type"`     // "text" / "image_url" / "audio_url" / ...
	Text     string      `json:"text,omitempty"`
	ImageURL *ImageURL   `json:"image_url,omitempty"`
	AudioURL *AudioURL   `json:"audio_url,omitempty"`
	Data     interface{} `json:"data,omitempty"` // 兜底字段，Adapter 按 type 自解析
}

type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto"/"low"/"high"
}

type AudioURL struct {
	URL string `json:"url"`
}

// MessageRole 消息角色
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// Message 统一消息结构
// Assistant 的 ToolCalls / Reasoning 字段允许为空
type Message struct {
	Role        MessageRole `json:"role"`
	// ContentText 纯文本消息用这个字段（最常见）
	ContentText string       `json:"content_text,omitempty"`
	// ContentParts 多模态消息用这个字段
	ContentParts []ContentPart `json:"content_parts,omitempty"`

	// Name 工具消息场景标识是哪个 tool；可空
	Name string `json:"name,omitempty"`

	// 助手消息专用
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// 推理过程（reasoning 模型的思考内容）
	// 注意：Agent 不会把 reasoning 直接展示给用户，但可审计
	Reasoning string `json:"reasoning,omitempty"`

	// 工具结果消息专用
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall 模型返回的工具调用请求
type ToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // 通常 "function"
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// ToolSpec 工具规格（发给模型的）
// 区别于 internal/agent/tools.go 的本地 ToolInfo：
//   - ToolSpec 是不依赖具体 Go 函数签名的"协议级"描述
//   - 同一个本地 tool 可以映射成多个 ToolSpec（多 provider 多字段）
type ToolSpec struct {
	Type     string             `json:"type"` // "function"
	Function ToolSpecFunction   `json:"function"`
	ReadOnly bool               `json:"read_only,omitempty"` // 标注是否只读（影响 fallback 规则）
}

type ToolSpecFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
	Strict      bool                   `json:"strict,omitempty"`
}

// LLMRequest 一次 LLM 调用的统一请求
// V1 只放"当前业务用得到的字段"，其他用 map[string]any 兜底
// 阶段 3 起，按"每加一种 provider 就多一个字段"的节奏补
type LLMRequest struct {
	Model    string    `json:"model"`           // 模型名
	Messages []Message `json:"messages"`        // 对话历史
	Tools    []ToolSpec `json:"tools,omitempty"` // 工具定义

	// ===== 生成控制 =====
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Stop        []string `json:"stop,omitempty"`

	// ===== 结构化输出 =====
	// ResponseFormat 在 OpenAI 协议上是 response_format 对象
	// 在 Anthropic 协议上由 Adapter 自行决定是否转化为 tool_use
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// ===== 流式 =====
	Stream bool `json:"stream"`

	// ===== 元信息（不发给上游）=====
	// Profile 在 Adapter 选择时被传入，用于按 model 能力调整请求
	Profile ModelProfile `json:"profile,omitempty"`
	// RouteStrategy 路由策略（影响重试/降级）
	RouteStrategy RouteStrategy `json:"route_strategy,omitempty"`
	// ProviderHint provider 协议提示（"openai_compat" / "anthropic" / "sensenova" / ...）
	// Adapter 决定如何把 LLMRequest 转成 provider 协议
	ProviderHint string `json:"provider_hint,omitempty"`
}

// ResponseFormat 响应结构约束
type ResponseFormat struct {
	Type       string      `json:"type"` // "text" / "json_object" / "json_schema"
	JSONSchema interface{} `json:"json_schema,omitempty"`
}

// LLMResponse 一次 LLM 调用的非流式响应
type LLMResponse struct {
	// ID 厂商响应 ID（用于排错）
	ID string `json:"id"`
	// Model 实际使用的模型名（可能有 routing/alias）
	Model string `json:"model"`

	// Message 单条助手消息
	Message Message `json:"message"`

	// FinishReason "stop" / "tool_calls" / "length" / "content_filter" / "error"
	FinishReason string `json:"finish_reason"`

	// Usage token 消耗
	Usage Usage `json:"usage"`

	// CostUSD 本次调用估算成本
	CostUSD float64 `json:"cost_usd"`

	// Raw 厂商原始响应（排错用，不入业务逻辑）
	Raw []byte `json:"raw,omitempty"`
}

// LLMDelta 流式响应的一个增量
// 一次 LLM 调用的流可能产生多个 LLMDelta
type LLMDelta struct {
	// ContentText 文本增量
	ContentText string `json:"content_text,omitempty"`
	// Reasoning 推理内容增量
	Reasoning string `json:"reasoning,omitempty"`

	// ToolCalls 工具调用增量（仅在 tool_call 起始或参数追加时出现）
	// Index 标识是同一个 tool_call 的第几段（用于跨 chunk 拼接 arguments）
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`

	// FinishReason 在流结束时出现
	FinishReason string `json:"finish_reason,omitempty"`

	// Usage 流结束时可能附带
	Usage *Usage `json:"usage,omitempty"`
}

type ToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	// ArgumentsDelta 增量 JSON 片段（OpenAI 风格）
	ArgumentsDelta string `json:"arguments_delta,omitempty"`
}

// Usage token 消耗统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	// ReasoningTokens 推理模型专用（DeepSeek / GLM thinking 模式）
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}
