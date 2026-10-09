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
	tool.tools["demo"] = map[string]mcp.MCPToolInfo{
		"search": {Name: "search"},
	}
	if !usableMCPTool(tool) {
		t.Fatal("MCP tool with discovered function considered unusable")
	}
}
