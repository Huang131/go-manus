package tools

import (
	"context"
	"sort"
	"sync"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/mcp"
	"github.com/Huang131/go-manus/api/internal/model"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// MCPTool MCP 工具 (Model Context Protocol)
type MCPTool struct {
	mu      sync.RWMutex
	manager *mcp.MCPClientManager
	tools   map[string]map[string]mcp.MCPToolInfo // serverName -> toolName -> toolInfo
}

// NewMCPTool 创建 MCP 工具
func NewMCPTool() *MCPTool {
	return &MCPTool{
		tools: make(map[string]map[string]mcp.MCPToolInfo),
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

	t.mu.Lock()
	defer t.mu.Unlock()

	// agent.MCPConfig 已经是 model.MCPConfig，直接传递
	t.manager = mcp.NewMCPClientManager(cfg)

	// 初始化所有 MCP 客户端
	if err := t.manager.Initialize(ctx); err != nil {
		logger.Warn("MCP 客户端管理器初始化失败", logger.Err(err))
		// 不返回错误，继续运行
	}

	// 获取所有工具列表
	if t.manager != nil {
		allTools, err := t.manager.ListAllTools(ctx)
		if err != nil {
			logger.Warn("获取 MCP 工具列表失败", logger.Err(err))
		} else {
			// 转换为 map[string]map[string]MCPToolInfo
			t.tools = make(map[string]map[string]mcp.MCPToolInfo)
			for serverName, tools := range allTools {
				t.tools[serverName] = make(map[string]mcp.MCPToolInfo)
				for _, tool := range tools {
					t.tools[serverName][tool.Name] = tool
				}
			}
			// 统计工具数量
			total := 0
			for _, tools := range allTools {
				total += len(tools)
			}
			logger.Info("MCP 工具加载成功",
				logger.Int("servers", len(allTools)),
				logger.Int("tools", total))
		}
	}

	return nil
}

// GetTools 将发现的 MCP 工具转换为注册表可调用的动态 function schema。
func (t *MCPTool) GetTools() []map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]map[string]interface{}, 0)

	for serverName, tools := range t.tools {
		for _, tool := range tools {
			toolName := MCPFunctionPrefix + serverName + "_" + tool.Name
			description := "[" + serverName + "] " + tool.Description
			if description == "["+serverName+"] " {
				description = "[" + serverName + "] " + tool.Name
			}
			inputSchema := tool.InputSchema
			if inputSchema == nil {
				inputSchema = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}

			result = append(result, map[string]interface{}{
				"type":        llmcore.ToolTypeFunction,
				"name":        toolName,
				"description": description,
				"parameters":  inputSchema,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i]["name"].(string) < result[j]["name"].(string)
	})

	return result
}

// InvokeWithName 根据注册的动态 function name 调用对应 MCP server/tool。
func (t *MCPTool) InvokeWithName(functionName string, ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	t.mu.RLock()
	var serverName, toolName string
	for candidateServer, tools := range t.tools {
		for candidateName, tool := range tools {
			if functionName == MCPFunctionPrefix+candidateServer+"_"+tool.Name {
				serverName, toolName = candidateServer, candidateName
				break
			}
		}
		if serverName != "" {
			break
		}
	}
	manager := t.manager
	t.mu.RUnlock()
	if serverName == "" {
		return model.NewToolError("未知 MCP 工具: " + functionName), nil
	}
	if manager == nil {
		return model.NewToolError("MCP manager 未初始化"), nil
	}
	client, ok := manager.GetClient(serverName)
	if !ok {
		return model.NewToolError("MCP server 不存在: " + serverName), nil
	}
	logger.InfoContext(ctx, "调用 MCP 工具",
		logger.String("server", serverName),
		logger.String("tool", toolName))
	result, err := client.CallTool(ctx, toolName, params)
	if err != nil {
		return model.NewToolError(err.Error()), nil
	}
	if result == nil {
		return model.NewToolError("MCP 工具返回空结果"), nil
	}
	if result.IsError {
		return model.NewToolError(mcpResultText(result)), nil
	}
	return model.NewToolResultWithMessage(mcpResultText(result), map[string]interface{}{
		"server": serverName,
		"tool":   toolName,
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
	defer t.mu.Unlock()

	if t.manager != nil {
		t.manager.Close()
	}

	t.tools = make(map[string]map[string]mcp.MCPToolInfo)

	return nil
}
