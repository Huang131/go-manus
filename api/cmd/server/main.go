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

	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/internal/bootstrap"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

var (
	configPath = flag.String("config", "config.yaml", "config file path")
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server exited with error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := logger.Init(cfg.LogLevel); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	if err := validateConfig(cfg); err != nil {
		return err
	}

	app, err := bootstrap.Build(cfg, bootstrap.DefaultOptions())
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	defer app.Close()

	if app.Server == nil {
		return fmt.Errorf("server not initialized")
	}

	serverErr := make(chan error, 1)
	serve(app.Server, serverErr)

	logger.Info("Manus started successfully",
		zap.String("addr", cfg.Server.Addr()),
		zap.String("health_endpoint", "http://"+cfg.Server.Addr()+"/health"),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serverErr:
		return fmt.Errorf("serve HTTP server: %w", err)
	case sig := <-quit:
		logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
		logger.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			logger.Error("Server forced to shutdown", zap.Error(err))
			return err
		}
		logger.Info("Server shutdown completed gracefully")
		return nil
	}
}

func serve(server *http.Server, errs chan<- error) {
	go func() {
		logger.Info("Server starting",
			zap.String("addr", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errs <- err
			return
		}
		errs <- nil
	}()
}

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
