package handler

import (
	"strings"
	"time"

	"github.com/Huang131/go-manus/api/pkg/httpconst"
	"github.com/gin-gonic/gin"
)

const (
	sseEventSessions = "sessions"

	sessionStreamPollInterval = 5 * time.Second
)

// setSSEHeaders 统一设置 SSE 响应头，避免不同流式接口出现行为漂移。
func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", httpconst.ContentTypeSSE)
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no")
}

// writeSSEEvent 写出带 Redis Stream 游标的 SSE 事件。
// SSE 的 id 字段是断线续读游标；Run 事件 payload 中的 event_id 由 handler 写出时同步为该游标。
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
