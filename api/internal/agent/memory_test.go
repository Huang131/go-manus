package agent

import (
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestSimpleMemory_Add(t *testing.T) {
	mem := NewSimpleMemory(100)
	msg := &model.Message{
		Role:    "user",
		Message: "Hello, world!",
	}

	if err := mem.Add(msg); err != nil {
		t.Errorf("Add() error = %v", err)
	}

	if mem.Size() == 0 {
		t.Error("Add() did not add message")
	}
}

func TestSimpleMemory_GetMessages(t *testing.T) {
	mem := NewSimpleMemory(100)

	msg1 := &model.Message{Role: "user", Message: "Hello"}
	msg2 := &model.Message{Role: "assistant", Message: "Hi there"}

	mem.Add(msg1)
	mem.Add(msg2)

	messages := mem.GetMessages()
	if len(messages) != 2 {
		t.Errorf("GetMessages() got %d messages, want 2", len(messages))
	}
}

func TestSimpleMemory_GetLastN(t *testing.T) {
	mem := NewSimpleMemory(100)

	for i := 0; i < 10; i++ {
		mem.Add(&model.Message{Role: "user", Message: "Message"})
	}

	last3 := mem.GetLastN(3)
	if len(last3) != 3 {
		t.Errorf("GetLastN(3) got %d messages, want 3", len(last3))
	}
}

func TestSimpleMemory_Compact(t *testing.T) {
	mem := NewSimpleMemory(100)

	for i := 0; i < 20; i++ {
		mem.Add(&model.Message{Role: "user", Message: "Message"})
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

	mem.Add(&model.Message{Role: "user", Message: "Hello"})
	mem.Add(&model.Message{Role: "assistant", Message: "Hi"})

	mem.Clear()

	if mem.Size() != 0 {
		t.Error("Clear() should remove all messages")
	}
}
