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
	if tc.Name != "get_weather" {
		t.Errorf("expected name=get_weather, got %q", tc.Name)
	}
	if tc.Arguments != `{"city":"上海"}` {
		t.Errorf("expected arguments={\"city\":\"上海\"}, got %q", tc.Arguments)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason=tool_calls, got %q", resp.FinishReason)
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
	if resp.Message.ToolCalls[0].Name != "search_1" {
		t.Errorf("expected calls[0].name=search_1, got %q", resp.Message.ToolCalls[0].Name)
	}
	if resp.Message.ToolCalls[1].Name != "search_2" {
		t.Errorf("expected calls[1].name=search_2, got %q", resp.Message.ToolCalls[1].Name)
	}
	if resp.Message.ToolCalls[1].Arguments != `{"q":"bar"}` {
		t.Errorf("calls[1] args wrong: %q", resp.Message.ToolCalls[1].Arguments)
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

// TestPlanSchema_Required 阶段 1a 提前定义 PlanSchema
// 业务期望：必填字段、enum、pattern 等约束都在
func TestPlanSchema_Required(t *testing.T) {
	if PlanSchema["type"] != "object" {
		t.Fatalf("expected type=object, got %v", PlanSchema["type"])
	}
	required, ok := PlanSchema["required"].([]string)
	if !ok {
		t.Fatalf("required is not []string: %T", PlanSchema["required"])
	}
	expectRequired := []string{"message", "goal", "title", "language", "steps"}
	for _, r := range expectRequired {
		found := false
		for _, rr := range required {
			if rr == r {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PlanSchema.required missing %q", r)
		}
	}
	props, ok := PlanSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties is not map")
	}
	// language 必须是 enum
	lang, ok := props["language"].(map[string]interface{})
	if !ok {
		t.Fatal("language property missing")
	}
	enum, ok := lang["enum"].([]string)
	if !ok || len(enum) < 2 {
		t.Errorf("language.enum 缺失或不足: %v", lang["enum"])
	}
}

// TestEstimateContextTokens 业务期望：粗估 1 token ≈ 4 char（基于 ASCII）
// 注：中文是 1 char ≈ 1 token（甚至 1 char 算 1.5+ token），所以估算用 ASCII 更稳定
func TestEstimateContextTokens(t *testing.T) {
	msgs := []Message{
		{ContentText: repeat("a", 16)}, // 16 chars
		{ContentText: repeat("b", 16)}, // 16 chars
		{ContentText: repeat("c", 8)},  // 8 chars
	}
	// total = 40 chars / 4 = 10 tokens
	tokens := EstimateContextTokens(msgs)
	if tokens != 10 {
		t.Errorf("expected 10 tokens, got %d", tokens)
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
