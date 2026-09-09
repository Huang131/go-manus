package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// 环境常量
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Config 应用程序配置
type Config struct {
	Env               string `mapstructure:"env"                 validate:"required,oneof=development production"` // 环境: development/production
	AppConfigFilepath string `mapstructure:"app_config_filepath"`                                                  // 应用配置文件路径

	// 日志配置
	Log LoggerConfig `mapstructure:"log"`

	// 数据库配置
	Database DatabaseConfig `mapstructure:"database" validate:"required"`

	// Redis 配置
	Redis RedisConfig `mapstructure:"redis" validate:"required"`

	// ObjectStorage 配置（支持 AWS S3 / 腾讯云 COS / 火山云 TOS / MinIO）
	OSS ObjectStorageConfig `mapstructure:"oss"`

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
	Server ServerConfig `mapstructure:"server" validate:"required"`

	// 文件清理配置
	FileCleanup FileCleanupConfig `mapstructure:"file_cleanup"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level"       validate:"required,oneof=debug info warn error"`
	Filename   string `mapstructure:"filename"`                     // 日志文件路径，为空则只输出到 stdout
	MaxSize    int    `mapstructure:"max_size"    validate:"gte=0"` // 单个日志文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups" validate:"gte=0"` // 保留的旧日志文件数量
	MaxAge     int    `mapstructure:"max_age"     validate:"gte=0"` // 旧日志文件保留天数
	Compress   bool   `mapstructure:"compress"`                     // 是否压缩旧日志
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"          validate:"required"`
	Port         int    `mapstructure:"port"          validate:"required,gt=0,lte=65535"`
	User         string `mapstructure:"user"          validate:"required"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"      validate:"required"`
	MaxOpenConns int    `mapstructure:"max_open_conns" validate:"gte=0"`
	MaxIdleConns int    `mapstructure:"max_idle_conns" validate:"gte=0"`
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

// ObjectStorageConfig 对象存储配置。
// 支持 AWS S3、腾讯云 COS、火山云 TOS、MinIO 等 S3 兼容存储。
// Provider 可选：aws / tencent / volcengine / minio / custom
type ObjectStorageConfig struct {
	Provider  string `mapstructure:"provider"`   // 存储服务商：aws / tencent / volcengine / minio / custom
	Endpoint  string `mapstructure:"endpoint"`   // 对象存储端点，留空则根据 Provider 自动推断
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
	Host               string   `mapstructure:"host"               validate:"required"`
	Port               int      `mapstructure:"port"               validate:"required,gt=0,lte=65535"`
	TrustedProxies     []string `mapstructure:"trusted_proxies"` // 受信反向代理 CIDR/IP，nginx/网关地址
	ReadTimeoutSec     int      `mapstructure:"read_timeout_sec"   validate:"gte=0"`
	WriteTimeoutSec    int      `mapstructure:"write_timeout_sec"  validate:"gte=0"`
	IdleTimeoutSec     int      `mapstructure:"idle_timeout_sec"   validate:"gte=0"`
	ShutdownTimeoutSec int      `mapstructure:"shutdown_timeout_sec" validate:"gte=0"` // 优雅关闭超时秒数，0 兜底 30s
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
	ToolCallTimeout int     `mapstructure:"tool_call_timeout"` // tool calling 超时秒数，兜底 15s
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

// FileCleanupConfig 文件清理配置
// 清理策略：只清理孤立文件（无关联会话或关联会话已删除）
// - session_id IS NULL：未关联会话的文件（如临时上传）
// - session.deleted_at IS NOT NULL：关联的会话已被软删除
// 不会清理关联活跃会话的文件，保证"会话文件与会话同寿命"
type FileCleanupConfig struct {
	ExpiresAfter string `mapstructure:"expires_after"` // 文件过期时间，如 "24h", "7d"
	BatchSize    int    `mapstructure:"batch_size"`    // 每批清理的文件数量
	Interval     int    `mapstructure:"interval"`      // 清理间隔（秒）
}

// LoadWithValidation 从指定路径加载并验证配置。
//
// 每次调用都创建独立的 viper 实例，避免全局状态在并发/重载场景下互相覆盖。
func LoadWithValidation(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 环境变量映射：仅作用于本实例
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// 全局 validator 实例（线程安全）
// tag → 可读描述映射。key 与 validate tag 完全一致，大小写敏感。
var tagDesc = map[string]string{
	"required": "cannot be empty",
	"oneof":    "must be one of [%s]", // Param 填入允许值列表
	"gt":       "must be greater than %s",
	"gte":      "must be ≥ %s",
	"lt":       "must be less than %s",
	"lte":      "must be ≤ %s",
	"eq":       "must be equal to %s",
	"ne":       "must not be equal to %s",
	"min":      "min length is %s",
	"max":      "max length is %s",
}

// formatFieldError 把单个字段校验错误转成运维友好的描述。
// 输出形如：server.port: must be ≤ 65535
func formatFieldError(fe validator.FieldError) string {
	tag := fe.Tag()
	desc, ok := tagDesc[tag]
	if !ok {
		return fmt.Sprintf("%s: invalid tag %q", fe.Namespace(), tag)
	}
	var msg string
	if fe.Param() != "" {
		// oneof/min/max 等带参数的 tag，填入参数。
		// 对于 oneof，把空格分隔的参数用 ", " 连接。
		if tag == "oneof" {
			msg = fmt.Sprintf(desc, strings.Join(strings.Fields(fe.Param()), ", "))
		} else {
			msg = fmt.Sprintf(desc, fe.Param())
		}
	} else {
		// 无参数 tag（如 required）直接用描述。
		msg = desc
	}
	return fmt.Sprintf("%s: %s", fe.Namespace(), msg)
}

var validate = validator.New(validator.WithRequiredStructEnabled())

// Validate 使用 struct tag 校验配置。
//
// 规则集中在各字段的 `validate:"..."` 标签中；新增字段时只需追加 tag 即可，
// 不必再修改 if 链，便于审计与单元测试。
//
// 同时补充各字段的默认值。
func (c *Config) Validate() error {
	if err := validate.Struct(c); err != nil {
		// 把 validator.ValidationErrors 转成运维友好描述，便于 CLI 排错。
		if ve, ok := err.(validator.ValidationErrors); ok {
			parts := make([]string, 0, len(ve))
			for _, fe := range ve {
				parts = append(parts, formatFieldError(fe))
			}
			return fmt.Errorf("config validation failed: %s", strings.Join(parts, "; "))
		}
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 补充默认值
	c.applyDefaults()

	return nil
}

// applyDefaults 补充配置字段的默认值
func (c *Config) applyDefaults() {
	// LLM 配置默认值
	if c.LLM.ToolCallTimeout == 0 {
		c.LLM.ToolCallTimeout = 15
	}

	// 文件清理配置默认值
	if c.FileCleanup.ExpiresAfter == "" {
		c.FileCleanup.ExpiresAfter = "168h" // 默认 7 天
	}
	if c.FileCleanup.BatchSize == 0 {
		c.FileCleanup.BatchSize = 100
	}
	if c.FileCleanup.Interval == 0 {
		c.FileCleanup.Interval = 3600 // 默认 1 小时
	}

	// Server 配置默认值
	if c.Server.ShutdownTimeoutSec == 0 {
		c.Server.ShutdownTimeoutSec = 30
	}
	if c.Server.ReadTimeoutSec == 0 {
		c.Server.ReadTimeoutSec = 30
	}
	if c.Server.IdleTimeoutSec == 0 {
		c.Server.IdleTimeoutSec = 60
	}
	// WriteTimeoutSec 保持 0，留给流式接口

	// 日志配置默认值
	if c.Log.MaxSize == 0 {
		c.Log.MaxSize = 100 // 默认 100MB
	}
	if c.Log.MaxBackups == 0 {
		c.Log.MaxBackups = 7
	}
	if c.Log.MaxAge == 0 {
		c.Log.MaxAge = 30
	}

	// 数据库连接池默认值
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}
}
