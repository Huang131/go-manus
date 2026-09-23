package tools

import (
	"context"
	"sync"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
	"github.com/Huang131/go-manus/api/internal/search"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// ToolProvider 负责工具组装与 MCP/A2A 热重载生命周期管理。
//
// 原 AgentService 既渲染消息、管理任务，又负责拼装工具和处理 MCP/A2A 客户端的
// 初始化/热重载/清理，职责过宽。把工具相关的状态与生命周期收敛到本协作者后，
// 服务的其他部分只需调用 Tools() 即可得到当前可用的工具集合。
type ToolProvider struct {
	mu           sync.RWMutex
	sandbox      sandbox.Sandbox
	browser      sandbox.Browser
	searchEngine search.SearchEngine
	mcpTool      *MCPTool
	a2aTool      *A2ATool
	retiredMCP   []*MCPTool
	retiredA2A   []*A2ATool
}

// NewToolProvider 构造工具提供器，并按需初始化 MCP/A2A 客户端。
// 只接收工具真正依赖的三个能力（沙箱、浏览器、搜索），不引入 agent 层的 Capabilities，
// 保持 agent → tools 单向依赖。
func NewToolProvider(
	ctx context.Context,
	sandbox sandbox.Sandbox,
	browser sandbox.Browser,
	searchEngine search.SearchEngine,
	mcpConfig *model.MCPConfig,
	a2aConfig *A2AConfig,
) *ToolProvider {
	p := &ToolProvider{
		sandbox:      sandbox,
		browser:      browser,
		searchEngine: searchEngine,
	}

	if mcpConfig != nil && len(mcpConfig.Servers) > 0 {
		p.mcpTool = NewMCPTool()
		if err := p.mcpTool.Initialize(ctx, mcpConfig); err != nil {
			logger.Warn("MCP 工具初始化失败，继续启动 Agent 服务", logger.Err(err))
		}
	}
	if a2aConfig != nil && len(a2aConfig.Agents) > 0 {
		p.a2aTool = NewA2ATool()
		if err := p.a2aTool.Initialize(ctx, a2aConfig); err != nil {
			logger.Warn("A2A 工具初始化失败，继续启动 Agent 服务", logger.Err(err))
		}
	}

	return p
}

// Tools 组装当前可用的工具集合。
func (p *ToolProvider) Tools(searchLimit int) []Tool {
	p.mu.RLock()
	sandbox := p.sandbox
	browser := p.browser
	searchEngine := p.searchEngine
	mcpTool := p.mcpTool
	a2aTool := p.a2aTool
	p.mu.RUnlock()

	tools := make([]Tool, 0)

	// 1. Shell 工具 (依赖 sandbox)
	if sandbox != nil {
		tools = append(tools, NewShellTool(sandbox))
	}

	// 2. File 工具 (依赖 sandbox)
	if sandbox != nil {
		tools = append(tools, NewFileTool(sandbox))
	}

	// 3. Browser 工具 (依赖 browser)
	if browser != nil {
		tools = append(tools, NewBrowserTool(browser))
	}

	// 4. Search 工具 (依赖 searchEngine)
	if searchEngine != nil {
		tools = append(tools, NewSearchTool(searchEngine, searchLimit))
	}

	// 5. Message 工具 (无需外部依赖)
	tools = append(tools, NewMessageTool())

	// 6. MCP 工具 (可选)
	if mcpTool != nil {
		tools = append(tools, mcpTool)
	}

	// 7. A2A 工具 (可选)
	if a2aTool != nil {
		tools = append(tools, a2aTool)
	}

	logger.Info("注册工具列表",
		logger.Int("count", len(tools)),
		logger.Any("tools", toolNames(tools)))

	return tools
}

// ReloadMCPConfig 重建 MCP 客户端，确保配置接口保存后立即生效。
func (p *ToolProvider) ReloadMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	if cfg == nil || len(cfg.Servers) == 0 {
		p.mu.Lock()
		oldTool := p.mcpTool
		p.mcpTool = nil
		if oldTool != nil {
			p.retiredMCP = append(p.retiredMCP, oldTool)
		}
		p.mu.Unlock()
		return nil
	}
	newTool := NewMCPTool()
	if err := newTool.Initialize(ctx, cfg); err != nil {
		return err
	}
	p.mu.Lock()
	oldTool := p.mcpTool
	p.mcpTool = newTool
	if oldTool != nil {
		p.retiredMCP = append(p.retiredMCP, oldTool)
	}
	p.mu.Unlock()
	return nil
}

// ReloadA2AConfig 重建 A2A 客户端，确保配置接口保存后立即生效。
func (p *ToolProvider) ReloadA2AConfig(ctx context.Context, cfg *A2AConfig) error {
	if cfg == nil || len(cfg.Agents) == 0 {
		p.mu.Lock()
		oldTool := p.a2aTool
		p.a2aTool = nil
		if oldTool != nil {
			p.retiredA2A = append(p.retiredA2A, oldTool)
		}
		p.mu.Unlock()
		return nil
	}
	newTool := NewA2ATool()
	if err := newTool.Initialize(ctx, cfg); err != nil {
		return err
	}
	p.mu.Lock()
	oldTool := p.a2aTool
	p.a2aTool = newTool
	if oldTool != nil {
		p.retiredA2A = append(p.retiredA2A, oldTool)
	}
	p.mu.Unlock()
	return nil
}

// Cleanup 释放工具持有的外部资源（MCP 子进程、A2A 连接）。
// 配置热重载产生的旧工具也需要一并释放。
func (p *ToolProvider) Cleanup() {
	p.mu.Lock()
	mcpTools := append([]*MCPTool{p.mcpTool}, p.retiredMCP...)
	a2aTools := append([]*A2ATool{p.a2aTool}, p.retiredA2A...)
	p.retiredMCP = nil
	p.retiredA2A = nil
	p.mu.Unlock()

	for _, tool := range mcpTools {
		if tool != nil {
			if err := tool.Cleanup(); err != nil {
				logger.Warn("清理 MCP 工具失败", logger.Err(err))
			}
		}
	}
	for _, tool := range a2aTools {
		if tool != nil {
			if err := tool.Cleanup(); err != nil {
				logger.Warn("清理 A2A 工具失败", logger.Err(err))
			}
		}
	}
}

// toolNames 提取工具名称列表，用于日志展示。
func toolNames(tools []Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}
