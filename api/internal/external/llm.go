package external

import (
	"context"
)

// LLMRequest LLM 请求参数
type LLMRequest struct {
	Messages       []map[string]interface{} `json:"messages"`
	Tools          []map[string]interface{} `json:"tools,omitempty"`
	ResponseFormat map[string]interface{}   `json:"response_format,omitempty"`
	ToolChoice     string                   `json:"tool_choice,omitempty"`
}

// LLMResponse LLM 响应
type LLMResponse struct {
	ID      string                   `json:"id"`
	Content string                   `json:"content"`
	ToolUse []map[string]interface{} `json:"tool_calls,omitempty"`
}

// LLM LLM 接口
type LLM interface {
	// Invoke 调用 LLM
	Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// ModelName 返回模型名称
	ModelName() string

	// Temperature 返回温度参数
	Temperature() float64

	// MaxTokens 返回最大 token 数
	MaxTokens() int
}

// LLMConfigProvider 每次调用前获取最新 LLM 配置。
// 返回 (nil, nil) 表示当前无 DB 配置，DynamicLLM 应回退到 fallback。
// 返回 error 表示读取失败，DynamicLLM 同样回退到 fallback。
type LLMConfigProvider func(ctx context.Context) (*OpenAIClientConfig, error)

// DynamicLLM 动态配置 LLM 客户端。
//
// 每次 Invoke 前通过 provider 获取最新配置（例如从数据库 app_configs 表读取），
// 让网页上的模型配置真正生效；当 DB 无配置或读取失败时，回退到环境变量 fallback。
//
// ModelName/Temperature/MaxTokens 三个 getter 仅返回 fallback 值，
// 生产代码不依赖它们做运行时决策（仅在测试中使用）。
type DynamicLLM struct {
	provider LLMConfigProvider
	fallback *OpenAIClientConfig
}

// NewDynamicLLM 创建动态配置 LLM 客户端。
func NewDynamicLLM(provider LLMConfigProvider, fallback *OpenAIClientConfig) *DynamicLLM {
	if fallback == nil {
		fallback = &OpenAIClientConfig{}
	}
	return &DynamicLLM{provider: provider, fallback: fallback}
}

// Invoke 调用 LLM，调用前先加载最新配置。
func (d *DynamicLLM) Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	cfg := d.fallback
	if d.provider != nil {
		if c, err := d.provider(ctx); err == nil && c != nil && c.BaseURL != "" {
			cfg = c
		}
	}
	return NewOpenAIClient(cfg).Invoke(ctx, req)
}

// ModelName 返回模型名称（fallback）。
func (d *DynamicLLM) ModelName() string { return d.fallback.ModelName }

// Temperature 返回温度参数（fallback）。
func (d *DynamicLLM) Temperature() float64 { return d.fallback.Temperature }

// MaxTokens 返回最大 token 数（fallback）。
func (d *DynamicLLM) MaxTokens() int { return d.fallback.MaxTokens }
