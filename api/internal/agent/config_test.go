package agent

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"

	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"
)

func TestRuntimeMCPConfigFiltersDisabledServersAndCopiesValues(t *testing.T) {
	persisted := &model.MCPConfig{Servers: []model.MCPServer{
		{Name: "enabled", Enabled: true, Command: "cmd", Args: []string{"one"}, Env: map[string]string{"KEY": "value"}},
		{Name: "disabled", Enabled: false, Command: "ignored"},
	}}

	got := RuntimeMCPConfig(persisted)
	if got == nil || len(got.Servers) != 1 || got.Servers[0].Name != "enabled" {
		t.Fatalf("RuntimeMCPConfig() = %+v, want enabled server only", got)
	}
	persisted.Servers[0].Args[0] = "changed"
	persisted.Servers[0].Env["KEY"] = "changed"
	if got.Servers[0].Args[0] != "one" || got.Servers[0].Env["KEY"] != "value" {
		t.Fatalf("runtime config shares mutable persisted data: %+v", got.Servers[0])
	}
}

func TestToolProviderOmitsEmptyDynamicTools(t *testing.T) {
	provider := toolspkg.NewToolProvider(context.Background(), nil, nil, nil, &model.MCPConfig{}, &toolspkg.A2AConfig{})
	got := provider.Tools(DefaultAgentConfig().MaxSearchResults)
	if len(got) != 1 || got[0].Name() != toolspkg.ToolNameMessage {
		t.Fatalf("Tools() = %v, want message tool only", got)
	}
}

func TestRuntimeA2AConfigFiltersDisabledServers(t *testing.T) {
	persisted := &model.A2AConfig{Servers: []model.A2AServer{
		{ID: "enabled", URL: "https://enabled.example", Enabled: true},
		{ID: "disabled", URL: "https://disabled.example", Enabled: false},
	}}

	got := RuntimeA2AConfig(persisted)
	if got == nil || len(got.Agents) != 1 {
		t.Fatalf("RuntimeA2AConfig() = %+v, want enabled server only", got)
	}
	if got.Agents[0].Name != "enabled" || got.Agents[0].URL != "https://enabled.example" {
		t.Fatalf("runtime A2A server = %+v", got.Agents[0])
	}
}
