package model

import (
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
