package handler

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type stubVNCService struct {
	vncURL string
	err    error
}

func (s *stubVNCService) CreateSession(ctx context.Context) (*model.Session, error) { return nil, nil }
func (s *stubVNCService) GetSession(ctx context.Context, id string) (*model.Session, error) {
	return nil, nil
}
func (s *stubVNCService) GetAllSessions(ctx context.Context) ([]*model.Session, error) {
	return nil, nil
}
func (s *stubVNCService) ListSessions(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	return nil, 0, nil
}
func (s *stubVNCService) DeleteSession(ctx context.Context, id string) error        { return nil }
func (s *stubVNCService) IncrementUnreadCount(ctx context.Context, id string) error { return nil }
func (s *stubVNCService) DecrementUnreadCount(ctx context.Context, id string) error { return nil }
func (s *stubVNCService) ClearUnreadCount(ctx context.Context, id string) error     { return nil }
func (s *stubVNCService) GetSessionFiles(ctx context.Context, id string) ([]*model.File, error) {
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
var _ repository.SessionRepository = (repository.SessionRepository)(nil)

func TestVNCProxy_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.GET("/api/sessions/:id/vnc", VNCProxy(&stubVNCService{err: errors.New("sandbox 地址未配置")}))

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/abc/vnc", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadGateway)
	}
}

func TestVNCProxy_ProxyEcho(t *testing.T) {
	gin.SetMode(gin.TestMode)

	echoServer := newLocalTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade echo server: %v", err)
			return
		}
		defer conn.Close()

		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	}))
	defer echoServer.Close()

	wsURL := "ws" + echoServer.URL[len("http"):]
	engine := gin.New()
	engine.GET("/api/sessions/:id/vnc", VNCProxy(&stubVNCService{vncURL: wsURL}))

	server := newLocalTestServer(t, engine)
	defer server.Close()

	dialURL := "ws" + server.URL[len("http"):] + "/api/sessions/abc/vnc"
	client, _, err := websocket.DefaultDialer.Dial(dialURL, nil)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer client.Close()

	if err := client.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("write message: %v", err)
	}
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, data, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if msgType != websocket.TextMessage {
		t.Fatalf("msgType = %d, want %d", msgType, websocket.TextMessage)
	}
	if string(data) != "hello" {
		t.Fatalf("echo = %q, want %q", string(data), "hello")
	}
}

func newLocalTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen local test server: %v", err)
	}

	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}
