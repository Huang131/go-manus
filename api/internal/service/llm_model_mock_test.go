package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// MockLLMModelRepository 仓储 mock，供所有 llm_model 相关测试共享
type MockLLMModelRepository struct {
	models        map[string]*model.LLMModel
	conflict      bool
	listErr       error
	getDefaultErr error
	createErr     error
	updateErr     error
	setDefaultErr error
}

func NewMockLLMModelRepository() *MockLLMModelRepository {
	return &MockLLMModelRepository{models: make(map[string]*model.LLMModel)}
}

func (m *MockLLMModelRepository) Create(ctx context.Context, mm *model.LLMModel) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.conflict {
		return &pgconn.PgError{Code: "23505"}
	}
	mm.UpdatedAt = time.Now()
	if mm.CreatedAt.IsZero() {
		mm.CreatedAt = mm.UpdatedAt
	}
	m.models[mm.ID] = mm
	return nil
}

func (m *MockLLMModelRepository) Update(ctx context.Context, mm *model.LLMModel) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.models[mm.ID]; !ok {
		return pgx.ErrNoRows
	}
	mm.UpdatedAt = time.Now()
	m.models[mm.ID] = mm
	return nil
}

func (m *MockLLMModelRepository) Delete(ctx context.Context, id string) error {
	delete(m.models, id)
	return nil
}

func (m *MockLLMModelRepository) GetByID(ctx context.Context, id string) (*model.LLMModel, error) {
	if mm, ok := m.models[id]; ok {
		return mm, nil
	}
	return nil, nil
}

func (m *MockLLMModelRepository) GetDefault(ctx context.Context) (*model.LLMModel, error) {
	if m.getDefaultErr != nil {
		return nil, m.getDefaultErr
	}
	for _, mm := range m.models {
		if mm.IsDefault {
			return mm, nil
		}
	}
	return nil, nil
}

func (m *MockLLMModelRepository) GetFirstEnabled(ctx context.Context) (*model.LLMModel, error) {
	for _, mm := range m.models {
		if mm.IsEnabled {
			return mm, nil
		}
	}
	return nil, nil
}

func (m *MockLLMModelRepository) List(ctx context.Context) ([]*model.LLMModel, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]*model.LLMModel, 0, len(m.models))
	for _, mm := range m.models {
		out = append(out, mm)
	}
	return out, nil
}

func (m *MockLLMModelRepository) ClearDefault(ctx context.Context) error {
	for _, mm := range m.models {
		mm.IsDefault = false
	}
	return nil
}

func (m *MockLLMModelRepository) SetDefault(ctx context.Context, id string) error {
	if m.setDefaultErr != nil {
		return m.setDefaultErr
	}
	mm, ok := m.models[id]
	if !ok {
		return pgx.ErrNoRows
	}
	mm.IsDefault = true
	return nil
}

func (m *MockLLMModelRepository) WithTx(ctx context.Context, fn func(repo repository.LLMModelRepository) error) error {
	return fn(m)
}

var _ repository.LLMModelRepository = (*MockLLMModelRepository)(nil)
