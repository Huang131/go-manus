package llmcore

import "testing"

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
// 业务期望：candidate 上下文比 current 小 → 不能 fallback
func TestCanFallbackTo_ContextShrink(t *testing.T) {
	big := fullProfile(ProtocolOpenAICompat)
	big.Capabilities.MaxContextTokens = 200000
	small := fullProfile(ProtocolOpenAICompat)
	small.Capabilities.MaxContextTokens = 8000
	if CanFallbackTo(small, big) {
		t.Fatal("expected false: context shrink")
	}
	if !CanFallbackTo(big, small) {
		t.Fatal("expected true: context expand is OK")
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

// TestCanFallbackAfterToolUse_NoTools
// 业务期望：没工具 → 允许 fallback
func TestCanFallbackAfterToolUse_NoTools(t *testing.T) {
	if !CanFallbackAfterToolUse(nil) {
		t.Fatal("expected true: no tools executed")
	}
	if !CanFallbackAfterToolUse([]ToolSpec{}) {
		t.Fatal("expected true: empty tools executed")
	}
}

// TestCanFallbackAfterToolUse_AllReadOnly
// 业务期望：全 ReadOnly → 允许 fallback
func TestCanFallbackAfterToolUse_AllReadOnly(t *testing.T) {
	tools := []ToolSpec{
		{ReadOnly: true, Function: ToolSpecFunction{Name: "search"}},
		{ReadOnly: true, Function: ToolSpecFunction{Name: "grep"}},
	}
	if !CanFallbackAfterToolUse(tools) {
		t.Fatal("expected true: all read-only")
	}
}

// TestCanFallbackAfterToolUse_HasWriteTool
// 业务期望：有非 ReadOnly 工具 → 禁止 fallback
func TestCanFallbackAfterToolUse_HasWriteTool(t *testing.T) {
	tools := []ToolSpec{
		{ReadOnly: true, Function: ToolSpecFunction{Name: "search"}},
		{ReadOnly: false, Function: ToolSpecFunction{Name: "shell"}},
	}
	if CanFallbackAfterToolUse(tools) {
		t.Fatal("expected false: shell is not read-only")
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

func fullCaps() ModelCapabilities {
	return ModelCapabilities{
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
