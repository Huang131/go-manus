package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestSimpleMemory_AppendAndMergePreserveOrder(t *testing.T) {
	mem := NewSimpleMemory()

	if err := mem.Add(llmcore.Message{Role: model.RoleUser, ContentText: "Hello"}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := mem.MergeMessages([]llmcore.Message{
		{Role: model.RoleAssistant, ContentText: "Hi there"},
		{Role: model.RoleUser, ContentText: "Continue"},
	}); err != nil {
		t.Fatalf("MergeMessages() error = %v", err)
	}

	messages := mem.GetMessages()
	if len(messages) != 3 {
		t.Fatalf("GetMessages() got %d messages, want 3", len(messages))
	}
	assertMessage := func(index int, role model.MessageRole, content string) {
		t.Helper()
		if messages[index].Role != role || messages[index].ContentText != content {
			t.Errorf("message[%d] = (%s, %q), want (%s, %q)", index, messages[index].Role, messages[index].ContentText, role, content)
		}
	}
	assertMessage(0, model.RoleUser, "Hello")
	assertMessage(1, model.RoleAssistant, "Hi there")
	assertMessage(2, model.RoleUser, "Continue")
}

func TestSimpleMemory_Clear(t *testing.T) {
	mem := NewSimpleMemory()

	mem.Add(llmcore.Message{Role: model.RoleUser, ContentText: "Hello"})
	mem.Add(llmcore.Message{Role: model.RoleAssistant, ContentText: "Hi"})

	mem.Clear()

	if len(mem.GetMessages()) != 0 {
		t.Error("Clear() should remove all messages")
	}
}
