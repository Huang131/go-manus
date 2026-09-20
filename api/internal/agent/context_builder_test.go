package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestContextBuilderKeepsToolCallAndResultTogether(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{MaxInputTokens: 50})
	history := []llmcore.Message{
		{Role: model.RoleUser, ContentText: "old message that should be dropped because it consumes the available context budget"},
		{Role: model.RoleAssistant, ToolCalls: []llmcore.ToolCall{{ID: "call-1", Type: "function", Function: llmcore.ToolCallFunction{Name: "search"}}}},
		{Role: model.RoleTool, ToolCallID: "call-1", ContentText: "tool result"},
	}

	got, err := builder.Build("system", history, "current")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("Build() messages = %+v, want system + tool pair + current", got)
	}
	if len(got[1].ToolCalls) != 1 || got[2].ToolCallID != "call-1" {
		t.Fatalf("tool pair was split: %+v", got)
	}
}

func TestContextBuilderDropsIncompleteToolGroup(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{MaxInputTokens: 100})
	history := []llmcore.Message{{
		Role:      model.RoleAssistant,
		ToolCalls: []llmcore.ToolCall{{ID: "call-1"}},
	}}

	got, err := builder.Build("system", history, "current")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 2 || got[0].Role != model.RoleSystem || got[1].Role != model.RoleUser {
		t.Fatalf("Build() retained incomplete tool group: %+v", got)
	}
}

func TestContextBuilderRejectsMinimumContextOverBudget(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{MaxInputTokens: 1})
	if _, err := builder.Build("system", nil, "current"); err == nil {
		t.Fatal("Build() error = nil, want context limit error")
	}
}
