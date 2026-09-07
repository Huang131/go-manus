package middleware

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"github.com/mooc-manus/go-manus/api/pkg/response"
)

// Recovery 错误恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录堆栈信息
				logger.Error("panic recovered",
					logger.Any("error", err),
					logger.String("stack", string(debug.Stack())),
				)

				response.Error(c, "Internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 请求前
		logger.Info("request started",
			logger.String("method", c.Request.Method),
			logger.String("path", c.Request.URL.Path),
			logger.String("client_ip", c.ClientIP()),
		)

		// 处理请求
		c.Next()

		// 请求后
		logger.Info("request finished",
			logger.String("method", c.Request.Method),
			logger.String("path", c.Request.URL.Path),
			logger.Int("status", c.Writer.Status()),
			logger.Int64("latency_ms", time.Since(start).Milliseconds()),
		)
	}
}

// CORS CORS 中间件
// 鉴权由后端在路由内部完成（不依赖浏览器 Cookie），因此 AllowAllOrigins 安全。
// 同时显式放行 SSE 所需的 Last-Event-ID 头与常见 Content-Type。
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = false // 配合 AllowAllOrigins，避免 gin-contrib/cors 拒绝
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "Last-Event-ID"}
	config.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	return cors.New(config)
}

// RequestID 请求 ID 中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
