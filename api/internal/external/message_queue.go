package external

import (
	"context"
	"time"
)

// MessageQueue 是任务链路实际使用的最小消息队列接口。
type MessageQueue interface {
	Put(ctx context.Context, streamName string, message interface{}) (string, error)
	GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error)
	Clear(ctx context.Context, streamName string) error
	SetRetention(ctx context.Context, streamName string, retention time.Duration) error
	IsEmpty(ctx context.Context, streamName string) (bool, error)
	Size(ctx context.Context, streamName string) (int64, error)
}

// StreamMessage 是消息队列返回的带游标消息。
// 游标必须保留 Redis Stream ID，SSE 客户端会用它继续读取后续事件。
type StreamMessage struct {
	ID   string
	Data interface{}
}

// BatchMessageQueue 是可选的批量读取能力。
// MessageQueue 保持最小接口，内存 mock 或其他实现可继续按单条读取工作。
type BatchMessageQueue interface {
	GetBlockingBatch(ctx context.Context, streamName string, startID string, count int, timeout ...time.Duration) ([]StreamMessage, error)
}

// TaskMessageQueue 保留语义化别名，便于任务域表达依赖。
type TaskMessageQueue = MessageQueue
