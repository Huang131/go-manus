package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

type cleanupRepoStub struct {
	files      []*model.File
	getCalls   int
	deletedIDs []string
}

func (r *cleanupRepoStub) GetBySessionAndFilename(ctx context.Context, sessionID, filename string) (*model.File, error) {
	return nil, nil
}

func (r *cleanupRepoStub) GetBySessionAndHash(ctx context.Context, sessionID, sha256 string) (*model.File, error) {
	return nil, nil
}
func (r *cleanupRepoStub) GetExpiredFiles(ctx context.Context, _ string, _ int64) ([]*model.File, error) {
	r.getCalls++
	if r.getCalls > 1 {
		return nil, nil
	}
	return r.files, nil
}

func (r *cleanupRepoStub) DeleteByIDs(_ context.Context, ids []string) (int64, error) {
	r.deletedIDs = append([]string(nil), ids...)
	return int64(len(ids)), nil
}

func (r *cleanupRepoStub) ListBySessionID(context.Context, string) ([]*model.File, error) {
	return nil, nil
}

func (r *cleanupRepoStub) Create(context.Context, *model.File) error            { return nil }
func (r *cleanupRepoStub) GetByID(context.Context, string) (*model.File, error) { return nil, nil }
func (r *cleanupRepoStub) GetBySessionAndID(context.Context, string, string) (*model.File, error) {
	return nil, nil
}
func (r *cleanupRepoStub) GetBySessionAndFilepath(context.Context, string, string) (*model.File, error) {
	return nil, nil
}
func (r *cleanupRepoStub) Update(context.Context, *model.File) error       { return nil }
func (r *cleanupRepoStub) Delete(context.Context, string) error            { return nil }
func (r *cleanupRepoStub) DeleteBySessionID(context.Context, string) error { return nil }
func (r *cleanupRepoStub) GetFilesBySessionIDs(context.Context, []string) ([]*model.File, error) {
	return nil, nil
}
func (r *cleanupRepoStub) WithTx(ctx context.Context, fn func(repository.FileRepository) error) error {
	return fn(r)
}

type cleanupStorageStub struct {
	deleteErr  error
	deletedKey []string
}

func (cleanupStorageStub) Upload(context.Context, string, io.Reader, int64, string) error { return nil }
func (cleanupStorageStub) Download(context.Context, string) (io.ReadCloser, error)        { return nil, nil }
func (s *cleanupStorageStub) Delete(_ context.Context, key string) error {
	s.deletedKey = append(s.deletedKey, key)
	return s.deleteErr
}
func (cleanupStorageStub) GetURL(context.Context, string) (string, error) { return "", nil }

func TestFileCleanupKeepsDatabaseRecordWhenStorageDeleteFails(t *testing.T) {
	repo := &cleanupRepoStub{
		files: []*model.File{
			{ID: "failed", Key: "failed-key", Size: 10},
			{ID: "success", Key: "success-key", Size: 20},
		},
	}
	storage := &cleanupStorageStub{deleteErr: context.DeadlineExceeded}
	svc := NewFileCleanupService(repo, storage)

	cleaned, err := svc.CleanExpiredFiles(context.Background(), "24h", 10)
	if err != nil {
		t.Fatalf("CleanExpiredFiles() error = %v", err)
	}
	if cleaned != 0 {
		t.Fatalf("CleanExpiredFiles() cleaned = %d, want 0 when storage deletion fails", cleaned)
	}
	if len(repo.deletedIDs) != 0 {
		t.Fatalf("DeleteByIDs() called with %v, want no database deletion", repo.deletedIDs)
	}
}

func TestFileCleanupStopsWhenWholeBatchStorageDeleteFails(t *testing.T) {
	repo := &cleanupRepoStub{
		files: []*model.File{{ID: "failed", Key: "failed-key", Size: 10}},
	}
	storage := &cleanupStorageStub{deleteErr: context.DeadlineExceeded}
	svc := NewFileCleanupService(repo, storage)

	done := make(chan struct{})
	go func() {
		_, _ = svc.CleanExpiredFiles(context.Background(), "24h", 10)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("CleanExpiredFiles() kept retrying a failed batch indefinitely")
	}
}

func TestFileCleanupSchedulerStopCancelsCleanupContext(t *testing.T) {
	scheduler := NewFileCleanupScheduler(nil, "24h", 10, time.Hour)
	scheduler.Start()
	scheduler.Stop()

	select {
	case <-scheduler.ctx.Done():
	default:
		t.Fatal("scheduler context should be canceled after Stop")
	}
}

func TestFileCleanupSchedulerStartIsIdempotent(t *testing.T) {
	scheduler := NewFileCleanupScheduler(nil, "24h", 10, time.Hour)
	scheduler.Start()
	scheduler.Start()
	scheduler.Stop()
}
