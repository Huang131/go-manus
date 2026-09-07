package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

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

	// 1. 先用默认级别初始化日志（确保早期错误可记录）
	if err := logger.Init("info"); err != nil {
		fmt.Fprintf(os.Stderr, "init logger failed: %v\n", err)
		return ExitCodeLoggerError
	}
	defer logger.Sync()

	logger.Info("Starting Manus server...")

	// 2. 加载并验证配置
	cfg, err := config.LoadWithValidation(*configPath)
	if err != nil {
		logger.Error("load config failed", zap.Error(err))
		return ExitCodeConfigError
	}

	// 3. 根据配置调整日志级别
	logger.SetLevel(cfg.LogLevel)
	logger.Info("config loaded successfully", zap.String("log_level", cfg.LogLevel))

	// 4. 构建应用
	app, err := bootstrap.Build(cfg, bootstrap.DefaultOptions())
	if err != nil {
		logger.Error("bootstrap failed", zap.Error(err))
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
		zap.String("addr", cfg.Server.Addr()),
		zap.String("health_endpoint", "http://"+cfg.Server.Addr()+"/health"),
	)

	// 6. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serverErr:
		// ListenAndServe 主动关闭（ErrServerClosed）走 OK；
		// 其他错误视为服务异常退出。
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", zap.Error(err))
			return ExitCodeServerError
		}
		return ExitCodeOK
	case sig := <-quit:
		logger.Info("shutdown signal received", zap.String("signal", sig.String()))
		logger.Info("shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.Shutdown(ctx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				// 超时未能在限定时间内关停所有连接，按 ShutdownErr 处理。
				// 监控层应把这类错误计入 SLO 告警。
				logger.Error("graceful shutdown timed out",
					zap.Duration("timeout", 30*time.Second),
					zap.Error(err),
				)
			} else {
				logger.Error("server forced to shutdown", zap.Error(err))
			}
			return ExitCodeShutdownErr
		}

		logger.Info("server shutdown completed gracefully")
		return ExitCodeOK
	}
}

// serve 异步启动 HTTP 服务器。
//
// ListenAndServe 正常返回 ErrServerClosed 时说明 http.Server 已被 Shutdown，
// 这时向 errs 写入 nil，让 main 的 select 走 OK 路径。
func serve(server *http.Server, errs chan<- error) {
	go func() {
		logger.Info("server starting", zap.String("addr", server.Addr))

		serveErr := server.ListenAndServe()
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errs <- serveErr
			return
		}
		errs <- nil
	}()
}

// classifyBootstrapError 把 bootstrap 阶段的错误映射到对应的退出码。
func classifyBootstrapError(err error) int {
	if err == nil {
		return ExitCodeOK
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "health check failed"):
		return ExitCodeHealthCheckErr
	case strings.Contains(msg, "initialize "):
		return ExitCodeInitError
	default:
		return ExitCodeInitError
	}
}
