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
	IsEmpty(ctx context.Context, streamName string) (bool, error)
	Size(ctx context.Context, streamName string) (int64, error)
}

// TaskMessageQueue 保留语义化别名，便于任务域表达依赖。
type TaskMessageQueue = MessageQueue
