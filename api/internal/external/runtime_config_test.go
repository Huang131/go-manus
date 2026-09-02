package external

import (
	"encoding/json"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/model"
)

func TestProtocolFromProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		baseURL  string
		want     string
	}{
		{name: "anthropic provider", provider: "anthropic", want: "anthropic"},
		{name: "claude model", provider: "openai", model: "claude-3-5-sonnet-20241022", want: "anthropic"},
		{name: "anthropic url", provider: "custom", baseURL: "https://api.anthropic.com", want: "anthropic"},
		{name: "default openai compat", provider: "openai", model: "gpt-4o", want: "openai_compat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProtocolFromProvider(tt.provider, tt.model, tt.baseURL)
			if string(got) != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestBuildRuntimeConfigFromModel(t *testing.T) {
	reqExtra := json.RawMessage(`{"reasoning_effort":"none"}`)
	m := &model.LLMModel{
		ID:          "1",
		Provider:    "anthropic",
		BaseURL:     "https://api.anthropic.com",
		ModelName:   "claude-3-5-sonnet-20241022",
		Temperature: 0.2,
		MaxTokens:   4096,
		RequestPolicy: model.RequestPolicy{
			DefaultTemperature: func() *float64 { v := 0.1; return &v }(),
			DefaultMaxTokens:   func() *int { v := 2048; return &v }(),
			ReasoningMode:      model.ReasoningOff,
			Extra: map[string]json.RawMessage{
				"reasoning_effort": reqExtra,
			},
		},
		CostPolicy: model.CostPolicy{
			InputPricePerMTokens:  1.5,
			OutputPricePerMTokens: 2.5,
			Currency:              "USD",
		},
	}

	cfg := BuildRuntimeConfigFromModel(m, 17)
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.Profile.Protocol != "anthropic" {
		t.Fatalf("protocol = %s, want anthropic", cfg.Profile.Protocol)
	}
	if cfg.ToolCallTimeout != 17 {
		t.Fatalf("timeout = %d, want 17", cfg.ToolCallTimeout)
	}
	if cfg.Profile.RequestPolicy.Extra["reasoning_effort"].Kind != "json" {
		t.Fatalf("extra kind = %s, want json", cfg.Profile.RequestPolicy.Extra["reasoning_effort"].Kind)
	}
	if cfg.Profile.RequestPolicy.ReasoningMode != "off" {
		t.Fatalf("reasoning mode = %s, want off", cfg.Profile.RequestPolicy.ReasoningMode)
	}
	if cfg.Temperature != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", cfg.Temperature)
	}
	if cfg.MaxTokens != 4096 {
		t.Fatalf("max tokens = %d, want 4096", cfg.MaxTokens)
	}
	if cfg.Profile.CostPolicy.InputPricePerMTokens != 1.5 {
		t.Fatalf("cost input = %v, want 1.5", cfg.Profile.CostPolicy.InputPricePerMTokens)
	}
}
