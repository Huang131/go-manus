package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// LLMModelService 多模型服务接口
type LLMModelService interface {
	List(ctx context.Context) ([]*model.LLMModel, error)
	GetByID(ctx context.Context, id string) (*model.LLMModel, error)
	Create(ctx context.Context, m *model.LLMModel) (*model.LLMModel, error)
	Update(ctx context.Context, m *model.LLMModel) (*model.LLMModel, error)
	UpdateRuntimeHealth(ctx context.Context, id string, health model.RuntimeHealth) error
	Delete(ctx context.Context, id string) error
	SetDefault(ctx context.Context, id string) error
	// UnsetDefault 取消默认模型（允许系统处于"无默认"状态，agent 启动时降级到第一个 enabled）
	UnsetDefault(ctx context.Context) error
	// GetDefaultForAgent 启动读取：default 优先，否则第一个 enabled。
	GetDefaultForAgent(ctx context.Context) (*model.LLMModel, error)
	Test(ctx context.Context, m *model.LLMModel) (*model.LLMModelTestResponse, error)
}

// DefaultLLMModelService 默认实现
type DefaultLLMModelService struct {
	repo       repository.LLMModelRepository
	defaultMu  sync.Mutex
	llmFactory external.LLMClientFactory
}

// NewLLMModelService 创建多模型服务
func NewLLMModelService(repo repository.LLMModelRepository) LLMModelService {
	return &DefaultLLMModelService{repo: repo, llmFactory: external.NewLLMClient}
}

// NewLLMModelServiceWithLLMFactory 允许测试注入客户端，避免测试依赖外部网络。
func NewLLMModelServiceWithLLMFactory(repo repository.LLMModelRepository, factory external.LLMClientFactory) LLMModelService {
	if factory == nil {
		factory = external.NewLLMClient
	}
	return &DefaultLLMModelService{repo: repo, llmFactory: factory}
}

// 业务错误
var (
	ErrModelNotFound       = apperr.NotFound("模型不存在")
	ErrModelNameRequired   = apperr.BadRequest("name/provider/base_url/model_name 不能为空")
	ErrModelConflict       = apperr.Conflict("同名同 provider+url+model 已存在")
	ErrModelDisabled       = apperr.FailedPrecondition("不能将停用的模型设为默认，请先启用")
	ErrModelAPIKeyRequired = apperr.BadRequest("API Key 不能为空")
)

// validateRequired 校验必填字段
func validateRequired(m *model.LLMModel) error {
	if strings.TrimSpace(m.Name) == "" ||
		strings.TrimSpace(m.Provider) == "" ||
		strings.TrimSpace(m.BaseURL) == "" ||
		strings.TrimSpace(m.ModelName) == "" {
		return ErrModelNameRequired
	}
	if m.Temperature < 0 || m.Temperature > 2 {
		m.Temperature = 0.7
	}
	if m.MaxTokens <= 0 {
		m.MaxTokens = 8192
	}
	if m.Tags == nil {
		m.Tags = []string{}
	}
	m.Capabilities = model.MergeDefaultCapabilities(m.Capabilities)
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
	if err := validateRequired(m); err != nil {
		return nil, err
	}
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	// 如果没显式设 default，且当前无 default → 自动设为 default
	autoSetDefault := !m.IsDefault

	err := s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		// 检查唯一约束（提前校验，依赖 DB 错误也可）
		// 直接插入，让 unique constraint 兜底
		if err := r.Create(ctx, m); err != nil {
			if isUniqueViolation(err) {
				return ErrModelConflict
			}
			return err
		}
		if autoSetDefault {
			// 当前表里没有 default 才自动设置
			cur, err := r.GetDefault(ctx)
			if err != nil {
				return err
			}
			if cur == nil {
				if err := r.ClearDefault(ctx); err != nil {
					return err
				}
				if err := r.SetDefault(ctx, m.ID); err != nil {
					return err
				}
				m.IsDefault = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Update 更新
func (s *DefaultLLMModelService) Update(ctx context.Context, m *model.LLMModel) (*model.LLMModel, error) {
	// 先查原值
	old, err := s.repo.GetByID(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, ErrModelNotFound
	}
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

	err = s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		if m.IsDefault && !old.IsDefault {
			if err := r.ClearDefault(ctx); err != nil {
				return err
			}
		}
		if err := r.Update(ctx, m); err != nil {
			if isUniqueViolation(err) {
				return ErrModelConflict
			}
			return err
		}
		if m.IsDefault && !old.IsDefault {
			if err := r.SetDefault(ctx, m.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// UpdateRuntimeHealth 仅更新运行时健康快照。
func (s *DefaultLLMModelService) UpdateRuntimeHealth(ctx context.Context, id string, health model.RuntimeHealth) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrModelNotFound
	}
	m.RuntimeHealth = health
	return s.repo.UpdateRuntimeHealth(ctx, id, health)
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
	return s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		// 如果是 default，先清掉 default 标记（事务内原子）
		if m.IsDefault {
			if err := r.ClearDefault(ctx); err != nil {
				return err
			}
		}
		return r.Delete(ctx, id)
	})
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
			return err
		}
		// 触发部分 unique 索引兜底
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
// 找不到 default → 降级到第一个 enabled。
func (s *DefaultLLMModelService) GetDefaultForAgent(ctx context.Context) (*model.LLMModel, error) {
	def, err := s.repo.GetDefault(ctx)
	if err == nil && def != nil {
		if def.IsEnabled {
			return def, nil
		}
	}
	// 降级
	first, err := s.repo.GetFirstEnabled(ctx)
	if err == nil && first != nil {
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
	if err := validateRequired(m); err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.APIKey) == "" {
		return nil, ErrModelAPIKeyRequired
	}

	start := time.Now()
	client := s.llmFactory(external.BuildRuntimeConfigFromModel(m, 15))
	if client == nil {
		return nil, apperr.Unavailable("模型客户端初始化失败")
	}
	resp, err := client.Invoke(ctx, &external.LLMRequest{Messages: []llmcore.Message{
		{Role: llmcore.RoleUser, ContentText: "连接测试：请只回复连接成功。"},
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
