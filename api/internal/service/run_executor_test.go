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
	mu        sync.Mutex
	status    model.RunStatus
	steps     []string
	finishErr error
	finished  chan struct{}
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

type executorEngineStub struct {
	result RunExecutionResult
	err    error
	panic  bool
	seen   RunExecutionInput
}

func (e *executorEngineStub) Execute(_ context.Context, input RunExecutionInput) (RunExecutionResult, error) {
	e.seen = input
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

type executorEventPublisherStub struct{}

func (executorEventPublisherStub) Publish(context.Context, string, model.BaseEvent) error { return nil }

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

func TestRunExecutorKeepsHandleWhenTerminalPersistenceFails(t *testing.T) {
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

	if _, err := executor.Cancel(context.Background(), "run-1"); err != nil {
		t.Fatal(err)
	}
	events := store.events()
	if !reflect.DeepEqual(events, []string{"start", "finish:succeeded", "request_cancel", "reconcile"}) {
		t.Fatalf("store events = %v", events)
	}
}
