package sandbox

import "time"

// maxDownloadBytes 下载文件大小上限（64MB），避免一次性读爆 API 进程内存。
const maxDownloadBytes = 64 << 20

const (
	// SandboxTimeoutHeader 从 Go 侧传播给沙箱的剩余预算请求头（毫秒）。
	// 沙箱据此把"锁等待 + 动作执行"切分成两段预算，各自耗尽前主动返回
	// 语义化状态码（503 繁忙 / 504 超时），而不是让上层裸超时。
	SandboxTimeoutHeader = "X-Sandbox-Timeout-Ms"
	// budgetReserve 预算传播时的往返余量：序列化/网络/连接建立也要消耗时间，
	// 全量播给沙箱会导致沙箱在预算耗尽前无任何可执行时间。
	budgetReserve = time.Second
)
