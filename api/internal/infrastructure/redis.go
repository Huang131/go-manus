package infrastructure

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"go.uber.org/zap"
)

// Redis Redis 客户端
type Redis struct {
	Client *redis.Client
}

// NewRedis 创建 Redis 客户端
func NewRedis(cfg *config.RedisConfig) (*Redis, error) {
	logger.Info("Initializing Redis connection...")

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info("Redis connection established",
		zap.String("addr", cfg.Addr()),
		zap.Int("db", cfg.DB),
	)

	return &Redis{Client: client}, nil
}

// Close 关闭客户端
func (r *Redis) Close() error {
	if r.Client != nil {
		err := r.Client.Close()
		logger.Info("Redis connection closed")
		return err
	}
	return nil
}

// HealthCheck 健康检查
func (r *Redis) HealthCheck(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}
