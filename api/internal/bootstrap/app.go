package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/internal/agent"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/handler"
	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/router"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/Huang131/go-manus/api/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// Options 控制 bootstrap 的装配行为。
type Options struct {
	EnablePostgres    bool
	EnableRedis       bool
	EnableStorage     bool
	EnableLLM         bool
	EnableSandbox     bool
	EnableBrowser     bool
	EnableSearch      bool
	EnableAgent       bool
	EnableRoutes      bool
	EnableHealthCheck bool
}

// DefaultOptions 返回生产默认装配选项。
func DefaultOptions() Options {
	return Options{
		EnablePostgres:    true,
		EnableRedis:       true,
		EnableStorage:     true,
		EnableLLM:         true,
		EnableSandbox:     true,
		EnableBrowser:     true,
		EnableSearch:      true,
		EnableAgent:       true,
		EnableRoutes:      true,
		EnableHealthCheck: true,
	}
}

// Factories 允许测试或不同启动模式替换底层客户端构造器。
type Factories struct {
	NewPostgres func(*config.DatabaseConfig) (*infrastructure.Postgres, error)
	NewRedis    func(*config.RedisConfig) (*infrastructure.Redis, error)
	NewOSS      func(*config.ObjectStorageConfig) (*infrastructure.OSS, error)
}

type repositories struct {
	appConfig repository.AppConfigRepository
	llmModel  repository.LLMModelRepository
	session   repository.SessionRepository
	file      repository.FileRepository
}

func newRepositories(db *infrastructure.Postgres) repositories {
	return repositories{
		appConfig: repository.NewAppConfigRepository(db),
		llmModel:  repository.NewLLMModelRepository(db),
		session:   repository.NewSessionRepository(db),
		file:      repository.NewFileRepository(db),
	}
}

func defaultFactories() Factories {
	return Factories{
		NewPostgres: infrastructure.NewPostgres,
		NewRedis:    infrastructure.NewRedis,
		NewOSS:      infrastructure.NewOSS,
	}
}

// App 代表已装配完成的应用。
type App struct {
	Config         *config.Config
	Server         *http.Server
	Engine         *gin.Engine
	Postgres       *infrastructure.Postgres
	Redis          *infrastructure.Redis
	OSS            *infrastructure.OSS
	Sandbox        *external.SandboxClient
	AgentService   *agent.AgentService
	SessionService service.SessionService
	FileService    service.FileService
	StatusService  service.StatusService
	AppConfigSvc   service.AppConfigService
	LLMModelSvc    service.LLMModelService

	repos         repositories
	shutdownHooks []func()
	closeOnce     sync.Once
}

// Close 关闭应用资源。关闭动作只执行一次，便于同时支持 defer 和显式关闭。
func (a *App) Close() {
	if a == nil {
		return
	}

	a.closeOnce.Do(func() {
		// 按注册顺序逆序关闭，保证上层服务先于底层连接退出。
		for i := len(a.shutdownHooks) - 1; i >= 0; i-- {
			if a.shutdownHooks[i] != nil {
				a.shutdownHooks[i]()
			}
		}
	})
}

// Shutdown 先停止 HTTP 服务，再释放应用依赖，统一生产和测试的退出顺序。
//
// 返回的错误会通过 context.DeadlineExceeded 区分超时与真正的关闭失败，
// 方便 main 据此选择不同的退出码或告警级别。
func (a *App) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}

	var shutdownErr error
	if a.Server != nil {
		shutdownErr = a.Server.Shutdown(ctx)
		if shutdownErr != nil && errors.Is(shutdownErr, context.DeadlineExceeded) {
			// 超时场景：把上下文错误包一层，方便调用方 errors.Is 判定。
			shutdownErr = fmt.Errorf("graceful shutdown timed out: %w", shutdownErr)
		}
	}
	a.Close()
	return shutdownErr
}

func normalizeFactories(factories Factories) Factories {
	defaults := defaultFactories()
	if factories.NewPostgres == nil {
		factories.NewPostgres = defaults.NewPostgres
	}
	if factories.NewRedis == nil {
		factories.NewRedis = defaults.NewRedis
	}
	if factories.NewOSS == nil {
		factories.NewOSS = defaults.NewOSS
	}
	return factories
}

func (a *App) initInfrastructure(cfg *config.Config, opts Options, factories Factories) error {
	var err error
	if opts.EnablePostgres {
		a.Postgres, err = factories.NewPostgres(&cfg.Database)
		if err != nil {
			if a.Postgres != nil {
				a.Postgres.Close()
			}
			return fmt.Errorf("postgres: %w: %w", ErrInitialize, err)
		}
		if a.Postgres == nil {
			return fmt.Errorf("postgres: %w: factory returned nil", ErrInitialize)
		}
		a.shutdownHooks = append(a.shutdownHooks, func() {
			a.Postgres.Close()
		})
	}

	if opts.EnableRedis {
		a.Redis, err = factories.NewRedis(&cfg.Redis)
		if err != nil {
			if a.Redis != nil {
				_ = a.Redis.Close()
			}
			return fmt.Errorf("redis: %w: %w", ErrInitialize, err)
		}
		if a.Redis == nil {
			return fmt.Errorf("redis: %w: factory returned nil", ErrInitialize)
		}
		a.shutdownHooks = append(a.shutdownHooks, func() {
			_ = a.Redis.Close()
		})
	}

	if opts.EnableStorage {
		oss, storageErr := factories.NewOSS(&cfg.OSS)
		if storageErr != nil {
			if oss != nil {
				_ = oss.Close()
			}
			logger.Warn("storage init failed, continuing without storage", logger.Err(storageErr))
		} else {
			a.OSS = oss
			if a.OSS == nil {
				return fmt.Errorf("storage: %w: factory returned nil", ErrInitialize)
			}
			a.shutdownHooks = append(a.shutdownHooks, func() {
				_ = a.OSS.Close()
			})
		}
	}

	return nil
}

func (a *App) initServices(cfg *config.Config) {
	a.repos = newRepositories(a.Postgres)
	a.SessionService = service.NewSessionServiceWithSandbox(a.repos.session, a.repos.file, cfg.Sandbox.Address)
	a.FileService = service.NewFileService(a.repos.file, a.OSS)
	a.StatusService = service.NewStatusService(a.Postgres, a.Redis, a.OSS)
	a.AppConfigSvc = service.NewAppConfigService(a.repos.appConfig)
	a.LLMModelSvc = service.NewLLMModelService(a.repos.llmModel)
}

func (a *App) initLLM(cfg *config.Config, opts Options) external.LLM {
	if !opts.EnableLLM || cfg.LLM.BaseURL == "" {
		return nil
	}

	fallbackLLMCfg := &external.LLMRuntimeConfig{
		Profile:         llmcore.ModelProfile{Protocol: llmcore.ProtocolOpenAICompat},
		BaseURL:         cfg.LLM.BaseURL,
		APIKey:          cfg.LLM.APIKey,
		ModelName:       cfg.LLM.ModelName,
		Temperature:     cfg.LLM.Temperature,
		MaxTokens:       cfg.LLM.MaxTokens,
		ToolCallTimeout: cfg.LLM.ToolCallTimeout,
	}
	routed := external.NewRoutedLLMFromSingleProvider(func(ctx context.Context) (*external.LLMRuntimeConfig, error) {
		if a.Postgres != nil && a.repos.llmModel != nil {
			if mid := external.ModelIDFromContext(ctx); mid != "" {
				chosen, err := a.repos.llmModel.GetByID(ctx, mid)
				if err != nil {
					logger.Warn("failed to get model by ID", logger.String("model_id", mid), logger.Err(err))
				} else if chosen != nil && chosen.IsEnabled {
					return external.BuildRuntimeConfigFromModel(chosen, cfg.LLM.ToolCallTimeout), nil
				}
			}
			if def, err := a.repos.llmModel.GetDefault(ctx); err != nil {
				logger.Warn("failed to get default model", logger.Err(err))
			} else if def != nil && def.IsEnabled {
				return external.BuildRuntimeConfigFromModel(def, cfg.LLM.ToolCallTimeout), nil
			}
			if first, err := a.repos.llmModel.GetFirstEnabled(ctx); err != nil {
				logger.Warn("failed to get first enabled model", logger.Err(err))
			} else if first != nil {
				return external.BuildRuntimeConfigFromModel(first, cfg.LLM.ToolCallTimeout), nil
			}
		}
		return nil, nil
	}, fallbackLLMCfg, nil)
	if a.Postgres != nil {
		routed.SetHealthStore(a.repos.llmModel)
	}
	return routed
}

func (a *App) initExternalClients(cfg *config.Config, opts Options) (external.LLM, external.Browser, external.SearchEngine, external.MessageQueue, *agent.MCPConfig, *agent.A2AConfig) {
	llm := a.initLLM(cfg, opts)

	if opts.EnableSandbox {
		a.Sandbox = external.NewSandboxClient(&cfg.Sandbox)
	}

	var browser external.Browser
	if opts.EnableBrowser && a.Sandbox != nil && cfg.Sandbox.Address != "" {
		browser = external.NewBrowserClient(a.Sandbox)
	}

	var search external.SearchEngine
	if opts.EnableSearch && (cfg.Search.BingAPIKey != "" || cfg.Search.GoogleAPIKey != "") {
		search = external.NewSearchEngine(&external.SearchConfig{
			Provider:       cfg.Search.Provider,
			BingAPIKey:     cfg.Search.BingAPIKey,
			GoogleAPIKey:   cfg.Search.GoogleAPIKey,
			SearchEngineID: cfg.Search.SearchEngineID,
		})
	}

	var mq external.MessageQueue
	if a.Redis != nil {
		mq = external.NewRedisStreamMessageQueue(a.Redis.Client)
	}

	return llm, browser, search, mq, newMCPConfig(cfg), newA2AConfig(cfg)
}

func newMCPConfig(cfg *config.Config) *agent.MCPConfig {
	if len(cfg.MCP.Servers) == 0 {
		return nil
	}
	servers := make([]agent.MCPServer, len(cfg.MCP.Servers))
	for i, server := range cfg.MCP.Servers {
		servers[i] = agent.MCPServer{
			Name: server.Name, Command: server.Command, Args: server.Args, Env: server.Env,
		}
	}
	return &agent.MCPConfig{Servers: servers, Timeout: cfg.MCP.Timeout}
}

func newA2AConfig(cfg *config.Config) *agent.A2AConfig {
	if len(cfg.A2A.Agents) == 0 {
		return nil
	}
	agents := make([]agent.A2AAgent, len(cfg.A2A.Agents))
	for i, a := range cfg.A2A.Agents {
		agents[i] = agent.A2AAgent{Name: a.Name, URL: a.URL, Metadata: a.Metadata}
	}
	return &agent.A2AConfig{Agents: agents, Timeout: cfg.A2A.Timeout}
}

func (a *App) initAgent(opts Options, llm external.LLM, browser external.Browser, search external.SearchEngine, mq external.MessageQueue, mcpConfig *agent.MCPConfig, a2aConfig *agent.A2AConfig) error {
	if !opts.EnableAgent {
		return nil
	}
	if a.Postgres == nil || mq == nil {
		return fmt.Errorf("agent requires postgres and redis")
	}
	if llm == nil {
		return fmt.Errorf("agent requires llm")
	}

	a.AgentService = agent.NewAgentService(
		a.repos.session, a.repos.file, a.repos.appConfig, llm, a.Sandbox,
		agent.DefaultAgentConfig(), mcpConfig, a2aConfig, browser, search, mq, a.OSS,
	)
	a.shutdownHooks = append(a.shutdownHooks, func() {
		if a.AgentService != nil {
			a.AgentService.Shutdown()
		}
	})
	return nil
}

func (a *App) initRoutes(cfg *config.Config, opts Options) {
	if !opts.EnableRoutes {
		return
	}

	engine := gin.New()
	// 把受信反向代理列表注入 Gin，避免使用默认的 0.0.0.0/0（生产环境不安全）。
	// 没配置（空切片）时回退到 ["127.0.0.1", "::1"]，只信任本机回环。
	trustedProxies := cfg.Server.TrustedProxies
	if len(trustedProxies) == 0 {
		trustedProxies = []string{"127.0.0.1", "::1"}
		logger.Warn("trusted_proxies not configured, falling back to loopback addresses")
	}
	if err := engine.SetTrustedProxies(trustedProxies); err != nil {
		logger.Warn("invalid trusted_proxies config, falling back to loopback",
			logger.Strings("configured", trustedProxies),
			logger.Err(err))
		_ = engine.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}
	engine.Use(middleware.Recovery(), middleware.Logger(), middleware.CORS(), middleware.RequestID())
	sessionHandler := handler.NewSessionHandler(a.SessionService, a.AgentService, a.Sandbox)
	fileHandler := handler.NewFileHandler(a.FileService, a.SessionService)
	statusHandler := handler.NewStatusHandler(a.StatusService)
	appConfigHandler := handler.NewAppConfigHandler(a.AppConfigSvc)
	llmModelHandler := handler.NewLLMModelHandler(a.LLMModelSvc)
	router.SetupRoutes(engine, &router.Handlers{
		Session: sessionHandler, File: fileHandler, Status: statusHandler,
		AppConfig: appConfigHandler, LLMModel: llmModelHandler,
	})

	a.Engine = engine
	// 超时由配置驱动；缺省时回落到保守的 30s 读 / 60s 空闲，写超时 0 留给流式接口。
	a.Server = &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      engine,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeoutSec) * time.Second,
	}
}

func (a *App) healthCheck(opts Options, cfg *config.Config) error {
	if !opts.EnableHealthCheck {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if a.Postgres != nil {
		if err := a.Postgres.HealthCheck(ctx); err != nil {
			return fmt.Errorf("postgres: %w: %w", ErrHealthCheckFailed, err)
		}
	}
	if a.Redis != nil {
		if err := a.Redis.HealthCheck(ctx); err != nil {
			return fmt.Errorf("redis: %w: %w", ErrHealthCheckFailed, err)
		}
	}
	if a.Sandbox != nil && cfg.Sandbox.Address != "" {
		if err := a.Sandbox.HealthCheck(ctx); err != nil {
			logger.Warn("Sandbox health check failed, continuing...", logger.Err(err))
		}
	}
	return nil
}

// Build 构建完整应用。
func Build(cfg *config.Config, opts Options) (*App, error) {
	return BuildWithFactories(cfg, opts, defaultFactories())
}

// BuildWithFactories 构建应用，并允许调用方注入基础设施构造器。
func BuildWithFactories(cfg *config.Config, opts Options, factories Factories) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	factories = normalizeFactories(factories)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	app := &App{Config: cfg}
	if err := app.initInfrastructure(cfg, opts, factories); err != nil {
		app.Close()
		return nil, err
	}
	app.initServices(cfg)
	llm, browser, search, mq, mcpConfig, a2aConfig := app.initExternalClients(cfg, opts)
	if err := app.initAgent(opts, llm, browser, search, mq, mcpConfig, a2aConfig); err != nil {
		app.Close()
		return nil, err
	}
	app.initRoutes(cfg, opts)
	if err := app.healthCheck(opts, cfg); err != nil {
		app.Close()
		return nil, err
	}
	return app, nil
}
