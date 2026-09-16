package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestPlanner_CreatePlanUsesStructuredRequestWithoutTools(t *testing.T) {
	mock := &mockLLM{responses: []*llmcore.LLMResponse{{Message: llmcore.Message{
		Role:        model.RoleAssistant,
		ContentText: `{"message":"可以计算","goal":"回答计算问题","title":"计算","language":"zh-CN","steps":[{"id":"step_1","description":"计算 1+3+4"}]}`,
	}}}}
	planner := NewPlannerAgent("s", DefaultAgentConfig(), mock, nil)
	_, _, err := planner.CreatePlan(context.Background(), &TaskInput{Message: llmcore.Message{ContentText: "1+3+4等于多少"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("LLM calls = %d, want 1", len(mock.calls))
	}
	if len(mock.calls[0].Tools) != 0 {
		t.Fatalf("planner request sent %d tools; planning is a structured-output phase, not tool execution", len(mock.calls[0].Tools))
	}
	if mock.calls[0].ResponseFormat == nil || mock.calls[0].ResponseFormat.Type != llmcore.ResponseFormatJSONObject {
		t.Fatalf("response format = %+v, want json_object", mock.calls[0].ResponseFormat)
	}
}

func TestPlanner_CreatePlanRejectsNonJSONContent(t *testing.T) {
	mock := &mockLLM{responses: []*llmcore.LLMResponse{{Message: llmcore.Message{
		Role:        model.RoleAssistant,
		ContentText: "8",
	}}}}
	planner := NewPlannerAgent("s", DefaultAgentConfig(), mock, nil)
	_, reply, err := planner.CreatePlan(context.Background(), &TaskInput{Message: llmcore.Message{ContentText: "1+3+4等于多少"}})
	if err == nil || !strings.Contains(err.Error(), "解析计划失败") {
		t.Fatalf("err = %v, want planner parse error", err)
	}
	if reply != "8" {
		t.Fatalf("reply = %q, want raw model content", reply)
	}
}
