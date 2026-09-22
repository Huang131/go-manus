package config

const (
	// LLM 默认工具调用超时时间（秒）。
	DefaultLLMToolCallTimeoutSec = 15
	// 沙箱 HTTP 请求超时（秒）。必须大于沙箱侧最慢动作超时
	// （sandbox/app/services/browser.py 的 NAVIGATE/SCREENSHOT_TIMEOUT_SECONDS = 90），
	// 否则 Go 客户端会先于沙箱超时，只能抛 context deadline exceeded，
	// 拿不到沙箱返回的语义化 504。
	DefaultSandboxHTTPTimeoutSec = 120
	DefaultSearchHTTPTimeoutSec  = 30

	// HTTP 服务默认超时时间（秒）。
	DefaultServerReadTimeoutSec     = 30
	DefaultServerIdleTimeoutSec     = 60
	DefaultServerShutdownTimeoutSec = 30

	// 文件清理默认参数。
	DefaultFileCleanupExpiresAfter = "168h"
	DefaultFileCleanupBatchSize    = 100
	DefaultFileCleanupIntervalSec  = 3600

	// 日志默认滚动参数。
	DefaultLogMaxSize    = 100
	DefaultLogMaxBackups = 7
	DefaultLogMaxAge     = 30

	// 数据库默认连接池参数。
	DefaultDatabaseMaxOpenConns = 25
	DefaultDatabaseMaxIdleConns = 5
)
