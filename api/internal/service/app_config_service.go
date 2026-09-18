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
	// mu 串行化配置合并写入：MCP/A2A 的更新都是读-改-写，
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

// getConfig 读取指定类型的默认配置并反序列化为具体结构。ConfigValue 为原始 JSON，
// 泛型统一处理 nil 判断与反序列化，消除各 Get 方法的样板。
func getConfig[T any](s *DefaultAppConfigService, ctx context.Context, configType model.AppConfigType) (*T, error) {
	cfg, err := s.repo.GetConfig(ctx, configType, model.AppConfigKeyDefault)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // 配置不存在，返回 nil
	}
	var v T
	if err := sonic.Unmarshal(cfg.ConfigValue, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetAgentConfig 获取 Agent 配置
func (s *DefaultAppConfigService) GetAgentConfig(ctx context.Context) (*model.AgentConfig, error) {
	return getConfig[model.AgentConfig](s, ctx, model.AppConfigTypeAgent)
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
	return getConfig[model.MCPConfig](s, ctx, model.AppConfigTypeMCP)
}

// UpdateMCPConfig 更新 MCP 配置
func (s *DefaultAppConfigService) UpdateMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mergeAndSaveMCPConfig(ctx, cfg)
}

// mergeAndSaveMCPConfig 合并服务器配置并写入
func (s *DefaultAppConfigService) mergeAndSaveMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	if cfg == nil {
		return apperr.BadRequest("MCP配置不能为空")
	}
	oldCfg, err := s.repo.GetConfig(ctx, model.AppConfigTypeMCP, model.AppConfigKeyDefault)
	if err != nil {
		return fmt.Errorf("读取现有 MCP 配置失败: %w", err)
	}
	if oldCfg != nil {
		var oldMCP model.MCPConfig
		if err := sonic.Unmarshal(oldCfg.ConfigValue, &oldMCP); err != nil {
			return fmt.Errorf("解析现有 MCP 配置失败: %w", err)
		}
		cfg.Servers = mergeMCPServers(oldMCP.Servers, cfg.Servers)
	}

	appConfig, err := newAppConfig(model.AppConfigTypeMCP, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// mergeMCPServers 按 ServerName 合并新旧服务器列表。
// 新传入的服务器覆盖同名的旧服务器；旧配置原有的顺序保持不变，新传入的新增项追加在末尾。
func mergeMCPServers(oldServers, incoming []model.MCPServer) []model.MCPServer {
	if len(oldServers) == 0 {
		return incoming
	}
	// 覆盖合并：后写的值生效（incoming 覆盖 old）
	serverMap := make(map[string]model.MCPServer, len(oldServers)+len(incoming))
	for _, server := range oldServers {
		serverMap[server.ServerName] = server
	}
	for _, server := range incoming {
		serverMap[server.ServerName] = server
	}
	// 顺序重建：先遍历旧配置（保持稳定顺序），再追加 incoming 中新增的服务器。
	merged := make([]model.MCPServer, 0, len(serverMap))
	seen := make(map[string]bool, len(serverMap))
	for _, server := range oldServers {
		if !seen[server.ServerName] {
			seen[server.ServerName] = true
			merged = append(merged, serverMap[server.ServerName])
		}
	}
	for _, server := range incoming {
		if !seen[server.ServerName] {
			seen[server.ServerName] = true
			merged = append(merged, serverMap[server.ServerName])
		}
	}
	return merged
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
			return s.mergeAndSaveMCPConfig(ctx, cfg)
		}
	}
	return apperr.NotFound("MCP服务器不存在")
}

// GetA2AConfig 获取 A2A 配置
func (s *DefaultAppConfigService) GetA2AConfig(ctx context.Context) (*model.A2AConfig, error) {
	return getConfig[model.A2AConfig](s, ctx, model.AppConfigTypeA2A)
}

// UpdateA2AConfig 更新 A2A 配置
func (s *DefaultAppConfigService) UpdateA2AConfig(ctx context.Context, cfg *model.A2AConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mergeAndSaveA2AConfig(ctx, cfg)
}

// mergeAndSaveA2AConfig 合并服务器配置并写入，不加锁（由调用方负责加锁）。
func (s *DefaultAppConfigService) mergeAndSaveA2AConfig(ctx context.Context, cfg *model.A2AConfig) error {
	if cfg == nil {
		return apperr.BadRequest("A2A配置不能为空")
	}
	oldCfg, err := s.repo.GetConfig(ctx, model.AppConfigTypeA2A, model.AppConfigKeyDefault)
	if err != nil {
		return fmt.Errorf("读取现有 A2A 配置失败: %w", err)
	}
	if oldCfg != nil {
		var oldA2A model.A2AConfig
		if err := sonic.Unmarshal(oldCfg.ConfigValue, &oldA2A); err != nil {
			return fmt.Errorf("解析现有 A2A 配置失败: %w", err)
		}
		cfg.Servers = mergeA2AServers(oldA2A.Servers, cfg.Servers)
	}

	appConfig, err := newAppConfig(model.AppConfigTypeA2A, model.AppConfigKeyDefault, cfg)
	if err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, appConfig)
}

// mergeA2AServers 按 ID 合并新旧服务器列表。
// 新传入的服务器覆盖同 ID 的旧服务器；旧配置原有的顺序保持不变，新传入的新增项追加在末尾。
func mergeA2AServers(oldServers, incoming []model.A2AServer) []model.A2AServer {
	if len(oldServers) == 0 {
		return incoming
	}
	serverMap := make(map[string]model.A2AServer, len(oldServers)+len(incoming))
	for _, server := range oldServers {
		serverMap[server.ID] = server
	}
	for _, server := range incoming {
		serverMap[server.ID] = server
	}
	merged := make([]model.A2AServer, 0, len(serverMap))
	seen := make(map[string]bool, len(serverMap))
	for _, server := range oldServers {
		if !seen[server.ID] {
			seen[server.ID] = true
			merged = append(merged, serverMap[server.ID])
		}
	}
	for _, server := range incoming {
		if !seen[server.ID] {
			seen[server.ID] = true
			merged = append(merged, serverMap[server.ID])
		}
	}
	return merged
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
