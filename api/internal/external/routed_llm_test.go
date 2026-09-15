package external

import (
	"errors"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// healthOfModel 通过 applyStoredHealth 读取路由器内存中指定模型的健康快照。
func healthOfModel(router *RoutedLLM, id string) LLMRuntimeHealth {
	cfg := &LLMRuntimeConfig{Profile: llmcore.ModelProfile{ID: id}}
	router.applyStoredHealth([]*LLMRuntimeConfig{cfg})
	return cfg.Health
}

func TestRoutedLLM_InvalidateHealth(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)

	router.RecordFailure("model-a", errors.New("boom"), 100*time.Millisecond)
	router.RecordFailure("model-a", errors.New("boom"), 120*time.Millisecond)

	if h := healthOfModel(router, "model-a"); h.RecentFailures != 2 || h.Status != model.HealthStateDegraded {
		t.Fatalf("before invalidate: got %+v, want 2 failures/degraded", h)
	}

	router.InvalidateHealth("model-a")

	if h := healthOfModel(router, "model-a"); h.RecentFailures != 0 || h.AverageLatencyMS != 0 {
		t.Fatalf("after invalidate: got %+v, want zero health", h)
	}

	// 失效后重新积累：从零开始，不与旧值混合。
	router.RecordSuccess("model-a", 30*time.Millisecond)
	h := healthOfModel(router, "model-a")
	if h.RecentFailures != 0 || h.AverageLatencyMS != 30 {
		t.Fatalf("after re-record: got %+v, want failures=0 latency=30", h)
	}
}

func TestRoutedLLM_GetHealth(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)

	// 无记录 / 空 id：零值
	if h := router.GetHealth("model-a"); h.Status != "" || h.RecentFailures != 0 || h.AverageLatencyMS != 0 {
		t.Fatalf("no record: got %+v, want zero value", h)
	}
	if h := router.GetHealth(""); h.Status != "" || h.RecentFailures != 0 || h.AverageLatencyMS != 0 {
		t.Fatalf("empty id: got %+v, want zero value", h)
	}

	// 记录两次失败：净计数 2、degraded、EMA=(100+300)/2
	router.RecordFailure("model-a", errors.New("boom"), 100*time.Millisecond)
	router.RecordFailure("model-a", errors.New("boom"), 300*time.Millisecond)
	h := router.GetHealth("model-a")
	if h.RecentFailures != 2 || h.Status != model.HealthStateDegraded {
		t.Fatalf("after failures: got %+v, want 2 failures/degraded", h)
	}
	if h.AverageLatencyMS != 200 {
		t.Fatalf("after failures: latency = %d, want 200", h.AverageLatencyMS)
	}

	// 建流即失败传 0 延迟：不污染 EMA，但计数仍累加并触发 unhealthy 阈值
	router.RecordFailure("model-a", errors.New("stream refused"), 0)
	h = router.GetHealth("model-a")
	if h.AverageLatencyMS != 200 {
		t.Fatalf("zero-latency failure: latency = %d, want unchanged 200", h.AverageLatencyMS)
	}
	if h.RecentFailures != 3 || h.Status != model.HealthStateUnhealthy {
		t.Fatalf("after third failure: got %+v, want 3 failures/unhealthy", h)
	}

	router.InvalidateHealth("model-a")
	if h := router.GetHealth("model-a"); h.RecentFailures != 0 || h.AverageLatencyMS != 0 {
		t.Fatalf("after invalidate: got %+v, want zero", h)
	}
}
