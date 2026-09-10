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
	files        []*model.File
	entered      chan struct{}
	release      chan struct{}
	getCalls     int
	deletedIDs   []string
	sessionFiles []*model.File
}

func (r *cleanupRepoStub) GetExpiredFiles(ctx context.Context, _ string, _ int64) ([]*model.File, error) {
	r.getCalls++
	if r.getCalls > 1 {
		return nil, nil
	}
	if r.entered != nil {
		close(r.entered)
	}
	if r.release != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-r.release:
		}
	}
	return r.files, nil
}

func (r *cleanupRepoStub) DeleteByIDs(_ context.Context, ids []string) (int64, error) {
	r.deletedIDs = append([]string(nil), ids...)
	return int64(len(ids)), nil
}

func (r *cleanupRepoStub) ListBySessionID(context.Context, string) ([]*model.File, error) {
	return r.sessionFiles, nil
}

func (r *cleanupRepoStub) Create(context.Context, *model.File) error            { return nil }
func (r *cleanupRepoStub) GetByID(context.Context, string) (*model.File, error) { return nil, nil }
func (r *cleanupRepoStub) GetBySessionAndFilepath(context.Context, string, string) (*model.File, error) {
	return nil, nil
}
func (r *cleanupRepoStub) Update(context.Context, *model.File) error                { return nil }
func (r *cleanupRepoStub) Delete(context.Context, string) error                     { return nil }
func (r *cleanupRepoStub) DeleteBySessionID(context.Context, string) error          { return nil }
func (r *cleanupRepoStub) CountExpiredFiles(context.Context, string) (int64, error) { return 0, nil }
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

func TestFileCleanupStatsReadableDuringCleanup(t *testing.T) {
	repo := &cleanupRepoStub{
		files:   []*model.File{{ID: "file-1", Key: "key-1", Size: 10}},
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	svc := NewFileCleanupService(repo, &cleanupStorageStub{})
	done := make(chan error, 1)
	go func() {
		_, err := svc.CleanExpiredFiles(context.Background(), "24h", 10)
		done <- err
	}()

	<-repo.entered
	statsRead := make(chan struct{})
	go func() {
		_ = svc.GetCleanupStats()
		close(statsRead)
	}()

	select {
	case <-statsRead:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("GetCleanupStats() blocked while cleanup was waiting on repository I/O")
	}

	close(repo.release)
	if err := <-done; err != nil {
		t.Fatalf("CleanExpiredFiles() error = %v", err)
	}
}

func TestFileCleanupStatsAccumulateAcrossCleanupTypes(t *testing.T) {
	repo := &cleanupRepoStub{
		files:        []*model.File{{ID: "expired", Key: "expired-key", Size: 10}},
		sessionFiles: []*model.File{{ID: "session", Key: "session-key", Size: 20}},
	}
	svc := NewFileCleanupService(repo, &cleanupStorageStub{})

	if cleaned, err := svc.CleanExpiredFiles(context.Background(), "24h", 10); err != nil || cleaned != 1 {
		t.Fatalf("CleanExpiredFiles() = (%d, %v), want (1, nil)", cleaned, err)
	}
	if cleaned, err := svc.CleanSessionFiles(context.Background(), "session-1"); err != nil || cleaned != 1 {
		t.Fatalf("CleanSessionFiles() = (%d, %v), want (1, nil)", cleaned, err)
	}

	stats := svc.GetCleanupStats()
	if stats.TotalCleanedFiles != 2 || stats.TotalCleanedSize != 30 {
		t.Fatalf("GetCleanupStats() = %+v, want totals files=2 size=30", stats)
	}
}

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
