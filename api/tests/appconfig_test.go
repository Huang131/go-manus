//go:build integration

package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/mooc-manus/go-manus/api/pkg/response"
	"github.com/stretchr/testify/assert"
)

// TestAppConfigAPI_LLMConfig_Lifecycle 测试 LLM 配置完整生命周期
func TestAppConfigAPI_LLMConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "llm", "default")

	// 1. Get 初始状态
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/llm", nil)
	testServer.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var getResp response.Response
	sonic.Unmarshal(getW.Body.Bytes(), &getResp)
	assert.Equal(t, 0, getResp.Code)

	// 2. Update 创建配置
	updateData := map[string]interface{}{
		"base_url":    "https://api.test.com/v1",
		"model_name":  "test-model",
		"api_key":     "test-key-123",
		"temperature": 0.9,
		"max_tokens":  2048,
	}
	updateJSON, _ := sonic.Marshal(updateData)
	updW := httptest.NewRecorder()
	updReq, _ := http.NewRequest("POST", "/api/app-config/llm", bytes.NewReader(updateJSON))
	updReq.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(updW, updReq)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := httptest.NewRecorder()
	getReq2, _ := http.NewRequest("GET", "/api/app-config/llm", nil)
	testServer.ServeHTTP(getW2, getReq2)
	assert.Equal(t, http.StatusOK, getW2.Code)

	var getResp2 response.Response
	sonic.Unmarshal(getW2.Body.Bytes(), &getResp2)
	assert.Equal(t, 0, getResp2.Code)

	data, ok := getResp2.Data.(map[string]interface{})
	assert.True(t, ok, "data should be a map")
	assert.Equal(t, "https://api.test.com/v1", data["base_url"])
	assert.Equal(t, "test-model", data["model_name"])
	assert.Equal(t, float64(0.9), data["temperature"])
}

// TestAppConfigAPI_AgentConfig_Lifecycle 测试 Agent 配置完整生命周期
func TestAppConfigAPI_AgentConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "agent", "default")

	// 1. Get 初始状态
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/agent", nil)
	testServer.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var getResp response.Response
	sonic.Unmarshal(getW.Body.Bytes(), &getResp)
	assert.Equal(t, 0, getResp.Code)

	// 2. Update 创建配置
	updateData := map[string]interface{}{
		"max_iterations":     20,
		"max_retries":        5,
		"max_search_results": 10,
	}
	updateJSON, _ := sonic.Marshal(updateData)
	updW := httptest.NewRecorder()
	updReq, _ := http.NewRequest("POST", "/api/app-config/agent", bytes.NewReader(updateJSON))
	updReq.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(updW, updReq)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := httptest.NewRecorder()
	getReq2, _ := http.NewRequest("GET", "/api/app-config/agent", nil)
	testServer.ServeHTTP(getW2, getReq2)
	assert.Equal(t, http.StatusOK, getW2.Code)

	var getResp2 response.Response
	sonic.Unmarshal(getW2.Body.Bytes(), &getResp2)
	assert.Equal(t, 0, getResp2.Code)

	data, ok := getResp2.Data.(map[string]interface{})
	assert.True(t, ok, "data should be a map")
	assert.Equal(t, float64(20), data["max_iterations"])
	assert.Equal(t, float64(5), data["max_retries"])
	assert.Equal(t, float64(10), data["max_search_results"])
}

// TestAppConfigAPI_MCPConfig_Lifecycle 测试 MCP 配置完整生命周期
func TestAppConfigAPI_MCPConfig_Lifecycle(t *testing.T) {
	defer CleanupAppConfig(t, "mcp", "default")

	// 1. Get 初始状态
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/mcp-servers", nil)
	testServer.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var getResp response.Response
	sonic.Unmarshal(getW.Body.Bytes(), &getResp)
	assert.Equal(t, 0, getResp.Code)

	// 2. Update 创建配置
	mcpConfig := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"server_name": "test-filesystem",
				"enabled":     true,
				"transport":   "stdio",
				"tools":       []string{"read_file", "write_file"},
			},
		},
	}
	updateJSON, _ := sonic.Marshal(mcpConfig)
	updW := httptest.NewRecorder()
	updReq, _ := http.NewRequest("POST", "/api/app-config/mcp-servers", bytes.NewReader(updateJSON))
	updReq.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(updW, updReq)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := httptest.NewRecorder()
	getReq2, _ := http.NewRequest("GET", "/api/app-config/mcp-servers", nil)
	testServer.ServeHTTP(getW2, getReq2)
	assert.Equal(t, http.StatusOK, getW2.Code)

	var getResp2 response.Response
	sonic.Unmarshal(getW2.Body.Bytes(), &getResp2)
	assert.Equal(t, 0, getResp2.Code)

	data, ok := getResp2.Data.(map[string]interface{})
	assert.True(t, ok, "data should be a map")

	servers, ok := data["servers"].([]interface{})
	assert.True(t, ok, "servers should be an array")
	assert.GreaterOrEqual(t, len(servers), 1, "should have at least 1 server")

	// 找到我们刚添加的 server
	var found map[string]interface{}
	for _, s := range servers {
		if srv, ok := s.(map[string]interface{}); ok && srv["server_name"] == "test-filesystem" {
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
	mcpConfig := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"server_name": "temp-server-to-delete",
				"enabled":     true,
				"transport":   "stdio",
				"tools":       []string{"test"},
			},
		},
	}
	updateJSON, _ := sonic.Marshal(mcpConfig)
	addW := httptest.NewRecorder()
	addReq, _ := http.NewRequest("POST", "/api/app-config/mcp-servers", bytes.NewReader(updateJSON))
	addReq.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(addW, addReq)
	assert.Equal(t, http.StatusOK, addW.Code)

	// 2. 删除服务器
	deleteW := httptest.NewRecorder()
	deleteReq, _ := http.NewRequest("POST", "/api/app-config/mcp-servers/temp-server-to-delete/delete", nil)
	testServer.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Logf("delete response body: %s", deleteW.Body.String())
	}
	assert.Equal(t, http.StatusOK, deleteW.Code)

	// 3. 验证服务器已被删除
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/mcp-servers", nil)
	testServer.ServeHTTP(getW, getReq)

	var getResp response.Response
	sonic.Unmarshal(getW.Body.Bytes(), &getResp)
	assert.Equal(t, 0, getResp.Code)

	if data, ok := getResp.Data.(map[string]interface{}); ok {
		if servers, ok := data["servers"].([]interface{}); ok {
			for _, s := range servers {
				if srv, ok := s.(map[string]interface{}); ok && srv["server_name"] == "temp-server-to-delete" {
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
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/a2a-servers", nil)
	testServer.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var getResp response.Response
	sonic.Unmarshal(getW.Body.Bytes(), &getResp)
	assert.Equal(t, 0, getResp.Code)

	// 2. Update 创建配置
	a2aConfig := map[string]interface{}{
		"servers": []map[string]interface{}{
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
	updateJSON, _ := sonic.Marshal(a2aConfig)
	updW := httptest.NewRecorder()
	updReq, _ := http.NewRequest("POST", "/api/app-config/a2a-servers", bytes.NewReader(updateJSON))
	updReq.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(updW, updReq)
	assert.Equal(t, http.StatusOK, updW.Code)

	// 3. Get 验证更新
	getW2 := httptest.NewRecorder()
	getReq2, _ := http.NewRequest("GET", "/api/app-config/a2a-servers", nil)
	testServer.ServeHTTP(getW2, getReq2)
	assert.Equal(t, http.StatusOK, getW2.Code)

	var getResp2 response.Response
	sonic.Unmarshal(getW2.Body.Bytes(), &getResp2)
	assert.Equal(t, 0, getResp2.Code)

	data, ok := getResp2.Data.(map[string]interface{})
	assert.True(t, ok, "data should be a map")

	servers, ok := data["servers"].([]interface{})
	assert.True(t, ok, "servers should be an array")
	assert.GreaterOrEqual(t, len(servers), 1, "should have at least 1 server")

	// 找到我们刚添加的 a2a server
	var found map[string]interface{}
	for _, s := range servers {
		if srv, ok := s.(map[string]interface{}); ok && srv["id"] == "test-a2a-1" {
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
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/llm", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAppConfigAPI_EmptyUpdate 测试空更新
func TestAppConfigAPI_EmptyUpdate(t *testing.T) {
	defer CleanupAppConfig(t, "agent", "default")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/agent", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
