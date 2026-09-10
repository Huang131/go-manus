package model

import (
	"encoding/json"
	"testing"
)

func TestLLMConfigResponseDoesNotExposeAPIKey(t *testing.T) {
	response := NewLLMConfigResponse(&LLMConfig{
		BaseURL:   "https://example.test",
		APIKey:    "secret",
		ModelName: "model",
	})

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if containsJSONKey(data, "api_key") {
		t.Fatalf("response contains api_key: %s", data)
	}
	if !containsJSONKey(data, "api_key_configured") {
		t.Fatalf("response does not contain api_key_configured: %s", data)
	}
}

func TestLLMConfigRequestPreservesAPIKeyForInternalUpdate(t *testing.T) {
	cfg := (LLMConfigRequest{
		BaseURL:   "https://example.test",
		APIKey:    "secret",
		ModelName: "model",
	}).NewLLMConfig()
	if cfg.APIKey != "secret" {
		t.Fatalf("NewLLMConfig().APIKey = %q, want secret", cfg.APIKey)
	}
}

func containsJSONKey(data []byte, key string) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return false
	}
	_, ok := fields[key]
	return ok
}
