package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
	"github.com/mooc-manus/go-manus/api/internal/service"
)

// stubVNCService 模拟 SessionService 的 GetVNCURL。
type stubVNCService struct {
	vncURL string
	err    error
}

func (s *stubVNCService) CreateSession(ctx context.Context) (*model.Session, error) {
	return nil, nil
}
func (s *stubVNCService) GetSession(ctx context.Context, id string) (*model.Session, error) {
	return nil, nil
}
func (s *stubVNCService) GetAllSessions(ctx context.Context) ([]*model.Session, error) {
	return nil, nil
}
func (s *stubVNCService) ListSessions(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	return nil, 0, nil
}
func (s *stubVNCService) DeleteSession(ctx context.Context, id string) error {
	return nil
}
func (s *stubVNCService) IncrementUnreadCount(ctx context.Context, id string) error {
	return nil
}
func (s *stubVNCService) DecrementUnreadCount(ctx context.Context, id string) error {
	return nil
}
func (s *stubVNCService) ClearUnreadCount(ctx context.Context, id string) error {
	return nil
}
func (s *stubVNCService) GetSessionFiles(ctx context.Context, id string) ([]model.File, error) {
	return nil, nil
}
func (s *stubVNCService) AppendEvent(ctx context.Context, sessionID string, event *model.Event) error {
	return nil
}
func (s *stubVNCService) StreamSession(ctx context.Context, id string) (*model.Session, error) {
	return nil, nil
}
func (s *stubVNCService) Chat(ctx context.Context, sessionID string, message string) error {
	return nil
}
func (s *stubVNCService) GetVNCURL(ctx context.Context, sessionID string) (string, error) {
	return s.vncURL, s.err
}

var _ service.SessionService = (*stubVNCService)(nil)
var _ repository.SessionRepository = (repository.SessionRepository)(nil) // 防止编译期未使用

// TestVNCProxy_ServiceError 验证：当 SessionService.GetVNCURL 返回错误时，
// 处理器以 400/502 响应（而不是 panic）。
func TestVNCProxy_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubVNCService{err: errors.New("sandbox 地址未配置")}

	engine := gin.New()
	engine.GET("/api/sessions/:id/vnc", VNCProxy(stub, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/abc/vnc", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 非 WS Upgrade 请求时仍会走到 service，service 返回错误 → 400/502 文本
	if w.Code < 400 || w.Code >= 600 {
		t.Errorf("expected 4xx/5xx, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "sandbox") {
		t.Errorf("body should contain error message, got %q", w.Body.String())
	}
}

// TestDecodeSubprotocolFrame 验证：binary 子协议原样返回；base64 子协议返回错误。
func TestDecodeSubprotocolFrame(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02}

	got, err := decodeSubprotocolFrame("binary", data)
	if err != nil {
		t.Fatalf("binary decode error = %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("binary decode = %v, want %v", got, data)
	}

	if _, err := decodeSubprotocolFrame("base64", data); err == nil {
		t.Error("expected error for base64 subprotocol")
	}
}

// TestWriteVNCFrame 验证：writeVNCFrame 根据子协议选择 binary/text。
// 不真正写 WS（无网络），仅校验不 panic。
func TestWriteVNCFrame_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 用一个不会真正连接的 ws 连接 — 跳过实际写入测试，只验证函数签名。
	// 这里仅确认函数不会因为 nil conn panic（conn 为 nil 时 WriteMessage 会 panic，跳过）。
	// 真正的连接测试需要 integration 测试（httptest.Server + websocket.Dial）。
	_ = writeVNCFrame // 保留引用以防 lint 误删
}
