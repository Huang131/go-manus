package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mooc-manus/go-manus/api/internal/external"
)

// mockTaskRunner 用于测试的 Mock TaskRunner
//
// 并发模型：
//   - 所有字段共享读写，用 sync.RWMutex 保护
//   - Invoke 在 RedisStreamTask.execute goroutine 中调用（生产者）
//   - 测试主线程读取字段做断言（消费者）
//   - 必须用 -race 验证，否则会偶发 data race 误判
type mockTaskRunner struct {
	mu            sync.RWMutex
	invokeCalled  bool
	invokeCtx     context.Context
	invokeTask    *RedisStreamTask
	destroyCalled bool
	onDoneCalled  bool
	onDoneTask    *RedisStreamTask
}

func (m *mockTaskRunner) Invoke(ctx context.Context, task *RedisStreamTask) error {
	m.mu.Lock()
	m.invokeCalled = true
	m.invokeCtx = ctx
	m.invokeTask = task
	m.mu.Unlock()

	// 模拟执行，阻塞一段时间后返回
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (m *mockTaskRunner) Destroy() error {
	m.mu.Lock()
	m.destroyCalled = true
	m.mu.Unlock()
	return nil
}

func (m *mockTaskRunner) OnDone(task *RedisStreamTask) {
	m.mu.Lock()
	m.onDoneCalled = true
	m.onDoneTask = task
	m.mu.Unlock()
}

// 断言辅助方法（用 RLock 减少竞争）
func (m *mockTaskRunner) wasInvokeCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.invokeCalled
}

func (m *mockTaskRunner) getInvokeCtx() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.invokeCtx
}

func (m *mockTaskRunner) getInvokeTask() *RedisStreamTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.invokeTask
}

// TestTaskRegistry_Register 测试任务注册
func TestTaskRegistry_Register(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	// 创建 mock MessageQueue
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	// 创建 mock TaskRunner
	runner := &mockTaskRunner{}

	// 创建 RedisStreamTask
	task := NewRedisStreamTask(mqWrapper, runner)

	// 验证任务已注册
	retrieved := defaultTaskRegistry.Get(task.ID())
	if retrieved == nil {
		t.Errorf("TaskRegistry.Get() returned nil, expected task")
	}
	if retrieved != task {
		t.Errorf("TaskRegistry.Get() returned wrong task")
	}

	// 验证任务数量
	if defaultTaskRegistry.Count() != 1 {
		t.Errorf("TaskRegistry.Count() = %d, expected 1", defaultTaskRegistry.Count())
	}

	// 清理
	task.Cancel()
}

// TestTaskRegistry_Unregister 测试任务移除
func TestTaskRegistry_Unregister(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	// 创建 mock MessageQueue
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	// 创建 mock TaskRunner
	runner := &mockTaskRunner{}

	// 创建 RedisStreamTask
	task := NewRedisStreamTask(mqWrapper, runner)
	taskID := task.ID()

	// 验证任务已注册
	if defaultTaskRegistry.Get(taskID) == nil {
		t.Errorf("TaskRegistry.Get() returned nil before unregister")
	}

	// 移除任务
	defaultTaskRegistry.Unregister(taskID)

	// 验证任务已移除
	if defaultTaskRegistry.Get(taskID) != nil {
		t.Errorf("TaskRegistry.Get() should return nil after unregister")
	}

	// 验证任务数量
	if defaultTaskRegistry.Count() != 0 {
		t.Errorf("TaskRegistry.Count() = %d, expected 0", defaultTaskRegistry.Count())
	}
}

// TestTaskRegistry_Get 测试获取任务
func TestTaskRegistry_Get(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	// 创建多个任务
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	tasks := make([]*RedisStreamTask, 3)
	for i := 0; i < 3; i++ {
		tasks[i] = NewRedisStreamTask(mqWrapper, &mockTaskRunner{})
	}

	// 验证可以获取所有任务
	for i, task := range tasks {
		retrieved := defaultTaskRegistry.Get(task.ID())
		if retrieved == nil {
			t.Errorf("TaskRegistry.Get() returned nil for task %d", i)
		}
		if retrieved != task {
			t.Errorf("TaskRegistry.Get() returned wrong task for index %d", i)
		}
	}

	// 验证任务数量
	if defaultTaskRegistry.Count() != 3 {
		t.Errorf("TaskRegistry.Count() = %d, expected 3", defaultTaskRegistry.Count())
	}

	// 清理
	for _, task := range tasks {
		task.Cancel()
	}
}

// TestTaskRegistry_List 测试列出所有任务
func TestTaskRegistry_List(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	// 创建多个任务
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	tasks := make([]*RedisStreamTask, 3)
	for i := 0; i < 3; i++ {
		tasks[i] = NewRedisStreamTask(mqWrapper, &mockTaskRunner{})
	}

	// 列出所有任务
	list := defaultTaskRegistry.List()
	if len(list) != 3 {
		t.Errorf("TaskRegistry.List() returned %d tasks, expected 3", len(list))
	}

	// 清理
	for _, task := range tasks {
		task.Cancel()
	}
}

// TestTaskRegistry_Clear 测试清除所有任务
func TestTaskRegistry_Clear(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	// 创建多个任务
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	for i := 0; i < 3; i++ {
		NewRedisStreamTask(mqWrapper, &mockTaskRunner{})
	}

	// 验证任务数量
	if defaultTaskRegistry.Count() != 3 {
		t.Errorf("TaskRegistry.Count() = %d, expected 3 before clear", defaultTaskRegistry.Count())
	}

	// 清除所有任务
	defaultTaskRegistry.Clear()

	// 验证任务数量
	if defaultTaskRegistry.Count() != 0 {
		t.Errorf("TaskRegistry.Count() = %d, expected 0 after clear", defaultTaskRegistry.Count())
	}
}

// mockMQWrapper 用于测试的 Mock MessageQueue 包装器
// 由于 RedisStreamMessageQueue 需要实际的 Redis 连接，我们使用一个简化的包装器
type mockMQWrapper struct {
	mq *external.RedisStreamMessageQueue
}

func (w *mockMQWrapper) Put(ctx context.Context, streamName string, message interface{}) (string, error) {
	// 返回模拟的消息 ID
	return "test-msg-id", nil
}

func (w *mockMQWrapper) Get(ctx context.Context, streamName string, startID string, blockMs *int) (string, interface{}, error) {
	return "", nil, nil
}

func (w *mockMQWrapper) GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error) {
	// 模拟阻塞获取，超时后返回
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	case <-time.After(1 * time.Second):
		return "", nil, nil
	}
}

func (w *mockMQWrapper) GetRange(ctx context.Context, streamName string, startID, endID string, limit int64) ([]*external.Message, error) {
	return []*external.Message{}, nil
}

func (w *mockMQWrapper) GetLatestID(ctx context.Context, streamName string) (string, error) {
	return "", nil
}

func (w *mockMQWrapper) Pop(ctx context.Context, streamName string) (string, interface{}, error) {
	return "", nil, nil
}

func (w *mockMQWrapper) Clear(ctx context.Context, streamName string) error {
	return nil
}

func (w *mockMQWrapper) IsEmpty(ctx context.Context, streamName string) (bool, error) {
	return true, nil
}

func (w *mockMQWrapper) Size(ctx context.Context, streamName string) (int64, error) {
	return 0, nil
}

func (w *mockMQWrapper) DeleteMessage(ctx context.Context, streamName string, messageID string) error {
	return nil
}

func (w *mockMQWrapper) Subscribe(ctx context.Context, streamName string, bufferSize int) (<-chan *external.Message, func()) {
	ch := make(chan *external.Message, bufferSize)
	return ch, func() { close(ch) }
}

func (w *mockMQWrapper) Close() error {
	return nil
}

// TestRedisStreamTask_DoneChan 测试 DoneChan 通道
func TestRedisStreamTask_DoneChan(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	runner := &mockTaskRunner{}
	task := NewRedisStreamTask(mqWrapper, runner)

	// 验证 DoneChan 存在且未关闭
	doneChan := task.DoneChan()
	if doneChan == nil {
		t.Error("DoneChan() returned nil")
	}

	// 验证任务未完成时 channel 未关闭
	select {
	case _, ok := <-doneChan:
		if ok {
			t.Error("DoneChan should not be closed before task completion")
		}
	default:
		// 预期行为：channel 未关闭，不阻塞
	}

	// 清理
	task.Cancel()
}

// TestRedisStreamTask_Invoke 测试后台执行
func TestRedisStreamTask_Invoke(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	runner := &mockTaskRunner{}
	task := NewRedisStreamTask(mqWrapper, runner)

	// 创建 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 验证 Invoke 调用成功
	err := task.Invoke(ctx)
	if err != nil {
		t.Errorf("Invoke() returned error: %v", err)
	}

	// 等待一段时间让 goroutine 执行
	time.Sleep(200 * time.Millisecond)

	// 验证 TaskRunner.Invoke 被调用
	if !runner.wasInvokeCalled() {
		t.Error("TaskRunner.Invoke() was not called")
	}

	// 验证上下文被正确传递
	if runner.getInvokeCtx() == nil {
		t.Error("TaskRunner.Invoke() was called with nil context")
	}

	// 验证任务被正确传递
	if runner.getInvokeTask() != task {
		t.Error("TaskRunner.Invoke() was called with wrong task")
	}

	// 清理
	task.Cancel()
}

// TestRedisStreamTask_Cancel 测试任务取消
func TestRedisStreamTask_Cancel(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	runner := &mockTaskRunner{}
	task := NewRedisStreamTask(mqWrapper, runner)

	// 验证任务未取消
	if task.Done() {
		t.Error("Done() should return false before cancel")
	}

	// 取消任务
	task.Cancel()

	// 验证任务已取消
	if !task.Done() {
		t.Error("Done() should return true after cancel")
	}

	// 再次取消不应该 panic
	task.Cancel()
}

// TestRedisStreamTask_PutInput_GetOutput 测试消息传递
func TestRedisStreamTask_PutInput_GetOutput(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	runner := &mockTaskRunner{}
	task := NewRedisStreamTask(mqWrapper, runner)

	ctx := context.Background()

	// 测试 PutInput
	testEvent := struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}{
		Type:    "test",
		Message: "hello",
	}

	msgID, err := task.PutInput(ctx, testEvent)
	if err != nil {
		t.Errorf("PutInput() returned error: %v", err)
	}
	if msgID == "" {
		t.Error("PutInput() returned empty message ID")
	}

	// 测试 InputStream
	inputStream := task.InputStream()
	if inputStream == nil {
		t.Error("InputStream() returned nil")
	}

	// 测试 OutputStream
	outputStream := task.OutputStream()
	if outputStream == nil {
		t.Error("OutputStream() returned nil")
	}

	// 清理
	task.Cancel()
}

// TestRedisStreamTask_ID 测试任务 ID 生成
func TestRedisStreamTask_ID(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	runner := &mockTaskRunner{}
	task := NewRedisStreamTask(mqWrapper, runner)

	// 验证任务 ID 非空
	taskID := task.ID()
	if taskID == "" {
		t.Error("ID() returned empty string")
	}

	// 验证任务 ID 唯一性
	task2 := NewRedisStreamTask(mqWrapper, &mockTaskRunner{})
	if task.ID() == task2.ID() {
		t.Error("Two tasks should have different IDs")
	}

	// 清理
	task.Cancel()
	task2.Cancel()
}
