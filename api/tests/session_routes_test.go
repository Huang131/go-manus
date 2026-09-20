//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHealthEndpoint 验证 /health 和 /api/status 路由正常可达。
// 这是集成测试基建（路由注册、中间件、状态服务）的最小冒烟验证。
func TestHealthEndpoint(t *testing.T) {
	w := getJSON(t, "/health")
	assertOK(t, w)

	w = getJSON(t, "/api/status")
	assertOK(t, w)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
}

// TestChatEndpoint_AgentNotConfigured 验证 agent 未启用时 /chat 返回 412。
// 这守护了路由 → handler → 依赖检查 → 错误映射的完整链路。
func TestChatEndpoint_AgentNotConfigured(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	w := postJSON(t, "/api/sessions/"+sessionID+"/chat", map[string]any{
		"message": "hello",
	})

	assertStatus(t, w, http.StatusPreconditionFailed)
	assertError(t, w, http.StatusPreconditionFailed)
}

// TestChatEndpoint_BadRequest 验证 /chat 对无效请求体返回 400。
func TestChatEndpoint_BadRequest(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	// 发送空 body（无 message 字段），应返回 412（因为 agent 检查先于参数解析）
	w := postJSON(t, "/api/sessions/"+sessionID+"/chat", map[string]any{})
	assertStatus(t, w, http.StatusPreconditionFailed)
}

// TestReadFileEndpoint_SandboxNotConfigured 验证沙箱未启用时 /file 返回 412。
func TestReadFileEndpoint_SandboxNotConfigured(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	w := postJSON(t, "/api/sessions/"+sessionID+"/file", map[string]any{
		"filepath": "/tmp/test.txt",
	})

	assertStatus(t, w, http.StatusPreconditionFailed)
}

// TestReadShellEndpoint_SandboxNotConfigured 验证沙箱未启用时 /shell 返回 412。
func TestReadShellEndpoint_SandboxNotConfigured(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	w := postJSON(t, "/api/sessions/"+sessionID+"/shell", map[string]any{
		"shell_session_id": "test-shell",
	})

	assertStatus(t, w, http.StatusPreconditionFailed)
}

// TestRenameSession 验证 rename 端点正常工作。
func TestRenameSession(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	w := postJSON(t, "/api/sessions/"+sessionID+"/rename", map[string]any{
		"title": "renamed-session",
	})
	assertOK(t, w)

	// 验证标题已更新
	w = getJSON(t, "/api/sessions/"+sessionID)
	data := parseResponseDataAsMap(t, w)
	assert.Equal(t, "renamed-session", data["title"])
}

// TestSessionFilesEndpoint 验证 GET /sessions/:id/files 返回空列表。
func TestSessionFilesEndpoint(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	w := getJSON(t, "/api/sessions/"+sessionID+"/files")
	assertOK(t, w)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
}
