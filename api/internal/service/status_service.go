package service

import (
	"context"
	"time"

	"github.com/Huang131/go-manus/api/internal/apperr"
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
	now   func() time.Time
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
		Status:    model.HealthStateHealthy,
		Timestamp: time.Now().Unix(),
		Services:  make(map[model.ServiceName]model.ServiceStatus),
	}

	// 检查 PostgreSQL
	if s.db != nil {
		postgresStatus := model.ServiceStatus{
			Name:   model.ServiceNamePostgres,
			Status: model.HealthStateHealthy,
		}
		if err := s.db.HealthCheck(ctx); err != nil {
			postgresStatus.Status = model.HealthStateUnhealthy
			postgresStatus.Error = apperr.ToInternal(err).Error()
			status.Status = model.HealthStateDegraded
		}
		status.Services[model.ServiceNamePostgres] = postgresStatus
	} else {
		status.Services[model.ServiceNamePostgres] = model.ServiceStatus{
			Name:   model.ServiceNamePostgres,
			Status: model.HealthStateSkipped,
		}
		status.Status = model.HealthStateDegraded
	}

	// 检查 Redis
	if s.redis != nil {
		redisStatus := model.ServiceStatus{
			Name:   model.ServiceNameRedis,
			Status: model.HealthStateHealthy,
		}
		if err := s.redis.HealthCheck(ctx); err != nil {
			redisStatus.Status = model.HealthStateUnhealthy
			redisStatus.Error = apperr.ToInternal(err).Error()
			status.Status = model.HealthStateDegraded
		}
		status.Services[model.ServiceNameRedis] = redisStatus
	} else {
		status.Services[model.ServiceNameRedis] = model.ServiceStatus{
			Name:   model.ServiceNameRedis,
			Status: model.HealthStateSkipped,
		}
		status.Status = model.HealthStateDegraded
	}

	// 检查 OSS
	if s.oss != nil {
		ossStatus := model.ServiceStatus{
			Name:   model.ServiceNameOSS,
			Status: model.HealthStateHealthy,
		}
		if err := s.oss.HealthCheck(ctx); err != nil {
			ossStatus.Status = model.HealthStateUnhealthy
			ossStatus.Error = apperr.ToInternal(err).Error()
			status.Status = model.HealthStateDegraded
		}
		status.Services[model.ServiceNameOSS] = ossStatus
	} else {
		status.Services[model.ServiceNameOSS] = model.ServiceStatus{
			Name:   model.ServiceNameOSS,
			Status: model.HealthStateSkipped,
		}
		status.Status = model.HealthStateDegraded
	}

	return status, nil
}
