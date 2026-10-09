package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Huang131/go-manus/api/internal/mcp"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestMCPToolRegistersDynamicFunctions(t *testing.T) {
	tool := NewMCPTool()
	tool.functions = map[string]mcpFunction{
		"mcp_demo_search": {
			serverName: "demo",
			toolName:   "search",
			tool:       mcp.MCPToolInfo{Name: "search", Description: "Search demo", InputSchema: map[string]interface{}{"type": "object"}},
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
	if err == nil {
		t.Fatal("InvokeWithName() error = nil, want infrastructure error")
	}
	if result != nil {
		t.Fatalf("InvokeWithName() result = %+v, want nil with infrastructure error", result)
	}
}

func TestMCPToolCleanupDropsLoadedTools(t *testing.T) {
	tool := NewMCPTool()
	tool.functions = map[string]mcpFunction{
		"mcp_demo_search": {serverName: "demo", toolName: "search", tool: mcp.MCPToolInfo{Name: "search"}},
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

func TestMCPToolInitializeRejectsDuplicateFunctionNames(t *testing.T) {
	manager := &fakeMCPManager{tools: map[string][]mcp.MCPToolInfo{
		"a_b": {{Name: "c"}},
		"a":   {{Name: "b_c"}},
	}}
	tool := NewMCPTool()
	tool.newManager = func(*model.MCPConfig) mcpToolManager { return manager }

	err := tool.Initialize(context.Background(), &model.MCPConfig{})
	if err == nil {
		t.Fatal("Initialize() error = nil, want duplicate function error")
	}
	if !manager.closed {
		t.Fatal("Initialize() did not close manager after duplicate function error")
	}
	if got := len(tool.GetTools()); got != 0 {
		t.Fatalf("GetTools() = %d after failed initialization, want 0", got)
	}
}

func TestMCPToolInitializeCleansUpManagerFailure(t *testing.T) {
	manager := &fakeMCPManager{initializeErr: errors.New("connect failed")}
	tool := NewMCPTool()
	tool.newManager = func(*model.MCPConfig) mcpToolManager { return manager }

	err := tool.Initialize(context.Background(), &model.MCPConfig{})
	if err == nil {
		t.Fatal("Initialize() error = nil, want manager failure")
	}
	if !manager.closed {
		t.Fatal("Initialize() did not close failed manager")
	}
}

func TestMCPToolInvokeWithNameSeparatesTransportAndRemoteErrors(t *testing.T) {
	t.Run("transport error returns Go error", func(t *testing.T) {
		client := &fakeMCPClient{callErr: errors.New("connection reset")}
		tool := initializedMCPTool(t, client)

		result, err := tool.InvokeWithName("mcp_demo_search", context.Background(), nil)
		if err == nil {
			t.Fatal("InvokeWithName() error = nil, want transport error")
		}
		if result != nil {
			t.Fatalf("InvokeWithName() result = %+v, want nil", result)
		}
	})

	t.Run("remote tool error returns failed result", func(t *testing.T) {
		client := &fakeMCPClient{result: &mcp.MCPToolResult{
			IsError: true,
			Content: []mcp.MCPContent{{Type: "text", Text: "invalid query"}},
		}}
		tool := initializedMCPTool(t, client)

		result, err := tool.InvokeWithName("mcp_demo_search", context.Background(), nil)
		if err != nil {
			t.Fatalf("InvokeWithName() error = %v, want nil", err)
		}
		if result == nil || result.Success || result.Message != "invalid query\n" {
			t.Fatalf("InvokeWithName() result = %+v, want failed remote result", result)
		}
	})
}

func initializedMCPTool(t *testing.T, client mcp.MCPClient) *MCPTool {
	t.Helper()
	manager := &fakeMCPManager{
		tools: map[string][]mcp.MCPToolInfo{
			"demo": {{Name: "search"}},
		},
		clients: map[string]mcp.MCPClient{"demo": client},
	}
	tool := NewMCPTool()
	tool.newManager = func(*model.MCPConfig) mcpToolManager { return manager }
	if err := tool.Initialize(context.Background(), &model.MCPConfig{}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	return tool
}

type fakeMCPManager struct {
	initializeErr error
	listErr       error
	tools         map[string][]mcp.MCPToolInfo
	clients       map[string]mcp.MCPClient
	closed        bool
}

func (m *fakeMCPManager) Initialize(context.Context) error { return m.initializeErr }

func (m *fakeMCPManager) GetClient(name string) (mcp.MCPClient, bool) {
	client, ok := m.clients[name]
	return client, ok
}

func (m *fakeMCPManager) ListAllTools(context.Context) (map[string][]mcp.MCPToolInfo, error) {
	return m.tools, m.listErr
}

func (m *fakeMCPManager) Close() error {
	m.closed = true
	return nil
}

type fakeMCPClient struct {
	callErr error
	result  *mcp.MCPToolResult
}

func (*fakeMCPClient) Connect(context.Context) error { return nil }

func (*fakeMCPClient) ListTools(context.Context) ([]mcp.MCPToolInfo, error) { return nil, nil }

func (c *fakeMCPClient) CallTool(context.Context, string, map[string]interface{}) (*mcp.MCPToolResult, error) {
	return c.result, c.callErr
}

func (*fakeMCPClient) Close() error { return nil }
