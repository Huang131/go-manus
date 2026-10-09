package tools

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/mcp"
)

func TestUsableMCPToolRequiresDiscoveredFunction(t *testing.T) {
	if usableMCPTool(nil) {
		t.Fatal("usableMCPTool(nil) = true, want false")
	}
	if usableMCPTool(NewMCPTool()) {
		t.Fatal("empty MCP tool considered usable")
	}

	tool := NewMCPTool()
	tool.functions["mcp_demo_search"] = mcpFunction{
		serverName: "demo",
		toolName:   "search",
		tool:       mcp.MCPToolInfo{Name: "search"},
	}
	if !usableMCPTool(tool) {
		t.Fatal("MCP tool with discovered function considered unusable")
	}
}

func TestToolProviderDefersRetiredStateCleanupUntilRelease(t *testing.T) {
	provider := NewToolProvider(nil, nil, nil, nil, nil, nil)
	set := provider.Acquire(10)
	oldState := provider.current

	if err := provider.ReloadMCPConfig(nil, nil); err != nil {
		t.Fatalf("ReloadMCPConfig(nil) error = %v", err)
	}
	if oldState.cleaned {
		t.Fatal("retired state cleaned while a ToolSet still held it")
	}

	set.Release()
	if !oldState.cleaned {
		t.Fatal("retired state was not cleaned after ToolSet release")
	}
}

func TestToolProviderKeepsSharedResourceAcrossMCPReload(t *testing.T) {
	provider := NewToolProvider(nil, nil, nil, nil, nil, nil)
	sharedA2A := NewA2ATool()
	oldMCP := NewMCPTool()
	provider.current = newToolSetState(oldMCP, sharedA2A)
	oldState := provider.current
	set := provider.Acquire(10)

	if err := provider.swapMCP(NewMCPTool()); err != nil {
		t.Fatalf("swapMCP() error = %v", err)
	}
	if oldState.a2a.owners != 2 {
		t.Fatalf("shared A2A owners = %d, want 2", oldState.a2a.owners)
	}

	set.Release()
	if oldState.a2a.owners != 1 {
		t.Fatalf("shared A2A owners after release = %d, want 1", oldState.a2a.owners)
	}
	provider.Cleanup()
}
