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
			name:    "invalid log_level",
			mutate:  func(c *Config) { c.LogLevel = "trace" },
			wantSub: "LogLevel",
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

// validConfig 返回一个能通过校验的最小配置；测试通过 mutate 修改后断言失败。
func validConfig() *Config {
	return &Config{
		Env:      "development",
		LogLevel: "info",
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
			Host: "0.0.0.0",
			Port: 8080,
		},
	}
}
