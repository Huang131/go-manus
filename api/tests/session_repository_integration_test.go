//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSessionRepo(t *testing.T) repository.SessionRepository {
	return repository.NewSessionRepository(testDB)
}

func TestSessionRepo_CreateAndGetByID(t *testing.T) {
	repo := testSessionRepo(t)
	session := &model.Session{
		ID:        uuid.New().String(),
		Title:     "integration-test-session",
		Status:    model.SessionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), session)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(context.Background(), session.ID) })

	got, err := repo.GetByID(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, session.ID, got.ID)
	assert.Equal(t, "integration-test-session", got.Title)
	assert.Equal(t, model.SessionStatusPending, got.Status)
}

func TestSessionRepo_GetByID_NotFoundReturnsNilNil(t *testing.T) {
	repo := testSessionRepo(t)
	got, err := repo.GetByID(context.Background(), "non-existent-id")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSessionRepo_AppendEvent_AtomicUpdate(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	event := &model.Event{
		ID:        uuid.New().String(),
		Type:      model.EventTypeMessage,
		CreatedAt: time.Now(),
		Data:      []byte(`{"content":"hello from integration test"}`),
	}

	err := repo.AppendEvent(context.Background(), sessionID, event)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	require.NotEmpty(t, got.Events)
	assert.Equal(t, model.EventTypeMessage, got.Events[0].Type)
}

func TestSessionRepo_AppendEvent_UpdatesLatestMessage(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	// AppendEvent 只从 MessageEvent 结构（message/role 字段）提取最新消息；
	// 且仅 assistant 回复计入未读数
	event := &model.Event{
		ID:        uuid.New().String(),
		Type:      model.EventTypeMessage,
		CreatedAt: time.Now(),
		Data:      []byte(`{"type":"message","role":"assistant","message":"latest message"}`),
	}

	err := repo.AppendEvent(context.Background(), sessionID, event)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Equal(t, "latest message", got.LatestMessage)
	assert.NotNil(t, got.LatestMessageAt)
	assert.Equal(t, 1, got.UnreadMessageCount)
}

func TestSessionRepo_AppendEvent_UserMessageNotUnread(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	// 用户自己发送的消息更新 latest_message 但不计入未读数
	event := &model.Event{
		ID:        uuid.New().String(),
		Type:      model.EventTypeMessage,
		CreatedAt: time.Now(),
		Data:      []byte(`{"type":"message","role":"user","message":"user question"}`),
	}

	err := repo.AppendEvent(context.Background(), sessionID, event)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Equal(t, "user question", got.LatestMessage)
	assert.Equal(t, 0, got.UnreadMessageCount)
}

func TestSessionRepo_GetMemory_NotFoundSessionReturnsEmptySlice(t *testing.T) {
	repo := testSessionRepo(t)

	messages, err := repo.GetMemory(context.Background(), "non-existent-session", "planner")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestSessionRepo_GetMemory_AgentNotInMemoryReturnsEmptySlice(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	messages, err := repo.GetMemory(context.Background(), sessionID, "planner")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestSessionRepo_SaveAndGetMemory_RoundTrip(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	messages := []llmcore.Message{
		{Role: llmcore.RoleUser, ContentText: "test message content"},
		{Role: llmcore.RoleAssistant, ContentText: "assistant response"},
	}

	err := repo.SaveMemory(context.Background(), sessionID, "planner", messages)
	require.NoError(t, err)

	got, err := repo.GetMemory(context.Background(), sessionID, "planner")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "test message content", got[0].ContentText)
	assert.Equal(t, llmcore.RoleUser, got[0].Role)
	assert.Equal(t, "assistant response", got[1].ContentText)
}

func TestSessionRepo_SoftDelete(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)

	got, err := repo.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	require.NotNil(t, got)

	err = repo.Delete(context.Background(), sessionID)
	require.NoError(t, err)

	got, err = repo.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSessionRepo_List_Pagination(t *testing.T) {
	repo := testSessionRepo(t)
	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		s := &model.Session{
			ID:        uuid.New().String(),
			Title:     "pagination-test",
			Status:    model.SessionStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(context.Background(), s))
		ids[i] = s.ID
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_ = repo.Delete(context.Background(), id)
		}
	})

	sessions, total, err := repo.List(context.Background(), 2, 0)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
	assert.GreaterOrEqual(t, total, 5)

	sessions2, _, err := repo.List(context.Background(), 2, 2)
	require.NoError(t, err)
	assert.Len(t, sessions2, 2)
}

func TestSessionRepo_GetAll_ExcludesSoftDeleted(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)

	all, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	found := false
	for _, s := range all {
		if s.ID == sessionID {
			found = true
			break
		}
	}
	assert.True(t, found)

	_ = repo.Delete(context.Background(), sessionID)

	all2, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	for _, s := range all2 {
		if s.ID == sessionID {
			t.Fatal("soft-deleted session should not appear in GetAll")
		}
	}
}

func TestSessionRepo_UpdateTitle(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	err := repo.UpdateTitle(context.Background(), sessionID, "renamed session")
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Equal(t, "renamed session", got.Title)
}

func TestSessionRepo_UpdateStatus(t *testing.T) {
	repo := testSessionRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	err := repo.UpdateStatus(context.Background(), sessionID, model.SessionStatusRunning)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionStatusRunning, got.Status)
}
