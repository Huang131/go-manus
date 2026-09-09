package external

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

type mockRuntimeHealthStore struct {
	updates map[string]modelRuntimeHealthSnapshot
}

type modelRuntimeHealthSnapshot struct {
	status           string
	recentFailures   int
	averageLatencyMS int
}

func newMockRuntimeHealthStore() *mockRuntimeHealthStore {
	return &mockRuntimeHealthStore{updates: make(map[string]modelRuntimeHealthSnapshot)}
}

func (m *mockRuntimeHealthStore) UpdateRuntimeHealth(ctx context.Context, id string, health model.RuntimeHealth) error {
	m.updates[id] = modelRuntimeHealthSnapshot{
		status:           health.Status,
		recentFailures:   health.RecentFailures,
		averageLatencyMS: health.AverageLatencyMS,
	}
	return nil
}

func TestRoutedLLM_FallbackOnRateLimit(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "primary",
				},
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					if cfg.ModelName == "primary" {
						return nil, llmcore.NewProviderError(llmcore.KindRateLimit, "openai_compat", cfg.ModelName, "rate limit")
					}
					return &LLMResponse{Content: cfg.ModelName}, nil
				},
			}
		},
	)

	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "backup" {
		t.Fatalf("content = %q, want backup", resp.Content)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %v, want 2 attempts", attempts)
	}
}

func TestRoutedLLM_PreferHealthyCandidate(t *testing.T) {
	var gotModel string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "degraded",
					Health: LLMRuntimeHealth{
						Status:           LLMHealthDegraded,
						RecentFailures:   3,
						AverageLatencyMS: 1200,
					},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "healthy",
					Health: LLMRuntimeHealth{
						Status:           LLMHealthHealthy,
						RecentFailures:   0,
						AverageLatencyMS: 300,
					},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			gotModel = cfg.ModelName
			return &stubLLM{name: cfg.ModelName}
		},
	)

	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if gotModel != "healthy" {
		t.Fatalf("got model %s, want healthy", gotModel)
	}
	if resp.Content != "healthy" {
		t.Fatalf("content = %q, want healthy", resp.Content)
	}
}

func TestRoutedLLM_DoNotFallbackAcrossProtocol(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolAnthropic,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "claude",
				},
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindRateLimit, "openai_compat", cfg.ModelName, "rate limit")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindRateLimit) {
		t.Fatalf("kind = %v, want rate limit", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("attempts = %v, want 1 attempt", attempts)
	}
}

func TestRoutedLLM_DoNotFallbackAfterToolUseSideEffect(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "primary",
				},
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: llmcore.RoleUser, ContentText: "hello"},
			{Role: llmcore.RoleTool, Name: "shell", ContentText: "result"},
		},
		Tools: []llmcore.ToolSpec{
			{
				Type: "function",
				Function: llmcore.ToolSpecFunction{
					Name:        "shell",
					Description: "shell",
					Parameters:  map[string]interface{}{"type": "object"},
				},
				ReadOnly: false,
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindServer) {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("attempts = %v, want 1 attempt", attempts)
	}
}

func TestRoutedLLM_AllowFallbackBeforeToolExecution(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "primary",
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{
				Role:      llmcore.RoleAssistant,
				ToolCalls: []llmcore.ToolCall{{ID: "tool-1", Type: "function", Function: llmcore.ToolCallFunction{Name: "shell", Arguments: "{}"}}},
			},
		},
		Tools: []llmcore.ToolSpec{{
			Type: "function",
			Function: llmcore.ToolSpecFunction{
				Name:        "shell",
				Description: "shell",
				Parameters:  map[string]interface{}{"type": "object"},
			},
			ReadOnly: false,
		}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %v, want 2 attempts", attempts)
	}
}

func TestRoutedLLM_RecordSuccessUpdatesRuntimeHealth(t *testing.T) {
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "slow",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 1000},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "fast",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 1000},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{name: cfg.ModelName}
		},
	)

	router.RecordSuccess("fast", 10*time.Millisecond)
	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "fast" {
		t.Fatalf("content = %q, want fast", resp.Content)
	}
}

func TestRoutedLLM_RecordFailureUpdatesRuntimeHealth(t *testing.T) {
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "primary",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 100},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "backup",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 100},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{name: cfg.ModelName}
		},
	)

	router.RecordFailure("primary", llmcore.NewProviderError(llmcore.KindServer, "openai_compat", "primary", "server error"), 20*time.Millisecond)
	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "backup" {
		t.Fatalf("content = %q, want backup", resp.Content)
	}
}

func TestRoutedLLM_PersistRuntimeHealth(t *testing.T) {
	store := newMockRuntimeHealthStore()
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						ID:           "model-1",
						Protocol:     llmcore.ProtocolOpenAICompat,
						Capabilities: openAITextProfile().Capabilities,
					},
					ModelName: "model-1",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					return &LLMResponse{Content: "ok"}, nil
				},
			}
		},
	)
	router.SetHealthStore(store)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	got, ok := store.updates["model-1"]
	if !ok {
		t.Fatal("expected runtime health to be persisted")
	}
	if got.status != LLMHealthHealthy {
		t.Fatalf("status = %s, want healthy", got.status)
	}
	if got.averageLatencyMS <= 0 {
		t.Fatalf("average latency = %d, want positive", got.averageLatencyMS)
	}
}

func openAITextProfile() llmcore.ModelProfile {
	return llmcore.ModelProfile{
		Protocol: llmcore.ProtocolOpenAICompat,
		Capabilities: llmcore.ModelCapabilities{
			SupportsText:      true,
			SupportsToolCalls: true,
			SupportsStreaming: true,
			MaxContextTokens:  4096,
			MaxOutputTokens:   1024,
		},
	}
}

// ============================================================================
// 健康状态记录与衰减逻辑单元测试
// ============================================================================

// getHealth 读取内部健康快照（测试辅助）
func (r *RoutedLLM) getHealth(modelKey string) LLMRuntimeHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.health[modelKey]
}

func TestRoutedLLM_RecordFailureEscalatesToUnhealthy(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)
	const key = "model-x"

	// 第 1 次失败：degraded, failures=1
	router.RecordFailure(key, nil, 10*time.Millisecond)
	h := router.getHealth(key)
	if h.Status != LLMHealthDegraded || h.RecentFailures != 1 {
		t.Fatalf("after 1st failure: status=%s failures=%d, want degraded/1", h.Status, h.RecentFailures)
	}

	// 第 2 次失败：仍 degraded, failures=2
	router.RecordFailure(key, nil, 10*time.Millisecond)
	h = router.getHealth(key)
	if h.Status != LLMHealthDegraded || h.RecentFailures != 2 {
		t.Fatalf("after 2nd failure: status=%s failures=%d, want degraded/2", h.Status, h.RecentFailures)
	}

	// 第 3 次失败：升级为 unhealthy, failures=3
	router.RecordFailure(key, nil, 10*time.Millisecond)
	h = router.getHealth(key)
	if h.Status != LLMHealthUnhealthy || h.RecentFailures != 3 {
		t.Fatalf("after 3rd failure: status=%s failures=%d, want unhealthy/3", h.Status, h.RecentFailures)
	}
}

func TestRoutedLLM_RecordSuccessDecaysFailuresAndRecovers(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)
	const key = "model-y"

	// 先失败 2 次：degraded, failures=2
	router.RecordFailure(key, nil, 10*time.Millisecond)
	router.RecordFailure(key, nil, 10*time.Millisecond)

	// 第 1 次成功：failures 衰减到 1，仍 degraded
	router.RecordSuccess(key, 50*time.Millisecond)
	h := router.getHealth(key)
	if h.RecentFailures != 1 || h.Status != LLMHealthDegraded {
		t.Fatalf("after 1st success: status=%s failures=%d, want degraded/1", h.Status, h.RecentFailures)
	}

	// 第 2 次成功：failures 归零，恢复 healthy
	router.RecordSuccess(key, 50*time.Millisecond)
	h = router.getHealth(key)
	if h.RecentFailures != 0 || h.Status != LLMHealthHealthy {
		t.Fatalf("after 2nd success: status=%s failures=%d, want healthy/0", h.Status, h.RecentFailures)
	}
}

func TestRoutedLLM_RecordSuccessTracksLatency(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)
	const key = "model-z"

	// 首次成功：延迟直接记录（10ms）
	router.RecordSuccess(key, 10*time.Millisecond)
	h := router.getHealth(key)
	if h.AverageLatencyMS != 10 {
		t.Fatalf("first latency = %d, want 10", h.AverageLatencyMS)
	}

	// 第二次成功：滑动平均 (10+30)/2 = 20
	router.RecordSuccess(key, 30*time.Millisecond)
	h = router.getHealth(key)
	if h.AverageLatencyMS != 20 {
		t.Fatalf("second latency = %d, want 20", h.AverageLatencyMS)
	}
}

func TestRoutedLLM_RecordIgnoresEmptyModelKey(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)

	router.RecordSuccess("", 10*time.Millisecond)
	router.RecordFailure("", nil, 10*time.Millisecond)

	router.mu.RLock()
	count := len(router.health)
	router.mu.RUnlock()
	if count != 0 {
		t.Fatalf("health map size = %d, want 0 (empty key should be ignored)", count)
	}
}

// ============================================================================
// plan() 排序逻辑单元测试
// ============================================================================

func TestRoutedLLM_PlanSortsByHealthThenLatency(t *testing.T) {
	catalog := []*LLMRuntimeConfig{
		{Profile: openAITextProfile(), ModelName: "unhealthy"},
		{Profile: openAITextProfile(), ModelName: "degraded-slow"},
		{Profile: openAITextProfile(), ModelName: "degraded-fast"},
		{Profile: openAITextProfile(), ModelName: "healthy-slow"},
		{Profile: openAITextProfile(), ModelName: "healthy-fast"},
	}
	router := NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		return catalog, nil
	}, nil, nil)

	// 通过 RecordFailure 构造 unhealthy（3 次失败）
	for i := 0; i < 3; i++ {
		router.RecordFailure("unhealthy", nil, 100*time.Millisecond)
	}
	// 构造 degraded：1 次失败 + 不同延迟
	router.RecordFailure("degraded-slow", nil, 2000*time.Millisecond)
	router.RecordFailure("degraded-fast", nil, 100*time.Millisecond)
	// 构造 healthy：记录不同延迟
	router.RecordSuccess("healthy-slow", 1500*time.Millisecond)
	router.RecordSuccess("healthy-fast", 100*time.Millisecond)

	plan := router.plan(context.Background(), &LLMRequest{})
	if len(plan) != 5 {
		t.Fatalf("plan size = %d, want 5", len(plan))
	}

	wantOrder := []string{"healthy-fast", "healthy-slow", "degraded-fast", "degraded-slow", "unhealthy"}
	for i, want := range wantOrder {
		if plan[i].ModelName != want {
			t.Fatalf("plan[%d] = %s, want %s (full order: %v)", i, plan[i].ModelName, want, modelNames(plan))
		}
	}
}

func TestRoutedLLM_PlanFallsBackWhenCatalogEmpty(t *testing.T) {
	router := NewRoutedLLM(nil, &LLMRuntimeConfig{
		Profile:   openAITextProfile(),
		ModelName: "fallback-only",
	}, nil)

	// catalog 为 nil → 应返回 [fallback]
	plan := router.plan(context.Background(), &LLMRequest{})
	if len(plan) != 1 || plan[0].ModelName != "fallback-only" {
		t.Fatalf("plan = %v, want [fallback-only]", modelNames(plan))
	}
}

func TestRoutedLLM_PlanPrefersFewerFailuresOnSameStatus(t *testing.T) {
	// 相同健康状态（degraded）下，失败次数更少者优先
	catalog := []*LLMRuntimeConfig{
		{Profile: openAITextProfile(), ModelName: "more-failures"},
		{Profile: openAITextProfile(), ModelName: "fewer-failures"},
	}
	router := NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		return catalog, nil
	}, nil, nil)

	// 两个模型都 degraded、延迟相同，但失败次数不同
	for i := 0; i < 2; i++ {
		router.RecordFailure("more-failures", nil, 100*time.Millisecond)
	}
	router.RecordFailure("fewer-failures", nil, 100*time.Millisecond)

	plan := router.plan(context.Background(), &LLMRequest{})
	if len(plan) < 2 {
		t.Fatalf("plan size = %d, want >= 2", len(plan))
	}
	if plan[0].ModelName != "fewer-failures" {
		t.Fatalf("plan[0] = %s, want fewer-failures", plan[0].ModelName)
	}
}

// ============================================================================
// persistHealth 边界单元测试
// ============================================================================

func TestRoutedLLM_PersistHealthSkipsModelsOnlyInMemory(t *testing.T) {
	// Profile.ID 为空的模型不持久化（configKey 回退到 ModelName，但 persistHealth 明确要求 Profile.ID）
	store := newMockRuntimeHealthStore()
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{Profile: openAITextProfile(), ModelName: "no-profile-id"},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{name: cfg.ModelName}
		},
	)
	router.SetHealthStore(store)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	if len(store.updates) != 0 {
		t.Fatalf("store updates = %v, want empty (no Profile.ID should skip persistence)", store.updates)
	}
}

// modelNames 提取 plan 中模型名列表（测试辅助，用于失败信息打印）
func modelNames(plan []*LLMRuntimeConfig) []string {
	names := make([]string, 0, len(plan))
	for _, cfg := range plan {
		if cfg != nil {
			names = append(names, cfg.ModelName)
		}
	}
	return names
}
