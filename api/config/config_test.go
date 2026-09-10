package config

import (
	"strings"
	"testing"
)

func TestValidate_RequiredFields(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantSub string
	}{
		{
			name:    "missing database.host",
			mutate:  func(c *Config) { c.Database.Host = "" },
			wantSub: "Database.Host",
		},
		{
			name:    "invalid env",
			mutate:  func(c *Config) { c.Env = "staging" },
			wantSub: "Env",
		},
		{
			name:    "invalid server.port",
			mutate:  func(c *Config) { c.Server.Port = 99999 },
			wantSub: "Server.Port",
		},
		{
			name:    "negative max_open_conns",
			mutate:  func(c *Config) { c.Database.MaxOpenConns = -1 },
			wantSub: "MaxOpenConns",
		},
		{
			name:    "negative sandbox http timeout",
			mutate:  func(c *Config) { c.Sandbox.HTTPTimeout = -1 },
			wantSub: "Sandbox.HTTPTimeout",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validConfig()
			tc.mutate(c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestValidate_OK(t *testing.T) {
	c := validConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestConfigApplyDefaultsUsesDomainDefaults(t *testing.T) {
	cfg := validConfig()
	cfg.Server.ShutdownTimeoutSec = 0

	cfg.applyDefaults()

	if cfg.LLM.ToolCallTimeout != DefaultLLMToolCallTimeoutSec {
		t.Fatalf("LLM tool call timeout = %d, want %d", cfg.LLM.ToolCallTimeout, DefaultLLMToolCallTimeoutSec)
	}
	if cfg.FileCleanup.ExpiresAfter != DefaultFileCleanupExpiresAfter {
		t.Fatalf("file cleanup expiry = %q, want %q", cfg.FileCleanup.ExpiresAfter, DefaultFileCleanupExpiresAfter)
	}
	if cfg.Server.ShutdownTimeoutSec != DefaultServerShutdownTimeoutSec {
		t.Fatalf("shutdown timeout = %d, want %d", cfg.Server.ShutdownTimeoutSec, DefaultServerShutdownTimeoutSec)
	}
	if cfg.Database.MaxOpenConns != DefaultDatabaseMaxOpenConns {
		t.Fatalf("database max open conns = %d, want %d", cfg.Database.MaxOpenConns, DefaultDatabaseMaxOpenConns)
	}
	if cfg.Sandbox.HTTPTimeout != DefaultSandboxHTTPTimeoutSec {
		t.Fatalf("sandbox HTTP timeout = %d, want %d", cfg.Sandbox.HTTPTimeout, DefaultSandboxHTTPTimeoutSec)
	}
	if cfg.Search.HTTPTimeout != DefaultSearchHTTPTimeoutSec {
		t.Fatalf("search HTTP timeout = %d, want %d", cfg.Search.HTTPTimeout, DefaultSearchHTTPTimeoutSec)
	}
}

// validConfig 返回一个能通过校验的最小配置；测试通过 mutate 修改后断言失败。
func validConfig() *Config {
	return &Config{
		Env: EnvDevelopment,
		Log: LoggerConfig{
			Level: "info",
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "u",
			Database: "d",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Server: ServerConfig{
			Host:               "0.0.0.0",
			Port:               8080,
			ShutdownTimeoutSec: 30,
		},
	}
}
