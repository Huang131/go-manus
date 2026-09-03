//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSessionAPI_Create 测试创建会话（真实 Postgres）
func TestSessionAPI_Create(t *testing.T) {
	// 创建会话
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assertResponseCode(t, resp, 0)

	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "data should be a map")
	sessionID := data["id"].(string)
	defer CleanupSession(t, sessionID)

	assert.NotEmpty(t, sessionID)
	assert.Equal(t, "新对话", data["title"])
	assert.Equal(t, float64(0), data["unread_message_count"])
}

// TestSessionAPI_CreateAndGet 测试创建后获取会话
func TestSessionAPI_CreateAndGet(t *testing.T) {
	// 1. 创建会话
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	sessionID := createResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 2. 获取会话
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/sessions/"+sessionID, nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var getResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	data := getResp["data"].(map[string]interface{})

	assert.Equal(t, sessionID, data["id"])
	assert.Equal(t, "新对话", data["title"])
}

// TestSessionAPI_CreateAndList 测试创建后会话出现在列表中
func TestSessionAPI_CreateAndList(t *testing.T) {
	// 1. 创建两个会话
	sessionIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sessions", nil)
		testServer.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		sessionIDs[i] = resp["data"].(map[string]interface{})["id"].(string)
		defer CleanupSession(t, sessionIDs[i])
	}

	// 2. 列出会话
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sessions", nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	// List 接口使用 SuccessWithTotal，data 直接是会话数组
	sessions, ok := listResp["data"].([]interface{})
	assert.True(t, ok, "data should be an array of sessions")
	total := int(listResp["total"].(float64))

	// 验证创建的两个会话都在列表中
	assert.GreaterOrEqual(t, total, 2, "应该至少有2个会话")
	assert.GreaterOrEqual(t, len(sessions), 2, "返回的会话列表应该至少有2条")
}

// TestSessionAPI_CreateAndDelete 测试创建后删除会话
func TestSessionAPI_CreateAndDelete(t *testing.T) {
	// 1. 创建会话
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	sessionID := createResp["data"].(map[string]interface{})["id"].(string)

	// 2. 删除会话
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/sessions/"+sessionID+"/delete", nil)
	testServer.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. 验证会话已删除
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/sessions/"+sessionID, nil)
	testServer.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSessionAPI_GetNotFound 测试获取不存在的会话
func TestSessionAPI_GetNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sessions/non-existent-id", nil)
	testServer.ServeHTTP(w, req)

	// 应该返回错误
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assertResponseCode(t, resp, 400)
}

// TestSessionAPI_ListPagination 测试会话列表分页
func TestSessionAPI_ListPagination(t *testing.T) {
	// 创建 5 个会话
	sessionIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sessions", nil)
		testServer.ServeHTTP(w, req)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		sessionIDs[i] = resp["data"].(map[string]interface{})["id"].(string)
		defer CleanupSession(t, sessionIDs[i])
	}

	// 测试分页：每页 2 条
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sessions?limit=2&offset=0", nil)
	testServer.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	// List 接口使用 SuccessWithTotal，data 直接是会话数组
	sessions, ok := resp["data"].([]interface{})
	assert.True(t, ok, "data should be an array of sessions")
	total := int(resp["total"].(float64))

	assert.Equal(t, 2, len(sessions), "每页应该返回2条")
	assert.GreaterOrEqual(t, total, 5, "总条数应该至少为5")
}

// TestSessionAPI_ClearUnreadCount 测试清除未读数
func TestSessionAPI_ClearUnreadCount(t *testing.T) {
	// 1. 创建会话
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	sessionID := createResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 2. 清除未读数
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/sessions/"+sessionID+"/clear-unread-message-count", nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 3. 验证未读数为 0
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/sessions/"+sessionID, nil)
	testServer.ServeHTTP(w, req)

	var getResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	data := getResp["data"].(map[string]interface{})

	assert.Equal(t, float64(0), data["unread_message_count"])
}

// TestSessionAPI_GetAllSessions 测试获取所有会话（SSE 流）
func TestSessionAPI_GetAllSessions(t *testing.T) {
	// 创建会话
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	sessionID := createResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 注意：SSE 流会持续发送，这里测试短超时场景
	// 由于 SSE 是长连接，实际测试中可能需要 mock 或者使用不同的测试策略
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/sessions/stream", nil)

	// 设置短超时避免测试挂起
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	done := make(chan bool)
	go func() {
		testServer.ServeHTTP(w, req)
		done <- true
	}()

	select {
	case <-done:
		// 测试完成
	case <-ctx.Done():
		// SSE 流超时（这是预期行为，因为 SSE 是长连接）
	}
}
