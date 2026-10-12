package service

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/mq"
	"github.com/stretchr/testify/require"
)

type runEventStreamQueueStub struct {
	bounds mq.StreamBounds
	batch  []mq.StreamMessage
}

func (q *runEventStreamQueueStub) Put(context.Context, string, interface{}) (string, error) {
	return "1-0", nil
}
func (q *runEventStreamQueueStub) GetBlocking(context.Context, string, string, ...time.Duration) (string, interface{}, error) {
	return "", nil, nil
}
func (q *runEventStreamQueueStub) Clear(context.Context, string) error { return nil }
func (q *runEventStreamQueueStub) SetRetention(context.Context, string, time.Duration) error {
	return nil
}
func (q *runEventStreamQueueStub) IsEmpty(context.Context, string) (bool, error) { return false, nil }
func (q *runEventStreamQueueStub) Size(context.Context, string) (int64, error) {
	return int64(len(q.batch)), nil
}
func (q *runEventStreamQueueStub) GetStreamBounds(context.Context, string) (mq.StreamBounds, error) {
	return q.bounds, nil
}
func (q *runEventStreamQueueStub) GetBlockingBatch(context.Context, string, string, int, ...time.Duration) ([]mq.StreamMessage, error) {
	return q.batch, nil
}

func TestRedisRunEventStreamReturnsExpiredForCursorBeforeRetentionWindow(t *testing.T) {
	queue := &runEventStreamQueueStub{bounds: mq.StreamBounds{
		Exists: true, FirstID: "200-0", LastID: "201-0",
	}}
	reader := NewRedisRunEventStream(queue)

	_, err := reader.Read(context.Background(), "run-1", "199-9")

	require.ErrorIs(t, err, ErrRunEventStreamExpired)
}

func TestRedisRunEventStreamKeepsCursorAtRetentionBoundary(t *testing.T) {
	queue := &runEventStreamQueueStub{
		bounds: mq.StreamBounds{Exists: true, FirstID: "200-0", LastID: "201-0"},
		batch:  []mq.StreamMessage{{ID: "201-0", Data: map[string]interface{}{"type": "done"}}},
	}
	reader := NewRedisRunEventStream(queue)

	events, err := reader.Read(context.Background(), "run-1", "200-0")

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "201-0", events[0].ID)
}

func TestRedisRunEventStreamDoesNotInspectBoundsForInitialSubscription(t *testing.T) {
	queue := &runEventStreamQueueStub{batch: []mq.StreamMessage{{ID: "200-0", Data: map[string]interface{}{"type": "done"}}}}
	reader := NewRedisRunEventStream(queue)

	events, err := reader.Read(context.Background(), "run-1", "")

	require.NoError(t, err)
	require.Len(t, events, 1)
}
