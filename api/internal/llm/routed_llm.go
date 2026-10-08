package llm

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

// ModelCatalogProvider 返回可路由的模型候选列表。
// 控制面把模型画像、请求策略、成本策略都收口到这里，运行时路由器只消费这一层。
type ModelCatalogProvider func(ctx context.Context) ([]*LLMRuntimeConfig, error)

// RoutedLLM 负责主模型选择、fallback、以及最小的路由约束。
// 它不做协议转换，只决定“用哪个模型执行一次请求”。
// 健康快照只在内存维护（重启即归零）：路由信号是瞬态属性，不落库。
type RoutedLLM struct {
	catalog  ModelCatalogProvider
	fallback *LLMRuntimeConfig
	factory  LLMClientFactory
	mu       sync.RWMutex
	health   map[string]LLMRuntimeHealth
}

// unhealthyFailureThreshold 连续失败达到该次数即判定模型为 unhealthy。
const unhealthyFailureThreshold = 3

var _ StreamingLLM = (*RoutedLLM)(nil)

// ErrModelNotAvailable 表示请求 ctx 指定的 model_id 在目录中不存在/被禁用。
// 语义上不允许静默降级到其他模型（用户选了什么就用什么，失败要显式报错），
// 调用方可用 errors.Is 识别并映射为 4xx。
var ErrModelNotAvailable = errors.New("requested model not available")

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

// InvalidateHealth 清除模型在内存中的运行时健康快照。
// 配置编辑/删除后调用：模型配置已变，旧快照不再可信（编辑即新模型），
// 后续调用从零开始重新积累。
func (r *RoutedLLM) InvalidateHealth(id string) {
	if id == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.health, id)
}

// GetHealth 读取模型的实时运行健康快照，无记录时返回零值。
// 供 API 层展示实时状态；与路由排序消费同一份内存数据。
func (r *RoutedLLM) GetHealth(id string) LLMRuntimeHealth {
	if id == "" {
		return LLMRuntimeHealth{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.health[id]
}

// Invoke 先选主模型，再按严格规则 fallback。
func (r *RoutedLLM) Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
	plan, err := r.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	attempts := 0
	var lastErr error
	for _, cfg := range plan {
		if cfg == nil {
			continue
		}
		start := time.Now()
		resp, err := r.factory(cfg).Invoke(ctx, req)
		latency := time.Since(start)
		attempts++
		if err == nil {
			r.RecordSuccess(cfg, latency)
			return resp, nil
		}
		// 请求上下文已取消或超时：继续 fallback 无意义（后续候选会在同一 ctx 上立即失败），
		// 且不属供应商故障，不污染健康统计、不再尝试备用模型。
		if ctx.Err() != nil {
			return nil, err
		}
		r.RecordFailure(cfg, err, latency)
		lastErr = err
		// 与 Stream 对齐：仅首次尝试失败、错误可 fallback、工具使用安全时才切换下一候选。
		// 用尝试计数而非 plan 下标判断——plan[0] 为 nil 时真正的首次尝试在更后面的下标。
		if attempts == 1 {
			var pe *llmcore.ProviderError
			if errors.As(err, &pe) && pe.Fallbackable && llmcore.CanFallbackAfterToolUse(req.Messages, req.Tools) {
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

// Stream 选择支持流式能力的模型并透传增量。
// 建连失败（Stream 同步返回 error，此时尚未向调用方发送任何增量）按与 Invoke
// 相同的严格规则 fallback：仅首次尝试、错误可 fallback、且工具使用安全
// （CanFallbackAfterToolUse：无已执行副作用工具，且未声明写工具）。
// 一旦建流成功则不做中途 fallback——增量可能已部分发送，中途换模型会产生内容拼接。
func (r *RoutedLLM) Stream(ctx context.Context, req *LLMRequest) (<-chan llmcore.LLMDelta, error) {
	plan, err := r.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	attempts := 0
	var lastErr error
	for _, cfg := range plan {
		if cfg == nil {
			continue
		}
		client := r.factory(cfg)
		if client == nil {
			continue
		}
		streaming, ok := client.(StreamingLLM)
		if !ok {
			continue
		}
		// 旧配置可能没有能力画像；此时以适配器是否真正实现 streaming 为准。
		if !cfg.Profile.Capabilities.SupportsStreaming && hasCapabilityProfile(cfg) {
			continue
		}
		ch, streamErr := streaming.Stream(ctx, req)
		attempts++
		if streamErr != nil {
			// 调用方取消/超时：后续候选会在同一 ctx 上立即失败，
			// 且不属供应商故障，不污染健康统计、不再 fallback。
			if ctx.Err() != nil {
				return nil, streamErr
			}
			r.RecordFailure(cfg, streamErr, 0)
			// 与 Invoke 对齐：仅首次尝试失败、错误可 fallback、无副作用工具时才切换下一候选
			if attempts == 1 {
				var pe *llmcore.ProviderError
				if errors.As(streamErr, &pe) && pe.Fallbackable && llmcore.CanFallbackAfterToolUse(req.Messages, req.Tools) {
					lastErr = streamErr
					continue
				}
			}
			return nil, streamErr
		}
		if ch == nil {
			err := errors.New("streaming llm returned nil channel")
			r.RecordFailure(cfg, err, 0)
			return nil, err
		}
		return r.trackStream(ctx, cfg, ch), nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	if modelID := ModelIDFromContext(ctx); modelID != "" {
		return nil, fmt.Errorf("no streaming llm model available: 指定模型 %s 不支持流式", modelID)
	}
	return nil, errors.New("no streaming llm model available")
}

func (r *RoutedLLM) trackStream(ctx context.Context, cfg *LLMRuntimeConfig, input <-chan llmcore.LLMDelta) <-chan llmcore.LLMDelta {
	output := make(chan llmcore.LLMDelta)
	go func() {
		start := time.Now()
		defer close(output)
		// 口径说明：这里的 latency 是整个流的消费时长（含下游排空速度），
		// 与 Invoke 的完整生成时长进同一个 AverageLatencyMS EMA。
		// 流式分钟级 vs 非流式秒级，混口径会稀释排序信号——
		// 轻量实现接受此权衡，仅作 tiebreaker 使用，不做精确延迟统计。
		streamErr := ""
		for {
			select {
			case <-ctx.Done():
				// 调用方主动取消不代表供应商故障，不污染健康统计。
				return
			case delta, ok := <-input:
				if !ok {
					latency := time.Since(start)
					if streamErr != "" {
						r.RecordFailure(cfg, errors.New(streamErr), latency)
					} else {
						r.RecordSuccess(cfg, latency)
					}
					return
				}
				// 协议约定：Error delta 表示本次流失败，发送后 channel 会关闭。
				if delta.Error != "" {
					streamErr = delta.Error
				}
				select {
				case output <- delta:
				case <-ctx.Done():
				}
			}
		}
	}()
	return output
}

func hasCapabilityProfile(cfg *LLMRuntimeConfig) bool {
	if cfg == nil {
		return false
	}
	caps := cfg.Profile.Capabilities
	return caps.SupportsText || caps.SupportsToolCalls || caps.SupportsStructuredOutput ||
		caps.SupportsJSONMode || caps.SupportsStrictStructuredOutput || caps.SupportsStreaming ||
		caps.SupportsVision || caps.SupportsReasoning || caps.MaxContextTokens > 0 || caps.MaxOutputTokens > 0
}

func (r *RoutedLLM) plan(ctx context.Context, req *LLMRequest) ([]*LLMRuntimeConfig, error) {
	catalog := []*LLMRuntimeConfig{r.fallback}
	if r.catalog != nil {
		if cfgs, err := r.catalog(ctx); err != nil {
			// 用户指定的模型不可用：显式报错，不允许静默换成其他模型
			if errors.Is(err, ErrModelNotAvailable) {
				return nil, err
			}
			// 目录源暂时不可用（如 DB 抖动）：降级用 env fallback，保持可用性
			logger.Warn("模型目录获取失败，降级使用 fallback 配置", logger.Err(err))
		} else if len(cfgs) > 0 {
			catalog = cfgs
			// 目录条目每次调用都是新构建的对象，注入健康快照不会竞写共享状态。
			// 注意只对目录条目注入：fallback 是共享单例，并发 plan 只持读锁，
			// 对它写 Health 会引入数据竞态（且单候选场景排序无意义）。
			r.applyStoredHealth(catalog)
		}
	}
	if len(catalog) == 0 {
		return []*LLMRuntimeConfig{r.fallback}, nil
	}

	// 请求级 model_id（"会话中途临时切换模型"）不受健康排序影响，强制作为 primary。
	// Auto 语义（业界惯例，对齐 Cursor）：
	//   - ctx 指定 model_id → 用户选定，粘性路由：plan 只含该模型，失败直接报错
	//   - 未指定 model_id（Auto）→ 系统路由，env fallback 作为最后兜底追加在 plan 尾部
	modelID := ModelIDFromContext(ctx)
	if modelID != "" {
		for _, cfg := range catalog {
			if cfg != nil && cfg.Profile.ID == modelID {
				return []*LLMRuntimeConfig{cfg}, nil
			}
		}
		return nil, fmt.Errorf("%w: %s", ErrModelNotAvailable, modelID)
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
	// Auto 路径：env fallback（部署方在配置文件里显式指定的保底模型）作为最后兜底。
	// 追加条件：真实配置过、不重复（按稳定模型名去重）、且主模型非空时须协议/能力兼容。
	if r.fallback != nil && r.fallback.ModelName != "" &&
		!containsModel(plan, r.fallback) {
		if primary == nil || llmcore.CanFallbackTo(r.fallback.Profile, primary.Profile) {
			plan = append(plan, r.fallback)
		}
	}
	return plan, nil
}

// containsModel 判断 plan 中是否已存在与 target 相同的模型（按稳定模型名去重，
// 避免 env fallback 与目录中同名模型被重复调用）。
func containsModel(plan []*LLMRuntimeConfig, target *LLMRuntimeConfig) bool {
	if target == nil || target.ModelName == "" {
		return false
	}
	for _, cfg := range plan {
		if cfg != nil && cfg.ModelName == target.ModelName {
			return true
		}
	}
	return false
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
		return HealthRankUnknown
	}
	switch cfg.Health.Status {
	case model.HealthStateHealthy:
		return 0
	case model.HealthStateDegraded:
		return 1
	case model.HealthStateUnhealthy:
		return 2
	default:
		// 未知状态（新模型、重启后无数据）排最后：新模型不抢占存量流量，
		// 只作为 fallback 候选被尝试；同分时由延迟与稳定排序打破平局。
		return HealthRankUnknown
	}
}

// RecordSuccess 写回一次成功调用的运行时健康状态。
//
// 这里是轻量级回写，不做复杂统计，只维护：
//   - 最近一次延迟
//   - 失败计数衰减
//   - 健康状态回升
//
// 只记录目录模型（有 Profile.ID）：健康快照只注入目录条目（见 applyStoredHealth），
// env fallback 单例的健康数据永不参与排序——写入只会积累无人读取的死数据。
func (r *RoutedLLM) RecordSuccess(cfg *LLMRuntimeConfig, latency time.Duration) {
	if cfg == nil || cfg.Profile.ID == "" {
		return
	}
	modelKey := cfg.Profile.ID
	r.mu.Lock()
	defer r.mu.Unlock()

	health := r.health[modelKey]
	health.AverageLatencyMS = updateLatencyMS(health.AverageLatencyMS, latency)
	if health.RecentFailures > 0 {
		health.RecentFailures--
	}
	if health.RecentFailures == 0 {
		health.Status = model.HealthStateHealthy
	} else {
		health.Status = model.HealthStateDegraded
	}
	r.health[modelKey] = health
}

// RecordFailure 写回一次失败调用的运行时健康状态。
// 同 RecordSuccess：只记录有 Profile.ID 的目录模型。
//
// 配置类错误（auth/not_found/bad_request/context_limit，见 isConfigError）
// 不计入失败计数：API key 配错、模型名写错是确定性问题，重试也不会成功，
// 把这类模型打成 unhealthy 会让路由永久避开一个"配置修好即正常"的模型，
// 且掩盖真实病因（排障时看到的是降权，而不是配置错误本身）。
func (r *RoutedLLM) RecordFailure(cfg *LLMRuntimeConfig, err error, latency time.Duration) {
	if cfg == nil || cfg.Profile.ID == "" {
		return
	}
	if isConfigError(err) {
		return
	}
	modelKey := cfg.Profile.ID
	r.mu.Lock()
	defer r.mu.Unlock()

	health := r.health[modelKey]
	// latency<=0 表示本次失败没有可用耗时（如 Stream 建流即失败），
	// 跳过 EMA 更新，避免被钳到 1ms 后把失败模型显得"更快"。
	if latency > 0 {
		health.AverageLatencyMS = updateLatencyMS(health.AverageLatencyMS, latency)
	}
	health.RecentFailures++
	switch {
	case health.RecentFailures >= unhealthyFailureThreshold:
		health.Status = model.HealthStateUnhealthy
	default:
		health.Status = model.HealthStateDegraded
	}
	r.health[modelKey] = health
}

// isConfigError 判断错误是否为配置/语义类问题（确定性失败，非供应商抖动）。
// 委托 llmcore.IsDeterministicKind 单一事实源——与 ProviderError.Fallbackable
// 共用同一份清单，新增 Kind 时两个决策自动保持一致。
func isConfigError(err error) bool {
	var pe *llmcore.ProviderError
	if !errors.As(err, &pe) {
		return false
	}
	return llmcore.IsDeterministicKind(pe.Kind)
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
