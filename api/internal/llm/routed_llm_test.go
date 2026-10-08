package llm

import (
	"context"
	"errors"
	"fmt"
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

// healthRecorder 把模型 ID 包装为可记录健康的目录配置（有 Profile.ID）。
func healthRecorder(id string) *LLMRuntimeConfig {
	return &LLMRuntimeConfig{Profile: llmcore.ModelProfile{ID: id}}
}

func TestRoutedLLM_InvalidateHealth(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)

	router.RecordFailure(healthRecorder("model-a"), errors.New("boom"), 100*time.Millisecond)
	router.RecordFailure(healthRecorder("model-a"), errors.New("boom"), 120*time.Millisecond)

	if h := healthOfModel(router, "model-a"); h.RecentFailures != 2 || h.Status != model.HealthStateDegraded {
		t.Fatalf("before invalidate: got %+v, want 2 failures/degraded", h)
	}

	router.InvalidateHealth("model-a")

	if h := healthOfModel(router, "model-a"); h.RecentFailures != 0 || h.AverageLatencyMS != 0 {
		t.Fatalf("after invalidate: got %+v, want zero health", h)
	}

	// 失效后重新积累：从零开始，不与旧值混合。
	router.RecordSuccess(healthRecorder("model-a"), 30*time.Millisecond)
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
	router.RecordFailure(healthRecorder("model-a"), errors.New("boom"), 100*time.Millisecond)
	router.RecordFailure(healthRecorder("model-a"), errors.New("boom"), 300*time.Millisecond)
	h := router.GetHealth("model-a")
	if h.RecentFailures != 2 || h.Status != model.HealthStateDegraded {
		t.Fatalf("after failures: got %+v, want 2 failures/degraded", h)
	}
	if h.AverageLatencyMS != 200 {
		t.Fatalf("after failures: latency = %d, want 200", h.AverageLatencyMS)
	}

	// 建流即失败传 0 延迟：不污染 EMA，但计数仍累加并触发 unhealthy 阈值
	router.RecordFailure(healthRecorder("model-a"), errors.New("stream refused"), 0)
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

// P2-6 守护：无 Profile.ID 的配置（env fallback 单例形态，configKey 退化为
// ModelName）不写入健康统计——其快照既不参与排序也无法按 ID 查询，
// 写入只会积累死数据。
func TestRoutedLLM_NoHealthForIDLessConfigs(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)

	// 仅 ModelName 无 ID：不记录
	noID := &LLMRuntimeConfig{Profile: routedTestProfile(), ModelName: "env-fallback"}
	router.RecordSuccess(noID, 10*time.Millisecond)
	router.RecordFailure(noID, errors.New("boom"), 10*time.Millisecond)
	router.RecordFailure(nil, errors.New("boom"), 10*time.Millisecond)

	if h := router.GetHealth("env-fallback"); h.Status != "" || h.RecentFailures != 0 {
		t.Fatalf("id-less config should not be recorded, got %+v", h)
	}
	// 无 ID 条目在 applyStoredHealth 下也不应读出任何健康（key 退化为 ModelName）
	if h := healthOfModel(router, "env-fallback"); h.Status != "" {
		t.Fatalf("id-less config should not affect stored health, got %+v", h)
	}

	// 对照组：有 Profile.ID 的目录模型正常记录
	router.RecordFailure(healthRecorder("catalog-model"), errors.New("boom"), 10*time.Millisecond)
	if h := router.GetHealth("catalog-model"); h.RecentFailures != 1 || h.Status != model.HealthStateDegraded {
		t.Fatalf("catalog model should be recorded, got %+v", h)
	}
}

// P3-4 守护：配置类错误（auth/not_found/bad_request/context_limit）不计入失败计数。
// API key 配错是确定性问题，把模型打成 unhealthy 会让路由永久避开一个
// "配置修好即正常"的模型，且掩盖真实病因。供应商抖动（5xx/限流/网络）照常计数。
func TestRoutedLLM_ConfigErrorsDoNotPolluteHealth(t *testing.T) {
	router := NewRoutedLLM(nil, nil, nil)

	// auth：不计数
	router.RecordFailure(healthRecorder("auth-bad"),
		llmcore.NewProviderError(llmcore.KindAuth, "openai_compat", "m", "401 invalid api key"), 10*time.Millisecond)
	if h := router.GetHealth("auth-bad"); h.RecentFailures != 0 || h.Status != "" {
		t.Fatalf("auth error should not be recorded, got %+v", h)
	}

	// bad_request（4xx 配置类）：不计数
	router.RecordFailure(healthRecorder("bad-req"),
		llmcore.NewProviderError(llmcore.KindBadRequest, "openai_compat", "m", "400 max_tokens must be at least 1"), 10*time.Millisecond)
	if h := router.GetHealth("bad-req"); h.RecentFailures != 0 {
		t.Fatalf("bad_request error should not be recorded, got %+v", h)
	}

	// not_found / context_limit / content_filter：不计数
	router.RecordFailure(healthRecorder("nf"),
		llmcore.NewProviderError(llmcore.KindNotFound, "openai_compat", "m", "404 model not found"), 10*time.Millisecond)
	router.RecordFailure(healthRecorder("ctx"),
		llmcore.NewProviderError(llmcore.KindContextLimit, "openai_compat", "m", "context too long"), 10*time.Millisecond)
	router.RecordFailure(healthRecorder("cf"),
		llmcore.NewProviderError(llmcore.KindContentFilter, "openai_compat", "m", "content filtered"), 10*time.Millisecond)
	if h := router.GetHealth("nf"); h.RecentFailures != 0 {
		t.Fatalf("not_found error should not be recorded, got %+v", h)
	}
	if h := router.GetHealth("ctx"); h.RecentFailures != 0 {
		t.Fatalf("context_limit error should not be recorded, got %+v", h)
	}
	// ContentFilter：模型本身健康，被内容安全拒绝是请求触发的确定性失败
	if h := router.GetHealth("cf"); h.RecentFailures != 0 {
		t.Fatalf("content_filter error should not be recorded, got %+v", h)
	}

	// 包装后的 auth（errors.As 穿透 %w 包装）：不计数
	wrapped := fmt.Errorf("invoke chain: %w",
		llmcore.NewProviderError(llmcore.KindAuth, "openai_compat", "m", "403 forbidden"))
	router.RecordFailure(healthRecorder("auth-wrapped"), wrapped, 10*time.Millisecond)
	if h := router.GetHealth("auth-wrapped"); h.RecentFailures != 0 {
		t.Fatalf("wrapped auth error should not be recorded, got %+v", h)
	}

	// 对照：供应商抖动照常计数（server / rate_limit / network / timeout / 非 ProviderError）
	router.RecordFailure(healthRecorder("srv"),
		llmcore.NewProviderError(llmcore.KindServer, "openai_compat", "m", "502 bad gateway"), 10*time.Millisecond)
	router.RecordFailure(healthRecorder("rl"),
		llmcore.NewProviderError(llmcore.KindRateLimit, "openai_compat", "m", "429 too many requests"), 10*time.Millisecond)
	router.RecordFailure(healthRecorder("plain"), errors.New("plain network error"), 10*time.Millisecond)
	for _, id := range []string{"srv", "rl", "plain"} {
		if h := router.GetHealth(id); h.RecentFailures != 1 || h.Status != model.HealthStateDegraded {
			t.Fatalf("transient error %s should be recorded, got %+v", id, h)
		}
	}
}

// ============================================================================
// Stream 建连失败 fallback（守护测试）
// ============================================================================

// routedTestProfile 本文件自持的标准 OpenAI 兼容文本模型画像，
// 不依赖 fallback_llm_test.go / llm_testing.go 中的助手，保证本文件可独立编译。
func routedTestProfile() llmcore.ModelProfile {
	return llmcore.ModelProfile{
		Protocol: llmcore.ProtocolOpenAICompat,
		Capabilities: model.ModelCapabilities{
			SupportsText:      true,
			SupportsToolCalls: true,
			SupportsStreaming: true,
			MaxContextTokens:  4096,
			MaxOutputTokens:   1024,
		},
	}
}

// routedCatalogModel 构造生产形态的目录条目：DB 目录模型必带 Profile.ID
// （健康统计只记录有 ID 的目录模型，见 RecordSuccess 注释）。
func routedCatalogModel(name string) *LLMRuntimeConfig {
	return &LLMRuntimeConfig{Profile: withProfileID(routedTestProfile(), name), ModelName: name}
}

// withProfileID 返回携带 ID 的画像副本。
func withProfileID(p llmcore.ModelProfile, id string) llmcore.ModelProfile {
	p.ID = id
	return p
}

// streamAttemptStub 可编程流式桩：streamErr 非 nil 时模拟建连失败（Stream 同步返回
// 错误、尚未发送任何增量），否则发送固定增量后关闭。calls 记录候选实际调用顺序。
type streamAttemptStub struct {
	name      string
	calls     *[]string
	streamErr error
	deltas    []llmcore.LLMDelta
}

func (s *streamAttemptStub) Invoke(context.Context, *LLMRequest) (*llmcore.LLMResponse, error) {
	return &llmcore.LLMResponse{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: s.name}}, nil
}

func (s *streamAttemptStub) ModelName() string    { return s.name }
func (s *streamAttemptStub) Temperature() float64 { return 0 }
func (s *streamAttemptStub) MaxTokens() int       { return 0 }

func (s *streamAttemptStub) Stream(context.Context, *LLMRequest) (<-chan llmcore.LLMDelta, error) {
	*s.calls = append(*s.calls, s.name)
	if s.streamErr != nil {
		return nil, s.streamErr
	}
	ch := make(chan llmcore.LLMDelta, len(s.deltas))
	for _, d := range s.deltas {
		ch <- d
	}
	close(ch)
	return ch, nil
}

// 建连失败（同步错误，尚未发送任何增量）应 fallback 到下一候选，
// 而不是直接把 primary 的错误抛给调用方。
func TestRoutedLLM_StreamConnectFailureFallsBack(t *testing.T) {
	var calls []string
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				routedCatalogModel("primary"),
				routedCatalogModel("backup"),
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			if cfg.ModelName == "primary" {
				return &streamAttemptStub{
					name:      "primary",
					calls:     &calls,
					streamErr: llmcore.NewProviderError(llmcore.KindServer, "openai_compat", "primary", "connect refused"),
				}
			}
			return &streamAttemptStub{
				name:   "backup",
				calls:  &calls,
				deltas: []llmcore.LLMDelta{{ContentText: "backup-token"}},
			}
		},
	)

	deltas, err := router.Stream(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	var got []llmcore.LLMDelta
	for d := range deltas {
		got = append(got, d)
	}
	if len(calls) != 2 || calls[0] != "primary" || calls[1] != "backup" {
		t.Fatalf("stream attempts = %v, want [primary backup]", calls)
	}
	if len(got) != 1 || got[0].ContentText != "backup-token" {
		t.Fatalf("deltas = %+v, want backup token", got)
	}
	// 建连失败计入 primary 的健康统计
	if h := router.GetHealth("primary"); h.RecentFailures != 1 || h.Status != model.HealthStateDegraded {
		t.Fatalf("primary health = %+v, want 1 failure/degraded", h)
	}
	// backup 建流成功、流正常结束 → 记为健康
	if h := router.GetHealth("backup"); h.Status != model.HealthStateHealthy {
		t.Fatalf("backup health = %+v, want healthy", h)
	}
}

// 鉴权类错误不可 fallback：直接返回，不消耗备用模型。
func TestRoutedLLM_StreamAuthErrorDoesNotFallback(t *testing.T) {
	var calls []string
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{Profile: routedTestProfile(), ModelName: "primary"},
				{Profile: routedTestProfile(), ModelName: "backup"},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			if cfg.ModelName == "primary" {
				return &streamAttemptStub{
					name:      "primary",
					calls:     &calls,
					streamErr: llmcore.NewProviderError(llmcore.KindAuth, "openai_compat", "primary", "invalid api key"),
				}
			}
			return &streamAttemptStub{name: "backup", calls: &calls, deltas: []llmcore.LLMDelta{{ContentText: "x"}}}
		},
	)

	_, err := router.Stream(context.Background(), &LLMRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindAuth) {
		t.Fatalf("error kind = %v, want KindAuth (err=%v)", err, err)
	}
	if len(calls) != 1 || calls[0] != "primary" {
		t.Fatalf("stream attempts = %v, want only [primary]", calls)
	}
}

// 调用方 ctx 已取消：错误直接返回，不再 fallback，也不污染健康统计。
func TestRoutedLLM_StreamCanceledContextSkipsFallbackAndHealth(t *testing.T) {
	var calls []string
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				routedCatalogModel("primary"),
				routedCatalogModel("backup"),
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			if cfg.ModelName == "primary" {
				return &streamAttemptStub{
					name:      "primary",
					calls:     &calls,
					streamErr: llmcore.NewProviderError(llmcore.KindServer, "openai_compat", "primary", "boom"),
				}
			}
			return &streamAttemptStub{name: "backup", calls: &calls, deltas: []llmcore.LLMDelta{{ContentText: "x"}}}
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := router.Stream(ctx, &LLMRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(calls) != 1 || calls[0] != "primary" {
		t.Fatalf("stream attempts = %v, want only [primary]", calls)
	}
	if h := router.GetHealth("primary"); h.RecentFailures != 0 {
		t.Fatalf("canceled ctx should not pollute health, got %+v", h)
	}
}

// ============================================================================
// 未知健康状态排序（守护测试）
// ============================================================================

// invokeOrderStub 记录调用顺序的非流式桩。
type invokeOrderStub struct {
	name  string
	calls *[]string
}

func (s *invokeOrderStub) Invoke(context.Context, *LLMRequest) (*llmcore.LLMResponse, error) {
	*s.calls = append(*s.calls, s.name)
	return &llmcore.LLMResponse{Message: llmcore.Message{Role: model.RoleAssistant, ContentText: s.name}}, nil
}

func (s *invokeOrderStub) ModelName() string    { return s.name }
func (s *invokeOrderStub) Temperature() float64 { return 0 }
func (s *invokeOrderStub) MaxTokens() int       { return 0 }

// 无健康数据的新模型排最后：即使存量模型已 degraded，也不被新模型抢占 primary；
// 新模型仅作为 fallback 候选在存量模型失败后才被尝试。
func TestRoutedLLM_UnknownHealthRanksAfterDegraded(t *testing.T) {
	var calls []string
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{Profile: routedTestProfile(), ModelName: "fresh"}, // 无健康数据
				{Profile: routedTestProfile(), ModelName: "flaky"}, // 将被标记 degraded
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &invokeOrderStub{name: cfg.ModelName, calls: &calls}
		},
	)
	router.RecordFailure(healthRecorder("flaky"), errors.New("boom"), 50*time.Millisecond)

	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Message.ContentText != "flaky" {
		t.Fatalf("content = %q, want flaky（degraded 优先于未知新模型）", resp.Message.ContentText)
	}
	if len(calls) != 1 || calls[0] != "flaky" {
		t.Fatalf("invoke attempts = %v, want [flaky]", calls)
	}
}

// 场景 C 守护（端到端）：历史执行过写工具（RoleTool.Name=shell），
// 本次请求 tools 被收窄为只读工具。fallback 判定必须禁止切换——
// 旧实现只看"当前 req.Tools 全只读"会误放行，跨模型续接会产生副作用上下文不一致。
func TestRoutedLLM_NoFallbackWhenToolsNarrowedAfterWriteExecution(t *testing.T) {
	var calls []string
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{Profile: routedTestProfile(), ModelName: "primary"},
				{Profile: routedTestProfile(), ModelName: "backup"},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
					calls = append(calls, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "hello"},
			{Role: model.RoleTool, Name: "shell", ToolCallID: "call-1", ContentText: "done"}, // 之前执行过 shell
		},
		// 本次收窄：只声明只读 search，不含 shell
		Tools: []llmcore.ToolSpec{{
			Type:     llmcore.ToolTypeFunction,
			Function: llmcore.ToolSpecFunction{Name: "search"},
			ReadOnly: true,
		}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(calls) != 1 || calls[0] != "primary" {
		t.Fatalf("invoke attempts = %v, want only [primary]（写副作用已发生，禁止 fallback）", calls)
	}
}

// attempts 计数守护：plan[0] 为 nil（目录含空条目）时，真正的首次尝试在下标 1。
// fallback 资格必须跟随尝试序号而不是 plan 下标，否则会错失 fallback。
func TestRoutedLLM_FallbackWorksWhenPlanHeadIsNil(t *testing.T) {
	var calls []string
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				nil, // 目录头部空条目
				{Profile: routedTestProfile(), ModelName: "primary"},
				{Profile: routedTestProfile(), ModelName: "backup"},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
					calls = append(calls, cfg.ModelName)
					if cfg.ModelName == "primary" {
						return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
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
	if len(calls) != 2 || calls[0] != "primary" || calls[1] != "backup" {
		t.Fatalf("invoke attempts = %v, want [primary backup]", calls)
	}
}

// errors.As 守护：ProviderError 被 fmt.Errorf("%w") 包装后 fallback 闸门仍须生效。
// 类型断言 err.(*ProviderError) 遇到包装会静默失效，导致可 fallback 的错误被当成不可切换。
func TestRoutedLLM_FallbackWithWrappedProviderError(t *testing.T) {
	var calls []string
	router := NewRoutedLLM(
		func(context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{Profile: routedTestProfile(), ModelName: "primary"},
				{Profile: routedTestProfile(), ModelName: "backup"},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
					calls = append(calls, cfg.ModelName)
					if cfg.ModelName == "primary" {
						pe := llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
						return nil, fmt.Errorf("invoke chain wrapped: %w", pe)
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
	if len(calls) != 2 || calls[0] != "primary" || calls[1] != "backup" {
		t.Fatalf("invoke attempts = %v, want [primary backup]（包装后的 ProviderError 仍应触发 fallback）", calls)
	}
}
