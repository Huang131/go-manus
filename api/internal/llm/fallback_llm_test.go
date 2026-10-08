package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestRoutedLLM_FallbackOnRateLimit(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: model.ModelCapabilities{
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
						Capabilities: model.ModelCapabilities{
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
				invoke: func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					if cfg.ModelName == "primary" {
						return nil, llmcore.NewProviderError(llmcore.KindRateLimit, "openai_compat", cfg.ModelName, "rate limit")
					}
					return &llmcore.LLMResponse{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: cfg.ModelName}}, nil
				},
			}
		},
	)

	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Message.ContentText != "backup" {
		t.Fatalf("content = %q, want backup", resp.Message.ContentText)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %v, want 2 attempts", attempts)
	}
}

func TestRoutedLLM_StreamUsesStreamingCandidate(t *testing.T) {
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{Profile: openAITextProfile(), ModelName: "streaming"},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &streamingStubLLM{
				stubLLM: &stubLLM{name: cfg.ModelName},
				deltas:  []llmcore.LLMDelta{{ContentText: "token"}},
			}
		},
	)

	streaming, ok := interface{}(router).(StreamingLLM)
	if !ok {
		t.Fatal("RoutedLLM should implement StreamingLLM")
	}
	deltas, err := streaming.Stream(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	var got []llmcore.LLMDelta
	for delta := range deltas {
		got = append(got, delta)
	}
	if len(got) != 1 || got[0].ContentText != "token" {
		t.Fatalf("deltas = %+v, want one token delta", got)
	}
}

// TestRoutedLLM_StreamErrorDeltaRecordsFailure 流中出现 Error delta（协议约定表示本次流失败）
// 后，channel 关闭时应记录失败而非成功，避免把失败模型标为健康、影响后续路由排序。
func TestRoutedLLM_StreamErrorDeltaRecordsFailure(t *testing.T) {
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{Profile: withProfileID(openAITextProfile(), "streaming"), ModelName: "streaming"},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &streamingStubLLM{
				stubLLM: &stubLLM{name: cfg.ModelName},
				deltas:  []llmcore.LLMDelta{{Error: "upstream stream failure"}},
			}
		},
	)

	deltas, err := router.Stream(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	sawError := false
	for d := range deltas {
		if d.Error != "" {
			sawError = true
		}
	}
	if !sawError {
		t.Fatal("expected an Error delta to be forwarded")
	}

	h := router.GetHealth("streaming")
	if h.Status != model.HealthStateDegraded || h.RecentFailures != 1 {
		t.Fatalf("health = %+v, want degraded/1 after stream error", h)
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
						Status:           model.HealthStateDegraded,
						RecentFailures:   3,
						AverageLatencyMS: 1200,
					},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "healthy",
					Health: LLMRuntimeHealth{
						Status:           model.HealthStateHealthy,
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
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if gotModel != "healthy" {
		t.Fatalf("got model %s, want healthy", gotModel)
	}
	if resp.Message.ContentText != "healthy" {
		t.Fatalf("content = %q, want healthy", resp.Message.ContentText)
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
						Capabilities: model.ModelCapabilities{
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
						Capabilities: model.ModelCapabilities{
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
				invoke: func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindRateLimit, "openai_compat", cfg.ModelName, "rate limit")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hello"}},
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
						Capabilities: model.ModelCapabilities{
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
						Capabilities: model.ModelCapabilities{
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
				invoke: func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hello"},
			{Role: model.RoleTool, Name: "shell", ContentText: "result"},
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
				invoke: func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{
				Role:      model.RoleAssistant,
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

// TestRoutedLLM_LatencyAffectsPlanOrder 测试成功记录后的延迟如何影响 plan 排序。
// 低延迟模型应优先于高延迟模型（相同健康状态）。
func TestRoutedLLM_LatencyAffectsPlanOrder(t *testing.T) {
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "slow",
					Health:    LLMRuntimeHealth{Status: model.HealthStateHealthy, AverageLatencyMS: 1000},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "fast",
					Health:    LLMRuntimeHealth{Status: model.HealthStateHealthy, AverageLatencyMS: 1000},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{name: cfg.ModelName}
		},
	)

	router.RecordSuccess(recordAs("fast"), 10*time.Millisecond)
	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Message.ContentText != "fast" {
		t.Fatalf("content = %q, want fast", resp.Message.ContentText)
	}
}

// TestRoutedLLM_DegradedModelIsSortedLower 测试健康状态降级后，模型在 plan 中排序优先级降低。
// 与 TestRoutedLLM_FallbackOnRateLimit（调用失败后 fallback）不同，这里测试的是排序变化。
func TestRoutedLLM_DegradedModelIsSortedLower(t *testing.T) {
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "primary",
					Health:    LLMRuntimeHealth{Status: model.HealthStateHealthy, AverageLatencyMS: 100},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "backup",
					Health:    LLMRuntimeHealth{Status: model.HealthStateHealthy, AverageLatencyMS: 100},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{name: cfg.ModelName}
		},
	)

	router.RecordFailure(recordAs("primary"), llmcore.NewProviderError(llmcore.KindServer, "openai_compat", "primary", "server error"), 20*time.Millisecond)
	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Message.ContentText != "backup" {
		t.Fatalf("content = %q, want backup", resp.Message.ContentText)
	}
}

// ============================================================================
// 健康状态记录与衰减逻辑单元测试
// ============================================================================

// recordAs 构造以 name 为 Profile.ID 的目录模型配置用于健康记录。
// 健康按 Profile.ID 记录；测试目录条目无 ID 时 configKey 退化为 ModelName，
// 两者同名即可命中同一把 key。
func recordAs(name string) *LLMRuntimeConfig {
	return &LLMRuntimeConfig{Profile: llmcore.ModelProfile{ID: name}, ModelName: name}
}

func TestRoutedLLM_RecordFailureEscalatesToUnhealthy(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)
	const key = "model-x"

	// 第 1 次失败：degraded, failures=1
	router.RecordFailure(recordAs(key), nil, 10*time.Millisecond)
	h := router.GetHealth(key)
	if h.Status != model.HealthStateDegraded || h.RecentFailures != 1 {
		t.Fatalf("after 1st failure: status=%s failures=%d, want degraded/1", h.Status, h.RecentFailures)
	}

	// 第 2 次失败：仍 degraded, failures=2
	router.RecordFailure(recordAs(key), nil, 10*time.Millisecond)
	h = router.GetHealth(key)
	if h.Status != model.HealthStateDegraded || h.RecentFailures != 2 {
		t.Fatalf("after 2nd failure: status=%s failures=%d, want degraded/2", h.Status, h.RecentFailures)
	}

	// 第 3 次失败：升级为 unhealthy, failures=3
	router.RecordFailure(recordAs(key), nil, 10*time.Millisecond)
	h = router.GetHealth(key)
	if h.Status != model.HealthStateUnhealthy || h.RecentFailures != 3 {
		t.Fatalf("after 3rd failure: status=%s failures=%d, want unhealthy/3", h.Status, h.RecentFailures)
	}
}

func TestRoutedLLM_RecordSuccessDecaysFailuresAndRecovers(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)
	const key = "model-y"

	// 先失败 2 次：degraded, failures=2
	router.RecordFailure(recordAs(key), nil, 10*time.Millisecond)
	router.RecordFailure(recordAs(key), nil, 10*time.Millisecond)

	// 第 1 次成功：failures 衰减到 1，仍 degraded
	router.RecordSuccess(recordAs(key), 50*time.Millisecond)
	h := router.GetHealth(key)
	if h.RecentFailures != 1 || h.Status != model.HealthStateDegraded {
		t.Fatalf("after 1st success: status=%s failures=%d, want degraded/1", h.Status, h.RecentFailures)
	}

	// 第 2 次成功：failures 归零，恢复 healthy
	router.RecordSuccess(recordAs(key), 50*time.Millisecond)
	h = router.GetHealth(key)
	if h.RecentFailures != 0 || h.Status != model.HealthStateHealthy {
		t.Fatalf("after 2nd success: status=%s failures=%d, want healthy/0", h.Status, h.RecentFailures)
	}
}

func TestRoutedLLM_RecordSuccessTracksLatency(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)
	const key = "model-z"

	// 首次成功：延迟直接记录（10ms）
	router.RecordSuccess(recordAs(key), 10*time.Millisecond)
	h := router.GetHealth(key)
	if h.AverageLatencyMS != 10 {
		t.Fatalf("first latency = %d, want 10", h.AverageLatencyMS)
	}

	// 第二次成功：滑动平均 (10+30)/2 = 20
	router.RecordSuccess(recordAs(key), 30*time.Millisecond)
	h = router.GetHealth(key)
	if h.AverageLatencyMS != 20 {
		t.Fatalf("second latency = %d, want 20", h.AverageLatencyMS)
	}
}

func TestRoutedLLM_RecordIgnoresEmptyModelKey(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)

	router.RecordSuccess(&LLMRuntimeConfig{}, 10*time.Millisecond)
	router.RecordFailure(&LLMRuntimeConfig{}, nil, 10*time.Millisecond)

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
		router.RecordFailure(recordAs("unhealthy"), nil, 100*time.Millisecond)
	}
	// 构造 degraded：1 次失败 + 不同延迟
	router.RecordFailure(recordAs("degraded-slow"), nil, 2000*time.Millisecond)
	router.RecordFailure(recordAs("degraded-fast"), nil, 100*time.Millisecond)
	// 构造 healthy：记录不同延迟
	router.RecordSuccess(recordAs("healthy-slow"), 1500*time.Millisecond)
	router.RecordSuccess(recordAs("healthy-fast"), 100*time.Millisecond)

	plan, _ := router.plan(context.Background(), &LLMRequest{})
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
	plan, _ := router.plan(context.Background(), &LLMRequest{})
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
		router.RecordFailure(recordAs("more-failures"), nil, 100*time.Millisecond)
	}
	router.RecordFailure(recordAs("fewer-failures"), nil, 100*time.Millisecond)

	plan, _ := router.plan(context.Background(), &LLMRequest{})
	if len(plan) < 2 {
		t.Fatalf("plan size = %d, want >= 2", len(plan))
	}
	if plan[0].ModelName != "fewer-failures" {
		t.Fatalf("plan[0] = %s, want fewer-failures", plan[0].ModelName)
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

// === Auto / 粘性路由语义（对齐 Cursor：选定不切换；Auto 才允许系统兜底） ===

// openAIModelWithID 构造带 DB ID 的模型配置
func openAIModelWithID(id, name string) *LLMRuntimeConfig {
	p := openAITextProfile()
	p.ID = id
	return &LLMRuntimeConfig{Profile: p, ModelName: name}
}

// TestRoutedLLM_AutoAppendsConfiguredEnvFallback Auto（无 model_id）时，
// 配置过的 env fallback 追加在 plan 末尾作为最后兜底。
func TestRoutedLLM_AutoAppendsConfiguredEnvFallback(t *testing.T) {
	catalog := []*LLMRuntimeConfig{openAIModelWithID("m-1", "db-model")}
	fallback := &LLMRuntimeConfig{Profile: openAITextProfile(), ModelName: "env-fallback"}
	router := NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		return catalog, nil
	}, fallback, nil)

	plan, err := router.plan(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("plan() error = %v", err)
	}
	if len(plan) != 2 || plan[0].ModelName != "db-model" || plan[1].ModelName != "env-fallback" {
		t.Fatalf("plan = %v, want [db-model env-fallback]", modelNames(plan))
	}
}

// TestRoutedLLM_AutoSkipsCrossProtocolEnvFallback Auto（无 model_id）时，
// 目录主模型为 Anthropic，而 env fallback 是 OpenAI 兼容协议：跨协议兜底语义错乱，
// 不应被追加到 plan。
func TestRoutedLLM_AutoSkipsCrossProtocolEnvFallback(t *testing.T) {
	anthropicProfile := openAITextProfile()
	anthropicProfile.Protocol = llmcore.ProtocolAnthropic
	catalog := []*LLMRuntimeConfig{{Profile: anthropicProfile, ModelName: "claude"}}
	fallback := &LLMRuntimeConfig{Profile: openAITextProfile(), ModelName: "env-openai"}
	router := NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		return catalog, nil
	}, fallback, nil)

	plan, err := router.plan(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("plan() error = %v", err)
	}
	if len(plan) != 1 || plan[0].ModelName != "claude" {
		t.Fatalf("plan = %v, want [claude] (cross-protocol fallback must be skipped)", modelNames(plan))
	}
}

// TestRoutedLLM_AutoSkipsUnconfiguredEnvFallback 未配置的 env fallback
// （零值占位）不应成为候选，避免对空模型名发起无效请求。
func TestRoutedLLM_AutoSkipsUnconfiguredEnvFallback(t *testing.T) {
	catalog := []*LLMRuntimeConfig{openAIModelWithID("m-1", "db-model")}
	router := NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		return catalog, nil
	}, nil, nil) // nil → 内部替换为零值占位配置

	plan, err := router.plan(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("plan() error = %v", err)
	}
	if len(plan) != 1 || plan[0].ModelName != "db-model" {
		t.Fatalf("plan = %v, want [db-model]", modelNames(plan))
	}
}

// TestRoutedLLM_SpecifiedModelIsSticky 用户选定模型时 plan 只含该模型，
// 即使配置了 env fallback 也不追加（失败直接报错，不静默切换）。
func TestRoutedLLM_SpecifiedModelIsSticky(t *testing.T) {
	catalog := []*LLMRuntimeConfig{openAIModelWithID("m-1", "db-model")}
	fallback := &LLMRuntimeConfig{Profile: openAITextProfile(), ModelName: "env-fallback"}
	router := NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		return catalog, nil
	}, fallback, nil)

	ctx := WithModelID(context.Background(), "m-1")
	plan, err := router.plan(ctx, &LLMRequest{})
	if err != nil {
		t.Fatalf("plan() error = %v", err)
	}
	if len(plan) != 1 || plan[0].ModelName != "db-model" {
		t.Fatalf("plan = %v, want [db-model] (sticky)", modelNames(plan))
	}
}

// TestRoutedLLM_UnknownModelIDErrors 选定的 model_id 不在目录中 → 显式报错。
func TestRoutedLLM_UnknownModelIDErrors(t *testing.T) {
	catalog := []*LLMRuntimeConfig{openAIModelWithID("m-1", "db-model")}
	fallback := &LLMRuntimeConfig{Profile: openAITextProfile(), ModelName: "env-fallback"}
	router := NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		return catalog, nil
	}, fallback, nil)

	ctx := WithModelID(context.Background(), "missing")
	if _, err := router.plan(ctx, &LLMRequest{}); !errors.Is(err, ErrModelNotAvailable) {
		t.Fatalf("plan() error = %v, want ErrModelNotAvailable", err)
	}
}
