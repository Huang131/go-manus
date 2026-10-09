package agent

import (
	"context"
	"errors"
	"testing"

	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

type waitingOutcomeTool struct{}

func (waitingOutcomeTool) Name() string        { return "waiting_tool" }
func (waitingOutcomeTool) Description() string { return "waits for user input" }
func (waitingOutcomeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (waitingOutcomeTool) ReadOnly() bool { return false }
func (waitingOutcomeTool) Invoke(context.Context, map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolError("unexpected generic invocation"), nil
}
func (waitingOutcomeTool) GetTools() []map[string]interface{} {
	return []map[string]interface{}{{
		"name":        "waiting_tool_ask",
		"description": "waits for input",
		"parameters":  map[string]interface{}{"type": "object"},
	}}
}
func (waitingOutcomeTool) InvokeWithName(string, context.Context, map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolResult(map[string]interface{}{
		"waiting_for_user":      true,
		"text":                  "需要补充信息",
		"attachments":           []string{"file-1"},
		"suggest_user_takeover": "browser",
	}), nil
}

func TestBaseAgentInvokeReturnsWaitingInputOutcome(t *testing.T) {
	mock := &mockLLM{responses: []*llmcore.LLMResponse{{
		Message: llmcore.Message{
			Role: model.RoleAssistant,
			ToolCalls: []llmcore.ToolCall{{
				ID:   "call-1",
				Type: "function",
				Function: llmcore.ToolCallFunction{
					Name:      "waiting_tool_ask",
					Arguments: `{}`,
				},
			}},
		},
	}}}
	agent := NewBaseAgent("react", "session-1", defaultAgentSettings(), mock, []toolspkg.Tool{waitingOutcomeTool{}})

	result, err := agent.Invoke(context.Background(), "system", "query")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Outcome.Kind != OutcomeWaitingInput {
		t.Fatalf("Outcome.Kind = %q, want %q", result.Outcome.Kind, OutcomeWaitingInput)
	}
	if result.Outcome.Waiting == nil || result.Outcome.Waiting.Question != "需要补充信息" || result.Outcome.Waiting.SuggestTakeover != "browser" {
		t.Fatalf("Outcome.Waiting = %+v", result.Outcome.Waiting)
	}
}

func TestWaitingInputFromToolResultPreservesDisplayFields(t *testing.T) {
	tests := []struct {
		name        string
		attachments interface{}
		want        []string
	}{
		{name: "single string", attachments: "a", want: []string{"a"}},
		{name: "decoded json array", attachments: []interface{}{"a", "b"}, want: []string{"a", "b"}},
		{name: "typed string array", attachments: []string{"a"}, want: []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.NewToolResult(map[string]interface{}{
				"text":                  "question",
				"attachments":           tt.attachments,
				"suggest_user_takeover": "none",
			})
			waiting := waitingInputFromToolResult(result)
			if waiting.Question != "question" || waiting.SuggestTakeover != "none" {
				t.Fatalf("waiting = %+v", waiting)
			}
			if len(waiting.Attachments) != len(tt.want) {
				t.Fatalf("attachments = %#v, want %#v", waiting.Attachments, tt.want)
			}
			for i := range tt.want {
				if waiting.Attachments[i] != tt.want[i] {
					t.Fatalf("attachments = %#v, want %#v", waiting.Attachments, tt.want)
				}
			}
		})
	}
}

func TestReActAgentExecuteStepReturnsCancelledOutcome(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	react := NewReActAgent("session-1", defaultAgentSettings(), &mockLLM{}, nil)
	step := &model.PlanStep{ID: "step-1", Status: model.ExecutionStatusPending}
	outcome, err := react.ExecuteStep(ctx, &model.Plan{Language: "zh"}, step, &TaskInput{
		Message: llmcore.Message{Role: model.RoleUser, ContentText: "continue"},
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ExecuteStep() error = %v, want context.Canceled", err)
	}
	if outcome == nil || outcome.Kind != OutcomeCancelled {
		t.Fatalf("ExecuteStep() outcome = %+v, want cancelled", outcome)
	}
	if step.Status == model.ExecutionStatusFailed || step.Error != "" {
		t.Fatalf("cancelled step was marked failed: %+v", step)
	}
}

func TestPlannerReActFlowWaitingOutcomeEmitsQuestionBeforeWait(t *testing.T) {
	mock := &mockLLM{responses: []*llmcore.LLMResponse{
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: `{"message":"开始执行","goal":"补充信息","title":"等待输入","language":"zh","steps":[{"id":"step-1","description":"询问用户"}]}`}},
		{Message: llmcore.Message{Role: model.RoleAssistant, ToolCalls: []llmcore.ToolCall{{
			ID:   "call-1",
			Type: "function",
			Function: llmcore.ToolCallFunction{
				Name:      "waiting_tool_ask",
				Arguments: `{}`,
			},
		}}}},
	}}
	flow := NewPlannerReActFlow("session-1", defaultAgentSettings(), mock, []toolspkg.Tool{waitingOutcomeTool{}})
	events := flow.Invoke(context.Background(), &TaskInput{
		Message: llmcore.Message{Role: model.RoleUser, ContentText: "do it"},
	})

	var received []model.BaseEvent
	for event := range events {
		received = append(received, event)
	}
	if len(received) < 2 {
		t.Fatalf("events = %d, want at least question and wait", len(received))
	}
	question, ok := received[len(received)-2].(*model.MessageEvent)
	if !ok || question.Role != model.RoleAssistant || question.Message != "需要补充信息" {
		t.Fatalf("penultimate event = %#v, want assistant question", received[len(received)-2])
	}
	if _, ok := received[len(received)-1].(*model.WaitEvent); !ok {
		t.Fatalf("last event = %#v, want wait", received[len(received)-1])
	}
	if flow.GetStatus() != FlowStatusWaiting {
		t.Fatalf("flow status = %s, want waiting", flow.GetStatus())
	}
	plan := flow.GetPlan()
	if plan == nil || len(plan.Steps) != 1 || plan.Steps[0].UserQuestion != "需要补充信息" {
		t.Fatalf("plan = %+v, want persisted question", plan)
	}
	if plan.Steps[0].ID != "step-1" {
		t.Fatalf("step id = %q, want step-1", plan.Steps[0].ID)
	}
}

func TestReActAgentExecuteStepReturnsWaitingInputWithStepID(t *testing.T) {
	mock := &mockLLM{responses: []*llmcore.LLMResponse{{
		Message: llmcore.Message{Role: model.RoleAssistant, ToolCalls: []llmcore.ToolCall{{
			ID: "call-1", Type: "function",
			Function: llmcore.ToolCallFunction{Name: "waiting_tool_ask", Arguments: `{}`},
		}},
		},
	}}}
	react := NewReActAgent("session-1", defaultAgentSettings(), mock, []toolspkg.Tool{waitingOutcomeTool{}})
	step := &model.PlanStep{ID: "step-1", Description: "ask", Status: model.ExecutionStatusPending}
	outcome, err := react.ExecuteStep(context.Background(), &model.Plan{Language: "zh"}, step, &TaskInput{
		Message: llmcore.Message{Role: model.RoleUser, ContentText: "continue"},
	})

	if err != nil {
		t.Fatalf("ExecuteStep() error = %v", err)
	}
	if outcome == nil || outcome.Kind != OutcomeWaitingInput || outcome.Waiting == nil {
		t.Fatalf("ExecuteStep() outcome = %+v, want waiting input", outcome)
	}
	if outcome.Waiting.StepID != "step-1" || outcome.Waiting.Question != "需要补充信息" {
		t.Fatalf("waiting input = %+v", outcome.Waiting)
	}
	if step.Status != model.ExecutionStatusRunning || step.UserQuestion != "需要补充信息" {
		t.Fatalf("step = %+v, want running with user question", step)
	}
}

func TestReActAgentExecuteStepReturnsTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantKind   OutcomeKind
		wantStatus model.ExecutionStatus
		wantText   string
	}{
		{name: "completed", response: `{"success":true,"result":"done","attachments":["a"]}`, wantKind: OutcomeCompleted, wantStatus: model.ExecutionStatusCompleted, wantText: "done"},
		{name: "business failure", response: `{"success":false,"result":"cannot continue"}`, wantKind: OutcomeFatalFailure, wantStatus: model.ExecutionStatusFailed, wantText: "cannot continue"},
		{name: "invalid response", response: `not-json`, wantKind: OutcomeFatalFailure, wantStatus: model.ExecutionStatusFailed, wantText: "not-json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLLM{responses: []*llmcore.LLMResponse{{
				Message: llmcore.Message{Role: model.RoleAssistant, ContentText: tt.response},
			}}}
			react := NewReActAgent("session-1", defaultAgentSettings(), mock, nil)
			step := &model.PlanStep{ID: "step-1", Status: model.ExecutionStatusPending}
			outcome, err := react.ExecuteStep(context.Background(), &model.Plan{Language: "zh"}, step, &TaskInput{
				Message: llmcore.Message{Role: model.RoleUser, ContentText: "continue"},
			})
			if err != nil {
				t.Fatalf("ExecuteStep() error = %v", err)
			}
			if outcome == nil || outcome.Kind != tt.wantKind || outcome.Text != tt.wantText {
				t.Fatalf("ExecuteStep() outcome = %+v, want kind=%s text=%q", outcome, tt.wantKind, tt.wantText)
			}
			if step.Status != tt.wantStatus {
				t.Fatalf("step status = %s, want %s", step.Status, tt.wantStatus)
			}
			if tt.wantKind == OutcomeFatalFailure && outcome.Failure == nil {
				t.Fatal("fatal outcome must retain diagnostic failure")
			}
		})
	}
}
