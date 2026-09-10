package external

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/llmcore"
)

// newAnthropicTestClient 创建使用 mock transport 的 Anthropic 客户端，
// captured 会被填充为请求 wire body。
func newAnthropicTestClient(t *testing.T, captured *map[string]interface{}, respStatus int, respPayload interface{}) *AnthropicClient {
	t.Helper()
	c := NewAnthropicClient(&AnthropicClientConfig{
		BaseURL:   "https://example.invalid",
		APIKey:    "test-key",
		ModelName: "claude-test",
		MaxTokens: 1024,
	})
	c.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		wire := make(map[string]interface{})
		_ = sonic.Unmarshal(body, &wire)
		*captured = wire
		return responseJSON(respStatus, respPayload)
	})
	return c
}

func TestNewAnthropicClient_DoesNotApplyGlobalTimeout(t *testing.T) {
	client := NewAnthropicClient(&AnthropicClientConfig{})
	if client.httpClient.Timeout != 0 {
		t.Fatalf("http client timeout = %s, want request-scoped timeout", client.httpClient.Timeout)
	}
}

func TestAnthropicClient_StreamProducesDeltas(t *testing.T) {
	c := newAnthropicTestClient(t, nil, http.StatusOK, nil)
	c.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var request map[string]interface{}
		if err := sonic.Unmarshal(body, &request); err != nil {
			return nil, err
		}
		if request["stream"] != true {
			t.Errorf("request stream = %v, want true", request["stream"])
		}
		streamBody := strings.Join([]string{
			"event: content_block_delta",
			"data: {\"delta\":{\"type\":\"text_delta\",\"text\":\"你好\"}}",
			"",
			"event: content_block_delta",
			"data: {\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"思考\"}}",
			"",
			"event: content_block_start",
			"data: {\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"tool-1\",\"name\":\"search\"}}",
			"",
			"event: content_block_delta",
			"data: {\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"q\\\":\\\"go\\\"}\"}}",
			"",
			"event: message_delta",
			"data: {\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":3}}",
			"",
			"event: message_stop",
			"data: {}",
			"",
		}, "\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(streamBody)),
		}, nil
	})

	deltas, err := c.Stream(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	var got []llmcore.LLMDelta
	for delta := range deltas {
		got = append(got, delta)
	}
	if len(got) != 5 {
		t.Fatalf("delta count = %d, want 5", len(got))
	}
	if got[0].ContentText != "你好" || got[1].Reasoning != "思考" {
		t.Fatalf("text/reasoning deltas = %+v", got[:2])
	}
	if len(got[2].ToolCalls) != 1 || got[2].ToolCalls[0].Name != "search" {
		t.Fatalf("tool start delta = %+v", got[2])
	}
	if got[3].ToolCalls[0].ArgumentsDelta != "{\"q\":\"go\"}" {
		t.Fatalf("tool args delta = %+v", got[3])
	}
	if got[4].FinishReason != "tool_use" || got[4].Usage == nil || got[4].Usage.CompletionTokens != 3 {
		t.Fatalf("finish delta = %+v", got[4])
	}
}

// === 请求侧：tool_use / tool_result / system / tools 转换 ===

func TestAnthropicClient_ToolUseRequestWire(t *testing.T) {
	var captured map[string]interface{}

	c := newAnthropicTestClient(t, &captured, 200, AnthropicResponse{
		ID:         "resp-1",
		Content:    []AnthropicContent{{Type: "text", Text: "ok"}},
		StopReason: "end_turn",
		Usage:      AnthropicUsage{InputTokens: 10, OutputTokens: 5},
	})

	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: llmcore.RoleSystem, ContentText: "你是助手"},
			{Role: llmcore.RoleUser, ContentText: "查一下北京天气"},
			{
				Role:        llmcore.RoleAssistant,
				ContentText: "我来查询",
				ToolCalls: []llmcore.ToolCall{
					{ID: "call-1", Type: "function", Function: llmcore.ToolCallFunction{Name: "search", Arguments: `{"query":"北京天气"}`}},
				},
			},
			// 连续两条 tool 消息：应合并进同一条 user 消息（角色交替约束）
			{Role: llmcore.RoleTool, ToolCallID: "call-1", ContentText: `{"temp":"26C"}`},
			{Role: llmcore.RoleTool, ToolCallID: "call-1", ContentText: `{"extra":"data"}`},
		},
		Tools: []llmcore.ToolSpec{
			{Type: "function", Function: llmcore.ToolSpecFunction{
				Name:        "search",
				Description: "搜索工具",
				Parameters:  map[string]interface{}{"type": "object"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	// system 抽取
	if s, _ := captured["system"].(string); !strings.Contains(s, "你是助手") {
		t.Errorf("system = %v, want contains 你是助手", captured["system"])
	}

	// tools 定义转 input_schema
	tools, _ := captured["tools"].([]interface{})
	if len(tools) != 1 {
		t.Fatalf("tools 数量 = %d, want 1", len(tools))
	}
	tool, _ := tools[0].(map[string]interface{})
	if tool["name"] != "search" {
		t.Errorf("tools[0].name = %v, want search", tool["name"])
	}
	if _, ok := tool["input_schema"].(map[string]interface{}); !ok {
		t.Errorf("tools[0].input_schema 缺失或形状错误: %v", tool["input_schema"])
	}

	// 消息序列：user / assistant(tool_use) / user(两条 tool_result 合并)
	msgs, _ := captured["messages"].([]interface{})
	if len(msgs) != 3 {
		t.Fatalf("messages 数量 = %d, want 3 (system 抽离 + tool 结果合并): %v", len(msgs), captured["messages"])
	}

	// assistant 消息：text 块 + tool_use 块，input 解析自 Arguments
	assistant, _ := msgs[1].(map[string]interface{})
	if assistant["role"] != "assistant" {
		t.Errorf("messages[1].role = %v, want assistant", assistant["role"])
	}
	assistantBlocks, _ := assistant["content"].([]interface{})
	if len(assistantBlocks) != 2 {
		t.Fatalf("assistant content 块数 = %d, want 2 (text + tool_use): %v", len(assistantBlocks), assistant["content"])
	}
	toolUse, _ := assistantBlocks[1].(map[string]interface{})
	if toolUse["type"] != "tool_use" || toolUse["id"] != "call-1" || toolUse["name"] != "search" {
		t.Errorf("tool_use 块错误: %v", toolUse)
	}
	input, _ := toolUse["input"].(map[string]interface{})
	if input["query"] != "北京天气" {
		t.Errorf("tool_use.input = %v, want query=北京天气", input)
	}

	// 两条 tool 消息合并成一条 user 消息、两个 tool_result 块
	toolUser, _ := msgs[2].(map[string]interface{})
	if toolUser["role"] != "user" {
		t.Errorf("messages[2].role = %v, want user", toolUser["role"])
	}
	resultBlocks, _ := toolUser["content"].([]interface{})
	if len(resultBlocks) != 2 {
		t.Fatalf("tool_result 块数 = %d, want 2 (相邻 tool 消息应合并): %v", len(resultBlocks), toolUser["content"])
	}
	first, _ := resultBlocks[0].(map[string]interface{})
	if first["type"] != "tool_result" || first["tool_use_id"] != "call-1" || first["content"] != `{"temp":"26C"}` {
		t.Errorf("tool_result 块错误: %v", first)
	}
}

// === 响应侧：text / thinking / tool_use 块解析 ===

func TestAnthropicClient_ResponseBlocks(t *testing.T) {
	var captured map[string]interface{}

	c := newAnthropicTestClient(t, &captured, 200, AnthropicResponse{
		ID:   "resp-2",
		Type: "message",
		Content: []AnthropicContent{
			{Type: "thinking", Thinking: "我需要先查天气"},
			{Type: "text", Text: "北京今天 26 度"},
			{Type: "tool_use", ID: "tc-9", Name: "search", Input: map[string]interface{}{"query": "天气"}},
		},
		StopReason: "tool_use",
		Usage:      AnthropicUsage{InputTokens: 20, OutputTokens: 30},
	})

	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "天气如何"}},
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if resp.Message.ContentText != "北京今天 26 度" {
		t.Errorf("ContentText = %q, want 北京今天 26 度", resp.Message.ContentText)
	}
	if resp.Message.Reasoning != "我需要先查天气" {
		t.Errorf("Reasoning = %q, want thinking 块内容", resp.Message.Reasoning)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("ToolCalls 数量 = %d, want 1", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.ID != "tc-9" || tc.Function.Name != "search" || tc.Function.Arguments != `{"query":"天气"}` {
		t.Errorf("ToolCall 解析错误: %+v", tc)
	}
	if resp.FinishReason != "tool_use" {
		t.Errorf("FinishReason = %q, want tool_use", resp.FinishReason)
	}
	if resp.Usage.PromptTokens != 20 || resp.Usage.CompletionTokens != 30 {
		t.Errorf("Usage 错误: %+v", resp.Usage)
	}
}

// === 纯文本请求：content 保持 string 形状（非块数组） ===

func TestAnthropicClient_TextOnlyWire(t *testing.T) {
	var captured map[string]interface{}

	c := newAnthropicTestClient(t, &captured, 200, AnthropicResponse{
		ID:         "resp-3",
		Content:    []AnthropicContent{{Type: "text", Text: "你好"}},
		StopReason: "end_turn",
	})

	resp, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	msgs, _ := captured["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("messages 数量 = %d, want 1", len(msgs))
	}
	msg, _ := msgs[0].(map[string]interface{})
	if content, ok := msg["content"].(string); !ok || content != "hi" {
		t.Errorf("纯文本消息 content 应为 string 形状, got %T: %v", msg["content"], msg["content"])
	}
	if resp.Message.ContentText != "你好" {
		t.Errorf("ContentText = %q, want 你好", resp.Message.ContentText)
	}
}

// === 错误分类：所有上游错误必须归一化为 *ProviderError，
// routed_llm 的 fallback 类型断言才能对 Anthropic 模型生效 ===

// newAnthropicErrorClient 构造自定义 transport 的客户端用于错误路径测试
func newAnthropicErrorClient(t *testing.T, roundTrip roundTripperFunc) *AnthropicClient {
	t.Helper()
	c := NewAnthropicClient(&AnthropicClientConfig{
		BaseURL:   "https://example.invalid",
		APIKey:    "test-key",
		ModelName: "claude-test",
		MaxTokens: 1024,
	})
	c.httpClient.Transport = roundTrip
	return c
}

func anthropicErrorResponse(status int) *http.Response {
	resp, _ := responseJSON(status, map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "error",
			"message": "upstream failure",
		},
	})
	return resp
}

// errKind 提取错误的 ProviderError.Kind，非 ProviderError 返回固定标记，便于断言失败时定位
func errKind(err error) llmcore.ErrorKind {
	if pe, ok := err.(*llmcore.ProviderError); ok {
		return pe.Kind
	}
	return "non-provider-error"
}

func TestAnthropicClient_HTTPErrorClassification(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		wantKind      llmcore.ErrorKind
		wantRetryable bool
		wantFallback  bool
	}{
		{name: "401 auth", status: http.StatusUnauthorized, wantKind: llmcore.KindAuth},
		{name: "403 auth", status: http.StatusForbidden, wantKind: llmcore.KindAuth},
		{name: "404 not_found", status: http.StatusNotFound, wantKind: llmcore.KindNotFound},
		{name: "400 bad_request", status: http.StatusBadRequest, wantKind: llmcore.KindBadRequest},
		{name: "429 rate_limit", status: http.StatusTooManyRequests, wantKind: llmcore.KindRateLimit, wantRetryable: true, wantFallback: true},
		{name: "408 timeout", status: http.StatusRequestTimeout, wantKind: llmcore.KindTimeout, wantRetryable: true, wantFallback: true},
		{name: "500 server", status: http.StatusInternalServerError, wantKind: llmcore.KindServer, wantRetryable: true, wantFallback: true},
		{name: "502 server", status: http.StatusBadGateway, wantKind: llmcore.KindServer, wantRetryable: true, wantFallback: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newAnthropicErrorClient(t, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return anthropicErrorResponse(tt.status), nil
			}))
			_, err := c.Invoke(context.Background(), &LLMRequest{
				Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !llmcore.IsKind(err, tt.wantKind) {
				t.Fatalf("error kind = %v, want %v (err=%v)", errKind(err), tt.wantKind, err)
			}
			pe := err.(*llmcore.ProviderError)
			if pe.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", pe.StatusCode, tt.status)
			}
			if pe.Provider != anthropicProvider || pe.Model != "claude-test" {
				t.Errorf("Provider/Model = %s/%s", pe.Provider, pe.Model)
			}
			if pe.Retryable != tt.wantRetryable {
				t.Errorf("Retryable = %v, want %v", pe.Retryable, tt.wantRetryable)
			}
			if pe.Fallbackable != tt.wantFallback {
				t.Errorf("Fallbackable = %v, want %v", pe.Fallbackable, tt.wantFallback)
			}
			if !strings.Contains(pe.Error(), "upstream failure") {
				t.Errorf("error message 应包含上游 message, got %q", pe.Error())
			}
		})
	}
}

func TestAnthropicClient_NetworkError(t *testing.T) {
	c := newAnthropicErrorClient(t, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp: connection refused")
	}))
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
	})
	if !llmcore.IsKind(err, llmcore.KindNetwork) {
		t.Fatalf("error kind = %v, want KindNetwork", errKind(err))
	}
	pe := err.(*llmcore.ProviderError)
	if !pe.Retryable || !pe.Fallbackable {
		t.Errorf("network error should be retryable/fallbackable, got %+v", pe)
	}
}

// 不支持 tool calling 的模型会在 toolCallTimeout 内无响应 → KindTimeout 且可 fallback
func TestAnthropicClient_ToolCallTimeout(t *testing.T) {
	c := NewAnthropicClient(&AnthropicClientConfig{
		BaseURL:         "https://example.invalid",
		APIKey:          "test-key",
		ModelName:       "claude-no-tools",
		MaxTokens:       1024,
		ToolCallTimeout: 1,
	})
	c.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})

	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
		Tools: []llmcore.ToolSpec{{
			Type:     llmcore.ToolTypeFunction,
			Function: llmcore.ToolSpecFunction{Name: "search"},
		}},
	})
	if !llmcore.IsKind(err, llmcore.KindTimeout) {
		t.Fatalf("error kind = %v, want KindTimeout (err=%v)", errKind(err), err)
	}
	if !err.(*llmcore.ProviderError).Fallbackable {
		t.Error("tool-call 超时应可 fallback 到其他模型")
	}
}

// 200 + 非法 JSON 属于协议错误：不重试、不 fallback（避免对坏模型反复熔断/切换）
func TestAnthropicClient_ProtocolError_NotFallbackable(t *testing.T) {
	c := newAnthropicErrorClient(t, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("not json at all")),
		}, nil
	}))
	_, err := c.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
	})
	pe, ok := err.(*llmcore.ProviderError)
	if !ok {
		t.Fatalf("err is not *ProviderError: %T", err)
	}
	if pe.Retryable || pe.Fallbackable {
		t.Errorf("protocol error should NOT be retryable/fallbackable, got %+v", pe)
	}
}

// Stream 路径的非 2xx 响应同样必须归一化为 ProviderError
func TestAnthropicClient_StreamHTTPError(t *testing.T) {
	c := newAnthropicErrorClient(t, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return anthropicErrorResponse(http.StatusTooManyRequests), nil
	}))
	_, err := c.Stream(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
	})
	if !llmcore.IsKind(err, llmcore.KindRateLimit) {
		t.Fatalf("error kind = %v, want KindRateLimit", errKind(err))
	}
	if !err.(*llmcore.ProviderError).Fallbackable {
		t.Error("429 should be fallbackable")
	}
}

// ToolCallTimeout 默认值兜底
func TestNewAnthropicClient_DefaultToolCallTimeout(t *testing.T) {
	c := NewAnthropicClient(&AnthropicClientConfig{})
	if c.toolCallTimeout != 15*time.Second {
		t.Fatalf("toolCallTimeout = %v, want 15s", c.toolCallTimeout)
	}
}
