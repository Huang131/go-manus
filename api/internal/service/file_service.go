package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/google/uuid"
)

// ErrStorageUnavailable 表示存储服务不可用
var ErrStorageUnavailable = errors.New("storage unavailable")

// FileService 文件服务接口
type FileService interface {
	UploadFile(ctx context.Context, sessionID, filename string, reader io.Reader, size int64, contentType string) (*model.File, error)
	DownloadFile(ctx context.Context, id string) (*model.File, io.ReadCloser, error)
	DownloadFileForSession(ctx context.Context, sessionID, id string) (*model.File, io.ReadCloser, error)
	GetFileInfo(ctx context.Context, id string) (*model.File, error)
	GetFileInfoForSession(ctx context.Context, sessionID, id string) (*model.File, error)
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
	// cleanupService 文件清理服务，与定时调度器共享同一实例以保持统计一致。
	cleanupService FileCleanupService
}

// NewFileService 创建文件服务
func NewFileService(repo repository.FileRepository, storage COSFileStorage) FileService {
	return &DefaultFileService{
		repo:    repo,
		storage: storage,
		// 默认创建独立清理实例；bootstrap 会注入与调度器共享的实例覆盖它。
		cleanupService: NewFileCleanupService(repo, storage),
	}
}

// SetCleanupService 注入清理服务实例，用于与定时调度器共享，保证统计一致。
func (s *DefaultFileService) SetCleanupService(cs FileCleanupService) {
	s.cleanupService = cs
}

// UploadFile 上传文件
func (s *DefaultFileService) UploadFile(ctx context.Context, sessionID, filename string, reader io.Reader, size int64, contentType string) (*model.File, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("%w: cannot upload file without storage", ErrStorageUnavailable)
	}

	fileID := uuid.New().String()
	ext := filepath.Ext(filename)
	key := "files/" + sessionID + "/" + fileID + ext

	// 流式计算内容哈希：TeeReader 在上传的同时喂给 sha256，无需二次读取
	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if err := s.storage.Upload(ctx, key, tee, size, contentType); err != nil {
		return nil, err
	}
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	file := &model.File{
		ID:        fileID,
		SessionID: sessionID,
		Filename:  filename,
		Filepath:  key,
		Key:       key,
		Extension: ext,
		MimeType:  contentType,
		Size:      size,
		Sha256:    contentHash,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, file); err != nil {
		// (session_id, sha256) 唯一索引命中 = 并发上传同内容：复用既有记录
		if existing, hashErr := s.repo.GetBySessionAndHash(ctx, sessionID, contentHash); hashErr == nil && existing != nil {
			_ = s.storage.Delete(ctx, key)
			return existing, nil
		}
		if cleanupErr := s.storage.Delete(ctx, key); cleanupErr != nil {
			logger.Warn("清理文件上传孤儿对象失败",
				logger.String("key", key),
				logger.Err(cleanupErr))
		}
		return nil, err
	}

	return file, nil
}

// DownloadFile 下载文件
func (s *DefaultFileService) DownloadFile(ctx context.Context, id string) (*model.File, io.ReadCloser, error) {
	return s.downloadFile(ctx, s.repo.GetByID, id)
}

// DownloadFileForSession 只允许下载指定会话拥有的文件。
func (s *DefaultFileService) DownloadFileForSession(ctx context.Context, sessionID, id string) (*model.File, io.ReadCloser, error) {
	return s.downloadFile(ctx, func(ctx context.Context, id string) (*model.File, error) {
		return s.repo.GetBySessionAndID(ctx, sessionID, id)
	}, id)
}

func (s *DefaultFileService) downloadFile(ctx context.Context, lookup func(context.Context, string) (*model.File, error), id string) (*model.File, io.ReadCloser, error) {
	file, err := lookup(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, apperr.NotFound("文件不存在")
	}

	if s.storage == nil {
		return nil, nil, fmt.Errorf("%w: cannot download file without storage", ErrStorageUnavailable)
	}
	reader, err := s.storage.Download(ctx, file.Key)
	return file, reader, err
}

// GetFileInfo 获取文件信息
func (s *DefaultFileService) GetFileInfo(ctx context.Context, id string) (*model.File, error) {
	return s.getFileInfo(ctx, s.repo.GetByID, id)
}

// GetFileInfoForSession 只返回指定会话拥有的文件元数据。
func (s *DefaultFileService) GetFileInfoForSession(ctx context.Context, sessionID, id string) (*model.File, error) {
	return s.getFileInfo(ctx, func(ctx context.Context, id string) (*model.File, error) {
		return s.repo.GetBySessionAndID(ctx, sessionID, id)
	}, id)
}

func (s *DefaultFileService) getFileInfo(ctx context.Context, lookup func(context.Context, string) (*model.File, error), id string) (*model.File, error) {
	file, err := lookup(ctx, id)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, apperr.NotFound("文件不存在")
	}
	return file, nil
}

// DeleteFile 删除文件
func (s *DefaultFileService) DeleteFile(ctx context.Context, id string) error {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if file == nil {
		return apperr.NotFound("文件不存在")
	}

	// 先删 DB 记录再删对象：DB 失败时对象还在（仅存储成本泄漏，可由
	// bucket 生命周期策略兜底）；反过来先删对象会让记录指向已删对象，
	// 之后每次下载必然 404/500。
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.storage != nil {
		if err := s.storage.Delete(ctx, file.Key); err != nil {
			// 记录已删、对象残留：孤儿对象不影响用户，记录告警便于对账
			logger.Warn("删除存储对象失败（记录已删，孤儿对象待对账）",
				logger.String("file_id", id),
				logger.String("key", file.Key),
				logger.Err(err))
		}
	}
	return nil
}
