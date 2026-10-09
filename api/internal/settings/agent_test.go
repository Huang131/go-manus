package settings

import "testing"

func TestDefaultAgentSettings(t *testing.T) {
	got := DefaultAgentSettings()
	if got.MaxIterations != 10 || got.MaxRetries != 3 || got.MaxSearchResults != 10 {
		t.Fatalf("DefaultAgentSettings() = %+v, want 10/3/10", got)
	}
}

func TestAgentSettingsValidateAcceptsBoundaries(t *testing.T) {
	tests := []AgentSettings{
		{MaxIterations: 1, MaxRetries: 1, MaxSearchResults: 1},
		{MaxIterations: 100, MaxRetries: 5, MaxSearchResults: 10},
	}
	for _, cfg := range tests {
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate(%+v) error = %v", cfg, err)
		}
	}
}

func TestAgentSettingsValidateRejectsOutOfRangeValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  AgentSettings
	}{
		{name: "iterations below minimum", cfg: AgentSettings{MaxIterations: 0, MaxRetries: 3, MaxSearchResults: 10}},
		{name: "iterations above maximum", cfg: AgentSettings{MaxIterations: 101, MaxRetries: 3, MaxSearchResults: 10}},
		{name: "retries below minimum", cfg: AgentSettings{MaxIterations: 10, MaxRetries: 0, MaxSearchResults: 10}},
		{name: "retries above maximum", cfg: AgentSettings{MaxIterations: 10, MaxRetries: 6, MaxSearchResults: 10}},
		{name: "search below minimum", cfg: AgentSettings{MaxIterations: 10, MaxRetries: 3, MaxSearchResults: 0}},
		{name: "search above provider maximum", cfg: AgentSettings{MaxIterations: 10, MaxRetries: 3, MaxSearchResults: 11}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil {
				t.Fatalf("Validate(%+v) error = nil, want range error", tt.cfg)
			}
		})
	}
}
