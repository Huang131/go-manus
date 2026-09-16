package llmcore

import "github.com/Huang131/go-manus/api/internal/model"

// ProviderProtocol 上游 provider 协议分类
// Adapter 选择就是基于这个字段
type ProviderProtocol string

const (
	ProtocolOpenAICompat ProviderProtocol = "openai_compat" // 多数中转站走这条
	ProtocolAnthropic    ProviderProtocol = "anthropic"
	ProtocolGemini       ProviderProtocol = "gemini"
	ProtocolCustom       ProviderProtocol = "custom"
)

// RouteStrategy 路由策略（控制面/Orchestrator 决定）
type RouteStrategy string

const (
	RoutePrimary    RouteStrategy = "primary"     // 主模型
	RouteFallback   RouteStrategy = "fallback"    // 主模型失败后用
	RouteUserPicked RouteStrategy = "user_picked" // 用户在 UI 临时指定
)

// ModelProfile 模型"画像"——Adapter 决定如何转换协议的唯一依据
// 来源：DB capabilities/request_policy 字段
type ModelProfile struct {
	// ID 模型在 DB 中的 ID
	ID string `json:"id"`
	// ProviderProtocol 协议
	Protocol ProviderProtocol `json:"protocol"`
	// BaseURL API 地址
	BaseURL string `json:"base_url"`
	// APIKey 鉴权
	APIKey string `json:"api_key,omitempty"`
	// ModelName 真实模型名
	ModelName string `json:"model_name"`

	// Capabilities 能力画像
	Capabilities model.ModelCapabilities `json:"capabilities"`
	// RequestPolicy 请求策略
	RequestPolicy RequestPolicy `json:"request_policy"`
	// CostPolicy 成本策略
	CostPolicy model.CostPolicy `json:"cost_policy"`
}

// RequestPolicy 请求侧策略
type RequestPolicy struct {
	DefaultTemperature *float64            `json:"default_temperature,omitempty"`
	DefaultMaxTokens   *int                `json:"default_max_tokens,omitempty"`
	ReasoningMode      model.ReasoningMode `json:"reasoning_mode"`
	// Extra provider 白名单参数
	// Adapter 必须按自身 allowlist 解析，**不允许原样透传**
	Extra map[string]ExtraParam `json:"extra,omitempty"`
}

// ExtraParam provider 额外参数
// 用 []byte 包一层，避免在协议层就把所有 provider 参数预先定义
type ExtraParam struct {
	// Kind "string" / "number" / "bool" / "json"
	// Adapter 收到 Extra 时按 Kind 反序列化
	Kind string `json:"kind"`
	Raw  []byte `json:"raw"`
}
