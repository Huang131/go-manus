package agent

import (
	"context"
	"strings"
	"testing"
)

func TestMCPToolInvokeValidatesParameters(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		want   string
	}{
		{name: "missing server", params: map[string]interface{}{"tool": "search"}, want: "server"},
		{name: "empty server", params: map[string]interface{}{"server": " ", "tool": "search"}, want: "server"},
		{name: "invalid server type", params: map[string]interface{}{"server": 1, "tool": "search"}, want: "server"},
		{name: "missing tool", params: map[string]interface{}{"server": "demo"}, want: "tool"},
		{name: "empty tool", params: map[string]interface{}{"server": "demo", "tool": " "}, want: "tool"},
		{name: "invalid tool type", params: map[string]interface{}{"server": "demo", "tool": true}, want: "tool"},
		{name: "invalid params type", params: map[string]interface{}{"server": "demo", "tool": "search", "params": "bad"}, want: "params"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewMCPTool().Invoke(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Invoke() error = %v", err)
			}
			if result == nil || result.Success {
				t.Fatalf("Invoke() result = %+v, want validation error", result)
			}
			if !strings.Contains(result.Message, tt.want) {
				t.Fatalf("Invoke() message = %q, want field %q", result.Message, tt.want)
			}
		})
	}
}

func TestMCPToolInvokeAllowsMissingParams(t *testing.T) {
	result, err := NewMCPTool().Invoke(context.Background(), map[string]interface{}{
		"server": "demo",
		"tool":   "search",
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result == nil || result.Success || result.Message != "MCP manager not initialized" {
		t.Fatalf("Invoke() result = %+v, want manager initialization error", result)
	}
}
