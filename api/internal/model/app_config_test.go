package model

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
)

func TestLLMConfigResponse_DoesNotExposeAPIKey(t *testing.T) {
	response := NewLLMConfigResponse(&LLMConfig{
		BaseURL:   "https://example.test",
		APIKey:    "secret",
		ModelName: "model",
	})

	data, err := sonic.MarshalString(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	// api_key 不应出现在响应中
	if strings.Contains(data, `"api_key"`) {
		t.Fatalf("response should not contain api_key, got: %s", data)
	}
	// api_key_configured 应该出现
	if !strings.Contains(data, `"api_key_configured"`) {
		t.Fatalf("response should contain api_key_configured, got: %s", data)
	}
}

func TestLLMConfigRequest_PreservesAPIKeyForInternalUpdate(t *testing.T) {
	cfg := LLMConfigRequest{
		BaseURL:   "https://example.test",
		APIKey:    "secret",
		ModelName: "model",
	}.NewLLMConfig()

	if cfg.APIKey != "secret" {
		t.Fatalf("NewLLMConfig().APIKey = %q, want secret", cfg.APIKey)
	}
}

func TestLLMConfigRequest_MarshalJSON(t *testing.T) {
	req := LLMConfigRequest{
		BaseURL:     "https://example.test",
		APIKey:      "secret",
		ModelName:   "model",
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	data, err := sonic.MarshalString(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	// 解析后检查各字段是否正确序列化
	var fields map[string]interface{}
	if err := sonic.UnmarshalString(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if v := fields["base_url"]; v != "https://example.test" {
		t.Fatalf("base_url = %v, want https://example.test", v)
	}
	if v := fields["api_key"]; v != "secret" {
		t.Fatalf("api_key = %v, want secret", v)
	}
	if v := fields["model_name"]; v != "model" {
		t.Fatalf("model_name = %v, want model", v)
	}
}

func TestLLMConfigResponse_APIKeyConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   *LLMConfig
		expected bool
	}{
		{
			name:     "with APIKey",
			config:   &LLMConfig{BaseURL: "https://example.test", APIKey: "secret"},
			expected: true,
		},
		{
			name:     "without APIKey",
			config:   &LLMConfig{BaseURL: "https://example.test"},
			expected: false,
		},
		{
			name:     "empty APIKey",
			config:   &LLMConfig{BaseURL: "https://example.test", APIKey: ""},
			expected: false,
		},
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := NewLLMConfigResponse(tt.config)

			data, err := sonic.MarshalString(response)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}

			hasKey := strings.Contains(data, `"api_key_configured":true`)
			if tt.expected && !hasKey {
				t.Fatalf("expected api_key_configured=true, got: %s", data)
			}
			if !tt.expected && hasKey {
				t.Fatalf("expected api_key_configured=false, got: %s", data)
			}
		})
	}
}
