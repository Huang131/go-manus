//go:build integration

// Package testsupport 提供仅供集成测试使用的环境配置。
package testsupport

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Huang131/go-manus/api/config"
)

const (
	envPostgresDSN = "API_TEST_POSTGRES_DSN"
	envRedisAddr   = "API_TEST_REDIS_ADDR"
	envRedisDB     = "API_TEST_REDIS_DB"
	envS3Endpoint  = "API_TEST_S3_ENDPOINT"
	envS3Bucket    = "API_TEST_S3_BUCKET"
)

// IntegrationEnv 集中保存集成测试配置和测试专用命名空间。
type IntegrationEnv struct {
	Config *config.Config
}

// LoadIntegrationEnv 加载基础 YAML，再用 API_TEST_* 环境变量覆盖连接信息。
func LoadIntegrationEnv(configPath string) (*IntegrationEnv, error) {
	cfg, err := config.LoadWithValidation(configPath)
	if err != nil {
		return nil, err
	}

	if err := applyPostgresEnv(&cfg.Database); err != nil {
		return nil, err
	}
	if err := applyRedisEnv(&cfg.Redis); err != nil {
		return nil, err
	}
	applyStorageEnv(&cfg.OSS)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate integration config: %w", err)
	}

	env := &IntegrationEnv{Config: cfg}
	if err := env.validateSafety(); err != nil {
		return nil, err
	}
	return env, nil
}

func applyPostgresEnv(cfg *config.DatabaseConfig) error {
	dsn, ok := os.LookupEnv(envPostgresDSN)
	if !ok || strings.TrimSpace(dsn) == "" {
		return nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("parse %s: %w", envPostgresDSN, err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("%s must use postgres scheme", envPostgresDSN)
	}
	if u.Hostname() == "" || u.Path == "" || u.Path == "/" {
		return fmt.Errorf("%s must include host and database", envPostgresDSN)
	}

	port := 5432
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return fmt.Errorf("parse %s port: %w", envPostgresDSN, err)
		}
	}

	cfg.Host = u.Hostname()
	cfg.Port = port
	cfg.Database = strings.TrimPrefix(u.Path, "/")
	if u.User != nil {
		cfg.User = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}
	return nil
}

func applyRedisEnv(cfg *config.RedisConfig) error {
	if addr, ok := os.LookupEnv(envRedisAddr); ok && strings.TrimSpace(addr) != "" {
		host, portText, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envRedisAddr, err)
		}
		port, err := strconv.Atoi(portText)
		if err != nil {
			return fmt.Errorf("parse %s port: %w", envRedisAddr, err)
		}
		if port <= 0 || port > 65535 {
			return fmt.Errorf("%s port must be between 1 and 65535", envRedisAddr)
		}
		cfg.Host = host
		cfg.Port = port
	}

	if dbText, ok := os.LookupEnv(envRedisDB); ok && strings.TrimSpace(dbText) != "" {
		db, err := strconv.Atoi(dbText)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envRedisDB, err)
		}
		cfg.DB = db
	}
	return nil
}

func applyStorageEnv(cfg *config.ObjectStorageConfig) {
	if endpoint, ok := os.LookupEnv(envS3Endpoint); ok {
		cfg.Endpoint = endpoint
	}
	if bucket, ok := os.LookupEnv(envS3Bucket); ok {
		cfg.Bucket = bucket
	}
}

func (e *IntegrationEnv) validateSafety() error {
	if e.Config.Env != config.EnvTest {
		return fmt.Errorf("integration tests require env=test")
	}
	if !isTestResource(e.Config.Database.Database) {
		return fmt.Errorf("refuse non-test PostgreSQL database %q", e.Config.Database.Database)
	}
	if e.Config.Redis.DB <= 0 {
		return fmt.Errorf("refuse Redis DB %d for integration tests", e.Config.Redis.DB)
	}
	if !isTestResource(e.Config.OSS.Bucket) {
		return fmt.Errorf("refuse non-test object storage bucket %q", e.Config.OSS.Bucket)
	}
	return nil
}

func isTestResource(name string) bool {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(name)), func(r rune) bool {
		return r < 'a' || r > 'z'
	})
	for _, part := range parts {
		if part == "test" {
			return true
		}
	}
	return false
}
