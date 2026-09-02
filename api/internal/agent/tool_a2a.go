package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// A2ATool A2A (Agent-to-Agent) 工具
// 参考 Python 版本的 A2ATool
type A2ATool struct {
	mu      sync.RWMutex
	manager *external.A2AClientManager
	config  *A2AConfig
}

// NewA2ATool 创建 A2A 工具
func NewA2ATool() *A2ATool {
	return &A2ATool{
		manager: external.NewA2AClientManager(),
	}
}

// Name 返回工具名称
func (t *A2ATool) Name() string {
	return "a2a"
}

// Description 返回工具描述
func (t *A2ATool) Description() string {
	return "用于与其他 Agent 进行通信。可以调用其他 Agent 处理特定任务。"
}

// Parameters 返回工具参数定义
func (t *A2ATool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: list_agents, call_agent",
				"enum":        []string{"list_agents", "call_agent"},
			},
			"agent_id": map[string]interface{}{
				"type":        "string",
				"description": "Agent 唯一标识 (调用 list_agents 获取)",
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "要分配给远程 Agent 完成的任务/需求描述",
			},
			"context": map[string]interface{}{
				"type":        "object",
				"description": "(可选) 任务上下文信息",
			},
		},
		"required": []string{"action"},
	}
}

// ReadOnly A2A 会触发远程 Agent 执行，保守视为有副作用。
func (t *A2ATool) ReadOnly() bool {
	return false
}

// Invoke 调用工具
// 参考 Python 版本的 call_remote_agent 工具
func (t *A2ATool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	action, _ := params["action"].(string)

	switch action {
	case "list_agents":
		return t.listAgents(ctx, params)
	case "call_agent":
		return t.callAgent(ctx, params)
	default:
		return t.callAgent(ctx, params) // 默认为 call_agent，保持向后兼容
	}
}

// listAgents 获取可用的远程 Agent 列表
// 对应 Python 版本的 get_remote_agent_cards 工具
func (t *A2ATool) listAgents(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	t.mu.RLock()
	manager := t.manager
	t.mu.RUnlock()

	if manager == nil {
		return model.NewToolError("A2A 客户端管理器未初始化"), nil
	}

	cards := manager.GetAgentCards()

	// 重组结构，将 id 填充到 agent_card 中
	agentCards := make([]map[string]interface{}, 0, len(cards))
	for id, card := range cards {
		agentCard := map[string]interface{}{
			"id":          id,
			"name":        card.Name,
			"description": card.Description,
			"url":         card.URL,
			"version":     card.Version,
			"enabled":     card.Enabled,
		}

		// 添加技能列表
		if len(card.Skills) > 0 {
			skills := make([]map[string]interface{}, 0, len(card.Skills))
			for _, skill := range card.Skills {
				skills = append(skills, map[string]interface{}{
					"id":          skill.ID,
					"name":        skill.Name,
					"description": skill.Description,
				})
			}
			agentCard["skills"] = skills
		}

		// 添加元数据
		if card.Metadata != nil {
			agentCard["metadata"] = card.Metadata
		}

		agentCards = append(agentCards, agentCard)
	}

	return model.NewToolResultWithMessage(
		fmt.Sprintf("获取到 %d 个可用的远程 Agent", len(agentCards)),
		map[string]interface{}{
			"agents": agentCards,
		},
	), nil
}

// callAgent 调用远程 Agent
// 对应 Python 版本的 call_remote_agent 工具
func (t *A2ATool) callAgent(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	agentID, _ := params["agent_id"].(string)
	task, _ := params["task"].(string)

	if agentID == "" {
		return model.NewToolError("agent_id 不能为空"), nil
	}

	if task == "" {
		return model.NewToolError("task 不能为空"), nil
	}

	t.mu.RLock()
	manager := t.manager
	t.mu.RUnlock()

	if manager == nil {
		return model.NewToolError("A2A 客户端管理器未初始化"), nil
	}

	logger.Info("调用远程 Agent",
		zap.String("agent_id", agentID),
		zap.String("task", task))

	// 调用远程 Agent
	result, err := manager.Invoke(ctx, agentID, task)
	if err != nil {
		logger.Error("调用远程 Agent 失败",
			zap.String("agent_id", agentID),
			zap.Error(err))
		return model.NewToolError(err.Error()), nil
	}

	// 处理结果
	if !result.Success {
		return model.NewToolError(result.Message), nil
	}

	// 提取响应文本
	responseText := t.extractResponseText(result.Data)

	return model.NewToolResultWithMessage(
		responseText,
		result.Data,
	), nil
}

// extractResponseText 从 A2A 响应中提取文本内容
func (t *A2ATool) extractResponseText(data interface{}) string {
	if data == nil {
		return "调用成功，但无返回内容"
	}

	// 如果是 map，尝试提取文本
	if dataMap, ok := data.(map[string]interface{}); ok {
		// 尝试从 result.message.parts 中提取文本
		if result, ok := dataMap["result"].(map[string]interface{}); ok {
			if message, ok := result["message"].(map[string]interface{}); ok {
				if parts, ok := message["parts"].([]interface{}); ok {
					var textBuilder string
					for _, part := range parts {
						if partMap, ok := part.(map[string]interface{}); ok {
							if text, ok := partMap["text"].(string); ok {
								textBuilder += text
							}
						}
					}
					if textBuilder != "" {
						return textBuilder
					}
				}
			}
		}

		// 尝试直接返回 JSON
		jsonBytes, _ := json.Marshal(data)
		return string(jsonBytes)
	}

	// 其他类型尝试 JSON 序列化
	jsonBytes, _ := json.Marshal(data)
	return string(jsonBytes)
}

// Initialize 初始化 A2A 工具
// 参考 Python 版本的 A2ATool.initialize()
func (t *A2ATool) Initialize(cfg *A2AConfig) error {
	if cfg == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.config = cfg

	// 构建服务器配置
	servers := make([]external.A2AServerConfig, 0, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		// 从 URL 中提取 base URL（去掉路径）
		baseURL := agent.URL
		if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
			baseURL = baseURL[:len(baseURL)-1]
		}

		servers = append(servers, external.A2AServerConfig{
			ID:      agent.Name, // 使用名称作为唯一 ID
			BaseURL: baseURL,
			Enabled: true,
		})
	}

	// 创建客户端管理器配置
	config := &external.A2AClientManagerConfig{
		Servers: servers,
	}

	// 初始化客户端管理器
	ctx := context.Background()
	if err := t.manager.Initialize(ctx, config); err != nil {
		logger.Error("A2A 客户端管理器初始化失败", zap.Error(err))
		return err
	}

	return nil
}

// Cleanup 清理 A2A 资源
func (t *A2ATool) Cleanup() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.manager != nil {
		if err := t.manager.Cleanup(); err != nil {
			return err
		}
	}

	t.config = nil
	return nil
}

// GetManager 获取 A2A 客户端管理器（用于测试）
func (t *A2ATool) GetManager() *external.A2AClientManager {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.manager
}
