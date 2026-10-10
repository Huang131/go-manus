package service

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
)

type runEventQueueStub struct {
	stream string
	value  interface{}
}

func (q *runEventQueueStub) Put(_ context.Context, stream string, value interface{}) (string, error) {
	q.stream, q.value = stream, value
	return "1-0", nil
}
func (*runEventQueueStub) GetBlocking(context.Context, string, string, ...time.Duration) (string, interface{}, error) {
	return "", nil, nil
}
func (*runEventQueueStub) Clear(context.Context, string) error                       { return nil }
func (*runEventQueueStub) SetRetention(context.Context, string, time.Duration) error { return nil }
func (*runEventQueueStub) IsEmpty(context.Context, string) (bool, error)             { return true, nil }
func (*runEventQueueStub) Size(context.Context, string) (int64, error)               { return 0, nil }

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
