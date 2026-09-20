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

const (
	// firstCleanupDelay 首次清理的延迟时间，留出系统启动完成的时间窗口。
	firstCleanupDelay = 5 * time.Minute
	// cleanupTimeout 单次清理操作的超时时间。
	cleanupTimeout = 5 * time.Minute
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
}

// DefaultFileCleanupService 默认文件清理服务实现
type DefaultFileCleanupService struct {
	repo      repository.FileRepository
	storage   FileStorage
	cleanupMu sync.Mutex
}

// NewFileCleanupService 创建文件清理服务
func NewFileCleanupService(repo repository.FileRepository, storage FileStorage) FileCleanupService {
	return &DefaultFileCleanupService{
		repo:    repo,
		storage: storage,
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
			return totalCleaned, fmt.Errorf("get expired files: %w", err)
		}

		if len(files) == 0 {
			// 没有更多过期文件
			break
		}

		// 清理这批文件
		cleaned, size, err := s.cleanFiles(ctx, files)
		if err != nil {
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

// FileCleanupScheduler 文件清理调度器
type FileCleanupScheduler struct {
	cleanupService FileCleanupService
	expireDuration string
	batchSize      int
	interval       time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	stopChan       chan struct{}
	stopOnce       sync.Once
	startOnce      sync.Once
	wg             sync.WaitGroup
}

// NewFileCleanupScheduler 创建文件清理调度器
func NewFileCleanupScheduler(
	cleanupService FileCleanupService,
	expireDuration string,
	batchSize int,
	interval time.Duration,
) *FileCleanupScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileCleanupScheduler{
		cleanupService: cleanupService,
		expireDuration: expireDuration,
		batchSize:      batchSize,
		interval:       interval,
		ctx:            ctx,
		cancel:         cancel,
		stopChan:       make(chan struct{}),
	}
}

// Start 启动清理调度器
func (s *FileCleanupScheduler) Start() {
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.run()
	})
	logger.Info("文件清理调度器已启动",
		logger.String("expire_duration", s.expireDuration),
		logger.Int("batch_size", s.batchSize),
		logger.Dur("interval", s.interval))
}

// Stop 停止清理调度器
func (s *FileCleanupScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
		s.cancel()
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
	ctx, cancel := context.WithTimeout(s.ctx, cleanupTimeout)
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
