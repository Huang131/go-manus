package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

type fixedTokenEstimator struct {
	messageTokens map[string]int
	toolTokens    int
}

func (e fixedTokenEstimator) EstimateMessage(message llmcore.Message) (int, error) {
	return e.messageTokens[message.ContentText], nil
}

func (e fixedTokenEstimator) EstimateTools([]llmcore.ToolSpec) (int, error) {
	return e.toolTokens, nil
}

func TestContextBuilderKeepsToolCallAndResultTogether(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{
		MaxContextTokens: 10,
		Estimator: fixedTokenEstimator{messageTokens: map[string]int{
			"system": 1,
			"old message that should be dropped because it consumes the available context budget": 8,
			"tool result": 2,
			"current":     1,
		}},
	})
	history := []llmcore.Message{
		{Role: model.RoleUser, ContentText: "old message that should be dropped because it consumes the available context budget"},
		{Role: model.RoleAssistant, ToolCalls: []llmcore.ToolCall{{ID: "call-1", Type: "function", Function: llmcore.ToolCallFunction{Name: "search"}}}},
		{Role: model.RoleTool, ToolCallID: "call-1", ContentText: "tool result"},
	}

	got, err := builder.Build(ContextRequest{SystemPrompt: "system", History: history, CurrentQuery: "current"})
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
	builder := NewContextBuilder(ContextPolicy{MaxContextTokens: 100})
	history := []llmcore.Message{{
		Role:      model.RoleAssistant,
		ToolCalls: []llmcore.ToolCall{{ID: "call-1"}},
	}}

	got, err := builder.Build(ContextRequest{SystemPrompt: "system", History: history, CurrentQuery: "current"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 2 || got[0].Role != model.RoleSystem || got[1].Role != model.RoleUser {
		t.Fatalf("Build() retained incomplete tool group: %+v", got)
	}
}

func TestContextBuilderRejectsMinimumContextOverBudget(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{MaxContextTokens: 1})
	if _, err := builder.Build(ContextRequest{SystemPrompt: "system", CurrentQuery: "current"}); err == nil {
		t.Fatal("Build() error = nil, want context limit error")
	}
}

func TestContextBuilderReservesOutputTokensBeforeSelectingHistory(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{
		MaxContextTokens:    10,
		OutputReserveTokens: 4,
		Estimator: fixedTokenEstimator{messageTokens: map[string]int{
			"system":  2,
			"old":     3,
			"current": 2,
		}},
	})

	got, err := builder.Build(ContextRequest{
		SystemPrompt: "system",
		History:      []llmcore.Message{{Role: model.RoleUser, ContentText: "old"}},
		CurrentQuery: "current",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Build() messages = %+v, want only required system and current messages", got)
	}
}

func TestContextBuilderCountsToolSchemasInTheSharedInputBudget(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{
		MaxContextTokens:    12,
		OutputReserveTokens: 2,
		Estimator: fixedTokenEstimator{
			messageTokens: map[string]int{
				"system":  2,
				"old":     4,
				"current": 2,
			},
			toolTokens: 3,
		},
	})

	got, err := builder.Build(ContextRequest{
		SystemPrompt: "system",
		History:      []llmcore.Message{{Role: model.RoleUser, ContentText: "old"}},
		CurrentQuery: "current",
		Tools:        []llmcore.ToolSpec{{Type: llmcore.ToolTypeFunction}},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Build() messages = %+v, want tool budget to exclude old history", got)
	}
}

func TestContextBuilderCompactKeepsCompleteToolPair(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{
		MaxContextTokens: 8,
		Estimator: fixedTokenEstimator{messageTokens: map[string]int{
			"system":       1,
			"old":          4,
			"tool request": 2,
			"tool result":  2,
		}},
	})

	messages := []llmcore.Message{
		{Role: model.RoleSystem, ContentText: "system"},
		{Role: model.RoleAssistant, ContentText: "old"},
		{Role: model.RoleAssistant, ContentText: "tool request", ToolCalls: []llmcore.ToolCall{{ID: "call-1"}}},
		{Role: model.RoleTool, ToolCallID: "call-1", ContentText: "tool result"},
	}

	got, err := builder.Compact(messages, nil)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}
	if len(got) != 3 || len(got[1].ToolCalls) != 1 || got[2].ToolCallID != "call-1" {
		t.Fatalf("Compact() split or dropped the tool pair: %+v", got)
	}
}

func TestContextBuilderCompactKeepsUserBeforeItsToolPair(t *testing.T) {
	builder := NewContextBuilder(ContextPolicy{
		MaxContextTokens: 20,
		Estimator: fixedTokenEstimator{messageTokens: map[string]int{
			"system":        1,
			"current query": 2,
			"tool request":  2,
			"tool result":   2,
		}},
	})

	messages := []llmcore.Message{
		{Role: model.RoleSystem, ContentText: "system"},
		{Role: model.RoleUser, ContentText: "current query"},
		{Role: model.RoleAssistant, ContentText: "tool request", ToolCalls: []llmcore.ToolCall{{ID: "call-1"}}},
		{Role: model.RoleTool, ToolCallID: "call-1", ContentText: "tool result"},
	}

	got, err := builder.Compact(messages, nil)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}
	if len(got) != len(messages) {
		t.Fatalf("Compact() messages = %+v, want all messages", got)
	}
	if got[1].Role != model.RoleUser || got[1].ContentText != "current query" {
		t.Fatalf("Compact() moved user message after tool messages: %+v", got)
	}
	if len(got[2].ToolCalls) != 1 || got[3].ToolCallID != "call-1" {
		t.Fatalf("Compact() split tool pair: %+v", got)
	}
}
