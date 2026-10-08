package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/Huang131/go-manus/api/internal/a2a"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

// A2ATool A2A (Agent-to-Agent) 工具
type A2ATool struct {
	mu      sync.RWMutex
	manager *a2a.A2AClientManager
}

// NewA2ATool 创建 A2A 工具
func NewA2ATool() *A2ATool {
	return &A2ATool{
		manager: a2a.NewA2AClientManager(),
	}
}

// Name 返回工具名称
func (t *A2ATool) Name() string {
	return ToolNameA2A
}

// Description 返回工具描述
func (t *A2ATool) Description() string {
	return "调用其他 Agent 处理任务。"
}

// Parameters 返回工具参数定义
func (t *A2ATool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型",
				"enum":        []string{A2AActionListAgents, A2AActionCallAgent},
			},
			"agent_id": map[string]interface{}{
				"type":        "string",
				"description": "Agent 唯一标识（来自 list_agents）",
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "任务描述",
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
func (t *A2ATool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	action, toolErr := requiredToolString(params, "action")
	if toolErr != nil {
		return toolErr, nil
	}

	switch action {
	case A2AActionListAgents:
		return t.listAgents(ctx, params)
	case A2AActionCallAgent:
		return t.callAgent(ctx, params)
	default:
		// 未知 action 必须 fail loud：静默兜底到 call_agent 会让拼错的 action
		// （如 listagent）触发真实的远程执行（有副作用）。返回错误交给模型自纠。
		return model.NewToolError(fmt.Sprintf(
			"未知的 action: %q，可选值: %s、%s",
			action, A2AActionListAgents, A2AActionCallAgent)), nil
	}
}

// listAgents 获取可用的远程 Agent 列表
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
		// 端点取 ResolveEndpoint，兼容 v1.0（supportedInterfaces）与 v0.3（根层 URL）。
		endpoint, _, _ := card.ResolveEndpoint()
		agentCard := map[string]interface{}{
			"id":          id,
			"name":        card.Name,
			"description": card.Description,
			"url":         endpoint,
			"version":     card.Version,
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
func (t *A2ATool) callAgent(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	agentID, toolErr := requiredToolString(params, "agent_id")
	if toolErr != nil {
		return toolErr, nil
	}
	task, toolErr := requiredToolString(params, "task")
	if toolErr != nil {
		return toolErr, nil
	}

	t.mu.RLock()
	manager := t.manager
	t.mu.RUnlock()

	if manager == nil {
		return model.NewToolError("A2A 客户端管理器未初始化"), nil
	}

	logger.InfoContext(ctx, "调用远程 Agent",
		logger.String("agent_id", agentID),
		logger.String("task", task))

	// 调用远程 Agent
	result, err := manager.Invoke(ctx, agentID, task)
	if err != nil {
		logger.ErrorContext(ctx, "调用远程 Agent 失败",
			logger.String("agent_id", agentID),
			logger.Err(err))
		return model.NewToolError(err.Error()), nil
	}

	// 提取响应文本
	responseText := result.ExtractText()
	if responseText == "" {
		responseText = "调用成功，但无返回内容"
	}

	return model.NewToolResultWithMessage(
		responseText,
		result,
	), nil
}

// Initialize 初始化 A2A 工具
func (t *A2ATool) Initialize(ctx context.Context, cfg *A2AConfig) error {
	if cfg == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// 构建服务器配置。Agent 以 Name 作为唯一 ID（manager 内部按 ID 建 map，
	// 同名后者会静默覆盖前者），这里显式去重：保留先配置者并告警。
	servers := make([]a2a.A2AServerConfig, 0, len(cfg.Agents))
	seen := make(map[string]struct{}, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		if _, dup := seen[agent.Name]; dup {
			logger.WarnContext(ctx, "A2A Agent 名称重复，已跳过后续配置（先配置者生效）",
				logger.String("name", agent.Name),
				logger.String("url", agent.URL))
			continue
		}
		seen[agent.Name] = struct{}{}

		// 从 URL 中提取 base URL（去掉路径）
		baseURL := agent.URL
		if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
			baseURL = baseURL[:len(baseURL)-1]
		}

		servers = append(servers, a2a.A2AServerConfig{
			ID:      agent.Name, // 使用名称作为唯一 ID
			BaseURL: baseURL,
		})
	}

	// 创建客户端管理器配置
	config := &a2a.A2AClientManagerConfig{
		Servers: servers,
	}

	// 初始化客户端管理器
	if err := t.manager.Initialize(ctx, config); err != nil {
		logger.ErrorContext(ctx, "A2A 客户端管理器初始化失败", logger.Err(err))
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

	return nil
}

// GetManager 获取 A2A 客户端管理器（用于测试）
func (t *A2ATool) GetManager() *a2a.A2AClientManager {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.manager
}
