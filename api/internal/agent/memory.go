package agent

import (
	"sync"

	"github.com/Huang131/go-manus/api/internal/llmcore"
)

// Memory 记忆接口
type Memory interface {
	// Add 添加消息到记忆
	Add(msg llmcore.Message) error
	// MergeMessages 批量合并消息到记忆（用于把一轮 ReAct 循环的完整对话写入记忆）
	MergeMessages(msgs []llmcore.Message) error
	// GetMessages 获取消息列表
	GetMessages() []llmcore.Message
	// Clear 清空记忆
	Clear()
}

// SimpleMemory 简单记忆实现
type SimpleMemory struct {
	mu       sync.RWMutex
	messages []llmcore.Message
}

// NewSimpleMemory 创建简单记忆
func NewSimpleMemory() *SimpleMemory {
	return &SimpleMemory{
		messages: make([]llmcore.Message, 0),
	}
}

func (m *SimpleMemory) Add(msg llmcore.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *SimpleMemory) MergeMessages(msgs []llmcore.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msgs...)
	return nil
}

func (m *SimpleMemory) GetMessages() []llmcore.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]llmcore.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *SimpleMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]llmcore.Message, 0)
}
