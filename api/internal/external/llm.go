package external

import (
	"context"
	"fmt"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
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

// llmStreamOverallTimeout 单条流式响应的整体截止时间。
// 远大于正常回复时长，只兜底"上游挂起、再也不吐增量也不结束"的场景，
// 避免任务在无 deadline 的 ctx 上永久挂起。
const llmStreamOverallTimeout = 10 * time.Minute

// streamContext 为流式请求派生带整体截止的 context，保留父 ctx 的取消语义。
func streamContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, llmStreamOverallTimeout)
}

// LLMRequest LLM 请求参数（阶段 1d：Messages / Tools / ResponseFormat 改用 llmcore 强类型）
//
// 业务侧只看到 llmcore 类型，看不到任何厂商协议；Adapter 负责把 llmcore 转 wire format。
type LLMRequest struct {
	Messages       []llmcore.Message       `json:"messages"`
	Tools          []llmcore.ToolSpec      `json:"tools,omitempty"`
	ResponseFormat *llmcore.ResponseFormat `json:"response_format,omitempty"`
	ToolChoice     string                  `json:"tool_choice,omitempty"`
}

// LLM LLM 接口
type LLM interface {
	// Invoke 调用 LLM，响应统一为 llmcore.LLMResponse 强类型
	Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error)

	// ModelName 返回模型名称
	ModelName() string

	// Temperature 返回温度参数
	Temperature() float64

	// MaxTokens 返回最大 token 数
	MaxTokens() int
}

// StreamingLLM 是可选的 token 流式能力。
// 保持为独立接口，避免要求所有已有 LLM mock 和非流式适配器同步实现。
type StreamingLLM interface {
	LLM
	Stream(ctx context.Context, req *LLMRequest) (<-chan llmcore.LLMDelta, error)
}

// LLMConfigProvider 每次调用前获取最新 LLM 配置。
// 返回 (nil, nil) 表示当前无 DB 配置，DynamicLLM 应回退到 fallback。
// 返回 error 表示读取失败，DynamicLLM 同样回退到 fallback。
type LLMConfigProvider func(ctx context.Context) (*LLMRuntimeConfig, error)

// DynamicLLM 动态配置 LLM 客户端。
//
// 每次 Invoke 前通过 provider 获取最新配置（例如从数据库 app_configs 表读取），
// 让网页上的模型配置真正生效；当 DB 无配置或读取失败时，回退到环境变量 fallback。
//
// ModelName/Temperature/MaxTokens 三个 getter 仅返回 fallback 值，
// 生产代码不依赖它们做运行时决策（仅在测试中使用）。
type DynamicLLM struct {
	provider LLMConfigProvider
	factory  LLMClientFactory
	fallback *LLMRuntimeConfig
}

var _ StreamingLLM = (*DynamicLLM)(nil)

// LLMClientFactory 根据运行时配置创建具体 LLM 客户端。
type LLMClientFactory func(cfg *LLMRuntimeConfig) LLM

// NewDynamicLLM 创建动态配置 LLM 客户端。
func NewDynamicLLM(provider LLMConfigProvider, fallback *LLMRuntimeConfig) *DynamicLLM {
	if fallback == nil {
		fallback = &LLMRuntimeConfig{
			Profile: llmcore.ModelProfile{Protocol: llmcore.ProtocolOpenAICompat},
		}
	}
	return &DynamicLLM{
		provider: provider,
		factory:  defaultLLMClientFactory,
		fallback: fallback,
	}
}

// NewDynamicLLMWithFactory 创建可注入工厂的动态配置客户端，便于测试和扩展。
func NewDynamicLLMWithFactory(provider LLMConfigProvider, fallback *LLMRuntimeConfig, factory LLMClientFactory) *DynamicLLM {
	if fallback == nil {
		fallback = &LLMRuntimeConfig{
			Profile: llmcore.ModelProfile{Protocol: llmcore.ProtocolOpenAICompat},
		}
	}
	if factory == nil {
		factory = defaultLLMClientFactory
	}
	return &DynamicLLM{
		provider: provider,
		factory:  factory,
		fallback: fallback,
	}
}

// Invoke 调用 LLM，调用前先加载最新配置。
//
// 阶段 1d 改造点：去掉 NormalizeLLMResponse 调用。
// Reasoning→Content 兜底是 agent 消费方（react_agent / planner_agent）的责任，
// 由调用方基于 LLMResponse.ReasoningContent 字段自行决定是否兜底。
// 协议层只保证"上游给什么字段就如实返回什么字段"。
func (d *DynamicLLM) Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
	cfg := d.fallback
	if d.provider != nil {
		if c, err := d.provider(ctx); err == nil && c != nil && c.BaseURL != "" {
			cfg = c
		}
	}
	if d.factory != nil {
		return d.factory(cfg).Invoke(ctx, req)
	}
	return NewOpenAIClient(runtimeConfigToOpenAIClientConfig(cfg)).Invoke(ctx, req)
}

// Stream 使用当前动态配置执行流式请求。
func (d *DynamicLLM) Stream(ctx context.Context, req *LLMRequest) (<-chan llmcore.LLMDelta, error) {
	cfg := d.fallback
	if d.provider != nil {
		if c, err := d.provider(ctx); err == nil && c != nil && c.BaseURL != "" {
			cfg = c
		}
	}
	factory := d.factory
	if factory == nil {
		factory = defaultLLMClientFactory
	}
	client := factory(cfg)
	if client == nil {
		return nil, fmt.Errorf("llm client factory returned nil")
	}
	streaming, ok := client.(StreamingLLM)
	if !ok {
		return nil, fmt.Errorf("llm client %q does not support streaming", client.ModelName())
	}
	return streaming.Stream(ctx, req)
}

// ModelName 返回模型名称（fallback）。
func (d *DynamicLLM) ModelName() string { return d.fallback.ModelName }

// Temperature 返回温度参数（fallback）。
func (d *DynamicLLM) Temperature() float64 { return d.fallback.Temperature }

// MaxTokens 返回最大 token 数（fallback）。
func (d *DynamicLLM) MaxTokens() int { return d.fallback.MaxTokens }

func defaultLLMClientFactory(cfg *LLMRuntimeConfig) LLM {
	if cfg == nil {
		cfg = &LLMRuntimeConfig{
			Profile: llmcore.ModelProfile{Protocol: llmcore.ProtocolOpenAICompat},
		}
	}
	switch cfg.Profile.Protocol {
	case llmcore.ProtocolAnthropic:
		// 与 OpenAI 分支对齐：策略/成本/工具超时必须透传，
		// 否则 DB 配置的 Claude 模型会丢失 reasoning 模式、成本恒 0。
		return NewAnthropicClient(&AnthropicClientConfig{
			BaseURL:         cfg.BaseURL,
			APIKey:          cfg.APIKey,
			ModelName:       cfg.ModelName,
			Temperature:     cfg.Temperature,
			MaxTokens:       cfg.MaxTokens,
			ToolCallTimeout: cfg.ToolCallTimeout,
			RequestPolicy:   cfg.Profile.RequestPolicy,
			CostPolicy:      cfg.Profile.CostPolicy,
			Version:         cfg.AnthropicVersion,
		})
	default:
		return NewOpenAIClient(runtimeConfigToOpenAIClientConfig(cfg))
	}
}
