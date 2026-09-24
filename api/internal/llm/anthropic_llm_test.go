package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
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
			// 事件 1: 文本增量
			"event: content_block_delta",
			`data: {"delta":{"type":"text_delta","text":"你好"}}`,
			"",
			// 事件 2: 思考增量
			"event: content_block_delta",
			`data: {"delta":{"type":"thinking_delta","thinking":"思考"}}`,
			"",
			// 事件 3: 工具调用开始
			"event: content_block_start",
			`data: {"index":1,"content_block":{"type":"tool_use","id":"tool-1","name":"search"}}`,
			"",
			// 事件 4: 工具参数增量
			"event: content_block_delta",
			`data: {"index":1,"delta":{"type":"input_json_delta","partial_json":"{\"q\":\"go\"}"}}`,
			"",
			// 事件 5: 消息结束
			"event: message_delta",
			`data: {"delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":3}}`,
			"",
			// 事件 6: 流结束
			"event: message_stop",
			`data: {}`,
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
	if got[4].FinishReason != llmcore.FinishReasonToolCalls || got[4].Usage == nil || got[4].Usage.CompletionTokens != 3 {
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
			{Role: model.RoleSystem, ContentText: "你是助手"},
			{Role: model.RoleUser, ContentText: "查一下北京天气"},
			{
				Role:        model.RoleAssistant,
				ContentText: "我来查询",
				ToolCalls: []llmcore.ToolCall{
					{ID: "call-1", Type: "function", Function: llmcore.ToolCallFunction{Name: "search", Arguments: `{"query":"北京天气"}`}},
				},
			},
			// 连续两条 tool 消息：应合并进同一条 user 消息（角色交替约束）
			{Role: model.RoleTool, ToolCallID: "call-1", ContentText: `{"temp":"26C"}`},
			{Role: model.RoleTool, ToolCallID: "call-1", ContentText: `{"extra":"data"}`},
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
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "天气如何"}},
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
	if resp.FinishReason != llmcore.FinishReasonToolCalls {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, llmcore.FinishReasonToolCalls)
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
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
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
	if resp.FinishReason != llmcore.FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, llmcore.FinishReasonStop)
	}
}

// TestNormalizeAnthropicStopReason 锁定 stop_reason → canonical 的翻译契约
func TestNormalizeAnthropicStopReason(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "tool_use", in: "tool_use", want: llmcore.FinishReasonToolCalls},
		{name: "end_turn", in: "end_turn", want: llmcore.FinishReasonStop},
		{name: "stop_sequence", in: "stop_sequence", want: llmcore.FinishReasonStop},
		{name: "max_tokens", in: "max_tokens", want: llmcore.FinishReasonLength},
		{name: "unknown passthrough", in: "pause_turn", want: "pause_turn"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAnthropicStopReason(tt.in); got != tt.want {
				t.Errorf("normalizeAnthropicStopReason(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
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
	type httpCase struct {
		name          string
		status        int
		wantKind      llmcore.ErrorKind
		wantRetryable bool
		wantFallback  bool
	}

	// HTTP 错误分类测试（Invoke 路径）
	httpErrorCases := []httpCase{
		{name: "401 auth", status: http.StatusUnauthorized, wantKind: llmcore.KindAuth},
		{name: "403 auth", status: http.StatusForbidden, wantKind: llmcore.KindAuth},
		{name: "404 not_found", status: http.StatusNotFound, wantKind: llmcore.KindNotFound},
		{name: "400 bad_request", status: http.StatusBadRequest, wantKind: llmcore.KindBadRequest},
		{name: "429 rate_limit", status: http.StatusTooManyRequests, wantKind: llmcore.KindRateLimit, wantRetryable: true, wantFallback: true},
		{name: "408 timeout", status: http.StatusRequestTimeout, wantKind: llmcore.KindTimeout, wantRetryable: true, wantFallback: true},
		{name: "500 server", status: http.StatusInternalServerError, wantKind: llmcore.KindServer, wantRetryable: true, wantFallback: true},
		{name: "502 server", status: http.StatusBadGateway, wantKind: llmcore.KindServer, wantRetryable: true, wantFallback: true},
		{name: "503 server", status: http.StatusServiceUnavailable, wantKind: llmcore.KindServer, wantRetryable: true, wantFallback: true},
	}

	for _, tt := range httpErrorCases {
		t.Run("invoke/"+tt.name, func(t *testing.T) {
			c := newAnthropicErrorClient(t, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return anthropicErrorResponse(tt.status), nil
			}))
			_, err := c.Invoke(context.Background(), &LLMRequest{
				Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !llmcore.IsKind(err, tt.wantKind) {
				t.Fatalf("error kind = %v, want %v (err=%v)", errKind(err), tt.wantKind, err)
			}
			pe, ok := err.(*llmcore.ProviderError)
			if !ok {
				t.Fatalf("err is not *ProviderError: %T", err)
			}
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

	// Invoke 路径：网络错误（原 NetworkError 测试）
	t.Run("invoke/network_error", func(t *testing.T) {
		c := newAnthropicErrorClient(t, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: connection refused")
		}))
		_, err := c.Invoke(context.Background(), &LLMRequest{
			Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
		})
		if !llmcore.IsKind(err, llmcore.KindNetwork) {
			t.Fatalf("error kind = %v, want KindNetwork", errKind(err))
		}
		pe, ok := err.(*llmcore.ProviderError)
		if !ok {
			t.Fatalf("err is not *ProviderError: %T", err)
		}
		if !pe.Retryable || !pe.Fallbackable {
			t.Errorf("network error should be retryable/fallbackable, got %+v", pe)
		}
	})

	// Stream 路径：HTTP 错误（原 StreamHTTPError 测试）
	t.Run("stream/http_error", func(t *testing.T) {
		c := newAnthropicErrorClient(t, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return anthropicErrorResponse(http.StatusTooManyRequests), nil
		}))
		_, err := c.Stream(context.Background(), &LLMRequest{
			Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
		})
		if !llmcore.IsKind(err, llmcore.KindRateLimit) {
			t.Fatalf("error kind = %v, want KindRateLimit", errKind(err))
		}
		pe, ok := err.(*llmcore.ProviderError)
		if !ok {
			t.Fatalf("err is not *ProviderError: %T", err)
		}
		if !pe.Fallbackable {
			t.Error("429 should be fallbackable")
		}
	})
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
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
		Tools: []llmcore.ToolSpec{{
			Type:     llmcore.ToolTypeFunction,
			Function: llmcore.ToolSpecFunction{Name: "search"},
		}},
	})
	if !llmcore.IsKind(err, llmcore.KindTimeout) {
		t.Fatalf("error kind = %v, want KindTimeout (err=%v)", errKind(err), err)
	}
	pe, ok := err.(*llmcore.ProviderError)
	if !ok {
		t.Fatalf("err is not *ProviderError: %T", err)
	}
	if !pe.Fallbackable {
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
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
	})
	pe, ok := err.(*llmcore.ProviderError)
	if !ok {
		t.Fatalf("err is not *ProviderError: %T", err)
	}
	if pe.Retryable || pe.Fallbackable {
		t.Errorf("protocol error should NOT be retryable/fallbackable, got %+v", pe)
	}
}

// StreamCancel 验证 context cancel 时 Stream 能正确退出且无 goroutine 泄漏
func TestAnthropicClient_StreamCancel(t *testing.T) {
	goroutineBefore := runtime.NumGoroutine()

	c := NewAnthropicClient(&AnthropicClientConfig{
		BaseURL:   "https://example.invalid",
		APIKey:    "test-key",
		ModelName: "claude-test",
		MaxTokens: 1024,
	})

	// 模拟慢速响应：发送部分数据后等待 context cancel
	slowStreamBody := strings.Join([]string{
		"event: message_start",
		`data: {"type":"message","message":{"id":"msg-1","usage":{"input_tokens":10,"output_tokens":0}}}`,
		"",
		"event: content_block_start",
		`data: {"index":0,"content_block":{"type":"text"}}`,
		"",
		"event: content_block_delta",
		`data: {"index":0,"delta":{"type":"text_delta","text":"慢"}}`,
		"", // 发送部分内容后，下一条消息会被 context cancel 阻塞
	}, "\n")

	c.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(slowStreamBody)),
		}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	deltas, err := c.Stream(ctx, &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	// 消费部分 deltas
	var got int
	for delta := range deltas {
		got++
		if got >= 2 {
			// 收到部分内容后取消 context
			cancel()
			break
		}
		// 忽略可能的 error delta
		if delta.Error != "" {
			t.Logf("stream error during partial consumption: %v", delta.Error)
		}
	}

	// 等待 goroutine 退出（Stream 的 reader goroutine 应在 cancel 后退出）
	time.Sleep(100 * time.Millisecond)
	goroutineAfter := runtime.NumGoroutine()

	// 验证：goroutine 数量应恢复到 cancel 前（允许 ±1 误差）
	// 主要验证：没有泄漏（cancel 后不应有新增 goroutine）
	if goroutineAfter > goroutineBefore+1 {
		t.Errorf("goroutine 泄漏: cancel 前=%d, cancel 后=%d", goroutineBefore, goroutineAfter)
	}

	// 验证：收到了一些 deltas（不是全部）
	if got == 0 {
		t.Error("应至少收到部分 deltas")
	}
}
