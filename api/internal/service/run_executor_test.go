package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	toolspkg "github.com/Huang131/go-manus/api/internal/agent/tools"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/settings"
)

type executorStoreStub struct {
	mu         sync.Mutex
	status     model.RunStatus
	steps      []string
	finishErr  error
	waitingErr error
	finished   chan struct{}
}

func (s *executorStoreStub) Start(_ context.Context, _ string) (*model.Run, error) {
	s.record("start")
	return &model.Run{
		ID: "run-1", SessionID: "session-1", Status: model.RunStatusRunning,
		SettingsSnapshot: settings.AgentSettings{MaxIterations: 10, MaxRetries: 3, MaxSearchResults: 10},
	}, nil
}

func (s *executorStoreStub) SaveWaitingInput(_ context.Context, _ string, _ model.RunExecutionSnapshot, _ *model.RunMessage) error {
	s.record("waiting")
	if s.waitingErr != nil {
		return s.waitingErr
	}
	s.status = model.RunStatusWaitingInput
	return nil
}

func (s *executorStoreStub) Finish(_ context.Context, _ string, result RunExecutionResult) error {
	s.record("finish:" + string(result.Kind))
	if s.finishErr != nil {
		return s.finishErr
	}
	s.status = runStatusForResult(result.Kind)
	if s.finished != nil {
		close(s.finished)
	}
	return nil
}

func runStatusForResult(kind RunExecutionKind) model.RunStatus {
	switch kind {
	case RunExecutionSucceeded:
		return model.RunStatusSucceeded
	case RunExecutionCancelled:
		return model.RunStatusCancelled
	default:
		return model.RunStatusFailed
	}
}

func (s *executorStoreStub) RequestCancel(_ context.Context, _ string) (*model.Run, error) {
	s.record("request_cancel")
	s.status = model.RunStatusCancelling
	return &model.Run{ID: "run-1", Status: model.RunStatusCancelling}, nil
}

func (s *executorStoreStub) ReconcileCancelling(_ context.Context, _ string) (*model.Run, error) {
	s.record("reconcile")
	s.status = model.RunStatusCancelled
	return &model.Run{ID: "run-1", Status: model.RunStatusCancelled}, nil
}

func (s *executorStoreStub) record(step string) {
	s.mu.Lock()
	s.steps = append(s.steps, step)
	s.mu.Unlock()
}

func (s *executorStoreStub) events() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.steps...)
}

type executorToolSetStub struct {
	released chan struct{}
	once     sync.Once
}

func (s *executorToolSetStub) Tools() []toolspkg.Tool { return nil }

func (s *executorToolSetStub) Release() {
	s.once.Do(func() { close(s.released) })
}

type executorToolProviderStub struct {
	set *executorToolSetStub
	err error
}

func (p *executorToolProviderStub) Acquire(_ settings.AgentSettings) (RunToolSet, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.set == nil {
		return nil, nil
	}
	return p.set, nil
}

type blockingToolProvider struct {
	set     *executorToolSetStub
	started chan struct{}
	release chan struct{}
}

func (p *blockingToolProvider) Acquire(settings.AgentSettings) (RunToolSet, error) {
	close(p.started)
	<-p.release
	return p.set, nil
}

type cancelBarrierStore struct {
	executorStoreStub
	requestStarted chan struct{}
	releaseRequest chan struct{}
}

func (s *cancelBarrierStore) RequestCancel(ctx context.Context, runID string) (*model.Run, error) {
	close(s.requestStarted)
	<-s.releaseRequest
	return s.executorStoreStub.RequestCancel(ctx, runID)
}

type strictStartCancellationStore struct {
	executorStoreStub
}

func (s *strictStartCancellationStore) ReconcileCancelling(ctx context.Context, runID string) (*model.Run, error) {
	if s.status != model.RunStatusCancelling {
		return nil, errors.New("run is not cancelling")
	}
	return s.executorStoreStub.ReconcileCancelling(ctx, runID)
}

// persistenceContextStore 记录终态写入收到的上下文，用于守护取消后持久化仍有时间上限的契约。
type persistenceContextStore struct {
	executorStoreStub
	persistenceCtx context.Context
	persistenceErr error
}

func (s *persistenceContextStore) Finish(ctx context.Context, runID string, result RunExecutionResult) error {
	s.persistenceCtx = ctx
	s.persistenceErr = ctx.Err()
	return s.executorStoreStub.Finish(ctx, runID, result)
}

type persistenceTimeoutStore struct {
	executorStoreStub
	finishCtx     context.Context
	finishErr     error
	requestErr    error
	reconcileErrs []error
}

func (s *persistenceTimeoutStore) Finish(ctx context.Context, _ string, _ RunExecutionResult) error {
	s.finishCtx = ctx
	<-ctx.Done()
	s.finishErr = ctx.Err()
	return ctx.Err()
}

func (s *persistenceTimeoutStore) RequestCancel(ctx context.Context, runID string) (*model.Run, error) {
	s.requestErr = ctx.Err()
	return s.executorStoreStub.RequestCancel(ctx, runID)
}

func (s *persistenceTimeoutStore) ReconcileCancelling(ctx context.Context, runID string) (*model.Run, error) {
	s.reconcileErrs = append(s.reconcileErrs, ctx.Err())
	if len(s.reconcileErrs) == 1 {
		return nil, errors.New("run is not cancelling")
	}
	return s.executorStoreStub.ReconcileCancelling(ctx, runID)
}

type executorEngineStub struct {
	result        RunExecutionResult
	err           error
	panic         bool
	seen          RunExecutionInput
	started       chan struct{}
	waitForCancel bool
	startedOnce   sync.Once
}

type cancellationRaceStore struct {
	mu            sync.Mutex
	status        model.RunStatus
	events        []string
	finishStarted chan struct{}
	releaseFinish chan struct{}
}

func (s *cancellationRaceStore) Start(context.Context, string) (*model.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, "start")
	s.status = model.RunStatusRunning
	return &model.Run{ID: "run-1", SessionID: "session-1", Status: s.status}, nil
}

func (s *cancellationRaceStore) SaveWaitingInput(context.Context, string, model.RunExecutionSnapshot, *model.RunMessage) error {
	return nil
}

func (s *cancellationRaceStore) Finish(_ context.Context, _ string, result RunExecutionResult) error {
	s.mu.Lock()
	s.events = append(s.events, "finish:"+string(result.Kind))
	s.mu.Unlock()
	close(s.finishStarted)
	<-s.releaseFinish
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == model.RunStatusCancelling {
		return errors.New("run cannot finish after cancellation")
	}
	s.status = runStatusForResult(result.Kind)
	return nil
}

func (s *cancellationRaceStore) RequestCancel(context.Context, string) (*model.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, "request_cancel")
	s.status = model.RunStatusCancelling
	return &model.Run{ID: "run-1", Status: s.status}, nil
}

func (s *cancellationRaceStore) ReconcileCancelling(context.Context, string) (*model.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, "reconcile")
	s.status = model.RunStatusCancelled
	return &model.Run{ID: "run-1", Status: s.status}, nil
}

func (s *cancellationRaceStore) snapshot() (model.RunStatus, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, append([]string(nil), s.events...)
}

func (e *executorEngineStub) Execute(ctx context.Context, input RunExecutionInput) (RunExecutionResult, error) {
	e.seen = input
	if e.started != nil {
		e.startedOnce.Do(func() { close(e.started) })
	}
	if e.waitForCancel {
		<-ctx.Done()
	}
	if e.panic {
		panic("engine panic")
	}
	return e.result, e.err
}

func TestRunExecutorPassesAttachmentContextsToEngine(t *testing.T) {
	engine := &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}}
	executor := NewRunExecutor(
		&executorStoreStub{},
		&executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}},
		engine,
	)
	contexts := []attachment.FileContext{{Filename: "report.txt", Content: "正文"}}
	handle, err := executor.Start(context.Background(), "run-1", nil, contexts)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := handle.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if len(engine.seen.AttachmentContexts) != 1 || engine.seen.AttachmentContexts[0].Content != "正文" {
		t.Fatalf("Engine attachment contexts = %+v, want loaded attachment body", engine.seen.AttachmentContexts)
	}
}

func newExecutorTestRun() *RunExecutor {
	return NewRunExecutor(&executorStoreStub{}, &executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}}, &executorEngineStub{
		result: RunExecutionResult{Kind: RunExecutionSucceeded},
	})
}

func waitExecutorDone(t *testing.T, handle *RunExecutionHandle) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := handle.Wait(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRunExecutorSuccessReleasesToolsBeforePersistingTerminal(t *testing.T) {
	store := &executorStoreStub{}
	toolSet := &executorToolSetStub{released: make(chan struct{})}
	engine := &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}}
	executor := NewRunExecutor(store, &executorToolProviderStub{set: toolSet}, engine)

	handle, err := executor.Start(context.Background(), "run-1", []llmcore.Message{{Role: model.RoleUser, ContentText: "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	waitExecutorDone(t, handle)

	if got, want := store.events(), []string{"start", "finish:succeeded"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
	select {
	case <-toolSet.released:
	default:
		t.Fatal("tool set was not released")
	}
	if len(engine.seen.Messages) != 1 || engine.seen.Messages[0].ContentText != "hello" {
		t.Fatalf("engine input messages = %+v", engine.seen.Messages)
	}
}

func TestRunExecutorBoundsTerminalPersistenceAfterCancelledExecution(t *testing.T) {
	store := &persistenceContextStore{}
	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()
	engine := &executorEngineStub{
		result:  RunExecutionResult{Kind: RunExecutionSucceeded},
		started: make(chan struct{}), waitForCancel: true,
	}
	executor := NewRunExecutor(
		store,
		&executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}},
		engine,
	)

	handle, err := executor.Start(parentCtx, "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-engine.started:
	case <-time.After(time.Second):
		t.Fatal("Engine did not start")
	}
	cancelParent()
	waitExecutorDone(t, handle)

	if store.persistenceCtx == nil {
		t.Fatal("terminal persistence did not receive a context")
	}
	if err := store.persistenceErr; err != nil {
		t.Fatalf("terminal persistence context err = %v, want nil during persistence", err)
	}
	deadline, ok := store.persistenceCtx.Deadline()
	if !ok {
		t.Fatal("terminal persistence context has no deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > runPersistenceTimeout {
		t.Fatalf("terminal persistence deadline remaining = %s, want within (0, %s]", remaining, runPersistenceTimeout)
	}
}

func TestRunExecutorUsesFreshContextToReconcilePersistenceTimeout(t *testing.T) {
	store := &persistenceTimeoutStore{}
	executor := NewRunExecutor(
		store,
		&executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}},
		&executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}},
	)
	executor.persistenceTimeout = 5 * time.Millisecond

	handle, err := executor.Start(context.Background(), "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Wait(context.Background()); err == nil {
		t.Fatal("Wait() error = nil, want persistence timeout")
	}
	if store.finishCtx == nil || store.finishErr == nil {
		t.Fatal("terminal persistence did not reach its timeout")
	}
	if store.requestErr != nil {
		t.Fatalf("request cancellation context err = %v, want nil", store.requestErr)
	}
	if len(store.reconcileErrs) != 2 || store.reconcileErrs[1] != nil {
		t.Fatalf("reconciliation context errors = %v, want a fresh active context", store.reconcileErrs)
	}
}

func TestRunExecutorCancelDuringToolAcquisitionDoesNotStartEngine(t *testing.T) {
	store := &executorStoreStub{}
	toolSet := &executorToolSetStub{released: make(chan struct{})}
	provider := &blockingToolProvider{
		set: toolSet, started: make(chan struct{}), release: make(chan struct{}),
	}
	engine := &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}}
	executor := NewRunExecutor(store, provider, engine)

	startResult := make(chan struct {
		handle *RunExecutionHandle
		err    error
	}, 1)
	go func() {
		handle, err := executor.Start(context.Background(), "run-1", nil)
		startResult <- struct {
			handle *RunExecutionHandle
			err    error
		}{handle: handle, err: err}
	}()

	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("tool acquisition did not start")
	}
	if _, err := executor.Cancel(context.Background(), "run-1"); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	close(provider.release)

	var result struct {
		handle *RunExecutionHandle
		err    error
	}
	select {
	case result = <-startResult:
	case <-time.After(time.Second):
		t.Fatal("Start() did not return")
	}
	if result.err != nil {
		t.Fatalf("Start() error = %v, want cancelled start to return a handle", result.err)
	}
	if result.handle == nil {
		t.Fatal("Start() handle = nil")
	}
	waitExecutorDone(t, result.handle)
	if engine.seen.RunID != "" {
		t.Fatalf("Engine started for cancelled Run: %+v", engine.seen)
	}
	select {
	case <-toolSet.released:
	default:
		t.Fatal("tool set was not released after cancelled acquisition")
	}
	if store.status != model.RunStatusCancelled {
		t.Fatalf("Run status = %s, want cancelled", store.status)
	}
}

func TestRunExecutorDoesNotAdmitEngineWhileCancellationIsPersisting(t *testing.T) {
	store := &cancelBarrierStore{
		requestStarted: make(chan struct{}), releaseRequest: make(chan struct{}),
	}
	toolSet := &executorToolSetStub{released: make(chan struct{})}
	provider := &blockingToolProvider{
		set: toolSet, started: make(chan struct{}), release: make(chan struct{}),
	}
	engine := &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}}
	executor := NewRunExecutor(store, provider, engine)

	startResult := make(chan struct {
		handle *RunExecutionHandle
		err    error
	}, 1)
	go func() {
		handle, err := executor.Start(context.Background(), "run-1", nil)
		startResult <- struct {
			handle *RunExecutionHandle
			err    error
		}{handle: handle, err: err}
	}()
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("tool acquisition did not start")
	}

	cancelResult := make(chan error, 1)
	go func() {
		_, err := executor.Cancel(context.Background(), "run-1")
		cancelResult <- err
	}()
	select {
	case <-store.requestStarted:
	case <-time.After(time.Second):
		t.Fatal("cancel request did not reach persistence")
	}
	close(provider.release)
	close(store.releaseRequest)
	if err := <-cancelResult; err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	result := <-startResult
	if result.err != nil {
		t.Fatalf("Start() error = %v", result.err)
	}
	waitExecutorDone(t, result.handle)
	if engine.seen.RunID != "" {
		t.Fatalf("Engine started while cancellation was persisting: %+v", engine.seen)
	}
	if store.status != model.RunStatusCancelled {
		t.Fatalf("Run status = %s, want cancelled", store.status)
	}
}

func TestRunExecutorParentCancellationDuringToolAcquisitionReconcilesRun(t *testing.T) {
	store := &strictStartCancellationStore{}
	toolSet := &executorToolSetStub{released: make(chan struct{})}
	provider := &blockingToolProvider{
		set: toolSet, started: make(chan struct{}), release: make(chan struct{}),
	}
	engine := &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}}
	executor := NewRunExecutor(store, provider, engine)
	parentCtx, cancelParent := context.WithCancel(context.Background())

	startResult := make(chan struct {
		handle *RunExecutionHandle
		err    error
	}, 1)
	go func() {
		handle, err := executor.Start(parentCtx, "run-1", nil)
		startResult <- struct {
			handle *RunExecutionHandle
			err    error
		}{handle: handle, err: err}
	}()
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("tool acquisition did not start")
	}
	cancelParent()
	close(provider.release)

	result := <-startResult
	if result.err != nil {
		t.Fatalf("Start() error = %v", result.err)
	}
	waitExecutorDone(t, result.handle)
	if engine.seen.RunID != "" {
		t.Fatalf("Engine started after parent cancellation: %+v", engine.seen)
	}
	select {
	case <-toolSet.released:
	default:
		t.Fatal("tool set was not released after parent cancellation")
	}
	if store.status != model.RunStatusCancelled {
		t.Fatalf("Run status = %s, want cancelled", store.status)
	}
}

type executorEventPublisherStub struct{}

func (executorEventPublisherStub) Publish(context.Context, string, model.BaseEvent) error { return nil }

type executorRetentionPublisherStub struct {
	marked chan string
}

func (p *executorRetentionPublisherStub) Publish(context.Context, string, model.BaseEvent) error {
	return nil
}

func (p *executorRetentionPublisherStub) MarkCompleted(_ context.Context, runID string) error {
	p.marked <- runID
	return nil
}

func TestRunExecutorPassesEventPublisherToEngine(t *testing.T) {
	engine := &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}}
	publisher := executorEventPublisherStub{}
	executor := NewRunExecutorWithEventPublisher(
		&executorStoreStub{},
		&executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}},
		engine,
		publisher,
	)

	handle, err := executor.Start(context.Background(), "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitExecutorDone(t, handle)
	if engine.seen.EventPublisher == nil {
		t.Fatal("engine input EventPublisher = nil")
	}
}

func TestRunExecutorMarksOnlyTerminalStreamsCompleted(t *testing.T) {
	tests := []struct {
		name       string
		result     RunExecutionResult
		wantMarked bool
	}{
		{name: "terminal", result: RunExecutionResult{Kind: RunExecutionSucceeded}, wantMarked: true},
		{name: "waiting input", result: RunExecutionResult{Kind: RunExecutionWaitingInput, Question: &model.RunMessage{ID: "question-1", RunID: "run-1", SessionID: "session-1", Role: model.RoleAssistant, Content: "confirm"}, Snapshot: model.RunExecutionSnapshot{SnapshotRevision: 1, PlanID: "plan-1", CurrentStepID: "step-1", Steps: []model.RunStepSnapshot{{ID: "step-1", Status: model.RunStepStatusRunning}}, WaitingCheckpoint: &model.WaitingCheckpoint{QuestionMessageID: "question-1", StepID: "step-1", ResumeMode: model.ResumeModeContinueStep}}}, wantMarked: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &executorRetentionPublisherStub{marked: make(chan string, 1)}
			executor := NewRunExecutorWithEventPublisher(&executorStoreStub{}, &executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}}, &executorEngineStub{result: tt.result}, publisher)
			handle, err := executor.Start(context.Background(), "run-1", nil)
			if err != nil {
				t.Fatal(err)
			}
			waitExecutorDone(t, handle)
			select {
			case runID := <-publisher.marked:
				if !tt.wantMarked || runID != "run-1" {
					t.Fatalf("marked run id = %q, want marked=%v", runID, tt.wantMarked)
				}
			default:
				if tt.wantMarked {
					t.Fatal("terminal stream was not marked completed")
				}
			}
		})
	}
}

func TestRunExecutorWaitingPersistsCheckpointAfterEngineExit(t *testing.T) {
	store := &executorStoreStub{}
	toolSet := &executorToolSetStub{released: make(chan struct{})}
	question := &model.RunMessage{ID: "question-1", RunID: "run-1", SessionID: "session-1", Role: model.RoleAssistant, Content: "confirm?"}
	engine := &executorEngineStub{result: RunExecutionResult{
		Kind:     RunExecutionWaitingInput,
		Snapshot: model.RunExecutionSnapshot{SnapshotRevision: 1, PlanID: "plan-1", CurrentStepID: "step-1", Steps: []model.RunStepSnapshot{{ID: "step-1", Description: "执行并请求用户确认", Status: model.RunStepStatusRunning}}, WaitingCheckpoint: &model.WaitingCheckpoint{QuestionMessageID: question.ID, StepID: "step-1", ResumeMode: model.ResumeModeContinueStep}},
		Question: question,
	}}
	executor := NewRunExecutor(store, &executorToolProviderStub{set: toolSet}, engine)

	handle, err := executor.Start(context.Background(), "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitExecutorDone(t, handle)

	if got, want := store.events(), []string{"start", "waiting"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
	if store.status != model.RunStatusWaitingInput {
		t.Fatalf("status = %s, want waiting_input", store.status)
	}
}

func TestRunExecutorReconcilesWaitingPersistenceFailure(t *testing.T) {
	store := &executorStoreStub{waitingErr: errors.New("database unavailable")}
	question := &model.RunMessage{ID: "question-1", RunID: "run-1", SessionID: "session-1", Role: model.RoleAssistant, Content: "confirm?"}
	engine := &executorEngineStub{result: RunExecutionResult{
		Kind:     RunExecutionWaitingInput,
		Snapshot: model.RunExecutionSnapshot{SnapshotRevision: 1, PlanID: "plan-1", CurrentStepID: "step-1", Steps: []model.RunStepSnapshot{{ID: "step-1", Description: "执行并请求用户确认", Status: model.RunStepStatusRunning}}, WaitingCheckpoint: &model.WaitingCheckpoint{QuestionMessageID: question.ID, StepID: "step-1", ResumeMode: model.ResumeModeContinueStep}},
		Question: question,
	}}
	executor := NewRunExecutor(store, &executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}}, engine)

	handle, err := executor.Start(context.Background(), "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Wait(context.Background()); err == nil {
		t.Fatal("Wait() error = nil, want waiting persistence error")
	}

	if store.status != model.RunStatusCancelled {
		t.Fatalf("status = %s, want cancelled after waiting persistence failure", store.status)
	}
	if got, want := store.events(), []string{"start", "waiting", "reconcile"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
}

func TestRunExecutorPanicBecomesFailedAndReleasesTools(t *testing.T) {
	store := &executorStoreStub{}
	toolSet := &executorToolSetStub{released: make(chan struct{})}
	executor := NewRunExecutor(store, &executorToolProviderStub{set: toolSet}, &executorEngineStub{panic: true})

	handle, err := executor.Start(context.Background(), "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitExecutorDone(t, handle)

	if got, want := store.events(), []string{"start", "finish:failed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
	select {
	case <-toolSet.released:
	default:
		t.Fatal("tool set was not released after panic")
	}
}

func TestRunExecutorCancelWithoutHandleReconcilesImmediately(t *testing.T) {
	store := &executorStoreStub{}
	executor := NewRunExecutor(store, &executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}}, &executorEngineStub{})

	if _, err := executor.Cancel(context.Background(), "run-1"); err != nil {
		t.Fatal(err)
	}
	if got, want := store.events(), []string{"request_cancel", "reconcile"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
}

func TestRunExecutorReconcilesCancellationAfterLateEngineResult(t *testing.T) {
	store := &cancellationRaceStore{finishStarted: make(chan struct{}), releaseFinish: make(chan struct{})}
	engine := &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}}
	executor := NewRunExecutor(store, &executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}}, engine)

	handle, err := executor.Start(context.Background(), "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-store.finishStarted:
	case <-time.After(time.Second):
		t.Fatal("terminal persistence did not start")
	}
	if _, err := executor.Cancel(context.Background(), "run-1"); err != nil {
		t.Fatal(err)
	}
	close(store.releaseFinish)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := handle.Wait(ctx); err == nil {
		t.Fatal("Wait() error = nil, want late terminal persistence error")
	}
	status, events := store.snapshot()
	if status != model.RunStatusCancelled {
		t.Fatalf("status = %s, want cancelled", status)
	}
	if got, want := events, []string{"start", "finish:succeeded", "request_cancel", "reconcile"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
}

func TestRunExecutorToolAcquisitionFailureReconcilesStartedRun(t *testing.T) {
	store := &executorStoreStub{}
	executor := NewRunExecutor(
		store,
		&executorToolProviderStub{err: errors.New("mcp unavailable")},
		&executorEngineStub{},
	)

	if _, err := executor.Start(context.Background(), "run-1", nil); err == nil {
		t.Fatal("Start() error = nil, want tool acquisition error")
	}
	if got, want := store.events(), []string{"start", "request_cancel", "reconcile"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
}

func TestRunExecutorNilToolSetReconcilesStartedRun(t *testing.T) {
	store := &executorStoreStub{}
	executor := NewRunExecutor(
		store,
		&executorToolProviderStub{},
		&executorEngineStub{},
	)

	if _, err := executor.Start(context.Background(), "run-1", nil); err == nil {
		t.Fatal("Start() error = nil, want nil tool set error")
	}
	if got, want := store.events(), []string{"start", "request_cancel", "reconcile"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store events = %v, want %v", got, want)
	}
}

func TestRunExecutorRemovesControlWhenTerminalPersistenceFails(t *testing.T) {
	store := &executorStoreStub{finishErr: errors.New("database unavailable")}
	executor := NewRunExecutor(store, &executorToolProviderStub{set: &executorToolSetStub{released: make(chan struct{})}}, &executorEngineStub{result: RunExecutionResult{Kind: RunExecutionSucceeded}})

	handle, err := executor.Start(context.Background(), "run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-handle.Done():
	case <-time.After(time.Second):
		t.Fatal("executor did not finish")
	}
	if err := handle.Wait(context.Background()); err == nil {
		t.Fatal("Wait() error = nil, want terminal persistence error")
	}
	executor.mu.Lock()
	_, found := executor.control["run-1"]
	executor.mu.Unlock()
	if found {
		t.Fatal("execution control remains after terminal persistence failure")
	}
	if store.status != model.RunStatusCancelled {
		t.Fatalf("status = %s, want cancelled after persistence failure", store.status)
	}
	events := store.events()
	if !reflect.DeepEqual(events, []string{"start", "finish:succeeded", "reconcile"}) {
		t.Fatalf("store events = %v", events)
	}
}
