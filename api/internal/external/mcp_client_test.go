package external

import (
	"testing"
)

// TestMCPClientManager_New 测试创建 MCP 客户端管理器
func TestMCPClientManager_New(t *testing.T) {
	manager := NewMCPClientManager(nil)
	if manager == nil {
		t.Error("NewMCPClientManager should return non-nil manager")
	}
}

// TestMCPClientManager_GetClient 测试获取不存在的客户端
func TestMCPClientManager_GetClient(t *testing.T) {
	manager := NewMCPClientManager(nil)
	_, ok := manager.GetClient("nonexistent")
	if ok {
		t.Error("GetClient should return false for nonexistent client")
	}
}

// TestMCPClientManager_ListAllTools_Empty 测试空管理器获取工具列表
func TestMCPClientManager_ListAllTools_Empty(t *testing.T) {
	manager := NewMCPClientManager(nil)
	tools, err := manager.ListAllTools(nil)
	if err != nil {
		t.Errorf("ListAllTools should not return error for nil config: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("ListAllTools should return empty map, got %d", len(tools))
	}
}

// TestMCPToolInfo 测试工具信息结构
func TestMCPToolInfo(t *testing.T) {
	info := MCPToolInfo{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	if info.Name != "test-tool" {
		t.Errorf("Name = %s, want test-tool", info.Name)
	}

	if info.Description != "A test tool" {
		t.Errorf("Description = %s, want A test tool", info.Description)
	}

	if info.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}
}

// TestMCPToolResult 测试工具结果结构
func TestMCPToolResult(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: "Hello, World!"},
		},
		IsError: false,
	}

	if len(result.Content) != 1 {
		t.Errorf("Content length = %d, want 1", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %s, want text", result.Content[0].Type)
	}

	if result.Content[0].Text != "Hello, World!" {
		t.Errorf("Content[0].Text = %s, want Hello, World!", result.Content[0].Text)
	}

	if result.IsError {
		t.Error("IsError should be false")
	}
}

// TestMCPToolResult_Error 测试错误结果
func TestMCPToolResult_Error(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: "Error message"},
		},
		IsError: true,
	}

	if !result.IsError {
		t.Error("IsError should be true for error result")
	}
}

// TestMCPRequest 测试 JSON-RPC 请求
func TestMCPRequest(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params: map[string]interface{}{
			"key": "value",
		},
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %s, want 2.0", req.JSONRPC)
	}

	if req.ID != 1 {
		t.Errorf("ID = %d, want 1", req.ID)
	}

	if req.Method != "tools/list" {
		t.Errorf("Method = %s, want tools/list", req.Method)
	}

	if req.Params == nil {
		t.Error("Params should not be nil")
	}
}

// TestMCPResponse 测试 JSON-RPC 响应
func TestMCPResponse(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"tools": []interface{}{},
		},
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %s, want 2.0", resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}

	if resp.Result == nil {
		t.Error("Result should not be nil")
	}

	if resp.Error != nil {
		t.Error("Error should be nil for success response")
	}
}

// TestMCPResponse_Error 测试错误响应
func TestMCPResponse_Error(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &MCPError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	if resp.Result != nil {
		t.Error("Result should be nil for error response")
	}

	if resp.Error == nil {
		t.Error("Error should not be nil for error response")
	}

	if resp.Error.Code != -32600 {
		t.Errorf("Error.Code = %d, want -32600", resp.Error.Code)
	}

	if resp.Error.Message != "Invalid Request" {
		t.Errorf("Error.Message = %s, want Invalid Request", resp.Error.Message)
	}
}

// TestMCPConfig 测试 MCP 配置
func TestMCPConfig(t *testing.T) {
	cfg := MCPConfig{
		Servers: []MCPConfigServer{
			{
				Name:    "test-server",
				Command: "echo",
				Args:    []string{"hello"},
				Env:     map[string]string{"KEY": "value"},
			},
		},
		Timeout: 30,
	}

	if len(cfg.Servers) != 1 {
		t.Errorf("Servers length = %d, want 1", len(cfg.Servers))
	}

	if cfg.Servers[0].Name != "test-server" {
		t.Errorf("Servers[0].Name = %s, want test-server", cfg.Servers[0].Name)
	}

	if cfg.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", cfg.Timeout)
	}
}

// TestStdioMCPClientConfig 测试 stdio 客户端配置
func TestStdioMCPClientConfig(t *testing.T) {
	cfg := StdioMCPClientConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		Env:     map[string]string{"DEBUG": "true"},
	}

	if cfg.Command != "npx" {
		t.Errorf("Command = %s, want npx", cfg.Command)
	}

	if len(cfg.Args) != 3 {
		t.Errorf("Args length = %d, want 3", len(cfg.Args))
	}

	if cfg.Env["DEBUG"] != "true" {
		t.Errorf("Env[DEBUG] = %s, want true", cfg.Env["DEBUG"])
	}
}

// TestMCPClientManager_Close 测试关闭空管理器
func TestMCPClientManager_Close(t *testing.T) {
	manager := NewMCPClientManager(nil)
	err := manager.Close()
	if err != nil {
		t.Errorf("Close should not return error for nil config: %v", err)
	}
}
