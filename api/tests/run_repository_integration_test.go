//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/settings"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRunRepository(t *testing.T) repository.RunRepository {
	t.Helper()
	return repository.NewRunRepository(testDB)
}

func newRunForTest(sessionID, idempotencyKey string) *model.Run {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &model.Run{
		ID:               uuid.NewString(),
		SessionID:        sessionID,
		Status:           model.RunStatusPending,
		IdempotencyKey:   idempotencyKey,
		SettingsSnapshot: settings.DefaultAgentSettings(),
		PromptHash:       "catalog-v1-test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func initialRunMessage(run *model.Run) *model.RunMessage {
	return &model.RunMessage{
		ID:             uuid.NewString(),
		SessionID:      run.SessionID,
		RunID:          run.ID,
		IdempotencyKey: run.IdempotencyKey,
		Role:           model.RoleUser,
		Content:        "请帮我分析这份报告",
		Attachments:    []string{"file-1"},
		CreatedAt:      run.CreatedAt,
	}
}

func TestRunRepo_CreateWithInitialMessage_PersistsOneAggregate(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "create-run-"+uuid.NewString())
	initial := initialRunMessage(run)
	created, inserted, err := repo.CreateWithInitialMessage(ctx, run, initial)
	require.NoError(t, err)
	require.True(t, inserted)
	require.NotNil(t, created)
	assert.Equal(t, run.ID, created.ID)

	got, err := repo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, run.SessionID, got.SessionID)
	assert.Equal(t, model.RunStatusPending, got.Status)
	assert.Equal(t, run.IdempotencyKey, got.IdempotencyKey)
	assert.Equal(t, run.SettingsSnapshot, got.SettingsSnapshot)
	assert.Equal(t, run.PromptHash, got.PromptHash)

	messages, err := repo.ListMessages(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, initial.ID, messages[0].ID)
	assert.Equal(t, model.RoleUser, messages[0].Role)
	assert.Equal(t, initial.Attachments, messages[0].Attachments)
}

func TestRunRepo_CreateWithInitialMessage_EnforcesSingleActiveRun(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	first := newRunForTest(sessionID, "first-"+uuid.NewString())
	_, inserted, err := repo.CreateWithInitialMessage(ctx, first, initialRunMessage(first))
	require.NoError(t, err)
	require.True(t, inserted)

	second := newRunForTest(sessionID, "second-"+uuid.NewString())
	_, _, err = repo.CreateWithInitialMessage(ctx, second, initialRunMessage(second))
	require.Error(t, err)

	got, err := repo.GetByID(ctx, second.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "failed aggregate creation must roll back the Run row")
}

// TestRunRepoGetActiveBySessionID 守护 Session 路由的执行定位契约：
// 活跃 Run 只由 runs.status 判定，终态 Run 不应再占用会话执行槽位。
func TestRunRepoGetActiveBySessionID(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "active-"+uuid.NewString())
	_, inserted, err := repo.CreateWithInitialMessage(ctx, run, initialRunMessage(run))
	require.NoError(t, err)
	require.True(t, inserted)

	active, err := repo.GetActiveBySessionID(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, run.ID, active.ID)

	changed, err := repo.TransitionStatus(ctx, run.ID, []model.RunStatus{model.RunStatusPending}, model.RunStatusCancelling)
	require.NoError(t, err)
	require.True(t, changed)
	changed, err = repo.TransitionStatus(ctx, run.ID, []model.RunStatus{model.RunStatusCancelling}, model.RunStatusCancelled)
	require.NoError(t, err)
	require.True(t, changed)
	active, err = repo.GetActiveBySessionID(ctx, sessionID)
	require.NoError(t, err)
	assert.Nil(t, active)
}

func TestRunRepo_CreateWithInitialMessage_ReusesOriginalAggregateForSameClientRequest(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	first := newRunForTest(sessionID, "retry-"+uuid.NewString())
	initial := initialRunMessage(first)
	created, inserted, err := repo.CreateWithInitialMessage(ctx, first, initial)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, first.ID, created.ID)

	retry := newRunForTest(sessionID, first.IdempotencyKey)
	retryInitial := initialRunMessage(retry)
	retryInitial.Content = initial.Content
	retryInitial.Attachments = append([]string(nil), initial.Attachments...)
	created, inserted, err = repo.CreateWithInitialMessage(ctx, retry, retryInitial)
	require.NoError(t, err)
	assert.False(t, inserted)
	assert.Equal(t, first.ID, created.ID)

	messages, err := repo.ListMessages(ctx, first.ID)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
}

func TestRunRepo_WaitAndResume_UsesSnapshotRevisionAndInputIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "wait-"+uuid.NewString())
	_, inserted, err := repo.CreateWithInitialMessage(ctx, run, initialRunMessage(run))
	require.NoError(t, err)
	require.True(t, inserted)

	changed, err := repo.TransitionStatus(ctx, run.ID, []model.RunStatus{model.RunStatusPending}, model.RunStatusRunning)
	require.NoError(t, err)
	require.True(t, changed)

	question := &model.RunMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		RunID:     run.ID,
		Role:      model.RoleAssistant,
		Content:   "请确认要继续的范围",
		CreatedAt: time.Now().UTC(),
	}
	snapshot := model.RunExecutionSnapshot{
		SnapshotRevision: 1,
		PlanID:           "plan-1",
		PlanRevision:     1,
		CurrentStepID:    "step-2",
		Steps: []model.RunStepSnapshot{
			{ID: "step-1", Description: "读取报告", Status: model.RunStepStatusCompleted, ResultSummary: "已读取报告"},
			{ID: "step-2", Description: "根据确认继续执行", Status: model.RunStepStatusRunning},
		},
		WaitingCheckpoint: &model.WaitingCheckpoint{
			QuestionMessageID: question.ID,
			StepID:            "step-2",
			ResumeMode:        model.ResumeModeContinueStep,
		},
	}

	changed, err = repo.EnterWaitingInput(ctx, run.ID, 0, snapshot, question)
	require.NoError(t, err)
	require.True(t, changed)

	staleChanged, err := repo.EnterWaitingInput(ctx, run.ID, 0, snapshot, question)
	require.NoError(t, err)
	assert.False(t, staleChanged)

	answer := &model.RunMessage{
		ID:               uuid.NewString(),
		SessionID:        sessionID,
		RunID:            run.ID,
		IdempotencyKey:   "answer-" + uuid.NewString(),
		ReplyToMessageID: question.ID,
		Role:             model.RoleUser,
		Content:          "继续全部范围",
		CreatedAt:        time.Now().UTC(),
	}
	changed, err = repo.ResumeWaitingInput(ctx, run.ID, 1, answer)
	require.NoError(t, err)
	require.True(t, changed)

	changed, err = repo.ResumeWaitingInput(ctx, run.ID, 1, answer)
	require.NoError(t, err)
	assert.True(t, changed, "same client input must be idempotent")

	got, err := repo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.RunStatusRunning, got.Status)
	assert.Equal(t, 1, got.SnapshotRevision)
	assert.Empty(t, got.WaitingMessageID)

	messages, err := repo.ListMessages(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, messages, 3, "stale wait and repeated input must not add messages")
	assert.Equal(t, answer.ID, messages[2].ID)
}

func TestRunRepo_EnterWaitingInput_RollsBackWhenQuestionBelongsToAnotherSession(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "waiting-rollback-"+uuid.NewString())
	_, inserted, err := repo.CreateWithInitialMessage(ctx, run, initialRunMessage(run))
	require.NoError(t, err)
	require.True(t, inserted)

	changed, err := repo.TransitionStatus(ctx, run.ID, []model.RunStatus{model.RunStatusPending}, model.RunStatusRunning)
	require.NoError(t, err)
	require.True(t, changed)

	question := &model.RunMessage{
		ID:        uuid.NewString(),
		SessionID: uuid.NewString(),
		RunID:     run.ID,
		Role:      model.RoleAssistant,
		Content:   "请确认继续范围",
		CreatedAt: time.Now().UTC(),
	}
	snapshot := model.RunExecutionSnapshot{
		SnapshotRevision: 1,
		PlanID:           "plan-rollback",
		PlanRevision:     1,
		CurrentStepID:    "step-1",
		Steps: []model.RunStepSnapshot{
			{ID: "step-1", Description: "执行并请求用户确认", Status: model.RunStepStatusRunning},
		},
		WaitingCheckpoint: &model.WaitingCheckpoint{
			QuestionMessageID: question.ID,
			StepID:            "step-1",
			ResumeMode:        model.ResumeModeContinueStep,
		},
	}

	changed, err = repo.EnterWaitingInput(ctx, run.ID, 0, snapshot, question)
	require.Error(t, err)
	assert.False(t, changed)

	got, err := repo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.RunStatusRunning, got.Status)
	assert.Zero(t, got.SnapshotRevision)
	assert.Empty(t, got.WaitingMessageID)

	messages, err := repo.ListMessages(ctx, run.ID)
	require.NoError(t, err)
	assert.Len(t, messages, 1, "failed wait transition must not persist the question message")
}

func TestRunRepo_FinishTerminal_CommitsStatusAndFinalMessageTogether(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "finish-"+uuid.NewString())
	initial := initialRunMessage(run)
	_, inserted, err := repo.CreateWithInitialMessage(ctx, run, initial)
	require.NoError(t, err)
	require.True(t, inserted)
	changed, err := repo.TransitionStatus(ctx, run.ID, []model.RunStatus{model.RunStatusPending}, model.RunStatusRunning)
	require.NoError(t, err)
	require.True(t, changed)

	final := &model.RunMessage{
		ID: uuid.NewString(), SessionID: sessionID, RunID: run.ID,
		Role: model.RoleAssistant, Content: "执行完成", CreatedAt: time.Now().UTC(),
	}
	changed, err = repo.FinishTerminal(ctx, run.ID, repository.TerminalTransition{Status: model.RunStatusSucceeded}, final)
	require.NoError(t, err)
	require.True(t, changed)

	got, err := repo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.RunStatusSucceeded, got.Status)
	assert.NotNil(t, got.FinishedAt)

	messages, err := repo.ListMessages(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, final.ID, messages[1].ID)
	assert.Equal(t, final.Content, messages[1].Content)
}

func TestRunRepo_FinishTerminal_RollsBackStatusWhenFinalMessageFails(t *testing.T) {
	ctx := context.Background()
	repo := testRunRepository(t)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "finish-rollback-"+uuid.NewString())
	initial := initialRunMessage(run)
	_, inserted, err := repo.CreateWithInitialMessage(ctx, run, initial)
	require.NoError(t, err)
	require.True(t, inserted)
	changed, err := repo.TransitionStatus(ctx, run.ID, []model.RunStatus{model.RunStatusPending}, model.RunStatusRunning)
	require.NoError(t, err)
	require.True(t, changed)

	// 复用初始消息 ID，让插入终态消息触发主键冲突；状态更新必须随事务回滚。
	final := &model.RunMessage{
		ID: initial.ID, SessionID: sessionID, RunID: run.ID,
		Role: model.RoleAssistant, Content: "这条消息不能落库", CreatedAt: time.Now().UTC(),
	}
	changed, err = repo.FinishTerminal(ctx, run.ID, repository.TerminalTransition{Status: model.RunStatusSucceeded}, final)
	require.Error(t, err)
	assert.False(t, changed)

	got, err := repo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.RunStatusRunning, got.Status)
	assert.Nil(t, got.FinishedAt)

	messages, err := repo.ListMessages(ctx, run.ID)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
}
