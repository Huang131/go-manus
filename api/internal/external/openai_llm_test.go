package external

import (
	"context"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mooc-manus/go-manus/api/internal/llmcore"
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "你好，我是助手。" {
		t.Errorf("Content = %q, want %q", resp.Content, "你好，我是助手。")
	}
	if resp.ReasoningContent != "" {
		t.Errorf("ReasoningContent should be empty, got %q", resp.ReasoningContent)
	}
	if len(resp.ToolUse) != 0 {
		t.Errorf("ToolUse should be empty, got %d", len(resp.ToolUse))
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
			{Role: llmcore.RoleUser, ContentText: "hello"},
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	// 协议层断言 1：Content 必须是上游原始 content（空）
	if resp.Content != "" {
		t.Errorf("Content should be EMPTY (no reasoning→content leak at protocol layer), got %q", resp.Content)
	}
	// 协议层断言 2：RawContent 必须是上游原始 content（空），不能是 reasoning
	if resp.RawContent != "" {
		t.Errorf("RawContent should be EMPTY (上游 content 为空), got %q", resp.RawContent)
	}
	// 协议层断言 3：ReasoningContent 必须独立保留
	if resp.ReasoningContent != "分析任务...拆解步骤..." {
		t.Errorf("ReasoningContent = %q, want %q", resp.ReasoningContent, "分析任务...拆解步骤...")
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "最终回答。" {
		t.Errorf("Content = %q, want %q", resp.Content, "最终回答。")
	}
	if resp.ReasoningContent != "思考中...先分析用户问题。" {
		t.Errorf("ReasoningContent = %q, want %q", resp.ReasoningContent, "思考中...先分析用户问题。")
	}
	// 防止 reasoning 覆盖 content
	if resp.Content == resp.ReasoningContent {
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.ReasoningContent != "thinking..." {
		t.Errorf("ReasoningContent = %q, want %q", resp.ReasoningContent, "thinking...")
	}
	if resp.Content != "ok" {
		t.Errorf("Content = %q, want %q", resp.Content, "ok")
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
			{Role: llmcore.RoleUser, ContentText: "上海天气"},
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
	if len(resp.ToolUse) != 1 {
		t.Fatalf("ToolUse len = %d, want 1", len(resp.ToolUse))
	}
	tc := resp.ToolUse[0]
	if tc.ID != "call_abc" {
		t.Errorf("tool.id = %v, want call_abc", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("tool.name = %v, want get_weather", tc.Name)
	}
	if tc.Arguments != `{"city":"上海"}` {
		t.Errorf("tool.arguments = %v", tc.Arguments)
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
		ReasoningMode:      llmcore.ReasoningOff,
		Extra: map[string]llmcore.ExtraParam{
			"reasoning_effort": {Kind: "json", Raw: []byte(`"high"`)},
			"bad_param":        {Kind: "json", Raw: []byte(`"leak"`)},
		},
	}
	c.costPolicy = llmcore.CostPolicy{
		InputPricePerMTokens:  1,
		OutputPricePerMTokens: 2,
		Currency:              "USD",
	}

	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: llmcore.RoleUser, ContentText: "hello"},
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
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
			{Role: llmcore.RoleUser, ContentText: "hi"},
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
