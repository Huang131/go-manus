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
		now:   time.Now,
	}
}

// GetHealthStatus 获取健康状态
func (s *DefaultStatusService) GetHealthStatus(ctx context.Context) (*model.HealthStatus, error) {
	now := s.now
	if now == nil {
		now = time.Now
	}
	status := &model.HealthStatus{
		Status:    model.HealthStateHealthy,
		Timestamp: now().Unix(),
		Services:  make(map[model.ServiceName]model.ServiceStatus),
	}

	// 依赖为 nil 时不注入对应 healthCheck，checkDependency 据此标为 skipped。
	var dbCheck, redisCheck, ossCheck func(context.Context) error
	if s.db != nil {
		dbCheck = s.db.HealthCheck
	}
	if s.redis != nil {
		redisCheck = s.redis.HealthCheck
	}
	if s.oss != nil {
		ossCheck = s.oss.HealthCheck
	}

	s.checkDependency(ctx, status, model.ServiceNamePostgres, dbCheck)
	s.checkDependency(ctx, status, model.ServiceNameRedis, redisCheck)
	s.checkDependency(ctx, status, model.ServiceNameOSS, ossCheck)

	return status, nil
}

// checkDependency 检查单个依赖健康并写入 status.Services。
// healthCheck 为 nil 表示该依赖未注入 → 标记 skipped，整体降级为 degraded；
// 检查失败 → 标记 unhealthy，整体降级为 degraded。
func (s *DefaultStatusService) checkDependency(
	ctx context.Context,
	status *model.HealthStatus,
	name model.ServiceName,
	healthCheck func(context.Context) error,
) {
	dep := model.ServiceStatus{Name: name, Status: model.HealthStateHealthy}
	switch {
	case healthCheck == nil:
		dep.Status = model.HealthStateSkipped
		status.Status = model.HealthStateDegraded
	default:
		if err := healthCheck(ctx); err != nil {
			dep.Status = model.HealthStateUnhealthy
			dep.Error = apperr.ToInternal(err).Error()
			status.Status = model.HealthStateDegraded
		}
	}
	status.Services[name] = dep
}
