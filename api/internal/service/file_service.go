package service

import (
	"context"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
)

// FileService 文件服务接口
type FileService interface {
	UploadFile(ctx context.Context, sessionID, filename string, reader io.Reader, size int64, contentType string) (*model.File, error)
	DownloadFile(ctx context.Context, id string) (*model.File, io.ReadCloser, error)
	GetFileInfo(ctx context.Context, id string) (*model.File, error)
	DeleteFile(ctx context.Context, id string) error
}

// COSFileStorage 接口（用于 COS 集成）
type COSFileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string) (string, error)
}

// DefaultFileService 文件服务默认实现
type DefaultFileService struct {
	repo    repository.FileRepository
	storage COSFileStorage
}

// NewFileService 创建文件服务
func NewFileService(repo repository.FileRepository, storage COSFileStorage) FileService {
	return &DefaultFileService{
		repo:    repo,
		storage: storage,
	}
}

// UploadFile 上传文件
func (s *DefaultFileService) UploadFile(ctx context.Context, sessionID, filename string, reader io.Reader, size int64, contentType string) (*model.File, error) {
	fileID := uuid.New().String()
	ext := filepath.Ext(filename)
	key := "files/" + sessionID + "/" + fileID + ext

	if s.storage != nil {
		if err := s.storage.Upload(ctx, key, reader, size, contentType); err != nil {
			return nil, err
		}
	}

	file := &model.File{
		ID:        fileID,
		SessionID: sessionID,
		Filename:  filename,
		Filepath:  key,
		Key:       key,
		Extension: ext,
		MimeType:  contentType,
		Size:      size,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, file); err != nil {
		return nil, err
	}
	return file, nil
}

// DownloadFile 下载文件
func (s *DefaultFileService) DownloadFile(ctx context.Context, id string) (*model.File, io.ReadCloser, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if s.storage != nil {
		reader, err := s.storage.Download(ctx, file.Key)
		return file, reader, err
	}
	return file, nil, nil
}

// GetFileInfo 获取文件信息
func (s *DefaultFileService) GetFileInfo(ctx context.Context, id string) (*model.File, error) {
	return s.repo.GetByID(ctx, id)
}

// DeleteFile 删除文件
func (s *DefaultFileService) DeleteFile(ctx context.Context, id string) error {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if s.storage != nil {
		if err := s.storage.Delete(ctx, file.Key); err != nil {
			return err
		}
	}
	return s.repo.Delete(ctx, id)
}
