package llmcore

import (
	"testing"
)

// TestMergeDeltas_ContentOnly 场景 1：纯 content 流
// 业务期望：拼出完整 content，reasoning 空，tool_calls 空
func TestMergeDeltas_ContentOnly(t *testing.T) {
	deltas := []LLMDelta{
		{ContentText: "你好，"},
		{ContentText: "我"},
		{ContentText: "是助手。"},
		{FinishReason: "stop", Usage: &Usage{PromptTokens: 10, CompletionTokens: 3, TotalTokens: 13}},
	}
	resp := MergeDeltas("test-model", deltas)

	if resp.Message.ContentText != "你好，我是助手。" {
		t.Fatalf("expected '你好，我是助手。', got %q", resp.Message.ContentText)
	}
	if resp.Message.Reasoning != "" {
		t.Fatalf("expected empty reasoning, got %q", resp.Message.Reasoning)
	}
	if len(resp.Message.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(resp.Message.ToolCalls))
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("expected finish_reason=stop, got %q", resp.FinishReason)
	}
	if resp.Usage.TotalTokens != 13 {
		t.Fatalf("expected usage.TotalTokens=13, got %d", resp.Usage.TotalTokens)
	}
}

// TestMergeDeltas_ReasoningAndContent 场景 2：reasoning + content 同时返回
// 模拟 deepseek-v4-flash / kimi-k3 的"双输出"
// 业务期望：reasoning 字段累加到 Message.Reasoning，content 独立累加；不混淆
func TestMergeDeltas_ReasoningAndContent(t *testing.T) {
	deltas := []LLMDelta{
		{Reasoning: "思考中..."},
		{Reasoning: "先分析用户问题。"},
		{ContentText: "最终回答。"},
		{FinishReason: "stop"},
	}
	resp := MergeDeltas("deepseek-v4-flash", deltas)

	if resp.Message.Reasoning != "思考中...先分析用户问题。" {
		t.Fatalf("expected reasoning=思考中...先分析用户问题。, got %q", resp.Message.Reasoning)
	}
	if resp.Message.ContentText != "最终回答。" {
		t.Fatalf("expected content=最终回答。, got %q", resp.Message.ContentText)
	}
	// 关键：reasoning 不能"污染" content
	if resp.Message.ContentText == resp.Message.Reasoning {
		t.Fatal("reasoning leaked into content")
	}
}

// TestMergeDeltas_ReasoningOnly 场景 3：只有 reasoning 没有 content
// 模拟 glm-5.2 默认思考模式
// 业务期望：ContentText 为空，Reasoning 累加；Agent 层应决定是否提示用户
func TestMergeDeltas_ReasoningOnly(t *testing.T) {
	deltas := []LLMDelta{
		{Reasoning: "分析任务..."},
		{Reasoning: "拆解步骤..."},
		{FinishReason: "stop"},
	}
	resp := MergeDeltas("glm-5.2", deltas)

	if resp.Message.ContentText != "" {
		t.Fatalf("expected empty content, got %q", resp.Message.ContentText)
	}
	if resp.Message.Reasoning != "分析任务...拆解步骤..." {
		t.Fatalf("expected reasoning 累加, got %q", resp.Message.Reasoning)
	}
}

// TestMergeDeltas_ToolCall 场景 4：tool_call 流（跨 chunk 拼 arguments）
// 模拟 sensenova / siliconflow / OpenAI 标准流式 tool_call
// 业务期望：跨 chunk 累积 arguments；finish_reason=tool_calls 才算有 tool_calls
func TestMergeDeltas_ToolCall(t *testing.T) {
	deltas := []LLMDelta{
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ID: "call_abc", Type: "function", Name: "get_weather"},
		}},
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ArgumentsDelta: `{"city":`},
		}},
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ArgumentsDelta: `"上海"}`},
		}},
		{FinishReason: "tool_calls", Usage: &Usage{PromptTokens: 5, CompletionTokens: 8, TotalTokens: 13}},
	}
	resp := MergeDeltas("MiniMax-M2.7", deltas)

	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("expected id=call_abc, got %q", tc.ID)
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("expected name=get_weather, got %q", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"city":"上海"}` {
		t.Errorf("expected arguments={\"city\":\"上海\"}, got %q", tc.Function.Arguments)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason=tool_calls, got %q", resp.FinishReason)
	}
}

// TestMergeDeltas_OnlyCanonicalFinishReasonKeepsToolCalls
// 契约：llmcore 只认 canonical 的 tool_calls。
// 厂商原始值（tool_use / function_call）必须由 Adapter 翻译后再进入本层，
// 未翻译的原始值一律视为非工具调用终止，丢弃 ToolCalls。
func TestMergeDeltas_OnlyCanonicalFinishReasonKeepsToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantCalls int
	}{
		{name: "canonical tool calls", in: FinishReasonToolCalls, wantCalls: 1},
		{name: "canonical stop", in: FinishReasonStop, wantCalls: 0},
		{name: "raw anthropic tool use", in: "tool_use", wantCalls: 0},
		{name: "raw legacy function call", in: "function_call", wantCalls: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := MergeDeltas("test", []LLMDelta{
				{ToolCalls: []ToolCallDelta{{Index: 0, ID: "call-1", Name: "search"}}},
				{FinishReason: tt.in},
			})
			if len(resp.Message.ToolCalls) != tt.wantCalls {
				t.Fatalf("finish reason %q: tool calls = %d, want %d",
					tt.in, len(resp.Message.ToolCalls), tt.wantCalls)
			}
		})
	}
}

// TestMergeDeltas_CanonicalToolCallsRetainsToolCall
// 业务期望：Adapter 归一化后的 tool_calls 能完整保留拼装好的 arguments
func TestMergeDeltas_CanonicalToolCallsRetainsToolCall(t *testing.T) {
	resp := MergeDeltas("claude-test", []LLMDelta{
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ID: "call-1", Type: ToolTypeFunction, Name: "search"},
		}},
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ArgumentsDelta: `{"q":"go"}`},
		}},
		{FinishReason: FinishReasonToolCalls},
	})
	if resp.FinishReason != FinishReasonToolCalls {
		t.Fatalf("finish reason = %q, want %q", resp.FinishReason, FinishReasonToolCalls)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(resp.Message.ToolCalls))
	}
	if resp.Message.ToolCalls[0].Function.Arguments != `{"q":"go"}` {
		t.Fatalf("arguments = %q, want {\"q\":\"go\"}", resp.Message.ToolCalls[0].Function.Arguments)
	}
}

// TestMergeDeltas_MultipleToolCalls 场景 5：并行多 tool_call
// 业务期望：按 Index 正确归位
func TestMergeDeltas_MultipleToolCalls(t *testing.T) {
	deltas := []LLMDelta{
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ID: "c1", Type: "function", Name: "search_1"},
			{Index: 1, ID: "c2", Type: "function", Name: "search_2"},
		}},
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ArgumentsDelta: `{"q":"foo"}`},
			{Index: 1, ArgumentsDelta: `{"q":"bar"}`},
		}},
		{FinishReason: "tool_calls"},
	}
	resp := MergeDeltas("test", deltas)

	if len(resp.Message.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(resp.Message.ToolCalls))
	}
	if resp.Message.ToolCalls[0].Function.Name != "search_1" {
		t.Errorf("expected calls[0].name=search_1, got %q", resp.Message.ToolCalls[0].Function.Name)
	}
	if resp.Message.ToolCalls[1].Function.Name != "search_2" {
		t.Errorf("expected calls[1].name=search_2, got %q", resp.Message.ToolCalls[1].Function.Name)
	}
	if resp.Message.ToolCalls[1].Function.Arguments != `{"q":"bar"}` {
		t.Errorf("calls[1] args wrong: %q", resp.Message.ToolCalls[1].Function.Arguments)
	}
}

// TestMergeDeltas_ToolCallThenContent 场景 6：tool_call 之后又有 content
// 某些 provider 会在 tool_call 流末尾补一句自然语言
// 业务期望：tool_calls 仍保留，content 也累加
func TestMergeDeltas_ToolCallThenContent(t *testing.T) {
	deltas := []LLMDelta{
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ID: "c1", Type: "function", Name: "search"},
		}},
		{ToolCalls: []ToolCallDelta{
			{Index: 0, ArgumentsDelta: `{}`},
		}},
		{ContentText: "我帮你查一下。"},
		{FinishReason: "stop"}, // 注意：这里 stop 不是 tool_calls
	}
	resp := MergeDeltas("test", deltas)

	// finish_reason=stop → tool_calls 应被清空
	if len(resp.Message.ToolCalls) != 0 {
		t.Errorf("expected no tool calls (finish=stop), got %d", len(resp.Message.ToolCalls))
	}
	if resp.Message.ContentText != "我帮你查一下。" {
		t.Errorf("content lost: %q", resp.Message.ContentText)
	}
}

// TestEstimateContextTokens_ASCII 业务期望：ASCII 1 token ≈ 4 chars
func TestEstimateContextTokens_ASCII(t *testing.T) {
	msgs := []Message{
		{ContentText: repeat("a", 16)}, // 16 ASCII chars
		{ContentText: repeat("b", 16)}, // 16 ASCII chars
		{ContentText: repeat("c", 8)},  // 8 ASCII chars
	}
	// 40 ASCII chars / 4 = 10 tokens
	tokens := EstimateContextTokens(msgs)
	if tokens != 10 {
		t.Errorf("expected 10 tokens, got %d", tokens)
	}
}

// TestEstimateContextTokens_NonASCII 业务期望：中文按字节计（3字节/字，保守偏大）
func TestEstimateContextTokens_NonASCII(t *testing.T) {
	msgs := []Message{
		{ContentText: "你好世界"}, // 12 字节 (4字*3字节) → 12/4 = 3 tokens
	}
	tokens := EstimateContextTokens(msgs)
	if tokens != 3 {
		t.Errorf("expected 3 tokens, got %d", tokens)
	}
}

// TestEstimateContextTokens_Mixed 业务期望：混合内容正确计算
func TestEstimateContextTokens_Mixed(t *testing.T) {
	msgs := []Message{
		{ContentText: "hello"},  // 5 ASCII 字节 → 5/4 = 1 token
		{ContentText: "你好"},     // 6 字节 (2字*3字节) → 6/4 = 1 token
		{ContentText: " world"}, // 6 ASCII 字节 → 6/4 = 1 token
	}
	// 5 + 6 + 6 = 17 字节，17/4 = 4 tokens (向下取整)
	tokens := EstimateContextTokens(msgs)
	if tokens != 4 {
		t.Errorf("expected 4 tokens (17 bytes / 4), got %d", tokens)
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
