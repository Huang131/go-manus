package service

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestFileCleanupKeepsDatabaseRecordWhenStorageDeleteFails(t *testing.T) {
	repo := NewMockFileRepository()
	repo.expiredFiles = []*model.File{
		{ID: "failed", Key: "failed-key", Size: 10},
		{ID: "success", Key: "success-key", Size: 20},
	}
	storage := &MockFileStorage{deleteErr: context.DeadlineExceeded}
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

// TestFileCleanupStopsWhenWholeBatchStorageDeleteFails 验证整批删除失败时
// 清理循环立即终止（确定性断言，不依赖超时等待）。
// expiredFilesRepeat 使仓储持续返回同一批文件，若循环不终止则会无限重试。
func TestFileCleanupStopsWhenWholeBatchStorageDeleteFails(t *testing.T) {
	repo := NewMockFileRepository()
	repo.expiredFiles = []*model.File{{ID: "failed", Key: "failed-key", Size: 10}}
	repo.expiredFilesRepeat = true
	storage := &MockFileStorage{deleteErr: context.DeadlineExceeded}
	svc := NewFileCleanupService(repo, storage)

	cleaned, err := svc.CleanExpiredFiles(context.Background(), "24h", 10)
	if err != nil {
		t.Fatalf("CleanExpiredFiles() error = %v, want graceful stop", err)
	}
	if cleaned != 0 {
		t.Fatalf("CleanExpiredFiles() cleaned = %d, want 0", cleaned)
	}
	if repo.expiredCalls != 1 {
		t.Fatalf("GetExpiredFiles called %d times, want 1 (batch must not be retried)", repo.expiredCalls)
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
