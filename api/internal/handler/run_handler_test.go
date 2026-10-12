package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

type runHandlerAppStub struct {
	created          service.CreateApplicationRunInput
	gotID            string
	cancelID         string
	result           *service.ApplicationRun
	history          *service.RunHistoryPage
	historySessionID string
	historyLimit     int
	historyOffset    int
	modelID          string
	submittedModelID string
}

func (s *runHandlerAppStub) Create(ctx context.Context, input service.CreateApplicationRunInput) (*service.ApplicationRun, error) {
	s.created = input
	s.modelID = llm.ModelIDFromContext(ctx)
	if s.result != nil {
		return s.result, nil
	}
	return &service.ApplicationRun{Run: &model.Run{ID: "run-1", SessionID: input.SessionID, Status: model.RunStatusRunning}}, nil
}

func (s *runHandlerAppStub) SubmitInput(ctx context.Context, _ string, _ service.SubmitInputRequest) (*service.ApplicationRun, error) {
	s.submittedModelID = llm.ModelIDFromContext(ctx)
	return &service.ApplicationRun{Run: &model.Run{ID: "run-1", Status: model.RunStatusRunning}}, nil
}
func (s *runHandlerAppStub) Get(_ context.Context, id string) (*model.Run, error) {
	s.gotID = id
	return &model.Run{ID: id, Status: model.RunStatusRunning}, nil
}
func (*runHandlerAppStub) GetActiveBySessionID(context.Context, string) (*model.Run, error) {
	return nil, nil
}
func (s *runHandlerAppStub) ListBySessionID(_ context.Context, sessionID string, limit, offset int) (*service.RunHistoryPage, error) {
	s.historySessionID, s.historyLimit, s.historyOffset = sessionID, limit, offset
	if s.history != nil {
		return s.history, nil
	}
	return &service.RunHistoryPage{Items: []service.RunHistoryItem{}, Total: 0, Limit: limit, Offset: offset}, nil
}
func (s *runHandlerAppStub) Cancel(_ context.Context, id string) (*model.Run, error) {
	s.cancelID = id
	return &model.Run{ID: id, Status: model.RunStatusCancelling}, nil
}

type runHandlerEventsStub struct {
	startID string
}

type expiredRunHandlerEventsStub struct{}

func (*expiredRunHandlerEventsStub) Read(context.Context, string, string) ([]*model.Event, error) {
	return nil, service.ErrRunEventStreamExpired
}

type blockingRunHandlerEventsStub struct {
	started chan struct{}
	release chan struct{}
}

// Read 用屏障模拟正在阻塞的事件读取；release 只供失败用例收尾，避免测试泄漏 goroutine。
func (s *blockingRunHandlerEventsStub) Read(ctx context.Context, _, _ string) ([]*model.Event, error) {
	close(s.started)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.release:
		return nil, context.Canceled
	}
}

func (s *runHandlerEventsStub) Read(_ context.Context, _, startID string) ([]*model.Event, error) {
	s.startID = startID
	return []*model.Event{{ID: "1710000000000-0", Type: model.EventTypeDone, Data: []byte(`{"status":"done"}`)}}, nil
}

func TestRunHandlerCreateUsesIdempotencyHeaderAndSessionRouteParam(t *testing.T) {
	app := &runHandlerAppStub{}
	router := setupRouter()
	router.POST("/sessions/:id/runs", NewRunHandler(app, nil).Create)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions/session-1/runs", strings.NewReader(`{"message":"分析","attachments":["file-1"],"model_id":"model-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "request-1")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "session-1", app.created.SessionID)
	require.Equal(t, "request-1", app.created.IdempotencyKey)
	require.Equal(t, []string{"file-1"}, app.created.AttachmentIDs)
	require.Equal(t, "model-1", app.modelID)

	var responseBody response.Response
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &responseBody))
	data, ok := responseBody.Data.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "run-1", data["id"])
	_, hasHandle := data["Handle"]
	require.False(t, hasHandle, "Run API must not expose internal execution handles")
}

func TestRunHandlerEventsPassesLastEventIDAndWritesSSEID(t *testing.T) {
	app := &runHandlerAppStub{}
	events := &runHandlerEventsStub{}
	router := setupRouter()
	router.GET("/runs/:runId/events", NewRunHandler(app, events).Events)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runs/run-1/events", nil)
	req.Header.Set("Last-Event-ID", "1710000000000-1")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "1710000000000-1", events.startID)
	require.Contains(t, w.Body.String(), "id: 1710000000000-0\n")
	require.Contains(t, w.Body.String(), "event: done\n")
	require.Contains(t, w.Body.String(), `"event_id":"1710000000000-0"`)
}

func TestRunHandlerEventsRejectsInvalidLastEventID(t *testing.T) {
	router := setupRouter()
	router.GET("/runs/:runId/events", NewRunHandler(&runHandlerAppStub{}, &runHandlerEventsStub{}).Events)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runs/run-1/events", nil)
	req.Header.Set("Last-Event-ID", "invalid")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NotContains(t, w.Body.String(), "event: stream_error")
}

func TestRunHandlerEventsReportsExpiredCursor(t *testing.T) {
	router := setupRouter()
	router.GET("/runs/:runId/events", NewRunHandler(&runHandlerAppStub{}, &expiredRunHandlerEventsStub{}).Events)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runs/run-1/events", nil)
	req.Header.Set("Last-Event-ID", "100-0")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "event: event_stream_expired\n")
	require.Contains(t, w.Body.String(), `"code":"event_stream_expired"`)
	require.Contains(t, w.Body.String(), `"run_id":"run-1"`)
}

// TestRunHandlerEventsStopsReadingOnDisconnect 守护 HTTP 请求取消向阻塞读取的传播，断线不报告服务故障。
func TestRunHandlerEventsStopsReadingOnDisconnect(t *testing.T) {
	events := &blockingRunHandlerEventsStub{started: make(chan struct{}), release: make(chan struct{})}
	router := setupRouter()
	router.GET("/runs/:runId/events", NewRunHandler(&runHandlerAppStub{}, events).Events)
	ctx, cancel := context.WithCancel(context.Background())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runs/run-1/events", nil).WithContext(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		router.ServeHTTP(w, req)
	}()
	t.Cleanup(func() {
		cancel()
		close(events.release)
		<-done
	})
	select {
	case <-events.started:
	case <-time.After(time.Second):
		t.Fatal("event reader did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("HTTP disconnect did not cancel the blocking event reader")
	}
	require.Empty(t, w.Body.String(), "client disconnect must not emit stream_error")
}

func TestRunHandlerCancelUsesRunID(t *testing.T) {
	app := &runHandlerAppStub{}
	router := setupRouter()
	router.POST("/runs/:runId/cancel", NewRunHandler(app, nil).Cancel)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/runs/run-7/cancel", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "run-7", app.cancelID)
}

func TestRunHandlerSubmitInputPassesRequestedModel(t *testing.T) {
	app := &runHandlerAppStub{}
	router := setupRouter()
	router.POST("/runs/:runId/input", NewRunHandler(app, nil).SubmitInput)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/runs/run-1/input", strings.NewReader(`{"idempotency_key":"reply-1","reply_to_message_id":"question-1","message":"继续","model_id":"model-1"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "model-1", app.submittedModelID)
}

func TestRunHandlerListBySessionPassesPagination(t *testing.T) {
	app := &runHandlerAppStub{history: &service.RunHistoryPage{
		Items: []service.RunHistoryItem{{Run: &model.Run{ID: "run-1", SessionID: "session-1"}, Messages: []*model.RunMessage{{ID: "message-1", Content: "你好"}}}},
		Total: 1, Limit: 20, Offset: 5,
	}}
	router := setupRouter()
	router.GET("/sessions/:id/runs", NewRunHandler(app, nil).ListBySession)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sessions/session-1/runs?limit=20&offset=5", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "session-1", app.historySessionID)
	require.Equal(t, 20, app.historyLimit)
	require.Equal(t, 5, app.historyOffset)
	var responseBody response.Response
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &responseBody))
	data, ok := responseBody.Data.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, float64(1), data["total"])
}
