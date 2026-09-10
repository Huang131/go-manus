package agent

import (
	"context"
	"strings"
	"sync"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// MCPTool MCP 工具 (Model Context Protocol)
type MCPTool struct {
	mu      sync.RWMutex
	config  *MCPConfig
	manager *external.MCPClientManager
	tools   map[string]map[string]external.MCPToolInfo // serverName -> toolName -> toolInfo
}

// NewMCPTool 创建 MCP 工具
func NewMCPTool() *MCPTool {
	return &MCPTool{
		tools: make(map[string]map[string]external.MCPToolInfo),
	}
}

// Name 返回工具名称
func (t *MCPTool) Name() string {
	return ToolNameMCP
}

// Description 返回工具描述
func (t *MCPTool) Description() string {
	return "用于调用 MCP (Model Context Protocol) 服务器提供的工具。"
}

// Parameters 返回工具参数定义
func (t *MCPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"server": map[string]interface{}{
				"type":        "string",
				"description": "MCP 服务器名称",
			},
			"tool": map[string]interface{}{
				"type":        "string",
				"description": "工具名称",
			},
			"params": map[string]interface{}{
				"type":        "object",
				"description": "工具参数",
			},
		},
		"required": []string{"server", "tool"},
	}
}

// ReadOnly MCP 工具可能映射到任意远端动作，保守视为有副作用。
func (t *MCPTool) ReadOnly() bool {
	return false
}

// Invoke 调用工具
func (t *MCPTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	serverName, toolName, paramsRaw, validationErr := parseMCPInvokeParams(params)
	if validationErr != "" {
		return model.NewToolError(validationErr), nil
	}

	t.mu.RLock()
	manager := t.manager
	t.mu.RUnlock()

	if manager == nil {
		return model.NewToolError("MCP manager not initialized"), nil
	}

	client, ok := manager.GetClient(serverName)
	if !ok {
		return model.NewToolError("MCP server not found: " + serverName), nil
	}

	logger.InfoContext(ctx, "调用 MCP 工具",
		logger.String("server", serverName),
		logger.String("tool", toolName))

	// 调用 MCP 工具
	result, err := client.CallTool(ctx, toolName, paramsRaw)
	if err != nil {
		logger.ErrorContext(ctx, "MCP 工具调用失败",
			logger.String("server", serverName),
			logger.String("tool", toolName),
			logger.Err(err))
		return model.NewToolError(err.Error()), nil
	}

	// 处理结果
	if result.IsError {
		var errorMsg string
		for _, content := range result.Content {
			errorMsg += content.Text + "\n"
		}
		return model.NewToolError(errorMsg), nil
	}

	// 构建成功结果
	var message string
	for _, content := range result.Content {
		if content.Type == llmcore.ContentTypeText {
			message += content.Text + "\n"
		}
	}

	return model.NewToolResultWithMessage(message, map[string]interface{}{
		"server": serverName,
		"tool":   toolName,
		"result": result,
	}), nil
}

// parseMCPInvokeParams 在访问远端 MCP manager 前校验模型生成的动态参数。
func parseMCPInvokeParams(params map[string]interface{}) (string, string, map[string]interface{}, string) {
	serverName, ok := params["server"].(string)
	if !ok || strings.TrimSpace(serverName) == "" {
		return "", "", nil, "server must be a non-empty string"
	}
	toolName, ok := params["tool"].(string)
	if !ok || strings.TrimSpace(toolName) == "" {
		return "", "", nil, "tool must be a non-empty string"
	}
	paramsValue, exists := params["params"]
	if !exists || paramsValue == nil {
		return strings.TrimSpace(serverName), strings.TrimSpace(toolName), map[string]interface{}{}, ""
	}
	paramsRaw, ok := paramsValue.(map[string]interface{})
	if !ok {
		return "", "", nil, "params must be an object"
	}
	return strings.TrimSpace(serverName), strings.TrimSpace(toolName), paramsRaw, ""
}

// Initialize 初始化 MCP 工具
func (t *MCPTool) Initialize(ctx context.Context, cfg *MCPConfig) error {
	if cfg == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.config = cfg

	// 将 agent.MCPConfig 转换为 external.MCPConfig
	externalConfig := &external.MCPConfig{
		Timeout: cfg.Timeout,
	}
	for _, server := range cfg.Servers {
		externalConfig.Servers = append(externalConfig.Servers, external.MCPConfigServer{
			Name:    server.Name,
			Command: server.Command,
			Args:    server.Args,
			Env:     server.Env,
		})
	}

	// 创建 MCP 客户端管理器
	t.manager = external.NewMCPClientManager(externalConfig)

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
			t.tools = make(map[string]map[string]external.MCPToolInfo)
			for serverName, tools := range allTools {
				t.tools[serverName] = make(map[string]external.MCPToolInfo)
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

// GetToolsForLLM 获取所有 MCP 工具的 schema 列表
//
// 阶段 1d 改造点：返回 []llmcore.ToolSpec 而非 []map，与 ToolRegistry.GetToolsForLLM 协议统一。
func (t *MCPTool) GetToolsForLLM() []llmcore.ToolSpec {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]llmcore.ToolSpec, 0)

	for serverName, tools := range t.tools {
		for _, tool := range tools {
			// 生成工具名称：mcp_{serverName}_{toolName}
			toolName := MCPFunctionPrefix + serverName + "_" + tool.Name

			// 描述前缀
			description := "[" + serverName + "] " + tool.Description
			if description == "["+serverName+"] " {
				description = "[" + serverName + "] " + tool.Name
			}

			// 输入 Schema
			inputSchema := tool.InputSchema
			if inputSchema == nil {
				inputSchema = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}

			result = append(result, llmcore.ToolSpec{
				Type: llmcore.ToolTypeFunction,
				Function: llmcore.ToolSpecFunction{
					Name:        toolName,
					Description: description,
					Parameters:  inputSchema,
				},
			})
		}
	}

	return result
}

// HasTool 检查工具是否存在
func (t *MCPTool) HasTool(toolName string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, tools := range t.tools {
		for _, tool := range tools {
			// 支持两种格式的检查：mcp_{server}_{name} 或 {name}
			expectedName := tool.Name
			fullName := MCPFunctionPrefix + t.getServerNamePrefix() + "_" + tool.Name

			if toolName == expectedName || toolName == fullName {
				return true
			}
		}
	}
	return false
}

// getServerNamePrefix 获取服务器名称前缀
func (t *MCPTool) getServerNamePrefix() string {
	// 返回第一个服务器名称作为前缀
	for serverName := range t.tools {
		return serverName
	}
	return ""
}

// Cleanup 清理 MCP 资源
func (t *MCPTool) Cleanup() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.manager != nil {
		t.manager.Close()
	}

	t.tools = make(map[string]map[string]external.MCPToolInfo)
	t.config = nil

	return nil
}
