package external

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

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
