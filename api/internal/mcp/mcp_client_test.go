package mcp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MCPClientManager 测试
// ============================================================================

func TestMCPClientManager_New(t *testing.T) {
	manager := NewMCPClientManager(nil)
	require.NotNil(t, manager, "NewMCPClientManager should return non-nil manager")
}

func TestMCPClientManager_GetClient(t *testing.T) {
	manager := NewMCPClientManager(nil)
	_, ok := manager.GetClient("nonexistent")
	assert.False(t, ok, "GetClient should return false for nonexistent client")
}

func TestMCPClientManager_ListAllTools_Empty(t *testing.T) {
	manager := NewMCPClientManager(nil)
	tools, err := manager.ListAllTools(nil)
	require.NoError(t, err, "ListAllTools should not return error for nil config")
	assert.Empty(t, tools, "ListAllTools should return empty map")
}

func TestMCPClientManager_Close(t *testing.T) {
	manager := NewMCPClientManager(nil)
	err := manager.Close()
	require.NoError(t, err, "Close should not return error for nil config")
}

// ============================================================================
// 错误类型测试
// ============================================================================

func TestMCPServerError(t *testing.T) {
	err := &MCPServerError{Code: 500, Message: "internal error"}
	assert.Equal(t, "MCP server error [500]: internal error", err.Error())
}

func TestErrClientClosed(t *testing.T) {
	assert.Equal(t, "MCP client is closed", ErrClientClosed.Error())
}

func TestErrClientNotInitialized(t *testing.T) {
	assert.Equal(t, "MCP client not initialized", ErrClientNotInitialized.Error())
}

func TestErrInvalidResponse(t *testing.T) {
	assert.Equal(t, "invalid response format", ErrInvalidResponse.Error())
}

// ============================================================================
// MCPToolResult 测试
// ============================================================================

func TestMCPToolResult_MCPToolError(t *testing.T) {
	tests := []struct {
		name     string
		result   *MCPToolResult
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "success result",
			result:   &MCPToolResult{IsError: false, Content: []MCPContent{{Type: "text", Text: "ok"}}},
			wantErr:  false,
			errMsg:   "",
		},
		{
			name:     "error result with content",
			result:   &MCPToolResult{IsError: true, Content: []MCPContent{{Type: "text", Text: "tool error"}}},
			wantErr:  true,
			errMsg:   "tool error",
		},
		{
			name:     "error result without content",
			result:   &MCPToolResult{IsError: true, Content: []MCPContent{}},
			wantErr:  false,
			errMsg:   "",
		},
		{
			name:     "nil result",
			result:   nil,
			wantErr:  false,
			errMsg:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result == nil {
				return // nil receiver test
			}
			err := tt.result.MCPToolError()
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// 解析函数测试
// ============================================================================

func TestParseToolInfos(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    []MCPToolInfo
		wantErr bool
	}{
		{
			name: "valid tools",
			input: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"name":        "read_file",
						"description": "Read a file",
						"inputSchema": map[string]interface{}{"type": "object"},
					},
					map[string]interface{}{
						"name":        "write_file",
						"description": "Write a file",
					},
				},
			},
			want: []MCPToolInfo{
				{Name: "read_file", Description: "Read a file", InputSchema: map[string]interface{}{"type": "object"}},
				{Name: "write_file", Description: "Write a file"},
			},
			wantErr: false,
		},
		{
			name:    "missing tools field",
			input:   map[string]interface{}{},
			want:    []MCPToolInfo{},
			wantErr: false,
		},
		{
			name:    "invalid tools format",
			input:   map[string]interface{}{"tools": "not an array"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "partial tool data",
			input: map[string]interface{}{
				"tools": []interface{}{
					"not a map",
					map[string]interface{}{"name": "valid_tool"},
					nil,
				},
			},
			want: []MCPToolInfo{
				{Name: "valid_tool"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseToolInfos(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidResponse))
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseToolResult(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
		want  *MCPToolResult
	}{
		{
			name: "valid content",
			input: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "Hello World"},
					map[string]interface{}{"type": "image", "text": "data:image/png;base64,..."},
				},
			},
			want: &MCPToolResult{
				Content: []MCPContent{
					{Type: "text", Text: "Hello World"},
					{Type: "image", Text: "data:image/png;base64,..."},
				},
			},
		},
		{
			name:  "missing content",
			input: map[string]interface{}{},
			want:  &MCPToolResult{Content: []MCPContent{}},
		},
		{
			name:  "invalid content format",
			input: map[string]interface{}{"content": "not an array"},
			want:  &MCPToolResult{Content: []MCPContent{}},
		},
		{
			name: "partial content items",
			input: map[string]interface{}{
				"content": []interface{}{
					"not a map",
					nil,
					map[string]interface{}{"type": "text", "text": "valid"},
				},
			},
			want: &MCPToolResult{
				Content: []MCPContent{
					{Type: "text", Text: "valid"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseToolResult(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// StdioMCPClient 配置测试
// ============================================================================

func TestNewStdioMCPClient(t *testing.T) {
	cfg := StdioMCPClientConfig{
		Command: "node",
		Args:    []string{"server.js"},
		Env:     map[string]string{"KEY": "value"},
	}

	client := NewStdioMCPClient("test-server", cfg)

	require.NotNil(t, client)
	assert.Equal(t, "test-server", client.serverName)
	assert.Equal(t, 1, client.nextID)
	assert.NotNil(t, client.cmd)
	assert.Equal(t, cfg.Env, client.env)
}

func TestStdioMCPClient_Close_Idempotent(t *testing.T) {
	client := &StdioMCPClient{
		serverName: "test",
		nextID:     1,
	}

	// 第一次 Close 应该成功
	err1 := client.Close()
	require.NoError(t, err1)
	assert.True(t, client.closed.Load())

	// 第二次 Close 应该幂等返回 nil
	err2 := client.Close()
	require.NoError(t, err2)
}

// ============================================================================
// JSON-RPC 解析测试
// ============================================================================

func TestMCPResponse_Error(t *testing.T) {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &MCPError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	require.NotNil(t, resp.Error)
	assert.Equal(t, -32600, resp.Error.Code)
	assert.Equal(t, "Invalid Request", resp.Error.Message)
}

func TestMCPResponse_Result(t *testing.T) {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"tools": []interface{}{},
		},
	}

	require.NotNil(t, resp.Result)
	result := resp.Result.(map[string]interface{})
	tools := result["tools"].([]interface{})
	assert.Empty(t, tools)
}