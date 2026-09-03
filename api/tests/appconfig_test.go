//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// assertResponseCode 辅助函数，处理 JSON 中 code 是 float64 的情况
func assertResponseCode(t *testing.T, resp map[string]interface{}, expectedCode int) {
	code := resp["code"]
	if f, ok := code.(float64); ok {
		assert.Equal(t, float64(expectedCode), f)
	} else {
		assert.Equal(t, expectedCode, code)
	}
}

// TestAppConfigAPI_UpdateAndGetLLMConfig 测试更新后获取 LLM 配置
func TestAppConfigAPI_UpdateAndGetLLMConfig(t *testing.T) {
	// 1. 更新配置（创建配置）
	updateData := map[string]interface{}{
		"base_url":    "https://api.test.com/v1",
		"model_name":  "test-model",
		"api_key":     "test-key-123",
		"temperature": 0.9,
		"max_tokens":  2048,
	}
	updateJSON, _ := json.Marshal(updateData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/llm", bytes.NewReader(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 2. 获取配置验证更新
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/llm", nil)
	testServer.ServeHTTP(getW, getReq)

	var getResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &getResp)
	assertResponseCode(t, getResp, 0)

	data, ok := getResp["data"].(map[string]interface{})
	assert.True(t, ok, "data should be a map")
	assert.NotNil(t, data)

	assert.Equal(t, "https://api.test.com/v1", data["base_url"])
	assert.Equal(t, "test-model", data["model_name"])
	assert.Equal(t, float64(0.9), data["temperature"])
}

// TestAppConfigAPI_UpdateAndGetAgentConfig 测试更新后获取 Agent 配置
func TestAppConfigAPI_UpdateAndGetAgentConfig(t *testing.T) {
	// 1. 更新配置
	updateData := map[string]interface{}{
		"max_iterations":     20,
		"max_retries":        5,
		"max_search_results": 10,
	}
	updateJSON, _ := json.Marshal(updateData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/agent", bytes.NewReader(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 2. 获取配置验证
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/agent", nil)
	testServer.ServeHTTP(getW, getReq)

	var getResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &getResp)
	assertResponseCode(t, getResp, 0)

	data, ok := getResp["data"].(map[string]interface{})
	assert.True(t, ok, "data should be a map")

	assert.Equal(t, float64(20), data["max_iterations"])
	assert.Equal(t, float64(5), data["max_retries"])
}

// TestAppConfigAPI_GetEmptyMCPConfig 测试获取空的 MCP 配置
func TestAppConfigAPI_GetEmptyMCPConfig(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/app-config/mcp-servers", nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assertResponseCode(t, resp, 0)

	// data 可能是 nil 或空对象
	data := resp["data"]
	if data != nil {
		_, ok := data.(map[string]interface{})
		assert.True(t, ok, "data should be a map or nil")
	}
}

// TestAppConfigAPI_UpdateAndGetMCPConfig 测试更新后获取 MCP 配置
func TestAppConfigAPI_UpdateAndGetMCPConfig(t *testing.T) {
	// 1. 更新 MCP 配置
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
	updateJSON, _ := json.Marshal(mcpConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/mcp-servers", bytes.NewReader(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 2. 获取配置验证
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/mcp-servers", nil)
	testServer.ServeHTTP(getW, getReq)

	var getResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &getResp)
	assertResponseCode(t, getResp, 0)

	data, ok := getResp["data"].(map[string]interface{})
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

// TestAppConfigAPI_DeleteMCPServer 测试删除 MCP 服务器
func TestAppConfigAPI_DeleteMCPServer(t *testing.T) {
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
	updateJSON, _ := json.Marshal(mcpConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/mcp-servers", bytes.NewReader(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. 删除服务器（路由参数名是 :server_name）
	deleteW := httptest.NewRecorder()
	deleteReq, _ := http.NewRequest("POST", "/api/app-config/mcp-servers/temp-server-to-delete/delete", nil)
	testServer.ServeHTTP(deleteW, deleteReq)

	// 打印响应体便于调试
	if deleteW.Code != http.StatusOK {
		t.Logf("delete response body: %s", deleteW.Body.String())
	}
	assert.Equal(t, http.StatusOK, deleteW.Code)

	// 3. 验证服务器已被删除
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/mcp-servers", nil)
	testServer.ServeHTTP(getW, getReq)

	var getResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &getResp)
	if data, ok := getResp["data"].(map[string]interface{}); ok {
		if servers, ok := data["servers"].([]interface{}); ok {
			for _, s := range servers {
				if srv, ok := s.(map[string]interface{}); ok && srv["server_name"] == "temp-server-to-delete" {
					t.Fatalf("server should be deleted, but still exists")
				}
			}
		}
	}
}

// TestAppConfigAPI_GetEmptyA2AConfig 测试获取空的 A2A 配置
func TestAppConfigAPI_GetEmptyA2AConfig(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/app-config/a2a-servers", nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assertResponseCode(t, resp, 0)
}

// TestAppConfigAPI_UpdateAndGetA2AConfig 测试更新后获取 A2A 配置
func TestAppConfigAPI_UpdateAndGetA2AConfig(t *testing.T) {
	// 1. 更新 A2A 配置（A2AConfig 的字段是 servers，合并按 id 合并）
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
	updateJSON, _ := json.Marshal(a2aConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/a2a-servers", bytes.NewReader(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 2. 获取配置验证
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/app-config/a2a-servers", nil)
	testServer.ServeHTTP(getW, getReq)

	var getResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &getResp)
	assertResponseCode(t, getResp, 0)

	data, ok := getResp["data"].(map[string]interface{})
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

	// 应该返回错误
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAppConfigAPI_EmptyUpdate 测试空更新
func TestAppConfigAPI_EmptyUpdate(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/app-config/agent", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	// 空更新应该返回成功
	assert.Equal(t, http.StatusOK, w.Code)
}
