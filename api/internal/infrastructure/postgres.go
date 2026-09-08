package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

// Postgres PostgreSQL 客户端
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres 创建 PostgreSQL 连接池
func NewPostgres(cfg *config.DatabaseConfig) (*Postgres, error) {
	logger.Info("Initializing PostgreSQL connection...")

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 测试连接
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("PostgreSQL connection established",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.String("database", cfg.Database),
	)

	return &Postgres{Pool: pool}, nil
}

// Close 关闭连接池
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
		logger.Info("PostgreSQL connection closed")
	}
}

// HealthCheck 健康检查
func (p *Postgres) HealthCheck(ctx context.Context) error {
	if p == nil || p.Pool == nil {
		return fmt.Errorf("postgres not initialized")
	}
	return p.Pool.Ping(ctx)
}
