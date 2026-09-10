package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

// SchedulerRunner 表示可启动和停止的后台调度器。
type SchedulerRunner interface {
	Start()
	Stop()
}

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
	repo      repository.FileRepository
	storage   COSFileStorage
	stats     CleanupStats
	statsMu   sync.RWMutex
	cleanupMu sync.Mutex
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
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()

	startTime := time.Now()
	totalCleaned := 0
	totalSize := int64(0)

	// 循环批量清理，直到没有过期文件
	for {
		// 获取一批过期文件
		files, err := s.repo.GetExpiredFiles(ctx, expireDuration, int64(batchSize))
		if err != nil {
			s.updateCleanupStats(totalCleaned, totalSize, startTime)
			return totalCleaned, fmt.Errorf("get expired files: %w", err)
		}

		if len(files) == 0 {
			// 没有更多过期文件
			break
		}

		// 清理这批文件
		cleaned, size, err := s.cleanFiles(ctx, files)
		if err != nil {
			s.updateCleanupStats(totalCleaned, totalSize, startTime)
			return totalCleaned, fmt.Errorf("clean expired file batch: %w", err)
		}

		totalCleaned += cleaned
		totalSize += size

		logger.Info("清理文件批次完成",
			logger.Int("batch_count", cleaned),
			logger.Int64("batch_size", size))
		if cleaned == 0 {
			// 对象存储删除全部失败时保留数据库记录，结束本轮并等待下次重试。
			break
		}
	}

	s.updateCleanupStats(totalCleaned, totalSize, startTime)

	logger.Info("清理过期文件完成",
		logger.Int("total_cleaned", totalCleaned),
		logger.Int64("total_size", totalSize),
		logger.Dur("duration", time.Since(startTime)))

	return totalCleaned, nil
}

// cleanFiles 清理文件列表
func (s *DefaultFileCleanupService) cleanFiles(ctx context.Context, files []*model.File) (int, int64, error) {
	if len(files) == 0 {
		return 0, 0, nil
	}

	// 只有对象存储删除成功的文件才允许删除数据库记录。
	fileIDs := make([]string, 0, len(files))
	var totalSize int64

	for _, file := range files {
		if s.storage != nil {
			if err := s.storage.Delete(ctx, file.Key); err != nil {
				logger.Warn("从对象存储删除文件失败",
					logger.String("file_id", file.ID),
					logger.String("key", file.Key),
					logger.Err(err))
				continue
			}
		}
		fileIDs = append(fileIDs, file.ID)
		totalSize += file.Size
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
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	startTime := time.Now()

	// 获取会话的所有文件
	files, err := s.repo.ListBySessionID(ctx, sessionID)
	if err != nil {
		return 0, err
	}

	if len(files) == 0 {
		return 0, nil
	}

	// 清理文件
	cleaned, size, err := s.cleanFiles(ctx, files)
	if err != nil {
		return 0, err
	}

	s.updateCleanupStats(cleaned, size, startTime)

	logger.Info("清理会话文件完成",
		logger.String("session_id", sessionID),
		logger.Int("cleaned_count", cleaned))

	return cleaned, nil
}

// GetExpiredFileCount 获取过期文件数量
func (s *DefaultFileCleanupService) GetExpiredFileCount(ctx context.Context, expireDuration string) (int64, error) {
	return s.repo.CountExpiredFiles(ctx, expireDuration)
}

// GetCleanupStats 获取清理统计信息
func (s *DefaultFileCleanupService) GetCleanupStats() *CleanupStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	stats := s.stats
	return &stats
}

// updateCleanupStats 只在内存更新期间持锁，避免数据库和对象存储 I/O 阻塞统计读取。
func (s *DefaultFileCleanupService) updateCleanupStats(cleaned int, size int64, startedAt time.Time) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.stats.TotalCleanedFiles += int64(cleaned)
	s.stats.TotalCleanedSize += size
	s.stats.LastCleanupTime = time.Now()
	s.stats.LastCleanupCount = cleaned
	s.stats.LastCleanupDuration = time.Since(startedAt).String()
}

// FileCleanupScheduler 文件清理调度器
type FileCleanupScheduler struct {
	cleanupService FileCleanupService
	expireDuration string
	batchSize      int
	interval       time.Duration
	stopChan       chan struct{}
	stopOnce       sync.Once
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
		logger.String("expire_duration", s.expireDuration),
		logger.Int("batch_size", s.batchSize),
		logger.Dur("interval", s.interval))
}

// Stop 停止清理调度器
func (s *FileCleanupScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
	s.wg.Wait()
	logger.Info("文件清理调度器已停止")
}

// run 运行清理循环
func (s *FileCleanupScheduler) run() {
	defer s.wg.Done()

	// 首次清理延迟（等待系统启动完成）
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
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()

	cleaned, err := s.cleanupService.CleanExpiredFiles(ctx, s.expireDuration, s.batchSize)
	if err != nil {
		logger.Error("定时清理失败", logger.Err(err))
		return
	}

	if cleaned > 0 {
		logger.Info("定时清理完成", logger.Int("cleaned_count", cleaned))
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
