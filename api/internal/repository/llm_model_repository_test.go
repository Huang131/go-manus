package repository

import (
	"encoding/json"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/model"
)

func TestRuntimeHealthJSONRoundTrip(t *testing.T) {
	src := model.RuntimeHealth{
		Status:           "unhealthy",
		RecentFailures:   4,
		AverageLatencyMS: 876,
	}

	b := runtimeHealthToJSON(src)
	if len(b) == 0 {
		t.Fatal("runtime health json should not be empty")
	}

	var got model.RuntimeHealth
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != src.Status || got.RecentFailures != src.RecentFailures || got.AverageLatencyMS != src.AverageLatencyMS {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, src)
	}
}
