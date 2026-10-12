package service

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/mq"
)

type runEventQueueStub struct {
	stream          string
	value           interface{}
	retentionStream string
	retention       time.Duration
}

func (q *runEventQueueStub) Put(_ context.Context, stream string, value interface{}) (string, error) {
	q.stream, q.value = stream, value
	return "1-0", nil
}
func (*runEventQueueStub) GetBlocking(context.Context, string, string, ...time.Duration) (string, interface{}, error) {
	return "", nil, nil
}
func (*runEventQueueStub) Clear(context.Context, string) error { return nil }
func (q *runEventQueueStub) SetRetention(_ context.Context, stream string, retention time.Duration) error {
	q.retentionStream, q.retention = stream, retention
	return nil
}
func (*runEventQueueStub) IsEmpty(context.Context, string) (bool, error) { return true, nil }
func (*runEventQueueStub) Size(context.Context, string) (int64, error)   { return 0, nil }

func TestRedisRunEventPublisherPublishesRunEnvelope(t *testing.T) {
	queue := &runEventQueueStub{}
	publisher := NewRedisRunEventPublisher(queue)
	event := model.NewMessageEvent(model.RoleAssistant, "hello")
	if err := publisher.Publish(context.Background(), "run-1", event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if queue.stream != "run:run-1:events" {
		t.Fatalf("stream = %q", queue.stream)
	}
	envelope, ok := queue.value.(model.Event)
	if !ok {
		t.Fatalf("published value = %T, want model.Event", queue.value)
	}
	if envelope.Type != model.EventTypeMessage || string(envelope.Data) == "" {
		t.Fatalf("envelope = %+v", envelope)
	}
}

func TestRedisRunEventPublisherMarksCompletedStream(t *testing.T) {
	queue := &runEventQueueStub{}
	publisher := NewRedisRunEventPublisher(queue)
	if err := publisher.MarkCompleted(context.Background(), "run-1"); err != nil {
		t.Fatalf("MarkCompleted() error = %v", err)
	}
	if queue.retentionStream != "run:run-1:events" {
		t.Fatalf("retention stream = %q", queue.retentionStream)
	}
	if queue.retention != mq.CompletedStreamRetention() {
		t.Fatalf("retention = %s, want %s", queue.retention, mq.CompletedStreamRetention())
	}
}
