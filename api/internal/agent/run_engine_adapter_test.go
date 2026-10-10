package agent

import (
	"context"
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
}

func (m *adapterLLM) Invoke(context.Context, *llm.LLMRequest) (*llmcore.LLMResponse, error) {
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
