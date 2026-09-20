//go:build integration

package testsupport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `
env: test
log:
  level: debug
database:
  host: localhost
  port: 5432
  user: postgres
  database: manus_test
redis:
  host: localhost
  port: 6379
  db: 1
oss:
  provider: minio
  endpoint: http://localhost:9000
  bucket: go-manus-test-files
server:
  host: 127.0.0.1
  port: 8080
`

func TestLoadIntegrationEnvAppliesOverrides(t *testing.T) {
	clearIntegrationOverrides(t)
	configPath := writeConfig(t, validConfig)
	t.Setenv(envPostgresDSN, "postgres://tester:secret@db.example:5544/custom_test?sslmode=disable")
	t.Setenv(envRedisAddr, "redis.example:6380")
	t.Setenv(envRedisDB, "3")
	t.Setenv(envS3Endpoint, "http://storage.example:9000")
	t.Setenv(envS3Bucket, "custom-test-files")

	env, err := LoadIntegrationEnv(configPath)
	if err != nil {
		t.Fatalf("LoadIntegrationEnv() error = %v", err)
	}

	if got := env.Config.Database.DSN(); !strings.Contains(got, "tester:secret@db.example:5544/custom_test") {
		t.Fatalf("PostgreSQL override not applied: %s", got)
	}
	if got := env.Config.Redis.Addr(); got != "redis.example:6380" || env.Config.Redis.DB != 3 {
		t.Fatalf("Redis override = %s db=%d", got, env.Config.Redis.DB)
	}
	if env.Config.OSS.Endpoint != "http://storage.example:9000" || env.Config.OSS.Bucket != "custom-test-files" {
		t.Fatalf("storage override = %s/%s", env.Config.OSS.Endpoint, env.Config.OSS.Bucket)
	}
}

func TestLoadIntegrationEnvRejectsUnsafeResources(t *testing.T) {
	clearIntegrationOverrides(t)
	tests := []struct {
		name       string
		envKey     string
		envValue   string
		wantErrSub string
	}{
		{name: "production database", envKey: envPostgresDSN, envValue: "postgres://postgres@localhost/manus", wantErrSub: "non-test PostgreSQL"},
		{name: "default Redis database", envKey: envRedisDB, envValue: "0", wantErrSub: "Redis DB 0"},
		{name: "negative Redis database", envKey: envRedisDB, envValue: "-1", wantErrSub: "Redis DB -1"},
		{name: "production bucket", envKey: envS3Bucket, envValue: "go-manus-files", wantErrSub: "non-test object storage"},
		{name: "misleading database name", envKey: envPostgresDSN, envValue: "postgres://postgres@localhost/contest", wantErrSub: "non-test PostgreSQL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := writeConfig(t, validConfig)
			t.Setenv(tt.envKey, tt.envValue)

			_, err := LoadIntegrationEnv(configPath)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("LoadIntegrationEnv() error = %v, want containing %q", err, tt.wantErrSub)
			}
		})
	}
}

func TestLoadIntegrationEnvRejectsInvalidOverrides(t *testing.T) {
	clearIntegrationOverrides(t)
	tests := []struct {
		name       string
		envKey     string
		envValue   string
		wantErrSub string
	}{
		{name: "PostgreSQL scheme", envKey: envPostgresDSN, envValue: "mysql://localhost/manus_test", wantErrSub: "postgres scheme"},
		{name: "Redis address", envKey: envRedisAddr, envValue: "localhost", wantErrSub: envRedisAddr},
		{name: "Redis port", envKey: envRedisAddr, envValue: "localhost:70000", wantErrSub: "between 1 and 65535"},
		{name: "Redis database", envKey: envRedisDB, envValue: "not-a-number", wantErrSub: envRedisDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := writeConfig(t, validConfig)
			t.Setenv(tt.envKey, tt.envValue)

			_, err := LoadIntegrationEnv(configPath)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("LoadIntegrationEnv() error = %v, want containing %q", err, tt.wantErrSub)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func clearIntegrationOverrides(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		envPostgresDSN,
		envRedisAddr,
		envRedisDB,
		envS3Endpoint,
		envS3Bucket,
	} {
		previous, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, previous)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}
