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
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/router"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/Huang131/go-manus/api/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================================
// 生命周期管理
// ============================================================================

// lifecycleManager 管理应用生命周期钩子。
//
// 启动钩子（startHooks）：按注册顺序执行，用于启动后台任务
// 关闭钩子（stopHooks）：逆序执行，用于优雅释放资源
//
// 设计原则：
//   - 启动顺序：先启动的先执行（如基础设施 → 服务 → Agent）
//   - 关闭顺序：后启动的后关闭（如 Agent → 服务 → 基础设施）
//   - 这样可以保证依赖关系：关闭时先断开依赖，再释放被依赖的资源
type lifecycleManager struct {
	startHooks []func() error // 启动钩子列表
	stopHooks  []func()       // 关闭钩子列表
	mu         sync.Mutex     // 保护钩子列表的并发安全
}

// AppendStartHook 注册启动钩子,启动时按注册顺序执行。
func (l *lifecycleManager) AppendStartHook(fn func() error) {
	if fn == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.startHooks = append(l.startHooks, fn)
}

// AppendStopHook 注册关闭钩子,关闭时按注册顺序逆序执行（后注册先关闭）。
func (l *lifecycleManager) AppendStopHook(fn func()) {
	if fn == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopHooks = append(l.stopHooks, fn)
}

// Start 执行所有启动钩子，按注册顺序
func (l *lifecycleManager) Start() error {
	l.mu.Lock()
	hooks := l.startHooks
	l.mu.Unlock()

	for _, fn := range hooks {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

// Stop 执行所有关闭钩子，按注册顺序逆序（后注册的先关闭）
func (l *lifecycleManager) Stop() {
	l.mu.Lock()
	hooks := l.stopHooks
	l.mu.Unlock()

	for i := len(hooks) - 1; i >= 0; i-- {
		if hooks[i] != nil {
			hooks[i]()
		}
	}
}

// newLifecycleManager 创建生命周期管理器
func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{}
}

// Options 控制 bootstrap 的装配行为。
type Options struct {
	EnablePostgres    bool // PostgreSQL 数据库（必须）
	EnableRedis       bool // Redis 缓存（必须）
	EnableStorage     bool // OSS 对象存储（可选）
	EnableLLM         bool // LLM 模型服务（可选）
	EnableSandbox     bool // 沙箱执行环境（可选）
	EnableBrowser     bool // 浏览器自动化（可选，依赖 Sandbox）
	EnableSearch      bool // 搜索引擎（可选）
	EnableAgent       bool // Agent 服务（必须，依赖 LLM 和 MQ）
	EnableRoutes      bool // HTTP 路由（必须）
	EnableHealthCheck bool // 启动时健康检查（可选）
}

// DefaultOptions 返回生产环境默认装配选项，全部启用。
//
// 使用场景：main 函数默认使用此选项
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

// ============================================================================
// 依赖注入工厂
// ============================================================================

// Factories 允许测试或不同启动模式替换底层客户端构造器。
//
// 设计目的：
//   - 单元测试：注入 mock 组件，无需真实连接数据库/Redis
//   - 集成测试：注入测试环境的特殊构造器
//   - 扩展性：允许自定义组件创建逻辑
//
// 使用方式：
//
//	factories := Factories{
//	    NewPostgres: func(cfg *config.DatabaseConfig) (*infrastructure.Postgres, error) {
//	        return mockPostgres, nil
//	    },
//	}
//	app, _ := BuildWithFactories(cfg, opts, factories)
type Factories struct {
	NewPostgres             func(*config.DatabaseConfig) (*infrastructure.Postgres, error) // PostgreSQL 构造器
	NewRedis                func(*config.RedisConfig) (*infrastructure.Redis, error)       // Redis 构造器
	NewOSS                  func(*config.ObjectStorageConfig) (*infrastructure.OSS, error) // OSS 构造器
	NewFileCleanupScheduler func(
		cleanupService service.FileCleanupService, // 清理服务
		expireDuration string, // 过期时间，如 "24h", "7d"
		batchSize int, // 每批清理数量
		interval time.Duration, // 执行间隔
	) service.SchedulerRunner // 文件清理调度器构造器
}

// repositories 聚合所有仓库（Repository）实例。
//
// Repository 模式：将数据访问逻辑封装在独立的仓库层，
// 便于单元测试时替换为 mock 实现。
type repositories struct {
	appConfig repository.AppConfigRepository // 应用配置仓库
	llmModel  repository.LLMModelRepository  // LLM 模型仓库
	session   repository.SessionRepository   // 会话仓库
	file      repository.FileRepository      // 文件仓库
}

// newRepositories 根据数据库连接创建所有仓库实例。
//
// 参数：
//   - db: PostgreSQL 数据库连接，不能为 nil
//
// 返回：初始化好的 repositories 实例
func newRepositories(db *infrastructure.Postgres) repositories {
	return repositories{
		appConfig: repository.NewAppConfigRepository(db),
		llmModel:  repository.NewLLMModelRepository(db),
		session:   repository.NewSessionRepository(db),
		file:      repository.NewFileRepository(db),
	}
}

// defaultFactories 返回默认的工厂函数集合。
//
// 这些函数创建真实的生产级组件，用于：
//   - Build() 函数使用
//   - normalizeFactories() 作为默认值填充
func defaultFactories() Factories {
	return Factories{
		NewPostgres: infrastructure.NewPostgres,
		NewRedis:    infrastructure.NewRedis,
		NewOSS:      infrastructure.NewOSS,
		NewFileCleanupScheduler: func(cleanupService service.FileCleanupService, expireDuration string, batchSize int, interval time.Duration) service.SchedulerRunner {
			return service.NewFileCleanupScheduler(cleanupService, expireDuration, batchSize, interval)
		},
	}
}

// normalizeFactories 填充未设置的工厂函数为默认值。
//
// 逻辑：如果调用方只设置了部分工厂函数，未设置的使用默认值填充
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
	if factories.NewFileCleanupScheduler == nil {
		factories.NewFileCleanupScheduler = defaults.NewFileCleanupScheduler
	}
	return factories
}

// ============================================================================
// 应用容器
// ============================================================================

// App 是所有组件的容器，提供统一的启动和关闭接口。
//
// 组件分层（自底向上）：
//   - 基础设施层：Postgres、Redis、OSS
//   - 仓库层：repos（私有）
//   - 服务层：SessionService、FileService、AgentService 等
//   - 外部客户端：LLM、Browser、Search、MessageQueue
//   - HTTP 层：Engine（路由）、Server（HTTP 服务器）
//
// 线程安全：
//   - Close() 和 Start() 方法使用 sync.Once 保证幂等
//   - 初始化过程（非并发调用）无需额外同步
type App struct {
	Config *config.Config // 全局配置（从配置文件加载）
	Server *http.Server   // HTTP 服务器（包含路由中间件）
	Engine *gin.Engine    // Gin 引擎（路由配置）

	// ===== 基础设施层 =====
	// 基础组件，提供数据持久化和缓存能力
	Postgres *infrastructure.Postgres // PostgreSQL 数据库连接
	Redis    *infrastructure.Redis    // Redis 缓存连接
	OSS      *infrastructure.OSS      // 对象存储服务（可选）

	// ===== 服务层 =====
	// 业务逻辑层，组合多个基础组件实现业务功能
	Sandbox        *external.SandboxClient  // 代码沙箱执行器
	AgentService   *agent.AgentService      // AI Agent 核心服务
	SessionService service.SessionService   // 会话管理服务
	FileService    service.FileService      // 文件管理服务
	StatusService  service.StatusService    // 状态查询服务
	AppConfigSvc   service.AppConfigService // 应用配置服务
	LLMModelSvc    service.LLMModelService  // LLM 模型管理服务

	// ===== 私有成员 =====
	repos     repositories      // 数据仓库层（不暴露给外部）
	lifecycle *lifecycleManager // 生命周期管理器
	closeOnce sync.Once         // 保证 Close() 只执行一次
	startOnce sync.Once         // 保证 Start() 只执行一次
	startErr  error             // 记录 Start() 的错误（避免闭包捕获问题）
}

// ===== 生命周期钩子 =====

// stopHook 注册关闭钩子，在应用关闭时逆序执行。
//
// 使用场景：
//   - 初始化组件时注册其关闭逻辑
//   - 确保所有资源被正确释放
func (a *App) stopHook(fn func()) {
	if a.lifecycle != nil {
		a.lifecycle.AppendStopHook(fn)
	}
}

// startHook 注册启动钩子，在应用启动时按顺序执行。
//
// 使用场景：
//   - 启动后台定时任务
//   - 初始化非关键的异步组件
func (a *App) startHook(fn func() error) {
	if a.lifecycle != nil {
		a.lifecycle.AppendStartHook(fn)
	}
}

// Close 关闭应用资源。关闭动作只执行一次，便于同时支持 defer 和显式关闭。
func (a *App) Close() {
	if a == nil {
		return
	}

	a.closeOnce.Do(func() {
		if a.lifecycle != nil {
			a.lifecycle.Stop()
		}
	})
}

// Start 启动应用注册的后台任务。
// 该方法幂等，便于 Build 完成后直接调用或由测试重复触发。
func (a *App) Start() error {
	if a == nil {
		return nil
	}

	a.startOnce.Do(func() {
		if a.lifecycle != nil {
			a.startErr = a.lifecycle.Start()
		}
	})
	return a.startErr
}

// Shutdown 优雅关闭应用，先停止 HTTP 服务，再释放资源。
//
// 关闭顺序：
//  1. Server.Shutdown(ctx) - 停止接受新请求，等待现有请求处理完成
//  2. app.Close() - 释放所有依赖资源
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
			// 超时场景：把上下文错误包一层，方便调用方 errors.Is 判定
			shutdownErr = fmt.Errorf("graceful shutdown timed out: %w", shutdownErr)
		}
	}
	a.Close()
	return shutdownErr
}

// ============================================================================
// 初始化函数
// ============================================================================

// initInfrastructure 初始化基础设施层组件。
//
// 初始化顺序：Postgres → Redis → OSS
// 关闭顺序：OSS → Redis → Postgres（通过 stopHook 逆序注册实现）
//
// 组件说明：
//   - Postgres：必须组件，初始化失败直接返回错误
//   - Redis：必须组件，初始化失败直接返回错误
//   - OSS：可选组件，初始化失败只记录警告日志，继续启动
//
// 参数：
//   - cfg: 全局配置
//   - opts: 装配选项，控制启用哪些组件
//   - factories: 组件构造器（支持注入 mock）
func (a *App) initInfrastructure(cfg *config.Config, opts Options, factories Factories) error {
	var err error

	// ---- PostgreSQL ----
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
		a.stopHook(func() {
			a.Postgres.Close()
		})
	}

	// ---- Redis ----
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
		a.stopHook(func() {
			_ = a.Redis.Close()
		})
	}

	// ---- OSS（可选）----
	if opts.EnableStorage {
		oss, storageErr := factories.NewOSS(&cfg.OSS)
		if storageErr != nil {
			// OSS 失败不影响启动，只记录警告
			if oss != nil {
				_ = oss.Close()
			}
			logger.Warn("storage init failed, continuing without storage", logger.Err(storageErr))
		} else {
			a.OSS = oss
			if a.OSS == nil {
				return fmt.Errorf("storage: %w: factory returned nil", ErrInitialize)
			}
			a.stopHook(func() {
				_ = a.OSS.Close()
			})
		}
	}

	return nil
}

// initServices 初始化服务层组件。
// 服务层组合基础设施层和仓库层，实现业务逻辑。
func (a *App) initServices(cfg *config.Config) {
	// 创建仓库层
	a.repos = newRepositories(a.Postgres)

	// 创建服务层
	a.SessionService = service.NewSessionServiceWithSandbox(a.repos.session, a.repos.file, cfg.Sandbox.Address)
	a.FileService = service.NewFileService(a.repos.file, a.OSS)
	a.StatusService = service.NewStatusService(a.Postgres, a.Redis, a.OSS)
	a.AppConfigSvc = service.NewAppConfigService(a.repos.appConfig)
	a.LLMModelSvc = service.NewLLMModelService(a.repos.llmModel)
}

// initLLM 初始化 LLM 路由器。
//
// LLM 路由器（RoutedLLM）负责：
//   - 模型选择：从数据库动态获取模型配置
//   - 健康追踪：记录各模型的响应延迟和失败次数
//   - 智能 fallback：主模型失败时自动切换备选模型
//   - 持久化：健康状态写入数据库，重启后不丢失
//
// 模型选择优先级：
//  1. 请求上下文中的 model_id（用户指定）
//  2. 数据库中的默认模型
//  3. 数据库中第一个启用的模型
//  4. 配置文件的 fallback 模型
//
// 返回值：
//   - 初始化好的 LLM 路由器，如果未启用或未配置则返回 nil
func (a *App) initLLM(cfg *config.Config, opts Options) external.LLM {
	// 前置检查：LLM 未启用或未配置 BaseURL
	if !opts.EnableLLM || cfg.LLM.BaseURL == "" {
		return nil
	}

	// 首次部署：数据库无任何模型时，把 env/配置文件里的模型落库为初始默认，
	// 让"界面管理模型"从第一天就有数据可用；同时路由器的 env fallback 仍然保留。
	a.seedDefaultModelFromEnv(cfg)

	// 创建 fallback 配置（基于配置文件，作为最后的保底）
	fallbackLLMCfg := &external.LLMRuntimeConfig{
		Profile:         llmcore.ModelProfile{Protocol: llmcore.ProtocolOpenAICompat},
		BaseURL:         cfg.LLM.BaseURL,
		APIKey:          cfg.LLM.APIKey,
		ModelName:       cfg.LLM.ModelName,
		Temperature:     cfg.LLM.Temperature,
		MaxTokens:       cfg.LLM.MaxTokens,
		ToolCallTimeout: cfg.LLM.ToolCallTimeout,
	}

	// 创建路由器
	routed := external.NewRoutedLLMFromSingleProvider(
		// provider 函数：从数据库动态获取模型配置
		func(ctx context.Context) (*external.LLMRuntimeConfig, error) {
			if a.Postgres != nil && a.repos.llmModel != nil {
				// 优先级 1：请求上下文指定的 model_id。
				// 用户明确选定的模型粘性路由：不存在/被禁用时显式报错，
				// 不允许静默降级到默认模型（对齐 Cursor 的"选定不切换"语义）。
				if mid := external.ModelIDFromContext(ctx); mid != "" {
					chosen, err := a.repos.llmModel.GetByID(ctx, mid)
					if err != nil {
						return nil, fmt.Errorf("%w: %s: %v", external.ErrModelNotAvailable, mid, err)
					}
					if chosen == nil || !chosen.IsEnabled {
						return nil, fmt.Errorf("%w: %s", external.ErrModelNotAvailable, mid)
					}
					return external.BuildRuntimeConfigFromModel(chosen, cfg.LLM.ToolCallTimeout), nil
				}
				// 优先级 2：默认模型
				if def, err := a.repos.llmModel.GetDefault(ctx); err != nil {
					logger.Warn("failed to get default model", logger.Err(err))
				} else if def != nil && def.IsEnabled {
					return external.BuildRuntimeConfigFromModel(def, cfg.LLM.ToolCallTimeout), nil
				}
				// 优先级 3：第一个启用的模型
				if first, err := a.repos.llmModel.GetFirstEnabled(ctx); err != nil {
					logger.Warn("failed to get first enabled model", logger.Err(err))
				} else if first != nil {
					return external.BuildRuntimeConfigFromModel(first, cfg.LLM.ToolCallTimeout), nil
				}
			}
			return nil, nil
		}, fallbackLLMCfg, nil)

	// 设置健康状态持久化（写入数据库）
	if a.Postgres != nil {
		routed.SetHealthStore(a.repos.llmModel)
	}
	return routed
}

// seedDefaultModelFromEnv 在模型表为空（首次部署）时，把 env/配置文件中的
// LLM 配置落库为初始默认模型。已存在任何模型时为 no-op，重复启动安全。
func (a *App) seedDefaultModelFromEnv(cfg *config.Config) {
	if a.Postgres == nil || a.repos.llmModel == nil || cfg.LLM.ModelName == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := a.repos.llmModel.List(ctx)
	if err != nil {
		logger.Warn("检查模型表失败，跳过 env 模型种子落库", logger.Err(err))
		return
	}
	if len(models) > 0 {
		return
	}

	protocol := external.ProtocolFromProvider("", cfg.LLM.ModelName, cfg.LLM.BaseURL)
	provider := "openai"
	if protocol == llmcore.ProtocolAnthropic {
		provider = "anthropic"
	}
	temperature := cfg.LLM.Temperature
	maxTokens := cfg.LLM.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	m := &model.LLMModel{
		ID:          uuid.New().String(),
		Name:        cfg.LLM.ModelName,
		Provider:    provider,
		BaseURL:     cfg.LLM.BaseURL,
		APIKey:      cfg.LLM.APIKey,
		ModelName:   cfg.LLM.ModelName,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Tags:        []string{},
		IsDefault:   true,
		IsEnabled:   true,
		Capabilities: model.MergeDefaultCapabilities(model.ModelCapabilities{
			SupportsVision:    false,
			SupportsReasoning: false,
		}),
	}
	if err := a.repos.llmModel.Create(ctx, m); err != nil {
		logger.Warn("env 模型种子落库失败", logger.Err(err))
		return
	}
	logger.Info("已从 env 配置创建初始默认模型",
		logger.String("model_name", cfg.LLM.ModelName),
		logger.String("provider", provider))
}

// externalClients 收拢 initExternalClients 产出的外部客户端，
// 避免 6 值返回与 initAgent 的长参数列表（Long Parameter List）。
type externalClients struct {
	llm       external.LLM
	browser   external.Browser
	search    external.SearchEngine
	mq        external.TaskMessageQueue
	mcpConfig *agent.MCPConfig
	a2aConfig *agent.A2AConfig
}

// initExternalClients 初始化外部客户端组件。
//
// 外部客户端包括：
//   - LLM：语言模型（通过 initLLM 初始化）
//   - Sandbox：代码沙箱执行器
//   - Browser：浏览器自动化（依赖 Sandbox）
//   - Search：搜索引擎（可选）
//   - MessageQueue：消息队列（使用 Redis Streams 实现）
//   - MCP：Model Context Protocol 配置
//   - A2A：Agent-to-Agent 通信配置
//
// 这些组件被传递给 Agent 服务，供 Agent 调用外部能力。
func (a *App) initExternalClients(cfg *config.Config, opts Options) *externalClients {
	clients := &externalClients{}

	// LLM 路由器
	clients.llm = a.initLLM(cfg, opts)

	// Sandbox 沙箱
	if opts.EnableSandbox {
		a.Sandbox = external.NewSandboxClient(&cfg.Sandbox)
	}

	// Browser 浏览器（依赖 Sandbox）
	if opts.EnableBrowser && a.Sandbox != nil && cfg.Sandbox.Address != "" {
		clients.browser = external.NewBrowserClient(a.Sandbox)
	}

	// Search 搜索引擎（可选，需要配置 API Key）
	if opts.EnableSearch && (cfg.Search.BingAPIKey != "" || cfg.Search.GoogleAPIKey != "") {
		clients.search = external.NewSearchEngine(&external.SearchConfig{
			Provider:       cfg.Search.Provider,
			BingAPIKey:     cfg.Search.BingAPIKey,
			GoogleAPIKey:   cfg.Search.GoogleAPIKey,
			SearchEngineID: cfg.Search.SearchEngineID,
			HTTPTimeout:    cfg.Search.HTTPTimeout,
		})
	}

	// MessageQueue 消息队列（使用 Redis Streams）
	if a.Redis != nil {
		clients.mq = external.NewRedisStreamMessageQueue(a.Redis.Client)
	}

	clients.mcpConfig = newMCPConfig(cfg)
	clients.a2aConfig = newA2AConfig(cfg)
	return clients
}

// newMCPConfig 从配置创建 MCP（Model Context Protocol）配置。
//
// MCP 是一种让 LLM 与外部工具交互的协议标准。
// 此函数从配置文件读取 MCP 服务器列表。
//
// 如果没有配置任何 MCP 服务器，返回 nil。
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

// newA2AConfig 从配置创建 A2A（Agent-to-Agent）通信配置。
//
// A2A 允许不同 Agent 之间直接通信，交换信息和任务。
// 此函数从配置文件读取 Agent 列表。
//
// 如果没有配置任何 Agent，返回 nil。
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

// initAgent 初始化 Agent 核心服务。
//
// Agent 服务是整个应用的核心，负责：
//   - 接收用户请求，解析意图
//   - 规划执行步骤，选择工具
//   - 调用 LLM 生成响应
//   - 管理会话状态
//
// 依赖检查：
//   - 必须有 PostgreSQL（存储会话）
//   - 必须有 MessageQueue（异步任务队列）
//   - 必须有 LLM（核心能力）
func (a *App) initAgent(opts Options, clients *externalClients) error {
	if !opts.EnableAgent {
		return nil
	}
	// 依赖检查
	if a.Postgres == nil || clients.mq == nil {
		return ErrAgentRequiresDependencies
	}
	if clients.llm == nil {
		return ErrAgentRequiresLLM
	}

	// 创建 Agent 服务
	a.AgentService = agent.NewAgentService(
		context.Background(),
		a.repos.session, a.repos.file, a.repos.appConfig, a.repos.llmModel, clients.llm, a.Sandbox,
		agent.DefaultAgentConfig(), clients.mcpConfig, clients.a2aConfig, clients.browser, clients.search, clients.mq, a.OSS,
	)
	a.stopHook(func() {
		if a.AgentService != nil {
			a.AgentService.Shutdown()
		}
	})
	return nil
}

// initSchedulers 初始化定时任务调度器。
//
// 当前只包含文件清理任务：
//   - 定期清理过期的临时文件
//   - 节省存储成本
//   - 分批处理，避免影响服务性能
//
// 前置条件：
//   - 必须启用 PostgreSQL（查询过期文件）
//   - 必须启用 Storage（删除 OSS 文件）
//   - 基础设施必须已初始化
//
// 注意：如果前置条件不满足，静默跳过，不返回错误
func (a *App) initSchedulers(cfg *config.Config, opts Options, factories Factories) error {
	// 前置条件检查
	if !opts.EnablePostgres || !opts.EnableStorage {
		return nil
	}
	if a.Postgres == nil || a.OSS == nil || !a.OSS.IsReady() {
		return nil
	}

	// 创建文件清理服务
	fileCleanupService := service.NewFileCleanupService(a.repos.file, a.OSS)

	// 创建调度器
	cleanupScheduler := factories.NewFileCleanupScheduler(
		fileCleanupService,                                  // cleanupService: 清理服务
		cfg.FileCleanup.ExpiresAfter,                        // expireDuration: 过期时间，如 "24h"
		cfg.FileCleanup.BatchSize,                           // batchSize: 每批清理数量
		time.Duration(cfg.FileCleanup.Interval)*time.Second, // interval: 执行间隔
	)
	if cleanupScheduler == nil {
		return fmt.Errorf("file cleanup scheduler: %w", ErrInitialize)
	}

	a.startHook(func() error {
		cleanupScheduler.Start()
		return nil
	})
	a.stopHook(func() {
		cleanupScheduler.Stop()
	})
	return nil
}

// initRoutes 初始化 HTTP 路由和服务器。
//
// 路由配置：
//   - 中间件：Recovery、RequestID、Logger、CORS
//   - 受信代理：默认只信任本机回环地址（安全加固）
//   - HTTP 超时：可配置读/写/空闲超时
//
// 注册的路由处理器：
//   - Session：会话管理
//   - File：文件上传下载
//   - Status：健康状态
//   - AppConfig：应用配置
//   - LLMModel：模型管理
func (a *App) initRoutes(cfg *config.Config, opts Options) {
	if !opts.EnableRoutes {
		return
	}

	// 创建 Gin 引擎
	engine := gin.New()

	// 配置受信反向代理（安全加固）
	// 默认只信任本机回环地址，防止 X-Forwarded-For 伪造
	trustedProxies := cfg.Server.TrustedProxies
	if len(trustedProxies) == 0 {
		trustedProxies = defaultTrustedProxies()
		logger.Warn("trusted_proxies not configured, falling back to loopback addresses")
	}
	if err := engine.SetTrustedProxies(trustedProxies); err != nil {
		logger.Warn("invalid trusted_proxies config, falling back to loopback",
			logger.Strings("configured", trustedProxies),
			logger.Err(err))
		_ = engine.SetTrustedProxies(defaultTrustedProxies())
	}

	// 注册中间件
	engine.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery(), middleware.CORS())

	// 创建并注册路由处理器
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

	// 创建 HTTP 服务器
	// 写超时设为 0，支持流式响应
	a.Server = &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      engine,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeoutSec) * time.Second,
	}
}

// healthCheck 执行启动时健康检查。
//
// 检查项：
//   - Postgres：必须，失败返回错误
//   - Redis：必须，失败返回错误
//   - Sandbox：可选，失败只记录警告
//
// 使用场景：
//   - 在服务真正启动前验证依赖可用性
//   - 提前发现配置问题，避免请求进来后才发现
//
// 注意：健康检查失败会阻止应用启动，但不会清理已分配的资源
// 调用方需要在 Build 失败时调用 app.Close() 清理资源
func (a *App) healthCheck(opts Options, cfg *config.Config) error {
	if !opts.EnableHealthCheck {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
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

// ============================================================================
// 应用构建入口
// ============================================================================

// Build 构建完整应用，使用默认工厂函数。
func Build(cfg *config.Config, opts Options) (*App, error) {
	return BuildWithFactories(cfg, opts, defaultFactories())
}

// BuildWithFactories 构建完整应用，允许注入自定义工厂函数。
//
// 初始化顺序（保证依赖关系）：
//  1. initInfrastructure - 基础设施层（数据库、缓存、存储）
//  2. initServices       - 服务层（依赖基础设施）
//  3. initExternalClients - 外部客户端（依赖服务层）
//  4. initAgent          - Agent 服务（依赖外部客户端）
//  5. initSchedulers     - 定时任务（依赖所有组件）
//  6. initRoutes         - HTTP 路由（依赖服务层）
//  7. healthCheck        - 健康检查
//  8. Start              - 启动后台任务
//
// 错误处理：
//   - 任何初始化步骤失败，都会调用 app.Close() 清理已分配资源
//   - 错误使用 fmt.Errorf 包装，保留错误链（%w）
//
// 参数：
//   - cfg：全局配置，不能为 nil
//   - opts：装配选项
//   - factories：组件工厂函数（支持注入 mock）
func BuildWithFactories(cfg *config.Config, opts Options, factories Factories) (*App, error) {
	// 参数校验
	if cfg == nil {
		return nil, ErrConfigNil
	}
	factories = normalizeFactories(factories)

	// 生产环境设置 Gin 为 Release 模式
	if cfg.Env == config.EnvProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 App 实例
	app := &App{
		Config:    cfg,
		lifecycle: newLifecycleManager(),
	}

	// 初始化基础设施层
	if err := app.initInfrastructure(cfg, opts, factories); err != nil {
		app.Close()
		return nil, err
	}

	// 初始化服务层
	app.initServices(cfg)

	// 初始化外部客户端
	clients := app.initExternalClients(cfg, opts)

	// 初始化 Agent 服务
	if err := app.initAgent(opts, clients); err != nil {
		app.Close()
		return nil, err
	}

	// 初始化定时任务
	if err := app.initSchedulers(cfg, opts, factories); err != nil {
		app.Close()
		return nil, err
	}

	// 初始化 HTTP 路由
	app.initRoutes(cfg, opts)

	// 健康检查
	if err := app.healthCheck(opts, cfg); err != nil {
		app.Close()
		return nil, err
	}

	// 启动后台任务
	if err := app.Start(); err != nil {
		app.Close()
		return nil, err
	}

	return app, nil
}
