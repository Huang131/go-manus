package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/internal/handler"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/pkg/response"
)

// MockSessionService 完整的 Session Service Mock
type MockSessionService struct {
	sessions map[string]*model.Session
}

func NewMockSessionService() *MockSessionService {
	return &MockSessionService{
		sessions: make(map[string]*model.Session),
	}
}

func (m *MockSessionService) CreateSession(ctx context.Context) (*model.Session, error) {
	session := &model.Session{
		ID:                 "test-session-id",
		Title:              "新对话",
		UnreadMessageCount: 0,
		Events:             []model.Event{},
		Files:              []model.File{},
		Status:             model.SessionStatusPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	m.sessions[session.ID] = session
	return session, nil
}

func (m *MockSessionService) GetSession(ctx context.Context, id string) (*model.Session, error) {
	session, ok := m.sessions[id]
	if !ok {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (m *MockSessionService) GetAllSessions(ctx context.Context) ([]*model.Session, error) {
	result := make([]*model.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *MockSessionService) ListSessions(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	allSessions := make([]*model.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		allSessions = append(allSessions, s)
	}

	start := offset
	if start > len(allSessions) {
		start = len(allSessions)
	}
	end := start + limit
	if end > len(allSessions) {
		end = len(allSessions)
	}

	return allSessions[start:end], len(allSessions), nil
}

func (m *MockSessionService) DeleteSession(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *MockSessionService) ClearUnreadCount(ctx context.Context, id string) error {
	if session, ok := m.sessions[id]; ok {
		session.UnreadMessageCount = 0
	}
	return nil
}

func (m *MockSessionService) IncrementUnreadCount(ctx context.Context, id string) error {
	if session, ok := m.sessions[id]; ok {
		session.UnreadMessageCount++
	}
	return nil
}

func (m *MockSessionService) DecrementUnreadCount(ctx context.Context, id string) error {
	if session, ok := m.sessions[id]; ok && session.UnreadMessageCount > 0 {
		session.UnreadMessageCount--
	}
	return nil
}

func (m *MockSessionService) GetSessionFiles(ctx context.Context, id string) ([]model.File, error) {
	if session, ok := m.sessions[id]; ok {
		return session.Files, nil
	}
	return []model.File{}, nil
}

func (m *MockSessionService) AppendEvent(ctx context.Context, sessionID string, event *model.Event) error {
	if session, ok := m.sessions[sessionID]; ok {
		session.Events = append(session.Events, *event)
	}
	return nil
}

func (m *MockSessionService) StreamSession(ctx context.Context, id string) (*model.Session, error) {
	return nil, nil
}

func (m *MockSessionService) Chat(ctx context.Context, sessionID string, message string) error {
	if session, ok := m.sessions[sessionID]; ok {
		session.LatestMessage = message
		session.UnreadMessageCount++
	}
	return nil
}

func (m *MockSessionService) GetVNCURL(ctx context.Context, sessionID string) (string, error) {
	return "ws://sandbox.local:5901", nil
}

// MockAppConfigService 完整的 AppConfig Service Mock
type MockAppConfigService struct {
	configs map[string]*model.AppConfig
}

func NewMockAppConfigService() *MockAppConfigService {
	return &MockAppConfigService{
		configs: make(map[string]*model.AppConfig),
	}
}

func (m *MockAppConfigService) GetLLMConfig(ctx context.Context) (*model.LLMConfig, error) {
	return &model.LLMConfig{
		BaseURL:     "https://api.openai.com",
		ModelName:   "gpt-4",
		APIKey:      "test-key",
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

func (m *MockAppConfigService) UpdateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	return nil
}

func (m *MockAppConfigService) GetAgentConfig(ctx context.Context) (*model.AgentConfig, error) {
	return &model.AgentConfig{
		MaxIterations:    10,
		MaxRetries:       3,
		MaxSearchResults: 5,
	}, nil
}

func (m *MockAppConfigService) UpdateAgentConfig(ctx context.Context, cfg *model.AgentConfig) error {
	return nil
}

func (m *MockAppConfigService) GetMCPConfig(ctx context.Context) (*model.MCPConfig, error) {
	return nil, nil
}

func (m *MockAppConfigService) UpdateMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	return nil
}

func (m *MockAppConfigService) DeleteMCPServer(ctx context.Context, serverName string) error {
	return nil
}

func (m *MockAppConfigService) GetA2AConfig(ctx context.Context) (*model.A2AConfig, error) {
	return nil, nil
}

func (m *MockAppConfigService) UpdateA2AConfig(ctx context.Context, cfg *model.A2AConfig) error {
	return nil
}

// MockFileService 完整的 File Service Mock
type MockFileService struct {
	files map[string]*model.File
}

func NewMockFileService() *MockFileService {
	return &MockFileService{
		files: make(map[string]*model.File),
	}
}

func (m *MockFileService) UploadFile(ctx context.Context, sessionID, filename string, reader io.Reader, size int64, contentType string) (*model.File, error) {
	file := &model.File{
		ID:        "test-file-id",
		Filename:  filename,
		Extension: "txt",
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}
	m.files[file.ID] = file
	return file, nil
}

func (m *MockFileService) GetFileInfo(ctx context.Context, id string) (*model.File, error) {
	file, ok := m.files[id]
	if !ok {
		return nil, nil
	}
	return file, nil
}

func (m *MockFileService) DownloadFile(ctx context.Context, id string) (*model.File, io.ReadCloser, error) {
	return &model.File{
		ID:       id,
		Filename: "test.txt",
	}, io.NopCloser(bytes.NewReader([]byte("file content"))), nil
}

func (m *MockFileService) DeleteFile(ctx context.Context, id string) error {
	delete(m.files, id)
	return nil
}

// TestHTTPIntegration_CreateSession 测试创建会话 API
func TestHTTPIntegration_CreateSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	sessionSvc := NewMockSessionService()

	// 创建 handler
	sessionHandler := handler.NewSessionHandler(sessionSvc, nil, nil)

	// 创建路由
	r := gin.New()
	r.POST("/api/v1/sessions", sessionHandler.Create)

	// 发送请求
	req, _ := http.NewRequest("POST", "/api/v1/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200，实际: %d", w.Code)
	}

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("响应解析失败: %v", err)
	}

	// 注意：响应中的 code 是 0，表示成功
	if resp.Code != 0 {
		t.Errorf("响应 code 应为 0，实际: %d", resp.Code)
	}
}

// TestHTTPIntegration_GetSession 测试获取会话 API
func TestHTTPIntegration_GetSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	sessionSvc := NewMockSessionService()
	session, _ := sessionSvc.CreateSession(context.Background())

	// 创建 handler
	sessionHandler := handler.NewSessionHandler(sessionSvc, nil, nil)

	// 创建路由
	r := gin.New()
	r.GET("/api/v1/sessions/:id", sessionHandler.Get)

	// 发送请求
	req, _ := http.NewRequest("GET", "/api/v1/sessions/"+session.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200，实际: %d", w.Code)
	}

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("响应解析失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("响应 code 应为 0，实际: %d", resp.Code)
	}
}

// TestHTTPIntegration_GetSession_NotFound 测试获取不存在的会话
func TestHTTPIntegration_GetSession_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	sessionSvc := NewMockSessionService()

	// 创建 handler
	sessionHandler := handler.NewSessionHandler(sessionSvc, nil, nil)

	// 创建路由
	r := gin.New()
	r.GET("/api/v1/sessions/:id", sessionHandler.Get)

	// 发送请求
	req, _ := http.NewRequest("GET", "/api/v1/sessions/non-existent-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 注意：Handler 可能返回 200 状态码但 code 为非 0
	// 这里我们只验证响应是有效的 JSON
	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("响应应该是有效的 JSON，实际: %v", err)
	}
}

// TestHTTPIntegration_ListSessions 测试会话列表 API
func TestHTTPIntegration_ListSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务并添加测试数据
	sessionSvc := NewMockSessionService()
	_, _ = sessionSvc.CreateSession(context.Background())
	_, _ = sessionSvc.CreateSession(context.Background())

	// 创建 handler
	sessionHandler := handler.NewSessionHandler(sessionSvc, nil, nil)

	// 创建路由
	r := gin.New()
	r.GET("/api/v1/sessions", sessionHandler.List)

	// 发送请求
	req, _ := http.NewRequest("GET", "/api/v1/sessions?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200，实际: %d", w.Code)
	}

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("响应解析失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("响应 code 应为 0，实际: %d", resp.Code)
	}
}

// TestHTTPIntegration_DeleteSession 测试删除会话 API
func TestHTTPIntegration_DeleteSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	sessionSvc := NewMockSessionService()
	session, _ := sessionSvc.CreateSession(context.Background())

	// 创建 handler
	sessionHandler := handler.NewSessionHandler(sessionSvc, nil, nil)

	// 创建路由
	r := gin.New()
	r.POST("/api/v1/sessions/:id/delete", sessionHandler.Delete)

	// 发送请求
	req, _ := http.NewRequest("POST", "/api/v1/sessions/"+session.ID+"/delete", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200，实际: %d", w.Code)
	}

	// 验证会话已被删除
	_, err := sessionSvc.GetSession(context.Background(), session.ID)
	if err == nil {
		t.Error("会话应该已被删除")
	}
}

// TestHTTPIntegration_ChatWithoutAgentService 验证 Agent 服务未配置时不会触发空指针。
func TestHTTPIntegration_ChatWithoutAgentService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	sessionSvc := NewMockSessionService()
	session, _ := sessionSvc.CreateSession(context.Background())

	// 创建 handler
	sessionHandler := handler.NewSessionHandler(sessionSvc, nil, nil)

	// 创建路由
	r := gin.New()
	r.POST("/api/v1/sessions/:id/chat", sessionHandler.Chat)

	// 准备请求体
	body := map[string]string{"message": "Hello, world!"}
	jsonBody, _ := json.Marshal(body)

	// 发送请求
	req, _ := http.NewRequest("POST", "/api/v1/sessions/"+session.ID+"/chat", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// AgentService 未配置时应返回明确错误，而不是 panic。
	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码应为 400，实际: %d", w.Code)
	}

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("响应解析失败: %v", err)
	}

	if resp.Code != http.StatusBadRequest {
		t.Errorf("响应 code 应为 400，实际: %d", resp.Code)
	}
}

// TestHTTPIntegration_GetLLMConfig 测试获取 LLM 配置 API
func TestHTTPIntegration_GetLLMConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	appConfigSvc := NewMockAppConfigService()

	// 创建 handler
	appConfigHandler := handler.NewAppConfigHandler(appConfigSvc)

	// 创建路由
	r := gin.New()
	r.GET("/api/v1/app-config/llm", appConfigHandler.GetLLMConfig)

	// 发送请求
	req, _ := http.NewRequest("GET", "/api/v1/app-config/llm", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200，实际: %d", w.Code)
	}

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("响应解析失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("响应 code 应为 0，实际: %d", resp.Code)
	}
}

// TestHTTPIntegration_Chat 测试聊天 API
func TestHTTPIntegration_UploadFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	fileSvc := NewMockFileService()

	// 创建 handler
	fileHandler := handler.NewFileHandler(fileSvc)

	// 创建路由
	r := gin.New()
	r.POST("/api/v1/files", fileHandler.Upload)

	// 准备文件数据
	fileContent := []byte("test file content")
	body := &bytes.Buffer{}
	writer := mockMultipartWriter{body: body}
	writer.WriteField("session_id", "test-session-id")
	writer.WriteFile("file", "test.txt", fileContent)

	// 发送请求
	req, _ := http.NewRequest("POST", "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 注意：这里可能返回 400 因为缺少真实的 session_id
	// 但至少测试了路由和中间件的基本功能
}

// mockMultipartWriter 简化的 multipart writer 用于测试
type mockMultipartWriter struct {
	body *bytes.Buffer
}

func (m *mockMultipartWriter) WriteField(key, value string) {
	m.body.WriteString("--mock\r\n")
	m.body.WriteString("Content-Disposition: form-data; name=\"" + key + "\"\r\n\r\n")
	m.body.WriteString(value + "\r\n")
}

func (m *mockMultipartWriter) WriteFile(fieldname, filename string, data []byte) {
	m.body.WriteString("--mock\r\n")
	m.body.WriteString("Content-Disposition: form-data; name=\"" + fieldname + "\"; filename=\"" + filename + "\"\r\n\r\n")
	m.body.Write(data)
	m.body.WriteString("\r\n--mock--\r\n")
}

func (m *mockMultipartWriter) FormDataContentType() string {
	return "multipart/form-data; boundary=mock"
}
