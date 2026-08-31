package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// 创建一个临时配置文件
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configContent := `
app:
  env: "test"
  log_level: "debug"
  app_config_filepath: ""

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  database: "manus"
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime: 3600

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

cos:
  secret_id: "test_id"
  secret_key: "test_key"
  region: "ap-guangzhou"
  bucket: "test-bucket"

sandbox:
  address: "http://localhost:8081"
  image: "ubuntu:22.04"
  ttl: 3600

llm:
  base_url: "https://api.openai.com"
  api_key: ""
  model_name: "gpt-4"
  temperature: 0.7
  max_tokens: 4096

search:
  provider: "bing"
  bing_api_key: ""
  google_api_key: ""

server:
  host: "0.0.0.0"
  port: 8080
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 验证配置
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %s, want localhost", cfg.Database.Host)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d, want 6379", cfg.Redis.Port)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load() should return error for nonexistent file")
	}
}
