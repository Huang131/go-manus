package external

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// RuntimeHealthStore 持久化运行时健康快照的最小接口。
type RuntimeHealthStore interface {
	UpdateRuntimeHealth(ctx context.Context, id string, health model.RuntimeHealth) error
}

// ModelCatalogProvider 返回可路由的模型候选列表。
// 控制面把模型画像、请求策略、成本策略都收口到这里，运行时路由器只消费这一层。
type ModelCatalogProvider func(ctx context.Context) ([]*LLMRuntimeConfig, error)

// RoutedLLM 负责主模型选择、fallback、以及最小的路由约束。
// 它不做协议转换，只决定“用哪个模型执行一次请求”。
type RoutedLLM struct {
	catalog  ModelCatalogProvider
	fallback *LLMRuntimeConfig
	factory  LLMClientFactory
	mu       sync.RWMutex
	health   map[string]LLMRuntimeHealth
	store    RuntimeHealthStore
}

// NewRoutedLLM 创建带 fallback 能力的路由器。
func NewRoutedLLM(catalog ModelCatalogProvider, fallback *LLMRuntimeConfig, factory LLMClientFactory) *RoutedLLM {
	if fallback == nil {
		fallback = &LLMRuntimeConfig{
			Profile: llmcore.ModelProfile{Protocol: llmcore.ProtocolOpenAICompat},
		}
	}
	if factory == nil {
		factory = defaultLLMClientFactory
	}
	return &RoutedLLM{
		catalog:  catalog,
		fallback: fallback,
		factory:  factory,
		health:   make(map[string]LLMRuntimeHealth),
	}
}

// SetHealthStore 设置运行时健康持久化目标。
func (r *RoutedLLM) SetHealthStore(store RuntimeHealthStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store = store
}

// NewRoutedLLMFromSingleProvider 兼容只提供单个候选的老入口。
func NewRoutedLLMFromSingleProvider(provider LLMConfigProvider, fallback *LLMRuntimeConfig, factory LLMClientFactory) *RoutedLLM {
	if provider == nil {
		return NewRoutedLLM(nil, fallback, factory)
	}
	return NewRoutedLLM(func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
		cfg, err := provider(ctx)
		if err != nil || cfg == nil {
			return nil, err
		}
		return []*LLMRuntimeConfig{cfg}, nil
	}, fallback, factory)
}

// Invoke 先选主模型，再按严格规则 fallback。
func (r *RoutedLLM) Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	plan := r.plan(ctx, req)
	var lastErr error
	for idx, cfg := range plan {
		if cfg == nil {
			continue
		}
		start := time.Now()
		resp, err := r.factory(cfg).Invoke(ctx, req)
		latency := time.Since(start)
		if err == nil {
			r.RecordSuccess(configKey(cfg), latency)
			r.persistHealth(ctx, cfg)
			return resp, nil
		}
		r.RecordFailure(configKey(cfg), err, latency)
		r.persistHealth(ctx, cfg)
		lastErr = err
		if idx == 0 {
			if pe, ok := err.(*llmcore.ProviderError); ok && pe.Fallbackable && canFallbackAfterToolUse(req) {
				continue
			}
		}
		break
	}
	if lastErr == nil {
		return nil, errors.New("no llm model available")
	}
	return nil, lastErr
}

func (r *RoutedLLM) plan(ctx context.Context, req *LLMRequest) []*LLMRuntimeConfig {
	catalog := []*LLMRuntimeConfig{r.fallback}
	if r.catalog != nil {
		if cfgs, err := r.catalog(ctx); err == nil && len(cfgs) > 0 {
			catalog = cfgs
		}
	}
	r.applyStoredHealth(catalog)
	if len(catalog) == 0 {
		return []*LLMRuntimeConfig{r.fallback}
	}
	sort.SliceStable(catalog, func(i, j int) bool {
		return betterHealth(catalog[i], catalog[j])
	})
	primary := catalog[0]
	plan := []*LLMRuntimeConfig{primary}
	for _, candidate := range catalog[1:] {
		if candidate == nil || primary == nil {
			continue
		}
		if llmcore.CanFallbackTo(candidate.Profile, primary.Profile) {
			plan = append(plan, candidate)
		}
	}
	return plan
}

func betterHealth(a, b *LLMRuntimeConfig) bool {
	ra := healthRank(a)
	rb := healthRank(b)
	if ra != rb {
		return ra < rb
	}
	if a == nil || b == nil {
		return a != nil
	}
	if a.Health.AverageLatencyMS != b.Health.AverageLatencyMS {
		return a.Health.AverageLatencyMS < b.Health.AverageLatencyMS
	}
	if a.Health.RecentFailures != b.Health.RecentFailures {
		return a.Health.RecentFailures < b.Health.RecentFailures
	}
	// 完全相等时保持原始顺序，避免无健康数据时打乱主备模型优先级。
	return false
}

func healthRank(cfg *LLMRuntimeConfig) int {
	if cfg == nil {
		return 3
	}
	switch cfg.Health.Status {
	case LLMHealthHealthy:
		return 0
	case LLMHealthDegraded:
		return 1
	case LLMHealthUnhealthy:
		return 2
	default:
		return 1
	}
}

// RecordSuccess 写回一次成功调用的运行时健康状态。
//
// 这里是轻量级回写，不做复杂统计，只维护：
//   - 最近一次延迟
//   - 失败计数衰减
//   - 健康状态回升
func (r *RoutedLLM) RecordSuccess(modelKey string, latency time.Duration) {
	if modelKey == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	health := r.health[modelKey]
	health.AverageLatencyMS = updateLatencyMS(health.AverageLatencyMS, latency)
	if health.RecentFailures > 0 {
		health.RecentFailures--
	}
	if health.RecentFailures == 0 {
		health.Status = LLMHealthHealthy
	} else {
		health.Status = LLMHealthDegraded
	}
	r.health[modelKey] = health
}

// RecordFailure 写回一次失败调用的运行时健康状态。
func (r *RoutedLLM) RecordFailure(modelKey string, err error, latency time.Duration) {
	if modelKey == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	health := r.health[modelKey]
	health.AverageLatencyMS = updateLatencyMS(health.AverageLatencyMS, latency)
	health.RecentFailures++
	switch {
	case health.RecentFailures >= 3:
		health.Status = LLMHealthUnhealthy
	default:
		health.Status = LLMHealthDegraded
	}
	r.health[modelKey] = health
	_ = err
}

func (r *RoutedLLM) persistHealth(ctx context.Context, cfg *LLMRuntimeConfig) {
	if cfg == nil || cfg.Profile.ID == "" {
		return
	}

	r.mu.RLock()
	health := r.health[configKey(cfg)]
	store := r.store
	r.mu.RUnlock()

	if store == nil {
		return
	}

	_ = store.UpdateRuntimeHealth(ctx, cfg.Profile.ID, runtimeHealthToModel(health))
}

func (r *RoutedLLM) applyStoredHealth(catalog []*LLMRuntimeConfig) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, cfg := range catalog {
		if cfg == nil {
			continue
		}
		if health, ok := r.health[configKey(cfg)]; ok {
			cfg.Health = health
		}
	}
}

func configKey(cfg *LLMRuntimeConfig) string {
	if cfg == nil {
		return ""
	}
	if cfg.Profile.ID != "" {
		return cfg.Profile.ID
	}
	if cfg.ModelName != "" {
		return cfg.ModelName
	}
	if cfg.BaseURL != "" {
		return cfg.BaseURL + "|" + string(cfg.Profile.Protocol)
	}
	return string(cfg.Profile.Protocol)
}

func updateLatencyMS(prev int, latency time.Duration) int {
	ms := int(latency / time.Millisecond)
	if ms <= 0 {
		ms = 1
	}
	if prev <= 0 {
		return ms
	}
	return (prev + ms) / 2
}

func canFallbackAfterToolUse(req *LLMRequest) bool {
	if req == nil {
		return true
	}
	hasExecutedTool := false
	for _, msg := range req.Messages {
		if msg.Role == llmcore.RoleTool {
			hasExecutedTool = true
			break
		}
	}
	if !hasExecutedTool {
		return true
	}
	if len(req.Tools) == 0 {
		return false
	}
	for _, tool := range req.Tools {
		if !tool.ReadOnly {
			return false
		}
	}
	return true
}

func (r *RoutedLLM) ModelName() string {
	if r.fallback == nil {
		return ""
	}
	return r.fallback.ModelName
}

func (r *RoutedLLM) Temperature() float64 {
	if r.fallback == nil {
		return 0
	}
	return r.fallback.Temperature
}

func (r *RoutedLLM) MaxTokens() int {
	if r.fallback == nil {
		return 0
	}
	return r.fallback.MaxTokens
}

func (r *RoutedLLM) String() string {
	return fmt.Sprintf("RoutedLLM(%s)", r.ModelName())
}
