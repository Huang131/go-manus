package agent

import (
	"context"
	"fmt"

	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/google/uuid"
)

// PlannerEngineAdapter 将现有 PlannerReActFlow 适配为 RunExecutor 的 Engine 边界。
// 它只负责事件收集和领域结果转换，不直接写 Repository、Session 或 Redis。
type PlannerEngineAdapter struct {
	LLM llm.LLM
}

// Execute 启动一次 Flow，并把最终计划、等待问题或终态消息转换为 Run 结果。
func (a *PlannerEngineAdapter) Execute(ctx context.Context, input service.RunExecutionInput) (service.RunExecutionResult, error) {
	if a == nil || a.LLM == nil {
		return service.RunExecutionResult{Kind: service.RunExecutionFailed, Error: fmt.Errorf("planner engine LLM is nil")}, nil
	}
	flow := NewPlannerReActFlow(input.SessionID, input.Settings, a.LLM, input.Tools)
	if input.Snapshot.WaitingCheckpoint != nil {
		plan, err := planFromSnapshot(input.Snapshot)
		if err != nil {
			return service.RunExecutionResult{Kind: service.RunExecutionFailed, Error: err}, nil
		}
		if err := flow.RestorePlan(plan, input.Messages); err != nil {
			return service.RunExecutionResult{Kind: service.RunExecutionFailed, Error: err}, nil
		}
	}
	message := lastUserMessage(input.Messages)
	if message == nil {
		return service.RunExecutionResult{Kind: service.RunExecutionFailed, Error: fmt.Errorf("run has no user message")}, nil
	}

	var lastAssistant string
	for event := range flow.Invoke(ctx, &TaskInput{Message: *message, AttachmentContexts: input.AttachmentContexts}) {
		if input.EventPublisher != nil {
			if err := input.EventPublisher.Publish(ctx, input.RunID, event); err != nil {
				logger.WarnContext(ctx, "发布 Run 实时事件失败",
					logger.String("run_id", input.RunID), logger.Err(err))
			}
		}
		if messageEvent, ok := event.(*model.MessageEvent); ok && messageEvent.Role == model.RoleAssistant {
			lastAssistant = messageEvent.Message
		}
	}
	plan := flow.GetPlan()
	if ctx.Err() != nil || flow.GetStatus() == FlowStatusCancelled {
		return service.RunExecutionResult{Kind: service.RunExecutionCancelled, Snapshot: snapshotFromPlan(plan, input.Snapshot)}, nil
	}
	if plan == nil {
		return service.RunExecutionResult{Kind: service.RunExecutionFailed, Error: fmt.Errorf("planner returned no plan")}, nil
	}
	if flow.GetStatus() == FlowStatusWaiting {
		return waitingResult(plan, input)
	}
	if flow.GetStatus() == FlowStatusFailed {
		return service.RunExecutionResult{Kind: service.RunExecutionFailed, Snapshot: snapshotFromPlan(plan, input.Snapshot), Text: lastAssistant}, nil
	}
	return service.RunExecutionResult{
		Kind:     service.RunExecutionSucceeded,
		Snapshot: snapshotFromPlan(plan, input.Snapshot),
		Text:     lastAssistant,
	}, nil
}

func lastUserMessage(messages []llmcore.Message) *llmcore.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == model.RoleUser {
			message := messages[i]
			return &message
		}
	}
	return nil
}

func snapshotFromPlan(plan *model.Plan, previous model.RunExecutionSnapshot) model.RunExecutionSnapshot {
	snapshot := previous.Clone()
	if plan == nil {
		return snapshot
	}
	snapshot.PlanID = plan.ID
	if snapshot.PlanID == "" {
		snapshot.PlanID = uuid.NewString()
	}
	snapshot.PlanTitle = plan.Title
	snapshot.PlanGoal = plan.Goal
	snapshot.PlanLanguage = plan.Language
	snapshot.PlanMessage = plan.Message
	snapshot.Steps = make([]model.RunStepSnapshot, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		status := model.RunStepStatusPending
		switch step.Status {
		case model.ExecutionStatusRunning:
			status = model.RunStepStatusRunning
		case model.ExecutionStatusCompleted:
			status = model.RunStepStatusCompleted
		case model.ExecutionStatusFailed:
			status = model.RunStepStatusFailed
		}
		snapshot.Steps = append(snapshot.Steps, model.RunStepSnapshot{ID: step.ID, Description: step.Description, Status: status, ResultSummary: step.Result, ArtifactRefs: append([]string(nil), step.Attachments...)})
		if !step.Done() {
			snapshot.CurrentStepID = step.ID
		}
	}
	snapshot.SnapshotRevision = previous.SnapshotRevision + 1
	return snapshot
}

func planFromSnapshot(snapshot model.RunExecutionSnapshot) (*model.Plan, error) {
	if err := snapshot.ValidateWaitingInput(); err != nil {
		return nil, fmt.Errorf("restore execution snapshot: %w", err)
	}
	plan := &model.Plan{
		ID: snapshot.PlanID, Title: snapshot.PlanTitle, Goal: snapshot.PlanGoal,
		Language: snapshot.PlanLanguage, Message: snapshot.PlanMessage,
		Status: model.ExecutionStatusRunning, Steps: make([]model.PlanStep, 0, len(snapshot.Steps)),
	}
	if plan.Language == "" {
		plan.Language = "zh"
	}
	for _, step := range snapshot.Steps {
		status := model.ExecutionStatusPending
		switch step.Status {
		case model.RunStepStatusRunning:
			status = model.ExecutionStatusRunning
		case model.RunStepStatusCompleted:
			status = model.ExecutionStatusCompleted
		case model.RunStepStatusFailed:
			status = model.ExecutionStatusFailed
		}
		plan.Steps = append(plan.Steps, model.PlanStep{
			ID: step.ID, Description: step.Description, Status: status,
			Success: status == model.ExecutionStatusCompleted,
			Result:  step.ResultSummary, Attachments: append([]string(nil), step.ArtifactRefs...),
		})
	}
	return plan, nil
}

func waitingResult(plan *model.Plan, input service.RunExecutionInput) (service.RunExecutionResult, error) {
	snapshot := snapshotFromPlan(plan, input.Snapshot)
	if snapshot.CurrentStepID == "" {
		return service.RunExecutionResult{Kind: service.RunExecutionFailed, Error: fmt.Errorf("waiting flow has no current step")}, nil
	}
	var step *model.PlanStep
	for i := range plan.Steps {
		if plan.Steps[i].ID == snapshot.CurrentStepID {
			step = &plan.Steps[i]
			break
		}
	}
	if step == nil || step.UserQuestion == "" {
		return service.RunExecutionResult{Kind: service.RunExecutionFailed, Error: fmt.Errorf("waiting flow has no question")}, nil
	}
	question := &model.RunMessage{ID: uuid.NewString(), SessionID: input.SessionID, RunID: input.RunID, Role: model.RoleAssistant, Content: step.UserQuestion}
	snapshot.WaitingCheckpoint = &model.WaitingCheckpoint{QuestionMessageID: question.ID, StepID: step.ID, ResumeMode: model.ResumeModeContinueStep}
	return service.RunExecutionResult{Kind: service.RunExecutionWaitingInput, Snapshot: snapshot, Question: question}, nil
}
