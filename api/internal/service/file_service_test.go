package service

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

type emptyFileRepo struct{}

func (emptyFileRepo) Create(context.Context, *model.File) error            { return nil }
func (emptyFileRepo) GetByID(context.Context, string) (*model.File, error) { return nil, nil }
func (emptyFileRepo) GetBySessionAndFilepath(context.Context, string, string) (*model.File, error) {
	return nil, nil
}
func (emptyFileRepo) Update(context.Context, *model.File) error                      { return nil }
func (emptyFileRepo) Delete(context.Context, string) error                           { return nil }
func (emptyFileRepo) DeleteBySessionID(context.Context, string) error                { return nil }
func (emptyFileRepo) ListBySessionID(context.Context, string) ([]*model.File, error) { return nil, nil }
func (emptyFileRepo) GetExpiredFiles(context.Context, string, int64) ([]*model.File, error) {
	return nil, nil
}
func (emptyFileRepo) CountExpiredFiles(context.Context, string) (int64, error) { return 0, nil }
func (emptyFileRepo) DeleteByIDs(context.Context, []string) (int64, error)     { return 0, nil }
func (emptyFileRepo) GetFilesBySessionIDs(context.Context, []string) ([]*model.File, error) {
	return nil, nil
}
func (r emptyFileRepo) WithTx(ctx context.Context, fn func(repository.FileRepository) error) error {
	return fn(r)
}

func TestFileServiceMissingFileReturnsNotFound(t *testing.T) {
	svc := NewFileService(emptyFileRepo{}, nil)
	ctx := context.Background()

	if _, err := svc.GetFileInfo(ctx, "missing"); err == nil {
		t.Fatal("GetFileInfo() error = nil, want not found")
	}
	if _, _, err := svc.DownloadFile(ctx, "missing"); err == nil {
		t.Fatal("DownloadFile() error = nil, want not found")
	}
	if err := svc.DeleteFile(ctx, "missing"); err == nil {
		t.Fatal("DeleteFile() error = nil, want not found")
	}
}
