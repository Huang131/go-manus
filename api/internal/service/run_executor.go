package service

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/settings"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

const runPersistenceTimeout = 30 * time.Second

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

type completedRunEventMarker interface {
	MarkCompleted(ctx context.Context, runID string) error
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
	// persistenceTimeout 限制每次独立持久化或收敛尝试的最长时间。
	persistenceTimeout time.Duration

	mu      sync.Mutex
	control map[string]*runControl
}

type runControl struct {
	handle          *RunExecutionHandle
	cancel          context.CancelFunc
	admission       sync.Mutex
	cancelPersisted bool
}

// NewRunExecutor 创建不带实时事件发布器的执行器，供局部装配和测试使用。
func NewRunExecutor(store RunExecutionStore, provider RunToolProvider, engine RunExecutionEngine) *RunExecutor {
	return NewRunExecutorWithEventPublisher(store, provider, engine, nil)
}

// NewRunExecutorWithEventPublisher 创建带实时事件发布器的执行器。
// 发布器是可选旁路；未配置时 Run 仍可独立完成持久化。
func NewRunExecutorWithEventPublisher(store RunExecutionStore, provider RunToolProvider, engine RunExecutionEngine, publisher RunEventPublisher) *RunExecutor {
	return &RunExecutor{
		store:              store,
		provider:           provider,
		engine:             engine,
		publisher:          publisher,
		persistenceTimeout: runPersistenceTimeout,
		control:            make(map[string]*runControl),
	}
}

// Start 先登记控制句柄，再启动 Run；同一进程内同一 Run 不重复执行。
func (e *RunExecutor) Start(ctx context.Context, runID string, messages []llmcore.Message, attachmentContexts ...[]attachment.FileContext) (*RunExecutionHandle, error) {
	if e.store == nil || e.provider == nil || e.engine == nil {
		return nil, errors.New("run executor dependencies are incomplete")
	}
	ctx, cancel := context.WithCancel(ctx)
	handle := &RunExecutionHandle{done: make(chan struct{})}
	control := &runControl{handle: handle, cancel: cancel}
	e.mu.Lock()
	if _, exists := e.control[runID]; exists {
		e.mu.Unlock()
		cancel()
		return nil, fmt.Errorf("run %q already has an execution handle", runID)
	}
	e.control[runID] = control
	e.mu.Unlock()

	run, err := e.store.Start(ctx, runID)
	if err != nil {
		e.removeControl(runID, control)
		cancel()
		return nil, fmt.Errorf("start run: %w", err)
	}
	if run == nil {
		e.removeControl(runID, control)
		cancel()
		return nil, errors.New("start run returned nil run")
	}
	toolSet, err := e.provider.Acquire(run.SettingsSnapshot)
	if err != nil {
		// Run 已经进入 running，工具无法装配时必须立即收敛，不能遗留 active Run。
		e.abortStart(ctx, runID, control)
		return nil, fmt.Errorf("acquire run tools: %w", err)
	}
	if toolSet == nil {
		err := errors.New("acquire run tools returned nil tool set")
		e.abortStart(ctx, runID, control)
		return nil, err
	}
	var contexts []attachment.FileContext
	if len(attachmentContexts) > 0 {
		contexts = append([]attachment.FileContext(nil), attachmentContexts[0]...)
	}

	// 句柄在工具装配期间已可被 Cancel 观察；持有 admission 完成最后检查和
	// Engine 入场，防止取消已落库但 context 尚未取消时插入执行。
	e.mu.Lock()
	current := e.control[runID]
	if current != control {
		e.mu.Unlock()
		toolSet.Release()
		cancel()
		return nil, fmt.Errorf("run %q execution handle was replaced", runID)
	}
	control.admission.Lock()
	defer control.admission.Unlock()
	if ctx.Err() != nil {
		cancelPersisted := control.cancelPersisted
		e.mu.Unlock()
		toolSet.Release()
		e.reconcileCancelledStart(ctx, runID, control, !cancelPersisted)
		return handle, nil
	}
	e.mu.Unlock()
	go e.execute(ctx, cancel, run, messages, contexts, toolSet, control)
	return handle, nil
}

func (e *RunExecutor) abortStart(ctx context.Context, runID string, control *runControl) {
	e.removeControl(runID, control)
	control.cancel()
	reconcileCtx, reconcileCancel := e.newPersistenceContext(ctx)
	defer reconcileCancel()
	if _, err := e.store.RequestCancel(reconcileCtx, runID); err != nil {
		// RequestCancel 是幂等状态迁移；启动补偿不能因一次瞬时存储故障
		// 放弃收敛，否则进程内已无句柄的 Run 会永久占用 active 槽位。
		if _, retryErr := e.store.RequestCancel(reconcileCtx, runID); retryErr != nil {
			logger.Warn("Run 启动失败后无法请求取消", logger.String("run_id", runID), logger.Err(retryErr))
			return
		}
	}
	if _, err := e.store.ReconcileCancelling(reconcileCtx, runID); err != nil {
		logger.Warn("Run 启动失败后无法收敛取消", logger.String("run_id", runID), logger.Err(err))
	}
}

func (e *RunExecutor) reconcileCancelledStart(ctx context.Context, runID string, control *runControl, requestCancel bool) {
	e.removeControl(runID, control)
	control.cancel()
	reconcileCtx, reconcileCancel := e.newPersistenceContext(ctx)
	defer reconcileCancel()
	if requestCancel {
		if _, err := e.store.RequestCancel(reconcileCtx, runID); err != nil {
			logger.Warn("Run 在启动前被取消后无法请求取消", logger.String("run_id", runID), logger.Err(err))
			control.handle.setResult(err)
			return
		}
	}
	if _, err := e.store.ReconcileCancelling(reconcileCtx, runID); err != nil {
		logger.Warn("Run 在启动前被取消后无法收敛", logger.String("run_id", runID), logger.Err(err))
		control.handle.setResult(err)
		return
	}
	control.handle.setResult(nil)
}

func (e *RunExecutor) execute(ctx context.Context, cancel context.CancelFunc, run *model.Run, messages []llmcore.Message, attachmentContexts []attachment.FileContext, toolSet RunToolSet, control *runControl) {
	defer cancel()
	result, err := e.executeEngine(ctx, RunExecutionInput{
		RunID: run.ID, SessionID: run.SessionID, Settings: run.SettingsSnapshot, Snapshot: run.ExecutionSnapshot,
		Messages: append([]llmcore.Message(nil), messages...), Tools: toolSet.Tools(), AttachmentContexts: attachmentContexts, EventPublisher: e.publisher,
	})
	// Engine 已经退出后才释放外部工具连接，避免 in-flight 调用使用已关闭资源。
	toolSet.Release()

	// 取消只停止 Engine；状态收尾继续使用独立且有界的持久化上下文。
	// WithoutCancel 保留“取消后仍写终态”的语义，WithTimeout 防止存储异常时执行 goroutine 无限滞留。
	persistenceCtx, persistenceCancel := e.newPersistenceContext(ctx)
	defer persistenceCancel()
	persistErr := e.persistResult(persistenceCtx, run, result, err)
	if persistErr != nil {
		persistenceCancel()
		reconcileCtx, reconcileCancel := e.newPersistenceContext(ctx)
		e.reconcileAfterPersistenceFailure(reconcileCtx, run.ID)
		reconcileCancel()
	}
	if persistErr == nil && e.publisher != nil && isTerminalExecution(result, err) {
		if marker, ok := e.publisher.(completedRunEventMarker); ok {
			if markErr := marker.MarkCompleted(persistenceCtx, run.ID); markErr != nil {
				// 事件流只是实时旁路，TTL 失败不能回滚已提交的 Run 终态。
				logger.Warn("完成事件流保留时间设置失败", logger.String("run_id", run.ID), logger.Err(markErr))
			}
		}
	}
	// Engine 和 ToolSet 已退出，控制句柄不能因持久化失败滞留在进程内。
	// 先删除再通知 Wait，保证后续 Cancel 立即走数据库 reconciliation。
	e.removeControl(run.ID, control)
	control.handle.setResult(persistErr)
}

func (e *RunExecutor) newPersistenceContext(parent context.Context) (context.Context, context.CancelFunc) {
	timeout := e.persistenceTimeout
	if timeout <= 0 {
		timeout = runPersistenceTimeout
	}
	return context.WithTimeout(context.WithoutCancel(parent), timeout)
}

// reconcileAfterPersistenceFailure 避免结果写入失败后留下没有进程句柄的活跃 Run。
// 先处理已经进入 cancelling 的竞争窗口；仍处于活跃状态时再请求取消并收敛。
func (e *RunExecutor) reconcileAfterPersistenceFailure(ctx context.Context, runID string) {
	if _, err := e.store.ReconcileCancelling(ctx, runID); err == nil {
		return
	}
	if _, err := e.store.RequestCancel(ctx, runID); err != nil {
		logger.Warn("执行结果持久化失败后无法请求取消 Run", logger.String("run_id", runID), logger.Err(err))
		return
	}
	if _, err := e.store.ReconcileCancelling(ctx, runID); err != nil {
		logger.Warn("执行结果持久化失败后无法收敛 Run", logger.String("run_id", runID), logger.Err(err))
	}
}

func isTerminalExecution(result RunExecutionResult, engineErr error) bool {
	if engineErr != nil {
		return true
	}
	switch result.Kind {
	case RunExecutionSucceeded, RunExecutionFailed, RunExecutionCancelled:
		return true
	default:
		return false
	}
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
	e.mu.Lock()
	control := e.control[runID]
	e.mu.Unlock()
	if control == nil {
		if _, err := e.store.RequestCancel(ctx, runID); err != nil {
			return nil, fmt.Errorf("request run cancel: %w", err)
		}
		return e.store.ReconcileCancelling(ctx, runID)
	}
	control.admission.Lock()
	defer control.admission.Unlock()
	run, err := e.store.RequestCancel(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("request run cancel: %w", err)
	}
	control.cancelPersisted = true
	if handleEnded(control.handle) {
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
