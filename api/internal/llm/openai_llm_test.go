package llm

import (
	"context"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// newTestClient 构造一个 baseURL 指向 test server 的 OpenAIClient
func newTestClient(t *testing.T, baseURL string) *OpenAIClient {
	t.Helper()
	c := NewOpenAIClient(&OpenAIClientConfig{
		BaseURL:         baseURL,
		APIKey:          "test-key",
		ModelName:       "test-model",
		Temperature:     0.7,
		MaxTokens:       1024,
		ToolCallTimeout: 2,
	})
	// 缩短请求总超时，让 timeout 测试不会被全局 120s 卡住
	c.httpClient.Timeout = 5 * time.Second
	return c
}

func TestNewOpenAIClient_DoesNotApplyGlobalTimeout(t *testing.T) {
	client := NewOpenAIClient(&OpenAIClientConfig{})
	if client.httpClient.Timeout != 0 {
		t.Fatalf("http client timeout = %s, want request-scoped timeout", client.httpClient.Timeout)
	}
}

func TestOpenAIClient_StreamProducesDeltas(t *testing.T) {
	var requestBody map[string]interface{}
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		if err := sonic.Unmarshal(body, &requestBody); err != nil {
			return nil, err
		}
		streamBody := strings.Join([]string{
			"data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"你好\",\"reasoning_content\":\"思考\"}}]}",
			"",
			"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-1\",\"type\":\"function\",\"function\":{\"name\":\"search\",\"arguments\":\"{\\\"q\\\":\"}}]}}]}",
			"",
			"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"go\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":3,\"total_tokens\":5}}",
			"",
			"data: [DONE]",
			"",
		}, "\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(streamBody)),
		}, nil
	})

	deltas, err := c.Stream(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	var got []llmcore.LLMDelta
	for delta := range deltas {
		got = append(got, delta)
	}
	if requestBody["stream"] != true {
		t.Fatalf("request stream = %v, want true", requestBody["stream"])
	}
	if len(got) != 3 {
		t.Fatalf("delta count = %d, want 3", len(got))
	}
	if got[0].ContentText != "你好" || got[0].Reasoning != "思考" {
		t.Fatalf("first delta = %+v", got[0])
	}
	if len(got[1].ToolCalls) != 1 || got[1].ToolCalls[0].Name != "search" {
		t.Fatalf("tool start delta = %+v", got[1])
	}
	if got[2].ToolCalls[0].ArgumentsDelta != "\"go\"}" || got[2].FinishReason != "tool_calls" {
		t.Fatalf("tool finish delta = %+v", got[2])
	}
	if got[2].Usage == nil || got[2].Usage.TotalTokens != 5 {
		t.Fatalf("usage delta = %+v", got[2].Usage)
	}
}

// TestNormalizeOpenAIFinishReason 锁定 finish_reason → canonical 的翻译契约
func TestNormalizeOpenAIFinishReason(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "legacy function_call", in: "function_call", want: llmcore.FinishReasonToolCalls},
		{name: "canonical tool_calls", in: "tool_calls", want: llmcore.FinishReasonToolCalls},
		{name: "canonical stop", in: "stop", want: llmcore.FinishReasonStop},
		{name: "canonical length", in: "length", want: llmcore.FinishReasonLength},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeOpenAIFinishReason(tt.in); got != tt.want {
				t.Errorf("normalizeOpenAIFinishReason(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestOpenAIClient_StreamStopsWhenContextCanceled(t *testing.T) {
	started := make(chan struct{})
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		close(started)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       &contextBlockingReader{ctx: r.Context()},
		}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	deltas, err := c.Stream(ctx, &LLMRequest{})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	<-started
	cancel()

	for range deltas {
	}
}

type contextBlockingReader struct {
	ctx context.Context
}

func (r *contextBlockingReader) Read([]byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func (r *contextBlockingReader) Close() error {
	return nil
}

// rawOK 把任意 JSON 写入 200 响应
func rawOK(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	data, _ := sonic.Marshal(payload)
	w.Write(data)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTransportClient(t *testing.T, handler func(*http.Request) (*http.Response, error)) *OpenAIClient {
	t.Helper()
	c := NewOpenAIClient(&OpenAIClientConfig{
		BaseURL:     "https://example.invalid",
		APIKey:      "test-key",
		ModelName:   "test-model",
		Temperature: 0.7,
		MaxTokens:   1024,
	})
	c.httpClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: roundTripperFunc(handler),
	}
	return c
}

func responseJSON(status int, payload interface{}) (*http.Response, error) {
	body, err := sonic.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}, nil
}

// === 1. content-only ===

func TestOpenAIClient_ContentOnly(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		// 校验基本请求
		body, _ := io.ReadAll(r.Body)
		var got map[string]interface{}
		_ = sonic.Unmarshal(body, &got)
		if got["model"] != "test-model" {
			t.Errorf("request.model = %v, want test-model", got["model"])
		}
		return responseJSON(http.StatusOK, map[string]interface{}{
			"id":    "chatcmpl-1",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "你好，我是助手。",
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 8,
				"total_tokens":      18,
			},
		})
	})
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Message.ContentText != "你好，我是助手。" {
		t.Errorf("Content = %q, want %q", resp.Message.ContentText, "你好，我是助手。")
	}
	if resp.Message.Reasoning != "" {
		t.Errorf("ReasoningContent should be empty, got %q", resp.Message.Reasoning)
	}
	if len(resp.Message.ToolCalls) != 0 {
		t.Errorf("ToolUse should be empty, got %d", len(resp.Message.ToolCalls))
	}
}

// === 1b. request wire format ===

func TestOpenAIClient_RequestWireFormat(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		var got map[string]interface{}
		_ = sonic.Unmarshal(body, &got)

		msgs, ok := got["messages"].([]interface{})
		if !ok || len(msgs) != 1 {
			t.Fatalf("messages = %v, want 1 message", got["messages"])
		}
		first, _ := msgs[0].(map[string]interface{})
		if _, ok := first["content_text"]; ok {
			t.Fatalf("wire message should not contain content_text: %v", first)
		}
		if first["content"] != "hello" {
			t.Fatalf("wire message content = %v, want hello", first["content"])
		}
		if first["role"] != "user" {
			t.Fatalf("wire message role = %v, want user", first["role"])
		}

		tools, ok := got["tools"].([]interface{})
		if !ok || len(tools) != 1 {
			t.Fatalf("tools = %v, want 1 tool", got["tools"])
		}
		tool, _ := tools[0].(map[string]interface{})
		if _, ok := tool["read_only"]; ok {
			t.Fatalf("wire tool should not contain read_only: %v", tool)
		}
		return responseJSON(http.StatusOK, map[string]interface{}{
			"id":    "chatcmpl-wire",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "ok",
					},
				},
			},
			"usage": map[string]interface{}{},
		})
	})
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hello"},
		},
		Tools: []llmcore.ToolSpec{
			{
				Type: "function",
				Function: llmcore.ToolSpecFunction{
					Name:        "search",
					Description: "search",
					Parameters:  map[string]interface{}{"type": "object"},
				},
				ReadOnly: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
}

// === 2. reasoning-only：验证协议层不做 reasoning→content 兜底 ===
//
// 阶段 1d 关键断言：OpenAIClient.Invoke 是协议层出口，不该做"业务兜底"。
// Reasoning→Content 兜底已下沉到 agent 消费侧（react_agent / planner_agent），
// 协议层只保证"上游给什么字段就如实返回什么字段"。
// 本测试只断言 OpenAIClient.Invoke 自身的"协议层不混淆"契约。

func TestOpenAIClient_ReasoningOnly_NoContentLeak(t *testing.T) {
	// glm-5.2 默认思考模式：上游只返回 reasoning_content，无 content
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusOK, map[string]interface{}{
			"id":    "chatcmpl-2",
			"model": "glm-5.2",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":              "assistant",
						"content":           "",
						"reasoning_content": "分析任务...拆解步骤...",
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     5,
				"completion_tokens": 6,
				"total_tokens":      11,
			},
		})
	})
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	// 协议层断言 1：Content 必须是上游原始 content（空）
	if resp.Message.ContentText != "" {
		t.Errorf("Content should be EMPTY (no reasoning→content leak at protocol layer), got %q", resp.Message.ContentText)
	}
	// 协议层断言 2：RawContent 必须是上游原始 content（空），不能是 reasoning
	if resp.Message.ContentText != "" {
		t.Errorf("RawContent should be EMPTY (上游 content 为空), got %q", resp.Message.ContentText)
	}
	// 协议层断言 3：ReasoningContent 必须独立保留
	if resp.Message.Reasoning != "分析任务...拆解步骤..." {
		t.Errorf("ReasoningContent = %q, want %q", resp.Message.Reasoning, "分析任务...拆解步骤...")
	}
}

// === 3. content + reasoning 同时返回 ===

func TestOpenAIClient_ContentAndReasoning_Separated(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusOK, map[string]interface{}{
			"id":    "chatcmpl-3",
			"model": "deepseek-v4-flash",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":              "assistant",
						"content":           "最终回答。",
						"reasoning_content": "思考中...先分析用户问题。",
					},
				},
			},
			"usage": map[string]interface{}{},
		})
	})
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Message.ContentText != "最终回答。" {
		t.Errorf("Content = %q, want %q", resp.Message.ContentText, "最终回答。")
	}
	if resp.Message.Reasoning != "思考中...先分析用户问题。" {
		t.Errorf("ReasoningContent = %q, want %q", resp.Message.Reasoning, "思考中...先分析用户问题。")
	}
	// 防止 reasoning 覆盖 content
	if resp.Message.ContentText == resp.Message.Reasoning {
		t.Fatal("reasoning leaked into content")
	}
}

// === 3b. reasoning_content 缺位、reasoning 字段命中（部分厂商命名差异） ===

func TestOpenAIClient_ReasoningAltField(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusOK, map[string]interface{}{
			"id":    "chatcmpl-3b",
			"model": "some-vendor",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":      "assistant",
						"content":   "ok",
						"reasoning": "thinking...", // 仅 reasoning 字段
					},
				},
			},
			"usage": map[string]interface{}{},
		})
	})
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Message.Reasoning != "thinking..." {
		t.Errorf("ReasoningContent = %q, want %q", resp.Message.Reasoning, "thinking...")
	}
	if resp.Message.ContentText != "ok" {
		t.Errorf("Content = %q, want %q", resp.Message.ContentText, "ok")
	}
}

// === 4. tool_call 返回 ===

func TestOpenAIClient_ToolCall(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusOK, map[string]interface{}{
			"id":    "chatcmpl-4",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "tool_calls",
					"message": map[string]interface{}{
						"role": "assistant",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_abc",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "get_weather",
									"arguments": `{"city":"上海"}`,
								},
							},
						},
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     5,
				"completion_tokens": 8,
				"total_tokens":      13,
			},
		})
	})
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "上海天气"},
		},
		Tools: []llmcore.ToolSpec{
			{
				Type: "function",
				Function: llmcore.ToolSpecFunction{
					Name:        "get_weather",
					Description: "get weather",
					Parameters:  map[string]interface{}{"type": "object"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("ToolUse len = %d, want 1", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("tool.id = %v, want call_abc", tc.ID)
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("tool.name = %v, want get_weather", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"city":"上海"}` {
		t.Errorf("tool.arguments = %v", tc.Function.Arguments)
	}
}

func TestOpenAIClient_RequestPolicyAndCost(t *testing.T) {
	var got map[string]interface{}

	temp := 0.2
	maxTokens := 4096
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		_ = sonic.Unmarshal(body, &got)
		return responseJSON(http.StatusOK, map[string]interface{}{
			"id":    "chatcmpl-policy",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "ok",
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     1000,
				"completion_tokens": 2000,
				"total_tokens":      3000,
			},
		})
	})
	c.requestPolicy = llmcore.RequestPolicy{
		DefaultTemperature: &temp,
		DefaultMaxTokens:   &maxTokens,
		ReasoningMode:      model.ReasoningOff,
		Extra: map[string]llmcore.ExtraParam{
			"reasoning_effort": {Kind: "json", Raw: []byte(`"high"`)},
			"bad_param":        {Kind: "json", Raw: []byte(`"leak"`)},
		},
	}
	c.costPolicy = model.CostPolicy{
		InputPricePerMTokens:  1,
		OutputPricePerMTokens: 2,
		Currency:              "USD",
	}

	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	if got["temperature"] != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", got["temperature"])
	}
	if got["max_tokens"] != float64(4096) {
		t.Fatalf("max_tokens = %v, want 4096", got["max_tokens"])
	}
	if got["reasoning_effort"] != "none" {
		t.Fatalf("reasoning_effort = %v, want none", got["reasoning_effort"])
	}
	if _, ok := got["bad_param"]; ok {
		t.Fatalf("bad_param should not be forwarded: %v", got)
	}
	if resp.Usage.PromptTokens != 1000 || resp.Usage.CompletionTokens != 2000 || resp.Usage.TotalTokens != 3000 {
		t.Fatalf("usage = %+v, want 1000/2000/3000", resp.Usage)
	}
	if resp.CostUSD != 0.005 {
		t.Fatalf("cost = %v, want 0.005", resp.CostUSD)
	}
}

// === 5. 401 → KindAuth, not Fallbackable ===

func TestOpenAIClient_401_Auth(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusUnauthorized, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
				"code":    "invalid_api_key",
			},
		})
	})
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindAuth) {
		t.Fatalf("error kind = %v, want KindAuth", err)
	}
	pe, ok := err.(*llmcore.ProviderError)
	if !ok {
		t.Fatalf("err is not *ProviderError: %T", err)
	}
	if pe.Fallbackable {
		t.Error("401 should NOT be fallbackable (auth error)")
	}
	if pe.Retryable {
		t.Error("401 should NOT be retryable (auth error)")
	}
}

// === 6. 429 → KindRateLimit, Retryable, Fallbackable ===

func TestOpenAIClient_429_RateLimit(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusTooManyRequests, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Rate limit exceeded",
				"type":    "rate_limit_error",
			},
		})
	})
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindRateLimit) {
		t.Fatalf("error kind = %v, want KindRateLimit", err)
	}
	pe := err.(*llmcore.ProviderError)
	if !pe.Retryable {
		t.Error("429 should be retryable")
	}
	if !pe.Fallbackable {
		t.Error("429 should be fallbackable")
	}
}

// === 7. 5xx → KindServer ===

func TestOpenAIClient_5xx_Server(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusBadGateway, map[string]interface{}{
			"error": map[string]interface{}{"message": "upstream down"},
		})
	})
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindServer) {
		t.Fatalf("error kind = %v, want KindServer", err)
	}
	pe := err.(*llmcore.ProviderError)
	if pe.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d, want %d", pe.StatusCode, http.StatusBadGateway)
	}
}

// === 8. 协议错误 → KindUnknown, not Retryable, not Fallbackable ===

func TestOpenAIClient_ProtocolError_NotFallbackable(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("not json at all")),
		}, nil
	})
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hi"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	pe, ok := err.(*llmcore.ProviderError)
	if !ok {
		t.Fatalf("err is not *ProviderError: %T", err)
	}
	if pe.Retryable {
		t.Error("protocol error should NOT be retryable")
	}
	if pe.Fallbackable {
		t.Error("protocol error should NOT be fallbackable")
	}
}

// === 9. max_tokens wire 契约：未配置（0）必须省略 ===
//
// 守护协议契约：OpenAI Chat API 要求 max_tokens >= 1。
// 配置链路（DB → runtimeConfigToOpenAIClientConfig）中 max_tokens 可为 0（未配置），
// 此时必须从 wire 省略该字段——下发 "max_tokens": 0 会被上游 400 拒绝
// （"max_tokens must be at least 1"）。修复前该测试失败，可复现该 400 问题。

func newWireCaptureClient(t *testing.T, maxTokens int, respond func() (*http.Response, error)) (*OpenAIClient, *map[string]interface{}) {
	t.Helper()
	var got map[string]interface{}
	c := NewOpenAIClient(&OpenAIClientConfig{
		BaseURL:   "https://example.invalid",
		APIKey:    "test-key",
		ModelName: "test-model",
		MaxTokens: maxTokens,
	})
	c.httpClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			_ = sonic.Unmarshal(body, &got)
			return respond()
		}),
	}
	return c, &got
}

func okChatResponse() (*http.Response, error) {
	return responseJSON(http.StatusOK, map[string]interface{}{
		"id":    "chatcmpl-wire",
		"model": "test-model",
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"finish_reason": "stop",
				"message":       map[string]interface{}{"role": "assistant", "content": "ok"},
			},
		},
		"usage": map[string]interface{}{},
	})
}

func TestOpenAIClient_MaxTokensWireContract(t *testing.T) {
	userMsg := []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}}

	t.Run("Invoke 未配置时省略 max_tokens", func(t *testing.T) {
		c, gotPtr := newWireCaptureClient(t, 0, okChatResponse)
		if _, err := c.Invoke(context.Background(), &LLMRequest{Messages: userMsg}); err != nil {
			t.Fatalf("Invoke: %v", err)
		}
		got := *gotPtr
		if _, ok := got["max_tokens"]; ok {
			t.Fatalf("未配置时 wire 不应包含 max_tokens（下发 0 会被上游 400 拒绝），got max_tokens=%v", got["max_tokens"])
		}
	})

	t.Run("Stream 未配置时省略 max_tokens 并显式请求 usage", func(t *testing.T) {
		streamBody := strings.Join([]string{
			`data: {"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`,
			``,
			`data: {"choices":[],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5,"completion_tokens_details":{"reasoning_tokens":2}}}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")
		c, gotPtr := newWireCaptureClient(t, 0, func() (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(streamBody)),
			}, nil
		})
		var lastUsage *llmcore.Usage
		deltas, err := c.Stream(context.Background(), &LLMRequest{Messages: userMsg})
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}
		for delta := range deltas {
			if delta.Error != "" {
				t.Fatalf("stream error: %s", delta.Error)
			}
			if delta.Usage != nil {
				lastUsage = delta.Usage
			}
		}
		got := *gotPtr
		if _, ok := got["max_tokens"]; ok {
			t.Fatalf("未配置时 stream wire 不应包含 max_tokens，got %v", got["max_tokens"])
		}
		// 严格 OpenAI 协议流式默认不返回 usage，必须显式开启
		opts, ok := got["stream_options"].(map[string]interface{})
		if !ok || opts["include_usage"] != true {
			t.Fatalf("stream_options = %v, want include_usage=true", got["stream_options"])
		}
		if lastUsage == nil {
			t.Fatal("流式 usage 帧未被解析")
		}
		if lastUsage.ReasoningTokens != 2 {
			t.Fatalf("ReasoningTokens = %d, want 2（completion_tokens_details 归一化）", lastUsage.ReasoningTokens)
		}
	})

	t.Run("policy 默认值填充 max_tokens", func(t *testing.T) {
		c, gotPtr := newWireCaptureClient(t, 0, okChatResponse)
		defaultMax := 2048
		c.requestPolicy = llmcore.RequestPolicy{DefaultMaxTokens: &defaultMax}
		if _, err := c.Invoke(context.Background(), &LLMRequest{Messages: userMsg}); err != nil {
			t.Fatalf("Invoke: %v", err)
		}
		if got := *gotPtr; got["max_tokens"] != float64(2048) {
			t.Fatalf("max_tokens = %v, want 2048（policy 默认值）", got["max_tokens"])
		}
	})

	t.Run("显式配置时正常下发", func(t *testing.T) {
		c, gotPtr := newWireCaptureClient(t, 512, okChatResponse)
		if _, err := c.Invoke(context.Background(), &LLMRequest{Messages: userMsg}); err != nil {
			t.Fatalf("Invoke: %v", err)
		}
		if got := *gotPtr; got["max_tokens"] != float64(512) {
			t.Fatalf("max_tokens = %v, want 512", got["max_tokens"])
		}
	})
}

// === 10. 200 + error body：上游真实错误不被 "no choices" 覆盖 ===
//
// 守护协议契约：部分兼容网关（one-api/new-api 等）用 200 + error body 转发上游错误。
// Adapter 必须消费 error.message 并以 KindServer 上报，而不是丢失后报 "no choices"。

func TestOpenAIClient_200WithErrorBodySurfacesUpstreamMessage(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return responseJSON(http.StatusOK, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Upstream model overloaded",
				"type":    "server_error",
				"code":    "overloaded",
			},
		})
	})
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindServer) {
		t.Fatalf("error kind = %v, want KindServer (err=%v)", err, err)
	}
	if !strings.Contains(err.Error(), "Upstream model overloaded") {
		t.Fatalf("错误信息应携带上游 message，got: %v", err)
	}
}

// === 11. 超时归因：父 ctx 到期 ≠ tool calling 子超时 ===
//
// 守护排障语义：只有 tool-call 子超时（Invoke 对带 tools 的请求设置）才允许归因为
// "模型可能不支持 tool calling"；父 ctx 先到期时必须是普通 request timeout，
// 否则全局超时会被误报为模型能力问题。

func blockingTransport(t *testing.T) roundTripperFunc {
	t.Helper()
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() != nil {
			return nil, r.Context().Err()
		}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
}

func TestOpenAIClient_ParentDeadlineNotMisattributedToToolCalling(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return blockingTransport(t)(r)
	})
	c.toolCallTimeout = 5 * time.Second // 子超时远大于父超时

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	<-ctx.Done() // 等父 ctx 真正到期，消除与子超时的同时刻竞争

	_, err := c.Invoke(ctx, &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
		Tools: []llmcore.ToolSpec{{
			Type:     llmcore.ToolTypeFunction,
			Function: llmcore.ToolSpecFunction{Name: "search"},
		}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindTimeout) {
		t.Fatalf("error kind = %v, want KindTimeout (err=%v)", err, err)
	}
	if strings.Contains(err.Error(), "tool calling") {
		t.Fatalf("父 ctx 到期不应归因为 tool calling 不支持，got: %v", err)
	}
}

func TestOpenAIClient_ToolCallTimeoutStillAttributed(t *testing.T) {
	c := newTransportClient(t, func(r *http.Request) (*http.Response, error) {
		return blockingTransport(t)(r)
	})
	c.toolCallTimeout = 50 * time.Millisecond // 子超时先到期，父 ctx（Background）永不到期

	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
		Tools: []llmcore.ToolSpec{{
			Type:     llmcore.ToolTypeFunction,
			Function: llmcore.ToolSpecFunction{Name: "search"},
		}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindTimeout) {
		t.Fatalf("error kind = %v, want KindTimeout (err=%v)", err, err)
	}
	if !strings.Contains(err.Error(), "tool calling") {
		t.Fatalf("tool-call 子超时应保留归因信息，got: %v", err)
	}
}
