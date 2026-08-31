package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// TaskRegistryInterface 任务注册表接口
// 支持依赖注入，便于测试和多实例隔离
type TaskRegistryInterface interface {
	Register(task *RedisStreamTask)
	Unregister(taskID string)
	Get(taskID string) *RedisStreamTask
	List() []*RedisStreamTask
	Count() int
	CleanupCompleted() int
	Clear()
}

// DefaultTaskRegistry 默认任务注册表实现
// 使用 sync.RWMutex 而不是 channel：Go 风格更直接
type DefaultTaskRegistry struct {
	mu        sync.RWMutex
	tasks     map[string]*RedisStreamTask
	doneChans map[string]chan struct{} // 用于取消和清理
}

// NewDefaultTaskRegistry 创建默认任务注册表
func NewDefaultTaskRegistry() *DefaultTaskRegistry {
	return &DefaultTaskRegistry{
		tasks:     make(map[string]*RedisStreamTask),
		doneChans: make(map[string]chan struct{}),
	}
}

// defaultTaskRegistry 全局默认注册表（向后兼容）
var defaultTaskRegistry = NewDefaultTaskRegistry()

// Register 注册任务到注册表
func (r *DefaultTaskRegistry) Register(task *RedisStreamTask) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.id] = task
	r.doneChans[task.id] = task.doneChan // 保存 doneChan 引用
	logger.Debug("任务注册到注册表",
		zap.String("task_id", task.id))
}

// Unregister 从注册表移除任务
// 注意：不关闭 doneChan，因为 doneChan 的关闭由 RedisStreamTask 自己管理
// 这样可以避免多次关闭同一个 channel 导致的 panic
func (r *DefaultTaskRegistry) Unregister(taskID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tasks, taskID)
	// 只从 doneChans 中移除引用，不关闭 channel
	delete(r.doneChans, taskID)
	logger.Debug("任务从注册表移除",
		zap.String("task_id", taskID))
}

// CleanupCompleted 清理已完成的任务
// 定期调用此方法可以释放已完成任务的内存
// 注意：不关闭 doneChan，因为 doneChan 的关闭由 RedisStreamTask 自己管理
func (r *DefaultTaskRegistry) CleanupCompleted() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	var toRemove []string
	for id, task := range r.tasks {
		if task.IsDone() {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		delete(r.tasks, id)
		// 只移除引用，不关闭 channel
		delete(r.doneChans, id)
		logger.Debug("清理已完成任务",
			zap.String("task_id", id))
	}

	return len(toRemove)
}

// Get 根据任务ID获取任务
func (r *DefaultTaskRegistry) Get(taskID string) *RedisStreamTask {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tasks[taskID]
}

// List 返回所有任务
func (r *DefaultTaskRegistry) List() []*RedisStreamTask {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tasks := make([]*RedisStreamTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Count 返回任务数量
func (r *DefaultTaskRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tasks)
}

// Clear 清除所有任务
// 注意：此方法不会取消任务本身，只是将任务从注册表中移除
// 如果需要同时取消任务，请先调用每个任务的 Cancel() 方法
func (r *DefaultTaskRegistry) Clear() {
	r.mu.Lock()
	tasks := r.tasks
	r.tasks = make(map[string]*RedisStreamTask)
	r.doneChans = make(map[string]chan struct{})
	r.mu.Unlock()

	// 注意：这里不调用 task.Cancel()，避免死锁
	// 任务的生命周期管理由调用方负责
	logger.Info("任务注册表已清空",
		zap.Int("task_count", len(tasks)))
}

// TaskRunner 任务运行器接口（对齐 Python 版本）
type TaskRunner interface {
	// Invoke 执行任务
	Invoke(ctx context.Context, task *RedisStreamTask) error
	// Destroy 销毁运行器
	Destroy() error
	// OnDone 任务完成时的回调
	OnDone(task *RedisStreamTask)
}

// RedisStreamTask 基于 Redis Stream 的任务实现
// 对齐 Python 版本的 RedisStreamTask 架构
type RedisStreamTask struct {
	id           string
	runner       TaskRunner
	inputStream  external.MessageQueue
	outputStream external.MessageQueue
	cancelFunc   context.CancelFunc
	done         atomic.Bool
	doneChan     chan struct{}
	mu           sync.RWMutex
	registry     TaskRegistryInterface // 任务注册表（用于注销）

	// 缓存的流适配器实例，确保游标状态跨调用保持（避免每次新建导致重复消费）
	inputStreamInstance  *TaskStream
	outputStreamInstance *TaskStream
}

// IsDone 检查任务是否已完成
func (t *RedisStreamTask) IsDone() bool {
	return t.done.Load()
}

// TaskStream Redis Stream 适配器
// 将 external.MessageQueue 适配为 Task 需要的 Stream 接口
type TaskStream struct {
	mq         external.MessageQueue
	streamName string
	lastID     string // 上次读取的消息 ID，用于游标推进；空值表示从头开始读取
	mu         sync.Mutex
}

// NewTaskStream 创建任务流适配器
func NewTaskStream(mq external.MessageQueue, streamName string) *TaskStream {
	return &TaskStream{
		mq:         mq,
		streamName: streamName,
		lastID:     "",
	}
}

// Put 放入消息
func (s *TaskStream) Put(ctx context.Context, data string) (string, error) {
	return s.mq.Put(ctx, s.streamName, data)
}

// Pop 取出消息（阻塞）
// 使用消息 ID 游标推进：首次从 "0" 开始读取，之后以上次返回的消息 ID 作为起点，
// 避免同一条消息因 XRead 只读不删而被重复消费。
func (s *TaskStream) Pop(ctx context.Context) (string, string, error) {
	s.mu.Lock()
	startID := s.lastID
	if startID == "" {
		// 首次读取时使用 "0" 从已有消息开始
		startID = "0"
	}
	s.mu.Unlock()

	// 使用 GetBlocking 实现阻塞 Pop
	id, data, err := s.mq.GetBlocking(ctx, s.streamName, startID, 3*DefaultBlockTimeout)
	if err != nil {
		return "", "", err
	}
	if data == nil {
		return "", "", nil
	}

	// 推进游标：记录本次读取到的消息 ID，避免重复消费
	if id != "" {
		s.mu.Lock()
		s.lastID = id
		s.mu.Unlock()
	}

	// 将 data 转换为字符串
	dataStr, ok := data.(string)
	if !ok {
		// 如果是其他类型，序列化为 JSON
		bytes, _ := json.Marshal(data)
		dataStr = string(bytes)
	}
	return id, dataStr, nil
}

// IsEmpty 检查是否为空
func (s *TaskStream) IsEmpty(ctx context.Context) (bool, error) {
	return s.mq.IsEmpty(ctx, s.streamName)
}

// Len 返回队列长度
func (s *TaskStream) Len(ctx context.Context) (int, error) {
	size, err := s.mq.Size(ctx, s.streamName)
	return int(size), err
}

// DefaultBlockTimeout 默认阻塞超时时间
const DefaultBlockTimeout = 3e9 // 3秒（纳秒）

// NewRedisStreamTask 创建基于 Redis Stream 的任务
// 参数:
//   - mq: 消息队列
//   - runner: 任务运行器
//   - registry: 任务注册表（可选，为 nil 时使用全局默认注册表）
func NewRedisStreamTask(mq external.MessageQueue, runner TaskRunner, registry ...TaskRegistryInterface) *RedisStreamTask {
	taskID := uuid.New().String()

	// 如果没有传入注册表，使用全局默认注册表（向后兼容）
	var reg TaskRegistryInterface = defaultTaskRegistry
	if len(registry) > 0 && registry[0] != nil {
		reg = registry[0]
	}

	task := &RedisStreamTask{
		id:           taskID,
		runner:       runner,
		inputStream:  mq,
		outputStream: mq,
		doneChan:     make(chan struct{}),
		registry:     reg,
	}

	// 注册到注册表
	reg.Register(task)

	logger.Info("创建 RedisStreamTask",
		zap.String("task_id", taskID),
		zap.String("input_stream", task.inputStreamName()),
		zap.String("output_stream", task.outputStreamName()))

	return task
}

// inputStreamName 返回输入流名称
func (t *RedisStreamTask) inputStreamName() string {
	return fmt.Sprintf("task:input:%s", t.id)
}

// outputStreamName 返回输出流名称
func (t *RedisStreamTask) outputStreamName() string {
	return fmt.Sprintf("task:output:%s", t.id)
}

// ID 返回任务ID
func (t *RedisStreamTask) ID() string {
	return t.id
}

// Done 返回任务是否完成
func (t *RedisStreamTask) Done() bool {
	return t.done.Load()
}

// DoneChan 返回任务完成的通知 channel
func (t *RedisStreamTask) DoneChan() <-chan struct{} {
	return t.doneChan
}

// InputStream 返回输入流（惰性创建并缓存实例，保证游标状态跨调用保持）
func (t *RedisStreamTask) InputStream() Stream {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inputStreamInstance == nil {
		t.inputStreamInstance = NewTaskStream(t.inputStream, t.inputStreamName())
	}
	return t.inputStreamInstance
}

// OutputStream 返回输出流（惰性创建并缓存实例）
func (t *RedisStreamTask) OutputStream() Stream {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.outputStreamInstance == nil {
		t.outputStreamInstance = NewTaskStream(t.outputStream, t.outputStreamName())
	}
	return t.outputStreamInstance
}

// Invoke 启动任务执行（后台 goroutine）
// 对齐 Python: await task.invoke() -> asyncio.create_task(self._execute_task())
func (t *RedisStreamTask) Invoke(ctx context.Context) error {
	if t.Done() {
		return fmt.Errorf("task already done")
	}

	t.mu.Lock()
	if t.cancelFunc != nil {
		t.mu.Unlock()
		return fmt.Errorf("task already invoked")
	}

	ctx, cancel := context.WithCancel(ctx)
	t.cancelFunc = cancel
	t.mu.Unlock()

	// 启动后台 goroutine 执行任务
	go t.execute(ctx)

	logger.Info("RedisStreamTask 已启动",
		zap.String("task_id", t.id))

	return nil
}

// execute 任务执行逻辑（后台运行）
func (t *RedisStreamTask) execute(ctx context.Context) {
	defer t.onDone()

	// 调用 TaskRunner 执行任务
	if t.runner != nil {
		if err := t.runner.Invoke(ctx, t); err != nil {
			logger.Error("TaskRunner.Invoke 失败",
				zap.String("task_id", t.id),
				zap.Error(err))
		}
	}
}

// onDone 任务完成时的回调
func (t *RedisStreamTask) onDone() {
	t.done.Store(true)

	// 关闭完成通知 channel
	close(t.doneChan)

	// 调用 runner 的 OnDone 回调
	if t.runner != nil {
		t.runner.OnDone(t)
	}

	// 从注册表移除
	if t.registry != nil {
		t.registry.Unregister(t.id)
	}

	logger.Info("RedisStreamTask 执行完成",
		zap.String("task_id", t.id))
}

// Cancel 取消任务
func (t *RedisStreamTask) Cancel() bool {
	if t.Done() {
		return true
	}

	t.mu.Lock()
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	t.mu.Unlock()

	// 立即设置 done 标志，确保 Done() 返回 true
	t.done.Store(true)

	// 关闭完成通知 channel
	select {
	case <-t.doneChan:
		// channel 已关闭
	default:
		close(t.doneChan)
	}

	if t.registry != nil {
		t.registry.Unregister(t.id)
	}
	logger.Info("RedisStreamTask 已取消",
		zap.String("task_id", t.id))

	return true
}

// PutInput 往输入流放入消息
func (t *RedisStreamTask) PutInput(ctx context.Context, event interface{}) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event: %w", err)
	}
	return t.inputStream.Put(ctx, t.inputStreamName(), string(data))
}

// GetOutput 获取输出流中的事件
func (t *RedisStreamTask) GetOutput(ctx context.Context, startID string, blockTimeout ...int) ([]*model.Event, error) {
	ms := 0
	if len(blockTimeout) > 0 {
		ms = blockTimeout[0]
	}

	var timeout time.Duration
	if ms > 0 {
		timeout = time.Duration(ms) * time.Millisecond
	} else {
		timeout = 3 * time.Second
	}

	id, data, err := t.outputStream.GetBlocking(ctx, t.outputStreamName(), startID, timeout)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	// 解析事件
	var event model.Event
	if err := json.Unmarshal([]byte(data.(string)), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	event.ID = id

	return []*model.Event{&event}, nil
}

// SubscribeOutput 订阅输出流
func (t *RedisStreamTask) SubscribeOutput(ctx context.Context, bufferSize int) (<-chan *model.Event, func()) {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	eventChan := make(chan *model.Event, bufferSize)
	stopChan := make(chan struct{})

	cancel := func() {
		close(stopChan)
	}

	go func() {
		defer close(eventChan)

		lastID := ""
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopChan:
				return
			default:
				// 获取下一条消息
				id, data, err := t.outputStream.GetBlocking(ctx, t.outputStreamName(), lastID, 3*time.Second)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					logger.Warn("获取输出消息失败",
						zap.String("task_id", t.id),
						zap.Error(err))
					time.Sleep(time.Second)
					continue
				}
				if data == nil {
					continue
				}

				// 解析事件
				var event model.Event
				dataStr, _ := data.(string)
				if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
					logger.Warn("解析事件失败",
						zap.String("task_id", t.id),
						zap.Error(err))
					continue
				}
				event.ID = id
				lastID = id

				// 发送到 channel
				select {
				case eventChan <- &event:
				case <-ctx.Done():
					return
				case <-stopChan:
					return
				}
			}
		}
	}()

	return eventChan, cancel
}
