package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestSimpleMemory_Add(t *testing.T) {
	mem := NewSimpleMemory()
	msg := llmcore.Message{
		Role:        model.RoleUser,
		ContentText: "Hello, world!",
	}

	if err := mem.Add(msg); err != nil {
		t.Errorf("Add() error = %v", err)
	}

	if len(mem.GetMessages()) == 0 {
		t.Error("Add() did not add message")
	}
}

func TestSimpleMemory_GetMessages(t *testing.T) {
	mem := NewSimpleMemory()

	mem.Add(llmcore.Message{Role: model.RoleUser, ContentText: "Hello"})
	mem.Add(llmcore.Message{Role: model.RoleAssistant, ContentText: "Hi there"})

	messages := mem.GetMessages()
	if len(messages) != 2 {
		t.Errorf("GetMessages() got %d messages, want 2", len(messages))
	}
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
