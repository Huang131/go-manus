package llm

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
)

func TestDynamicLLM_UsesFactory(t *testing.T) {
	var gotProtocol string
	d := NewDynamicLLMWithFactory(
		func(ctx context.Context) (*LLMRuntimeConfig, error) {
			return &LLMRuntimeConfig{
				Profile:   llmcore.ModelProfile{Protocol: llmcore.ProtocolAnthropic},
				BaseURL:   "https://api.anthropic.com",
				ModelName: "claude-3-5-sonnet-20241022",
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			gotProtocol = string(cfg.Profile.Protocol)
			return &stubLLM{name: cfg.ModelName}
		},
	)

	resp, err := d.Invoke(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if gotProtocol != "anthropic" {
		t.Fatalf("got protocol %s, want anthropic", gotProtocol)
	}
	if resp.Message.ContentText != "claude-3-5-sonnet-20241022" {
		t.Fatalf("content = %q, want model name", resp.Message.ContentText)
	}
}
