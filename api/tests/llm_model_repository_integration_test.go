//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLLMModelRepo(t *testing.T) repository.LLMModelRepository {
	return repository.NewLLMModelRepository(testDB)
}

func cleanupLLMModel(t *testing.T, id string) {
	t.Helper()
	ctx, cancel := NewTestContext()
	defer cancel()
	_, err := testDB.Pool.Exec(ctx, "DELETE FROM llm_models WHERE id = $1", id)
	if err != nil {
		t.Logf("清理 llm model %s 失败: %v", id, err)
	}
}

func newTestModel() *model.LLMModel {
	return &model.LLMModel{
		ID:          uuid.New().String(),
		Name:        "test-model-" + uuid.New().String()[:8],
		Provider:    "openai",
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      "test-key",
		ModelName:   "gpt-4o-mini-" + uuid.New().String()[:8],
		Temperature: 0.7,
		MaxTokens:   4096,
		Tags:        []string{"vision", "tools"},
		IsDefault:   false,
		IsEnabled:   true,
		SortOrder:   100,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Capabilities: model.ModelCapabilities{
			SupportsText:                   true,
			SupportsToolCalls:              true,
			SupportsStructuredOutput:       true,
			SupportsJSONMode:               false,
			SupportsStrictStructuredOutput: false,
		},
		RequestPolicy: model.RequestPolicy{
			ReasoningMode: model.ReasoningOff,
		},
		CostPolicy: model.CostPolicy{
			InputPricePerMTokens:  0.15,
			OutputPricePerMTokens: 0.6,
			Currency:              "USD",
		},
		RuntimeHealth: model.RuntimeHealth{
			Status:           model.HealthStateHealthy,
			RecentFailures:   0,
			AverageLatencyMS: 500,
		},
	}
}

// ===== Create =====

func TestLLMModelRepo_CreateAndGetByID(t *testing.T) {
	repo := testLLMModelRepo(t)
	m := newTestModel()

	err := repo.Create(context.Background(), m)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m.ID) })

	got, err := repo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, m.ID, got.ID)
	assert.Equal(t, m.Name, got.Name)
	assert.Equal(t, m.Provider, got.Provider)
	assert.Equal(t, m.ModelName, got.ModelName)
	assert.Equal(t, m.Temperature, got.Temperature)
	assert.Equal(t, m.MaxTokens, got.MaxTokens)
	assert.Equal(t, m.Tags, got.Tags)
	assert.False(t, got.IsDefault)
	assert.True(t, got.IsEnabled)
	assert.Equal(t, m.Capabilities.SupportsToolCalls, got.Capabilities.SupportsToolCalls)
	assert.Equal(t, m.CostPolicy.Currency, got.CostPolicy.Currency)
	assert.Equal(t, m.RuntimeHealth.Status, got.RuntimeHealth.Status)
}

// ===== GetByID Not Found =====

func TestLLMModelRepo_GetByID_NotFoundReturnsNilNil(t *testing.T) {
	repo := testLLMModelRepo(t)

	got, err := repo.GetByID(context.Background(), "non-existent-id")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// ===== Update =====

func TestLLMModelRepo_Update(t *testing.T) {
	repo := testLLMModelRepo(t)
	m := newTestModel()
	m.IsDefault = true
	err := repo.Create(context.Background(), m)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m.ID) })

	// 先清 default，避免 update 时违反 is_default 唯一约束
	_, err = testDB.Pool.Exec(context.Background(), "UPDATE llm_models SET is_default = FALSE WHERE id = $1", m.ID)
	require.NoError(t, err)

	m.Name = "updated-name"
	m.Temperature = 0.9
	m.IsEnabled = false
	m.Capabilities.SupportsText = false
	m.CostPolicy.InputPricePerMTokens = 0.2

	err = repo.Update(context.Background(), m)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "updated-name", got.Name)
	assert.Equal(t, 0.9, got.Temperature)
	assert.False(t, got.IsEnabled)
	assert.False(t, got.Capabilities.SupportsText)
	assert.Equal(t, 0.2, got.CostPolicy.InputPricePerMTokens)
}

// ===== UpdateRuntimeHealth =====

func TestLLMModelRepo_UpdateRuntimeHealth(t *testing.T) {
	repo := testLLMModelRepo(t)
	m := newTestModel()
	err := repo.Create(context.Background(), m)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m.ID) })

	newHealth := model.RuntimeHealth{
		Status:           model.HealthStateDegraded,
		RecentFailures:   5,
		AverageLatencyMS: 3000,
	}
	err = repo.UpdateRuntimeHealth(context.Background(), m.ID, newHealth)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.HealthStateDegraded, got.RuntimeHealth.Status)
	assert.Equal(t, 5, got.RuntimeHealth.RecentFailures)
	assert.Equal(t, 3000, got.RuntimeHealth.AverageLatencyMS)
}

// ===== Delete =====

func TestLLMModelRepo_Delete(t *testing.T) {
	repo := testLLMModelRepo(t)
	m := newTestModel()
	err := repo.Create(context.Background(), m)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), m.ID)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLLMModelRepo_Delete_NotFoundNoError(t *testing.T) {
	repo := testLLMModelRepo(t)

	// pgx Delete Exec 不返回错误，即使行不存在
	err := repo.Delete(context.Background(), "non-existent-id")
	require.NoError(t, err)
}

// ===== GetDefault =====

func TestLLMModelRepo_GetDefault_NoDefaultReturnsNilNil(t *testing.T) {
	repo := testLLMModelRepo(t)

	got, err := repo.GetDefault(context.Background())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLLMModelRepo_GetDefault_ReturnsDefaultModel(t *testing.T) {
	repo := testLLMModelRepo(t)

	// 先清空所有 default
	_, err := testDB.Pool.Exec(context.Background(), "UPDATE llm_models SET is_default = FALSE WHERE is_default = TRUE")
	require.NoError(t, err)

	m := newTestModel()
	m.IsDefault = true
	err = repo.Create(context.Background(), m)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m.ID) })

	got, err := repo.GetDefault(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, m.ID, got.ID)
	assert.True(t, got.IsDefault)
}

// ===== GetFirstEnabled =====

func TestLLMModelRepo_GetFirstEnabled_NoEnabledReturnsNilNil(t *testing.T) {
	repo := testLLMModelRepo(t)

	// 禁用所有模型
	_, err := testDB.Pool.Exec(context.Background(), "UPDATE llm_models SET is_enabled = FALSE")
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := NewTestContext()
		defer cancel()
		_, _ = testDB.Pool.Exec(ctx, "UPDATE llm_models SET is_enabled = TRUE")
	})

	got, err := repo.GetFirstEnabled(context.Background())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLLMModelRepo_GetFirstEnabled_ReturnsFirstBySortOrder(t *testing.T) {
	repo := testLLMModelRepo(t)

	m1 := newTestModel()
	m1.Name = "first-model"
	m1.SortOrder = 1
	err := repo.Create(context.Background(), m1)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m1.ID) })

	m2 := newTestModel()
	m2.Name = "second-model"
	m2.SortOrder = 2
	err = repo.Create(context.Background(), m2)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m2.ID) })

	got, err := repo.GetFirstEnabled(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, m1.ID, got.ID)
}

// ===== List =====

func TestLLMModelRepo_List_Empty(t *testing.T) {
	// 清空所有模型后再测
	ctx, cancel := NewTestContext()
	defer cancel()
	_, err := testDB.Pool.Exec(ctx, "DELETE FROM llm_models")
	require.NoError(t, err)

	repo := testLLMModelRepo(t)
	models, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, models)
}

func TestLLMModelRepo_List_WithData(t *testing.T) {
	repo := testLLMModelRepo(t)

	m1 := newTestModel()
	m1.SortOrder = 1
	err := repo.Create(context.Background(), m1)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m1.ID) })

	m2 := newTestModel()
	m2.SortOrder = 2
	err = repo.Create(context.Background(), m2)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m2.ID) })

	models, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(models), 2)
}

// ===== ClearDefault / SetDefault =====

func TestLLMModelRepo_SetDefault_RoundTrip(t *testing.T) {
	repo := testLLMModelRepo(t)

	// 准备：清空所有 default，创建一个 enabled 的模型
	_, err := testDB.Pool.Exec(context.Background(), "UPDATE llm_models SET is_default = FALSE WHERE is_default = TRUE")
	require.NoError(t, err)

	m := newTestModel()
	m.IsDefault = false
	err = repo.Create(context.Background(), m)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m.ID) })

	// 设置 default
	err = repo.WithTx(context.Background(), func(txRepo repository.LLMModelRepository) error {
		if err := txRepo.ClearDefault(context.Background(), nil); err != nil {
			return err
		}
		return txRepo.SetDefault(context.Background(), nil, m.ID)
	})
	require.NoError(t, err)

	got, err := repo.GetDefault(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, m.ID, got.ID)
	assert.True(t, got.IsDefault)
}

func TestLLMModelRepo_SetDefault_ClearExisting(t *testing.T) {
	repo := testLLMModelRepo(t)

	_, err := testDB.Pool.Exec(context.Background(), "UPDATE llm_models SET is_default = FALSE WHERE is_default = TRUE")
	require.NoError(t, err)

	m1 := newTestModel()
	m1.Name = "should-be-cleared"
	m1.IsDefault = true
	err = repo.Create(context.Background(), m1)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m1.ID) })

	m2 := newTestModel()
	m2.Name = "new-default"
	m2.IsDefault = false
	err = repo.Create(context.Background(), m2)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m2.ID) })

	err = repo.WithTx(context.Background(), func(txRepo repository.LLMModelRepository) error {
		if err := txRepo.ClearDefault(context.Background(), nil); err != nil {
			return err
		}
		return txRepo.SetDefault(context.Background(), nil, m2.ID)
	})
	require.NoError(t, err)

	got1, err := repo.GetByID(context.Background(), m1.ID)
	require.NoError(t, err)
	require.NotNil(t, got1)
	assert.False(t, got1.IsDefault, "旧 default 已被清除")

	got2, err := repo.GetDefault(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, m2.ID, got2.ID)
}

// ===== WithTx =====

func TestLLMModelRepo_WithTx_Commit(t *testing.T) {
	repo := testLLMModelRepo(t)
	m := newTestModel()

	err := repo.WithTx(context.Background(), func(txRepo repository.LLMModelRepository) error {
		return txRepo.Create(context.Background(), m)
	})
	require.NoError(t, err)
	t.Cleanup(func() { cleanupLLMModel(t, m.ID) })

	// 提交后数据应存在
	got, err := repo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, m.ID, got.ID)
}

func TestLLMModelRepo_WithTx_Rollback(t *testing.T) {
	repo := testLLMModelRepo(t)
	m := newTestModel()
	m.ID = "rollback-test-" + uuid.New().String()

	err := repo.WithTx(context.Background(), func(txRepo repository.LLMModelRepository) error {
		if err := txRepo.Create(context.Background(), m); err != nil {
			return err
		}
		return assert.AnError // 故意返回错误触发 rollback
	})
	require.Error(t, err)

	// rollback 后数据不应存在
	got, err := repo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}
