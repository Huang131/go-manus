package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/llm/sse"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/httpconst"
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
	costPolicy      model.CostPolicy
	httpClient      *http.Client
}

var _ StreamingLLM = (*OpenAIClient)(nil)

// OpenAIClientConfig OpenAI 客户端配置
type OpenAIClientConfig struct {
	BaseURL         string                `mapstructure:"base_url"`
	APIKey          string                `mapstructure:"api_key"`
	ModelName       string                `mapstructure:"model_name"`
	Temperature     float64               `mapstructure:"temperature"`
	MaxTokens       int                   `mapstructure:"max_tokens"`
	ToolCallTimeout int                   `mapstructure:"tool_call_timeout"` // tool calling 请求超时秒数，默认 15
	RequestPolicy   llmcore.RequestPolicy `mapstructure:"request_policy"`
	CostPolicy      model.CostPolicy      `mapstructure:"cost_policy"`
}

func (c *OpenAIClientConfig) setDefaults() {
	if c.ToolCallTimeout == 0 {
		c.ToolCallTimeout = DefaultToolCallTimeout
	}
}

// NewOpenAIClient 创建 OpenAI 客户端
func NewOpenAIClient(cfg *OpenAIClientConfig) *OpenAIClient {
	cfg.setDefaults()
	return &OpenAIClient{
		// 尾斜杠归一：配置 http://api.example.com/v1/ 时避免拼出 //chat/completions
		baseURL:         strings.TrimSuffix(cfg.BaseURL, "/"),
		apiKey:          cfg.APIKey,
		modelName:       cfg.ModelName,
		temperature:     cfg.Temperature,
		maxTokens:       cfg.MaxTokens,
		toolCallTimeout: time.Duration(cfg.ToolCallTimeout) * time.Second,
		requestPolicy:   cfg.RequestPolicy,
		costPolicy:      cfg.CostPolicy,
		httpClient:      &http.Client{},
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
	StreamOptions   *openAIStreamOptions    `json:"stream_options,omitempty"`
	ResponseFormat  *llmcore.ResponseFormat `json:"response_format,omitempty"`
	ReasoningEffort *string                 `json:"reasoning_effort,omitempty"`
}

// openAIStreamOptions OpenAI 流式请求选项。
// 严格 OpenAI 协议在流式模式下默认不返回 usage，必须显式开启 include_usage，
// 否则流式调用的 token 统计与成本恒为 0（DeepSeek/vLLM 等网关默认带，但不可依赖）。
type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
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
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// buildChatRequest 把 llmcore 请求转换为 OpenAI 兼容 wire format，Invoke/Stream 共用。
// stream=true 时额外设置 Stream 和 stream_options.include_usage
// （严格 OpenAI 协议流式默认不返回 usage，必须显式开启）。
func (c *OpenAIClient) buildChatRequest(req *LLMRequest, stream bool) openAIChatRequest {
	chatReq := openAIChatRequest{
		Model:          c.modelName,
		Messages:       toOpenAIMessages(req.Messages),
		Tools:          toOpenAITools(req.Tools),
		Temperature:    func() *float64 { t := c.effectiveTemperature(); return &t }(),
		ResponseFormat: req.ResponseFormat,
	}
	// max_tokens=0 表示未配置：必须省略字段而不是下发 0。
	// OpenAI 兼容协议要求 max_tokens >= 1，下发 0 会被上游 400 拒绝
	// （"max_tokens must be at least 1"）。
	if maxTok := c.effectiveMaxTokens(); maxTok > 0 {
		chatReq.MaxTokens = &maxTok
	}
	chatReq.ReasoningEffort = c.effectiveReasoningEffort()
	if req.ToolChoice != "" {
		chatReq.ToolChoice = req.ToolChoice
	}
	if stream {
		chatReq.Stream = true
		chatReq.StreamOptions = &openAIStreamOptions{IncludeUsage: true}
	}
	return chatReq
}

// Invoke 调用 OpenAI Chat API，返回统一的 llmcore.LLMResponse。
func (c *OpenAIClient) Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("llm request is nil")
	}

	reqBody, err := sonic.Marshal(c.buildChatRequest(req, false))
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
	httpReq.Header.Set("Content-Type", httpconst.ContentTypeJSON)
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", httpconst.AuthBearerPrefix+c.apiKey)
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// 超时归因：只有父 ctx 仍然存活时，DeadlineExceeded 才是 tool calling 子超时；
		// 父 ctx（如全局超时）先到期时不能归因为"模型不支持 tool calling"，否则误导排障。
		if len(req.Tools) > 0 && ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
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

	// 非 2xx：分类为 ProviderError，让阶段 3 fallback 决策有 ErrorKind。
	// 接受全部 2xx（部分网关用 201/204 转发成功响应）。
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, classifyHTTPError(resp.StatusCode, respBody, "openai_compat", c.modelName)
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

	// 部分兼容网关（one-api/new-api 等）用 200 + error body 转发上游错误。
	// 优先消费 error 字段，避免真实错误被 "no choices in response" 覆盖。
	if e := chatResp.Error; e != nil && e.Message != "" {
		pe := llmcore.NewProviderError(llmcore.KindServer, "openai_compat", c.modelName,
			fmt.Sprintf("上游在 200 响应中返回错误: %s (type=%s, code=%s)", e.Message, e.Type, e.Code))
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
			Role:        model.RoleAssistant,
			ContentText: choice.Message.Content,
			Reasoning:   reasoning,
			ToolCalls:   toolCalls,
		},
		FinishReason: normalizeOpenAIFinishReason(choice.FinishReason),
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

// normalizeOpenAIFinishReason 把 OpenAI 兼容协议的 finish_reason 翻译为 llmcore canonical 值。
// stop/tool_calls/length/content_filter 与 canonical 同名，只有旧版 function_call 需要翻译；
// 未知值原样透传，保留可观测性。
func normalizeOpenAIFinishReason(raw string) string {
	if raw == openAIFinishReasonFunctionCall {
		return llmcore.FinishReasonToolCalls
	}
	return raw
}

// openAIStreamChunk 是 OpenAI 兼容 SSE 的单个 data JSON。
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		// 与非流式 openAIChatResponse.Usage 相同的双形状兼容：
		// reasoning_tokens（DeepSeek / GMI 顶层）与 completion_tokens_details.reasoning_tokens（OpenAI 标准）
		ReasoningTokens   int `json:"reasoning_tokens,omitempty"`
		CompletionDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage,omitempty"`
}

// Stream 调用 OpenAI 兼容 API 的 SSE 接口，逐个返回 token 增量。
// HTTP 建连错误在返回前同步返回；响应体解析错误通过最后一个 Error delta 传递。
func (c *OpenAIClient) Stream(ctx context.Context, req *LLMRequest) (<-chan llmcore.LLMDelta, error) {
	if req == nil {
		return nil, fmt.Errorf("llm request is nil")
	}

	reqBody, err := sonic.Marshal(c.buildChatRequest(req, true))
	if err != nil {
		return nil, fmt.Errorf("marshal stream request: %w", err)
	}
	// 整体截止的 cancel 由 reader goroutine 释放，不能 defer 在 Stream 里，
	// 否则 Stream 返回即取消、流还没被消费。
	streamCtx, cancelStream := streamContext(ctx)
	httpReq, err := http.NewRequestWithContext(streamCtx, http.MethodPost, c.baseURL+openAIChatCompletionsPath, bytes.NewReader(reqBody))
	if err != nil {
		cancelStream()
		return nil, fmt.Errorf("create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", httpconst.ContentTypeJSON)
	httpReq.Header.Set("Accept", httpconst.ContentTypeSSE)
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", httpconst.AuthBearerPrefix+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		cancelStream()
		if errors.Is(err, context.DeadlineExceeded) {
			pe := llmcore.NewProviderError(llmcore.KindTimeout, "openai_compat", c.modelName, "stream request timeout")
			pe.StatusCode = http.StatusGatewayTimeout
			pe.Cause = err
			return nil, pe
		}
		pe := llmcore.NewProviderError(llmcore.KindNetwork, "openai_compat", c.modelName, "stream network error")
		pe.Cause = err
		return nil, pe
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		cancelStream()
		if readErr != nil {
			return nil, fmt.Errorf("read stream error response: %w", readErr)
		}
		return nil, classifyHTTPError(resp.StatusCode, body, "openai_compat", c.modelName)
	}

	deltas := make(chan llmcore.LLMDelta)
	// cancel 由 reader goroutine 结束时释放（含整体截止定时器）。
	go c.readOpenAIStream(streamCtx, resp.Body, deltas, cancelStream)
	return deltas, nil
}

func (c *OpenAIClient) readOpenAIStream(ctx context.Context, body io.ReadCloser, deltas chan<- llmcore.LLMDelta, cancel context.CancelFunc) {
	defer cancel()
	defer close(deltas)
	defer body.Close()

	err := sse.ParseWithContext(ctx, body, sse.DefaultConfig, func(ctx context.Context, frame sse.Frame) bool {
		// 空 data 帧跳过
		if len(frame.Data) == 0 {
			return true
		}

		// OpenAI SSE 结束标记
		if string(frame.Data) == openAISSEEndToken {
			return false
		}

		var chunk openAIStreamChunk
		if err := sonic.Unmarshal(frame.Data, &chunk); err != nil {
			sendStreamDelta(ctx, deltas, llmcore.LLMDelta{Error: fmt.Sprintf("decode stream chunk: %v", err)})
			return false
		}

		var usage *llmcore.Usage
		if chunk.Usage != nil {
			usage = &llmcore.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
				ReasoningTokens:  reasoningTokens(chunk.Usage.ReasoningTokens, chunk.Usage.CompletionDetails.ReasoningTokens),
			}
		}

		// 只有 usage 没有 choices 的帧（通常在开头或结尾）
		if len(chunk.Choices) == 0 && usage != nil {
			return sendStreamDelta(ctx, deltas, llmcore.LLMDelta{Usage: usage})
		}

		for _, choice := range chunk.Choices {
			reasoning := choice.Delta.ReasoningContent
			if reasoning == "" {
				reasoning = choice.Delta.Reasoning
			}
			delta := llmcore.LLMDelta{
				ContentText:  choice.Delta.Content,
				Reasoning:    reasoning,
				FinishReason: normalizeOpenAIFinishReason(choice.FinishReason),
				Usage:        usage,
			}
			for _, toolCall := range choice.Delta.ToolCalls {
				delta.ToolCalls = append(delta.ToolCalls, llmcore.ToolCallDelta{
					Index:          toolCall.Index,
					ID:             toolCall.ID,
					Type:           toolCall.Type,
					Name:           toolCall.Function.Name,
					ArgumentsDelta: toolCall.Function.Arguments,
				})
			}
			if delta.ContentText == "" && delta.Reasoning == "" && len(delta.ToolCalls) == 0 && delta.FinishReason == "" {
				continue
			}
			if !sendStreamDelta(ctx, deltas, delta) {
				return false
			}
		}
		return true
	})

	// context 已取消时下游早已收不到错误，静默退出即可，其余错误才上报。
	if err != nil && !errors.Is(err, context.Canceled) {
		sendStreamDelta(ctx, deltas, llmcore.LLMDelta{Error: fmt.Sprintf("read stream: %v", err)})
	}
}

func sendStreamDelta(ctx context.Context, deltas chan<- llmcore.LLMDelta, delta llmcore.LLMDelta) bool {
	select {
	case deltas <- delta:
		return true
	case <-ctx.Done():
		return false
	}
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
	case model.ReasoningOff:
		v := openAIReasoningEffortNone
		return &v
	case model.ReasoningLow:
		v := openAIReasoningEffortLow
		return &v
	case model.ReasoningHigh:
		v := openAIReasoningEffortHigh
		return &v
		// ReasoningAuto 有意落空：OpenAI 协议没有 auto 档（reasoning_effort 仅
		// none/low/medium/high），不发参数让模型用默认行为。
		// 注意与 Anthropic 适配器语义不同：那边 Auto 映射 ThinkingBudgetLow。
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

func estimateCostUSD(policy model.CostPolicy, usage llmcore.Usage) float64 {
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

// classifyHTTPError 把 HTTP 状态码分类为 llmcore.ErrorKind，并包装为 ProviderError。
// 参考 MULTI_LLM_ADAPTER_DESIGN.md "错误分类"：
//   - 401/403 → auth         不重试不 fallback
//   - 429     → rate_limit   必要时 fallback（本层只分类，退避策略由上层路由重试决定）
//   - 5xx     → server       有限重试，失败后 fallback
//   - 4xx     → bad_request  不重试
//   - 其他    → unknown
func classifyHTTPError(status int, body []byte, provider, modelName string) error {
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

	pe := llmcore.NewProviderError(kind, provider, modelName,
		fmt.Sprintf("status=%d: %s", status, upstreamMsg))
	pe.StatusCode = status
	return pe
}

// truncateBody 截断响应体用于错误消息。按 rune 边界回退，
// 避免字节截断切断多字节 UTF-8 字符导致日志乱码。
func truncateBody(b []byte) string {
	const max = 512
	if len(b) > max {
		cut := max
		for cut > 0 && !utf8.RuneStart(b[cut]) {
			cut--
		}
		return string(b[:cut]) + "..."
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
