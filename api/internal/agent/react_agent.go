package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/llmcore"
	"github.com/mooc-manus/go-manus/api/internal/model"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
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
func (a *ReActAgent) ExecuteStep(ctx context.Context, plan *model.Plan, step *model.PlanStep, message *model.Message) error {
	// 构建附件字符串
	attachments := ""
	for _, att := range message.Attachments {
		attachments += fmt.Sprintf("- %s\n", att)
	}

	// 构建附件内容段
	contextSection := BuildAttachmentContextSection(message.AttachmentContexts)

	// 构建提示词
	prompt := ExecutionPrompt
	prompt = strings.Replace(prompt, "{message}", message.Message, 1)
	prompt = strings.Replace(prompt, "{attachments}", attachments, 1)
	prompt = strings.Replace(prompt, "{context}", contextSection, 1)
	prompt = strings.Replace(prompt, "{language}", plan.Language, 1)
	prompt = strings.Replace(prompt, "{step}", step.Description, 1)

	// 添加系统提示词
	systemPrompt := SystemPrompt + "\n" + ReActSystemPrompt

	// 添加记忆上下文
	memoryContext := a.GetMemoryContext()

	// 构建消息（包含系统提示和记忆）
	fullQuery := systemPrompt + "\n\n"
	if memoryContext != "" {
		fullQuery += "历史上下文:\n" + memoryContext + "\n\n"
	}
	fullQuery += prompt

	// 使用完整的 ReAct 循环调用 LLM
	result, err := a.Invoke(ctx, fullQuery)
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
		logger.Warn("JSON 解析失败，尝试直接提取结果",
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

	logger.Info("ReActAgent 执行步骤完成",
		logger.String("step_id", step.ID),
		logger.Bool("success", step.Success),
		logger.String("result", step.Result))

	// 添加到记忆
	_ = a.AddMemory(ctx, message)

	return nil
}

// Summarize 总结任务执行结果
func (a *ReActAgent) Summarize(ctx context.Context) (string, []string, error) {
	// 构建提示词
	prompt := SummarizePrompt

	// 添加系统提示词
	systemPrompt := SystemPrompt + "\n" + ReActSystemPrompt

	// 添加记忆上下文
	memoryContext := a.GetMemoryContext()

	// 构建消息历史（阶段 1d：改 llmcore.Message 强类型）
	messages := []llmcore.Message{
		{Role: llmcore.RoleSystem, ContentText: systemPrompt},
	}

	// 添加记忆上下文
	if memoryContext != "" {
		messages = append(messages, llmcore.Message{
			Role:        llmcore.RoleSystem,
			ContentText: "任务执行上下文:\n" + memoryContext,
		})
	}

	messages = append(messages, llmcore.Message{
		Role:        llmcore.RoleUser,
		ContentText: prompt,
	})

	// 调用 LLM
	resp, _, err := a.invokeWithEmptyRetry(ctx, &external.LLMRequest{
		Messages: messages,
	}, a.config.MaxRetries)
	if err != nil {
		return "", nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析响应（使用 JSON 修复解析器）
	var result struct {
		Message     string   `json:"message"`
		Attachments []string `json:"attachments"`
	}

	if err := a.jsonParser.Parse(resp.Content, &result); err != nil {
		// 如果 JSON 解析失败，返回原始内容
		return resp.Content, nil, nil
	}

	logger.Info("ReActAgent 任务总结完成")
	return result.Message, result.Attachments, nil
}

// GetMemoryContext 获取记忆上下文
func (a *ReActAgent) GetMemoryContext() string {
	messages := a.GetMemory()
	if len(messages) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, msg := range messages {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		builder.WriteString(fmt.Sprintf("[%s] %s\n", role, msg.Message))
	}

	return builder.String()
}
