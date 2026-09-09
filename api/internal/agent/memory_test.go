package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
)

func TestSimpleMemory_Add(t *testing.T) {
	mem := NewSimpleMemory(100)
	msg := llmcore.Message{
		Role:        llmcore.RoleUser,
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
	mem := NewSimpleMemory(100)

	mem.Add(llmcore.Message{Role: llmcore.RoleUser, ContentText: "Hello"})
	mem.Add(llmcore.Message{Role: llmcore.RoleAssistant, ContentText: "Hi there"})

	messages := mem.GetMessages()
	if len(messages) != 2 {
		t.Errorf("GetMessages() got %d messages, want 2", len(messages))
	}
}

func TestSimpleMemory_Compact(t *testing.T) {
	mem := NewSimpleMemory(100)

	for i := 0; i < 20; i++ {
		mem.Add(llmcore.Message{Role: llmcore.RoleUser, ContentText: "Message"})
	}

	if err := mem.Compact(5); err != nil {
		t.Errorf("Compact() error = %v", err)
	}

	// Compact 应该保留最后 5 条消息
	messages := mem.GetMessages()
	if len(messages) != 5 {
		t.Errorf("Compact(5) should keep 5 messages, got %d", len(messages))
	}
}

func TestSimpleMemory_Clear(t *testing.T) {
	mem := NewSimpleMemory(100)

	mem.Add(llmcore.Message{Role: llmcore.RoleUser, ContentText: "Hello"})
	mem.Add(llmcore.Message{Role: llmcore.RoleAssistant, ContentText: "Hi"})

	mem.Clear()

	if len(mem.GetMessages()) != 0 {
		t.Error("Clear() should remove all messages")
	}
}
