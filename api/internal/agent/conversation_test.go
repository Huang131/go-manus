package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/bytedance/sonic"
)

func TestConversationMessagesRestoresPersistedUserAndAssistantMessages(t *testing.T) {
	userData, _ := sonic.Marshal(model.NewMessageEvent(model.RoleUser, "previous question"))
	assistantData, _ := sonic.Marshal(model.NewMessageDoneEvent("message-1", "previous answer", "stop"))
	events := []model.Event{
		{Type: model.EventTypeMessage, Data: userData},
		{Type: model.EventTypePlan, Data: []byte(`{"status":"created"}`)},
		{Type: model.EventTypeMessageDone, Data: assistantData},
		{Type: model.EventTypeMessage, Data: []byte(`{"broken":`)},
	}

	got := conversationMessages(events)
	if len(got) != 2 {
		t.Fatalf("conversationMessages() length = %d, want 2", len(got))
	}
	if got[0].Role != model.RoleUser || got[0].ContentText != "previous question" {
		t.Fatalf("user message = %+v", got[0])
	}
	if got[1].Role != model.RoleAssistant || got[1].ContentText != "previous answer" {
		t.Fatalf("assistant message = %+v", got[1])
	}
}

func TestNewAgentTaskRunnerLoadsInitialConversation(t *testing.T) {
	history := []llmcore.Message{
		{Role: model.RoleUser, ContentText: "previous question"},
		{Role: model.RoleAssistant, ContentText: "previous answer"},
	}
	runner := NewAgentTaskRunner(&AgentTaskRunnerConfig{
		SessionID:       "session-1",
		AgentConfig:     DefaultAgentConfig(),
		InitialMessages: history,
	})

	plannerMessages := runner.flow.planner.memory.GetMessages()
	reactMessages := runner.flow.react.memory.GetMessages()
	if len(plannerMessages) != 2 || len(reactMessages) != 2 {
		t.Fatalf("initial history lengths planner=%d react=%d", len(plannerMessages), len(reactMessages))
	}
	history[0].ContentText = "mutated"
	if plannerMessages[0].ContentText != "previous question" || reactMessages[0].ContentText != "previous question" {
		t.Fatal("runner retained caller-owned history slice")
	}
}
