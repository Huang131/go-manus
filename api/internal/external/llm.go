package external

import (
	"context"

	"github.com/mooc-manus/go-manus/api/internal/llmcore"
)

// llmModelIDKey ctx 中携带"本次请求要用的模型 ID"的 key。
// DynamicLLM 的 provider 闭包会从 ctx 读这个 key，
// 用于实现"单次 chat 临时切换模型"而无需改 default。
type llmModelIDKey struct{}

// WithModelID 返回带 modelID 的 ctx。空 modelID 等于未设置（走 default）。
func WithModelID(ctx context.Context, modelID string) context.Context {
	if modelID == "" {
		return ctx
	}
	return context.WithValue(ctx, llmModelIDKey{}, modelID)
}

// ModelIDFromContext 读取 ctx 中的 modelID，未设置时返回空字符串。
func ModelIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(llmModelIDKey{}).(string); ok {
		return v
	}
	return ""
}

// LLMRequest LLM 请求参数（阶段 1d：Messages / Tools / ResponseFormat 改用 llmcore 强类型）
//
// 业务侧只看到 llmcore 类型，看不到任何厂商协议；Adapter 负责把 llmcore 转 wire format。
type LLMRequest struct {
	Messages       []llmcore.Message    `json:"messages"`
	Tools          []llmcore.ToolSpec   `json:"tools,omitempty"`
	ResponseFormat *llmcore.ResponseFormat `json:"response_format,omitempty"`
	ToolChoice     string               `json:"tool_choice,omitempty"`
}

// LLMResponse LLM 响应（阶段 1d：ToolUse 改 []llmcore.ToolCall）
type LLMResponse struct {
	ID               string            `json:"id"`
	Content          string            `json:"content"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	RawContent       string            `json:"raw_content,omitempty"`
	ToolUse          []llmcore.ToolCall `json:"tool_calls,omitempty"`
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
//
// 阶段 1d 改造点：去掉 NormalizeLLMResponse 调用。
// Reasoning→Content 兜底是 agent 消费方（react_agent / planner_agent）的责任，
// 由调用方基于 LLMResponse.ReasoningContent 字段自行决定是否兜底。
// 协议层只保证"上游给什么字段就如实返回什么字段"。
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