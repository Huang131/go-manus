package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
)

// MockLLMModelRepository 仓储 mock
type MockLLMModelRepository struct {
	models       map[string]*model.LLMModel
	defaultID    string
	conflict     bool
	listErr      error
	getDefaultOK bool
}

func NewMockLLMModelRepository() *MockLLMModelRepository {
	return &MockLLMModelRepository{models: make(map[string]*model.LLMModel)}
}

func (m *MockLLMModelRepository) Create(ctx context.Context, mm *model.LLMModel) error {
	if m.conflict {
		return errors.New("duplicate key")
	}
	mm.UpdatedAt = time.Now()
	if mm.CreatedAt.IsZero() {
		mm.CreatedAt = mm.UpdatedAt
	}
	m.models[mm.ID] = mm
	return nil
}

func (m *MockLLMModelRepository) Update(ctx context.Context, mm *model.LLMModel) error {
	mm.UpdatedAt = time.Now()
	m.models[mm.ID] = mm
	return nil
}

func (m *MockLLMModelRepository) Delete(ctx context.Context, id string) error {
	delete(m.models, id)
	if m.defaultID == id {
		m.defaultID = ""
	}
	return nil
}

func (m *MockLLMModelRepository) GetByID(ctx context.Context, id string) (*model.LLMModel, error) {
	if mm, ok := m.models[id]; ok {
		return mm, nil
	}
	return nil, nil
}

func (m *MockLLMModelRepository) GetDefault(ctx context.Context) (*model.LLMModel, error) {
	if m.defaultID == "" {
		return nil, nil
	}
	if mm, ok := m.models[m.defaultID]; ok {
		return mm, nil
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

func (m *MockLLMModelRepository) ClearDefault(ctx context.Context, tx pgx.Tx) error {
	for _, mm := range m.models {
		mm.IsDefault = false
	}
	m.defaultID = ""
	return nil
}

func (m *MockLLMModelRepository) SetDefault(ctx context.Context, tx pgx.Tx, id string) error {
	if mm, ok := m.models[id]; ok {
		mm.IsDefault = true
		m.defaultID = id
	}
	return nil
}

func (m *MockLLMModelRepository) WithTx(ctx context.Context, fn func(repo repository.LLMModelRepository) error) error {
	return fn(m)
}

var _ repository.LLMModelRepository = (*MockLLMModelRepository)(nil)

// ====================== 测试 ======================

func TestLLMModelService_Create(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())
	m, err := svc.Create(context.Background(), &model.LLMModel{
		Name: "test", Provider: "openai", BaseURL: "https://x", APIKey: "k",
		ModelName: "gpt-4", Temperature: 0.7, MaxTokens: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == "" {
		t.Error("ID should be assigned")
	}
	if !m.IsDefault {
		t.Error("first model should be auto-default")
	}
}

func TestLLMModelService_Create_Validation(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())
	_, err := svc.Create(context.Background(), &model.LLMModel{Name: "", Provider: "x", BaseURL: "y", ModelName: "z"})
	if err == nil {
		t.Error("should reject empty name")
	}
}

func TestLLMModelService_Delete_Default(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	m, _ := svc.Create(context.Background(), &model.LLMModel{
		Name: "a", Provider: "p", BaseURL: "u", ModelName: "mn",
	})
	err := svc.Delete(context.Background(), m.ID)
	if !errors.Is(err, ErrDeleteDefaultModel) {
		t.Errorf("want ErrDeleteDefaultModel, got %v", err)
	}
}

func TestLLMModelService_Delete_NotDefault(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	first, _ := svc.Create(context.Background(), &model.LLMModel{
		Name: "first", Provider: "p", BaseURL: "u1", ModelName: "m1", IsEnabled: true,
	})
	second, _ := svc.Create(context.Background(), &model.LLMModel{
		Name: "second", Provider: "p", BaseURL: "u2", ModelName: "m2",
		IsDefault: false, IsEnabled: true,
	})
	if !first.IsDefault {
		t.Fatal("first should be default")
	}
	if err := svc.Delete(context.Background(), second.ID); err != nil {
		t.Fatal(err)
	}
}

func TestLLMModelService_SetDefault(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	first, _ := svc.Create(context.Background(), &model.LLMModel{
		Name: "first", Provider: "p", BaseURL: "u1", ModelName: "m1", IsEnabled: true,
	})
	second, _ := svc.Create(context.Background(), &model.LLMModel{
		Name: "second", Provider: "p", BaseURL: "u2", ModelName: "m2", IsEnabled: true,
	})
	if err := svc.SetDefault(context.Background(), second.ID); err != nil {
		t.Fatal(err)
	}
	cur, _ := repo.GetDefault(context.Background())
	if cur == nil || cur.ID != second.ID {
		t.Errorf("default should be second, got %v", cur)
	}
	if first.IsDefault {
		t.Error("first should not be default anymore")
	}
}

func TestLLMModelService_GetDefaultForAgent_DefaultEnabled(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	svc.Create(context.Background(), &model.LLMModel{
		Name: "d", Provider: "p", BaseURL: "u", ModelName: "m", IsDefault: true, IsEnabled: true,
	})
	m := svc.GetDefaultForAgent(context.Background())
	if m.Name != "d" {
		t.Errorf("got %s", m.Name)
	}
}

func TestLLMModelService_GetDefaultForAgent_DefaultDisabledFallback(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	svc.Create(context.Background(), &model.LLMModel{
		Name: "d", Provider: "p", BaseURL: "u", ModelName: "m",
		IsDefault: true, IsEnabled: false, SortOrder: 99,
	})
	svc.Create(context.Background(), &model.LLMModel{
		Name: "fallback", Provider: "p", BaseURL: "u2", ModelName: "m2",
		IsDefault: false, IsEnabled: true, SortOrder: 1,
	})
	m := svc.GetDefaultForAgent(context.Background())
	if m.Name != "fallback" {
		t.Errorf("want fallback, got %s", m.Name)
	}
}

func TestLLMModelService_GetDefaultForAgent_PanicWhenEmpty(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic when no model available")
		}
	}()
	svc.GetDefaultForAgent(context.Background())
}

func TestLLMModelService_Update_KeepAPIKey(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	m, _ := svc.Create(context.Background(), &model.LLMModel{
		Name: "k", Provider: "p", BaseURL: "u", ModelName: "mn", APIKey: "secret-123",
	})
	// 空 APIKey 更新
	_, err := svc.Update(context.Background(), &model.LLMModel{
		ID: m.ID, Name: "k2", Provider: "p", BaseURL: "u", ModelName: "mn", APIKey: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := svc.GetByID(context.Background(), m.ID)
	if updated.APIKey != "secret-123" {
		t.Errorf("APIKey should be preserved, got %s", updated.APIKey)
	}
}
