package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/response"
)

// sessionServiceStub 只返回预设结果并记录 handler 传入的参数。
// 它不维护 Session 状态，避免在 handler 测试中重写 service 业务语义。
type sessionServiceStub struct {
	createResult *model.Session
	createErr    error
	getResult    *model.Session
	getErr       error
	listResult   []*model.Session
	listTotal    int
	listErr      error
	filesResult  []*model.File
	filesErr     error

	listLimit      int
	listOffset     int
	deletedID      string
	clearedID      string
	filesSessionID string
}

func (s *sessionServiceStub) CreateSession(context.Context) (*model.Session, error) {
	return s.createResult, s.createErr
}

func (s *sessionServiceStub) GetSession(context.Context, string) (*model.Session, error) {
	return s.getResult, s.getErr
}

func (s *sessionServiceStub) GetAllSessions(context.Context) ([]*model.Session, error) {
	return s.listResult, s.listErr
}

func (s *sessionServiceStub) ListSessions(_ context.Context, limit, offset int) ([]*model.Session, int, error) {
	s.listLimit = limit
	s.listOffset = offset
	return s.listResult, s.listTotal, s.listErr
}

func (s *sessionServiceStub) DeleteSession(_ context.Context, id string) error {
	s.deletedID = id
	return nil
}

func (s *sessionServiceStub) RenameSession(context.Context, string, string) error {
	return nil
}

func (s *sessionServiceStub) ClearUnreadCount(_ context.Context, id string) error {
	s.clearedID = id
	return nil
}

func (s *sessionServiceStub) GetSessionFiles(_ context.Context, id string) ([]*model.File, error) {
	s.filesSessionID = id
	return s.filesResult, s.filesErr
}

func (s *sessionServiceStub) AppendEvent(context.Context, string, *model.Event) error {
	return nil
}

func (s *sessionServiceStub) GetVNCURL(context.Context, string) (string, error) {
	return "ws://sandbox.local:5901", nil
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func decodeHandlerResponse(t *testing.T, w *httptest.ResponseRecorder) response.Response {
	t.Helper()
	var resp response.Response
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func TestSessionHandler_Create(t *testing.T) {
	svc := &sessionServiceStub{createResult: &model.Session{ID: "session-1", Title: "新对话"}}
	router := setupRouter()
	router.POST("/sessions", NewSessionHandler(svc, nil, nil).Create)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/sessions", nil))

	require.Equal(t, http.StatusOK, w.Code)
	resp := decodeHandlerResponse(t, w)
	require.Equal(t, 0, resp.Code)
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "新对话", data["title"])
}

func TestSessionHandler_Get(t *testing.T) {
	t.Run("returns session and clears unread count", func(t *testing.T) {
		svc := &sessionServiceStub{getResult: &model.Session{ID: "session-1", UnreadMessageCount: 5}}
		router := setupRouter()
		router.GET("/sessions/:id", NewSessionHandler(svc, nil, nil).Get)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sessions/session-1", nil))

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "session-1", svc.clearedID)
		assert.Equal(t, 0, svc.getResult.UnreadMessageCount)
	})

	t.Run("maps service not found", func(t *testing.T) {
		svc := &sessionServiceStub{getErr: apperr.NotFound("会话不存在")}
		router := setupRouter()
		router.GET("/sessions/:id", NewSessionHandler(svc, nil, nil).Get)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sessions/missing", nil))

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, http.StatusNotFound, decodeHandlerResponse(t, w).Code)
	})
}

func TestSessionHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantLimit  int
		wantOffset int
	}{
		{name: "defaults", wantStatus: http.StatusOK, wantLimit: 20},
		{name: "explicit pagination", query: "?limit=10&offset=3", wantStatus: http.StatusOK, wantLimit: 10, wantOffset: 3},
		{name: "invalid limit", query: "?limit=0", wantStatus: http.StatusBadRequest},
		{name: "invalid offset", query: "?offset=-1", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sessionServiceStub{listResult: []*model.Session{{ID: "session-1"}}, listTotal: 1}
			router := setupRouter()
			router.GET("/sessions", NewSessionHandler(svc, nil, nil).List)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sessions"+tt.query, nil))

			require.Equal(t, tt.wantStatus, w.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, tt.wantLimit, svc.listLimit)
				assert.Equal(t, tt.wantOffset, svc.listOffset)
			}
		})
	}
}

func TestSessionHandler_Delete(t *testing.T) {
	svc := &sessionServiceStub{}
	router := setupRouter()
	router.POST("/sessions/:id/delete", NewSessionHandler(svc, nil, nil).Delete)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/sessions/session-1/delete", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", svc.deletedID)
}

func TestSessionHandler_ClearUnread(t *testing.T) {
	svc := &sessionServiceStub{}
	router := setupRouter()
	router.POST("/sessions/:id/clear-unread", NewSessionHandler(svc, nil, nil).ClearUnread)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/sessions/session-1/clear-unread", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", svc.clearedID)
}

func TestSessionHandler_GetFiles(t *testing.T) {
	svc := &sessionServiceStub{filesResult: []*model.File{}}
	router := setupRouter()
	router.GET("/sessions/:id/files", NewSessionHandler(svc, nil, nil).GetFiles)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sessions/session-1/files", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", svc.filesSessionID)
}

func TestNewSSEContext_PreservesRequestValuesWithoutCancellation(t *testing.T) {
	type contextKey string
	key := contextKey("request_id")
	requestCtx, requestCancel := context.WithCancel(context.WithValue(context.Background(), key, "req-123"))
	requestCancel()

	eventCtx, cancel := newSSEContext(requestCtx)
	defer cancel()

	assert.Equal(t, "req-123", eventCtx.Value(key))
	assert.NoError(t, eventCtx.Err())
}
