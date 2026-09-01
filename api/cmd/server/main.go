package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/internal/agent"
	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/handler"
	"github.com/mooc-manus/go-manus/api/internal/infrastructure"
	"github.com/mooc-manus/go-manus/api/internal/repository"
	"github.com/mooc-manus/go-manus/api/internal/router"
	"github.com/mooc-manus/go-manus/api/internal/service"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"github.com/mooc-manus/go-manus/api/pkg/middleware"
)

var (
	configPath    = flag.String("config", "config.yaml", "config file path")
	runMigrations = flag.Bool("migrate", false, "run database migrations before starting")
)

func main() {
	flag.Parse()

	// 1. 加载配置
	if err := config.Init(*configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	cfg := config.Get()
	logger.Info("Configuration loaded successfully",
		zap.String("env", cfg.Env),
		zap.String("log_level", cfg.LogLevel),
	)

	// 2. 初始化日志
	if err := logger.Init(cfg.LogLevel); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 3. 验证配置
	if err := validateConfig(cfg); err != nil {
		logger.Error("Configuration validation failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("Configuration validated successfully")

	// 4. 设置 Gin 模式
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 5. 初始化基础设施层
	logger.Info("Initializing infrastructure layer...")

	db, err := infrastructure.NewPostgres(&cfg.Database)
	if err != nil {
		logger.Error("Failed to init postgres", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("PostgreSQL connection established",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Database),
	)
	defer db.Close()

	redis, err := infrastructure.NewRedis(&cfg.Redis)
	if err != nil {
		logger.Error("Failed to init redis", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("Redis connection established",
		zap.String("addr", cfg.Redis.Addr()),
	)
	defer redis.Close()

	cos, err := infrastructure.NewS3Storage(&cfg.COS)
	if err != nil {
		logger.Warn("Storage initialization failed, continuing without storage", zap.Error(err))
	} else {
		logger.Info("S3-compatible storage client initialized",
			zap.String("endpoint", cfg.COS.Endpoint),
			zap.String("region", cfg.COS.Region),
			zap.String("bucket", cfg.COS.Bucket),
		)
	}

	// 6. 初始化外部服务层
	logger.Info("Initializing external services...")

	// 6.1 LLM 客户端 (OpenAI 兼容) — 支持从 llm_models 表动态加载配置
	// 提前创建配置仓库，供 LLM 动态配置 provider 复用（后续步骤 8 复用该实例）。
	configRepo := repository.NewAppConfigRepository(db)
	llmModelRepo := repository.NewLLMModelRepository(db)

	var llm external.LLM
	fallbackLLMCfg := &external.OpenAIClientConfig{
		BaseURL:         cfg.LLM.BaseURL,
		APIKey:          cfg.LLM.APIKey,
		ModelName:       cfg.LLM.ModelName,
		Temperature:     cfg.LLM.Temperature,
		MaxTokens:       cfg.LLM.MaxTokens,
		ToolCallTimeout: cfg.LLM.ToolCallTimeout,
	}
	if cfg.LLM.BaseURL != "" {
		llm = external.NewDynamicLLM(
			func(ctx context.Context) (*external.OpenAIClientConfig, error) {
				// 优先：llm_models 表 default 模型
				def, err := llmModelRepo.GetDefault(ctx)
				if err == nil && def != nil && def.IsEnabled {
					return &external.OpenAIClientConfig{
						BaseURL:         def.BaseURL,
						APIKey:          def.APIKey,
						ModelName:       def.ModelName,
						Temperature:     def.Temperature,
						MaxTokens:       def.MaxTokens,
						ToolCallTimeout: cfg.LLM.ToolCallTimeout,
					}, nil
				}
				// 兜底：llm_models 第一个 enabled
				first, err := llmModelRepo.GetFirstEnabled(ctx)
				if err == nil && first != nil {
					return &external.OpenAIClientConfig{
						BaseURL:         first.BaseURL,
						APIKey:          first.APIKey,
						ModelName:       first.ModelName,
						Temperature:     first.Temperature,
						MaxTokens:       first.MaxTokens,
						ToolCallTimeout: cfg.LLM.ToolCallTimeout,
					}, nil
				}
				// 兜底 2：env
				return nil, nil
			},
			fallbackLLMCfg,
		)
		logger.Info("LLM client initialized (dynamic, llm_models table)",
			zap.String("base_url", cfg.LLM.BaseURL),
			zap.String("model", cfg.LLM.ModelName),
		)
	}

	// 6.2 Sandbox 客户端
	sandbox := external.NewSandboxClient(&cfg.Sandbox)
	logger.Info("Sandbox client initialized",
		zap.String("address", cfg.Sandbox.Address),
	)

	// 6.3 Browser 客户端 (基于 Sandbox)
	var browser external.Browser
	if cfg.Sandbox.Address != "" {
		browser = external.NewBrowserClient(sandbox)
		logger.Info("Browser client initialized")
	}

	// 6.4 Search 客户端
	var search external.SearchEngine
	if cfg.Search.BingAPIKey != "" || cfg.Search.GoogleAPIKey != "" {
		search = external.NewSearchEngine(&external.SearchConfig{
			Provider:       cfg.Search.Provider,
			BingAPIKey:     cfg.Search.BingAPIKey,
			GoogleAPIKey:   cfg.Search.GoogleAPIKey,
			SearchEngineID: cfg.Search.SearchEngineID,
		})
		logger.Info("Search engine initialized",
			zap.String("provider", cfg.Search.Provider),
		)
	}

	// 6.5 记录外部服务状态 (将在 Agent 服务中使用)
	logger.Info("External services status",
		zap.Bool("llm_configured", llm != nil),
		zap.Bool("browser_configured", browser != nil),
		zap.Bool("search_configured", search != nil),
	)

	// 7. 启动时健康检查
	logger.Info("Running startup health checks...")
	if err := startupHealthCheck(db, redis, sandbox, &cfg.Sandbox); err != nil {
		logger.Error("Startup health check failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("All health checks passed")

	// 8. 初始化 Repository 层
	sessionRepo := repository.NewSessionRepository(db)
	fileRepo := repository.NewFileRepository(db)
	// configRepo 已在步骤 6.1 提前创建（供 LLM 动态配置 provider 使用），此处复用
	logger.Info("Repository layer initialized")

	// 9. 初始化 Service 层
	sessionService := service.NewSessionServiceWithSandbox(sessionRepo, cfg.Sandbox.Address)
	fileService := service.NewFileService(fileRepo, cos)
	statusService := service.NewStatusService(db, redis, cos)
	appConfigService := service.NewAppConfigService(configRepo)
	llmModelService := service.NewLLMModelService(llmModelRepo)
	logger.Info("Service layer initialized")

	// 9.1 构造 MCP 配置
	var mcpConfig *agent.MCPConfig
	if len(cfg.MCP.Servers) > 0 {
		servers := make([]agent.MCPServer, len(cfg.MCP.Servers))
		for i, s := range cfg.MCP.Servers {
			servers[i] = agent.MCPServer{
				Name:    s.Name,
				Command: s.Command,
				Args:    s.Args,
				Env:     s.Env,
			}
		}
		mcpConfig = &agent.MCPConfig{
			Servers: servers,
			Timeout: cfg.MCP.Timeout,
		}
		logger.Info("MCP config prepared",
			zap.Int("servers", len(cfg.MCP.Servers)))
	}

	// 9.2 构造 A2A 配置
	var a2aConfig *agent.A2AConfig
	if len(cfg.A2A.Agents) > 0 {
		agents := make([]agent.A2AAgent, len(cfg.A2A.Agents))
		for i, a := range cfg.A2A.Agents {
			agents[i] = agent.A2AAgent{
				Name:     a.Name,
				URL:      a.URL,
				Metadata: a.Metadata,
			}
		}
		a2aConfig = &agent.A2AConfig{
			Agents:  agents,
			Timeout: cfg.A2A.Timeout,
		}
		logger.Info("A2A config prepared",
			zap.Int("agents", len(cfg.A2A.Agents)))
	}

	// 9.3 初始化 Agent Service
	agentConfig := agent.DefaultAgentConfig()
	mq := external.NewRedisStreamMessageQueue(redis.Client)
	agentService := agent.NewAgentService(
		sessionRepo,
		fileRepo,
		configRepo,
		llm,
		sandbox,
		agentConfig,
		mcpConfig,
		a2aConfig,
		browser,
		search,
		mq,
		cos,
	)
	logger.Info("Agent service initialized",
		zap.Bool("browser_enabled", browser != nil),
		zap.Bool("search_enabled", search != nil),
		zap.Bool("mcp_enabled", mcpConfig != nil),
		zap.Bool("a2a_enabled", a2aConfig != nil),
	)

	// 注册 AgentService 关闭清理
	defer func() {
		if agentService != nil {
			agentService.Shutdown()
		}
	}()

	// 10. 初始化 Handler 层
	sessionHandler := handler.NewSessionHandler(sessionService, agentService, sandbox)
	fileHandler := handler.NewFileHandler(fileService)
	statusHandler := handler.NewStatusHandler(statusService)
	appConfigHandler := handler.NewAppConfigHandler(appConfigService)
	llmModelHandler := handler.NewLLMModelHandler(llmModelService)
	logger.Info("Handler layer initialized")

	// 11. 创建 Gin 引擎
	engine := gin.New()
	engine.Use(middleware.Recovery())
	engine.Use(middleware.Logger())
	engine.Use(middleware.CORS())
	engine.Use(middleware.RequestID())

	// 12. 注册路由
	router.SetupRoutes(engine, &router.Handlers{
		Session:   sessionHandler,
		File:      fileHandler,
		Status:    statusHandler,
		AppConfig: appConfigHandler,
		LLMModel:  llmModelHandler,
	})
	logger.Info("Routes registered")

	// 13. 创建 HTTP 服务器
	// 注意：SSE 流需要长时间保持连接，WriteTimeout 设置为 0 表示无超时
	// 生产环境建议使用反向代理（如 nginx）来处理超时控制
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE 流需要无超时
		IdleTimeout:  60 * time.Second,
	}

	// 14. 启动服务器 (goroutine)
	go func() {
		logger.Info("Server starting",
			zap.String("addr", cfg.Server.Addr()),
			zap.String("env", cfg.Env),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", zap.Error(err))
			os.Exit(1)
		}
	}()

	logger.Info("Manus started successfully",
		zap.String("addr", cfg.Server.Addr()),
		zap.String("health_endpoint", "http://"+cfg.Server.Addr()+"/health"),
	)

	// 15. 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutdown signal received",
		zap.String("signal", sig.String()),
	)
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	} else {
		logger.Info("Server shutdown completed gracefully")
	}

	// 16. 关闭基础设施层 (按相反顺序)
	logger.Info("Closing infrastructure connections...")
	redis.Close()
	db.Close()
	if cos != nil {
		cos.Close()
	}
	logger.Info("All connections closed")

	logger.Info("Manus stopped")
}

// validateConfig 验证配置有效性
func validateConfig(cfg *config.Config) error {
	if cfg.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if cfg.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}
	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Server.Port == 0 {
		return fmt.Errorf("server.port is required")
	}
	return nil
}

// startupHealthCheck 启动时健康检查
func startupHealthCheck(db *infrastructure.Postgres, redis *infrastructure.Redis, sandbox *external.SandboxClient, sandboxCfg *config.SandboxConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.HealthCheck(ctx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}

	if err := redis.HealthCheck(ctx); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	// Sandbox 健康检查是可选的
	if sandbox != nil && sandboxCfg != nil && sandboxCfg.Address != "" {
		if err := sandbox.HealthCheck(ctx); err != nil {
			logger.Warn("Sandbox health check failed, continuing...", zap.Error(err))
		}
	}

	return nil
}
