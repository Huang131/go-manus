package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// ReActAgent 基于 ReAct 架构的执行 Agent
type ReActAgent struct {
	BaseAgent
}

// NewReActAgent 创建执行 Agent
func NewReActAgent(
	sessionID string,
	config *AgentConfig,
	llm external.LLM,
	tools []Tool,
) *ReActAgent {
	agent := &ReActAgent{}
	agent.BaseAgent = *NewBaseAgent("react", sessionID, config, llm, tools)
	return agent
}

// ExecuteStep 执行单个步骤
func (a *ReActAgent) ExecuteStep(ctx context.Context, plan *model.Plan, step *model.PlanStep, input *TaskInput) error {
	message := &input.Message

	// 构建附件字符串
	attachments := ""
	for _, att := range message.Attachments {
		attachments += fmt.Sprintf("- %s\n", att)
	}

	// 构建附件内容段
	contextSection := BuildAttachmentContextSection(input.AttachmentContexts)

	// 构建提示词
	prompt := ExecutionPrompt
	prompt = strings.Replace(prompt, "{message}", message.ContentText, 1)
	prompt = strings.Replace(prompt, "{attachments}", attachments, 1)
	prompt = strings.Replace(prompt, "{context}", contextSection, 1)
	prompt = strings.Replace(prompt, "{language}", plan.Language, 1)
	prompt = strings.Replace(prompt, "{step}", step.Description, 1)

	// 总结阶段是面向用户的纯文本输出，不能叠加要求结构化 JSON 的 ReAct 系统提示。
	systemPrompt := SystemPrompt

	// 使用完整的 ReAct 循环调用 LLM（记忆以原生消息注入，不再拼字符串）
	// 步骤执行的响应是结构化 JSON；先完整聚合后解析，避免把半截 JSON 当作用户消息展示。
	result, err := a.InvokeWithoutStreaming(ctx, systemPrompt, prompt)
	if err != nil {
		// 如果是等待用户输入的错误，记录用户问题到 step
		if err == ErrWaitForUser {
			step.UserQuestion = result.UserQuestion
			step.Status = model.ExecutionStatusRunning
			return ErrWaitForUser
		}
		step.Status = model.ExecutionStatusFailed
		step.Error = err.Error()
		return fmt.Errorf("LLM调用失败: %w", err)
	}

	// 检查是否是等待用户输入
	if result.WaitForUser {
		step.UserQuestion = result.UserQuestion
		step.Status = model.ExecutionStatusRunning
		return ErrWaitForUser
	}

	// 解析响应（使用 JSON 修复解析器）
	var stepResult struct {
		Success     bool     `json:"success"`
		Result      string   `json:"result"`
		Attachments []string `json:"attachments"`
	}

	if err := a.jsonParser.Parse(result.Content, &stepResult); err != nil {
		// 如果 JSON 解析失败，将整个响应作为结果
		logger.WarnContext(ctx, "JSON 解析失败，尝试直接提取结果",
			logger.String("content", result.Content),
			logger.Err(err))
		step.Success = false
		step.Result = result.Content
		step.Status = model.ExecutionStatusFailed
		step.Error = "JSON 解析失败"
		return nil
	}

	// 更新步骤状态
	step.Success = stepResult.Success
	step.Result = stepResult.Result
	step.Attachments = stepResult.Attachments
	if stepResult.Success {
		step.Status = model.ExecutionStatusCompleted
	} else {
		step.Status = model.ExecutionStatusFailed
		step.Error = stepResult.Result
	}

	logger.InfoContext(ctx, "ReActAgent 执行步骤完成",
		logger.String("step_id", step.ID),
		logger.Bool("success", step.Success),
		logger.String("result", step.Result))

	return nil
}

// Summarize 总结任务执行结果
func (a *ReActAgent) Summarize(ctx context.Context) (string, []string, bool, error) {
	// 构建提示词
	prompt := SummarizePrompt

	// 添加系统提示词
	systemPrompt := SystemPrompt + "\n" + ReActSystemPrompt

	// 构建消息历史：system + 记忆原生消息 + 总结请求
	messages := a.buildConversationMessages(systemPrompt, prompt)

	// 总结面向用户展示，使用非结构化文本流以便前端按 token 增量渲染。
	// 若模型仍返回旧版 JSON，下面的解析逻辑仍可兼容并提取 message。
	resp, emitted, err := a.invokeLLMWithEmission(ctx, &external.LLMRequest{
		Messages: messages,
	}, true)
	if err != nil {
		return "", nil, false, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析响应（使用 JSON 修复解析器）
	var result struct {
		Message     string   `json:"message"`
		Attachments []string `json:"attachments"`
	}

	if err := a.jsonParser.Parse(resp.Message.ContentText, &result); err != nil {
		// 如果 JSON 解析失败，返回原始内容
		return resp.Message.ContentText, nil, emitted, nil
	}

	logger.InfoContext(ctx, "ReActAgent 任务总结完成")
	return result.Message, result.Attachments, emitted, nil
}
