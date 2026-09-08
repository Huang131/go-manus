package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/internal/infrastructure"
)

func TestAppCloseIsIdempotent(t *testing.T) {
	var closeCount int
	app := &App{
		shutdownHooks: []func(){
			func() {
				closeCount++
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
		shutdownHooks: []func(){
			func() {
				closeCount++
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
