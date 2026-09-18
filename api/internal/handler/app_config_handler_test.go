package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/agent"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/gin-gonic/gin"
)

type appConfigServiceStub struct {
	service.AppConfigService
}

func (appConfigServiceStub) UpdateAgentConfig(context.Context, *model.AgentConfig) error {
	return nil
}

type configReloaderStub struct {
	agentConfig *agent.AgentConfig
}

func (r *configReloaderStub) ReloadAgentConfig(cfg *agent.AgentConfig) {
	r.agentConfig = cfg
}

func (*configReloaderStub) ReloadMCPConfig(context.Context, *agent.MCPConfig) error { return nil }
func (*configReloaderStub) ReloadA2AConfig(context.Context, *agent.A2AConfig) error { return nil }

func TestUpdateAgentConfigReloadsSearchLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reloader := &configReloaderStub{}
	h := NewAppConfigHandler(appConfigServiceStub{}, reloader)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/config/agent", strings.NewReader(
		`{"max_iterations":8,"max_retries":2,"max_search_results":6}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateAgentConfig(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if reloader.agentConfig == nil || reloader.agentConfig.MaxSearchResults != 6 {
		t.Fatalf("reloaded config = %+v, want max_search_results=6", reloader.agentConfig)
	}
}
