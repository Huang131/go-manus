package handler

import (
	"strings"
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

// writeSSEEvent 写出带 Redis Stream 游标的 SSE 事件。
// SSE 的 id 字段是断线续读唯一可信的游标，业务 payload 中的 event_id 仍保持 UUID 语义。
func writeSSEEvent(c *gin.Context, eventType, streamID string, payload []byte) error {
	var b strings.Builder
	if streamID != "" {
		b.WriteString("id: ")
		b.WriteString(streamID)
		b.WriteString("\n")
	}
	if eventType != "" {
		b.WriteString("event: ")
		b.WriteString(eventType)
		b.WriteString("\n")
	}
	b.WriteString("data: ")
	b.Write(payload)
	b.WriteString("\n\n")
	if _, err := c.Writer.WriteString(b.String()); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}
