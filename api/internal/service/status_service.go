package service

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
)

// StatusService 状态服务接口
type StatusService interface {
	GetHealthStatus(ctx context.Context) (*model.HealthStatus, error)
}

// DefaultStatusService 状态服务默认实现
type DefaultStatusService struct {
	db    *infrastructure.Postgres
	redis *infrastructure.Redis
	oss   *infrastructure.OSS
}

// NewStatusService 创建状态服务
func NewStatusService(db *infrastructure.Postgres, redis *infrastructure.Redis, oss *infrastructure.OSS) StatusService {
	return &DefaultStatusService{
		db:    db,
		redis: redis,
		oss:   oss,
	}
}

// GetHealthStatus 获取健康状态
func (s *DefaultStatusService) GetHealthStatus(ctx context.Context) (*model.HealthStatus, error) {
	status := &model.HealthStatus{
		Status:    "healthy",
		Timestamp: model.TimeFunc().Unix(),
		Services:  make(map[string]model.ServiceStatus),
	}

	// 检查 PostgreSQL
	postgresStatus := model.ServiceStatus{Name: "postgres", Status: "healthy"}
	if err := s.db.HealthCheck(ctx); err != nil {
		postgresStatus.Status = "unhealthy"
		postgresStatus.Error = err.Error()
		status.Status = "degraded"
	}
	status.Services["postgres"] = postgresStatus

	// 检查 Redis
	redisStatus := model.ServiceStatus{Name: "redis", Status: "healthy"}
	if err := s.redis.HealthCheck(ctx); err != nil {
		redisStatus.Status = "unhealthy"
		redisStatus.Error = err.Error()
		status.Status = "degraded"
	}
	status.Services["redis"] = redisStatus

	// 检查 OSS
	if s.oss != nil {
		ossStatus := model.ServiceStatus{Name: "oss", Status: "healthy"}
		if err := s.oss.HealthCheck(ctx); err != nil {
			ossStatus.Status = "unhealthy"
			ossStatus.Error = err.Error()
		}
		status.Services["oss"] = ossStatus
	}

	return status, nil
}
