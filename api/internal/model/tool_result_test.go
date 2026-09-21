package model

import (
	"strings"
	"testing"
)

func TestNewToolResult(t *testing.T) {
	data := map[string]string{"key": "value"}
	result := NewToolResult(data)

	if !result.Success {
		t.Error("NewToolResult() Success should be true")
	}
	if result.Data == nil {
		t.Error("NewToolResult() Data should not be nil")
	}
}

func TestNewToolResultWithMessage(t *testing.T) {
	result := NewToolResultWithMessage("success message", "data")

	if !result.Success {
		t.Error("NewToolResultWithMessage() Success should be true")
	}
	if result.Message != "success message" {
		t.Errorf("NewToolResultWithMessage() Message = %s, want success message", result.Message)
	}
}

func TestNewToolError(t *testing.T) {
	result := NewToolError("error message")

	if result.Success {
		t.Error("NewToolError() Success should be false")
	}
	if result.Message != "error message" {
		t.Errorf("NewToolError() Message = %s, want error message", result.Message)
	}
}

func TestToolResult_FromSandbox(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		wantSuccess bool
	}{
		{"success code 200", 200, true},
		{"success code 201", 201, true},
		{"success code 204", 204, true},
		{"error code 400", 400, false},
		{"error code 401", 401, false},
		{"server error 500", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ToolResult{}
			result.FromSandbox(tt.code, "", nil)
			if result.Success != tt.wantSuccess {
				t.Errorf("FromSandbox() Success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

// TestToolResult_LLMJSONStripsDisplay 保证仅用于 UI 的重数据不会进入 LLM 上下文。
func TestToolResult_LLMJSONStripsDisplay(t *testing.T) {
	result := NewToolResultWithMessage("ok", map[string]interface{}{"path": "/tmp/a.png"}).
		WithDisplay("screenshot", "data:image/png;base64,AAAA")

	if !strings.Contains(result.JSON(), "data:image/png;base64,AAAA") {
		t.Fatalf("JSON() should keep display for UI, got %s", result.JSON())
	}
	if strings.Contains(result.LLMJSON(), "base64") {
		t.Fatalf("LLMJSON() should strip display, got %s", result.LLMJSON())
	}
	if !strings.Contains(result.LLMJSON(), "/tmp/a.png") {
		t.Fatalf("LLMJSON() should keep data, got %s", result.LLMJSON())
	}
}

// TestToolResult_WithDisplayIsChainable 验证多次追加展示数据不互相覆盖。
func TestToolResult_WithDisplayIsChainable(t *testing.T) {
	result := NewToolResult(nil).WithDisplay("screenshot", "a").WithDisplay("url", "https://example.com")

	if result.Display["screenshot"] != "a" || result.Display["url"] != "https://example.com" {
		t.Fatalf("Display = %#v, want both entries", result.Display)
	}
}

func TestToolResult_LLMJSONOnNil(t *testing.T) {
	var result *ToolResult
	if !strings.Contains(result.LLMJSON(), "success") {
		t.Fatalf("nil LLMJSON() = %q, want error payload", result.LLMJSON())
	}
}
