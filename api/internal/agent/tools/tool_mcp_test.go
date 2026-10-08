package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/mcp"
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

// TestMCPToolCleanupDropsLoadedTools 覆盖 Cleanup 的重置语义：清理后不得再暴露
// 已加载的工具。该路径此前零覆盖，且顺带清理了一个只写不读的死字段 config。
func TestMCPToolCleanupDropsLoadedTools(t *testing.T) {
	tool := NewMCPTool()
	// 直接注入已加载状态，模拟 Initialize 成功后的 tools 缓存（避免依赖真实 MCP server）。
	tool.tools = map[string]map[string]mcp.MCPToolInfo{
		"demo": {"search": {Name: "search"}},
	}

	if !tool.HasTool("search") {
		t.Fatal("precondition failed: injected tool should be visible")
	}

	if err := tool.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v, want nil", err)
	}

	if tool.HasTool("search") {
		t.Error("Cleanup() should drop loaded tools")
	}
	if got := len(tool.GetToolsForLLM()); got != 0 {
		t.Errorf("GetToolsForLLM() after Cleanup = %d tools, want 0", got)
	}
}
