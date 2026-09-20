package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestFileServiceMissingFileReturnsNotFound(t *testing.T) {
	svc := NewFileService(NewMockFileRepository(), nil)
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
	svc := NewFileService(NewMockFileRepository(), nil)

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
	repo := NewMockFileRepository()
	storage := &MockFileStorage{}
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

func TestFileServiceUploadSameNameAndSizeStillChecksContent(t *testing.T) {
	repo := NewMockFileRepository()
	repo.existingByName = &model.File{ID: "old", Size: 5}
	storage := &MockFileStorage{}
	svc := NewFileService(repo, storage)

	file, err := svc.UploadFile(context.Background(), "session-1", "doc.txt", strings.NewReader("hello"), 5, "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if file.ID == "old" {
		t.Fatal("same name and size must not reuse an existing file before hashing content")
	}
	if storage.uploadedKey == "" {
		t.Fatal("content must be uploaded so its hash can be verified")
	}
}

func TestFileServiceDownloadFileWithStorage(t *testing.T) {
	file := &model.File{ID: "file-1", SessionID: "session-1", Key: "files/session-1/file-1.txt"}
	repo := NewMockFileRepository()
	repo.file = file
	storage := &MockFileStorage{}
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

func TestFileServiceSessionScopedLookupRejectsOtherSession(t *testing.T) {
	file := &model.File{ID: "file-1", SessionID: "session-1", Key: "key"}
	repo := NewMockFileRepository()
	repo.file = file
	svc := NewFileService(repo, &MockFileStorage{})

	if _, err := svc.GetFileInfoForSession(context.Background(), "session-2", "file-1"); err == nil {
		t.Fatal("GetFileInfoForSession() error = nil, want not found")
	}
	if _, _, err := svc.DownloadFileForSession(context.Background(), "session-2", "file-1"); err == nil {
		t.Fatal("DownloadFileForSession() error = nil, want not found")
	}
}

func TestFileServiceSessionScopedLookupReturnsOwnedFile(t *testing.T) {
	file := &model.File{ID: "file-1", SessionID: "session-1", Filename: "doc.txt", Key: "key"}
	repo := NewMockFileRepository()
	repo.file = file
	svc := NewFileService(repo, &MockFileStorage{})

	got, err := svc.GetFileInfoForSession(context.Background(), "session-1", "file-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != file.ID {
		t.Fatalf("GetFileInfoForSession() = %+v, want %q", got, file.ID)
	}
}

func TestFileServiceDownloadFileNilStorageReturnsStorageUnavailable(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "k"}
	repo := NewMockFileRepository()
	repo.file = file
	svc := NewFileService(repo, nil)

	got, reader, err := svc.DownloadFile(context.Background(), "file-1")
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("DownloadFile() error = %v, want ErrStorageUnavailable", err)
	}
	if got != nil || reader != nil {
		t.Errorf("DownloadFile() = (%v, %v), want (nil, nil)", got, reader)
	}
}

func TestFileServiceGetFileInfoFound(t *testing.T) {
	file := &model.File{ID: "file-1", Filename: "doc.txt"}
	repo := NewMockFileRepository()
	repo.file = file
	svc := NewFileService(repo, nil)

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
	repo := NewMockFileRepository()
	repo.file = file
	storage := &MockFileStorage{}
	svc := NewFileService(repo, storage)

	if err := svc.DeleteFile(context.Background(), "file-1"); err != nil {
		t.Fatal(err)
	}
	if len(storage.deletedKeys) != 1 || storage.deletedKeys[0] != "files/session-1/file-1.txt" {
		t.Errorf("storage.Delete keys = %v, want [files/session-1/file-1.txt]", storage.deletedKeys)
	}
	if repo.deletedID != "file-1" {
		t.Errorf("repo.Delete id = %q, want file-1", repo.deletedID)
	}
}

// TestFileServiceDeleteFileStorageError 存储删除失败不再使整个操作失败：
// DB 记录先删（避免反向僵尸），残留对象仅告警、由 bucket 生命周期策略兜底。
func TestFileServiceDeleteFileStorageError(t *testing.T) {
	file := &model.File{ID: "file-1", Key: "k"}
	storage := &MockFileStorage{deleteErr: errors.New("s3 delete failed")}
	repo := NewMockFileRepository()
	repo.file = file
	svc := NewFileService(repo, storage)

	if err := svc.DeleteFile(context.Background(), "file-1"); err != nil {
		t.Fatalf("DeleteFile() error = %v, want nil (storage failure is non-fatal)", err)
	}
	if repo.deletedID != "file-1" {
		t.Fatalf("repo.Delete not called, deletedID = %q", repo.deletedID)
	}
}
