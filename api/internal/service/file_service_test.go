package service

import (
	"context"
	"errors"
	"io"
	"strings"
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

// stubFileRepo 可配置的文件仓储 stub
type stubFileRepo struct {
	file       *model.File // GetByID 返回值
	getErr     error
	createErr  error
	deleteErr  error
	createdKey string // 记录 Create 收到的文件 Key
	deletedID  string // 记录 Delete 收到的 ID
}

func (s *stubFileRepo) Create(_ context.Context, f *model.File) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.createdKey = f.Key
	return nil
}
func (s *stubFileRepo) GetByID(_ context.Context, id string) (*model.File, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.file != nil && s.file.ID == id {
		return s.file, nil
	}
	return nil, nil
}
func (s *stubFileRepo) GetBySessionAndFilepath(context.Context, string, string) (*model.File, error) {
	return nil, nil
}
func (s *stubFileRepo) Update(context.Context, *model.File) error { return nil }
func (s *stubFileRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedID = id
	return nil
}
func (s *stubFileRepo) DeleteBySessionID(context.Context, string) error { return nil }
func (s *stubFileRepo) ListBySessionID(context.Context, string) ([]*model.File, error) {
	return nil, nil
}
func (s *stubFileRepo) GetExpiredFiles(context.Context, string, int64) ([]*model.File, error) {
	return nil, nil
}
func (s *stubFileRepo) CountExpiredFiles(context.Context, string) (int64, error) { return 0, nil }
func (s *stubFileRepo) DeleteByIDs(context.Context, []string) (int64, error)     { return 0, nil }
func (s *stubFileRepo) GetFilesBySessionIDs(context.Context, []string) ([]*model.File, error) {
	return nil, nil
}
func (s *stubFileRepo) WithTx(ctx context.Context, fn func(repository.FileRepository) error) error {
	return fn(s)
}

// stubStorage 可配置的对象存储 stub
type stubStorage struct {
	uploadErr   error
	downloadErr error
	deleteErr   error
	uploadedKey string
	downloaded  string
	deletedKey  string
}

func (s *stubStorage) Upload(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	s.uploadedKey = key
	return nil
}
func (s *stubStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	if s.downloadErr != nil {
		return nil, s.downloadErr
	}
	s.downloaded = key
	return io.NopCloser(strings.NewReader("file-content")), nil
}
func (s *stubStorage) Delete(_ context.Context, key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedKey = key
	return nil
}
func (s *stubStorage) GetURL(_ context.Context, key string) (string, error) {
	return "http://storage.local/" + key, nil
}

var _ COSFileStorage = (*stubStorage)(nil)
var _ repository.FileRepository = (*stubFileRepo)(nil)

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

func TestFileServiceUploadFileNoStorageReturnsError(t *testing.T) {
	svc := NewFileService(emptyFileRepo{}, nil)

	_, err := svc.UploadFile(context.Background(), "session-1", "test.txt",
		strings.NewReader("hello"), 5, "text/plain")
	if err == nil {
		t.Fatal("UploadFile() error = nil, want storage unavailable")
	}
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("UploadFile() error = %v, want ErrStorageUnavailable", err)
	}
}

func TestFileServiceUploadFileSuccess(t *testing.T) {
	repo := &stubFileRepo{}
	storage := &stubStorage{}
	svc := NewFileService(repo, storage)

	file, err := svc.UploadFile(context.Background(), "session-1", "doc.txt",
		strings.NewReader("hello"), 5, "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if file.ID == "" {
		t.Error("UploadFile() file.ID is empty")
	}
	if file.SessionID != "session-1" {
		t.Errorf("file.SessionID = %q, want session-1", file.SessionID)
	}
	if file.Filename != "doc.txt" {
		t.Errorf("file.Filename = %q, want doc.txt", file.Filename)
	}
	if file.Extension != ".txt" {
		t.Errorf("file.Extension = %q, want .txt", file.Extension)
	}
	if file.MimeType != "text/plain" {
		t.Errorf("file.MimeType = %q, want text/plain", file.MimeType)
	}
	if file.Size != 5 {
		t.Errorf("file.Size = %d, want 5", file.Size)
	}
	// 对象存储 key 格式：files/{sessionID}/{fileID}{ext}
	if storage.uploadedKey != file.Key {
		t.Errorf("storage.Upload key = %q, want %q", storage.uploadedKey, file.Key)
	}
	if repo.createdKey != file.Key {
		t.Errorf("repo.Create filepath = %q, want %q", repo.createdKey, file.Key)
	}
}

func TestFileServiceUploadFileStorageError(t *testing.T) {
	storage := &stubStorage{uploadErr: errors.New("s3 unavailable")}
	svc := NewFileService(&stubFileRepo{}, storage)

	_, err := svc.UploadFile(context.Background(), "session-1", "doc.txt",
		strings.NewReader("hello"), 5, "text/plain")
	if err == nil || err.Error() != "s3 unavailable" {
		t.Fatalf("UploadFile() error = %v, want storage error", err)
	}
}

func TestFileServiceUploadFileRepoCreateError(t *testing.T) {
	repo := &stubFileRepo{createErr: errors.New("db write failed")}
	storage := &stubStorage{}
	svc := NewFileService(repo, storage)

	_, err := svc.UploadFile(context.Background(), "session-1", "doc.txt",
		strings.NewReader("hello"), 5, "text/plain")
	if err == nil || err.Error() != "db write failed" {
		t.Fatalf("UploadFile() error = %v, want repository error", err)
	}
	if storage.deletedKey == "" {
		t.Fatal("UploadFile() should compensate by deleting uploaded object")
	}
}

func TestFileServiceDownloadFileWithStorage(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "files/session-1/file-1.txt"}
	repo := &stubFileRepo{file: file}
	storage := &stubStorage{}
	svc := NewFileService(repo, storage)

	got, reader, err := svc.DownloadFile(context.Background(), "file-1")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if got.ID != "file-1" {
		t.Errorf("DownloadFile() file.ID = %q, want file-1", got.ID)
	}
	if storage.downloaded != "files/session-1/file-1.txt" {
		t.Errorf("storage.Download key = %q, want files/session-1/file-1.txt", storage.downloaded)
	}
	content, _ := io.ReadAll(reader)
	if string(content) != "file-content" {
		t.Errorf("DownloadFile() content = %q, want file-content", string(content))
	}
}

func TestFileServiceDownloadFileStorageError(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "k"}
	storage := &stubStorage{downloadErr: errors.New("s3 read failed")}
	svc := NewFileService(&stubFileRepo{file: file}, storage)

	_, _, err := svc.DownloadFile(context.Background(), "file-1")
	if err == nil || err.Error() != "s3 read failed" {
		t.Fatalf("DownloadFile() error = %v, want storage error", err)
	}
}

func TestFileServiceDownloadFileNilStorageReturnsNilReader(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "k"}
	svc := NewFileService(&stubFileRepo{file: file}, nil)

	got, reader, err := svc.DownloadFile(context.Background(), "file-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || reader != nil {
		t.Errorf("DownloadFile() = (%v, %v), want (file, nil)", got, reader)
	}
}

func TestFileServiceGetFileInfoFound(t *testing.T) {
	file := &model.File{ID: "file-1", Filename: "doc.txt"}
	svc := NewFileService(&stubFileRepo{file: file}, nil)

	got, err := svc.GetFileInfo(context.Background(), "file-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Filename != "doc.txt" {
		t.Errorf("GetFileInfo() filename = %q, want doc.txt", got.Filename)
	}
}

func TestFileServiceDeleteFileDeletesStorageAndRepo(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "files/session-1/file-1.txt"}
	repo := &stubFileRepo{file: file}
	storage := &stubStorage{}
	svc := NewFileService(repo, storage)

	if err := svc.DeleteFile(context.Background(), "file-1"); err != nil {
		t.Fatal(err)
	}
	if storage.deletedKey != "files/session-1/file-1.txt" {
		t.Errorf("storage.Delete key = %q, want files/session-1/file-1.txt", storage.deletedKey)
	}
	if repo.deletedID != "file-1" {
		t.Errorf("repo.Delete id = %q, want file-1", repo.deletedID)
	}
}

// TestFileServiceDeleteFileStorageError 存储删除失败不再使整个操作失败：
// DB 记录先删（避免反向僵尸），残留对象仅告警、由 bucket 生命周期策略兜底。
func TestFileServiceDeleteFileStorageError(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "k"}
	storage := &stubStorage{deleteErr: errors.New("s3 delete failed")}
	repo := &stubFileRepo{file: file}
	svc := NewFileService(repo, storage)

	if err := svc.DeleteFile(context.Background(), "file-1"); err != nil {
		t.Fatalf("DeleteFile() error = %v, want nil (storage failure is non-fatal)", err)
	}
	if repo.deletedID != "file-1" {
		t.Fatalf("repo.Delete not called, deletedID = %q", repo.deletedID)
	}
}

func TestFileServiceDeleteFileRepoError(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "k"}
	repo := &stubFileRepo{file: file, deleteErr: errors.New("db delete failed")}
	svc := NewFileService(repo, &stubStorage{})

	err := svc.DeleteFile(context.Background(), "file-1")
	if err == nil || err.Error() != "db delete failed" {
		t.Fatalf("DeleteFile() error = %v, want repository error", err)
	}
}
