package service

import (
	"context"
	"fmt"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/mq"
	"github.com/bytedance/sonic"
)

// RunEventReader 是 HTTP/SSE 读取 Run 实时事件的最小边界。
// PostgreSQL 仍是 Run 终态事实来源，Redis 只提供短期事件和断线续读。
type RunEventReader interface {
	Read(ctx context.Context, runID, startID string) ([]*model.Event, error)
}

type redisRunEventStream struct {
	queue mq.TaskMessageQueue
}

// NewRedisRunEventStream 创建按 Run 隔离的 Redis Stream 读取器。
func NewRedisRunEventStream(queue mq.TaskMessageQueue) RunEventReader {
	return &redisRunEventStream{queue: queue}
}

func (s *redisRunEventStream) Read(ctx context.Context, runID, startID string) ([]*model.Event, error) {
	if s == nil || s.queue == nil {
		return nil, fmt.Errorf("run event stream queue is nil")
	}
	if runID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	if startID == "" {
		startID = "0-0"
	}
	stream := RunEventStreamName(runID)
	if batch, ok := s.queue.(mq.BatchMessageQueue); ok {
		messages, err := batch.GetBlockingBatch(ctx, stream, startID, 100, mq.DefaultBlockTimeout)
		if err != nil {
			return nil, err
		}
		return decodeRunEvents(messages)
	}
	id, raw, err := s.queue.GetBlocking(ctx, stream, startID, mq.DefaultBlockTimeout)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	return decodeRunEvents([]mq.StreamMessage{{ID: id, Data: raw}})
}

func decodeRunEvents(messages []mq.StreamMessage) ([]*model.Event, error) {
	events := make([]*model.Event, 0, len(messages))
	for _, message := range messages {
		if message.Data == nil {
			continue
		}
		data, err := sonic.Marshal(message.Data)
		if err != nil {
			return nil, fmt.Errorf("encode run event: %w", err)
		}
		var event model.Event
		if err := sonic.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("decode run event: %w", err)
		}
		event.ID = message.ID
		events = append(events, &event)
	}
	return events, nil
}
