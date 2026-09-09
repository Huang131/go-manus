package bootstrap

import "time"

const healthCheckTimeout = 10 * time.Second

// defaultTrustedProxies 每次返回新切片，避免调用方修改共享底层数组。
func defaultTrustedProxies() []string {
	return []string{"127.0.0.1", "::1"}
}
