package mq

import (
	"context"
	"time"
)

// MessageQueue 是 Run 事件发布所需的最小写入接口。
type MessageQueue interface {
	Put(ctx context.Context, streamName string, message interface{}) (string, error)
	SetRetention(ctx context.Context, streamName string, retention time.Duration) error
}

// StreamReader 是 Run 事件流读取所需的最小读取接口。
type StreamReader interface {
	GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error)
}

// StreamMessage 是消息队列返回的带游标消息。
// 游标必须保留 Redis Stream ID，SSE 客户端会用它继续读取后续事件。
type StreamMessage struct {
	ID   string
	Data interface{}
}

// StreamBounds 描述 Redis Stream 当前保留窗口的首尾游标。
// 空流以 Exists=false 表示；这与普通的 Size=0 查询不同，调用方可以据此判断游标是否已经过期。
type StreamBounds struct {
	Exists  bool
	FirstID string
	LastID  string
}

// StreamBoundsQueue 是消息队列的可选游标边界能力。
// 保持为可选接口，避免把 Redis 专有的保留窗口语义强加给所有队列实现。
type StreamBoundsQueue interface {
	GetStreamBounds(ctx context.Context, streamName string) (StreamBounds, error)
}

// BatchMessageQueue 是可选的批量读取能力。
// 生产 Redis 实现使用批量读取降低 SSE 事件流的往返次数。
type BatchMessageQueue interface {
	GetBlockingBatch(ctx context.Context, streamName string, startID string, count int, timeout ...time.Duration) ([]StreamMessage, error)
}
