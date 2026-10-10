package agent

import (
	"context"
	"strings"
	"sync"
	"testing"

	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"
	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/internal/settings"
)

type adapterLLM struct {
	responses []*llmcore.LLMResponse
	requests  [][]llmcore.Message
	calls     int
}

func (m *adapterLLM) Invoke(_ context.Context, request *llm.LLMRequest) (*llmcore.LLMResponse, error) {
	m.calls++
	m.requests = append(m.requests, append([]llmcore.Message(nil), request.Messages...))
	if len(m.responses) == 0 {
		return &llmcore.LLMResponse{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "{\"success\":true,\"result\":\"done\"}"}}, nil
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return response, nil
}

func (m *adapterLLM) ModelName() string    { return "adapter-test" }
func (m *adapterLLM) Temperature() float64 { return 0 }
func (m *adapterLLM) MaxTokens() int       { return 1024 }

func adapterSettings() settings.AgentSettings { return settings.DefaultAgentSettings() }

func TestPlannerEngineAdapterConvertsCompletedFlow(t *testing.T) {
	adapter := &PlannerEngineAdapter{LLM: &adapterLLM{responses: []*llmcore.LLMResponse{
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "{\"message\":\"开始\",\"goal\":\"完成任务\",\"title\":\"任务\",\"language\":\"zh\",\"steps\":[{\"id\":\"step-1\",\"description\":\"执行\"}]}"}},
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "{\"success\":true,\"result\":\"done\"}"}},
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "summary"}},
	}}}
	result, err := adapter.Execute(context.Background(), service.RunExecutionInput{
		RunID: "run-1", SessionID: "session-1", Settings: adapterSettings(),
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "do it"}},
		Tools:    []toolspkg.Tool{waitingOutcomeTool{}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Kind != service.RunExecutionSucceeded || result.Text != "summary" {
		t.Fatalf("result = %+v, want succeeded summary", result)
	}
	if result.Snapshot.PlanID == "" || len(result.Snapshot.Steps) != 1 {
		t.Fatalf("snapshot = %+v, want one step", result.Snapshot)
	}
}

func TestPlannerEngineAdapterWaitingInputBuildsResumableQuestion(t *testing.T) {
	adapter := &PlannerEngineAdapter{LLM: &adapterLLM{responses: []*llmcore.LLMResponse{
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "{\"message\":\"开始\",\"goal\":\"确认\",\"title\":\"任务\",\"language\":\"zh\",\"steps\":[{\"id\":\"step-1\",\"description\":\"确认\",\"user_question\":\"请确认\"}]}"}},
		{Message: llmcore.Message{Role: model.RoleAssistant, ToolCalls: []llmcore.ToolCall{{ID: "call-1", Type: "function", Function: llmcore.ToolCallFunction{Name: "waiting_tool_ask", Arguments: "{}"}}}}},
	}}}
	result, err := adapter.Execute(context.Background(), service.RunExecutionInput{
		RunID: "run-1", SessionID: "session-1", Settings: adapterSettings(),
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "do it"}},
		Snapshot: model.RunExecutionSnapshot{SnapshotRevision: 0},
		Tools:    []toolspkg.Tool{waitingOutcomeTool{}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Kind != service.RunExecutionWaitingInput || result.Question == nil {
		t.Fatalf("result = %+v, want waiting question", result)
	}
	if result.Question.RunID != "run-1" || result.Question.SessionID != "session-1" {
		t.Fatalf("question ownership = %+v", result.Question)
	}
	if result.Snapshot.SnapshotRevision != 1 {
		t.Fatalf("snapshot revision = %d, want 1", result.Snapshot.SnapshotRevision)
	}
	if result.Snapshot.WaitingCheckpoint.QuestionMessageID != result.Question.ID {
		t.Fatalf("checkpoint = %+v, question = %+v", result.Snapshot.WaitingCheckpoint, result.Question)
	}
}

func TestPlannerEngineAdapterMapsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	adapter := &PlannerEngineAdapter{LLM: &adapterLLM{}}
	result, err := adapter.Execute(ctx, service.RunExecutionInput{
		RunID: "run-1", SessionID: "session-1", Settings: adapterSettings(),
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "do it"}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Kind != service.RunExecutionCancelled {
		t.Fatalf("result kind = %s, want cancelled", result.Kind)
	}
}

func TestPlannerEngineAdapterResumesWaitingPlanWithoutReplanning(t *testing.T) {
	llm := &adapterLLM{responses: []*llmcore.LLMResponse{
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "{\"success\":true,\"result\":\"已确认并执行\"}"}},
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "已完成"}},
	}}
	adapter := &PlannerEngineAdapter{LLM: llm}
	result, err := adapter.Execute(context.Background(), service.RunExecutionInput{
		RunID: "run-1", SessionID: "session-1", Settings: adapterSettings(),
		Snapshot: model.RunExecutionSnapshot{
			SnapshotRevision: 1,
			PlanID:           "plan-1",
			CurrentStepID:    "step-2",
			Steps: []model.RunStepSnapshot{
				{ID: "step-1", Description: "收集任务信息", Status: model.RunStepStatusCompleted, ResultSummary: "已收集信息"},
				{ID: "step-2", Description: "根据用户确认继续执行", Status: model.RunStepStatusRunning},
			},
			WaitingCheckpoint: &model.WaitingCheckpoint{QuestionMessageID: "question-1", StepID: "step-2", ResumeMode: model.ResumeModeContinueStep},
		},
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "帮我完成任务"},
			{Role: model.RoleAssistant, ContentText: "请确认继续"},
			{Role: model.RoleUser, ContentText: "我同意执行下一阶段"},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Kind != service.RunExecutionSucceeded || result.Text != "已完成" {
		t.Fatalf("result = %+v, want resumed success", result)
	}
	if llm.calls != 2 {
		t.Fatalf("LLM calls = %d, want 2 without replanning", llm.calls)
	}
	if countText(llm.requests[0], "我同意执行下一阶段") != 1 {
		t.Fatalf("step request duplicated resumed user input: %#v", llm.requests[0])
	}
}

type adapterEventPublisher struct {
	mu     sync.Mutex
	events []model.BaseEvent
	err    error
}

func (p *adapterEventPublisher) Publish(_ context.Context, _ string, event model.BaseEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return p.err
}

func (p *adapterEventPublisher) snapshot() []model.BaseEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]model.BaseEvent(nil), p.events...)
}

func TestPlannerEngineAdapterPublishesFlowEventsWithoutChangingResult(t *testing.T) {
	publisher := &adapterEventPublisher{err: context.DeadlineExceeded}
	adapter := &PlannerEngineAdapter{LLM: &adapterLLM{responses: []*llmcore.LLMResponse{
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "{\"message\":\"开始\",\"goal\":\"完成\",\"title\":\"任务\",\"language\":\"zh\",\"steps\":[{\"id\":\"step-1\",\"description\":\"执行\"}]}"}},
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "{\"success\":true,\"result\":\"完成\"}"}},
		{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: "总结"}},
	}}}
	result, err := adapter.Execute(context.Background(), service.RunExecutionInput{
		RunID: "run-1", SessionID: "session-1", Settings: adapterSettings(),
		Messages:       []llmcore.Message{{Role: model.RoleUser, ContentText: "执行"}},
		EventPublisher: publisher,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Kind != service.RunExecutionSucceeded || result.Text != "总结" {
		t.Fatalf("result = %+v, want successful result despite publisher error", result)
	}
	events := publisher.snapshot()
	if len(events) == 0 {
		t.Fatal("event publisher received no flow events")
	}
}

func countText(messages []llmcore.Message, text string) int {
	count := 0
	for _, message := range messages {
		count += strings.Count(message.ContentText, text)
	}
	return count
}
