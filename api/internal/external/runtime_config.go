package external

import (
	"encoding/json"
	"strings"

	"github.com/mooc-manus/go-manus/api/internal/llmcore"
	"github.com/mooc-manus/go-manus/api/internal/model"
)

// LLMRuntimeConfig 运行时模型配置。
//
// 这里承载的是“可用于实际发请求”的信息，而不是单纯的数据库字段。
// DynamicLLM 会基于它选择具体适配器。
type LLMRuntimeConfig struct {
	Profile          llmcore.ModelProfile
	BaseURL          string
	APIKey           string
	ModelName        string
	Temperature      float64
	MaxTokens        int
	ToolCallTimeout  int
	AnthropicVersion string
	Health           LLMRuntimeHealth
}

// LLMRuntimeHealth 路由时的轻量健康摘要。
// 这里只收口“能不能优先选”，不承担完整观测。
type LLMRuntimeHealth struct {
	Status           string
	RecentFailures   int
	AverageLatencyMS int
}

const (
	LLMHealthHealthy   = "healthy"
	LLMHealthDegraded  = "degraded"
	LLMHealthUnhealthy = "unhealthy"
)

// ProtocolFromProvider 把数据库里的 provider 字段映射成协议类型。
//
// 当前阶段只显式区分 Anthropic 与 OpenAI 兼容协议。
// 其他 provider 先按 OpenAI 兼容处理，避免把未落地的协议假装支持。
func ProtocolFromProvider(provider, modelName, baseURL string) llmcore.ProviderProtocol {
	provider = strings.ToLower(strings.TrimSpace(provider))
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	baseURL = strings.ToLower(strings.TrimSpace(baseURL))

	switch {
	case provider == "anthropic":
		return llmcore.ProtocolAnthropic
	case strings.HasPrefix(modelName, "claude"):
		return llmcore.ProtocolAnthropic
	case strings.Contains(baseURL, "anthropic.com"):
		return llmcore.ProtocolAnthropic
	default:
		return llmcore.ProtocolOpenAICompat
	}
}

// BuildRuntimeConfigFromModel 把数据库模型条目转换成运行时配置。
//
// 这一步把 model 层的字段收口成 llmcore / external 可直接消费的结构，
// 避免 main.go 里散落一堆手工字段拷贝。
func BuildRuntimeConfigFromModel(m *model.LLMModel, toolCallTimeout int) *LLMRuntimeConfig {
	if m == nil {
		return nil
	}
	policy := llmcore.RequestPolicy{
		DefaultTemperature: m.RequestPolicy.DefaultTemperature,
		DefaultMaxTokens:   m.RequestPolicy.DefaultMaxTokens,
		ReasoningMode:      llmcore.ReasoningMode(m.RequestPolicy.ReasoningMode),
		Extra:              convertRequestPolicyExtra(m.RequestPolicy.Extra),
	}
	effectiveTemperature := m.Temperature
	if effectiveTemperature == 0 && policy.DefaultTemperature != nil {
		effectiveTemperature = *policy.DefaultTemperature
	}
	effectiveMaxTokens := m.MaxTokens
	if effectiveMaxTokens == 0 && policy.DefaultMaxTokens != nil {
		effectiveMaxTokens = *policy.DefaultMaxTokens
	}
	costPolicy := llmcore.CostPolicy{
		InputPricePerMTokens:  m.CostPolicy.InputPricePerMTokens,
		OutputPricePerMTokens: m.CostPolicy.OutputPricePerMTokens,
		Currency:              m.CostPolicy.Currency,
	}

	return &LLMRuntimeConfig{
		Profile: llmcore.ModelProfile{
			ID:        m.ID,
			Protocol:  ProtocolFromProvider(m.Provider, m.ModelName, m.BaseURL),
			BaseURL:   m.BaseURL,
			APIKey:    m.APIKey,
			ModelName: m.ModelName,
			Capabilities: llmcore.ModelCapabilities{
				SupportsText:                   m.Capabilities.SupportsText,
				SupportsToolCalls:              m.Capabilities.SupportsToolCalls,
				SupportsStructuredOutput:       m.Capabilities.SupportsStructuredOutput,
				SupportsJSONMode:               m.Capabilities.SupportsJSONMode,
				SupportsStrictStructuredOutput: m.Capabilities.SupportsStrictStructuredOutput,
				SupportsStreaming:              m.Capabilities.SupportsStreaming,
				SupportsVision:                 m.Capabilities.SupportsVision,
				SupportsReasoning:              m.Capabilities.SupportsReasoning,
				MaxContextTokens:               m.Capabilities.MaxContextTokens,
				MaxOutputTokens:                m.Capabilities.MaxOutputTokens,
			},
			RequestPolicy: policy,
			CostPolicy:    costPolicy,
		},
		BaseURL:         m.BaseURL,
		APIKey:          m.APIKey,
		ModelName:       m.ModelName,
		Temperature:     effectiveTemperature,
		MaxTokens:       effectiveMaxTokens,
		ToolCallTimeout: toolCallTimeout,
		Health:          runtimeHealthFromModel(m.RuntimeHealth),
	}
}

func convertRequestPolicyExtra(in map[string]json.RawMessage) map[string]llmcore.ExtraParam {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]llmcore.ExtraParam, len(in))
	for k, raw := range in {
		if len(raw) == 0 {
			continue
		}
		out[k] = llmcore.ExtraParam{
			Kind: "json",
			Raw:  append(json.RawMessage(nil), raw...),
		}
	}
	return out
}

func runtimeConfigToOpenAIClientConfig(cfg *LLMRuntimeConfig) *OpenAIClientConfig {
	if cfg == nil {
		return &OpenAIClientConfig{}
	}
	return &OpenAIClientConfig{
		BaseURL:         cfg.BaseURL,
		APIKey:          cfg.APIKey,
		ModelName:       cfg.ModelName,
		Temperature:     cfg.Temperature,
		MaxTokens:       cfg.MaxTokens,
		ToolCallTimeout: cfg.ToolCallTimeout,
		RequestPolicy:   cfg.Profile.RequestPolicy,
		CostPolicy:      cfg.Profile.CostPolicy,
	}
}

func runtimeHealthFromModel(h model.RuntimeHealth) LLMRuntimeHealth {
	status := h.Status
	if status == "" {
		status = LLMHealthHealthy
	}
	return LLMRuntimeHealth{
		Status:           status,
		RecentFailures:   h.RecentFailures,
		AverageLatencyMS: h.AverageLatencyMS,
	}
}

func runtimeHealthToModel(h LLMRuntimeHealth) model.RuntimeHealth {
	status := h.Status
	if status == "" {
		status = LLMHealthHealthy
	}
	return model.RuntimeHealth{
		Status:           status,
		RecentFailures:   h.RecentFailures,
		AverageLatencyMS: h.AverageLatencyMS,
	}
}
