package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"go.uber.org/zap"
)

// FileCleanupService 文件清理服务接口
type FileCleanupService interface {
	// CleanExpiredFiles 清理过期文件
	// expireDuration: 文件过期时间，如 "24h", "7d"
	// batchSize: 每批清理的文件数量
	// 返回: 清理的文件数量
	CleanExpiredFiles(ctx context.Context, expireDuration string, batchSize int) (int, error)

	// CleanSessionFiles 清理指定会话的所有文件
	CleanSessionFiles(ctx context.Context, sessionID string) (int, error)

	// GetExpiredFileCount 获取过期文件数量
	GetExpiredFileCount(ctx context.Context, expireDuration string) (int64, error)

	// GetCleanupStats 获取清理统计信息
	GetCleanupStats() *CleanupStats
}

// CleanupStats 清理统计信息
type CleanupStats struct {
	TotalCleanedFiles   int64     `json:"total_cleaned_files"`   // 总清理文件数
	TotalCleanedSize    int64     `json:"total_cleaned_size"`    // 总清理大小（字节）
	LastCleanupTime     time.Time `json:"last_cleanup_time"`     // 上次清理时间
	LastCleanupCount    int       `json:"last_cleanup_count"`    // 上次清理文件数
	LastCleanupDuration string    `json:"last_cleanup_duration"` // 上次清理耗时
}

// DefaultFileCleanupService 默认文件清理服务实现
type DefaultFileCleanupService struct {
	repo    repository.FileRepository
	storage COSFileStorage
	stats   CleanupStats
	mu      sync.RWMutex
}

// NewFileCleanupService 创建文件清理服务
func NewFileCleanupService(repo repository.FileRepository, storage COSFileStorage) FileCleanupService {
	return &DefaultFileCleanupService{
		repo:    repo,
		storage: storage,
		stats:   CleanupStats{},
	}
}

// CleanExpiredFiles 清理过期文件
func (s *DefaultFileCleanupService) CleanExpiredFiles(ctx context.Context, expireDuration string, batchSize int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()
	totalCleaned := 0
	totalSize := int64(0)

	// 循环批量清理，直到没有过期文件
	for {
		// 获取一批过期文件
		files, err := s.repo.GetExpiredFiles(ctx, expireDuration, int64(batchSize))
		if err != nil {
			return totalCleaned, err
		}

		if len(files) == 0 {
			// 没有更多过期文件
			break
		}

		// 清理这批文件
		cleaned, size, err := s.cleanFiles(ctx, files)
		if err != nil {
			logger.Error("清理文件批次失败", zap.Error(err))
			break
		}

		totalCleaned += cleaned
		totalSize += size

		logger.Info("清理文件批次完成",
			zap.Int("batch_count", cleaned),
			zap.Int64("batch_size", size))
	}

	// 更新统计信息
	s.stats.TotalCleanedFiles += int64(totalCleaned)
	s.stats.TotalCleanedSize += totalSize
	s.stats.LastCleanupTime = time.Now()
	s.stats.LastCleanupCount = totalCleaned
	s.stats.LastCleanupDuration = time.Since(startTime).String()

	logger.Info("清理过期文件完成",
		zap.Int("total_cleaned", totalCleaned),
		zap.Int64("total_size", totalSize),
		zap.Duration("duration", time.Since(startTime)))

	return totalCleaned, nil
}

// cleanFiles 清理文件列表
func (s *DefaultFileCleanupService) cleanFiles(ctx context.Context, files []*model.File) (int, int64, error) {
	if len(files) == 0 {
		return 0, 0, nil
	}

	// 收集文件 ID 和总大小
	fileIDs := make([]string, len(files))
	var totalSize int64

	for i, file := range files {
		fileIDs[i] = file.ID
		totalSize += file.Size
	}

	// 从对象存储中删除文件
	if s.storage != nil {
		for _, file := range files {
			if err := s.storage.Delete(ctx, file.Key); err != nil {
				logger.Warn("从对象存储删除文件失败",
					zap.String("file_id", file.ID),
					zap.String("key", file.Key),
					zap.Error(err))
			}
		}
	}

	// 从数据库中删除文件记录
	deleted, err := s.repo.DeleteByIDs(ctx, fileIDs)
	if err != nil {
		return 0, 0, err
	}

	return int(deleted), totalSize, nil
}

// CleanSessionFiles 清理指定会话的所有文件
func (s *DefaultFileCleanupService) CleanSessionFiles(ctx context.Context, sessionID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取会话的所有文件
	files, err := s.repo.ListBySessionID(ctx, sessionID)
	if err != nil {
		return 0, err
	}

	if len(files) == 0 {
		return 0, nil
	}

	// 清理文件
	cleaned, _, err := s.cleanFiles(ctx, files)
	if err != nil {
		return 0, err
	}

	// 更新统计信息
	s.stats.LastCleanupTime = time.Now()
	s.stats.LastCleanupCount = cleaned

	logger.Info("清理会话文件完成",
		zap.String("session_id", sessionID),
		zap.Int("cleaned_count", cleaned))

	return cleaned, nil
}

// GetExpiredFileCount 获取过期文件数量
func (s *DefaultFileCleanupService) GetExpiredFileCount(ctx context.Context, expireDuration string) (int64, error) {
	return s.repo.CountExpiredFiles(ctx, expireDuration)
}

// GetCleanupStats 获取清理统计信息
func (s *DefaultFileCleanupService) GetCleanupStats() *CleanupStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stats
	return &stats
}

// FileCleanupScheduler 文件清理调度器
type FileCleanupScheduler struct {
	cleanupService FileCleanupService
	expireDuration string
	batchSize      int
	interval       time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewFileCleanupScheduler 创建文件清理调度器
func NewFileCleanupScheduler(
	cleanupService FileCleanupService,
	expireDuration string,
	batchSize int,
	interval time.Duration,
) *FileCleanupScheduler {
	return &FileCleanupScheduler{
		cleanupService: cleanupService,
		expireDuration: expireDuration,
		batchSize:      batchSize,
		interval:       interval,
		stopChan:       make(chan struct{}),
	}
}

// Start 启动清理调度器
func (s *FileCleanupScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	logger.Info("文件清理调度器已启动",
		zap.String("expire_duration", s.expireDuration),
		zap.Int("batch_size", s.batchSize),
		zap.Duration("interval", s.interval))
}

// Stop 停止清理调度器
func (s *FileCleanupScheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	logger.Info("文件清理调度器已停止")
}

// run 运行清理循环
func (s *FileCleanupScheduler) run() {
	defer s.wg.Done()

	// 首次清理延迟（等待系统启动完成）
	firstCleanupDelay := time.Minute * 5
	timer := time.NewTimer(firstCleanupDelay)
	defer timer.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-timer.C:
			s.cleanup()
			timer.Reset(s.interval)
		}
	}
}

// cleanup 执行清理
func (s *FileCleanupScheduler) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	cleaned, err := s.cleanupService.CleanExpiredFiles(ctx, s.expireDuration, s.batchSize)
	if err != nil {
		logger.Error("定时清理失败", zap.Error(err))
		return
	}

	if cleaned > 0 {
		logger.Info("定时清理完成", zap.Int("cleaned_count", cleaned))
	}
}

// DefaultFileService 添加清理相关方法
// CleanExpiredFiles 清理过期文件（便捷方法）
func (s *DefaultFileService) CleanExpiredFiles(ctx context.Context, expireDuration string, batchSize int) (int, error) {
	// 创建一个临时的清理服务
	cleanupService := NewFileCleanupService(s.repo, s.storage)
	return cleanupService.CleanExpiredFiles(ctx, expireDuration, batchSize)
}

// GetExpiredFileCount 获取过期文件数量（便捷方法）
func (s *DefaultFileService) GetExpiredFileCount(ctx context.Context, expireDuration string) (int64, error) {
	return s.repo.CountExpiredFiles(ctx, expireDuration)
}

// ValidateExpireDuration 验证过期时间格式
func ValidateExpireDuration(duration string) error {
	_, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("无效的过期时间格式: %s, 正确格式如: 24h, 7d, 30m", duration)
	}
	return nil
}
