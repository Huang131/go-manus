package config

const (
	// LLM 默认工具调用超时时间（秒）。
	DefaultLLMToolCallTimeoutSec = 15

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
