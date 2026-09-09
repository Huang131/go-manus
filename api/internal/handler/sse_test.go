package handler

import (
	"testing"

	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

func TestSetSSEHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	setSSEHeaders(ctx)

	want := map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"Connection":        "keep-alive",
		"Transfer-Encoding": "chunked",
		"X-Accel-Buffering": "no",
	}
	for key, value := range want {
		if got := ctx.Writer.Header().Get(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
}
