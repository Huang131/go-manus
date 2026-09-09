package external

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
)

// OpenAIClient OpenAI 兼容 API 客户端
type OpenAIClient struct {
	baseURL         string
	apiKey          string
	modelName       string
	temperature     float64
	maxTokens       int
	toolCallTimeout time.Duration // tool calling 请求超时
	requestPolicy   llmcore.RequestPolicy
	costPolicy      llmcore.CostPolicy
	httpClient      *http.Client
}

// OpenAIClientConfig OpenAI 客户端配置
type OpenAIClientConfig struct {
	BaseURL         string                `mapstructure:"base_url"`
	APIKey          string                `mapstructure:"api_key"`
	ModelName       string                `mapstructure:"model_name"`
	Temperature     float64               `mapstructure:"temperature"`
	MaxTokens       int                   `mapstructure:"max_tokens"`
	ToolCallTimeout int                   `mapstructure:"tool_call_timeout"` // tool calling 请求超时秒数，默认 15
	RequestPolicy   llmcore.RequestPolicy `mapstructure:"request_policy"`
	CostPolicy      llmcore.CostPolicy    `mapstructure:"cost_policy"`
}

func (c *OpenAIClientConfig) setDefaults() {
	if c.ToolCallTimeout == 0 {
		c.ToolCallTimeout = 15
	}
}

// NewOpenAIClient 创建 OpenAI 客户端
func NewOpenAIClient(cfg *OpenAIClientConfig) *OpenAIClient {
	cfg.setDefaults()
	return &OpenAIClient{
		baseURL:         cfg.BaseURL,
		apiKey:          cfg.APIKey,
		modelName:       cfg.ModelName,
		temperature:     cfg.Temperature,
		maxTokens:       cfg.MaxTokens,
		toolCallTimeout: time.Duration(cfg.ToolCallTimeout) * time.Second,
		requestPolicy:   cfg.RequestPolicy,
		costPolicy:      cfg.CostPolicy,
		httpClient: &http.Client{
			Timeout: defaultExternalHTTPTimeout,
		},
	}
}

// openAIChatRequest OpenAI Chat API 请求 wire format
type openAIChatRequest struct {
	Model           string                  `json:"model"`
	Messages        []openAIMessage         `json:"messages"`
	Tools           []openAIToolSpec        `json:"tools,omitempty"`
	ToolChoice      interface{}             `json:"tool_choice,omitempty"`
	Temperature     *float64                `json:"temperature,omitempty"`
	MaxTokens       *int                    `json:"max_tokens,omitempty"`
	Stream          bool                    `json:"stream,omitempty"`
	ResponseFormat  *llmcore.ResponseFormat `json:"response_format,omitempty"`
	ReasoningEffort *string                 `json:"reasoning_effort,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIToolSpec struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Strict      bool                   `json:"strict,omitempty"`
}

// openAIChatResponse OpenAI Chat API 响应 wire format
// 注意：上游 reason 模型字段可能是 reasoning_content（DeepSeek / GMI / MiniMax）
// 或 reasoning（Anthropic 风格），两者都解析到 llmcore.Message.Reasoning
type openAIChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			// 兼容 reasoning_content（DeepSeek / GMI / sensenova） 和 reasoning（部分 provider）两种命名
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
			Refusal          string `json:"refusal"`
			ToolCalls        []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens      int `json:"prompt_tokens"`
		CompletionTokens  int `json:"completion_tokens"`
		TotalTokens       int `json:"total_tokens"`
		ReasoningTokens   int `json:"reasoning_tokens,omitempty"`
		CompletionDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Message    string `json:"message"`
		Type       string `json:"type"`
		Code       string `json:"code"`
		RetryAfter int    `json:"retry_after"`
	} `json:"error,omitempty"`
}

// Invoke 调用 OpenAI Chat API
//
// 阶段 1d 改造点：
//  1. 入参 Messages / Tools / ResponseFormat 用 llmcore 强类型，直接 marshal
//  2. 内部按 llmcore 协议解析上游响应（Content / ReasoningContent / ToolCalls 严格分离）
//  3. 去掉"content 为空时把 reasoning 当 content 兜底"的违规逻辑
//     —— Reasoning 不再覆盖 Content；调用方读 ReasoningContent 字段
//  4. 上游错误按 401/403/429/5xx/timeout 分类为 llmcore.ProviderError，
//     为阶段 3 fallback 矩阵提供 ErrorKind 钩子
//  5. 对外返回 llmcore.LLMResponse（与 anthropic adapter 一致的统一响应形状）
func (c *OpenAIClient) Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
	// 构建 wire format 请求
	temp := c.effectiveTemperature()
	maxTok := c.effectiveMaxTokens()
	chatReq := openAIChatRequest{
		Model:          c.modelName,
		Messages:       toOpenAIMessages(req.Messages),
		Tools:          toOpenAITools(req.Tools),
		Temperature:    &temp,
		MaxTokens:      &maxTok,
		ResponseFormat: req.ResponseFormat,
	}
	chatReq.ReasoningEffort = c.effectiveReasoningEffort()
	if req.ToolChoice != "" {
		chatReq.ToolChoice = req.ToolChoice
	}

	// 序列化请求
	reqBody, err := sonic.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 构建 HTTP 请求上下文：tool calling 请求使用独立超时（默认 15s），
	// 不依赖硬编码模型黑名单——任何不支持 tool calling 的模型都会在此时超时失败，
	// 而不是等到全局 120s 才暴露。
	httpCtx := ctx
	if len(req.Tools) > 0 {
		var cancel context.CancelFunc
		httpCtx, cancel = context.WithTimeout(ctx, c.toolCallTimeout)
		defer cancel()
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, c.baseURL+openAIChatCompletionsPath, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// tool calling 超时特殊处理
		if len(req.Tools) > 0 && errors.Is(err, context.DeadlineExceeded) {
			pe := llmcore.NewProviderError(llmcore.KindTimeout, "openai_compat", c.modelName,
				fmt.Sprintf("tool calling 请求超时（%v），模型 %q 可能不支持 tool calling", c.toolCallTimeout, c.modelName))
			pe.StatusCode = http.StatusGatewayTimeout
			pe.Cause = err
			return nil, pe
		}
		if errors.Is(err, context.DeadlineExceeded) {
			pe := llmcore.NewProviderError(llmcore.KindTimeout, "openai_compat", c.modelName, "request timeout")
			pe.StatusCode = http.StatusGatewayTimeout
			pe.Cause = err
			return nil, pe
		}
		pe := llmcore.NewProviderError(llmcore.KindNetwork, "openai_compat", c.modelName, "network error")
		pe.Cause = err
		return nil, pe
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		pe := llmcore.NewProviderError(llmcore.KindNetwork, "openai_compat", c.modelName, "read response body")
		pe.StatusCode = resp.StatusCode
		pe.Cause = fmt.Errorf("read response: %w", err)
		return nil, pe
	}

	// 非 2xx：分类为 ProviderError，让阶段 3 fallback 决策有 ErrorKind
	if resp.StatusCode != http.StatusOK {
		return nil, c.classifyHTTPError(resp.StatusCode, respBody)
	}

	// 解析响应
	var chatResp openAIChatResponse
	if err := sonic.Unmarshal(respBody, &chatResp); err != nil {
		// 协议错误：熔断该模型（不重试不 fallback）
		pe := llmcore.NewProviderError(llmcore.KindUnknown, "openai_compat", c.modelName,
			fmt.Sprintf("unmarshal response: %v", err))
		pe.StatusCode = resp.StatusCode
		pe.Cause = err
		pe.Retryable = false
		pe.Fallbackable = false
		return nil, pe
	}

	if len(chatResp.Choices) == 0 {
		pe := llmcore.NewProviderError(llmcore.KindUnknown, "openai_compat", c.modelName, "no choices in response")
		pe.StatusCode = resp.StatusCode
		pe.Retryable = false
		pe.Fallbackable = false
		return nil, pe
	}

	// === llmcore 协议层归一化 ===
	choice := chatResp.Choices[0]

	// 推理字段归一化：reasoning_content（OpenAI 标准/DeepSeek） 和 reasoning（部分厂商） 任一非空都进 Reasoning
	reasoning := choice.Message.ReasoningContent
	if reasoning == "" {
		reasoning = choice.Message.Reasoning
	}

	// 工具调用归一化
	var toolCalls []llmcore.ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls = make([]llmcore.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			toolCalls[i] = llmcore.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: llmcore.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	// === 转为 llmcore.LLMResponse ===
	// Content 严格取 content，不被 reasoning 覆盖；Reasoning 单独携带
	out := &llmcore.LLMResponse{
		ID:    chatResp.ID,
		Model: chatResp.Model,
		Message: llmcore.Message{
			Role:        llmcore.RoleAssistant,
			ContentText: choice.Message.Content,
			Reasoning:   reasoning,
			ToolCalls:   toolCalls,
		},
		FinishReason: choice.FinishReason,
		Usage: llmcore.Usage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
			ReasoningTokens:  reasoningTokens(chatResp.Usage.ReasoningTokens, chatResp.Usage.CompletionDetails.ReasoningTokens),
		},
	}
	out.CostUSD = estimateCostUSD(c.costPolicy, out.Usage)
	return out, nil
}

func (c *OpenAIClient) effectiveTemperature() float64 {
	if c.requestPolicy.DefaultTemperature != nil {
		return *c.requestPolicy.DefaultTemperature
	}
	return c.temperature
}

func (c *OpenAIClient) effectiveMaxTokens() int {
	if c.requestPolicy.DefaultMaxTokens != nil {
		return *c.requestPolicy.DefaultMaxTokens
	}
	return c.maxTokens
}

func (c *OpenAIClient) effectiveReasoningEffort() *string {
	switch c.requestPolicy.ReasoningMode {
	case llmcore.ReasoningOff:
		v := "none"
		return &v
	case llmcore.ReasoningLow:
		v := "low"
		return &v
	case llmcore.ReasoningHigh:
		v := "high"
		return &v
	}

	if extra, ok := c.requestPolicy.Extra["reasoning_effort"]; ok {
		var v string
		if err := sonic.Unmarshal(extra.Raw, &v); err == nil && v != "" {
			return &v
		}
	}
	return nil
}

func reasoningTokens(topLevel, details int) int {
	if details != 0 {
		return details
	}
	return topLevel
}

func estimateCostUSD(policy llmcore.CostPolicy, usage llmcore.Usage) float64 {
	if policy.InputPricePerMTokens == 0 && policy.OutputPricePerMTokens == 0 {
		return 0
	}
	in := float64(usage.PromptTokens) / 1_000_000.0 * policy.InputPricePerMTokens
	out := float64(usage.CompletionTokens) / 1_000_000.0 * policy.OutputPricePerMTokens
	return in + out
}

func toOpenAIMessages(messages []llmcore.Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(messages))
	for _, msg := range messages {
		wire := openAIMessage{
			Role:       string(msg.Role),
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}
		if len(msg.ContentParts) > 0 {
			wire.Content = msg.ContentParts
		} else {
			wire.Content = msg.ContentText
		}
		if len(msg.ToolCalls) > 0 {
			wire.ToolCalls = make([]openAIToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				wire.ToolCalls = append(wire.ToolCalls, openAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openAIFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		out = append(out, wire)
	}
	return out
}

func toOpenAITools(tools []llmcore.ToolSpec) []openAIToolSpec {
	out := make([]openAIToolSpec, 0, len(tools))
	for _, t := range tools {
		out = append(out, openAIToolSpec{
			Type: t.Type,
			Function: openAIToolFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
				Strict:      t.Function.Strict,
			},
		})
	}
	return out
}

// classifyHTTPError 把 HTTP 状态码分类为 llmcore.ErrorKind
// 参考 MULTI_LLM_ADAPTER_DESIGN.md "错误分类"：
//   - 401/403 → auth         不重试不 fallback
//   - 429     → rate_limit   读 Retry-After 退避，必要时 fallback
//   - 5xx     → server       有限重试，失败后 fallback
//   - 4xx     → bad_request  不重试
//   - 其他    → unknown
func (c *OpenAIClient) classifyHTTPError(status int, body []byte) error {
	kind := llmcore.KindUnknown
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		kind = llmcore.KindAuth
	case status == http.StatusTooManyRequests:
		kind = llmcore.KindRateLimit
	case status == http.StatusRequestTimeout:
		kind = llmcore.KindTimeout
	case status == http.StatusNotFound:
		kind = llmcore.KindNotFound
	case status >= 500:
		kind = llmcore.KindServer
	case status >= 400:
		kind = llmcore.KindBadRequest
	}

	// 尝试从 body 提取上游 error.message
	var probe struct {
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	_ = sonic.Unmarshal(body, &probe)
	upstreamMsg := ""
	if probe.Error != nil {
		upstreamMsg = probe.Error.Message
	}
	if upstreamMsg == "" {
		upstreamMsg = truncateBody(body)
	}

	pe := llmcore.NewProviderError(kind, "openai_compat", c.modelName,
		fmt.Sprintf("status=%d: %s", status, upstreamMsg))
	pe.StatusCode = status
	// classifyHTTPError 的 kind 已经被 isRetryable/isFallbackable 决定
	// 这里用 NewProviderError 默认规则即可
	return pe
}

func truncateBody(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}

// ModelName 返回模型名称
func (c *OpenAIClient) ModelName() string {
	return c.modelName
}

// Temperature 返回温度参数
func (c *OpenAIClient) Temperature() float64 {
	return c.temperature
}

// MaxTokens 返回最大 token 数
func (c *OpenAIClient) MaxTokens() int {
	return c.maxTokens
}
