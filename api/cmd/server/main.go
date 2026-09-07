package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/internal/bootstrap"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// 退出码定义
const (
	ExitCodeOK             = 0  // 正常退出
	ExitCodeUsage          = 1  // 命令行参数错误（flag 解析失败等）
	ExitCodeConfigError    = 10 // 配置错误
	ExitCodeLoggerError    = 11 // 日志初始化失败
	ExitCodeInitError      = 12 // 初始化失败（数据库、Redis 等依赖）
	ExitCodeHealthCheckErr = 13 // 启动后健康检查失败
	ExitCodeServerError    = 14 // HTTP 服务器启动/运行失败
	ExitCodeShutdownErr    = 15 // 关闭失败
)

var (
	configPath = flag.String("config", "config.yaml", "config file path")
)

func main() {
	code := run()
	os.Exit(code)
}

// run 返回退出码
func run() int {
	flag.Parse()

	// 1. 先用info级别初始化日志（确保早期错误可记录）。
	// 之后再根据配置重建一次，支持文件切割等高级特性。
	if err := logger.Init(logger.LevelInfo); err != nil {
		fmt.Fprintf(os.Stderr, "init logger failed: %v\n", err)
		return ExitCodeLoggerError
	}
	defer logger.Sync()

	logger.Info("Starting Manus server...")

	// 2. 加载并验证配置
	cfg, err := config.LoadWithValidation(*configPath)
	if err != nil {
		logger.Error("load config failed", logger.Err(err))
		return ExitCodeConfigError
	}

	// 3. 根据配置重建 logger：可启用文件切割、压缩等高级特性。
	// cfg.Log 与 logger.Config 字段同构，直接类型转换。
	if err := logger.InitWithConfig(logger.Config(cfg.Log)); err != nil {
		fmt.Fprintf(os.Stderr, "reinit logger failed: %v\n", err)
		return ExitCodeLoggerError
	}
	logger.Info("config loaded successfully", logger.String("log_level", cfg.Log.Level))

	// 4. 构建应用
	app, err := bootstrap.Build(cfg, bootstrap.DefaultOptions())
	if err != nil {
		logger.Error("bootstrap failed", logger.Err(err))
		return classifyBootstrapError(err)
	}
	defer app.Close()

	if app.Server == nil {
		logger.Error("server not initialized")
		return ExitCodeInitError
	}

	// 5. 启动服务器
	serverErr := make(chan error, 1)
	serve(app.Server, serverErr)

	logger.Info("Manus started successfully",
		logger.String("addr", cfg.Server.Addr()),
		logger.String("health_endpoint", "http://"+cfg.Server.Addr()+"/health"),
	)

	// 6. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serverErr:
		// http.ErrServerClosed = 优雅关闭（正常退出）
		// 其他非 nil 错误 = 服务器异常
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", logger.Err(err))
			return ExitCodeServerError
		}
		return ExitCodeOK
	case sig := <-quit:
		logger.Info("shutdown signal received", logger.String("signal", sig.String()))
		logger.Info("shutting down server...")

		// ShutdownTimeoutSec 为 0 时兜底 30s。
		timeout := time.Duration(cfg.Server.ShutdownTimeoutSec)
		if timeout == 0 {
			timeout = 30
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := app.Shutdown(ctx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				// 超时未能在限定时间内关停所有连接，按 ShutdownErr 处理。
				// 监控层应把这类错误计入 SLO 告警。
				logger.Error("graceful shutdown timed out",
					logger.Dur("timeout", timeout),
					logger.Err(err),
				)
			} else {
				logger.Error("server forced to shutdown", logger.Err(err))
			}
			return ExitCodeShutdownErr
		}

		logger.Info("server shutdown completed gracefully")
		return ExitCodeOK
	}
}

// serve 异步启动 HTTP 服务器。
//
// ListenAndServe 返回值直接透传给 errs：
//   - http.ErrServerClosed = 优雅关闭（由 run() 判断为正常退出）
//   - 其他非 nil 错误 = 服务器启动/运行失败
//   - nil = 永不返回（服务器持续运行）
func serve(server *http.Server, errs chan<- error) {
	go func() {
		logger.Info("server starting", logger.String("addr", server.Addr))
		errs <- server.ListenAndServe()
	}()
}

// classifyBootstrapError 把 bootstrap 阶段的错误映射到对应的退出码。
//
// 优先通过 errors.Is 匹配 sentinel error；这样不依赖错误消息文本，
// 未来修改错误描述不会破坏分类逻辑。
func classifyBootstrapError(err error) int {
	if err == nil {
		return ExitCodeOK
	}
	switch {
	case errors.Is(err, bootstrap.ErrHealthCheckFailed):
		return ExitCodeHealthCheckErr
	case errors.Is(err, bootstrap.ErrInitialize):
		return ExitCodeInitError
	default:
		return ExitCodeInitError
	}
}
