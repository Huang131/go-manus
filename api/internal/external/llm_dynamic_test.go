package external

import (
	"context"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/llmcore"
)

type stubLLM struct {
	name   string
	invoke func(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}

func (s *stubLLM) Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if s.invoke != nil {
		return s.invoke(ctx, req)
	}
	return &LLMResponse{Content: s.name}, nil
}

func (s *stubLLM) ModelName() string    { return s.name }
func (s *stubLLM) Temperature() float64 { return 0.7 }
func (s *stubLLM) MaxTokens() int       { return 1024 }

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
	if resp.Content != "claude-3-5-sonnet-20241022" {
		t.Fatalf("content = %q, want model name", resp.Content)
	}
}

func TestRoutedLLMFromSingleProvider(t *testing.T) {
	var gotModel string
	router := NewRoutedLLMFromSingleProvider(
		func(ctx context.Context) (*LLMRuntimeConfig, error) {
			return &LLMRuntimeConfig{
				Profile:   llmcore.ModelProfile{Protocol: llmcore.ProtocolOpenAICompat},
				ModelName: "single-model",
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			gotModel = cfg.ModelName
			return &stubLLM{name: cfg.ModelName}
		},
	)

	resp, err := router.Invoke(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if gotModel != "single-model" {
		t.Fatalf("got model %s, want single-model", gotModel)
	}
	if resp.Content != "single-model" {
		t.Fatalf("content = %q, want single-model", resp.Content)
	}
}
