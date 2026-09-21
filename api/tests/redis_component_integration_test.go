//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/mq"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRedisQueue(t *testing.T) (*mq.RedisStreamMessageQueue, string) {
	t.Helper()
	require.NotNil(t, testApp.Redis)

	queue := mq.NewRedisStreamMessageQueue(testApp.Redis.Client)
	stream := "test:redis-component:" + uuid.NewString()
	t.Cleanup(func() {
		ctx, cancel := NewTestContext()
		defer cancel()
		if err := queue.Clear(ctx, stream); err != nil {
			t.Logf("清理 Redis stream 失败: %v", err)
		}
	})
	return queue, stream
}

func TestRedisComponent_PutAndGetBlockingRoundTrip(t *testing.T) {
	queue, stream := testRedisQueue(t)
	ctx, cancel := NewTestContext()
	defer cancel()

	want := map[string]any{"type": "message", "content": "hello"}
	id, err := queue.Put(ctx, stream, want)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	gotID, got, err := queue.GetBlocking(ctx, stream, "0-0")
	require.NoError(t, err)
	assert.Equal(t, id, gotID)
	assert.Equal(t, want, got)

	size, err := queue.Size(ctx, stream)
	require.NoError(t, err)
	assert.Equal(t, int64(1), size)
}

func TestRedisComponent_BatchReadUsesExclusiveCursor(t *testing.T) {
	queue, stream := testRedisQueue(t)
	ctx, cancel := NewTestContext()
	defer cancel()

	for i := 0; i < 3; i++ {
		_, err := queue.Put(ctx, stream, map[string]any{"index": i})
		require.NoError(t, err)
	}

	first, err := queue.GetBlockingBatch(ctx, stream, "0-0", 2)
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, float64(0), first[0].Data.(map[string]any)["index"])
	assert.Equal(t, float64(1), first[1].Data.(map[string]any)["index"])

	second, err := queue.GetBlockingBatch(ctx, stream, first[1].ID, 2)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Equal(t, float64(2), second[0].Data.(map[string]any)["index"])
}

func TestRedisComponent_GetBlockingHonorsCancellation(t *testing.T) {
	queue, stream := testRedisQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := queue.GetBlocking(ctx, stream, "$")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRedisComponent_RetentionAndClear(t *testing.T) {
	queue, stream := testRedisQueue(t)
	ctx, cancel := NewTestContext()
	defer cancel()

	_, err := queue.Put(ctx, stream, map[string]string{"value": "retained"})
	require.NoError(t, err)

	initialTTL, err := testApp.Redis.Client.TTL(ctx, stream).Result()
	require.NoError(t, err)
	assert.Greater(t, initialTTL, time.Duration(0))

	require.NoError(t, queue.SetRetention(ctx, stream, mq.CompletedStreamRetention()))
	completedTTL, err := testApp.Redis.Client.TTL(ctx, stream).Result()
	require.NoError(t, err)
	assert.Greater(t, completedTTL, time.Duration(0))
	assert.LessOrEqual(t, completedTTL, initialTTL)

	require.NoError(t, queue.Clear(ctx, stream))
	empty, err := queue.IsEmpty(ctx, stream)
	require.NoError(t, err)
	assert.True(t, empty)
}
