package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

// ErrRunTracePersistence 标识关键回放事件无法持久化。
// Adapter 据此将 Run 收敛为失败，避免成功状态掩盖不可追溯的执行过程。
var ErrRunTracePersistence = errors.New("run trace persistence failed")

// RunTraceStore 是长期回放事件的最小持久化边界。
type RunTraceStore interface {
	Append(ctx context.Context, runID string, event *model.RunTraceEvent) error
}

// RunTraceEventPublisher 先写入可回放事件，再尽力向短期实时流发送原始事件。
type RunTraceEventPublisher struct {
	store RunTraceStore
	redis RunEventPublisher
}

// NewRunTraceEventPublisher 创建同时服务长期回放与 Redis 实时消费的发布器。
func NewRunTraceEventPublisher(store RunTraceStore, redis RunEventPublisher) *RunTraceEventPublisher {
	return &RunTraceEventPublisher{store: store, redis: redis}
}

// Publish 对白名单事件先持久化；Redis 是旁路，失败只记录告警。
func (p *RunTraceEventPublisher) Publish(ctx context.Context, runID string, event model.BaseEvent) error {
	if p == nil {
		return errors.New("run trace event publisher is nil")
	}
	if event == nil {
		return errors.New("run trace event is nil")
	}
	if model.IsPersistableRunTraceEvent(event.GetType()) {
		if err := p.appendTrace(ctx, runID, event); err != nil {
			return err
		}
	}
	if p.redis == nil {
		return nil
	}
	if err := p.redis.Publish(ctx, runID, event); err != nil {
		logger.WarnContext(ctx, "发布 Run Redis 实时事件失败", logger.String("run_id", runID), logger.Err(err))
	}
	return nil
}

// MarkCompleted 保持复合发布器的 Redis 流保留期语义。
func (p *RunTraceEventPublisher) MarkCompleted(ctx context.Context, runID string) error {
	if p == nil || p.redis == nil {
		return nil
	}
	marker, ok := p.redis.(completedRunEventMarker)
	if !ok {
		return nil
	}
	return marker.MarkCompleted(ctx, runID)
}

func (p *RunTraceEventPublisher) appendTrace(ctx context.Context, runID string, event model.BaseEvent) error {
	if p.store == nil {
		return fmt.Errorf("%w: trace store is nil", ErrRunTracePersistence)
	}
	payload, err := model.SafeRunTracePayload(event)
	if err != nil {
		return fmt.Errorf("%w: build run trace payload: %v", ErrRunTracePersistence, err)
	}
	trace := &model.RunTraceEvent{
		RunID:     runID,
		Type:      event.GetType(),
		Data:      payload,
		CreatedAt: time.Now().UTC(),
	}
	if err := p.store.Append(ctx, runID, trace); err != nil {
		return fmt.Errorf("%w: append run trace event: %w", ErrRunTracePersistence, err)
	}
	return nil
}
