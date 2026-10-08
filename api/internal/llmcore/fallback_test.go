package llmcore

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

// TestCanFallbackTo_ProtocolMismatch
// 业务期望：OpenAI <-> Anthropic 不能 fallback
func TestCanFallbackTo_ProtocolMismatch(t *testing.T) {
	a := fullProfile(ProtocolOpenAICompat)
	b := fullProfile(ProtocolAnthropic)
	if CanFallbackTo(a, b) {
		t.Fatal("expected false: protocol mismatch")
	}
}

// TestCanFallbackTo_ContextShrink
// 业务期望：candidate 上下文比 current 小（带 SafeContextRatio）→ 不能 fallback
func TestCanFallbackTo_ContextShrink(t *testing.T) {
	big := fullProfile(ProtocolOpenAICompat)
	big.Capabilities.MaxContextTokens = 200000
	small := fullProfile(ProtocolOpenAICompat)
	small.Capabilities.MaxContextTokens = 128000 // 200000/0.8=250000，所以不能 fallback
	if CanFallbackTo(small, big) {
		t.Fatal("expected false: context shrink beyond safe ratio")
	}
	if !CanFallbackTo(big, small) {
		t.Fatal("expected true: context expand is OK")
	}
}

// TestCanFallbackTo_MaxOutputTokensShrink
// 业务期望：candidate MaxOutputTokens 比 current 小 → 不能 fallback（生成会截断）
func TestCanFallbackTo_MaxOutputTokensShrink(t *testing.T) {
	// current=4096, candidate=1024，candidate 小不能替换 current
	current := fullProfile(ProtocolOpenAICompat)
	current.Capabilities.MaxOutputTokens = 4096
	candidate := fullProfile(ProtocolOpenAICompat)
	candidate.Capabilities.MaxOutputTokens = 1024
	if CanFallbackTo(candidate, current) {
		t.Fatal("expected false: max_output_tokens shrink")
	}
	// 反向：current=4096 能替换 candidate=1024（能力更强）
	if !CanFallbackTo(current, candidate) {
		t.Fatal("expected true: max_output_tokens expand is OK")
	}
}

// TestCanFallbackTo_VisionDowngrade
// 业务期望：vision 模型不能 fallback 到非 vision
func TestCanFallbackTo_VisionDowngrade(t *testing.T) {
	withVision := fullProfile(ProtocolOpenAICompat)
	withVision.Capabilities.SupportsVision = true
	noVision := fullProfile(ProtocolOpenAICompat)
	if CanFallbackTo(noVision, withVision) {
		t.Fatal("expected false: vision downgrade")
	}
}

// TestCanFallbackTo_CapabilityEquivalent
// 业务期望：能力等价时 fallback
func TestCanFallbackTo_CapabilityEquivalent(t *testing.T) {
	a := fullProfile(ProtocolOpenAICompat)
	b := fullProfile(ProtocolOpenAICompat)
	if !CanFallbackTo(a, b) {
		t.Fatal("expected true: identical profiles")
	}
}

// TestCanFallbackAfterToolUse_NoToolsExecuted
// 业务期望：没有任何工具执行过（无 RoleTool 消息）→ 允许 fallback，
// 即使本次声明的 tools 里有写工具（声明 ≠ 执行）。
func TestCanFallbackAfterToolUse_NoToolsExecuted(t *testing.T) {
	tools := []ToolSpec{
		{ReadOnly: true, Function: ToolSpecFunction{Name: "search"}},
		{ReadOnly: false, Function: ToolSpecFunction{Name: "shell"}},
	}
	if !CanFallbackAfterToolUse(nil, tools) {
		t.Fatal("expected true: no tools executed")
	}
	if !CanFallbackAfterToolUse([]Message{
		{Role: model.RoleUser, ContentText: "hi"},
		{Role: model.RoleAssistant, ContentText: "hello"},
	}, tools) {
		t.Fatal("expected true: no RoleTool in history")
	}
}

// TestCanFallbackAfterToolUse_DeclaredWriteToolBlocks
// 场景 A 守护（声明即禁止）：历史只执行过只读工具，但本次声明的 tools
// 含非 ReadOnly 工具（尚未执行）→ 禁止 fallback。
// 跨模型续接会改变写工具的调用决策，行为漂移不可控，宁可保守。
func TestCanFallbackAfterToolUse_DeclaredWriteToolBlocks(t *testing.T) {
	tools := []ToolSpec{
		{ReadOnly: true, Function: ToolSpecFunction{Name: "search"}},
		{ReadOnly: false, Function: ToolSpecFunction{Name: "shell"}}, // 声明未执行
	}
	messages := []Message{
		{Role: model.RoleTool, Name: "search", ToolCallID: "call-1", ContentText: "result"},
	}
	if CanFallbackAfterToolUse(messages, tools) {
		t.Fatal("expected false: declared write tool blocks fallback even before execution")
	}
}

// TestCanFallbackAfterToolUse_ExecutedWriteTool
// 业务期望：已执行非 ReadOnly 工具 → 禁止 fallback。
func TestCanFallbackAfterToolUse_ExecutedWriteTool(t *testing.T) {
	tools := []ToolSpec{
		{ReadOnly: true, Function: ToolSpecFunction{Name: "search"}},
		{ReadOnly: false, Function: ToolSpecFunction{Name: "shell"}},
	}
	messages := []Message{
		{Role: model.RoleTool, Name: "shell", ToolCallID: "call-1", ContentText: "done"},
	}
	if CanFallbackAfterToolUse(messages, tools) {
		t.Fatal("expected false: shell executed (write side effect)")
	}
}

// TestCanFallbackAfterToolUse_ToolsNarrowedAfterWriteExecution
// 场景 C 守护：历史执行过写工具，但本次 tools 被收窄（不含该工具）→
// 查不到 ReadOnly 声明，必须保守禁止，不能因为"当前 tools 全只读"而放行。
func TestCanFallbackAfterToolUse_ToolsNarrowedAfterWriteExecution(t *testing.T) {
	narrowed := []ToolSpec{
		{ReadOnly: true, Function: ToolSpecFunction{Name: "search"}},
	}
	messages := []Message{
		{Role: model.RoleTool, Name: "shell", ToolCallID: "call-1", ContentText: "done"}, // 之前执行过 shell
	}
	if CanFallbackAfterToolUse(messages, narrowed) {
		t.Fatal("expected false: executed tool missing from declared tools (narrowed)")
	}
}

// TestCanFallbackAfterToolUse_ExecutedButToolsEmpty
// 场景 B 守护：已执行过工具但本次不声明任何 tools（如 Summarize/Planner 路径）→
// 无 ReadOnly 依据，保守禁止。
func TestCanFallbackAfterToolUse_ExecutedButToolsEmpty(t *testing.T) {
	messages := []Message{
		{Role: model.RoleTool, Name: "search", ToolCallID: "call-1", ContentText: "result"},
	}
	if CanFallbackAfterToolUse(messages, nil) {
		t.Fatal("expected false: no ReadOnly declaration available")
	}
}

// TestCanFallbackAfterToolUse_ExecutedWithoutName
// RoleTool 缺 Name（异常构造/历史 fixture）→ 无法判定，保守禁止。
func TestCanFallbackAfterToolUse_ExecutedWithoutName(t *testing.T) {
	tools := []ToolSpec{
		{ReadOnly: true, Function: ToolSpecFunction{Name: "search"}},
	}
	messages := []Message{
		{Role: model.RoleTool, ToolCallID: "call-1", ContentText: "result"}, // 无 Name
	}
	if CanFallbackAfterToolUse(messages, tools) {
		t.Fatal("expected false: RoleTool without Name cannot be verified")
	}
}

// TestProviderError_Kind
// 业务期望：ProviderError 携带 Kind，方便 Orchestrator 决策
func TestProviderError_Kind(t *testing.T) {
	pe := NewProviderError(KindRateLimit, "openai", "gpt-4", "rate limit hit")
	if pe.Kind != KindRateLimit {
		t.Errorf("expected kind=rate_limit, got %s", pe.Kind)
	}
	if !pe.Retryable {
		t.Error("rate_limit should be retryable")
	}
	if !pe.Fallbackable {
		t.Error("rate_limit should be fallbackable")
	}
}

// TestProviderError_AuthNotFallbackable
// 业务期望：鉴权错误不能 fallback（配错 key 换 model 也救不了）
func TestProviderError_AuthNotFallbackable(t *testing.T) {
	pe := NewProviderError(KindAuth, "openai", "gpt-4", "invalid api key")
	if pe.Fallbackable {
		t.Error("auth should NOT be fallbackable")
	}
	if pe.Retryable {
		t.Error("auth should NOT be retryable")
	}
}

// TestIsKind
// 业务期望：errors.Is/As 链能正确判断 Kind
func TestIsKind(t *testing.T) {
	pe := NewProviderError(KindContextLimit, "openai", "gpt-4", "context too long")
	if !IsKind(pe, KindContextLimit) {
		t.Error("expected IsKind to match")
	}
	if IsKind(pe, KindAuth) {
		t.Error("expected IsKind to NOT match different kind")
	}
}

func fullCaps() model.ModelCapabilities {
	return model.ModelCapabilities{
		SupportsText:                   true,
		SupportsToolCalls:              true,
		SupportsStructuredOutput:       true,
		SupportsJSONMode:               true,
		SupportsStrictStructuredOutput: true,
		SupportsStreaming:              true,
		SupportsVision:                 false,
		SupportsReasoning:              false,
		MaxContextTokens:               128000,
		MaxOutputTokens:                8192,
	}
}

func fullProfile(p ProviderProtocol) ModelProfile {
	return ModelProfile{Protocol: p, Capabilities: fullCaps()}
}
