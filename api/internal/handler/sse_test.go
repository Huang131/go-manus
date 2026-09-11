package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

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
	got := mergeEventMetadata(context.Background(), &model.Event{
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

func TestMergeEventMetadata_NullPayloadReturnsOriginalData(t *testing.T) {
	input := []byte("null")
	got := mergeEventMetadata(context.Background(), &model.Event{
		ID:        "event-1",
		CreatedAt: time.Unix(123, 0).UTC(),
		Data:      input,
	})
	if string(got) != string(input) {
		t.Fatalf("mergeEventMetadata() = %s, want original null payload", got)
	}
}

func TestWriteSSEEventIncludesStreamCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	if err := writeSSEEvent(ctx, "message_delta", "1710000000000-0", []byte(`{"delta":"你"}`)); err != nil {
		t.Fatalf("writeSSEEvent() error = %v", err)
	}
	want := "id: 1710000000000-0\nevent: message_delta\ndata: {\"delta\":\"你\"}\n\n"
	if recorder.Body.String() != want {
		t.Fatalf("SSE body = %q, want %q", recorder.Body.String(), want)
	}
}
