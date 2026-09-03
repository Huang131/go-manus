package service

import (
	"context"
	"errors"
	"github.com/bytedance/sonic"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
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

func (m *MockAppConfigRepository) GetConfig(ctx context.Context, configType, configKey string) (*model.AppConfig, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, c := range m.configs {
		if c.ConfigType == configType && c.ConfigKey == configKey {
			// 存储时已经统一为 []byte，直接返回即可
			return c, nil
		}
	}
	return nil, nil
}

func (m *MockAppConfigRepository) SaveConfig(ctx context.Context, config *model.AppConfig) error {
	if m.createErr != nil {
		return m.createErr
	}
	// 如果传入的 ConfigValue 已经是 []byte（测试用例中预先 sonic.Marshal 的结果），
	// 直接存储；否则将其 Marshal 为 []byte，模拟生产 repo 的持久化行为。
	var storedValue []byte
	switch v := config.ConfigValue.(type) {
	case []byte:
		storedValue = v
	case nil:
		storedValue = nil
	default:
		b, err := sonic.Marshal(v)
		if err != nil {
			return err
		}
		storedValue = b
	}
	stored := &model.AppConfig{
		ID:          config.ID,
		ConfigType:  config.ConfigType,
		ConfigKey:   config.ConfigKey,
		ConfigValue: storedValue,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}
	// 删除旧配置
	key := config.ConfigType + ":" + config.ConfigKey
	delete(m.configs, key)
	m.configs[key] = stored
	return nil
}

func (m *MockAppConfigRepository) DeleteConfig(ctx context.Context, configType, configKey string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	key := configType + ":" + configKey
	delete(m.configs, key)
	return nil
}

func (m *MockAppConfigRepository) ListConfigs(ctx context.Context, configType string) ([]*model.AppConfig, error) {
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

func TestAppConfigService_GetLLMConfig(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	// 创建 LLM 配置
	llmConfig := &model.LLMConfig{
		BaseURL:     "https://api.openai.com",
		ModelName:   "gpt-4",
		APIKey:      "test-key",
		Temperature: 0.7,
		MaxTokens:   4096,
	}
	configValue, _ := sonic.Marshal(llmConfig)
	config := &model.AppConfig{
		ConfigType:  "llm",
		ConfigKey:   "default",
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	// 获取 LLM 配置
	retrieved, err := svc.GetLLMConfig(context.Background())
	if err != nil {
		t.Fatalf("GetLLMConfig() error = %v", err)
	}

	if retrieved.BaseURL != "https://api.openai.com" {
		t.Errorf("LLMConfig.BaseURL = %s, want https://api.openai.com", retrieved.BaseURL)
	}
	if retrieved.ModelName != "gpt-4" {
		t.Errorf("LLMConfig.ModelName = %s, want gpt-4", retrieved.ModelName)
	}
	if retrieved.APIKey != "test-key" {
		t.Errorf("LLMConfig.APIKey = %s, want test-key", retrieved.APIKey)
	}
}

func TestAppConfigService_GetLLMConfig_NotFound(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	llmConfig, err := svc.GetLLMConfig(context.Background())
	if err != nil {
		t.Fatalf("GetLLMConfig() error = %v", err)
	}

	if llmConfig != nil {
		t.Error("GetLLMConfig() should return nil when not found")
	}
}

func TestAppConfigService_UpdateLLMConfig(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	// 先创建默认 LLM 配置
	llmConfig := &model.LLMConfig{
		BaseURL:     "https://api.openai.com",
		ModelName:   "gpt-4",
		APIKey:      "old-key",
		Temperature: 0.7,
		MaxTokens:   4096,
	}
	configValue, _ := sonic.Marshal(llmConfig)
	config := &model.AppConfig{
		ConfigType:  "llm",
		ConfigKey:   "default",
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	// 更新配置 (api_key 为空时保留旧值)
	newConfig := &model.LLMConfig{
		BaseURL:     "https://api.anthropic.com",
		ModelName:   "claude-3",
		APIKey:      "", // 空值，不覆盖旧值
		Temperature: 0.7,
		MaxTokens:   8192,
	}

	err := svc.UpdateLLMConfig(context.Background(), newConfig)
	if err != nil {
		t.Fatalf("UpdateLLMConfig() error = %v", err)
	}

	// 验证更新后的配置
	updated, _ := svc.GetLLMConfig(context.Background())
	if updated.BaseURL != "https://api.anthropic.com" {
		t.Errorf("Updated BaseURL = %s, want https://api.anthropic.com", updated.BaseURL)
	}
	if updated.ModelName != "claude-3" {
		t.Errorf("Updated ModelName = %s, want claude-3", updated.ModelName)
	}
	if updated.APIKey != "old-key" {
		t.Errorf("Updated APIKey = %s, want old-key (preserved)", updated.APIKey)
	}
	if updated.MaxTokens != 8192 {
		t.Errorf("Updated MaxTokens = %d, want 8192", updated.MaxTokens)
	}
}

func TestAppConfigService_UpdateLLMConfig_WithNewApiKey(t *testing.T) {
	repo := NewMockAppConfigRepository()
	svc := NewAppConfigService(repo)

	// 先创建默认 LLM 配置
	llmConfig := &model.LLMConfig{
		BaseURL:   "https://api.openai.com",
		ModelName: "gpt-4",
		APIKey:    "old-key",
	}
	configValue, _ := sonic.Marshal(llmConfig)
	config := &model.AppConfig{
		ConfigType:  "llm",
		ConfigKey:   "default",
		ConfigValue: configValue,
	}
	repo.SaveConfig(context.Background(), config)

	// 更新配置 (提供新的 api_key)
	newConfig := &model.LLMConfig{
		BaseURL:   "https://api.anthropic.com",
		ModelName: "claude-3",
		APIKey:    "new-key",
	}

	err := svc.UpdateLLMConfig(context.Background(), newConfig)
	if err != nil {
		t.Fatalf("UpdateLLMConfig() error = %v", err)
	}

	// 验证 api_key 已更新
	updated, _ := svc.GetLLMConfig(context.Background())
	if updated.APIKey != "new-key" {
		t.Errorf("Updated APIKey = %s, want new-key", updated.APIKey)
	}
}

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
		ConfigType:  "agent",
		ConfigKey:   "default",
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
		ConfigType:  "mcp",
		ConfigKey:   "default",
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
		ConfigType:  "a2a",
		ConfigKey:   "default",
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

	// 添加新服务器
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
		ConfigType:  "mcp",
		ConfigKey:   "default",
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

	_, err := svc.GetLLMConfig(context.Background())
	if err == nil {
		t.Error("GetLLMConfig() should return error when repository fails")
	}
}
