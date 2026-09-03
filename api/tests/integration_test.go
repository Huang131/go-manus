//go:build integration

// Package integration 包含真实环境的集成测试。
//
// 测试特点：
//   - 使用真实的 Postgres、Redis、MinIO（需先启动 ../../scripts/test-env-up.sh）
//   - 每个测试用例独立，使用唯一标识避免数据污染
//   - 测试完成后自动清理数据
//
// 运行方式：
//
//	# 先确保开发 Docker (docker-compose.yml) 已启动
//	# 初始化测试数据（在共享 Docker 上创建 manus_test DB 等）
//	cd ../../scripts && ./test-env-up.sh
//
//	# 运行测试
//	go test -tags=integration -v ./tests/...
//
//	# 清理测试数据（可选）
//	cd ../../scripts && ./test-env-down.sh
package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/internal/handler"
	"github.com/mooc-manus/go-manus/api/internal/infrastructure"
	"github.com/mooc-manus/go-manus/api/internal/repository"
	"github.com/mooc-manus/go-manus/api/internal/router"
	"github.com/mooc-manus/go-manus/api/internal/service"
)

var (
	testServer *gin.Engine
	testCfg    *config.Config
	testDB     *infrastructure.Postgres
	testRedis  *infrastructure.Redis
)

// TestMain 初始化测试环境
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// 加载测试配置
	cfg, err := config.Load("../config.test.yaml")
	if err != nil {
		log.Fatalf("加载测试配置失败: %v", err)
	}

	// 连接真实数据库
	db, err := infrastructure.NewPostgres(&cfg.Database)
	if err != nil {
		log.Fatalf("连接测试数据库失败: %v", err)
	}

	// 连接真实 Redis
	redis, err := infrastructure.NewRedis(&cfg.Redis)
	if err != nil {
		log.Fatalf("连接测试 Redis 失败: %v", err)
	}

	// 连接真实 MinIO (S3)
	cosClient, err := infrastructure.NewS3Storage(&cfg.COS)
	if err != nil {
		log.Fatalf("连接测试 MinIO 失败: %v", err)
	}

	// 初始化仓储
	sessionRepo := repository.NewSessionRepository(db)
	fileRepo := repository.NewFileRepository(db)
	appConfigRepo := repository.NewAppConfigRepository(db)
	llmModelRepo := repository.NewLLMModelRepository(db)

	// 初始化服务
	sessionSvc := service.NewSessionServiceWithSandbox(sessionRepo, fileRepo, "")
	fileSvc := service.NewFileService(fileRepo, cosClient)
	appConfigSvc := service.NewAppConfigService(appConfigRepo)
	llmModelSvc := service.NewLLMModelService(llmModelRepo)

	// 初始化处理器
	// 注意：agent 和 sandbox 传 nil，因为集成测试不涉及 AI 交互功能
	// Chat/Stop/ReadFile/ReadShell 接口会返回 "服务未配置" 错误，这是预期行为
	handlers := &router.Handlers{
		Session:   handler.NewSessionHandler(sessionSvc, nil, nil),
		File:      handler.NewFileHandler(fileSvc, sessionSvc),
		AppConfig: handler.NewAppConfigHandler(appConfigSvc),
		LLMModel:  handler.NewLLMModelHandler(llmModelSvc),
	}

	// 创建测试服务器
	engine := gin.New()
	router.SetupRoutes(engine, handlers)

	// 保存全局变量
	testCfg = cfg
	testDB = db
	testRedis = redis
	testServer = engine

	// 运行测试
	code := m.Run()

	// 清理
	db.Pool.Close()
	redis.Close()

	os.Exit(code)
}

// NewTestContext 返回带超时的测试上下文
func NewTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// CleanupSession 清理测试会话
func CleanupSession(t *testing.T, sessionID string) {
	ctx, cancel := NewTestContext()
	defer cancel()

	_, err := testDB.Pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		t.Logf("清理会话 %s 失败: %v", sessionID, err)
	}
}

// CleanupFile 清理测试文件记录
func CleanupFile(t *testing.T, fileID string) {
	ctx, cancel := NewTestContext()
	defer cancel()

	_, err := testDB.Pool.Exec(ctx, "DELETE FROM files WHERE id = $1", fileID)
	if err != nil {
		t.Logf("清理文件记录 %s 失败: %v", fileID, err)
	}
}

// CleanupAppConfig 清理测试配置
func CleanupAppConfig(t *testing.T, configType, configKey string) {
	ctx, cancel := NewTestContext()
	defer cancel()

	_, err := testDB.Pool.Exec(ctx, "DELETE FROM app_configs WHERE config_type = $1 AND config_key = $2", configType, configKey)
	if err != nil {
		t.Logf("清理配置 %s/%s 失败: %v", configType, configKey, err)
	}
}
