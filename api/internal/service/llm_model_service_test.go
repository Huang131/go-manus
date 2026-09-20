package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestLLMModelService_Create_MapsUniqueViolationToConflict(t *testing.T) {
	repo := NewMockLLMModelRepository()
	repo.conflict = true
	svc := NewLLMModelService(repo)

	_, err := svc.Create(context.Background(), &model.LLMModel{
		Name: "model", Provider: "provider", BaseURL: "https://example.test", ModelName: "m",
	})
	if !errors.Is(err, ErrModelConflict) {
		t.Fatalf("Create() error = %v, want conflict", err)
	}
}

// TestLLMModelService_Update_RowNotFoundMapsToNotFound 校验通过后行已不存在（注入
// pgx.ErrNoRows 模拟并发删除）：事务内 UPDATE 影响 0 行，应映射为 404 而非 500。
func TestLLMModelService_Update_RowNotFoundMapsToNotFound(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	m, err := svc.Create(context.Background(), &model.LLMModel{
		Name: "model", Provider: "provider", BaseURL: "https://example.test", ModelName: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	repo.updateErr = pgx.ErrNoRows
	_, err = svc.Update(context.Background(), &model.LLMModel{
		ID: m.ID, Name: "model", Provider: "provider", BaseURL: "https://example.test", ModelName: "m",
	})
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("Update() error = %v, want ErrModelNotFound", err)
	}
}

// TestLLMModelService_SetDefault_RowNotFoundMapsToNotFound GetByID 校验后行已不存在（注入
// pgx.ErrNoRows 模拟并发删除）：事务内 SetDefault 影响 0 行，应映射为 404，而非静默成功。
func TestLLMModelService_SetDefault_RowNotFoundMapsToNotFound(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	svc.Create(context.Background(), &model.LLMModel{
		Name: "first", Provider: "p", BaseURL: "https://u1", ModelName: "m1", IsEnabled: true,
	})
	second, err := svc.Create(context.Background(), &model.LLMModel{
		Name: "second", Provider: "p", BaseURL: "https://u2", ModelName: "m2", IsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	repo.setDefaultErr = pgx.ErrNoRows
	err = svc.SetDefault(context.Background(), second.ID)
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("SetDefault() error = %v, want ErrModelNotFound", err)
	}
}

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

func TestLLMModelService_Create_PreservesPartialCapabilities(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())
	m, err := svc.Create(context.Background(), &model.LLMModel{
		Name:      "test",
		Provider:  "openai",
		BaseURL:   "https://x",
		ModelName: "gpt-4",
		IsEnabled: true,
		Capabilities: model.ModelCapabilities{
			SupportsVision: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !m.Capabilities.SupportsVision {
		t.Error("SupportsVision should be preserved")
	}
	if m.Capabilities.MaxContextTokens == 0 {
		t.Error("MaxContextTokens should be filled from default")
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
	// 当前业务规则已开放：允许删除 default 模型（agent 启动会降级到第一个 enabled）
	if err := svc.Delete(context.Background(), m.ID); err != nil {
		t.Errorf("Delete(default model) should succeed, got %v", err)
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
	m, err := svc.GetDefaultForAgent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
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
	m, err := svc.GetDefaultForAgent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "fallback" {
		t.Errorf("want fallback, got %s", m.Name)
	}
}

func TestLLMModelService_GetDefaultForAgent_ReturnsUnavailableWhenEmpty(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())
	_, err := svc.GetDefaultForAgent(context.Background())
	if !errors.Is(err, apperr.Unavailable("")) {
		t.Fatalf("err = %v, want unavailable", err)
	}
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

func TestLLMModelService_List_ReturnsModels(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)

	_, err := svc.Create(context.Background(), &model.LLMModel{
		Name: "m1", Provider: "p", BaseURL: "u", ModelName: "mn1",
	})
	if err != nil {
		t.Fatal(err)
	}

	models, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("List() returned %d models, want 1", len(models))
	}
	if models[0].Name != "m1" {
		t.Errorf("List()[0].Name = %q, want m1", models[0].Name)
	}
}

func TestLLMModelService_List_Empty(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())

	models, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 0 {
		t.Fatalf("List() returned %d models, want 0", len(models))
	}
}

func TestLLMModelService_GetByID_NotFound(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())

	_, err := svc.GetByID(context.Background(), "missing-id")
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrModelNotFound", err)
	}
}

func TestLLMModelService_UnsetDefault_ClearsDefaultFlag(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)

	// 创建一个 default 模型
	m, err := svc.Create(context.Background(), &model.LLMModel{
		Name: "d", Provider: "p", BaseURL: "u", ModelName: "mn",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !m.IsDefault {
		t.Fatal("first created model should be default")
	}

	if err := svc.UnsetDefault(context.Background()); err != nil {
		t.Fatal(err)
	}

	// GetDefault 应回到 nil（无 default 模型）
	def, err := repo.GetDefault(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if def != nil {
		t.Fatalf("GetDefault() after UnsetDefault() = %v, want nil", def)
	}
}

// stubRuntimeHealthReader 注入固定健康快照的读取器
type stubRuntimeHealthReader struct {
	health external.LLMRuntimeHealth
}

func (s *stubRuntimeHealthReader) GetHealth(id string) external.LLMRuntimeHealth {
	return s.health
}

func TestLLMModelService_GetRuntimeHealth(t *testing.T) {
	repo := NewMockLLMModelRepository()
	svc := NewLLMModelService(repo)
	m, err := svc.Create(context.Background(), &model.LLMModel{
		Name: "h", Provider: "p", BaseURL: "u", ModelName: "mn",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 未注入 reader：模型存在，返回零值健康
	h, err := svc.GetRuntimeHealth(context.Background(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != "" || h.RecentFailures != 0 || h.AverageLatencyMS != 0 {
		t.Fatalf("without reader: got %+v, want zero health", h)
	}

	// 注入 reader：返回路由器内存中的实时值
	svc.SetRuntimeHealthReader(&stubRuntimeHealthReader{
		health: external.LLMRuntimeHealth{
			Status:           model.HealthStateDegraded,
			RecentFailures:   2,
			AverageLatencyMS: 350,
		},
	})
	h, err = svc.GetRuntimeHealth(context.Background(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != model.HealthStateDegraded || h.RecentFailures != 2 || h.AverageLatencyMS != 350 {
		t.Fatalf("with reader: got %+v, want degraded/2/350", h)
	}

	// 模型不存在：ErrModelNotFound
	_, err = svc.GetRuntimeHealth(context.Background(), "missing-id")
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("err = %v, want ErrModelNotFound", err)
	}
}
