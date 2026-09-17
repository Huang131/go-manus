package service

import (
	"context"
	"fmt"
	"sync"
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
	UpdateMCPServerEnabled(ctx context.Context, serverName string, enabled bool) error
	GetA2AConfig(ctx context.Context) (*model.A2AConfig, error)
	UpdateA2AConfig(ctx context.Context, cfg *model.A2AConfig) error
	DeleteA2AServer(ctx context.Context, id string) error
	UpdateA2AServerEnabled(ctx context.Context, id string, enabled bool) error
}

// DefaultAppConfigService 应用配置服务默认实现
type DefaultAppConfigService struct {
	repo repository.AppConfigRepository
	// mu 串行化配置合并写入：MCP/A2A/LLM 的更新都是读-改-写，
	// 并发写会互相覆盖丢条目（单进程内互斥即可覆盖全部入口）
	mu sync.Mutex
}

// NewAppConfigService 创建应用配置服务
func NewAppConfigService(repo repository.AppConfigRepository) AppConfigService {
	return &DefaultAppConfigService{repo: repo}
}

// newAppConfig 构造一条待保存的配置：将具体配置序列化为原始 JSON，
// 并集中填好 ID 与创建/更新时间，消除各 Update 方法里的样板。
func newAppConfig(configType model.AppConfigType, configKey string, value any) (*model.AppConfig, error) {
	raw, err := sonic.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode config value: %w", err)
	}
	now := time.Now()
	return &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  configType,
		ConfigKey:   configKey,
		ConfigValue: raw,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetLLMConfig 获取 LLM 配置
func (s *DefaultAppConfigService) GetLLMConfig(ctx context.Context) (*model.LLMConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, model.AppConfigTypeLLM, model.AppConfigKeyDefault)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}

	var llmConfig model.LLMConfig
	if err := sonic.Unmarshal(cfg.ConfigValue, &llmConfig); err != nil {
		return nil, err
	}
	return &llmConfig, nil
}

// UpdateLLMConfig 更新 LLM 配置 (如果 api_key 为空则保留旧值)
func (s *DefaultAppConfigService) UpdateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateLLMConfig(ctx, cfg)
}

func (s *DefaultAppConfigService) updateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	if cfg == nil {
		return apperr.BadRequest("LLM配置不能为空")
	}
	// 如果 api_key 为空，保留旧值
	if cfg.APIKey == "" {
		oldCfg, err := s.GetLLMConfig(ctx)
		if err != nil {
			return fmt.Errorf("读取现有 LLM 配置失败: %w", err)
		}
		if oldCfg != nil {
			cfg.APIKey = oldCfg.APIKey
		}
	}

	appConfig, err := newAppConfig(model.AppConfigTypeLLM, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// GetAgentConfig 获取 Agent 配置
func (s *DefaultAppConfigService) GetAgentConfig(ctx context.Context) (*model.AgentConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, model.AppConfigTypeAgent, model.AppConfigKeyDefault)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}
	var agentConfig model.AgentConfig
	if err := sonic.Unmarshal(cfg.ConfigValue, &agentConfig); err != nil {
		return nil, err
	}
	return &agentConfig, nil
}

// UpdateAgentConfig 更新 Agent 配置
func (s *DefaultAppConfigService) UpdateAgentConfig(ctx context.Context, cfg *model.AgentConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveAgentConfig(ctx, cfg)
}

func (s *DefaultAppConfigService) saveAgentConfig(ctx context.Context, cfg *model.AgentConfig) error {
	appConfig, err := newAppConfig(model.AppConfigTypeAgent, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// GetMCPConfig 获取 MCP 配置
func (s *DefaultAppConfigService) GetMCPConfig(ctx context.Context) (*model.MCPConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, model.AppConfigTypeMCP, model.AppConfigKeyDefault)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}
	var mcpConfig model.MCPConfig
	if err := sonic.Unmarshal(cfg.ConfigValue, &mcpConfig); err != nil {
		return nil, err
	}
	return &mcpConfig, nil
}

// UpdateMCPConfig 更新 MCP 配置 (合并服务器列表)
func (s *DefaultAppConfigService) UpdateMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateMCPConfig(ctx, cfg)
}

func (s *DefaultAppConfigService) updateMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	if cfg == nil {
		return apperr.BadRequest("MCP配置不能为空")
	}
	// 获取现有配置
	oldCfg, err := s.GetMCPConfig(ctx)
	if err != nil {
		return fmt.Errorf("读取现有 MCP 配置失败: %w", err)
	}
	if oldCfg != nil {
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

	appConfig, err := newAppConfig(model.AppConfigTypeMCP, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// DeleteMCPServer 删除 MCP 服务器
func (s *DefaultAppConfigService) DeleteMCPServer(ctx context.Context, serverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteMCPServer(ctx, serverName)
}

func (s *DefaultAppConfigService) deleteMCPServer(ctx context.Context, serverName string) error {
	cfg, err := s.GetMCPConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil {
		return apperr.NotFound("MCP服务器不存在")
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
	appConfig, err := newAppConfig(model.AppConfigTypeMCP, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// UpdateMCPServerEnabled 更新单个 MCP 服务状态，保持其他服务配置不变。
func (s *DefaultAppConfigService) UpdateMCPServerEnabled(ctx context.Context, serverName string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateMCPServerEnabled(ctx, serverName, enabled)
}

func (s *DefaultAppConfigService) updateMCPServerEnabled(ctx context.Context, serverName string, enabled bool) error {
	cfg, err := s.GetMCPConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil {
		return apperr.NotFound("MCP服务器不存在")
	}
	for i := range cfg.Servers {
		if cfg.Servers[i].ServerName == serverName {
			cfg.Servers[i].Enabled = enabled
			return s.updateMCPConfig(ctx, cfg)
		}
	}
	return apperr.NotFound("MCP服务器不存在")
}

// GetA2AConfig 获取 A2A 配置
func (s *DefaultAppConfigService) GetA2AConfig(ctx context.Context) (*model.A2AConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, model.AppConfigTypeA2A, model.AppConfigKeyDefault)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}
	var a2aConfig model.A2AConfig
	if err := sonic.Unmarshal(cfg.ConfigValue, &a2aConfig); err != nil {
		return nil, err
	}
	return &a2aConfig, nil
}

// UpdateA2AConfig 更新 A2A 配置
func (s *DefaultAppConfigService) UpdateA2AConfig(ctx context.Context, cfg *model.A2AConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateA2AConfig(ctx, cfg)
}

func (s *DefaultAppConfigService) updateA2AConfig(ctx context.Context, cfg *model.A2AConfig) error {
	if cfg == nil {
		return apperr.BadRequest("A2A配置不能为空")
	}
	// 获取现有配置
	oldCfg, err := s.GetA2AConfig(ctx)
	if err != nil {
		return fmt.Errorf("读取现有 A2A 配置失败: %w", err)
	}
	if oldCfg != nil {
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

	appConfig, err := newAppConfig(model.AppConfigTypeA2A, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// DeleteA2AServer 删除一个 A2A 服务配置。
func (s *DefaultAppConfigService) DeleteA2AServer(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteA2AServer(ctx, id)
}

func (s *DefaultAppConfigService) deleteA2AServer(ctx context.Context, id string) error {
	cfg, err := s.GetA2AConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil {
		return apperr.NotFound("A2A服务器不存在")
	}
	filtered := make([]model.A2AServer, 0, len(cfg.Servers))
	found := false
	for _, server := range cfg.Servers {
		if server.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, server)
	}
	if !found {
		return apperr.NotFound("A2A服务器不存在")
	}
	cfg.Servers = filtered
	return s.saveA2AConfig(ctx, cfg)
}

// UpdateA2AServerEnabled 更新单个 A2A 服务状态。
func (s *DefaultAppConfigService) UpdateA2AServerEnabled(ctx context.Context, id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateA2AServerEnabled(ctx, id, enabled)
}

func (s *DefaultAppConfigService) updateA2AServerEnabled(ctx context.Context, id string, enabled bool) error {
	cfg, err := s.GetA2AConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil {
		return apperr.NotFound("A2A服务器不存在")
	}
	for i := range cfg.Servers {
		if cfg.Servers[i].ID == id {
			cfg.Servers[i].Enabled = enabled
			return s.saveA2AConfig(ctx, cfg)
		}
	}
	return apperr.NotFound("A2A服务器不存在")
}

func (s *DefaultAppConfigService) saveA2AConfig(ctx context.Context, cfg *model.A2AConfig) error {
	appConfig, err := newAppConfig(model.AppConfigTypeA2A, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}
