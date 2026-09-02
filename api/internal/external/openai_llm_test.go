package external

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	_ = json.NewEncoder(w).Encode(payload)
}

// === 1. content-only ===

func TestOpenAIClient_ContentOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 校验基本请求
		body, _ := io.ReadAll(r.Body)
		var got map[string]interface{}
		_ = json.Unmarshal(body, &got)
		if got["model"] != "test-model" {
			t.Errorf("request.model = %v, want test-model", got["model"])
		}
		rawOK(w, map[string]interface{}{
			"id":    "chatcmpl-1",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"index":        0,
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
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{
			{"role": "user", "content": "hi"},
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

// === 2. reasoning-only：验证协议层不做 reasoning→content 兜底 ===
//
// 阶段 1c 关键断言：OpenAIClient.Invoke 是协议层出口，不该做"业务兜底"。
// 业务兜底仍在 NormalizeLLMResponse 里，由 DynamicLLM.Invoke 显式调用，
// 那是 react_agent 真实依赖的兼容逻辑（阶段 1d 才迁移到 agent 消费侧）。
// 本测试只断言 OpenAIClient.Invoke 自身的"协议层不混淆"契约。

func TestOpenAIClient_ReasoningOnly_NoContentLeak(t *testing.T) {
	// glm-5.2 默认思考模式：上游只返回 reasoning_content，无 content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawOK(w, map[string]interface{}{
			"id":    "chatcmpl-2",
			"model": "glm-5.2",
			"choices": []map[string]interface{}{
				{
					"index":        0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":             "assistant",
						"content":          "",
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
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{
			{"role": "user", "content": "hi"},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawOK(w, map[string]interface{}{
			"id":    "chatcmpl-3",
			"model": "deepseek-v4-flash",
			"choices": []map[string]interface{}{
				{
					"index":        0,
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":             "assistant",
						"content":          "最终回答。",
						"reasoning_content": "思考中...先分析用户问题。",
					},
				},
			},
			"usage": map[string]interface{}{},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{
			{"role": "user", "content": "hi"},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawOK(w, map[string]interface{}{
			"id":    "chatcmpl-3b",
			"model": "some-vendor",
			"choices": []map[string]interface{}{
				{
					"index":        0,
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
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawOK(w, map[string]interface{}{
			"id":    "chatcmpl-4",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"index":        0,
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
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{{"role": "user", "content": "上海天气"}},
		Tools: []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "get_weather",
					"description": "get weather",
					"parameters":  map[string]interface{}{"type": "object"},
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
	if tc["id"] != "call_abc" {
		t.Errorf("tool.id = %v, want call_abc", tc["id"])
	}
	fn, _ := tc["function"].(map[string]interface{})
	if fn["name"] != "get_weather" {
		t.Errorf("tool.function.name = %v, want get_weather", fn["name"])
	}
	if fn["arguments"] != `{"city":"上海"}` {
		t.Errorf("tool.function.arguments = %v", fn["arguments"])
	}
}

// === 5. 401 → KindAuth, not Fallbackable ===

func TestOpenAIClient_401_Auth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
				"code":    "invalid_api_key",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Rate limit exceeded",
				"type":    "rate_limit_error",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{"message": "upstream down"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回非 JSON
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json at all"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
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