package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/mq"
	"github.com/bytedance/sonic"
)

// RunEventReader 是 HTTP/SSE 读取 Run 实时事件的最小边界。
// PostgreSQL 仍是 Run 终态事实来源，Redis 只提供短期事件和断线续读。
type RunEventReader interface {
	Read(ctx context.Context, runID, startID string) ([]*model.Event, error)
}

// ErrRunEventStreamExpired 表示 Last-Event-ID 已早于 Redis 当前保留窗口。
// Redis 事件不是最终事实，客户端收到该错误后必须查询 Run 快照并按当前状态重新订阅。
var ErrRunEventStreamExpired = errors.New("run event stream expired")

// ErrInvalidRunEventCursor 表示 Last-Event-ID 不是合法的 Redis Stream ID。
var ErrInvalidRunEventCursor = errors.New("invalid run event cursor")

// ValidateRunEventCursor 校验 SSE 续读游标；空值表示初次订阅。
func ValidateRunEventCursor(cursor string) error {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return nil
	}
	if _, _, ok := parseStreamID(cursor); !ok {
		return ErrInvalidRunEventCursor
	}
	return nil
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
	startID = strings.TrimSpace(startID)
	hasCursor := startID != ""
	if err := ValidateRunEventCursor(startID); err != nil {
		return nil, err
	}
	if startID == "" {
		startID = "0-0"
	}
	stream := RunEventStreamName(runID)
	if hasCursor {
		if boundsQueue, ok := s.queue.(mq.StreamBoundsQueue); ok {
			bounds, err := boundsQueue.GetStreamBounds(ctx, stream)
			if err != nil {
				return nil, fmt.Errorf("inspect run event stream: %w", err)
			}
			if !bounds.Exists || streamIDLess(startID, bounds.FirstID) || streamIDLess(bounds.LastID, startID) {
				return nil, ErrRunEventStreamExpired
			}
		}
	}
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

func streamIDLess(left, right string) bool {
	leftMillis, leftSequence, leftOK := parseStreamID(left)
	rightMillis, rightSequence, rightOK := parseStreamID(right)
	if !leftOK || !rightOK {
		return false
	}
	return leftMillis < rightMillis || (leftMillis == rightMillis && leftSequence < rightSequence)
}

func parseStreamID(value string) (int64, int64, bool) {
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	millis, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	sequence, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if millis < 0 || sequence < 0 {
		return 0, 0, false
	}
	return millis, sequence, true
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
