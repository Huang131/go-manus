package service

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// MockAppConfigRepository 用于测试的 AppConfig Repository Mock
type MockAppConfigRepository struct {
	configs   map[string]*model.AppConfig
	createErr error
	getErr    error
	deleteErr error
}

func NewMockAppConfigRepository() *MockAppConfigRepository {
	return &MockAppConfigRepository{
		configs: make(map[string]*model.AppConfig),
	}
}

func (m *MockAppConfigRepository) GetConfig(ctx context.Context, configType model.AppConfigType, configKey string) (*model.AppConfig, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, c := range m.configs {
		if c.ConfigType == configType && c.ConfigKey == configKey {
			return c, nil
		}
	}
	return nil, nil
}

func (m *MockAppConfigRepository) SaveConfig(ctx context.Context, config *model.AppConfig) error {
	if m.createErr != nil {
		return m.createErr
	}
	// ConfigValue 已是原始 JSON（json.RawMessage），直接存储，模拟生产 repo 的持久化行为。
	stored := &model.AppConfig{
		ID:          config.ID,
		ConfigType:  config.ConfigType,
		ConfigKey:   config.ConfigKey,
		ConfigValue: config.ConfigValue,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}
	// 删除旧配置
	key := string(config.ConfigType) + ":" + config.ConfigKey
	delete(m.configs, key)
	m.configs[key] = stored
	return nil
}

func (m *MockAppConfigRepository) DeleteConfig(ctx context.Context, configType model.AppConfigType, configKey string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	key := string(configType) + ":" + configKey
	delete(m.configs, key)
	return nil
}

func (m *MockAppConfigRepository) ListConfigs(ctx context.Context, configType model.AppConfigType) ([]*model.AppConfig, error) {
	var result []*model.AppConfig
	for _, c := range m.configs {
		if c.ConfigType == configType {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *MockAppConfigRepository) ListAllConfigs(ctx context.Context) ([]*model.AppConfig, error) {
	result := make([]*model.AppConfig, 0, len(m.configs))
	for _, c := range m.configs {
		result = append(result, c)
	}
	return result, nil
}

func (m *MockAppConfigRepository) WithTx(ctx context.Context, fn func(repo repository.AppConfigRepository) error) error {
	return fn(m)
}

// 确保 Mock 实现正确的接口
var _ repository.AppConfigRepository = (*MockAppConfigRepository)(nil)

func TestAppConfigService_GetAgentConfig(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	agentConfig := &model.AgentConfig{
		MaxIterations:    10,
		MaxRetries:       3,
		MaxSearchResults: 5,
	}
	configValue, _ := sonic.Marshal(agentConfig)
	config := &model.AppConfig{
		ConfigType:  model.AppConfigTypeAgent,
		ConfigKey:   model.AppConfigKeyDefault,
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	retrieved, err := svc.GetAgentConfig(context.Background())
	if err != nil {
		t.Fatalf("GetAgentConfig() error = %v", err)
	}

	if retrieved.MaxIterations != 10 {
		t.Errorf("AgentConfig.MaxIterations = %d, want 10", retrieved.MaxIterations)
	}
	if retrieved.MaxRetries != 3 {
		t.Errorf("AgentConfig.MaxRetries = %d, want 3", retrieved.MaxRetries)
	}
	if retrieved.MaxSearchResults != 5 {
		t.Errorf("AgentConfig.MaxSearchResults = %d, want 5", retrieved.MaxSearchResults)
	}
}

func TestAppConfigService_GetMCPConfig(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	mcpConfig := &model.MCPConfig{
		Servers: []model.MCPServer{
			{
				ServerName: "filesystem",
				Enabled:    true,
				Transport:  "stdio",
				Tools:      []string{"read_file", "write_file"},
			},
		},
	}
	configValue, _ := sonic.Marshal(mcpConfig)
	config := &model.AppConfig{
		ConfigType:  model.AppConfigTypeMCP,
		ConfigKey:   model.AppConfigKeyDefault,
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	retrieved, err := svc.GetMCPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetMCPConfig() error = %v", err)
	}

	if len(retrieved.Servers) != 1 {
		t.Errorf("MCPConfig.Servers length = %d, want 1", len(retrieved.Servers))
	}
	if retrieved.Servers[0].ServerName != "filesystem" {
		t.Errorf("MCPConfig.Servers[0].ServerName = %s, want filesystem", retrieved.Servers[0].ServerName)
	}
	if !retrieved.Servers[0].Enabled {
		t.Error("MCPConfig.Servers[0].Enabled should be true")
	}
}

func TestAppConfigService_GetA2AConfig(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	a2aConfig := &model.A2AConfig{
		Servers: []model.A2AServer{
			{
				ID:          "server-1",
				Name:        "Test Server",
				Description: "A test A2A server",
				InputModes:  []string{"text", "json"},
				OutputModes: []string{"text", "json"},
				Streaming:   true,
				Enabled:     true,
			},
		},
	}
	configValue, _ := sonic.Marshal(a2aConfig)
	config := &model.AppConfig{
		ConfigType:  model.AppConfigTypeA2A,
		ConfigKey:   model.AppConfigKeyDefault,
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	retrieved, err := svc.GetA2AConfig(context.Background())
	if err != nil {
		t.Fatalf("GetA2AConfig() error = %v", err)
	}

	if len(retrieved.Servers) != 1 {
		t.Errorf("A2AConfig.Servers length = %d, want 1", len(retrieved.Servers))
	}
	if !retrieved.Servers[0].Enabled {
		t.Error("A2AConfig.Servers[0].Enabled should be true")
	}
	if retrieved.Servers[0].Streaming != true {
		t.Error("A2AConfig.Servers[0].Streaming should be true")
	}
}

func TestAppConfigService_UpdateMCPConfig(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	// 先创建默认 MCP 配置
	mcpConfig := &model.MCPConfig{
		Servers: []model.MCPServer{
			{ServerName: "server-1", Enabled: true, Transport: "stdio"},
		},
	}
	configValue, _ := sonic.Marshal(mcpConfig)
	config := &model.AppConfig{
		ConfigType:  "mcp",
		ConfigKey:   "default",
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	// 添加新服务器（测试同名覆盖 + 新增场景）
	newConfig := &model.MCPConfig{
		Servers: []model.MCPServer{
			{ServerName: "server-1", Enabled: true, Transport: "stdio"},
			{ServerName: "server-2", Enabled: false, Transport: "http"},
		},
	}

	err := svc.UpdateMCPConfig(context.Background(), newConfig)
	if err != nil {
		t.Fatalf("UpdateMCPConfig() error = %v", err)
	}

	updated, _ := svc.GetMCPConfig(context.Background())
	if len(updated.Servers) != 2 {
		t.Errorf("Updated MCPConfig.Servers length = %d, want 2", len(updated.Servers))
	}
}

func TestAppConfigService_DeleteMCPServer(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	// 先创建默认 MCP 配置
	mcpConfig := &model.MCPConfig{
		Servers: []model.MCPServer{
			{ServerName: "server-1", Enabled: true, Transport: "stdio"},
			{ServerName: "server-2", Enabled: true, Transport: "http"},
		},
	}
	configValue, _ := sonic.Marshal(mcpConfig)
	config := &model.AppConfig{
		ConfigType:  model.AppConfigTypeMCP,
		ConfigKey:   model.AppConfigKeyDefault,
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	// 删除服务器
	err := svc.DeleteMCPServer(context.Background(), "server-1")
	if err != nil {
		t.Fatalf("DeleteMCPServer() error = %v", err)
	}

	updated, _ := svc.GetMCPConfig(context.Background())
	if len(updated.Servers) != 1 {
		t.Errorf("Updated MCPConfig.Servers length = %d, want 1", len(updated.Servers))
	}
	if updated.Servers[0].ServerName != "server-2" {
		t.Errorf("Remaining server should be server-2, got %s", updated.Servers[0].ServerName)
	}
}

func TestAppConfigService_GetConfig_RepositoryError(t *testing.T) {
	repo := NewMockAppConfigRepository()
	repo.getErr = errors.New("database error")
	svc := NewAppConfigService(repo)

	_, err := svc.GetMCPConfig(context.Background())
	if err == nil {
		t.Error("GetMCPConfig() should return error when repository fails")
	}
}

func TestAppConfigService_DeleteMCPServer_NotFoundWithoutConfig(t *testing.T) {
	svc := NewAppConfigService(NewMockAppConfigRepository())

	err := svc.DeleteMCPServer(context.Background(), "missing")
	if err == nil {
		t.Fatal("DeleteMCPServer() error = nil, want not found")
	}
}

func TestAppConfigService_UpdateMCPServerEnabled(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)
	if err := svc.UpdateMCPConfig(context.Background(), &model.MCPConfig{Servers: []model.MCPServer{{ServerName: "s", Enabled: false}}}); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateMCPServerEnabled(context.Background(), "s", true); err != nil {
		t.Fatal(err)
	}
	cfg, err := svc.GetMCPConfig(context.Background())
	if err != nil || cfg == nil || !cfg.Servers[0].Enabled {
		t.Fatalf("enabled state not persisted: cfg=%+v err=%v", cfg, err)
	}
}

func TestAppConfigService_DeleteAndUpdateA2AServer(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)
	if err := svc.UpdateA2AConfig(context.Background(), &model.A2AConfig{Servers: []model.A2AServer{{ID: "a", URL: "http://a", Enabled: true}, {ID: "b", URL: "http://b", Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateA2AServerEnabled(context.Background(), "a", false); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteA2AServer(context.Background(), "b"); err != nil {
		t.Fatal(err)
	}
	cfg, err := svc.GetA2AConfig(context.Background())
	if err != nil || cfg == nil || len(cfg.Servers) != 1 || cfg.Servers[0].Enabled {
		t.Fatalf("A2A mutation not persisted: cfg=%+v err=%v", cfg, err)
	}
}
