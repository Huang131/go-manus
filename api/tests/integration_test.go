//go:build integration

// Package integration 包含真实环境的集成测试。
//
// 测试特点：
//   - 使用真实的 Postgres、Redis、MinIO（需先启动 ../../scripts/test-env-up.sh）
//   - 每个测试用例独立，使用唯一标识避免数据污染
//   - 测试完成后自动清理数据
//
// 运行方式：
//
//	# 先确保开发 Docker (docker-compose.yml) 已启动
//	# 初始化测试数据（在共享 Docker 上创建 manus_test DB 等）
//	cd ../../scripts && ./test-env-up.sh
//
//	# 运行测试
//	go test -tags=integration -v ./tests/...
//
//	# 清理测试数据（可选）
//	cd ../../scripts && ./test-env-down.sh
package integration

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"mime/multipart"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/internal/bootstrap"
	"github.com/mooc-manus/go-manus/api/internal/infrastructure"
	"github.com/mooc-manus/go-manus/api/pkg/response"
)

var (
	testServer *gin.Engine
	testCfg    *config.Config
	testApp    *bootstrap.App
	testDB     *infrastructure.Postgres
)

// TestMain 初始化测试环境
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// 加载测试配置
	cfg, err := config.Load("../config.test.yaml")
	if err != nil {
		log.Fatalf("加载测试配置失败: %v", err)
	}

	app, err := bootstrap.Build(cfg, bootstrap.Options{
		EnablePostgres:    true,
		EnableRedis:       true,
		EnableStorage:     true,
		EnableLLM:         false,
		EnableSandbox:     false,
		EnableBrowser:     false,
		EnableSearch:      false,
		EnableAgent:       false,
		EnableRoutes:      true,
		EnableHealthCheck: true,
	})
	if err != nil {
		log.Fatalf("构建测试应用失败: %v", err)
	}

	// 保存全局变量
	testCfg = cfg
	testApp = app
	testServer = app.Engine
	testDB = app.Postgres

	// 运行测试
	code := m.Run()

	// 清理
	if testApp != nil {
		testApp.Close()
	}

	os.Exit(code)
}

// NewTestContext 返回带超时的测试上下文
func NewTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// CleanupSession 清理测试会话
func CleanupSession(t *testing.T, sessionID string) {
	t.Helper()
	ctx, cancel := NewTestContext()
	defer cancel()

	_, err := testApp.Postgres.Pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		t.Logf("清理会话 %s 失败: %v", sessionID, err)
	}
}

// CleanupFile 清理测试文件记录
func CleanupFile(t *testing.T, fileID string) {
	t.Helper()
	ctx, cancel := NewTestContext()
	defer cancel()

	_, err := testApp.Postgres.Pool.Exec(ctx, "DELETE FROM files WHERE id = $1", fileID)
	if err != nil {
		t.Logf("清理文件记录 %s 失败: %v", fileID, err)
	}
}

// CleanupAppConfig 清理测试配置
func CleanupAppConfig(t *testing.T, configType, configKey string) {
	t.Helper()
	ctx, cancel := NewTestContext()
	defer cancel()

	_, err := testApp.Postgres.Pool.Exec(ctx, "DELETE FROM app_configs WHERE config_type = $1 AND config_key = $2", configType, configKey)
	if err != nil {
		t.Logf("清理配置 %s/%s 失败: %v", configType, configKey, err)
	}
}

// ==================== 通用 HTTP 请求辅助函数 ====================

// doRequest 发送 HTTP 请求并返回响应
func doRequest(t *testing.T, method, path string, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	testServer.ServeHTTP(w, req)
	return w
}

// postJSON 发送 POST JSON 请求
func postJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	bodyJSON, err := sonic.Marshal(body)
	if err != nil {
		t.Fatalf("序列化请求体失败: %v", err)
	}
	return doRequest(t, "POST", path, bodyJSON, "application/json")
}

// getJSON 发送 GET 请求
func getJSON(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, "GET", path, nil, "")
}

// putJSON 发送 PUT JSON 请求
func putJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	bodyJSON, err := sonic.Marshal(body)
	if err != nil {
		t.Fatalf("序列化请求体失败: %v", err)
	}
	return doRequest(t, "PUT", path, bodyJSON, "application/json")
}

// deleteRequest 发送 DELETE 请求
func deleteRequest(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, "DELETE", path, nil, "")
}

// postFormData 发送 POST multipart/form-data 请求
func postFormData(t *testing.T, path string, fields map[string]string, fileName string, fileContent []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := makeMultipartFile(fields, fileName, fileContent)
	return doRequest(t, "POST", path, body.Bytes(), contentType)
}

// parseResponse 解析 HTTP 响应为 Response 结构
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) response.Response {
	t.Helper()
	var resp response.Response
	if err := sonic.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return resp
}

// parseTotalResponse 解析 HTTP 响应为 TotalResponse 结构
func parseTotalResponse(t *testing.T, w *httptest.ResponseRecorder) response.TotalResponse {
	t.Helper()
	var resp response.TotalResponse
	if err := sonic.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return resp
}

// parseResponseDataAsMap 解析响应 data 字段为 map[string]any
func parseResponseDataAsMap(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	resp := parseResponse(t, w)
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("data 应为 map[string]any，实际为 %T", resp.Data)
	}
	return data
}

// parseResponseDataAsArray 解析响应 data 字段为 []any
func parseResponseDataAsArray(t *testing.T, w *httptest.ResponseRecorder) []any {
	t.Helper()
	resp := parseResponse(t, w)
	data, ok := resp.Data.([]any)
	if !ok {
		t.Fatalf("data 应为 []any，实际为 %T", resp.Data)
	}
	return data
}

// makeMultipartFile 构造 multipart/form-data 请求体
func makeMultipartFile(fields map[string]string, fileName string, fileContent []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}

	if fileName != "" && len(fileContent) > 0 {
		part, err := writer.CreateFormFile("file", fileName)
		if err == nil {
			_, _ = part.Write(fileContent)
		}
	}

	_ = writer.Close()
	return body, writer.FormDataContentType()
}
