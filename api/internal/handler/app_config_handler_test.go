package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/agent"
	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/internal/settings"
	"github.com/gin-gonic/gin"
)

type appConfigServiceStub struct {
	service.AppConfigService
	agentSettings   *settings.AgentSettings
	updatedSettings *settings.AgentSettings
	updateErr       error
}

func (s appConfigServiceStub) GetAgentSettings(context.Context) (*settings.AgentSettings, error) {
	return s.agentSettings, nil
}

func (s *appConfigServiceStub) UpdateAgentSettings(_ context.Context, cfg *settings.AgentSettings) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	copy := *cfg
	s.updatedSettings = &copy
	return nil
}

type configReloaderStub struct{}

func (*configReloaderStub) ReloadMCPConfig(context.Context, *model.MCPConfig) error { return nil }
func (*configReloaderStub) ReloadA2AConfig(context.Context, *agent.A2AConfig) error { return nil }

func TestUpdateAgentSettingsPersistsSearchLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &appConfigServiceStub{}
	h := NewAppConfigHandler(svc, &configReloaderStub{})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/config/agent", strings.NewReader(
		`{"max_iterations":8,"max_retries":2,"max_search_results":6}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateAgentSettings(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if svc.updatedSettings == nil || svc.updatedSettings.MaxSearchResults != 6 {
		t.Fatalf("persisted settings = %+v, want max_search_results=6", svc.updatedSettings)
	}
}

func TestGetAgentSettingsReturnsDefaultsWhenRecordIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAppConfigHandler(&appConfigServiceStub{})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/config/agent", nil)

	h.GetAgentSettings(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, fragment := range []string{`"max_iterations":10`, `"max_retries":3`, `"max_search_results":10`} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("body = %s, want fragment %s", body, fragment)
		}
	}
}

func TestUpdateAgentSettingsReturnsBadRequestForInvalidSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAppConfigHandler(&appConfigServiceStub{updateErr: apperr.BadRequest("invalid settings")})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/config/agent", strings.NewReader(
		`{"max_iterations":0,"max_retries":3,"max_search_results":10}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateAgentSettings(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
}
