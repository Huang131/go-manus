package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"net/http/httptest"

	"github.com/Huang131/go-manus/api/internal/model"
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

func TestMergeEventMetadata(t *testing.T) {
	createdAt := time.Unix(123, 0).UTC()
	got := mergeEventMetadata(&model.Event{
		CreatedAt: createdAt,
		Data:      json.RawMessage(`{"message":"hello"}`),
	})
	var payload map[string]interface{}
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatalf("mergeEventMetadata() returned invalid JSON: %v", err)
	}
	if payload["event_id"] != "" || payload["created_at"] != float64(123) {
		t.Fatalf("mergeEventMetadata() = %v, want event metadata", payload)
	}
}
