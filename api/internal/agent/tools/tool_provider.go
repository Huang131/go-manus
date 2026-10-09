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
	current      *toolSetState
}

// ToolSet 是一次任务使用的工具快照。
// 快照持有 MCP/A2A 状态的引用，避免配置热重载时关闭仍被任务使用的连接。
type ToolSet struct {
	tools   []Tool
	release func()
	once    sync.Once
}

// Tools 返回工具快照的副本，调用方可以安全地构造自己的注册表。
func (s *ToolSet) Tools() []Tool {
	if s == nil {
		return nil
	}
	return append([]Tool(nil), s.tools...)
}

// Release 释放任务对工具状态的引用，重复调用不会重复释放。
func (s *ToolSet) Release() {
	if s == nil || s.release == nil {
		return
	}
	s.once.Do(s.release)
}

type toolSetState struct {
	mcp     *mcpResource
	a2a     *a2aResource
	refs    int
	retired bool
	cleaned bool
}

type mcpResource struct {
	tool   *MCPTool
	owners int
}

type a2aResource struct {
	tool   *A2ATool
	owners int
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

	var mcpTool *MCPTool
	if mcpConfig != nil && len(mcpConfig.Servers) > 0 {
		mcpTool = NewMCPTool()
		if err := mcpTool.Initialize(ctx, mcpConfig); err != nil {
			logger.Warn("MCP 工具初始化失败，继续启动 Agent 服务", logger.Err(err))
			_ = mcpTool.Cleanup()
			mcpTool = nil
		} else if !usableMCPTool(mcpTool) {
			logger.Warn("MCP 未发现可用工具，跳过注册")
			_ = mcpTool.Cleanup()
			mcpTool = nil
		}
	}
	var a2aTool *A2ATool
	if a2aConfig != nil && len(a2aConfig.Agents) > 0 {
		a2aTool = NewA2ATool()
		if err := a2aTool.Initialize(ctx, a2aConfig); err != nil {
			logger.Warn("A2A 工具初始化失败，继续启动 Agent 服务", logger.Err(err))
			_ = a2aTool.Cleanup()
			a2aTool = nil
		}
	}
	p.current = newToolSetState(mcpTool, a2aTool)

	return p
}

// Acquire 获取当前工具快照，并增加对应外部连接状态的引用计数。
func (p *ToolProvider) Acquire(searchLimit int) *ToolSet {
	p.mu.Lock()
	sandbox := p.sandbox
	browser := p.browser
	searchEngine := p.searchEngine
	state := p.current
	if state != nil {
		state.refs++
	}
	p.mu.Unlock()

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
	if state != nil && state.mcp != nil {
		tools = append(tools, state.mcp.tool)
	}

	// 7. A2A 工具 (可选)
	if state != nil && state.a2a != nil {
		tools = append(tools, state.a2a.tool)
	}

	logger.Info("注册工具列表",
		logger.Int("count", len(tools)),
		logger.Any("tools", toolNames(tools)))

	set := &ToolSet{tools: tools}
	if state != nil {
		set.release = func() { p.release(state) }
	}
	return set
}

// ReloadMCPConfig 重建 MCP 客户端，确保配置接口保存后立即生效。
func (p *ToolProvider) ReloadMCPConfig(ctx context.Context, cfg *model.MCPConfig) error {
	var newTool *MCPTool
	if cfg == nil || len(cfg.Servers) == 0 {
		return p.swapMCP(nil)
	}
	newTool = NewMCPTool()
	if err := newTool.Initialize(ctx, cfg); err != nil {
		_ = newTool.Cleanup()
		return err
	}
	if !usableMCPTool(newTool) {
		logger.Warn("MCP 未发现可用工具，保留当前工具配置")
		_ = newTool.Cleanup()
		return nil
	}
	return p.swapMCP(newTool)
}

func (p *ToolProvider) swapMCP(tool *MCPTool) error {
	p.mu.Lock()
	oldState := p.current
	if oldState == nil {
		oldState = &toolSetState{}
	}
	newState := newToolSetState(tool, nil)
	newState.a2a = oldState.a2a
	if newState.a2a != nil {
		newState.a2a.owners++
	}
	p.current = newState
	oldState.retired = true
	cleanState := p.releasableLocked(oldState)
	p.mu.Unlock()
	cleanupToolResources(cleanState)
	return nil
}

func usableMCPTool(tool *MCPTool) bool {
	return tool != nil && len(tool.GetTools()) > 0
}

// ReloadA2AConfig 重建 A2A 客户端，确保配置接口保存后立即生效。
func (p *ToolProvider) ReloadA2AConfig(ctx context.Context, cfg *A2AConfig) error {
	var newTool *A2ATool
	if cfg == nil || len(cfg.Agents) == 0 {
		return p.swapA2A(nil)
	}
	newTool = NewA2ATool()
	if err := newTool.Initialize(ctx, cfg); err != nil {
		_ = newTool.Cleanup()
		return err
	}
	return p.swapA2A(newTool)
}

func (p *ToolProvider) swapA2A(tool *A2ATool) error {
	p.mu.Lock()
	oldState := p.current
	if oldState == nil {
		oldState = &toolSetState{}
	}
	newState := newToolSetState(nil, tool)
	newState.mcp = oldState.mcp
	if newState.mcp != nil {
		newState.mcp.owners++
	}
	p.current = newState
	oldState.retired = true
	cleanState := p.releasableLocked(oldState)
	p.mu.Unlock()
	cleanupToolResources(cleanState)
	return nil
}

// Cleanup 释放工具持有的外部资源（MCP 子进程、A2A 连接）。
// 配置热重载产生的旧工具也需要一并释放。
func (p *ToolProvider) Cleanup() {
	p.mu.Lock()
	state := p.current
	p.current = nil
	if state != nil {
		state.retired = true
	}
	cleanState := p.releasableLocked(state)
	p.mu.Unlock()
	cleanupToolResources(cleanState)
}

func (p *ToolProvider) release(state *toolSetState) {
	p.mu.Lock()
	if state.refs > 0 {
		state.refs--
	}
	cleanState := p.releasableLocked(state)
	p.mu.Unlock()
	cleanupToolResources(cleanState)
}

type toolCleanup struct {
	mcp *MCPTool
	a2a *A2ATool
}

func (p *ToolProvider) releasableLocked(state *toolSetState) *toolCleanup {
	if state == nil || !state.retired || state.refs != 0 || state.cleaned {
		return nil
	}
	state.cleaned = true
	cleanup := &toolCleanup{}
	if state.mcp != nil {
		state.mcp.owners--
		if state.mcp.owners == 0 {
			cleanup.mcp = state.mcp.tool
		}
	}
	if state.a2a != nil {
		state.a2a.owners--
		if state.a2a.owners == 0 {
			cleanup.a2a = state.a2a.tool
		}
	}
	return cleanup
}

func cleanupToolResources(cleanup *toolCleanup) {
	if cleanup == nil {
		return
	}
	if cleanup.mcp != nil {
		if err := cleanup.mcp.Cleanup(); err != nil {
			logger.Warn("清理 MCP 工具失败", logger.Err(err))
		}
	}
	if cleanup.a2a != nil {
		if err := cleanup.a2a.Cleanup(); err != nil {
			logger.Warn("清理 A2A 工具失败", logger.Err(err))
		}
	}
}

func newToolSetState(mcpTool *MCPTool, a2aTool *A2ATool) *toolSetState {
	state := &toolSetState{}
	if mcpTool != nil {
		state.mcp = &mcpResource{tool: mcpTool, owners: 1}
	}
	if a2aTool != nil {
		state.a2a = &a2aResource{tool: a2aTool, owners: 1}
	}
	return state
}

// toolNames 提取工具名称列表，用于日志展示。
func toolNames(tools []Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}
