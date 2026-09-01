package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// PlannerAgent 规划 Agent
type PlannerAgent struct {
	BaseAgent
}

// NewPlannerAgent 创建规划 Agent
func NewPlannerAgent(
	sessionID string,
	config *AgentConfig,
	llm external.LLM,
	tools []Tool,
) *PlannerAgent {
	agent := &PlannerAgent{}
	agent.BaseAgent = *NewBaseAgent("planner", sessionID, config, llm, tools)
	return agent
}

// CreatePlan 根据消息创建计划
func (a *PlannerAgent) CreatePlan(ctx context.Context, message *model.Message) (*model.Plan, string, error) {
	// 构建附件字符串
	attachments := ""
	for _, att := range message.Attachments {
		attachments += fmt.Sprintf("- %s\n", att)
	}

	// 构建附件内容段（已加载到 LLM 上下文的文件正文）
	contextSection := BuildAttachmentContextSection(message.AttachmentContexts)

	// 构建提示词
	prompt := CreatePlanPrompt
	prompt = strings.Replace(prompt, "{message}", message.Message, 1)
	prompt = strings.Replace(prompt, "{attachments}", attachments, 1)
	prompt = strings.Replace(prompt, "{context}", contextSection, 1)

	// 添加系统提示词
	systemPrompt := SystemPrompt + "\n" + PlannerSystemPrompt

	// 构建消息历史
	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": prompt},
	}

	// 调用 LLM（与原项目对齐：planner 阶段强制 JSON 输出，抑制 CoT 泄露）
	resp, _, err := a.invokeWithEmptyRetry(ctx, &external.LLMRequest{
		Messages: messages,
		Tools:    a.GetToolsForLLM(),
		ResponseFormat: map[string]interface{}{
			"type": "json_object",
		},
	}, a.config.MaxRetries)
	if err != nil {
		return nil, "", fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析响应（使用 JSON 修复解析器）
	var result struct {
		Message  string `json:"message"`
		Goal     string `json:"goal"`
		Title    string `json:"title"`
		Language string `json:"language"`
		Steps    []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		} `json:"steps"`
	}

	if err := a.jsonParser.Parse(resp.Content, &result); err != nil {
		// 降级：模型未按 JSON 格式输出计划（常见于简单问候场景，推理模型直接给出对话式回答）。
		// 把原始文本作为 assistant 回复，返回空步骤计划，让 flow 走"无步骤直接完成"路径。
		logger.Warn("计划 JSON 解析失败，降级为直接回复",
			zap.String("session_id", a.sessionID),
			zap.Int("content_len", len(resp.Content)),
			zap.Error(err))
		reply := strings.TrimSpace(resp.Content)
		if reply == "" {
			return nil, "", fmt.Errorf("解析计划失败: %w", err)
		}
		title := []rune(reply)
		if len(title) > 20 {
			title = title[:20]
		}
		return &model.Plan{
			Title:    string(title),
			Language: "zh",
			Message:  reply,
			Status:   model.ExecutionStatusCompleted,
			Steps:    []model.PlanStep{},
		}, reply, nil
	}

	// 构建 Plan
	plan := &model.Plan{
		Title:    result.Title,
		Goal:     result.Goal,
		Language: result.Language,
		Message:  result.Message,
		Status:   model.ExecutionStatusCompleted,
		Steps:    make([]model.PlanStep, 0, len(result.Steps)),
	}

	for _, step := range result.Steps {
		plan.Steps = append(plan.Steps, model.PlanStep{
			ID:          step.ID,
			Description: step.Description,
			Status:      model.ExecutionStatusPending,
			Success:     false,
		})
	}

	// 添加到记忆
	_ = a.AddMemory(ctx, message)

	return plan, result.Message, nil
}

// UpdatePlan 根据执行结果更新计划
func (a *PlannerAgent) UpdatePlan(ctx context.Context, plan *model.Plan, completedStep *model.PlanStep) (*model.Plan, error) {
	// 序列化计划和步骤
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("序列化计划失败: %w", err)
	}
	stepJSON, err := json.Marshal(completedStep)
	if err != nil {
		return nil, fmt.Errorf("序列化步骤失败: %w", err)
	}

	// 构建提示词
	prompt := UpdatePlanPrompt
	prompt = strings.Replace(prompt, "{plan}", string(planJSON), 1)
	prompt = strings.Replace(prompt, "{step}", string(stepJSON), 1)

	// 添加系统提示词
	systemPrompt := SystemPrompt + "\n" + PlannerSystemPrompt

	// 构建消息历史
	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": prompt},
	}

	// 调用 LLM（planner 阶段强制 JSON 输出，与原项目对齐）
	resp, _, err := a.invokeWithEmptyRetry(ctx, &external.LLMRequest{
		Messages: messages,
		Tools:    a.GetToolsForLLM(),
		ResponseFormat: map[string]interface{}{
			"type": "json_object",
		},
	}, a.config.MaxRetries)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析响应（使用 JSON 修复解析器）
	var result struct {
		Steps []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		} `json:"steps"`
	}

	if err := a.jsonParser.Parse(resp.Content, &result); err != nil {
		return nil, fmt.Errorf("解析更新计划失败: %w", err)
	}

	// 找到第一个未完成的步骤索引
	firstPendingIndex := -1
	for i, step := range plan.Steps {
		if !step.Done() {
			firstPendingIndex = i
			break
		}
	}

	// 更新计划
	if firstPendingIndex >= 0 {
		// 保留已完成和当前步骤，添加新步骤
		newSteps := make([]model.PlanStep, 0, len(result.Steps)+firstPendingIndex+1)
		newSteps = append(newSteps, plan.Steps[:firstPendingIndex+1]...)

		for _, step := range result.Steps {
			newSteps = append(newSteps, model.PlanStep{
				ID:          step.ID,
				Description: step.Description,
				Status:      model.ExecutionStatusPending,
				Success:     false,
			})
		}

		plan.Steps = newSteps
	}

	return plan, nil
}
