package agent

import (
	"context"
	"fmt"
	"sync"
)

// Task 任务接口
type Task interface {
	// InputStream 返回输入流
	InputStream() Stream
	// OutputStream 返回输出流
	OutputStream() Stream
}

// Stream 消息流接口
type Stream interface {
	// Put 放入消息
	Put(ctx context.Context, data string) (string, error)
	// Pop 取出消息
	Pop(ctx context.Context) (string, string, error)
	// IsEmpty 检查是否为空
	IsEmpty(ctx context.Context) (bool, error)
	// Len 返回队列长度
	Len(ctx context.Context) (int, error)
}

// MemoryStream 基于内存的消息流 (测试用)
type MemoryStream struct {
	mu       sync.Mutex
	messages []streamMessage
}

type streamMessage struct {
	id      string
	content string
}

// NewMemoryStream 创建内存流
func NewMemoryStream() *MemoryStream {
	return &MemoryStream{
		messages: make([]streamMessage, 0),
	}
}

func (s *MemoryStream) Put(ctx context.Context, data string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("msg_%d", len(s.messages)+1)
	s.messages = append(s.messages, streamMessage{
		id:      id,
		content: data,
	})
	return id, nil
}

func (s *MemoryStream) Pop(ctx context.Context) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.messages) == 0 {
		return "", "", nil
	}

	msg := s.messages[0]
	s.messages = s.messages[1:]
	return msg.id, msg.content, nil
}

func (s *MemoryStream) IsEmpty(ctx context.Context) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages) == 0, nil
}

func (s *MemoryStream) Len(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages), nil
}

// TaskImpl 任务实现
type TaskImpl struct {
	inputStream  Stream
	outputStream Stream
}

// NewTask 创建任务
func NewTask(input, output Stream) *TaskImpl {
	return &TaskImpl{
		inputStream:  input,
		outputStream: output,
	}
}

func (t *TaskImpl) InputStream() Stream {
	return t.inputStream
}

func (t *TaskImpl) OutputStream() Stream {
	return t.outputStream
}

// SimpleTask 简单任务实现
type SimpleTask struct {
	*TaskImpl
}

func NewSimpleTask() *SimpleTask {
	return &SimpleTask{
		TaskImpl: NewTask(NewMemoryStream(), NewMemoryStream()),
	}
}
