package service

import (
	"context"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/google/uuid"
)

// AppConfigService 应用配置服务接口
type AppConfigService interface {
	GetLLMConfig(ctx context.Context) (*model.LLMConfig, error)
	UpdateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error
	GetAgentConfig(ctx context.Context) (*model.AgentConfig, error)
	UpdateAgentConfig(ctx context.Context, cfg *model.AgentConfig) error
	GetMCPConfig(ctx context.Context) (*model.MCPConfig, error)
	UpdateMCPConfig(ctx context.Context, cfg *model.MCPConfig) error
	DeleteMCPServer(ctx context.Context, serverName string) error
	GetA2AConfig(ctx context.Context) (*model.A2AConfig, error)
	UpdateA2AConfig(ctx context.Context, cfg *model.A2AConfig) error
}

// DefaultAppConfigService 应用配置服务默认实现
type DefaultAppConfigService struct {
	repo repository.AppConfigRepository
}

// NewAppConfigService 创建应用配置服务
func NewAppConfigService(repo repository.AppConfigRepository) AppConfigService {
	return &DefaultAppConfigService{repo: repo}
}

// unmarshalConfigValue 安全解析配置值
func unmarshalConfigValue(data interface{}, v interface{}) error {
	var jsonData []byte
	switch val := data.(type) {
	case []byte:
		jsonData = val
	case string:
		jsonData = []byte(val)
	default:
		return apperr.BadRequest("unsupported config value type")
	}
	return sonic.Unmarshal(jsonData, v)
}

// GetLLMConfig 获取 LLM 配置
func (s *DefaultAppConfigService) GetLLMConfig(ctx context.Context) (*model.LLMConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, "llm", "default")
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}

	var llmConfig model.LLMConfig
	if err := unmarshalConfigValue(cfg.ConfigValue, &llmConfig); err != nil {
		return nil, err
	}
	return &llmConfig, nil
}

// UpdateLLMConfig 更新 LLM 配置 (如果 api_key 为空则保留旧值)
func (s *DefaultAppConfigService) UpdateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	// 如果 api_key 为空，保留旧值
	if cfg.APIKey == "" {
		oldCfg, err := s.GetLLMConfig(ctx)
		if err == nil && oldCfg != nil {
			cfg.APIKey = oldCfg.APIKey
		}
	}

	appConfig := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  "llm",
		ConfigKey:   "default",
		ConfigValue: cfg,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// GetAgentConfig 获取 Agent 配置
func (s *DefaultAppConfigService) GetAgentConfig(ctx context.Context) (*model.AgentConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, "agent", "default")
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}
	var agentConfig model.AgentConfig
	if err := unmarshalConfigValue(cfg.ConfigValue, &agentConfig); err != nil {
		return nil, err
	}
	return &agentConfig, nil
}

// UpdateAgentConfig 更新 Agent 配置
func (s *DefaultAppConfigService) UpdateAgentConfig(ctx context.Context, cfg *model.AgentConfig) error {
	appConfig := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  "agent",
		ConfigKey:   "default",
		ConfigValue: cfg,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// GetMCPConfig 获取 MCP 配置
func (s *DefaultAppConfigService) GetMCPConfig(ctx context.Context) (*model.MCPConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, "mcp", "default")
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}
	var mcpConfig model.MCPConfig
	if err := unmarshalConfigValue(cfg.ConfigValue, &mcpConfig); err != nil {
		return nil, err
	}
	return &mcpConfig, nil
}

// UpdateMCPConfig 更新 MCP 配置 (合并服务器列表)
func (s *DefaultAppConfigService) UpdateMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	// 获取现有配置
	oldCfg, err := s.GetMCPConfig(ctx)
	if err == nil && oldCfg != nil {
		// 合并服务器配置
		existingServers := make(map[string]model.MCPServer)
		for _, server := range oldCfg.Servers {
			existingServers[server.ServerName] = server
		}
		for _, server := range cfg.Servers {
			existingServers[server.ServerName] = server
		}
		cfg.Servers = make([]model.MCPServer, 0, len(existingServers))
		for _, server := range existingServers {
			cfg.Servers = append(cfg.Servers, server)
		}
	}

	appConfig := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  "mcp",
		ConfigKey:   "default",
		ConfigValue: cfg,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// DeleteMCPServer 删除 MCP 服务器
func (s *DefaultAppConfigService) DeleteMCPServer(ctx context.Context, serverName string) error {
	cfg, err := s.GetMCPConfig(ctx)
	if err != nil {
		return err
	}

	// 查找并删除服务器
	found := false
	newServers := make([]model.MCPServer, 0, len(cfg.Servers))
	for _, server := range cfg.Servers {
		if server.ServerName == serverName {
			found = true
			continue
		}
		newServers = append(newServers, server)
	}
	if !found {
		return apperr.NotFound("MCP服务器不存在")
	}

	cfg.Servers = newServers
	appConfig := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  "mcp",
		ConfigKey:   "default",
		ConfigValue: cfg,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// GetA2AConfig 获取 A2A 配置
func (s *DefaultAppConfigService) GetA2AConfig(ctx context.Context) (*model.A2AConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, "a2a", "default")
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}
	var a2aConfig model.A2AConfig
	if err := unmarshalConfigValue(cfg.ConfigValue, &a2aConfig); err != nil {
		return nil, err
	}
	return &a2aConfig, nil
}

// UpdateA2AConfig 更新 A2A 配置
func (s *DefaultAppConfigService) UpdateA2AConfig(ctx context.Context, cfg *model.A2AConfig) error {
	// 获取现有配置
	oldCfg, err := s.GetA2AConfig(ctx)
	if err == nil && oldCfg != nil {
		// 合并服务器配置
		existingServers := make(map[string]model.A2AServer)
		for _, server := range oldCfg.Servers {
			existingServers[server.ID] = server
		}
		for _, server := range cfg.Servers {
			existingServers[server.ID] = server
		}
		cfg.Servers = make([]model.A2AServer, 0, len(existingServers))
		for _, server := range existingServers {
			cfg.Servers = append(cfg.Servers, server)
		}
	}

	appConfig := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  "a2a",
		ConfigKey:   "default",
		ConfigValue: cfg,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return s.repo.SaveConfig(ctx, appConfig)
}
