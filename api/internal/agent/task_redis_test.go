package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/mq"
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
	invoked       chan struct{}
	invokeOnce    sync.Once
}

type panicTaskRunner struct{}

func (panicTaskRunner) Invoke(context.Context, *RedisStreamTask) error {
	panic("test panic")
}

func (panicTaskRunner) Destroy() error          { return nil }
func (panicTaskRunner) OnDone(*RedisStreamTask) {}

type blockingTaskRunner struct {
	started chan struct{}
	release chan struct{}
}

func (r *blockingTaskRunner) Invoke(ctx context.Context, _ *RedisStreamTask) error {
	close(r.started)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.release:
		return nil
	}
}

func (r *blockingTaskRunner) Destroy() error          { return nil }
func (r *blockingTaskRunner) OnDone(*RedisStreamTask) {}

func (m *mockTaskRunner) Invoke(ctx context.Context, task *RedisStreamTask) error {
	m.mu.Lock()
	m.invokeCalled = true
	m.invokeCtx = ctx
	m.invokeTask = task
	m.mu.Unlock()

	if m.invoked != nil {
		m.invokeOnce.Do(func() { close(m.invoked) })
	}
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

func (m *mockTaskRunner) wasDestroyCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.destroyCalled
}

// mockMQWrapper 用于测试的 Mock MessageQueue 包装器
// 由于 RedisStreamMessageQueue 需要实际的 Redis 连接，我们使用一个简化的包装器
type mockMQWrapper struct {
	mq             *mq.RedisStreamMessageQueue
	mu             sync.Mutex
	retentionCalls []retentionCall
}

type batchTaskOutputMQ struct {
	mockMQWrapper
	batchCalls    int
	blockingCalls int
	streamName    string
	startID       string
	count         int
	timeout       time.Duration
	messages      []mq.StreamMessage
}

func (m *batchTaskOutputMQ) GetBlocking(ctx context.Context, streamName, startID string, timeout ...time.Duration) (string, interface{}, error) {
	m.blockingCalls++
	return "", nil, fmt.Errorf("unexpected single-message read")
}

func (m *batchTaskOutputMQ) GetBlockingBatch(_ context.Context, streamName, startID string, count int, timeout ...time.Duration) ([]mq.StreamMessage, error) {
	m.batchCalls++
	m.streamName = streamName
	m.startID = startID
	m.count = count
	if len(timeout) > 0 {
		m.timeout = timeout[0]
	}
	return m.messages, nil
}

type retentionCall struct {
	streamName string
	retention  time.Duration
}

func (w *mockMQWrapper) Put(ctx context.Context, streamName string, message interface{}) (string, error) {
	// 返回模拟的消息 ID
	return "test-msg-id", nil
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

func (w *mockMQWrapper) Clear(ctx context.Context, streamName string) error {
	return nil
}

func (w *mockMQWrapper) IsEmpty(ctx context.Context, streamName string) (bool, error) {
	return true, nil
}

func (w *mockMQWrapper) Size(ctx context.Context, streamName string) (int64, error) {
	return 0, nil
}

func (w *mockMQWrapper) SetRetention(ctx context.Context, streamName string, retention time.Duration) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.retentionCalls = append(w.retentionCalls, retentionCall{
		streamName: streamName,
		retention:  retention,
	})
	return nil
}

func (w *mockMQWrapper) getRetentionCalls() []retentionCall {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]retentionCall(nil), w.retentionCalls...)
}

// TestRedisStreamTask_DoneChan 测试 DoneChan 通道
func TestRedisStreamTask_DoneChan(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &mq.RedisStreamMessageQueue{}
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

	mq := &mq.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	runner := &mockTaskRunner{invoked: make(chan struct{})}
	task := NewRedisStreamTask(mqWrapper, runner)

	// 创建 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 验证 Invoke 调用成功
	err := task.Invoke(ctx)
	if err != nil {
		t.Errorf("Invoke() returned error: %v", err)
	}

	select {
	case <-runner.invoked:
	case <-time.After(time.Second):
		t.Fatal("TaskRunner.Invoke() was not called")
	}

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

func TestRedisStreamTask_PanicStillFinishes(t *testing.T) {
	defaultTaskRegistry.Clear()
	task := NewRedisStreamTask(&mockMQWrapper{}, panicTaskRunner{})
	if err := task.Invoke(context.Background()); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	select {
	case <-task.DoneChan():
	case <-time.After(time.Second):
		t.Fatal("task did not finish after runner panic")
	}
	if !task.Done() {
		t.Fatal("task Done() = false after runner panic")
	}
}

func TestRedisStreamTask_FinishDestroysRunner(t *testing.T) {
	defaultTaskRegistry.Clear()
	runner := &mockTaskRunner{}
	task := NewRedisStreamTask(&mockMQWrapper{}, runner)

	task.Cancel()

	if !runner.wasDestroyCalled() {
		t.Fatal("TaskRunner.Destroy() was not called when task finished")
	}
}

func TestRedisStreamTask_FinishSetsStreamRetention(t *testing.T) {
	defaultTaskRegistry.Clear()
	queue := &mockMQWrapper{}
	task := NewRedisStreamTask(queue, &mockTaskRunner{})

	task.Cancel()

	calls := queue.getRetentionCalls()
	if len(calls) != 2 {
		t.Fatalf("retention calls = %d, want 2", len(calls))
	}
	wantRetention := mq.CompletedStreamRetention()
	for _, call := range calls {
		if call.retention != wantRetention {
			t.Fatalf("retention for %s = %s, want %s", call.streamName, call.retention, wantRetention)
		}
	}
}

func TestRedisStreamTask_InvokeAfterCancelReturnsError(t *testing.T) {
	defaultTaskRegistry.Clear()
	runner := &mockTaskRunner{}
	task := NewRedisStreamTask(&mockMQWrapper{}, runner)
	task.Cancel()

	if err := task.Invoke(context.Background()); err == nil {
		t.Fatal("Invoke() succeeded after task was canceled")
	}
	if runner.wasInvokeCalled() {
		t.Fatal("TaskRunner.Invoke() was called after task cancellation")
	}
}

func TestRedisStreamTask_CancelKeepsRegistryUntilRunnerExits(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	runner := &blockingTaskRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	task := NewRedisStreamTask(&mockMQWrapper{}, runner, registry)
	if err := task.Invoke(context.Background()); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	<-runner.started

	task.Cancel()
	if registry.Get(task.ID()) == nil {
		t.Fatal("task was unregistered before runner exited")
	}

	close(runner.release)
	select {
	case <-task.DoneChan():
	case <-time.After(time.Second):
		t.Fatal("task did not finish")
	}
	select {
	case <-task.FinishedChan():
	case <-time.After(time.Second):
		t.Fatal("task did not finish cleanup")
	}
	if registry.Get(task.ID()) != nil {
		t.Fatal("task remained registered after runner exited")
	}
}

func TestRedisStreamTask_FinishedChangesAfterRunnerExit(t *testing.T) {
	runner := &blockingTaskRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	task := NewRedisStreamTask(&mockMQWrapper{}, runner)
	if err := task.Invoke(context.Background()); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	<-runner.started
	task.Cancel()
	if task.Finished() {
		t.Fatal("Finished() = true before runner exit")
	}
	close(runner.release)
	select {
	case <-task.DoneChan():
	case <-time.After(time.Second):
		t.Fatal("task did not finish")
	}
	select {
	case <-task.FinishedChan():
	case <-time.After(time.Second):
		t.Fatal("task did not finish cleanup")
	}
	if !task.Finished() {
		t.Fatal("Finished() = false after runner exit")
	}
}

func TestRedisStreamTask_GetOutputReadsBufferedEventsWhenStartIDEmpty(t *testing.T) {
	mq := newInMemoryMessageQueue()
	task := NewRedisStreamTask(mq, &mockTaskRunner{})
	defer task.Cancel()

	eventJSON, err := json.Marshal(&model.Event{
		ID:   "event-1",
		Type: model.EventTypeDone,
		Data: json.RawMessage(`{"message":"done"}`),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if _, err := mq.Put(context.Background(), "task:output:"+task.ID(), string(eventJSON)); err != nil {
		t.Fatalf("put buffered event: %v", err)
	}

	events, err := task.GetOutput(context.Background(), "", 20)
	if err != nil {
		t.Fatalf("GetOutput() error = %v", err)
	}
	if len(events) != 1 || events[0].Type != model.EventTypeDone {
		t.Fatalf("GetOutput() = %+v, want one buffered done event", events)
	}
}

func TestValidStreamID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"", false},
		{"0", true},
		{"0-0", true},
		{"$", true},
		{"1710000000000-0", true},
		{"uuid-value", false},
		{"1710000000000", false},
	}
	for _, tt := range tests {
		if got := ValidStreamID(tt.id); got != tt.valid {
			t.Errorf("ValidStreamID(%q) = %v, want %v", tt.id, got, tt.valid)
		}
	}
}

func TestReadTaskOutputAfterTaskUnregistered(t *testing.T) {
	mq := newInMemoryMessageQueue()
	task := NewRedisStreamTask(mq, &mockTaskRunner{})

	eventJSON, err := json.Marshal(&model.Event{
		Type: model.EventTypeDone,
		Data: json.RawMessage(`{"message":"done"}`),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if _, err := mq.Put(context.Background(), "task:output:"+task.ID(), string(eventJSON)); err != nil {
		t.Fatalf("put event: %v", err)
	}

	task.Cancel()
	if defaultTaskRegistry.Get(task.ID()) != nil {
		t.Fatalf("task should be unregistered after cancel")
	}

	events, err := ReadTaskOutput(context.Background(), mq, task.ID(), "", 20)
	if err != nil {
		t.Fatalf("ReadTaskOutput() error = %v", err)
	}
	if len(events) != 1 || events[0].Type != model.EventTypeDone {
		t.Fatalf("ReadTaskOutput() = %+v, want retained done event", events)
	}
}

func TestReadTaskOutputPrefersBatchQueue(t *testing.T) {
	mq := &batchTaskOutputMQ{messages: []mq.StreamMessage{
		{ID: "1710000000000-1", Data: `{"type":"message_delta","data":{"delta":"你"}}`},
		{ID: "1710000000000-2", Data: `{"type":"message_done","data":{"content":"你好"}}`},
	}}

	events, err := ReadTaskOutput(context.Background(), mq, "task-1", "1710000000000-0", 250)
	if err != nil {
		t.Fatalf("ReadTaskOutput() error = %v", err)
	}
	if mq.batchCalls != 1 || mq.blockingCalls != 0 {
		t.Fatalf("queue calls = batch:%d blocking:%d, want batch:1 blocking:0", mq.batchCalls, mq.blockingCalls)
	}
	if mq.streamName != "task:output:task-1" || mq.startID != "1710000000000-0" {
		t.Fatalf("batch arguments = stream %q start %q", mq.streamName, mq.startID)
	}
	if mq.count != maxTaskOutputBatch || mq.timeout != 250*time.Millisecond {
		t.Fatalf("batch options = count:%d timeout:%s", mq.count, mq.timeout)
	}
	if len(events) != 2 || events[0].ID != "1710000000000-1" || events[1].ID != "1710000000000-2" {
		t.Fatalf("events = %+v, want ordered stream IDs", events)
	}
	if events[0].Type != model.EventTypeMessageDelta || events[1].Type != model.EventTypeMessageDone {
		t.Fatalf("event types = %q, %q", events[0].Type, events[1].Type)
	}
}

func TestParseTaskOutputMessagesBatchSkipsNilData(t *testing.T) {
	events, err := parseTaskOutputMessages([]mq.StreamMessage{
		{ID: "1710000000000-1", Data: nil},
		{ID: "1710000000000-2", Data: map[string]interface{}{
			"type": model.EventTypeDone,
			"data": map[string]interface{}{"message": "done"},
		}},
	})
	if err != nil {
		t.Fatalf("parseTaskOutputMessages() error = %v", err)
	}
	if len(events) != 1 || events[0].ID != "1710000000000-2" || events[0].Type != model.EventTypeDone {
		t.Fatalf("events = %+v, want one done event with stream ID", events)
	}
}

// TestRedisStreamTask_Cancel 测试任务取消
func TestRedisStreamTask_Cancel(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &mq.RedisStreamMessageQueue{}
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

func TestStreamDataStringMarshalsNonStringData(t *testing.T) {
	got, err := streamDataString(map[string]string{"message": "hello"})
	if err != nil {
		t.Fatalf("streamDataString() error = %v", err)
	}
	if got != `{"message":"hello"}` {
		t.Fatalf("streamDataString() = %q, want JSON object", got)
	}
}

// TestRedisStreamTask_PutInput_GetOutput 测试消息传递
func TestRedisStreamTask_PutInput_GetOutput(t *testing.T) {
	// 清理注册表
	defaultTaskRegistry.Clear()

	mq := &mq.RedisStreamMessageQueue{}
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

	mq := &mq.RedisStreamMessageQueue{}
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
