package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// 全局配置单例
var (
	cfg  *Config
	once sync.Once
)

// Config 应用程序配置
type Config struct {
	Env               string `mapstructure:"env"`                 // 环境: development/production
	LogLevel          string `mapstructure:"log_level"`           // 日志级别
	AppConfigFilepath string `mapstructure:"app_config_filepath"` // 应用配置文件路径

	// 数据库配置
	Database DatabaseConfig `mapstructure:"database"`

	// Redis 配置
	Redis RedisConfig `mapstructure:"redis"`

	// COS 配置
	COS COSConfig `mapstructure:"cos"`

	// Sandbox 配置
	Sandbox SandboxConfig `mapstructure:"sandbox"`

	// LLM 配置
	LLM LLMConfig `mapstructure:"llm"`

	// Search 配置
	Search SearchConfig `mapstructure:"search"`

	// MCP 配置
	MCP MCPConfig `mapstructure:"mcp"`

	// A2A 配置
	A2A A2AConfig `mapstructure:"a2a"`

	// HTTP 服务配置
	Server ServerConfig `mapstructure:"server"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// DSN 返回 PostgreSQL 连接字符串
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		d.User, d.Password, d.Host, d.Port, d.Database)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}

// Addr 返回 Redis 地址
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// COSConfig 对象存储配置（支持 AWS S3 / MinIO / 腾讯云 COS S3 兼容模式）
type COSConfig struct {
	Endpoint  string `mapstructure:"endpoint"`   // 对象存储端点（如 http://localhost:9000）
	SecretID  string `mapstructure:"secret_id"`  // 访问密钥 ID
	SecretKey string `mapstructure:"secret_key"` // 访问密钥密码
	Region    string `mapstructure:"region"`     // 区域
	Bucket    string `mapstructure:"bucket"`     // 存储桶名称
}

// SandboxConfig 沙箱配置
type SandboxConfig struct {
	Address    string `mapstructure:"address"`
	Image      string `mapstructure:"image"`
	NamePrefix string `mapstructure:"name_prefix"`
	TTLMinutes int    `mapstructure:"ttl_minutes"`
	Network    string `mapstructure:"network"`
	ChromeArgs string `mapstructure:"chrome_args"`
	HTTPSProxy string `mapstructure:"https_proxy"`
	HTTPProxy  string `mapstructure:"http_proxy"`
	NoProxy    string `mapstructure:"no_proxy"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// Addr 返回服务地址
func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// LLMConfig LLM 配置
type LLMConfig struct {
	BaseURL         string  `mapstructure:"base_url"`
	APIKey          string  `mapstructure:"api_key"`
	ModelName       string  `mapstructure:"model_name"`
	Temperature     float64 `mapstructure:"temperature"`
	MaxTokens       int     `mapstructure:"max_tokens"`
	ToolCallTimeout int     `mapstructure:"tool_call_timeout"` // tool calling 超时秒数，默认 15
}

// SearchConfig 搜索配置
type SearchConfig struct {
	Provider       string `mapstructure:"provider"` // "bing" or "google"
	BingAPIKey     string `mapstructure:"bing_api_key"`
	GoogleAPIKey   string `mapstructure:"google_api_key"`
	SearchEngineID string `mapstructure:"search_engine_id"` // Google Custom Search Engine ID
}

// MCPConfig MCP 配置
type MCPConfig struct {
	Servers []MCPServer `mapstructure:"servers"`
	Timeout int         `mapstructure:"timeout"`
}

// MCPServer MCP 服务器配置
type MCPServer struct {
	Name    string            `mapstructure:"name"`
	Command string            `mapstructure:"command"`
	Args    []string          `mapstructure:"args"`
	Env     map[string]string `mapstructure:"env"`
}

// A2AConfig A2A 配置
type A2AConfig struct {
	Agents  []A2AAgent `mapstructure:"agents"`
	Timeout int        `mapstructure:"timeout"`
}

// A2AAgent A2A Agent 配置
type A2AAgent struct {
	Name     string            `mapstructure:"name"`
	URL      string            `mapstructure:"url"`
	Metadata map[string]string `mapstructure:"metadata"`
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 环境变量映射
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// Get 返回全局配置
func Get() *Config {
	return cfg
}

// Init 初始化配置
func Init(configPath string) error {
	var err error
	once.Do(func() {
		cfg, err = Load(configPath)
	})
	return err
}
