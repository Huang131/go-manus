package external

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestRedisStreamMessageQueue_Put 测试添加消息
func TestRedisStreamMessageQueue_Put(t *testing.T) {
	// 需要实际的 Redis 连接才能测试，这里只测试接口
	t.Skip("需要 Redis 服务器才能运行")
}

// TestRedisStreamMessageQueue_Get 测试获取消息
func TestRedisStreamMessageQueue_Get(t *testing.T) {
	t.Skip("需要 Redis 服务器才能运行")
}

// TestRedisStreamMessageQueue_Pop 测试弹出消息
func TestRedisStreamMessageQueue_Pop(t *testing.T) {
	t.Skip("需要 Redis 服务器才能运行")
}

// TestRedisStreamMessageQueue_Size 测试获取队列大小
func TestRedisStreamMessageQueue_Size(t *testing.T) {
	t.Skip("需要 Redis 服务器才能运行")
}

// TestMessageQueue_Interface 测试接口实现
func TestMessageQueue_Interface(t *testing.T) {
	// 确保 RedisStreamMessageQueue 实现了 MessageQueue 接口
	var _ MessageQueue = (*RedisStreamMessageQueue)(nil)
}

// TestMessageQueue_Operations 测试基本操作流程
func TestMessageQueue_Operations(t *testing.T) {
	t.Skip("需要 Redis 服务器才能运行")
}

// mockMQ 用于测试的 Mock 消息队列
type mockMQ struct{}

func (m *mockMQ) Put(ctx context.Context, streamName string, message interface{}) (string, error) {
	return "1234567890-0", nil
}

func (m *mockMQ) Get(ctx context.Context, streamName string, startID string, blockMs *int) (string, interface{}, error) {
	return "1234567890-0", "test message", nil
}

func (m *mockMQ) GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error) {
	// 模拟阻塞获取
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return "1234567890-0", "test message", nil
	}
}

func (m *mockMQ) Pop(ctx context.Context, streamName string) (string, interface{}, error) {
	return "1234567890-0", "test message", nil
}

func (m *mockMQ) Clear(ctx context.Context, streamName string) error {
	return nil
}

func (m *mockMQ) IsEmpty(ctx context.Context, streamName string) (bool, error) {
	return true, nil
}

func (m *mockMQ) Size(ctx context.Context, streamName string) (int64, error) {
	return 0, nil
}

func (m *mockMQ) DeleteMessage(ctx context.Context, streamName string, messageID string) error {
	return nil
}

func (m *mockMQ) GetRange(ctx context.Context, streamName string, startID, endID string, limit int64) ([]*Message, error) {
	return []*Message{
		{ID: "1234567890-0", Data: "test message", Stream: streamName},
	}, nil
}

func (m *mockMQ) GetLatestID(ctx context.Context, streamName string) (string, error) {
	return "1234567890-0", nil
}

func (m *mockMQ) Subscribe(ctx context.Context, streamName string, bufferSize int) (<-chan *Message, func()) {
	msgChan := make(chan *Message, bufferSize)
	stopped := make(chan struct{})
	var closeOnce sync.Once

	closeChan := func() {
		closeOnce.Do(func() {
			close(msgChan)
		})
	}

	// 取消订阅：仅发出停止信号，由 goroutine 统一负责关闭 channel，
	// 避免 "goroutine 发送中" 与 "取消方 close" 之间的 send-on-closed 竞态。
	cancel := func() {
		select {
		case <-stopped:
			// 已停止
		default:
			close(stopped)
		}
	}

	// 模拟一条消息的投递，然后阻塞直到订阅被取消
	go func() {
		defer closeChan()

		msg := &Message{
			ID:     "test-id",
			Data:   "test message",
			Stream: streamName,
		}

		// 发送消息：select 在 stopped/ctx.Done 时优先退出，避免向已关闭 channel 发送
		select {
		case <-ctx.Done():
			return
		case <-stopped:
			return
		case msgChan <- msg:
		}

		// 发送完成后等待停止信号，再退出（defer 关闭 channel）
		select {
		case <-ctx.Done():
		case <-stopped:
		}
	}()

	return msgChan, cancel
}

func (m *mockMQ) Close() error {
	return nil
}

// TestMessageQueue_Mock 测试 Mock 实现
func TestMessageQueue_Mock(t *testing.T) {
	mq := &mockMQ{}

	// 测试 Put
	id, err := mq.Put(context.Background(), "test-stream", map[string]string{"key": "value"})
	if err != nil {
		t.Errorf("Put failed: %v", err)
	}
	if id == "" {
		t.Error("Put returned empty ID")
	}

	// 测试 Get
	_, msg, err := mq.Get(context.Background(), "test-stream", "", nil)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if msg == nil {
		t.Error("Get returned nil message")
	}

	// 测试 IsEmpty
	empty, err := mq.IsEmpty(context.Background(), "test-stream")
	if err != nil {
		t.Errorf("IsEmpty failed: %v", err)
	}
	if !empty {
		t.Error("IsEmpty should return true for empty queue")
	}

	// 测试 Size
	size, err := mq.Size(context.Background(), "test-stream")
	if err != nil {
		t.Errorf("Size failed: %v", err)
	}
	if size != 0 {
		t.Errorf("Size should be 0, got %d", size)
	}

	// 测试 Pop
	_, _, err = mq.Pop(context.Background(), "test-stream")
	if err != nil {
		t.Errorf("Pop failed: %v", err)
	}

	// 测试 Clear
	err = mq.Clear(context.Background(), "test-stream")
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}

	// 测试 DeleteMessage
	err = mq.DeleteMessage(context.Background(), "test-stream", "1234567890-0")
	if err != nil {
		t.Errorf("DeleteMessage failed: %v", err)
	}

	// 测试 Close
	err = mq.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestMessageQueue_WithBlocking 测试阻塞获取
func TestMessageQueue_WithBlocking(t *testing.T) {
	mq := &mockMQ{}

	blockMs := 1000
	_, _, err := mq.Get(context.Background(), "test-stream", "$", &blockMs)
	if err != nil {
		t.Errorf("Get with blocking failed: %v", err)
	}
}

// TestMessageQueue_DataTypes 测试不同数据类型
func TestMessageQueue_DataTypes(t *testing.T) {
	mq := &mockMQ{}

	testCases := []struct {
		name    string
		message interface{}
	}{
		{"string", "hello"},
		{"int", 123},
		{"map", map[string]interface{}{"key": "value"}},
		{"array", []int{1, 2, 3}},
		{"struct", struct{ Name string }{Name: "test"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mq.Put(context.Background(), "test-stream", tc.message)
			if err != nil {
				t.Errorf("Put failed for %s: %v", tc.name, err)
			}
		})
	}
}

// TestRedisStreamMessageQueue_AcquireLock 测试分布式锁
func TestRedisStreamMessageQueue_AcquireLock(t *testing.T) {
	t.Skip("需要 Redis 服务器才能运行")

	// 这个测试需要实际的 Redis 连接
	// 可以使用 miniredis 或真实的 Redis 服务器进行测试
}

// TestContextTimeout 测试上下文超时
func TestContextTimeout(t *testing.T) {
	mq := &mockMQ{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 快速完成的操作应该不受影响
	_, err := mq.Put(ctx, "test-stream", "test")
	if err != nil {
		t.Errorf("Put with timeout context failed: %v", err)
	}
}

// TestMessageQueue_Subscribe 测试消息订阅
func TestMessageQueue_Subscribe(t *testing.T) {
	mq := &mockMQ{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 订阅消息
	msgChan, unsubscribe := mq.Subscribe(ctx, "test-stream", 10)

	// 接收消息
	select {
	case <-ctx.Done():
		t.Error("Context cancelled unexpectedly")
	case msg := <-msgChan:
		if msg == nil {
			t.Error("Received nil message")
		}
		if msg.ID != "test-id" {
			t.Errorf("Expected message ID 'test-id', got '%s'", msg.ID)
		}
	}

	// 取消订阅
	unsubscribe()

	// 确认 channel 已关闭
	_, ok := <-msgChan
	if ok {
		t.Error("Channel should be closed after unsubscribe")
	}
}

// TestMessageQueue_SubscribeWithCancel 测试取消订阅
func TestMessageQueue_SubscribeWithCancel(t *testing.T) {
	mq := &mockMQ{}
	ctx, cancel := context.WithCancel(context.Background())

	// 订阅消息
	msgChan, unsubscribe := mq.Subscribe(ctx, "test-stream", 10)

	// 立即取消
	cancel()
	unsubscribe()

	// 等待一小段时间确保 goroutine 退出
	time.Sleep(50 * time.Millisecond)

	// 先排空 channel 中可能残留的消息，再确认 channel 已关闭。
	// mock 的语义是：cancel/unsubscribe 不会丢弃已入 buffer 的消息，
	// 因此 channel 关闭前 buffer 里可能还有一条测试消息。
	for {
		_, ok := <-msgChan
		if !ok {
			break
		}
	}
}

// TestGetBlocking_Basic 测试 GetBlocking 基本功能
func TestGetBlocking_Basic(t *testing.T) {
	mq := &mockMQ{}

	// 测试基本获取
	id, data, err := mq.GetBlocking(context.Background(), "test-stream", "", 100*time.Millisecond)
	if err != nil {
		t.Errorf("GetBlocking failed: %v", err)
	}
	if id == "" {
		t.Error("GetBlocking returned empty ID")
	}
	if data == nil {
		t.Error("GetBlocking returned nil data")
	}
}

// TestGetBlocking_WithStartID 测试带 startID 的获取
func TestGetBlocking_WithStartID(t *testing.T) {
	mq := &mockMQ{}

	// 测试带 startID 获取
	id, _, err := mq.GetBlocking(context.Background(), "test-stream", "1234567890-0", 100*time.Millisecond)
	if err != nil {
		t.Errorf("GetBlocking with startID failed: %v", err)
	}
	if id != "1234567890-0" {
		t.Errorf("Expected ID '1234567890-0', got '%s'", id)
	}
}

// TestGetBlocking_ContextCancel 测试 context 取消时能正确退出
func TestGetBlocking_ContextCancel(t *testing.T) {
	mq := &mockMQ{}

	ctx, cancel := context.WithCancel(context.Background())

	// 立即取消 context
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// GetBlocking 应该立即返回，因为 context 被取消
	_, _, err := mq.GetBlocking(ctx, "test-stream", "", 5*time.Second)
	if err == nil {
		t.Error("GetBlocking should return error when context is cancelled")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestGetBlocking_DefaultTimeout 测试默认超时
func TestGetBlocking_DefaultTimeout(t *testing.T) {
	mq := &mockMQ{}

	// 不指定 timeout 参数，使用默认值
	id, _, err := mq.GetBlocking(context.Background(), "test-stream", "")
	if err != nil {
		t.Errorf("GetBlocking with default timeout failed: %v", err)
	}
	if id == "" {
		t.Error("GetBlocking returned empty ID")
	}
}
