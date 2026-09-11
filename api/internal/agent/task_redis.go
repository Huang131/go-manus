package agent

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"

	"github.com/Huang131/go-manus/api/pkg/logger"
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
	mu    sync.RWMutex
	tasks map[string]*RedisStreamTask
}

// NewDefaultTaskRegistry 创建默认任务注册表
func NewDefaultTaskRegistry() *DefaultTaskRegistry {
	return &DefaultTaskRegistry{
		tasks: make(map[string]*RedisStreamTask),
	}
}

// defaultTaskRegistry 全局默认注册表（向后兼容）
var defaultTaskRegistry = NewDefaultTaskRegistry()

// Register 注册任务到注册表
func (r *DefaultTaskRegistry) Register(task *RedisStreamTask) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.id] = task
	logger.Debug("任务注册到注册表",
		logger.String("task_id", task.id))
}

// Unregister 从注册表移除任务
// 注意：不关闭 doneChan，因为 doneChan 的关闭由 RedisStreamTask 自己管理
// 这样可以避免多次关闭同一个 channel 导致的 panic
func (r *DefaultTaskRegistry) Unregister(taskID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tasks, taskID)
	logger.Debug("任务从注册表移除",
		logger.String("task_id", taskID))
}

// CleanupCompleted 清理已完成的任务
// 定期调用此方法可以释放已完成任务的内存
// 注意：不关闭 doneChan，因为 doneChan 的关闭由 RedisStreamTask 自己管理
func (r *DefaultTaskRegistry) CleanupCompleted() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	var toRemove []string
	for id, task := range r.tasks {
		// Done 只表示任务已请求结束；Finished 才表示 runner 和保留窗口处理完毕。
		if task.Finished() {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		delete(r.tasks, id)
		logger.Debug("清理已完成任务",
			logger.String("task_id", id))
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
	r.mu.Unlock()

	// 注意：这里不调用 task.Cancel()，避免死锁
	// 任务的生命周期管理由调用方负责
	logger.Info("任务注册表已清空",
		logger.Int("task_count", len(tasks)))
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
	inputStream  external.TaskMessageQueue
	outputStream external.TaskMessageQueue
	cancelFunc   context.CancelFunc
	done         atomic.Bool
	finished     atomic.Bool
	invoked      atomic.Bool
	doneChan     chan struct{}
	doneOnce     sync.Once
	finishOnce   sync.Once
	destroyOnce  sync.Once
	mu           sync.RWMutex
	registry     TaskRegistryInterface // 任务注册表（用于注销）
	onFinished   func()

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
	mq         external.TaskMessageQueue
	streamName string
	lastID     string // 上次读取的消息 ID，用于游标推进；空值表示从头开始读取
	mu         sync.Mutex
}

// NewTaskStream 创建任务流适配器
func NewTaskStream(mq external.TaskMessageQueue, streamName string) *TaskStream {
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
	id, data, err := s.mq.GetBlocking(ctx, s.streamName, startID, DefaultBlockTimeout)
	if err != nil {
		return "", "", err
	}
	if data == nil {
		return "", "", nil
	}

	// 推进游标：记录本次读取到的消息 ID，避免重复消费
	dataStr, err := streamDataString(data)
	if err != nil {
		return "", "", fmt.Errorf("marshal stream data: %w", err)
	}
	if id != "" {
		s.mu.Lock()
		s.lastID = id
		s.mu.Unlock()
	}

	return id, dataStr, nil
}

// streamDataString 将消息队列返回的数据统一转换为 JSON 文本。
// 队列实现可能直接返回字符串，也可能返回已解析的 map，不能通过断言失败静默变成空数据。
func streamDataString(data interface{}) (string, error) {
	if value, ok := data.(string); ok {
		return value, nil
	}
	encoded, err := sonic.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
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

// DefaultBlockTimeout 默认阻塞超时时间。
const DefaultBlockTimeout = 3 * time.Second

const maxTaskOutputBatch = 64

// NewRedisStreamTask 创建基于 Redis Stream 的任务
// 参数:
//   - mq: 消息队列
//   - runner: 任务运行器
//   - registry: 任务注册表（可选，为 nil 时使用全局默认注册表）
func NewRedisStreamTask(mq external.TaskMessageQueue, runner TaskRunner, registry ...TaskRegistryInterface) *RedisStreamTask {
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
		logger.String("task_id", taskID),
		logger.String("input_stream", task.inputStreamName()),
		logger.String("output_stream", task.outputStreamName()))

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

// Finished 表示任务 runner 已退出且资源清理完成。
func (t *RedisStreamTask) Finished() bool {
	return t.finished.Load()
}

// DoneChan 返回任务完成的通知 channel
func (t *RedisStreamTask) DoneChan() <-chan struct{} {
	return t.doneChan
}

// SetOnFinished 设置任务完成后的回调。
// 回调只会执行一次，用于让上层清理 session 到 task 的索引。
func (t *RedisStreamTask) SetOnFinished(callback func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onFinished = callback
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
	t.mu.Lock()
	if t.Done() {
		t.mu.Unlock()
		return fmt.Errorf("task already done")
	}
	if t.cancelFunc != nil {
		t.mu.Unlock()
		return fmt.Errorf("task already invoked")
	}

	ctx, cancel := context.WithCancel(ctx)
	t.cancelFunc = cancel
	t.invoked.Store(true)
	t.mu.Unlock()

	// 启动后台 goroutine 执行任务
	go t.execute(ctx)

	logger.Info("RedisStreamTask 已启动",
		logger.String("task_id", t.id))

	return nil
}

// execute 任务执行逻辑（后台运行）
func (t *RedisStreamTask) execute(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("TaskRunner.Invoke panic",
				logger.String("task_id", t.id),
				logger.Any("panic", recovered),
				logger.String("stack", string(debug.Stack())))
		}
		t.onDone()
		t.destroyRunner()
	}()

	// 调用 TaskRunner 执行任务
	if t.runner != nil {
		if err := t.runner.Invoke(ctx, t); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.Info("TaskRunner.Invoke 因上下文结束",
					logger.String("task_id", t.id),
					logger.Err(err))
				return
			}
			logger.Error("TaskRunner.Invoke 失败",
				logger.String("task_id", t.id),
				logger.Err(err))
		}
	}
}

// onDone 任务完成时的回调
func (t *RedisStreamTask) onDone() {
	t.finish("RedisStreamTask 执行完成")
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

	t.finish("RedisStreamTask 已取消")
	if !t.invoked.Load() {
		t.destroyRunner()
	}

	return true
}

// finish 统一处理任务终结，避免 Cancel 和 execute 并发时重复清理。
func (t *RedisStreamTask) finish(logMessage string) {
	t.finishOnce.Do(func() {
		t.done.Store(true)
		t.doneOnce.Do(func() { close(t.doneChan) })
		logger.Info(logMessage, logger.String("task_id", t.id))
	})
}

// setStreamRetention 为已结束任务设置较短保留窗口，给 SSE 断线续读留出时间。
// 使用独立 context，避免取消任务时原始请求 context 已经失效。
func (t *RedisStreamTask) setStreamRetention() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	streams := []struct {
		queue      external.TaskMessageQueue
		streamName string
	}{
		{queue: t.inputStream, streamName: t.inputStreamName()},
		{queue: t.outputStream, streamName: t.outputStreamName()},
	}
	for _, stream := range streams {
		if err := stream.queue.SetRetention(ctx, stream.streamName, external.CompletedStreamRetention()); err != nil {
			logger.Warn("设置任务流完成保留时间失败",
				logger.String("task_id", t.id),
				logger.String("stream", stream.streamName),
				logger.Err(err))
		}
	}
}

func (t *RedisStreamTask) destroyRunner() {
	t.destroyOnce.Do(func() {
		if t.runner != nil {
			t.runner.OnDone(t)
			if err := t.runner.Destroy(); err != nil {
				logger.Warn("销毁任务运行器失败",
					logger.String("task_id", t.id),
					logger.Err(err))
			}
		}
		if t.registry != nil {
			t.registry.Unregister(t.id)
		}
		t.mu.RLock()
		callback := t.onFinished
		t.mu.RUnlock()
		if callback != nil {
			callback()
		}
		// runner 已退出后再设置短 TTL，避免其尾部写入把完成窗口重新延长。
		t.setStreamRetention()
		t.finished.Store(true)
	})
}

// PutInput 往输入流放入消息
func (t *RedisStreamTask) PutInput(ctx context.Context, event interface{}) (string, error) {
	data, err := sonic.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event: %w", err)
	}
	return t.inputStream.Put(ctx, t.inputStreamName(), string(data))
}

// GetOutput 获取输出流中的事件
func (t *RedisStreamTask) GetOutput(ctx context.Context, startID string, blockTimeout ...int) ([]*model.Event, error) {
	return readTaskOutput(ctx, t.outputStream, t.outputStreamName(), startID, blockTimeout...)
}

// ReadTaskOutput 从任务输出流读取事件，不要求任务仍在内存注册表中。
// 任务完成后 runner 会被释放，但 Redis 保留窗口内仍允许 SSE 续读。
func ReadTaskOutput(ctx context.Context, mq external.TaskMessageQueue, taskID, startID string, blockTimeout ...int) ([]*model.Event, error) {
	if mq == nil {
		return nil, fmt.Errorf("task message queue is nil")
	}
	return readTaskOutput(ctx, mq, fmt.Sprintf("task:output:%s", taskID), startID, blockTimeout...)
}

func readTaskOutput(ctx context.Context, mq external.TaskMessageQueue, streamName, startID string, blockTimeout ...int) ([]*model.Event, error) {
	ms := 0
	if len(blockTimeout) > 0 {
		ms = blockTimeout[0]
	}

	var timeout time.Duration
	if ms > 0 {
		timeout = time.Duration(ms) * time.Millisecond
	} else {
		timeout = DefaultBlockTimeout
	}

	if startID == "" {
		startID = "0"
	}
	if !ValidStreamID(startID) {
		return nil, fmt.Errorf("invalid stream cursor %q", startID)
	}
	if batchMQ, ok := mq.(external.BatchMessageQueue); ok {
		messages, err := batchMQ.GetBlockingBatch(ctx, streamName, startID, maxTaskOutputBatch, timeout)
		if err != nil {
			return nil, err
		}
		return parseTaskOutputMessages(messages)
	}

	id, data, err := mq.GetBlocking(ctx, streamName, startID, timeout)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	return parseTaskOutputMessages([]external.StreamMessage{{ID: id, Data: data}})
}

// parseTaskOutputMessages 将队列消息转换为领域事件，并保持流游标顺序。
func parseTaskOutputMessages(messages []external.StreamMessage) ([]*model.Event, error) {
	events := make([]*model.Event, 0, len(messages))
	for _, message := range messages {
		if message.Data == nil {
			continue
		}
		dataStr, err := streamDataString(message.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize event: %w", err)
		}
		var event model.Event
		if err := sonic.UnmarshalString(dataStr, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %w", err)
		}
		event.ID = message.ID
		events = append(events, &event)
	}
	return events, nil
}

// validStreamID 校验 Redis Stream 游标，阻止业务 UUID 被误当作 XREAD 起点。
// ValidStreamID 判断字符串是否为 Redis Stream 的合法游标。
func ValidStreamID(id string) bool {
	if id == "$" || id == "0" || id == "0-0" {
		return true
	}
	parts := strings.Split(id, "-")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, part := range parts {
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// SubscribeOutput 订阅输出流
func (t *RedisStreamTask) SubscribeOutput(ctx context.Context, bufferSize int) (<-chan *model.Event, func()) {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	eventChan := make(chan *model.Event, bufferSize)
	subCtx, cancelContext := context.WithCancel(ctx)
	var cancelOnce sync.Once
	cancel := func() {
		cancelOnce.Do(cancelContext)
	}

	go func() {
		defer close(eventChan)
		defer cancelContext()

		// 订阅需要先消费流中已经缓冲的事件；空游标在 Redis 中表示从最新位置开始，
		// 会导致任务启动前产生的事件被跳过。
		lastID := "0"
		for {
			select {
			case <-subCtx.Done():
				return
			default:
				// 获取下一条消息
				id, data, err := t.outputStream.GetBlocking(subCtx, t.outputStreamName(), lastID, DefaultBlockTimeout)
				if err != nil {
					if subCtx.Err() != nil {
						return
					}
					logger.Warn("获取输出消息失败",
						logger.String("task_id", t.id),
						logger.Err(err))
					timer := time.NewTimer(time.Second)
					select {
					case <-timer.C:
					case <-subCtx.Done():
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						return
					}
					continue
				}
				if data == nil {
					continue
				}

				// 解析事件
				var event model.Event
				dataStr, err := streamDataString(data)
				if err != nil {
					logger.Warn("序列化输出消息失败",
						logger.String("task_id", t.id),
						logger.Err(err))
					continue
				}
				if err := sonic.Unmarshal([]byte(dataStr), &event); err != nil {
					logger.Warn("解析事件失败",
						logger.String("task_id", t.id),
						logger.Err(err))
					continue
				}
				event.ID = id
				lastID = id

				// 发送到 channel
				select {
				case eventChan <- &event:
				case <-subCtx.Done():
					return
				}
			}
		}
	}()

	return eventChan, cancel
}
