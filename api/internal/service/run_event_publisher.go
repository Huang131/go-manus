package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/mq"
)

const runEventStreamPrefix = "run:"

// RunEventStreamName 返回一次 Run 的短期实时事件流名称。
func RunEventStreamName(runID string) string {
	return runEventStreamPrefix + strings.TrimSpace(runID) + ":events"
}

// RedisRunEventPublisher 将 Flow 事件写入按 Run 隔离的 Redis Stream。
// Redis 只提供实时事件和断线续读窗口，Run/Message 终态仍由 PostgreSQL 保存。
type RedisRunEventPublisher struct {
	queue mq.TaskMessageQueue
}

// NewRedisRunEventPublisher 创建 Redis Run 事件发布器。
func NewRedisRunEventPublisher(queue mq.TaskMessageQueue) *RedisRunEventPublisher {
	return &RedisRunEventPublisher{queue: queue}
}

// Publish 发布一个带事件类型和时间的 envelope；Redis Stream ID 由队列生成并作为续读游标。
func (p *RedisRunEventPublisher) Publish(ctx context.Context, runID string, event model.BaseEvent) error {
	if p == nil || p.queue == nil {
		return errors.New("run event publisher queue is nil")
	}
	if strings.TrimSpace(runID) == "" {
		return errors.New("run event publisher run id is empty")
	}
	if event == nil {
		return errors.New("run event publisher event is nil")
	}
	raw := event.ToJSON()
	if strings.TrimSpace(raw) == "" {
		return errors.New("run event publisher event payload is empty")
	}
	envelope := model.Event{
		Type:      event.GetType(),
		CreatedAt: time.Now().UTC(),
		Data:      []byte(raw),
	}
	if _, err := p.queue.Put(ctx, RunEventStreamName(runID), envelope); err != nil {
		return fmt.Errorf("publish run event: %w", err)
	}
	return nil
}

// MarkCompleted 缩短终态 Run 的事件流保留时间，保留断线续读窗口。
// waiting_input 不调用此方法，避免用户停留期间事件过早过期。
func (p *RedisRunEventPublisher) MarkCompleted(ctx context.Context, runID string) error {
	if p == nil || p.queue == nil {
		return errors.New("run event publisher queue is nil")
	}
	if strings.TrimSpace(runID) == "" {
		return errors.New("run event publisher run id is empty")
	}
	if err := p.queue.SetRetention(ctx, RunEventStreamName(runID), mq.CompletedStreamRetention()); err != nil {
		return fmt.Errorf("mark completed run event stream: %w", err)
	}
	return nil
}
