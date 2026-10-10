package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

type runHandlerAppStub struct {
	created  service.CreateApplicationRunInput
	gotID    string
	cancelID string
	result   *service.ApplicationRun
}

func (s *runHandlerAppStub) Create(_ context.Context, input service.CreateApplicationRunInput) (*service.ApplicationRun, error) {
	s.created = input
	if s.result != nil {
		return s.result, nil
	}
	return &service.ApplicationRun{Run: &model.Run{ID: "run-1", SessionID: input.SessionID, Status: model.RunStatusRunning}}, nil
}

func (s *runHandlerAppStub) SubmitInput(context.Context, string, service.SubmitInputRequest) (*service.ApplicationRun, error) {
	return &service.ApplicationRun{Run: &model.Run{ID: "run-1", Status: model.RunStatusRunning}}, nil
}
func (s *runHandlerAppStub) Get(_ context.Context, id string) (*model.Run, error) {
	s.gotID = id
	return &model.Run{ID: id, Status: model.RunStatusRunning}, nil
}
func (*runHandlerAppStub) GetActiveBySessionID(context.Context, string) (*model.Run, error) {
	return nil, nil
}
func (s *runHandlerAppStub) Cancel(_ context.Context, id string) (*model.Run, error) {
	s.cancelID = id
	return &model.Run{ID: id, Status: model.RunStatusCancelling}, nil
}

type runHandlerEventsStub struct {
	startID string
}

func (s *runHandlerEventsStub) Read(_ context.Context, _, startID string) ([]*model.Event, error) {
	s.startID = startID
	return []*model.Event{{ID: "1710000000000-0", Type: model.EventTypeDone, Data: []byte(`{"status":"done"}`)}}, nil
}

func TestRunHandlerCreateUsesIdempotencyHeaderAndSessionRouteParam(t *testing.T) {
	app := &runHandlerAppStub{}
	router := setupRouter()
	router.POST("/sessions/:sessionId/runs", NewRunHandler(app, nil).Create)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions/session-1/runs", strings.NewReader(`{"message":"分析","attachments":["file-1"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "request-1")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "session-1", app.created.SessionID)
	require.Equal(t, "request-1", app.created.IdempotencyKey)
	require.Equal(t, []string{"file-1"}, app.created.AttachmentIDs)

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
