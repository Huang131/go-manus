package external

import (
	"context"
	"testing"
	"time"
)

func TestMessageQueueInterface(t *testing.T) {
	var _ MessageQueue = (*RedisStreamMessageQueue)(nil)
}

type mockMQ struct{}

func (m *mockMQ) Put(context.Context, string, interface{}) (string, error) {
	return "1234567890-0", nil
}

func (m *mockMQ) GetBlocking(ctx context.Context, _ string, _ string, _ ...time.Duration) (string, interface{}, error) {
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
		return "1234567890-0", "test message", nil
	}
}

func (m *mockMQ) Clear(context.Context, string) error {
	return nil
}

func (m *mockMQ) IsEmpty(context.Context, string) (bool, error) {
	return true, nil
}

func (m *mockMQ) Size(context.Context, string) (int64, error) {
	return 0, nil
}

func TestMessageQueueMockOperations(t *testing.T) {
	mq := &mockMQ{}
	ctx := context.Background()

	id, err := mq.Put(ctx, "test-stream", map[string]string{"key": "value"})
	if err != nil || id == "" {
		t.Fatalf("Put() = (%q, %v)", id, err)
	}

	if empty, err := mq.IsEmpty(ctx, "test-stream"); err != nil || !empty {
		t.Fatalf("IsEmpty() = (%v, %v), want (true, nil)", empty, err)
	}

	if size, err := mq.Size(ctx, "test-stream"); err != nil || size != 0 {
		t.Fatalf("Size() = (%d, %v), want (0, nil)", size, err)
	}

	if err := mq.Clear(ctx, "test-stream"); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
}

func TestMessageQueueGetBlockingHonorsContext(t *testing.T) {
	mq := &mockMQ{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := mq.GetBlocking(ctx, "test-stream", "0", time.Second)
	if err != context.Canceled {
		t.Fatalf("GetBlocking() error = %v, want context.Canceled", err)
	}
}

func TestMessageQueueDefaultTimeouts(t *testing.T) {
	if defaultBlockTimeout != 3*time.Second {
		t.Fatalf("default block timeout = %s, want 3s", defaultBlockTimeout)
	}
	if maxBlockTimeout != 5*time.Second {
		t.Fatalf("max block timeout = %s, want 5s", maxBlockTimeout)
	}
	if streamMaxLen <= 0 {
		t.Fatalf("stream max len = %d, want positive", streamMaxLen)
	}
	if streamRetention <= 0 {
		t.Fatalf("stream retention = %s, want positive", streamRetention)
	}
}
