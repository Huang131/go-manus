package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/mcp"
	"github.com/Huang131/go-manus/api/internal/model"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// MCPTool MCP 工具 (Model Context Protocol)
type MCPTool struct {
	mu         sync.RWMutex
	manager    mcpToolManager
	newManager func(*model.MCPConfig) mcpToolManager
	functions  map[string]mcpFunction
}

// mcpToolManager is the narrow lifecycle and routing contract MCPTool needs.
// Keeping it local makes the tool contract testable without starting a process.
type mcpToolManager interface {
	Initialize(context.Context) error
	GetClient(string) (mcp.MCPClient, bool)
	ListAllTools(context.Context) (map[string][]mcp.MCPToolInfo, error)
	Close() error
}

type mcpFunction struct {
	serverName string
	toolName   string
	tool       mcp.MCPToolInfo
}

// NewMCPTool 创建 MCP 工具
func NewMCPTool() *MCPTool {
	return &MCPTool{
		newManager: func(cfg *model.MCPConfig) mcpToolManager {
			return mcp.NewMCPClientManager(cfg)
		},
		functions: make(map[string]mcpFunction),
	}
}

// Name 返回工具名称
func (t *MCPTool) Name() string {
	return ToolNameMCP
}

// Description 返回工具描述
func (t *MCPTool) Description() string {
	return "调用 MCP 服务器提供的工具。"
}

// Parameters 返回工具参数定义
func (t *MCPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// ReadOnly MCP 工具可能映射到任意远端动作，保守视为有副作用。
func (t *MCPTool) ReadOnly() bool {
	return false
}

// Invoke 调用工具
func (t *MCPTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolError("MCP 工具必须通过发现的 function name 调用"), nil
}

// Initialize 初始化 MCP 工具
func (t *MCPTool) Initialize(ctx context.Context, cfg *model.MCPConfig) error {
	if cfg == nil {
		return nil
	}

	manager := t.newManager(cfg)
	if manager == nil {
		return fmt.Errorf("create MCP manager: nil manager")
	}
	if err := manager.Initialize(ctx); err != nil {
		_ = manager.Close()
		return fmt.Errorf("initialize MCP manager: %w", err)
	}

	allTools, err := manager.ListAllTools(ctx)
	if err != nil {
		_ = manager.Close()
		return fmt.Errorf("discover MCP tools: %w", err)
	}
	functions, err := buildMCPFunctionIndex(allTools)
	if err != nil {
		_ = manager.Close()
		return err
	}

	t.mu.Lock()
	if t.manager != nil {
		t.mu.Unlock()
		_ = manager.Close()
		return fmt.Errorf("MCP tool is already initialized")
	}
	t.manager = manager
	t.functions = functions
	t.mu.Unlock()

	logger.Info("MCP 工具加载成功",
		logger.Int("servers", len(allTools)),
		logger.Int("tools", len(functions)))

	return nil
}

func buildMCPFunctionIndex(allTools map[string][]mcp.MCPToolInfo) (map[string]mcpFunction, error) {
	functions := make(map[string]mcpFunction)
	for serverName, tools := range allTools {
		for _, tool := range tools {
			functionName := MCPFunctionPrefix + serverName + "_" + tool.Name
			if previous, exists := functions[functionName]; exists {
				return nil, fmt.Errorf("duplicate MCP function %q from %s/%s and %s/%s", functionName, previous.serverName, previous.toolName, serverName, tool.Name)
			}
			functions[functionName] = mcpFunction{
				serverName: serverName,
				toolName:   tool.Name,
				tool:       tool,
			}
		}
	}
	return functions, nil
}

// GetTools 将发现的 MCP 工具转换为注册表可调用的动态 function schema。
func (t *MCPTool) GetTools() []map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]map[string]interface{}, 0)

	for functionName, function := range t.functions {
		description := "[" + function.serverName + "] " + function.tool.Description
		if description == "["+function.serverName+"] " {
			description = "[" + function.serverName + "] " + function.toolName
		}
		inputSchema := function.tool.InputSchema
		if inputSchema == nil {
			inputSchema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		result = append(result, map[string]interface{}{
			"type":        llmcore.ToolTypeFunction,
			"name":        functionName,
			"description": description,
			"parameters":  inputSchema,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i]["name"].(string) < result[j]["name"].(string)
	})

	return result
}

// InvokeWithName 根据注册的动态 function name 调用对应 MCP server/tool。
func (t *MCPTool) InvokeWithName(functionName string, ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	t.mu.RLock()
	function, exists := t.functions[functionName]
	manager := t.manager
	t.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unknown MCP function: %s", functionName)
	}
	if manager == nil {
		return nil, fmt.Errorf("MCP manager is not initialized")
	}
	client, ok := manager.GetClient(function.serverName)
	if !ok {
		return nil, fmt.Errorf("MCP server is unavailable: %s", function.serverName)
	}
	logger.InfoContext(ctx, "调用 MCP 工具",
		logger.String("server", function.serverName),
		logger.String("tool", function.toolName))
	result, err := client.CallTool(ctx, function.toolName, params)
	if err != nil {
		return nil, fmt.Errorf("call MCP tool %s: %w", functionName, err)
	}
	if result == nil {
		return nil, fmt.Errorf("MCP tool %s returned an empty result", functionName)
	}
	if result.IsError {
		return model.NewToolError(mcpResultText(result)), nil
	}
	return model.NewToolResultWithMessage(mcpResultText(result), map[string]interface{}{
		"server": function.serverName,
		"tool":   function.toolName,
		"result": result,
	}), nil
}

func mcpResultText(result *mcp.MCPToolResult) string {
	if result == nil {
		return ""
	}
	var text string
	for _, content := range result.Content {
		if content.Type == llmcore.ContentTypeText {
			text += content.Text + "\n"
		}
	}
	return text
}

// Cleanup 清理 MCP 资源
func (t *MCPTool) Cleanup() error {
	t.mu.Lock()
	manager := t.manager
	t.manager = nil
	t.functions = make(map[string]mcpFunction)
	t.mu.Unlock()

	if manager != nil {
		return manager.Close()
	}

	return nil
}
