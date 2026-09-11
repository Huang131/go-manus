package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"
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

func (m *mockTaskRunner) wasDestroyCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.destroyCalled
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
	mq             *external.RedisStreamMessageQueue
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
	messages      []external.StreamMessage
}

func (m *batchTaskOutputMQ) GetBlocking(ctx context.Context, streamName, startID string, timeout ...time.Duration) (string, interface{}, error) {
	m.blockingCalls++
	return "", nil, fmt.Errorf("unexpected single-message read")
}

func (m *batchTaskOutputMQ) GetBlockingBatch(_ context.Context, streamName, startID string, count int, timeout ...time.Duration) ([]external.StreamMessage, error) {
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
	mq := &mockMQWrapper{}
	task := NewRedisStreamTask(mq, &mockTaskRunner{})

	task.Cancel()

	calls := mq.getRetentionCalls()
	if len(calls) != 2 {
		t.Fatalf("retention calls = %d, want 2", len(calls))
	}
	wantRetention := external.CompletedStreamRetention()
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
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		if registry.Get(task.ID()) == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("task remained registered after runner exited")
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
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		if task.Finished() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("Finished() remained false after runner exit")
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
	mq := &batchTaskOutputMQ{messages: []external.StreamMessage{
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
	events, err := parseTaskOutputMessages([]external.StreamMessage{
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

func TestRedisStreamTask_SubscribeOutputCancelIsIdempotent(t *testing.T) {
	defaultTaskRegistry.Clear()

	task := NewRedisStreamTask(&mockMQWrapper{}, &mockTaskRunner{})
	events, cancel := task.SubscribeOutput(context.Background(), 1)

	cancel()
	cancel()

	select {
	case _, ok := <-events:
		if ok {
			for range events {
			}
		}
	case <-time.After(time.Second):
		t.Fatal("SubscribeOutput channel was not closed after cancellation")
	}
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
