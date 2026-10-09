package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/mcp"
)

func TestMCPToolRegistersDynamicFunctions(t *testing.T) {
	tool := NewMCPTool()
	tool.tools = map[string]map[string]mcp.MCPToolInfo{
		"demo": {
			"search": {Name: "search", Description: "Search demo", InputSchema: map[string]interface{}{"type": "object"}},
		},
	}

	functions := tool.GetTools()
	if len(functions) != 1 {
		t.Fatalf("GetTools() count = %d, want 1", len(functions))
	}
	if functions[0]["name"] != "mcp_demo_search" {
		t.Fatalf("function name = %v, want mcp_demo_search", functions[0]["name"])
	}

	registry := NewToolRegistry()
	registry.Register(tool)
	specs := registry.GetToolsForLLM()
	if len(specs) != 1 || specs[0].Function.Name != "mcp_demo_search" {
		t.Fatalf("registry specs = %+v, want one dynamic MCP function", specs)
	}
}

func TestMCPToolInvokeWithNameRejectsUnknownFunction(t *testing.T) {
	result, err := NewMCPTool().InvokeWithName("mcp_demo_missing", context.Background(), nil)
	if err != nil {
		t.Fatalf("InvokeWithName() error = %v", err)
	}
	if result == nil || result.Success || !strings.Contains(result.Message, "未知 MCP 工具") {
		t.Fatalf("InvokeWithName() result = %+v, want unknown function error", result)
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

	if got := len(tool.GetTools()); got != 1 {
		t.Fatalf("precondition dynamic tools = %d, want 1", got)
	}

	if err := tool.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v, want nil", err)
	}

	if got := len(tool.GetTools()); got != 0 {
		t.Errorf("GetTools() after Cleanup = %d tools, want 0", got)
	}
}
