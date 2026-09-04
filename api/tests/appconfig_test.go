//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppConfigAPI_LLMConfig_Lifecycle 测试 LLM 配置完整生命周期
func TestAppConfigAPI_LLMConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "llm", "default")

	// 1. Get 初始状态
	getW := getJSON(t, "/api/app-config/llm")
	assert.Equal(t, http.StatusOK, getW.Code)

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	// 2. Update 创建配置
	updateData := map[string]any{
		"base_url":    "https://api.test.com/v1",
		"model_name":  "test-model",
		"api_key":     "test-key-123",
		"temperature": 0.9,
		"max_tokens":  2048,
	}
	updW := postJSON(t, "/api/app-config/llm", updateData)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := getJSON(t, "/api/app-config/llm")
	assert.Equal(t, http.StatusOK, getW2.Code)

	resp2 := parseResponse(t, getW2)
	assert.Equal(t, 0, resp2.Code)

	data := parseResponseDataAsMap(t, getW2)
	assert.Equal(t, "https://api.test.com/v1", data["base_url"])
	assert.Equal(t, "test-model", data["model_name"])
	assert.Equal(t, float64(0.9), data["temperature"])
}

// TestAppConfigAPI_AgentConfig_Lifecycle 测试 Agent 配置完整生命周期
func TestAppConfigAPI_AgentConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "agent", "default")

	// 1. Get 初始状态
	getW := getJSON(t, "/api/app-config/agent")
	assert.Equal(t, http.StatusOK, getW.Code)

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	// 2. Update 创建配置
	updateData := map[string]any{
		"max_iterations":     20,
		"max_retries":        5,
		"max_search_results": 10,
	}
	updW := postJSON(t, "/api/app-config/agent", updateData)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := getJSON(t, "/api/app-config/agent")
	assert.Equal(t, http.StatusOK, getW2.Code)

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
	assert.Equal(t, http.StatusOK, getW.Code)

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	// 2. Update 创建配置
	mcpConfig := map[string]any{
		"servers": []map[string]any{
			{
				"server_name": "test-filesystem",
				"enabled":     true,
				"transport":   "stdio",
				"tools":       []string{"read_file", "write_file"},
			},
		},
	}
	updW := postJSON(t, "/api/app-config/mcp-servers", mcpConfig)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := getJSON(t, "/api/app-config/mcp-servers")
	assert.Equal(t, http.StatusOK, getW2.Code)

	resp2 := parseResponse(t, getW2)
	assert.Equal(t, 0, resp2.Code)

	data := parseResponseDataAsMap(t, getW2)

	servers := data["servers"].([]any)
	assert.GreaterOrEqual(t, len(servers), 1, "should have at least 1 server")

	// 找到我们刚添加的 server
	var found map[string]any
	for _, s := range servers {
		if srv, ok := s.(map[string]any); ok && srv["server_name"] == "test-filesystem" {
			found = srv
			break
		}
	}
	assert.NotNil(t, found, "should find test-filesystem server")
	assert.Equal(t, "stdio", found["transport"])
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
				"transport":   "stdio",
				"tools":       []string{"test"},
			},
		},
	}
	addW := postJSON(t, "/api/app-config/mcp-servers", mcpConfig)
	assert.Equal(t, http.StatusOK, addW.Code)

	// 2. 删除服务器
	deleteW := postJSON(t, "/api/app-config/mcp-servers/temp-server-to-delete/delete", nil)
	if deleteW.Code != http.StatusOK {
		t.Logf("delete response body: %s", deleteW.Body.String())
	}
	assert.Equal(t, http.StatusOK, deleteW.Code)

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
	assert.Equal(t, http.StatusOK, getW.Code)

	resp := parseResponse(t, getW)
	assert.Equal(t, 0, resp.Code)

	// 2. Update 创建配置
	a2aConfig := map[string]any{
		"servers": []map[string]any{
			{
				"id":                 "test-a2a-1",
				"name":               "test-agent",
				"description":        "A test A2A agent",
				"input_modes":        []string{"text"},
				"output_modes":       []string{"text"},
				"streaming":          true,
				"push_notifications": false,
				"enabled":            true,
			},
		},
	}
	updW := postJSON(t, "/api/app-config/a2a-servers", a2aConfig)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := getJSON(t, "/api/app-config/a2a-servers")
	assert.Equal(t, http.StatusOK, getW2.Code)

	resp2 := parseResponse(t, getW2)
	assert.Equal(t, 0, resp2.Code)

	data := parseResponseDataAsMap(t, getW2)

	servers := data["servers"].([]any)
	assert.GreaterOrEqual(t, len(servers), 1, "should have at least 1 server")

	// 找到我们刚添加的 a2a server
	var found map[string]any
	for _, s := range servers {
		if srv, ok := s.(map[string]any); ok && srv["id"] == "test-a2a-1" {
			found = srv
			break
		}
	}
	assert.NotNil(t, found, "should find test-a2a-1 server")
	assert.Equal(t, "test-agent", found["name"])
	assert.Equal(t, "A test A2A agent", found["description"])
}

// TestAppConfigAPI_InvalidJSON 测试无效 JSON
func TestAppConfigAPI_InvalidJSON(t *testing.T) {
	w := doRequest(t, "POST", "/api/app-config/llm", []byte("invalid json"), "application/json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAppConfigAPI_EmptyUpdate 测试空更新
func TestAppConfigAPI_EmptyUpdate(t *testing.T) {
	defer CleanupAppConfig(t, "agent", "default")

	w := doRequest(t, "POST", "/api/app-config/agent", []byte("{}"), "application/json")
	assert.Equal(t, http.StatusOK, w.Code)
}
