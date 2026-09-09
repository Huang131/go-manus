package handler

import (
	"time"

	"github.com/gin-gonic/gin"
)

const (
	sseEventSessions = "sessions"
	sseEventMessage  = "message"
	sseEventTaskID   = "task_id"

	sessionStreamPollInterval = 5 * time.Second
	sessionStreamTimeout      = 30 * time.Minute
	sessionHeartbeatInterval  = 15 * time.Second
	sessionEventPollInterval  = 100 * time.Millisecond
)

// setSSEHeaders 统一设置 SSE 响应头，避免不同流式接口出现行为漂移。
func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no")
}
