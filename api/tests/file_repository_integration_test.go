//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFileRepo(t *testing.T) repository.FileRepository {
	return repository.NewFileRepository(testDB)
}

// deleteSessionDirect 直接将 sessions.deleted_at 置为当前时间（软删除），
// 用于构造"会话已删除、文件成为孤立文件"的清理场景。
func deleteSessionDirect(t *testing.T, sessionID string) error {
	t.Helper()
	ctx, cancel := NewTestContext()
	defer cancel()
	_, err := testDB.Pool.Exec(ctx,
		"UPDATE sessions SET deleted_at = NOW() WHERE id = $1", sessionID)
	return err
}

func TestFileRepo_CreateAndGetByID(t *testing.T) {
	repo := testFileRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	f := &model.File{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Filename:  "test.txt",
		Filepath:  "/tmp/test.txt",
		Key:       "test-key-" + uuid.New().String(),
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      1024,
		CreatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), f)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(context.Background(), f.ID) })

	got, err := repo.GetByID(context.Background(), f.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, f.Filename, got.Filename)
	assert.Equal(t, sessionID, got.SessionID)
}

func TestFileRepo_GetByID_NotFoundReturnsNil(t *testing.T) {
	repo := testFileRepo(t)
	got, err := repo.GetByID(context.Background(), "non-existent-file")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFileRepo_GetExpiredFiles_NoExpiredFiles(t *testing.T) {
	repo := testFileRepo(t)

	files, err := repo.GetExpiredFiles(context.Background(), "24h", 100)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestFileRepo_GetExpiredFiles_WithExpiredFile(t *testing.T) {
	repo := testFileRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	// 创建时间戳为 48h 前的文件（模拟已过期）
	oldTime := time.Now().Add(-48 * time.Hour)
	f := &model.File{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Filename:  "old-file.txt",
		Filepath:  "/tmp/old.txt",
		Key:       "old-key-" + uuid.New().String(),
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      512,
		CreatedAt: oldTime,
	}
	err := repo.Create(context.Background(), f)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(context.Background(), f.ID) })

	// 业务语义：文件与会话同寿命，只有关联会话已软删除的过期文件才可清理
	err = deleteSessionDirect(t, sessionID)
	require.NoError(t, err)

	// 用 24h 作为过期阈值，应能查到 48h 前创建、会话已删除的文件
	files, err := repo.GetExpiredFiles(context.Background(), "24h", 100)
	require.NoError(t, err)
	found := false
	for _, file := range files {
		if file.ID == f.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "expired file with soft-deleted session should be returned with 24h threshold")
}

func TestFileRepo_GetExpiredFiles_ActiveSession_NotExpired(t *testing.T) {
	repo := testFileRepo(t)
	sessionID := createSessionForTest(t)

	// 关联活跃会话的旧文件不应被清理
	oldTime := time.Now().Add(-48 * time.Hour)
	f := &model.File{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Filename:  "orphan-but-active-session.txt",
		Filepath:  "/tmp/orphan.txt",
		Key:       "orphan-key-" + uuid.New().String(),
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      256,
		CreatedAt: oldTime,
	}
	err := repo.Create(context.Background(), f)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(context.Background(), f.ID) })

	files, err := repo.GetExpiredFiles(context.Background(), "24h", 100)
	require.NoError(t, err)
	for _, file := range files {
		if file.ID == f.ID {
			t.Fatal("file with active session should not be returned as expired")
		}
	}
}

func TestFileRepo_CountExpiredFiles(t *testing.T) {
	repo := testFileRepo(t)

	count, err := repo.CountExpiredFiles(context.Background(), "24h")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}

func TestFileRepo_Delete_PhysicalDelete(t *testing.T) {
	repo := testFileRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	f := &model.File{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Filename:  "to-delete.txt",
		Filepath:  "/tmp/delete.txt",
		Key:       "delete-key-" + uuid.New().String(),
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      100,
		CreatedAt: time.Now(),
	}
	err := repo.Create(context.Background(), f)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), f.ID)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), f.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFileRepo_GetBySessionAndFilepath(t *testing.T) {
	repo := testFileRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	key := "filepath-key-" + uuid.New().String()
	f := &model.File{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Filename:  "path-test.txt",
		Filepath:  "/tmp/path-test.txt",
		Key:       key,
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      200,
		CreatedAt: time.Now(),
	}
	err := repo.Create(context.Background(), f)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Delete(context.Background(), f.ID) })

	got, err := repo.GetBySessionAndFilepath(context.Background(), sessionID, "/tmp/path-test.txt")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, f.ID, got.ID)
}

func TestFileRepo_GetBySessionAndFilepath_NotFound(t *testing.T) {
	repo := testFileRepo(t)
	got, err := repo.GetBySessionAndFilepath(context.Background(), "non-existent-session", "/tmp/no-such-file.txt")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFileRepo_ListBySessionID(t *testing.T) {
	repo := testFileRepo(t)
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		f := &model.File{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Filename:  "file.txt",
			Filepath:  "/tmp/file.txt",
			Key:       "list-key-" + uuid.New().String(),
			Extension: ".txt",
			MimeType:  "text/plain",
			Size:      int64(i + 1),
			CreatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(context.Background(), f))
		ids[i] = f.ID
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_ = repo.Delete(context.Background(), id)
		}
	})

	files, err := repo.ListBySessionID(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Len(t, files, 3)
}
