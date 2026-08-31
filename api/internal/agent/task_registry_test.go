package agent

import (
	"sync"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/external"
)

// TestDefaultTaskRegistry_Register 测试默认注册表的注册功能
func TestDefaultTaskRegistry_Register(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}
	runner := &mockTaskRunner{}

	task := NewRedisStreamTask(mqWrapper, runner, registry)

	// 验证任务已注册
	retrieved := registry.Get(task.ID())
	if retrieved == nil {
		t.Error("registry.Get() returned nil")
	}
	if retrieved != task {
		t.Error("registry.Get() returned wrong task")
	}

	// 验证任务数量
	if registry.Count() != 1 {
		t.Errorf("Count() = %d, expected 1", registry.Count())
	}
}

// TestDefaultTaskRegistry_Unregister 测试默认注册表的移除功能
func TestDefaultTaskRegistry_Unregister(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}
	runner := &mockTaskRunner{}

	task := NewRedisStreamTask(mqWrapper, runner, registry)
	taskID := task.ID()

	// 验证任务已注册
	if registry.Get(taskID) == nil {
		t.Error("registry.Get() returned nil before unregister")
	}

	// 移除任务
	registry.Unregister(taskID)

	// 验证任务已移除
	if registry.Get(taskID) != nil {
		t.Error("registry.Get() should return nil after unregister")
	}

	// 验证任务数量
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, expected 0", registry.Count())
	}
}

// TestDefaultTaskRegistry_List 测试列出所有任务
func TestDefaultTaskRegistry_List(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	// 创建多个任务
	tasks := make([]*RedisStreamTask, 5)
	for i := 0; i < 5; i++ {
		tasks[i] = NewRedisStreamTask(mqWrapper, &mockTaskRunner{}, registry)
	}

	// 列出所有任务
	list := registry.List()
	if len(list) != 5 {
		t.Errorf("List() returned %d tasks, expected 5", len(list))
	}

	// 清理
	registry.Clear()
}

// TestDefaultTaskRegistry_Count 测试任务计数
func TestDefaultTaskRegistry_Count(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	// 初始状态
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, expected 0 initially", registry.Count())
	}

	// 添加任务
	for i := 0; i < 3; i++ {
		NewRedisStreamTask(mqWrapper, &mockTaskRunner{}, registry)
	}
	if registry.Count() != 3 {
		t.Errorf("Count() = %d, expected 3", registry.Count())
	}

	// 移除一个任务
	task := registry.List()[0]
	registry.Unregister(task.ID())
	if registry.Count() != 2 {
		t.Errorf("Count() = %d, expected 2 after unregister", registry.Count())
	}

	// 清理
	registry.Clear()
}

// TestDefaultTaskRegistry_Clear 测试清空注册表
func TestDefaultTaskRegistry_Clear(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	// 创建多个任务
	for i := 0; i < 3; i++ {
		NewRedisStreamTask(mqWrapper, &mockTaskRunner{}, registry)
	}
	if registry.Count() != 3 {
		t.Errorf("Count() = %d, expected 3", registry.Count())
	}

	// 清空
	registry.Clear()
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, expected 0 after clear", registry.Count())
	}
}

// TestDefaultTaskRegistry_ConcurrentAccess 测试并发访问
func TestDefaultTaskRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	var wg sync.WaitGroup
	startChan := make(chan struct{})

	// 并发注册
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startChan
			for j := 0; j < 10; j++ {
				task := NewRedisStreamTask(mqWrapper, &mockTaskRunner{}, registry)
				_ = task.ID()
			}
		}()
	}

	// 并发读取
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startChan
			for j := 0; j < 100; j++ {
				_ = registry.Count()
				_ = registry.List()
			}
		}()
	}

	// 启动所有 goroutine
	close(startChan)
	wg.Wait()

	// 验证最终状态
	count := registry.Count()
	if count != 100 {
		t.Errorf("Count() = %d, expected 100", count)
	}

	// 清理
	registry.Clear()
}

// TestRedisStreamTask_WithCustomRegistry 测试使用自定义注册表
func TestRedisStreamTask_WithCustomRegistry(t *testing.T) {
	// 创建自定义注册表
	customRegistry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}
	runner := &mockTaskRunner{}

	// 使用自定义注册表创建任务
	task := NewRedisStreamTask(mqWrapper, runner, customRegistry)

	// 验证任务在自定义注册表中
	if customRegistry.Get(task.ID()) == nil {
		t.Error("Task should be in custom registry")
	}

	// 清理
	task.Cancel()
}

// TestRedisStreamTask_RegistryNotNil 测试任务持有正确的注册表引用
func TestRedisStreamTask_RegistryNotNil(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}
	runner := &mockTaskRunner{}

	task := NewRedisStreamTask(mqWrapper, runner, registry)

	// 取消任务时应该使用注册表
	task.Cancel()

	// 验证任务已从注册表移除
	if registry.Get(task.ID()) != nil {
		t.Error("Task should be removed after Cancel()")
	}
}

// TestNewRedisStreamTask_DefaultRegistry 测试未指定注册表时使用默认注册表
func TestNewRedisStreamTask_DefaultRegistry(t *testing.T) {
	// 清理默认注册表
	defaultTaskRegistry.Clear()

	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}
	runner := &mockTaskRunner{}

	// 不指定注册表（使用可变参数）
	task := NewRedisStreamTask(mqWrapper, runner)

	// 验证任务在默认注册表中
	if defaultTaskRegistry.Get(task.ID()) == nil {
		t.Error("Task should be in default registry")
	}

	// 清理
	defaultTaskRegistry.Clear()
}

// TestDefaultTaskRegistry_CleanupCompleted 测试清理已完成任务
// 注意：由于 Cancel() 会自动调用 Unregister()，CleanupCompleted 主要用于
// 清理那些通过其他方式（如外部超时）标记为完成但尚未从注册表移除的任务
func TestDefaultTaskRegistry_CleanupCompleted(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}

	// 创建多个任务
	tasks := make([]*RedisStreamTask, 3)
	for i := 0; i < 3; i++ {
		tasks[i] = NewRedisStreamTask(mqWrapper, &mockTaskRunner{}, registry)
	}

	// 初始状态：3 个任务都未完成
	if registry.Count() != 3 {
		t.Errorf("Count() = %d, expected 3 initially", registry.Count())
	}

	// CleanupCompleted 不应该清理未完成的任务
	cleaned := registry.CleanupCompleted()
	if cleaned != 0 {
		t.Errorf("CleanupCompleted() cleaned %d, expected 0 for running tasks", cleaned)
	}

	// 取消任务（会从注册表自动移除）
	tasks[0].Cancel()
	tasks[2].Cancel()

	// 验证任务已从注册表移除（Cancel 自动调用 Unregister）
	if registry.Count() != 1 {
		t.Errorf("Count() = %d, expected 1 after cancel", registry.Count())
	}

	// 清理
	registry.Clear()
}

// TestDefaultTaskRegistry_DoubleUnregister 测试重复移除任务不会 panic
func TestDefaultTaskRegistry_DoubleUnregister(t *testing.T) {
	registry := NewDefaultTaskRegistry()
	mq := &external.RedisStreamMessageQueue{}
	mqWrapper := &mockMQWrapper{mq: mq}
	runner := &mockTaskRunner{}

	task := NewRedisStreamTask(mqWrapper, runner, registry)
	taskID := task.ID()

	// 第一次移除
	registry.Unregister(taskID)
	if registry.Get(taskID) != nil {
		t.Error("Task should be removed after first unregister")
	}

	// 第二次移除不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DoubleUnregister() panicked: %v", r)
		}
	}()
	registry.Unregister(taskID)
}

// TestDefaultTaskRegistry_UnregisterNonexistent 测试移除不存在的任务不会 panic
func TestDefaultTaskRegistry_UnregisterNonexistent(t *testing.T) {
	registry := NewDefaultTaskRegistry()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unregister(Nonexistent) panicked: %v", r)
		}
	}()
	registry.Unregister("nonexistent-task-id")
}
