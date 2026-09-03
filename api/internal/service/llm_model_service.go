package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
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
	// GetDefaultForAgent 启动读取：default 优先，否则第一个 enabled
	// 找不到则 panic (与原"必须配 LLM" 行为一致)
	GetDefaultForAgent(ctx context.Context) *model.LLMModel
}

// DefaultLLMModelService 默认实现
type DefaultLLMModelService struct {
	repo repository.LLMModelRepository
}

// NewLLMModelService 创建多模型服务
func NewLLMModelService(repo repository.LLMModelRepository) LLMModelService {
	return &DefaultLLMModelService{repo: repo}
}

// 业务错误
var (
	ErrModelNotFound     = errors.New("模型不存在")
	ErrModelNameRequired = errors.New("name/provider/base_url/model_name 不能为空")
	ErrModelConflict     = errors.New("同名同 provider+url+model 已存在")
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
			return ErrModelConflict
		}
		if autoSetDefault {
			// 当前表里没有 default 才自动设置
			cur, _ := r.GetDefault(ctx)
			if cur == nil {
				if err := r.ClearDefault(ctx, nil); err != nil {
					return err
				}
				if err := r.SetDefault(ctx, nil, m.ID); err != nil {
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
			if err := r.ClearDefault(ctx, nil); err != nil {
				return err
			}
		}
		if err := r.Update(ctx, m); err != nil {
			return ErrModelConflict
		}
		if m.IsDefault && !old.IsDefault {
			if err := r.SetDefault(ctx, nil, m.ID); err != nil {
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
			if err := r.ClearDefault(ctx, nil); err != nil {
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
		return errors.New("不能将停用的模型设为默认，请先启用")
	}

	return s.repo.WithTx(ctx, func(r repository.LLMModelRepository) error {
		if err := r.ClearDefault(ctx, nil); err != nil {
			return err
		}
		if err := r.SetDefault(ctx, nil, id); err != nil {
			return err
		}
		// 触发部分 unique 索引兜底
		return nil
	})
}

// UnsetDefault 取消默认模型（清空 default 标记，agent 启动时会降级到第一个 enabled）
func (s *DefaultLLMModelService) UnsetDefault(ctx context.Context) error {
	return s.repo.ClearDefault(ctx, nil)
}

// GetDefaultForAgent agent 启动读默认模型
// 找不到 default → 降级到第一个 enabled
// 都没有 → panic（强制要求至少配 1 个 enabled 模型）
func (s *DefaultLLMModelService) GetDefaultForAgent(ctx context.Context) *model.LLMModel {
	def, err := s.repo.GetDefault(ctx)
	if err == nil && def != nil {
		if def.IsEnabled {
			return def
		}
	}
	// 降级
	first, err := s.repo.GetFirstEnabled(ctx)
	if err == nil && first != nil {
		return first
	}
	panic("no LLM model available: please add and enable at least one model in settings")
}

// 防止 pgx 未使用
var _ pgx.Tx
