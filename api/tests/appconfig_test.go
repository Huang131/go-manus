//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppConfigAPI_AgentConfig_Lifecycle 测试 Agent 配置完整生命周期
func TestAppConfigAPI_AgentConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "agent", "default")

	// 1. Get 初始状态
	getW := getJSON(t, "/api/app-config/agent")
	require.Equal(t, http.StatusOK, getW.Code, getW.Body.String())

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	// 2. Update 创建配置
	updateData := map[string]any{
		"max_iterations":     20,
		"max_retries":        5,
		"max_search_results": 10,
	}
	updW := postJSON(t, "/api/app-config/agent", updateData)
	require.Equal(t, http.StatusOK, updW.Code, updW.Body.String())

	// 3. Get 验证更新
	getW2 := getJSON(t, "/api/app-config/agent")
	require.Equal(t, http.StatusOK, getW2.Code, getW2.Body.String())

	resp2 := parseResponse(t, getW2)
	assert.Equal(t, 0, resp2.Code)

	data := parseResponseDataAsMap(t, getW2)
	assert.Equal(t, float64(20), data["max_iterations"])
	assert.Equal(t, float64(5), data["max_retries"])
	assert.Equal(t, float64(10), data["max_search_results"])
}

// TestAppConfigAPI_MCPConfig_Lifecycle 测试 MCP 配置完整生命周期
func TestAppConfigAPI_MCPConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "mcp", "default")

	// 1. Get 初始状态
	getW := getJSON(t, "/api/app-config/mcp-servers")
	require.Equal(t, http.StatusOK, getW.Code, getW.Body.String())

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	// 2. Update 创建配置
	mcpConfig := map[string]any{
		"servers": []map[string]any{
			{
				"server_name": "test-filesystem",
				"enabled":     true,
				"command":     "test-mcp-server",
				"args":        []string{"--stdio"},
			},
		},
	}
	updW := postJSON(t, "/api/app-config/mcp-servers", mcpConfig)
	require.Equal(t, http.StatusOK, updW.Code, updW.Body.String())

	// 3. Get 验证更新
	getW2 := getJSON(t, "/api/app-config/mcp-servers")
	require.Equal(t, http.StatusOK, getW2.Code, getW2.Body.String())

	resp2 := parseResponse(t, getW2)
	assert.Equal(t, 0, resp2.Code)

	data := parseResponseDataAsMap(t, getW2)

	servers, ok := data["servers"].([]any)
	require.True(t, ok, "servers should be an array: %#v", data["servers"])
	require.NotEmpty(t, servers)

	// 找到我们刚添加的 server
	var found map[string]any
	for _, s := range servers {
		if srv, ok := s.(map[string]any); ok && srv["server_name"] == "test-filesystem" {
			found = srv
			break
		}
	}
	require.NotNil(t, found, "should find test-filesystem server")
	assert.Equal(t, "test-mcp-server", found["command"])
}

// TestAppConfigAPI_MCPConfig_Delete 测试删除 MCP 服务器
func TestAppConfigAPI_MCPConfig_Delete(t *testing.T) {
	defer CleanupAppConfig(t, "mcp", "default")

	// 1. 添加 MCP 服务器
	mcpConfig := map[string]any{
		"servers": []map[string]any{
			{
				"server_name": "temp-server-to-delete",
				"enabled":     true,
				"command":     "temporary-mcp-server",
			},
		},
	}
	addW := postJSON(t, "/api/app-config/mcp-servers", mcpConfig)
	require.Equal(t, http.StatusOK, addW.Code, addW.Body.String())

	// 2. 删除服务器
	deleteW := postJSON(t, "/api/app-config/mcp-servers/temp-server-to-delete/delete", nil)
	if deleteW.Code != http.StatusOK {
		t.Logf("delete response body: %s", deleteW.Body.String())
	}
	require.Equal(t, http.StatusOK, deleteW.Code, deleteW.Body.String())

	// 3. 验证服务器已被删除
	getW := getJSON(t, "/api/app-config/mcp-servers")

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	if data, ok := resp.Data.(map[string]any); ok {
		if servers, ok := data["servers"].([]any); ok {
			for _, s := range servers {
				if srv, ok := s.(map[string]any); ok && srv["server_name"] == "temp-server-to-delete" {
					t.Fatalf("server should be deleted, but still exists")
				}
			}
		}
	}
}

// TestAppConfigAPI_A2AConfig_Lifecycle 测试 A2A 配置完整生命周期
func TestAppConfigAPI_A2AConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "a2a", "default")

	// 1. Get 初始状态
	getW := getJSON(t, "/api/app-config/a2a-servers")
	require.Equal(t, http.StatusOK, getW.Code, getW.Body.String())

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	// 2. Update 创建配置
	a2aConfig := map[string]any{
		"servers": []map[string]any{
			{
				"id":      "test-a2a-1",
				"url":     "http://test-a2a.example",
				"enabled": true,
			},
		},
	}
	updW := postJSON(t, "/api/app-config/a2a-servers", a2aConfig)
	require.Equal(t, http.StatusOK, updW.Code, updW.Body.String())

	// 3. Get 验证更新
	getW2 := getJSON(t, "/api/app-config/a2a-servers")
	require.Equal(t, http.StatusOK, getW2.Code, getW2.Body.String())

	resp2 := parseResponse(t, getW2)
	assert.Equal(t, 0, resp2.Code)

	data := parseResponseDataAsMap(t, getW2)

	servers, ok := data["servers"].([]any)
	require.True(t, ok, "servers should be an array: %#v", data["servers"])
	require.NotEmpty(t, servers)

	// 找到我们刚添加的 a2a server
	var found map[string]any
	for _, s := range servers {
		if srv, ok := s.(map[string]any); ok && srv["id"] == "test-a2a-1" {
			found = srv
			break
		}
	}
	require.NotNil(t, found, "should find test-a2a-1 server")
	assert.Equal(t, "http://test-a2a.example", found["url"])
}
