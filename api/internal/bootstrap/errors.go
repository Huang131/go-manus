package bootstrap

import "errors"

// Sentinel errors，便于 main 端通过 errors.Is 分类退出码，
// 而非依赖错误消息字符串匹配。
var (
	// ErrInitialize 表示依赖初始化失败（数据库/Redis/对象存储等）。
	// 包装规则：fmt.Errorf("...: %w", ErrInitialize)。
	ErrInitialize = errors.New("bootstrap initialize failed")

	// ErrHealthCheckFailed 表示启动后健康检查失败。
	// 包装规则：fmt.Errorf("...: %w", ErrHealthCheckFailed)。
	ErrHealthCheckFailed = errors.New("bootstrap health check failed")
)
