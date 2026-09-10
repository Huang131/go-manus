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

func testAppConfigRepo(t *testing.T) repository.AppConfigRepository {
	return repository.NewAppConfigRepository(testDB)
}

func TestAppConfigRepo_GetConfig_NotFoundReturnsNilNil(t *testing.T) {
	repo := testAppConfigRepo(t)
	got, err := repo.GetConfig(context.Background(), model.AppConfigTypeLLM, "non-existent-key")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestAppConfigRepo_SaveAndGetConfig_RoundTrip(t *testing.T) {
	repo := testAppConfigRepo(t)
	cfg := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  model.AppConfigTypeLLM,
		ConfigKey:   model.AppConfigKeyDefault,
		ConfigValue: []byte(`{"api_key":"sk-test-key","model":"gpt-4"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.SaveConfig(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.DeleteConfig(context.Background(), cfg.ConfigType, cfg.ConfigKey)
	})

	got, err := repo.GetConfig(context.Background(), model.AppConfigTypeLLM, model.AppConfigKeyDefault)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, cfg.ConfigType, got.ConfigType)
	assert.Equal(t, cfg.ConfigKey, got.ConfigKey)
	assert.NotEmpty(t, got.ConfigValue)
}

func TestAppConfigRepo_SaveConfig_UpdateExisting(t *testing.T) {
	repo := testAppConfigRepo(t)
	cfgType := model.AppConfigTypeLLM
	cfgKey := model.AppConfigKeyDefault

	// 第一次写入
	cfg1 := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  cfgType,
		ConfigKey:   cfgKey,
		ConfigValue: []byte(`{"model":"gpt-3.5"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, repo.SaveConfig(context.Background(), cfg1))
	t.Cleanup(func() { _ = repo.DeleteConfig(context.Background(), cfgType, cfgKey) })

	// 同 key 第二次写入（upsert）
	cfg2 := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  cfgType,
		ConfigKey:   cfgKey,
		ConfigValue: []byte(`{"model":"gpt-4"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, repo.SaveConfig(context.Background(), cfg2))

	// 验证更新成功
	got, err := repo.GetConfig(context.Background(), cfgType, cfgKey)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.JSONEq(t, `{"model":"gpt-4"}`, string(got.ConfigValue.([]byte)))
}

func TestAppConfigRepo_DeleteConfig(t *testing.T) {
	repo := testAppConfigRepo(t)
	cfgType := model.AppConfigTypeLLM
	cfgKey := "delete-test-key"

	cfg := &model.AppConfig{
		ID:          uuid.New().String(),
		ConfigType:  cfgType,
		ConfigKey:   cfgKey,
		ConfigValue: []byte(`{"test":true}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, repo.SaveConfig(context.Background(), cfg))

	err := repo.DeleteConfig(context.Background(), cfgType, cfgKey)
	require.NoError(t, err)

	got, err := repo.GetConfig(context.Background(), cfgType, cfgKey)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestAppConfigRepo_ListConfigs(t *testing.T) {
	repo := testAppConfigRepo(t)
	cfgType := model.AppConfigTypeAgent

	keys := []string{"list-test-key-a", "list-test-key-b"}
	for i, key := range keys {
		cfg := &model.AppConfig{
			ID:          uuid.New().String(),
			ConfigType:  cfgType,
			ConfigKey:   key,
			ConfigValue: []byte(`{"index":` + string(rune('0'+i)) + `}`),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, repo.SaveConfig(context.Background(), cfg))
		t.Cleanup(func() {
			_ = repo.DeleteConfig(context.Background(), cfgType, key)
		})
	}

	list, err := repo.ListConfigs(context.Background(), cfgType)
	require.NoError(t, err)
	found := 0
	for _, c := range list {
		if c.ConfigKey == keys[0] || c.ConfigKey == keys[1] {
			found++
			if _, ok := c.ConfigValue.([]byte); !ok {
				t.Fatalf("ListConfigs() ConfigValue type = %T, want []byte", c.ConfigValue)
			}
		}
	}
	assert.Equal(t, 2, found)
}

func TestAppConfigRepo_WithTx_CommitsOnSuccess(t *testing.T) {
	repo := testAppConfigRepo(t)
	cfgType := model.AppConfigTypeMCP
	cfgKey := "tx-test-key"

	err := repo.WithTx(context.Background(), func(r repository.AppConfigRepository) error {
		return r.SaveConfig(context.Background(), &model.AppConfig{
			ID:          uuid.New().String(),
			ConfigType:  cfgType,
			ConfigKey:   cfgKey,
			ConfigValue: []byte(`{"tx":true}`),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.DeleteConfig(context.Background(), cfgType, cfgKey) })

	got, err := repo.GetConfig(context.Background(), cfgType, cfgKey)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestAppConfigRepo_WithTx_RollsBackOnError(t *testing.T) {
	repo := testAppConfigRepo(t)
	cfgType := model.AppConfigTypeMCP
	cfgKey := "tx-rollback-key"

	err := repo.WithTx(context.Background(), func(r repository.AppConfigRepository) error {
		if err := r.SaveConfig(context.Background(), &model.AppConfig{
			ID:          uuid.New().String(),
			ConfigType:  cfgType,
			ConfigKey:   cfgKey,
			ConfigValue: []byte(`{"should_not_commit":true}`),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}); err != nil {
			return err
		}
		return assert.AnError
	})
	require.Error(t, err)

	got, err := repo.GetConfig(context.Background(), cfgType, cfgKey)
	require.NoError(t, err)
	assert.Nil(t, got, "failed tx should not persist changes")
}
