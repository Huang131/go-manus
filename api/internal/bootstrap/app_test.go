package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/service"
)

func TestAppCloseIsIdempotent(t *testing.T) {
	var closeCount int
	app := &App{
		lifecycle: &lifecycleManager{
			stopHooks: []func(){
				func() {
					closeCount++
				},
			},
		},
	}

	app.Close()
	app.Close()

	if closeCount != 1 {
		t.Fatalf("Close() called hooks %d times, want 1", closeCount)
	}
}

func TestAppCloseHandlesNilReceiver(t *testing.T) {
	var app *App
	app.Close()
}

func TestAppShutdownClosesRegisteredResources(t *testing.T) {
	var closeCount int
	app := &App{
		lifecycle: &lifecycleManager{
			stopHooks: []func(){
				func() {
					closeCount++
				},
			},
		},
	}

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if closeCount != 1 {
		t.Fatalf("Shutdown() closed resources %d times, want 1", closeCount)
	}
}

func TestAppStartRunsRegisteredHooksOnce(t *testing.T) {
	var startCount int
	app := &App{
		lifecycle: &lifecycleManager{},
	}
	app.startHook(func() error {
		startCount++
		return nil
	})

	if err := app.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := app.Start(); err != nil {
		t.Fatalf("Start() second call error = %v", err)
	}
	if startCount != 1 {
		t.Fatalf("Start() called hooks %d times, want 1", startCount)
	}
}

func TestInitExternalClientsDoesNotCreateBrowserWithoutSandboxAddress(t *testing.T) {
	app := &App{}
	_, browser, _, _, _, _ := app.initExternalClients(&config.Config{}, Options{
		EnableSandbox: true,
		EnableBrowser: true,
	})
	if browser != nil {
		t.Fatal("browser should be nil when sandbox address is empty")
	}
}

func TestBuildTestModeWithRoutes(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080},
	}

	app, err := Build(cfg, Options{
		EnablePostgres:    false,
		EnableRedis:       false,
		EnableStorage:     false,
		EnableLLM:         false,
		EnableSandbox:     false,
		EnableBrowser:     false,
		EnableSearch:      false,
		EnableAgent:       false,
		EnableRoutes:      true,
		EnableHealthCheck: false,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if app == nil {
		t.Fatal("Build() app is nil")
	}
	if app.Engine == nil {
		t.Fatal("Build() engine is nil")
	}
	if app.Server == nil {
		t.Fatal("Build() server is nil")
	}
	if app.Postgres != nil || app.Redis != nil {
		t.Fatal("Build() should not initialize disabled infrastructure")
	}
	app.Close()
}

func TestBuildZeroOptionsDisablesAllComponents(t *testing.T) {
	cfg := &config.Config{}

	app, err := Build(cfg, Options{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer app.Close()

	if app.Postgres != nil || app.Redis != nil || app.Engine != nil {
		t.Fatal("zero Options should not initialize components")
	}
}

func TestBuildRejectsNilInfrastructureFromFactory(t *testing.T) {
	cfg := &config.Config{}

	_, err := BuildWithFactories(cfg, Options{
		EnablePostgres: true,
	}, Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Fatal("Build() error = nil, want nil infrastructure error")
	}
	if !errors.Is(err, ErrInitialize) {
		t.Fatalf("Build() error = %v, want errors.Is(_, ErrInitialize)", err)
	}
}

func TestBuildInitializeErrorWrapsSentinel(t *testing.T) {
	cfg := &config.Config{}

	_, err := BuildWithFactories(cfg, Options{
		EnablePostgres: true,
	}, Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			return nil, errors.New("dial tcp: connection refused")
		},
	})
	if err == nil {
		t.Fatal("Build() error = nil, want initialization error")
	}
	if !errors.Is(err, ErrInitialize) {
		t.Fatalf("Build() error = %v, want errors.Is(_, ErrInitialize)", err)
	}
}

func TestBuildRejectsNilConfigSentinel(t *testing.T) {
	_, err := Build(nil, Options{})
	if err == nil {
		t.Fatal("Build(nil) error = nil, want config sentinel")
	}
	if !errors.Is(err, ErrConfigNil) {
		t.Fatalf("Build(nil) error = %v, want errors.Is(_, ErrConfigNil)", err)
	}
}

func TestBuildWithFactoriesInjectsFileCleanupScheduler(t *testing.T) {
	cfg := &config.Config{
		FileCleanup: config.FileCleanupConfig{
			ExpiresAfter: "24h",
			BatchSize:    100,
			Interval:     3600,
		},
	}
	var schedulerStartCount int
	var schedulerStopCount int

	app, err := BuildWithFactories(cfg, Options{
		EnablePostgres:    true,
		EnableRedis:       false,
		EnableStorage:     true,
		EnableLLM:         false,
		EnableSandbox:     false,
		EnableBrowser:     false,
		EnableSearch:      false,
		EnableAgent:       false,
		EnableRoutes:      false,
		EnableHealthCheck: false,
	}, Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			return &infrastructure.Postgres{}, nil
		},
		NewOSS: func(*config.ObjectStorageConfig) (*infrastructure.OSS, error) {
			return infrastructure.NewOSS(&config.ObjectStorageConfig{
				Provider:  "custom",
				Endpoint:  "http://127.0.0.1:9000",
				SecretID:  "test",
				SecretKey: "test",
				Bucket:    "test-bucket",
			})
		},
		NewFileCleanupScheduler: func(service.FileCleanupService, string, int, time.Duration) service.SchedulerRunner {
			return &mockScheduler{
				onStart: func() { schedulerStartCount++ },
				onStop:  func() { schedulerStopCount++ },
			}
		},
	})
	if err != nil {
		t.Fatalf("BuildWithFactories() error = %v", err)
	}
	if app == nil {
		t.Fatal("BuildWithFactories() app is nil")
	}
	defer app.Close()

	if schedulerStartCount != 1 {
		t.Fatalf("scheduler start count = %d, want 1", schedulerStartCount)
	}
	if schedulerStopCount != 0 {
		t.Fatalf("scheduler stop count before Close = %d, want 0", schedulerStopCount)
	}

	app.Close()
	if schedulerStopCount != 1 {
		t.Fatalf("scheduler stop count after Close = %d, want 1", schedulerStopCount)
	}
}

func TestInitAgentReturnsSentinelErrors(t *testing.T) {
	app := &App{lifecycle: newLifecycleManager()}

	if err := app.initAgent(Options{EnableAgent: true}, nil, nil, nil, nil, nil, nil); !errors.Is(err, ErrAgentRequiresDependencies) {
		t.Fatalf("initAgent() error = %v, want ErrAgentRequiresDependencies", err)
	}

	app.Postgres = &infrastructure.Postgres{}
	if err := app.initAgent(Options{EnableAgent: true}, nil, nil, nil, &mockMessageQueue{}, nil, nil); !errors.Is(err, ErrAgentRequiresLLM) {
		t.Fatalf("initAgent() error = %v, want ErrAgentRequiresLLM", err)
	}
}

type mockMessageQueue struct{}

func (m *mockMessageQueue) Put(ctx context.Context, streamName string, message interface{}) (string, error) {
	return "", nil
}
func (m *mockMessageQueue) GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error) {
	return "", nil, nil
}
func (m *mockMessageQueue) Clear(ctx context.Context, streamName string) error {
	return nil
}
func (m *mockMessageQueue) SetRetention(ctx context.Context, streamName string, retention time.Duration) error {
	return nil
}
func (m *mockMessageQueue) IsEmpty(ctx context.Context, streamName string) (bool, error) {
	return true, nil
}
func (m *mockMessageQueue) Size(ctx context.Context, streamName string) (int64, error) {
	return 0, nil
}

func TestBuildTestModeWithRoutesHealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080},
	}

	app, err := Build(cfg, Options{
		EnablePostgres:    false,
		EnableRedis:       false,
		EnableStorage:     false,
		EnableLLM:         false,
		EnableSandbox:     false,
		EnableBrowser:     false,
		EnableSearch:      false,
		EnableAgent:       false,
		EnableRoutes:      true,
		EnableHealthCheck: false,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer app.Close()

	// 构造 GET /health 请求
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// 执行请求（不走网络，直接用 Engine）
	app.Engine.ServeHTTP(w, req)

	// 断言：不能是 500 错误（修复前会 panic 导致 500）
	if w.Code == http.StatusInternalServerError {
		t.Fatalf("GET /health returned 500, expected 200 or degraded status")
	}

	// 期望 200，返回包含 service status 的 JSON
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health returned %d, want 200", w.Code)
	}

	// 解析响应，验证结构
	var resp struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("response code = %d, want 0", resp.Code)
	}

	if resp.Data == nil {
		t.Fatal("response missing 'data' field")
	}

	if resp.Data["status"] == nil {
		t.Fatal("response data missing 'status' field")
	}

	services, ok := resp.Data["services"].(map[string]interface{})
	if !ok {
		t.Fatal("response data missing 'services' field")
	}

	// 验证 skipped 状态（因为 Postgres/Redis/OSS 未启用）
	for _, name := range []string{"postgres", "redis", "oss"} {
		if s, exists := services[name]; exists {
			if svc, ok := s.(map[string]interface{}); ok {
				if svc["status"] != "skipped" {
					t.Errorf("service %s status = %v, want 'skipped'", name, svc["status"])
				}
			}
		}
	}
}

type mockScheduler struct {
	onStart func()
	onStop  func()
}

func (m *mockScheduler) Start() {
	if m.onStart != nil {
		m.onStart()
	}
}

func (m *mockScheduler) Stop() {
	if m.onStop != nil {
		m.onStop()
	}
}

// ============================================================================
// initInfrastructure 单元测试
// ============================================================================

func TestInitInfrastructurePostgresFailure(t *testing.T) {
	app := &App{lifecycle: newLifecycleManager()}
	cfg := &config.Config{}
	opts := Options{EnablePostgres: true}
	factories := Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			return nil, errors.New("connection refused")
		},
	}

	err := app.initInfrastructure(cfg, opts, factories)
	if err == nil {
		t.Fatal("initInfrastructure() error = nil, want postgres failure error")
	}
	if !errors.Is(err, ErrInitialize) {
		t.Fatalf("error should wrap ErrInitialize, got: %v", err)
	}
	// 验证关闭钩子没有被注册（因为初始化失败）
	app.Close() // 不应 panic
}

func TestInitInfrastructureRedisFailureWithNilCheck(t *testing.T) {
	app := &App{lifecycle: newLifecycleManager()}
	cfg := &config.Config{}
	opts := Options{EnablePostgres: true, EnableRedis: true}
	factories := Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			return &infrastructure.Postgres{}, nil
		},
		NewRedis: func(*config.RedisConfig) (*infrastructure.Redis, error) {
			return nil, errors.New("redis connection refused")
		},
	}

	err := app.initInfrastructure(cfg, opts, factories)
	if err == nil {
		t.Fatal("initInfrastructure() error = nil, want redis failure error")
	}
	if !errors.Is(err, ErrInitialize) {
		t.Fatalf("error should wrap ErrInitialize, got: %v", err)
	}
	// Postgres 应该已注册关闭钩子
	app.Close()
}

func TestInitInfrastructureOSSFailureDoesNotBlockStartup(t *testing.T) {
	app := &App{lifecycle: newLifecycleManager()}
	cfg := &config.Config{}
	opts := Options{EnablePostgres: true, EnableRedis: true, EnableStorage: true}
	factories := Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			return &infrastructure.Postgres{}, nil
		},
		NewRedis: func(*config.RedisConfig) (*infrastructure.Redis, error) {
			return &infrastructure.Redis{}, nil
		},
		NewOSS: func(*config.ObjectStorageConfig) (*infrastructure.OSS, error) {
			return nil, errors.New("oss connection refused")
		},
	}

	// OSS 失败不应该返回错误，而是优雅降级
	err := app.initInfrastructure(cfg, opts, factories)
	if err != nil {
		t.Fatalf("initInfrastructure() error = %v, want nil (OSS failure should be graceful)", err)
	}
	if app.OSS != nil {
		t.Fatal("app.OSS should be nil when OSS initialization fails")
	}
	// Postgres 和 Redis 应该初始化成功
	if app.Postgres == nil || app.Redis == nil {
		t.Fatal("Postgres and Redis should be initialized despite OSS failure")
	}
	app.Close()
}

func TestInitInfrastructureSkipsDisabledComponents(t *testing.T) {
	app := &App{lifecycle: newLifecycleManager()}
	cfg := &config.Config{}
	opts := Options{
		EnablePostgres: false,
		EnableRedis:    false,
		EnableStorage:  false,
	}
	factories := Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			t.Fatal("NewPostgres should not be called when EnablePostgres is false")
			return nil, nil
		},
	}

	err := app.initInfrastructure(cfg, opts, factories)
	if err != nil {
		t.Fatalf("initInfrastructure() error = %v, want nil", err)
	}
	if app.Postgres != nil || app.Redis != nil || app.OSS != nil {
		t.Fatal("no infrastructure should be initialized when all are disabled")
	}
	app.Close()
}

func TestInitInfrastructureNilFactoryReturnsNil(t *testing.T) {
	app := &App{lifecycle: newLifecycleManager()}
	cfg := &config.Config{}
	opts := Options{EnablePostgres: true}
	// 使用空的 factories，normalizeFactories 会填充默认值
	factories := normalizeFactories(Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			return nil, nil // factory 返回 nil 但没有错误
		},
	})

	err := app.initInfrastructure(cfg, opts, factories)
	if err == nil {
		t.Fatal("initInfrastructure() should return error when factory returns nil")
	}
	if !strings.Contains(err.Error(), "factory returned nil") {
		t.Fatalf("error should mention 'factory returned nil', got: %v", err)
	}
}

// ============================================================================
// BuildWithFactories 依赖注入单元测试
// ============================================================================

func TestBuildWithFactoriesPartialInjection(t *testing.T) {
	cfg := &config.Config{}
	var postgresCalled bool

	// 只注入 Postgres mock，验证注入的 factory 被保留（normalizeFactories 不覆盖非 nil 字段）。
	// Redis/Storage 禁用，避免未注入时 normalizeFactories 填充的默认 factory 发起真实连接；
	// "部分注入 + 默认填充"的语义由 TestNormalizeFactoriesPartialInjection 覆盖。
	app, err := BuildWithFactories(cfg, Options{
		EnablePostgres:    true,
		EnableRedis:       false,
		EnableStorage:     false,
		EnableLLM:         false,
		EnableAgent:       false,
		EnableRoutes:      false,
		EnableHealthCheck: false,
	}, Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			postgresCalled = true
			return &infrastructure.Postgres{}, nil
		},
	})

	if err != nil {
		t.Fatalf("BuildWithFactories() error = %v", err)
	}
	if app == nil {
		t.Fatal("BuildWithFactories() app is nil")
	}
	if !postgresCalled {
		t.Fatal("NewPostgres should be called")
	}
	app.Close()
}

func TestBuildWithFactoriesFactoryErrorCleanedUp(t *testing.T) {
	cfg := &config.Config{}
	var postgresInitialized, redisCalled bool

	// Postgres 初始化成功，但 Redis 失败
	_, err := BuildWithFactories(cfg, Options{
		EnablePostgres:    true,
		EnableRedis:       true,
		EnableStorage:     false,
		EnableLLM:         false,
		EnableAgent:       false,
		EnableRoutes:      false,
		EnableHealthCheck: false,
	}, Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			postgresInitialized = true
			return &infrastructure.Postgres{}, nil
		},
		NewRedis: func(*config.RedisConfig) (*infrastructure.Redis, error) {
			redisCalled = true
			return nil, errors.New("redis error")
		},
	})

	if err == nil {
		t.Fatal("BuildWithFactories() should return error when Redis fails")
	}
	if !postgresInitialized {
		t.Fatal("Postgres should be initialized before Redis fails")
	}
	if !redisCalled {
		t.Fatal("Redis factory should be called")
	}
	// 验证错误链包含 ErrInitialize
	if !errors.Is(err, ErrInitialize) {
		t.Fatalf("error should wrap ErrInitialize, got: %v", err)
	}
}

func TestBuildWithFactoriesProductionModeSetsGinReleaseMode(t *testing.T) {
	cfg := &config.Config{Env: config.EnvProduction}
	gin.SetMode(gin.DebugMode) // 先重置为 debug 模式

	app, err := BuildWithFactories(cfg, Options{
		EnableRoutes:      true,
		EnableHealthCheck: false,
	}, Factories{})

	if err != nil {
		t.Fatalf("BuildWithFactories() error = %v", err)
	}
	if app == nil || app.Engine == nil {
		t.Fatal("Engine should be initialized")
	}

	// 验证 gin 模式已设置为 release
	if gin.Mode() != gin.ReleaseMode {
		t.Errorf("gin.Mode() = %s, want %s", gin.Mode(), gin.ReleaseMode)
	}
	app.Close()
}

// ============================================================================
// normalizeFactories 单元测试
// ============================================================================

func TestNormalizeFactoriesPartialInjection(t *testing.T) {
	// 测试部分注入：只注入 Redis，其他使用默认值
	factories := Factories{
		NewRedis: func(*config.RedisConfig) (*infrastructure.Redis, error) {
			return &infrastructure.Redis{}, nil
		},
	}

	normalized := normalizeFactories(factories)

	// NewRedis 应该保持注入的值
	if normalized.NewRedis == nil {
		t.Fatal("NewRedis should be set")
	}
	// NewPostgres 和 NewOSS 应该使用默认值
	if normalized.NewPostgres == nil {
		t.Fatal("NewPostgres should use default")
	}
	if normalized.NewOSS == nil {
		t.Fatal("NewOSS should use default")
	}
	if normalized.NewFileCleanupScheduler == nil {
		t.Fatal("NewFileCleanupScheduler should use default")
	}
}

func TestNormalizeFactoriesFullInjection(t *testing.T) {
	// 用标记变量验证注入的函数是否被调用
	var customPostgresCalled, customRedisCalled bool

	customFactories := Factories{
		NewPostgres: func(*config.DatabaseConfig) (*infrastructure.Postgres, error) {
			customPostgresCalled = true
			return nil, nil
		},
		NewRedis: func(*config.RedisConfig) (*infrastructure.Redis, error) {
			customRedisCalled = true
			return nil, nil
		},
	}

	// 测试 normalizeFactories 是否保留注入的函数
	normalized := normalizeFactories(customFactories)

	// 如果 normalizeFactories 正确保留注入的函数，那么调用 normalized 的函数会触发我们的标记
	normalized.NewPostgres(nil)
	normalized.NewRedis(nil)

	if !customPostgresCalled {
		t.Fatal("NewPostgres should call custom function, not default")
	}
	if !customRedisCalled {
		t.Fatal("NewRedis should call custom function, not default")
	}
}

// ============================================================================
// mock 类型补充
// ============================================================================
