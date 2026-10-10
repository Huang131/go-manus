package service

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/settings"
)

// RunExecutionKind 是 Engine adapter 返回给 RunExecutor 的有限结果集合。
type RunExecutionKind string

const (
	RunExecutionSucceeded    RunExecutionKind = "succeeded"
	RunExecutionWaitingInput RunExecutionKind = "waiting_input"
	RunExecutionFailed       RunExecutionKind = "failed"
	RunExecutionCancelled    RunExecutionKind = "cancelled"
)

// RunExecutionInput 是一次 Engine 调用的不可变输入快照。
type RunExecutionInput struct {
	RunID     string
	SessionID string
	Settings  settings.AgentSettings
	Snapshot  model.RunExecutionSnapshot
	Messages  []llmcore.Message
	Tools     []toolspkg.Tool
	// AttachmentContexts 是本轮输入已加载的文件正文，只存在于执行期，不写入 Run 快照。
	AttachmentContexts []attachment.FileContext
	// EventPublisher 只承载短期实时事件；发布失败不能改变 Run 持久化结果。
	EventPublisher RunEventPublisher
}

// RunEventPublisher 发布一次执行的实时事件。PostgreSQL 中的 Run/Message 仍是事实来源。
type RunEventPublisher interface {
	Publish(ctx context.Context, runID string, event model.BaseEvent) error
}

// RunExecutionResult 是 Engine adapter 返回的领域结果，不包含持久化副作用。
type RunExecutionResult struct {
	Kind     RunExecutionKind
	Snapshot model.RunExecutionSnapshot
	Question *model.RunMessage
	Message  *model.RunMessage
	Text     string
	Error    error
}

// RunExecutionEngine 是 Planner/ReAct Engine 的稳定适配边界。
type RunExecutionEngine interface {
	Execute(ctx context.Context, input RunExecutionInput) (RunExecutionResult, error)
}

// RunToolSet 是一次执行独占的工具快照。
type RunToolSet interface {
	Tools() []toolspkg.Tool
	Release()
}

// RunToolProvider 为 RunExecutor 提供冻结的工具集合。
type RunToolProvider interface {
	Acquire(settings.AgentSettings) (RunToolSet, error)
}

// ToolProviderAdapter 将已有 ToolProvider 适配到 RunExecutor 边界。
// 该适配器只在生产切换检查点接入，4A 本身不改变现有 Chat 路径。
type ToolProviderAdapter struct {
	Provider *toolspkg.ToolProvider
}

func (a ToolProviderAdapter) Acquire(agentSettings settings.AgentSettings) (RunToolSet, error) {
	if a.Provider == nil {
		return nil, errors.New("tool provider is nil")
	}
	return a.Provider.Acquire(agentSettings.MaxSearchResults), nil
}

// AgentToolProviderAdapter 将 AgentService 暴露的窄工具快照能力适配到 Run 边界。
type AgentToolProviderAdapter struct {
	Provider interface {
		AcquireTools(settings.AgentSettings) *toolspkg.ToolSet
	}
}

func (a AgentToolProviderAdapter) Acquire(agentSettings settings.AgentSettings) (RunToolSet, error) {
	if a.Provider == nil {
		return nil, errors.New("agent tool provider is nil")
	}
	set := a.Provider.AcquireTools(agentSettings)
	if set == nil {
		return nil, errors.New("agent tool provider returned nil tool set")
	}
	return set, nil
}

// RunExecutionStore 是 RunExecutor 需要的最小持久化边界。
type RunExecutionStore interface {
	Start(ctx context.Context, runID string) (*model.Run, error)
	SaveWaitingInput(ctx context.Context, runID string, snapshot model.RunExecutionSnapshot, question *model.RunMessage) error
	Finish(ctx context.Context, runID string, result RunExecutionResult) error
	RequestCancel(ctx context.Context, runID string) (*model.Run, error)
	ReconcileCancelling(ctx context.Context, runID string) (*model.Run, error)
}

// RunExecutionHandle 表示一次执行的进程内控制句柄。
// 业务状态在数据库中，句柄只负责取消、等待和错误观察。
type RunExecutionHandle struct {
	done chan struct{}

	mu    sync.Mutex
	err   error
	ended bool
}

func (h *RunExecutionHandle) Done() <-chan struct{} { return h.done }

func (h *RunExecutionHandle) Wait(ctx context.Context) error {
	select {
	case <-h.done:
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *RunExecutionHandle) setResult(err error) {
	h.mu.Lock()
	h.err = err
	h.ended = true
	h.mu.Unlock()
	close(h.done)
}

// RunExecutor 负责一次 Run 的 Engine 生命周期和 ToolSet 释放顺序。
type RunExecutor struct {
	store     RunExecutionStore
	provider  RunToolProvider
	engine    RunExecutionEngine
	publisher RunEventPublisher

	mu      sync.Mutex
	control map[string]*runControl
}

type runControl struct {
	handle *RunExecutionHandle
	cancel context.CancelFunc
}

// NewRunExecutor 创建未接入生产路由的执行器。
func NewRunExecutor(store RunExecutionStore, provider RunToolProvider, engine RunExecutionEngine) *RunExecutor {
	return NewRunExecutorWithEventPublisher(store, provider, engine, nil)
}

// NewRunExecutorWithEventPublisher 创建带实时事件发布器的执行器。
// 发布器是可选旁路；未配置时 Run 仍可独立完成持久化。
func NewRunExecutorWithEventPublisher(store RunExecutionStore, provider RunToolProvider, engine RunExecutionEngine, publisher RunEventPublisher) *RunExecutor {
	return &RunExecutor{
		store:     store,
		provider:  provider,
		engine:    engine,
		publisher: publisher,
		control:   make(map[string]*runControl),
	}
}

// Start 原子登记控制句柄后启动 Engine；同一进程内同一 Run 不重复执行。
func (e *RunExecutor) Start(ctx context.Context, runID string, messages []llmcore.Message, attachmentContexts ...[]attachment.FileContext) (*RunExecutionHandle, error) {
	if e.store == nil || e.provider == nil || e.engine == nil {
		return nil, errors.New("run executor dependencies are incomplete")
	}
	run, err := e.store.Start(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("start run: %w", err)
	}
	if run == nil {
		return nil, errors.New("start run returned nil run")
	}
	toolSet, err := e.provider.Acquire(run.SettingsSnapshot)
	if err != nil {
		// Run 已经进入 running，工具无法装配时必须立即收敛，不能遗留 active Run。
		if _, cancelErr := e.Cancel(ctx, runID); cancelErr != nil {
			return nil, fmt.Errorf("acquire run tools: %w; reconcile cancellation: %v", err, cancelErr)
		}
		return nil, fmt.Errorf("acquire run tools: %w", err)
	}
	if toolSet == nil {
		err := errors.New("acquire run tools returned nil tool set")
		if _, cancelErr := e.Cancel(ctx, runID); cancelErr != nil {
			return nil, fmt.Errorf("%w; reconcile cancellation: %v", err, cancelErr)
		}
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	handle := &RunExecutionHandle{done: make(chan struct{})}
	control := &runControl{handle: handle, cancel: cancel}
	e.mu.Lock()
	if _, exists := e.control[runID]; exists {
		e.mu.Unlock()
		cancel()
		toolSet.Release()
		return nil, fmt.Errorf("run %q already has an execution handle", runID)
	}
	e.control[runID] = control
	e.mu.Unlock()

	var contexts []attachment.FileContext
	if len(attachmentContexts) > 0 {
		contexts = append([]attachment.FileContext(nil), attachmentContexts[0]...)
	}
	go e.execute(ctx, cancel, run, messages, contexts, toolSet, control)
	return handle, nil
}

func (e *RunExecutor) execute(ctx context.Context, cancel context.CancelFunc, run *model.Run, messages []llmcore.Message, attachmentContexts []attachment.FileContext, toolSet RunToolSet, control *runControl) {
	defer cancel()
	result, err := e.executeEngine(ctx, RunExecutionInput{
		RunID: run.ID, SessionID: run.SessionID, Settings: run.SettingsSnapshot, Snapshot: run.ExecutionSnapshot,
		Messages: append([]llmcore.Message(nil), messages...), Tools: toolSet.Tools(), AttachmentContexts: attachmentContexts, EventPublisher: e.publisher,
	})
	// Engine 已经退出后才释放外部工具连接，避免 in-flight 调用使用已关闭资源。
	toolSet.Release()

	// 取消只停止 Engine；状态收尾必须继续使用可用的持久化上下文。
	persistErr := e.persistResult(context.WithoutCancel(ctx), run, result, err)
	if persistErr == nil {
		e.removeControl(run.ID, control)
	}
	control.handle.setResult(persistErr)
}

func (e *RunExecutor) executeEngine(ctx context.Context, input RunExecutionInput) (result RunExecutionResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = RunExecutionResult{Kind: RunExecutionFailed, Error: fmt.Errorf("engine panic: %v", recovered)}
			err = fmt.Errorf("engine panic: %v\n%s", recovered, debug.Stack())
		}
	}()
	result, err = e.engine.Execute(ctx, input)
	if ctx.Err() != nil {
		return RunExecutionResult{Kind: RunExecutionCancelled, Error: ctx.Err()}, nil
	}
	if err != nil {
		return RunExecutionResult{Kind: RunExecutionFailed, Error: err}, nil
	}
	if result.Kind == "" {
		return RunExecutionResult{Kind: RunExecutionFailed, Error: errors.New("engine returned empty result kind")}, nil
	}
	return result, nil
}

func (e *RunExecutor) persistResult(ctx context.Context, run *model.Run, result RunExecutionResult, engineErr error) error {
	if engineErr != nil {
		result = RunExecutionResult{Kind: RunExecutionFailed, Error: engineErr}
	}
	switch result.Kind {
	case RunExecutionWaitingInput:
		if result.Question == nil {
			return errors.New("waiting input result has no question")
		}
		if err := e.store.SaveWaitingInput(ctx, run.ID, result.Snapshot, result.Question); err != nil {
			return fmt.Errorf("persist waiting input: %w", err)
		}
		return nil
	case RunExecutionCancelled:
		return e.store.Finish(ctx, run.ID, result)
	case RunExecutionFailed:
		return e.store.Finish(ctx, run.ID, result)
	case RunExecutionSucceeded:
		return e.store.Finish(ctx, run.ID, result)
	default:
		result.Kind = RunExecutionFailed
		result.Error = errors.New("unknown engine result kind")
		return e.store.Finish(ctx, run.ID, result)
	}
}

// Cancel 先持久化 cancelling，再通知句柄；句柄缺失或已退出时立即 reconciliation。
func (e *RunExecutor) Cancel(ctx context.Context, runID string) (*model.Run, error) {
	run, err := e.store.RequestCancel(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("request run cancel: %w", err)
	}
	e.mu.Lock()
	control := e.control[runID]
	e.mu.Unlock()
	if control == nil || handleEnded(control.handle) {
		return e.store.ReconcileCancelling(ctx, runID)
	}
	control.cancel()
	return run, nil
}

func handleEnded(handle *RunExecutionHandle) bool {
	handle.mu.Lock()
	defer handle.mu.Unlock()
	return handle.ended
}

func (e *RunExecutor) removeControl(runID string, control *runControl) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if current, ok := e.control[runID]; ok && current == control {
		delete(e.control, runID)
	}
}
