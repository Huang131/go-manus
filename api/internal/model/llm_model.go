package model

import (
	"encoding/json"
	"time"
)

// LLMModel 一个 LLM 模型条目
// 支持多模型并存，agent 启动读取 is_default=true 的那一个
type LLMModel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`     // 用户自定义显示名 e.g. "我的 Claude"
	Provider    string    `json:"provider"` // openai/anthropic/google/deepseek/custom
	BaseURL     string    `json:"base_url"`
	APIKey      string    `json:"api_key,omitempty"`
	ModelName   string    `json:"model_name"` // 真实模型名 e.g. claude-3-5-sonnet-20241022
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Tags        []string  `json:"tags"` // 能力标签: vision/tools/long_ctx/...
	IsDefault   bool      `json:"is_default"`
	IsEnabled   bool      `json:"is_enabled"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// ===== 阶段 0 新增：能力画像 / 请求策略 / 成本策略 =====
	// 用 JSON 字段透传到 DB，repository 层负责序列化
	Capabilities   ModelCapabilities `json:"capabilities"`
	RequestPolicy RequestPolicy     `json:"request_policy"`
	CostPolicy    CostPolicy        `json:"cost_policy"`
}

// ModelCapabilities 模型能力画像
// 字段是 LLM Control Plane 决策依据
type ModelCapabilities struct {
	SupportsText                 bool `json:"supports_text"`
	SupportsToolCalls            bool `json:"supports_tool_calls"`
	SupportsStructuredOutput     bool `json:"supports_structured_output"`
	SupportsJSONMode             bool `json:"supports_json_mode"`
	SupportsStrictStructuredOutput bool `json:"supports_strict_structured_output"`
	SupportsStreaming            bool `json:"supports_streaming"`
	SupportsVision               bool `json:"supports_vision"`
	SupportsReasoning            bool `json:"supports_reasoning"`

	MaxContextTokens int `json:"max_context_tokens"`
	MaxOutputTokens  int `json:"max_output_tokens"`
}

// ReasoningMode 思考模式档位
type ReasoningMode string

const (
	ReasoningAuto ReasoningMode = "auto"
	ReasoningOff  ReasoningMode = "off"
	ReasoningLow  ReasoningMode = "low"
	ReasoningHigh ReasoningMode = "high"
)

// RequestPolicy 请求侧策略
// 作用：把"模型差异"收敛到配置，而不是 if-else 散在 Adapter
type RequestPolicy struct {
	// DefaultTemperature 默认温度（pointer：nil 表示用上游默认）
	DefaultTemperature *float64 `json:"default_temperature,omitempty"`
	// DefaultMaxTokens 默认 max_tokens（pointer：nil 表示用上游默认）
	DefaultMaxTokens *int    `json:"default_max_tokens,omitempty"`
	ReasoningMode    ReasoningMode `json:"reasoning_mode"`
	// Extra provider 白名单参数
	// 例：sensenova deepseek-v4-flash → {"reasoning_effort": "none"}
	// 只能由 Adapter 按白名单解析，不允许原样透传
	Extra map[string]json.RawMessage `json:"extra,omitempty"`
}

// CostPolicy 成本策略（美元 / 百万 token）
type CostPolicy struct {
	InputPricePerMTokens  float64 `json:"input_price_per_m_tokens"`
	OutputPricePerMTokens float64 `json:"output_price_per_m_tokens"`
	Currency              string  `json:"currency"` // USD / CNY
}

// EstimateCostUSD 估算一次调用的成本（美元）
// tokens 单位：个 token
func (c CostPolicy) EstimateCostUSD(promptTokens, completionTokens int) float64 {
	if c.InputPricePerMTokens == 0 && c.OutputPricePerMTokens == 0 {
		return 0
	}
	in := float64(promptTokens) / 1_000_000.0 * c.InputPricePerMTokens
	out := float64(completionTokens) / 1_000_000.0 * c.OutputPricePerMTokens
	return in + out
}

// LLMModelListResponse 列表接口返回
type LLMModelListResponse struct {
	Models []*LLMModel `json:"models"`
}

// DefaultCapabilities 新建模型时的能力画像兜底
// 取"主流 OpenAI 兼容模型都具备的能力子集"
func DefaultCapabilities() ModelCapabilities {
	return ModelCapabilities{
		SupportsText:                   true,
		SupportsToolCalls:              true,
		SupportsStructuredOutput:       true,
		SupportsJSONMode:               true,
		SupportsStrictStructuredOutput: false, // 严格模式仅 OpenAI 较新模型有
		SupportsStreaming:              true,
		SupportsVision:                 false,
		SupportsReasoning:              false,
		MaxContextTokens:               128000,
		MaxOutputTokens:                8192,
	}
}
