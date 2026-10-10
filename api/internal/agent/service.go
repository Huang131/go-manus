// Package agent 提供 Planner/ReAct 所需的运行时能力装配。
package agent

import (
	"context"
	"sync"

	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
	"github.com/Huang131/go-manus/api/internal/search"
	"github.com/Huang131/go-manus/api/internal/settings"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

// AgentService 管理 Agent 运行时配置和工具快照。
// Run 执行边界负责请求、状态和生命周期；本类型只保留工具提供器及配置热更新。
type AgentService struct {
	mu            sync.RWMutex
	agentSettings settings.AgentSettings
	toolsProvider *toolspkg.ToolProvider
}

// NewAgentService 创建 Agent 运行时服务。
func NewAgentService(ctx context.Context, sandboxClient sandbox.Sandbox, browser sandbox.Browser, searchEngine search.SearchEngine, agentSettings settings.AgentSettings, mcpConfig *model.MCPConfig, a2aConfig *A2AConfig) *AgentService {
	return &AgentService{
		agentSettings: agentSettings,
		toolsProvider: toolspkg.NewToolProvider(ctx, sandboxClient, browser, searchEngine, mcpConfig, a2aConfig),
	}
}

// Shutdown 释放 MCP/A2A 等外部工具资源。
func (s *AgentService) Shutdown() {
	if s == nil {
		return
	}
	logger.Info("Agent 服务关闭中...")
	if s.toolsProvider != nil {
		s.toolsProvider.Cleanup()
	}
	logger.Info("Agent 服务已关闭")
}

// ReloadAgentSettings 原子替换后续 Run 使用的已验证配置。
func (s *AgentService) ReloadAgentSettings(next settings.AgentSettings) error {
	if err := next.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	s.agentSettings = next
	s.mu.Unlock()
	return nil
}

// ReloadMCPConfig 重建 MCP 客户端，确保配置接口保存后立即生效。
func (s *AgentService) ReloadMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	if s == nil || s.toolsProvider == nil {
		return nil
	}
	return s.toolsProvider.ReloadMCPConfig(ctx, cfg)
}

// ReloadA2AConfig 重建 A2A 客户端，确保配置接口保存后立即生效。
func (s *AgentService) ReloadA2AConfig(ctx context.Context, cfg *A2AConfig) error {
	if s == nil || s.toolsProvider == nil {
		return nil
	}
	return s.toolsProvider.ReloadA2AConfig(ctx, cfg)
}

// AcquireTools 为 Run 执行边界提供当前工具快照。
func (s *AgentService) AcquireTools(agentSettings settings.AgentSettings) *toolspkg.ToolSet {
	if s == nil || s.toolsProvider == nil {
		return nil
	}
	return s.toolsProvider.Acquire(agentSettings.MaxSearchResults)
}
