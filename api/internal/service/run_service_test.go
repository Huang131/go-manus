package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/settings"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunStore struct {
	run             *model.Run
	messages        []*model.RunMessage
	transitionCalls int
	terminal        repository.TerminalTransition
	recoveryStats   repository.RunRecoveryStats
}

func (f *fakeRunStore) RecoverStaleRuns(context.Context) (repository.RunRecoveryStats, error) {
	return f.recoveryStats, nil
}

func (f *fakeRunStore) CreateWithInitialMessage(_ context.Context, run *model.Run, initial *model.RunMessage) (*model.Run, bool, error) {
	if f.run != nil {
		if f.run.SessionID == run.SessionID && f.run.IdempotencyKey == run.IdempotencyKey {
			return f.run, false, nil
		}
		return nil, false, errors.New("active run conflict")
	}
	f.run = run
	f.messages = append(f.messages, initial)
	return run, true, nil
}

type conflictingRunStore struct {
	fakeRunStore
}

func (f *conflictingRunStore) CreateWithInitialMessage(_ context.Context, _ *model.Run, _ *model.RunMessage) (*model.Run, bool, error) {
	return nil, false, repository.ErrActiveRunExists
}

type startRefreshErrorStore struct {
	fakeRunStore
	getCalls int
}

func (f *startRefreshErrorStore) GetByID(_ context.Context, id string) (*model.Run, error) {
	f.getCalls++
	if f.getCalls > 1 {
		return nil, errors.New("refresh started run failed")
	}
	return f.fakeRunStore.GetByID(context.Background(), id)
}

type resumeNoRefreshStore struct {
	fakeRunStore
	getCalls int
}

func (f *resumeNoRefreshStore) GetByID(_ context.Context, id string) (*model.Run, error) {
	f.getCalls++
	if f.getCalls > 1 {
		return nil, errors.New("unexpected refresh after resume")
	}
	return f.fakeRunStore.GetByID(context.Background(), id)
}

type transitionNoRefreshStore struct {
	fakeRunStore
	getCalls int
}

func (f *transitionNoRefreshStore) GetByID(_ context.Context, id string) (*model.Run, error) {
	f.getCalls++
	if f.getCalls > 1 {
		return nil, errors.New("unexpected refresh after transition")
	}
	return f.fakeRunStore.GetByID(context.Background(), id)
}

type waitingNoRefreshStore struct {
	fakeRunStore
	getCalls int
}

func (f *waitingNoRefreshStore) GetByID(_ context.Context, id string) (*model.Run, error) {
	f.getCalls++
	if f.getCalls > 1 {
		return nil, errors.New("unexpected refresh after waiting transition")
	}
	return f.fakeRunStore.GetByID(context.Background(), id)
}

// staleInputRetryStore 模拟读取消息后，另一个相同幂等请求已在事务内写入回答的竞态。
type staleInputRetryStore struct {
	fakeRunStore
}

func (f *staleInputRetryStore) ResumeWaitingInput(_ context.Context, runID string, expectedRevision int, answer *model.RunMessage) (repository.ResumeWaitingInputResult, error) {
	if f.run == nil || f.run.ID != runID || f.run.SnapshotRevision != expectedRevision {
		return repository.ResumeWaitingInputResult{Outcome: repository.ResumeWaitingInputNoChange}, nil
	}
	f.run.Status = model.RunStatusRunning
	f.run.WaitingMessageID = ""
	f.messages = append(f.messages, answer)
	return repository.ResumeWaitingInputResult{Outcome: repository.ResumeWaitingInputAlreadyApplied, Run: f.run}, nil
}

func TestRunServiceCreateReturnsConflictWhenSessionAlreadyActive(t *testing.T) {
	svc := NewRunService(&conflictingRunStore{})
	_, err := svc.Create(context.Background(), CreateRunInput{
		SessionID: "session-1", IdempotencyKey: "request-1", PromptHash: "prompt-v1",
		SettingsSnapshot: settings.DefaultAgentSettings(), Content: "分析报告",
	})
	appErr, ok := err.(*apperr.Error)
	if !ok || appErr.Kind != apperr.KindConflict {
		t.Fatalf("Create() error = %T %v, want conflict", err, err)
	}
}

func (f *fakeRunStore) GetByID(_ context.Context, id string) (*model.Run, error) {
	if f.run == nil || f.run.ID != id {
		return nil, nil
	}
	// PostgreSQL Repository 会在每次查询时重建 Run；返回副本避免测试因共享指针
	// 掩盖“持久化后仍返回旧快照”的问题。
	cloned := *f.run
	cloned.ExecutionSnapshot = f.run.ExecutionSnapshot.Clone()
	return &cloned, nil
}

func (f *fakeRunStore) FindBySessionAndIdempotencyKey(_ context.Context, sessionID, idempotencyKey string) (*model.Run, error) {
	if f.run == nil || f.run.SessionID != sessionID || f.run.IdempotencyKey != idempotencyKey {
		return nil, nil
	}
	return f.GetByID(context.Background(), f.run.ID)
}

func (f *fakeRunStore) GetActiveBySessionID(_ context.Context, sessionID string) (*model.Run, error) {
	if f.run == nil || f.run.SessionID != sessionID || !f.run.Status.IsActive() {
		return nil, nil
	}
	return f.run, nil
}

func (f *fakeRunStore) ListBySessionID(_ context.Context, sessionID string, _, _ int) ([]*model.Run, int, error) {
	if f.run == nil || f.run.SessionID != sessionID {
		return nil, 0, nil
	}
	return []*model.Run{f.run}, 1, nil
}

func (f *fakeRunStore) ListMessages(_ context.Context, _ string) ([]*model.RunMessage, error) {
	return append([]*model.RunMessage(nil), f.messages...), nil
}

func (f *fakeRunStore) ListMessagesByRunIDs(_ context.Context, runIDs []string) (map[string][]*model.RunMessage, error) {
	result := make(map[string][]*model.RunMessage, len(runIDs))
	for _, runID := range runIDs {
		for _, message := range f.messages {
			if message.RunID == runID {
				result[runID] = append(result[runID], message)
			}
		}
	}
	return result, nil
}

func (f *fakeRunStore) TransitionStatus(_ context.Context, id string, from []model.RunStatus, to model.RunStatus) (bool, error) {
	f.transitionCalls++
	if f.run == nil || f.run.ID != id {
		return false, nil
	}
	for _, status := range from {
		if f.run.Status == status && status.CanTransitionTo(to) {
			f.run.Status = to
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRunStore) EnterWaitingInput(_ context.Context, runID string, expectedRevision int, snapshot model.RunExecutionSnapshot, question *model.RunMessage) (bool, error) {
	if f.run == nil || f.run.ID != runID || f.run.Status != model.RunStatusRunning || f.run.SnapshotRevision != expectedRevision {
		return false, nil
	}
	f.run.Status = model.RunStatusWaitingInput
	f.run.ExecutionSnapshot = snapshot.Clone()
	f.run.SnapshotRevision = snapshot.SnapshotRevision
	f.run.WaitingMessageID = question.ID
	f.messages = append(f.messages, question)
	return true, nil
}

func (f *fakeRunStore) ResumeWaitingInput(_ context.Context, runID string, expectedRevision int, answer *model.RunMessage) (repository.ResumeWaitingInputResult, error) {
	if f.run == nil || f.run.ID != runID || f.run.Status != model.RunStatusWaitingInput || f.run.SnapshotRevision != expectedRevision {
		return repository.ResumeWaitingInputResult{Outcome: repository.ResumeWaitingInputNoChange}, nil
	}
	f.run.Status = model.RunStatusRunning
	f.run.WaitingMessageID = ""
	f.messages = append(f.messages, answer)
	return repository.ResumeWaitingInputResult{Outcome: repository.ResumeWaitingInputResumed}, nil
}

func (f *fakeRunStore) FinishTerminal(_ context.Context, runID string, terminal repository.TerminalTransition, final *model.RunMessage) (bool, error) {
	if f.run == nil || f.run.ID != runID || !f.run.Status.CanTransitionTo(terminal.Status) {
		return false, nil
	}
	f.run.Status = terminal.Status
	f.run.ExecutionSnapshot = terminal.Snapshot.Clone()
	f.run.SnapshotRevision = terminal.Snapshot.SnapshotRevision
	f.run.ErrorCode = terminal.ErrorCode
	f.run.ErrorMessage = terminal.ErrorMessage
	f.terminal = terminal
	if final != nil {
		f.messages = append(f.messages, final)
	}
	return true, nil
}

func newFakeRunStore() *fakeRunStore {
	return &fakeRunStore{}
}

func waitingRunFixture() (*fakeRunStore, *model.RunMessage, *model.RunMessage) {
	now := time.Now().UTC()
	run := &model.Run{
		ID: "run-1", SessionID: "session-1", Status: model.RunStatusWaitingInput,
		SettingsSnapshot: settings.DefaultAgentSettings(), PromptHash: "prompt-v1",
		SnapshotRevision: 1, WaitingMessageID: "question-1", CreatedAt: now, UpdatedAt: now,
		ExecutionSnapshot: model.RunExecutionSnapshot{
			SnapshotRevision: 1, PlanID: "plan-1", CurrentStepID: "step-1",
			Steps:             []model.RunStepSnapshot{{ID: "step-1", Description: "分析任务并请求确认", Status: model.RunStepStatusRunning}},
			WaitingCheckpoint: &model.WaitingCheckpoint{QuestionMessageID: "question-1", StepID: "step-1", ResumeMode: model.ResumeModeContinueStep},
		},
	}
	initial := &model.RunMessage{ID: "input-1", SessionID: run.SessionID, RunID: run.ID, Role: model.RoleUser, Content: "分析报告", CreatedAt: now}
	question := &model.RunMessage{ID: "question-1", SessionID: run.SessionID, RunID: run.ID, Role: model.RoleAssistant, Content: "请确认", CreatedAt: now.Add(time.Second)}
	store := newFakeRunStore()
	store.run = run
	store.messages = []*model.RunMessage{initial, question}
	return store, initial, question
}

func TestRunServiceCreateBuildsIdempotentAggregate(t *testing.T) {
	store := newFakeRunStore()
	svc := NewRunService(store)
	input := CreateRunInput{
		SessionID: "session-1", IdempotencyKey: "request-1", PromptHash: "prompt-v1",
		SettingsSnapshot: settings.DefaultAgentSettings(), Content: "分析报告", Attachments: []string{"file-1"},
	}
	run, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if run.ID == "" || run.Status != model.RunStatusPending || run.SessionID != "session-1" {
		t.Fatalf("Create() returned incomplete run: %#v", run)
	}
	if len(store.messages) != 1 || store.messages[0].IdempotencyKey != "request-1" {
		t.Fatalf("initial message = %#v", store.messages)
	}
	retry, err := svc.Create(context.Background(), input)
	if err != nil || retry.ID != run.ID || len(store.messages) != 1 {
		t.Fatalf("idempotent Create() = %#v, %v, messages=%d", retry, err, len(store.messages))
	}
	input.Content = "不同内容"
	if _, err := svc.Create(context.Background(), input); err == nil {
		t.Fatal("Create() accepted different payload with the same idempotency key")
	}
}

func TestRunServiceFinishPersistsConcreteFailureMessage(t *testing.T) {
	store := newFakeRunStore()
	store.run = &model.Run{
		ID: "run-1", SessionID: "session-1", Status: model.RunStatusRunning,
		SettingsSnapshot: settings.DefaultAgentSettings(), PromptHash: "prompt-v1",
	}

	err := NewRunService(store).Finish(context.Background(), store.run.ID, RunExecutionResult{
		Kind:  RunExecutionFailed,
		Error: errors.New("provider deepseek/deepseek-v4-flash error [rate_limit] status=429: usage limit"),
	})
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if store.run.Status != model.RunStatusFailed || store.run.ErrorCode != "execution_failed" {
		t.Fatalf("run terminal state = %#v, want failed execution", store.run)
	}
	if store.run.ErrorMessage != "provider deepseek/deepseek-v4-flash error [rate_limit] status=429: usage limit" {
		t.Fatalf("error_message = %q, want provider detail", store.run.ErrorMessage)
	}
}

func TestRunServiceStartAndEnterWaitingInputUseCurrentRevision(t *testing.T) {
	store := newFakeRunStore()
	now := time.Now().UTC()
	store.run = &model.Run{
		ID: "run-1", SessionID: "session-1", Status: model.RunStatusPending,
		SettingsSnapshot: settings.DefaultAgentSettings(), PromptHash: "prompt-v1",
		CreatedAt: now, UpdatedAt: now,
	}
	started, err := NewRunService(store).Start(context.Background(), store.run.ID)
	if err != nil || started.Status != model.RunStatusRunning {
		t.Fatalf("Start() = %#v, %v", started, err)
	}
	question := &model.RunMessage{ID: "question-1", SessionID: store.run.SessionID, RunID: store.run.ID, Role: model.RoleAssistant, Content: "请确认"}
	snapshot := model.RunExecutionSnapshot{
		SnapshotRevision: 1, PlanID: "plan-1", CurrentStepID: "step-1",
		Steps:             []model.RunStepSnapshot{{ID: "step-1", Description: "分析任务并请求确认", Status: model.RunStepStatusRunning}},
		WaitingCheckpoint: &model.WaitingCheckpoint{QuestionMessageID: question.ID, StepID: "step-1", ResumeMode: model.ResumeModeContinueStep},
	}
	waiting, err := NewRunService(store).EnterWaitingInput(context.Background(), store.run.ID, snapshot, question)
	if err != nil || waiting.Status != model.RunStatusWaitingInput || waiting.WaitingMessageID != question.ID {
		t.Fatalf("EnterWaitingInput() = %#v, %v", waiting, err)
	}
}

func TestRunServiceEnterWaitingInputReturnsCommittedStateWithoutRefresh(t *testing.T) {
	base := newFakeRunStore()
	now := time.Now().UTC()
	base.run = &model.Run{
		ID: "run-1", SessionID: "session-1", Status: model.RunStatusRunning,
		SettingsSnapshot: settings.DefaultAgentSettings(), PromptHash: "prompt-v1",
		SnapshotRevision: 0, CreatedAt: now, UpdatedAt: now,
	}
	question := &model.RunMessage{ID: "question-1", SessionID: "session-1", RunID: "run-1", Role: model.RoleAssistant, Content: "请确认"}
	snapshot := model.RunExecutionSnapshot{
		SnapshotRevision: 1, PlanID: "plan-1", CurrentStepID: "step-1",
		Steps:             []model.RunStepSnapshot{{ID: "step-1", Description: "等待确认", Status: model.RunStepStatusRunning}},
		WaitingCheckpoint: &model.WaitingCheckpoint{QuestionMessageID: question.ID, StepID: "step-1", ResumeMode: model.ResumeModeContinueStep},
	}
	store := &waitingNoRefreshStore{fakeRunStore: *base}

	run, err := NewRunService(store).EnterWaitingInput(context.Background(), "run-1", snapshot, question)

	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, model.RunStatusWaitingInput, run.Status)
	require.Equal(t, question.ID, run.WaitingMessageID)
	require.Equal(t, 1, store.getCalls)
}

func TestRunServiceStartIsIdempotentForRunningRun(t *testing.T) {
	store := newFakeRunStore()
	store.run = &model.Run{
		ID: "run-1", SessionID: "session-1", Status: model.RunStatusRunning,
		SettingsSnapshot: settings.DefaultAgentSettings(), PromptHash: "prompt-v1",
	}

	started, err := NewRunService(store).Start(context.Background(), store.run.ID)

	require.NoError(t, err)
	require.Equal(t, model.RunStatusRunning, started.Status)
	require.Zero(t, store.transitionCalls, "a resumed running Run must not be transitioned again")
}

func TestRunServiceStartReturnsRunningRunWhenRefreshFailsAfterTransition(t *testing.T) {
	store := &startRefreshErrorStore{fakeRunStore: fakeRunStore{run: &model.Run{
		ID: "run-1", SessionID: "session-1", Status: model.RunStatusPending,
		SettingsSnapshot: settings.DefaultAgentSettings(), PromptHash: "prompt-v1",
	}}}

	run, err := NewRunService(store).Start(context.Background(), store.run.ID)

	if err == nil {
		t.Fatal("Start() error = nil, want refresh error")
	}
	if run == nil || run.Status != model.RunStatusRunning {
		t.Fatalf("Start() run = %#v, want running partial result", run)
	}
}

func TestRunServiceRecoversStaleRunsAfterRestart(t *testing.T) {
	store := &fakeRunStore{recoveryStats: repository.RunRecoveryStats{Interrupted: 2, Cancelled: 1}}

	stats, err := NewRunService(store).RecoverAfterRestart(context.Background())

	require.NoError(t, err)
	require.Equal(t, store.recoveryStats, stats)
}

func TestRunServiceSubmitInputRebuildsContextAndResumesWaitingRun(t *testing.T) {
	store, _, _ := waitingRunFixture()
	svc := NewRunService(store)
	result, err := svc.SubmitInput(context.Background(), "run-1", SubmitInputRequest{
		IdempotencyKey: "answer-1", ReplyToMessageID: "question-1", Content: "继续执行", Attachments: []string{"answer-file"},
	})
	if err != nil {
		t.Fatalf("SubmitInput() error = %v", err)
	}
	if result.Run.Status != model.RunStatusRunning || len(result.Messages) != 3 {
		t.Fatalf("resume result = %#v", result)
	}
	if result.Messages[0].Role != model.RoleUser || result.Messages[1].ContentText != "请确认" || result.Messages[2].ContentText != "继续执行" {
		t.Fatalf("rebuilt messages = %#v", result.Messages)
	}
	if result.Messages[2].Attachments[0] != "answer-file" {
		t.Fatalf("answer attachments = %#v", result.Messages[2].Attachments)
	}
}

func TestRunServiceSubmitInputReturnsCommittedRunningStateWithoutRefresh(t *testing.T) {
	base, _, _ := waitingRunFixture()
	store := &resumeNoRefreshStore{fakeRunStore: *base}
	result, err := NewRunService(store).SubmitInput(context.Background(), "run-1", SubmitInputRequest{
		IdempotencyKey: "answer-1", ReplyToMessageID: "question-1", Content: "继续执行",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Run)
	require.Equal(t, model.RunStatusRunning, result.Run.Status)
	require.Empty(t, result.Run.WaitingMessageID)
	require.True(t, result.StartExecution)
	require.Len(t, store.messages, 3)
	require.Equal(t, 1, store.getCalls)
}

func TestRunServiceSubmitInputIsIdempotentAfterFirstResume(t *testing.T) {
	store, _, _ := waitingRunFixture()
	svc := NewRunService(store)
	input := SubmitInputRequest{IdempotencyKey: "answer-1", ReplyToMessageID: "question-1", Content: "继续执行"}
	first, err := svc.SubmitInput(context.Background(), "run-1", input)
	if err != nil {
		t.Fatalf("first SubmitInput() error = %v", err)
	}
	second, err := svc.SubmitInput(context.Background(), "run-1", input)
	if err != nil {
		t.Fatalf("repeated SubmitInput() error = %v", err)
	}
	if second.Run.Status != model.RunStatusRunning || !second.AlreadyApplied || len(store.messages) != 3 {
		t.Fatalf("repeated SubmitInput() = %#v, messages=%d", second, len(store.messages))
	}
	if len(second.Messages) != 0 || !first.StartExecution {
		t.Fatalf("duplicate input should not start a second execution: first=%#v second=%#v", first, second)
	}
}

func TestRunServiceSubmitInputDoesNotRestartWhenTransactionFindsSameAnswer(t *testing.T) {
	base, _, _ := waitingRunFixture()
	store := &staleInputRetryStore{fakeRunStore: *base}
	svc := NewRunService(store)

	result, err := svc.SubmitInput(context.Background(), "run-1", SubmitInputRequest{
		IdempotencyKey: "answer-1", ReplyToMessageID: "question-1", Content: "继续执行",
	})

	require.NoError(t, err)
	require.True(t, result.AlreadyApplied)
	require.False(t, result.StartExecution)
	require.Empty(t, result.Messages)
}

func TestRunServiceSubmitInputRejectsInvalidSnapshotBeforeMutation(t *testing.T) {
	store, _, _ := waitingRunFixture()
	store.run.ExecutionSnapshot.Steps[0].Status = model.RunStepStatusCompleted
	store.run.ExecutionSnapshot.Steps[0].ResultSummary = ""
	svc := NewRunService(store)
	_, err := svc.SubmitInput(context.Background(), "run-1", SubmitInputRequest{
		IdempotencyKey: "answer-1", ReplyToMessageID: "question-1", Content: "继续执行",
	})
	appErr, ok := err.(*apperr.Error)
	if !ok || appErr.Kind != apperr.KindFailedPrecondition {
		t.Fatalf("SubmitInput() error = %T %v, want failed precondition", err, err)
	}
	if store.run.Status != model.RunStatusWaitingInput || len(store.messages) != 2 {
		t.Fatalf("invalid snapshot mutated store: status=%s messages=%d", store.run.Status, len(store.messages))
	}
}

func TestRunServiceSubmitInputRejectsCancellingRun(t *testing.T) {
	store, _, _ := waitingRunFixture()
	store.run.Status = model.RunStatusCancelling
	svc := NewRunService(store)
	_, err := svc.SubmitInput(context.Background(), store.run.ID, SubmitInputRequest{
		IdempotencyKey: "answer-1", ReplyToMessageID: "question-1", Content: "继续执行",
	})
	appErr, ok := err.(*apperr.Error)
	if !ok || appErr.Kind != apperr.KindConflict {
		t.Fatalf("SubmitInput() error = %T %v, want conflict", err, err)
	}
	if store.transitionCalls != 0 {
		t.Fatalf("SubmitInput() changed cancelling run")
	}
}

func TestRunServiceCancelAndReconcileAreIdempotent(t *testing.T) {
	store, _, _ := waitingRunFixture()
	svc := NewRunService(store)
	cancelled, err := svc.RequestCancel(context.Background(), store.run.ID)
	if err != nil || cancelled.Status != model.RunStatusCancelling {
		t.Fatalf("RequestCancel() = %#v, %v", cancelled, err)
	}
	reconciled, err := svc.ReconcileCancelling(context.Background(), store.run.ID)
	if err != nil || reconciled.Status != model.RunStatusCancelled {
		t.Fatalf("ReconcileCancelling() = %#v, %v", reconciled, err)
	}
	reconciled, err = svc.ReconcileCancelling(context.Background(), store.run.ID)
	if err != nil || reconciled.Status != model.RunStatusCancelled {
		t.Fatalf("repeated ReconcileCancelling() = %#v, %v", reconciled, err)
	}
}

func TestRunServiceRequestCancelReturnsCommittedStateWithoutRefresh(t *testing.T) {
	base, _, _ := waitingRunFixture()
	store := &transitionNoRefreshStore{fakeRunStore: *base}

	run, err := NewRunService(store).RequestCancel(context.Background(), "run-1")

	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, model.RunStatusCancelling, run.Status)
	require.Equal(t, 1, store.getCalls)
}

func TestRunServiceReconcileCancellingReturnsCommittedStateWithoutRefresh(t *testing.T) {
	base, _, _ := waitingRunFixture()
	base.run.Status = model.RunStatusCancelling
	store := &transitionNoRefreshStore{fakeRunStore: *base}

	run, err := NewRunService(store).ReconcileCancelling(context.Background(), "run-1")

	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, model.RunStatusCancelled, run.Status)
	require.Equal(t, 1, store.getCalls)
}

func TestRunServiceFinishPersistsTerminalMessageWithRunState(t *testing.T) {
	store, _, _ := waitingRunFixture()
	store.run.Status = model.RunStatusRunning
	service := NewRunService(store)
	snapshot := model.RunExecutionSnapshot{SnapshotRevision: 2, PlanID: "plan-1", CurrentStepID: "step-2"}
	summary := "# 完整文档总结\n\n" + strings.Repeat("文档的章节、操作流程和模块术语。\n", 40)

	err := service.Finish(context.Background(), store.run.ID, RunExecutionResult{
		Kind: RunExecutionSucceeded, Snapshot: snapshot, Text: summary,
	})
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if store.run.Status != model.RunStatusSucceeded || store.run.SnapshotRevision != 2 {
		t.Fatalf("terminal run = status %s snapshot revision %d", store.run.Status, store.run.SnapshotRevision)
	}
	if store.terminal.Snapshot.SnapshotRevision != 2 {
		t.Fatalf("terminal snapshot = %+v", store.terminal.Snapshot)
	}
	if len(store.messages) != 3 || store.messages[2].Role != model.RoleAssistant || store.messages[2].Content != summary {
		t.Fatalf("terminal messages = %+v", store.messages)
	}
}

func TestRunServiceFinishRejectsLateSuccessAfterCancellation(t *testing.T) {
	store, _, _ := waitingRunFixture()
	store.run.Status = model.RunStatusCancelling
	service := NewRunService(store)

	err := service.Finish(context.Background(), store.run.ID, RunExecutionResult{
		Kind: RunExecutionSucceeded, Text: "迟到结果",
	})
	appErr, ok := err.(*apperr.Error)
	if !ok || appErr.Kind != apperr.KindConflict {
		t.Fatalf("Finish() error = %T %v, want conflict", err, err)
	}
	if store.run.Status != model.RunStatusCancelling || len(store.messages) != 2 {
		t.Fatalf("late result changed run: status=%s messages=%d", store.run.Status, len(store.messages))
	}
}

func TestRunServiceGetMissingRunReturnsNotFound(t *testing.T) {
	svc := NewRunService(newFakeRunStore())
	_, err := svc.Get(context.Background(), uuid.NewString())
	appErr, ok := err.(*apperr.Error)
	if !ok || appErr.Kind != apperr.KindNotFound {
		t.Fatalf("Get() error = %T %v, want not found", err, err)
	}
}

func TestRunServiceGetActiveBySessionIDUsesRunStatus(t *testing.T) {
	store := newFakeRunStore()
	store.run = &model.Run{ID: "run-1", SessionID: "session-1", Status: model.RunStatusWaitingInput}
	svc := NewRunService(store)

	run, err := svc.GetActiveBySessionID(context.Background(), "session-1")
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, "run-1", run.ID)

	store.run.Status = model.RunStatusSucceeded
	run, err = svc.GetActiveBySessionID(context.Background(), "session-1")
	require.NoError(t, err)
	assert.Nil(t, run)
}
