package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// HealthInvalidator 失效路由器内存健康缓存的最小接口。
// *llm.RoutedLLM 实现了它；编辑/删除模型后调用，使“编辑即新模型”立即生效。
type HealthInvalidator interface {
	InvalidateHealth(id string)
}

// RuntimeHealthReader 读取路由器内存中的实时健康快照的最小接口。
// *llm.RoutedLLM 实现了它；API 层展示实时健康状态时调用。
type RuntimeHealthReader interface {
	GetHealth(id string) llm.LLMRuntimeHealth
}

// LLMModelService 多模型服务接口
type LLMModelService interface {
	List(ctx context.Context) ([]*model.LLMModel, error)
	GetByID(ctx context.Context, id string) (*model.LLMModel, error)
	Create(ctx context.Context, m *model.LLMModel) (*model.LLMModel, error)
	Update(ctx context.Context, m *model.LLMModel) (*model.LLMModel, error)
	Delete(ctx context.Context, id string) error
	SetDefault(ctx context.Context, id string) error
	// 取消默认模型（允许系统处于"无默认"状态，agent 启动时降级到第一个 enabled）
	UnsetDefault(ctx context.Context) error
	// 启动读取：default 优先，否则第一个 enabled。
	GetDefaultForAgent(ctx context.Context) (*model.LLMModel, error)
	Test(ctx context.Context, m *model.LLMModel) (*model.LLMModelTestResponse, error)
	// 读取模型的实时运行健康快照（路由器内存数据，重启归零）。
	GetRuntimeHealth(ctx context.Context, id string) (*model.RuntimeHealth, error)
	// 注入路由器内存健康缓存失效器（编辑/删除模型时触发）。
	SetHealthInvalidator(inv HealthInvalidator)
	// 注入路由器实时健康读取器（bootstrap 创建 RoutedLLM 后调用）。
	SetRuntimeHealthReader(reader RuntimeHealthReader)
}

// DefaultLLMModelService 默认实现
type DefaultLLMModelService struct {
	repo              repository.LLMModelRepository
	defaultMu         sync.Mutex
	llmFactory        llm.LLMClientFactory
	healthInvalidator HealthInvalidator
	healthReader      RuntimeHealthReader
}

// NewLLMModelService 创建多模型服务
func NewLLMModelService(repo repository.LLMModelRepository) LLMModelService {
	return &DefaultLLMModelService{repo: repo, llmFactory: llm.NewLLMClient}
}

// NewLLMModelServiceWithLLMFactory 允许测试注入客户端，避免测试依赖外部网络。
func NewLLMModelServiceWithLLMFactory(repo repository.LLMModelRepository, factory llm.LLMClientFactory) LLMModelService {
	if factory == nil {
		factory = llm.NewLLMClient
	}
	return &DefaultLLMModelService{repo: repo, llmFactory: factory}
}

// SetHealthInvalidator 注入运行时健康缓存失效器。
// bootstrap 在创建 RoutedLLM 后调用：编辑/删除模型时同步清掉内存健康快照。
func (s *DefaultLLMModelService) SetHealthInvalidator(inv HealthInvalidator) {
	s.healthInvalidator = inv
}

// invalidateHealth 失效指定模型的内存健康快照（未注入失效器时为 no-op）。
func (s *DefaultLLMModelService) invalidateHealth(id string) {
	if s.healthInvalidator != nil {
		s.healthInvalidator.InvalidateHealth(id)
	}
}

// SetRuntimeHealthReader 注入路由器实时健康读取器。
// bootstrap 在创建 RoutedLLM 后调用：API 层据此返回内存中的实时健康状态。
func (s *DefaultLLMModelService) SetRuntimeHealthReader(reader RuntimeHealthReader) {
	s.healthReader = reader
}

// GetRuntimeHealth 读取模型的实时运行健康快照。
// 健康数据只存路由器内存（重启归零）：模型无调用记录时返回零值，前端据此显示"暂无数据"。
func (s *DefaultLLMModelService) GetRuntimeHealth(ctx context.Context, id string) (*model.RuntimeHealth, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrModelNotFound
	}
	health := model.RuntimeHealth{}
	if s.healthReader != nil {
		h := s.healthReader.GetHealth(id)
		health.Status = h.Status
		health.RecentFailures = h.RecentFailures
		health.AverageLatencyMS = h.AverageLatencyMS
	}
	return &health, nil
}

// 业务错误
var (
	ErrModelNotFound       = apperr.NotFound("模型不存在")
	ErrModelNameRequired   = apperr.BadRequest("name/provider/base_url/model_name 不能为空")
	ErrModelConflict       = apperr.Conflict("同名同 provider+url+model 已存在")
	ErrModelDisabled       = apperr.FailedPrecondition("不能将停用的模型设为默认，请先启用")
	ErrModelAPIKeyRequired = apperr.BadRequest("API Key 不能为空")
)

// 模型默认值与温度边界。
const (
	defaultTemperature = 0.7
	defaultMaxTokens   = 8192
	temperatureMin     = 0.0
	temperatureMax     = 2.0

	// pgUniqueViolationCode 是 PostgreSQL 唯一约束冲突的 SQLSTATE 码。
	pgUniqueViolationCode = "23505"
)

// normalizeDefaults 归一化缺省/越界字段为默认值（会修改入参）。
// 与 validateRequired 分离：本函数负责"写默认值"，校验函数只读、命名不再误导。
func normalizeDefaults(m *model.LLMModel) {
	if m.Temperature < temperatureMin || m.Temperature > temperatureMax {
		m.Temperature = defaultTemperature
	}
	if m.MaxTokens <= 0 {
		m.MaxTokens = defaultMaxTokens
	}
	if m.Tags == nil {
		m.Tags = []string{}
	}
	m.Capabilities = model.MergeDefaultCapabilities(m.Capabilities)
}

// validateRequired 校验必填字段（只读，不修改入参）。
func validateRequired(m *model.LLMModel) error {
	if strings.TrimSpace(m.Name) == "" ||
		strings.TrimSpace(m.Provider) == "" ||
		strings.TrimSpace(m.BaseURL) == "" ||
		strings.TrimSpace(m.ModelName) == "" {
		return ErrModelNameRequired
	}
	return nil
}

// List 列表
func (s *DefaultLLMModelService) List(ctx context.Context) ([]*model.LLMModel, error) {
	return s.repo.List(ctx)
}

// GetByID 查单条
func (s *DefaultLLMModelService) GetByID(ctx context.Context, id string) (*model.LLMModel, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrModelNotFound
	}
	return m, nil
}

// Create 新增
func (s *DefaultLLMModelService) Create(ctx context.Context, m *model.LLMModel) (*model.LLMModel, error) {
	normalizeDefaults(m)
	if err := validateRequired(m); err != nil {
		return nil, err
	}
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	// 决定/切换 default 涉及"读当前 default + 清旧 + 插入"，整体持锁串行化，
	// 避免并发都读到"无 default"而双双设 default 触发唯一索引冲突。
	s.defaultMu.Lock()
	defer s.defaultMu.Unlock()

	err := s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		if !m.IsDefault {
			// 未显式设 default：库里无 default 时自动成为 default
			cur, err := r.GetDefault(ctx)
			if err != nil {
				return err
			}
			m.IsDefault = cur == nil
		}
		if m.IsDefault {
			// 成为 default 前先清掉旧的，避免违反 partial unique index
			if err := r.ClearDefault(ctx); err != nil {
				return err
			}
		}
		if err := r.Create(ctx, m); err != nil {
			if isUniqueViolation(err) {
				return ErrModelConflict
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Update 更新（编辑即新模型：失效健康缓存，后续调用从零重新积累）
func (s *DefaultLLMModelService) Update(ctx context.Context, m *model.LLMModel) (*model.LLMModel, error) {
	// 先查原值，确保存在且用于对比 IsDefault 变化
	old, err := s.repo.GetByID(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, ErrModelNotFound
	}
	normalizeDefaults(m)
	if err := validateRequired(m); err != nil {
		return nil, err
	}
	// api_key 为空则保留旧值
	if m.APIKey == "" {
		m.APIKey = old.APIKey
	}
	// 保留 sort_order / created_at
	m.CreatedAt = old.CreatedAt
	m.SortOrder = old.SortOrder
	m.UpdatedAt = time.Now()

	// 显式设为 default 时（false→true）加锁，避免与 ClearDefault 竞态。
	switchDefault := m.IsDefault && !old.IsDefault
	if switchDefault {
		s.defaultMu.Lock()
		defer s.defaultMu.Unlock()
	}

	err = s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		if switchDefault {
			if err := r.ClearDefault(ctx); err != nil {
				return err
			}
		}
		if err := r.Update(ctx, m); err != nil {
			if isUniqueViolation(err) {
				return ErrModelConflict
			}
			if errors.Is(err, pgx.ErrNoRows) {
				// UPDATE 影响 0 行：校验通过后模型被并发删除
				return ErrModelNotFound
			}
			return err
		}
		if switchDefault {
			if err := r.SetDefault(ctx, m.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 编辑即新模型：事务成功后失效内存健康缓存，后续调用从零重新积累。
	s.invalidateHealth(m.ID)
	return m, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode
}

// Delete 删除（允许删除默认模型；agent 启动时会降级到第一个 enabled）
func (s *DefaultLLMModelService) Delete(ctx context.Context, id string) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrModelNotFound
	}
	// 删除 default 模型时，ClearDefault 需与 SetDefault 串行化，
	// 否则并发 SetDefault 会把刚设置的新 default 误清。
	if m.IsDefault {
		s.defaultMu.Lock()
		defer s.defaultMu.Unlock()
	}
	if err := s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		// 如果是 default，先清掉 default 标记（事务内原子）
		if m.IsDefault {
			if err := r.ClearDefault(ctx); err != nil {
				return err
			}
		}
		return r.Delete(ctx, id)
	}); err != nil {
		return err
	}
	// 模型删除成功后：同步失效内存健康缓存，避免路由器残留已删除模型的 entry。
	s.invalidateHealth(id)
	return nil
}

// SetDefault 切换默认（事务内原子操作）
func (s *DefaultLLMModelService) SetDefault(ctx context.Context, id string) error {
	target, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrModelNotFound
	}
	if target.IsDefault {
		return nil
	}
	if !target.IsEnabled {
		return ErrModelDisabled
	}

	// 单进程内串行切换，避免并发请求同时清空并设置默认模型。
	s.defaultMu.Lock()
	defer s.defaultMu.Unlock()

	return s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		if err := r.ClearDefault(ctx); err != nil {
			return err
		}
		if err := r.SetDefault(ctx, id); err != nil {
			if isUniqueViolation(err) {
				return ErrModelConflict
			}
			if errors.Is(err, pgx.ErrNoRows) {
				// SetDefault 影响 0 行：GetByID 校验后模型被并发删除
				return ErrModelNotFound
			}
			return err
		}
		return nil
	})
}

// UnsetDefault 取消默认模型（清空 default 标记，agent 启动时会降级到第一个 enabled）
func (s *DefaultLLMModelService) UnsetDefault(ctx context.Context) error {
	// 与 SetDefault 持同一把锁：并发 Set+Unset 才不会把刚设好的 default 清掉
	s.defaultMu.Lock()
	defer s.defaultMu.Unlock()
	return s.repo.ClearDefault(ctx)
}

// GetDefaultForAgent agent 启动读默认模型。
// 找不到 default（或被禁用）→ 降级到第一个 enabled。
func (s *DefaultLLMModelService) GetDefaultForAgent(ctx context.Context) (*model.LLMModel, error) {
	def, err := s.repo.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	if def != nil && def.IsEnabled {
		return def, nil
	}
	// 降级
	first, err := s.repo.GetFirstEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if first != nil {
		return first, nil
	}
	return nil, apperr.Unavailable("no LLM model available: please add and enable at least one model in settings")
}

// Test 使用临时配置发起最小文本请求，验证地址、密钥和模型是否可用。
// 该调用不保存配置，也不携带工具或结构化输出约束，避免把配置测试误判为业务调用。
func (s *DefaultLLMModelService) Test(ctx context.Context, m *model.LLMModel) (*model.LLMModelTestResponse, error) {
	if m == nil {
		return nil, ErrModelNameRequired
	}
	normalizeDefaults(m)
	if err := validateRequired(m); err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.APIKey) == "" {
		return nil, ErrModelAPIKeyRequired
	}

	start := time.Now()
	client := s.llmFactory(llm.BuildRuntimeConfigFromModel(m, 15))
	if client == nil {
		return nil, apperr.Unavailable("模型客户端初始化失败")
	}
	resp, err := client.Invoke(ctx, &llm.LLMRequest{Messages: []llmcore.Message{
		{Role: model.RoleUser, ContentText: "连接测试：请只回复连接成功。"},
	}})
	if err != nil {
		return nil, mapModelTestError(err)
	}
	if resp == nil {
		return nil, apperr.Unavailable("模型未返回响应")
	}
	return &model.LLMModelTestResponse{
		ModelName: m.ModelName,
		LatencyMS: time.Since(start).Milliseconds(),
		Content:   truncateModelTestContent(resp.Message.ContentText),
	}, nil
}

func truncateModelTestContent(content string) string {
	const maxRunes = 500
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes]) + "..."
}

func mapModelTestError(err error) error {
	var providerErr *llmcore.ProviderError
	if !errors.As(err, &providerErr) {
		return apperr.Wrap(apperr.KindUnavailable, "模型连接失败", err)
	}
	msg := providerErr.Message
	if msg == "" {
		msg = "模型连接失败"
	}
	switch providerErr.Kind {
	case llmcore.KindAuth:
		return apperr.Unauthorized("模型认证失败: " + msg)
	case llmcore.KindNotFound:
		return apperr.NotFound("模型不存在: " + msg)
	case llmcore.KindBadRequest:
		return apperr.BadRequest("模型请求不兼容: " + msg)
	case llmcore.KindRateLimit, llmcore.KindNetwork, llmcore.KindTimeout, llmcore.KindServer:
		return apperr.Wrap(apperr.KindUnavailable, "模型暂时不可用: "+msg, err)
	default:
		return apperr.Wrap(apperr.KindInternal, "模型连接失败: "+msg, err)
	}
}
