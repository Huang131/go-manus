package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/jackc/pgx/v5"
)

// AppConfigRepository 应用配置仓储接口
type AppConfigRepository interface {
	GetConfig(ctx context.Context, configType model.AppConfigType, configKey string) (*model.AppConfig, error)
	SaveConfig(ctx context.Context, config *model.AppConfig) error
	DeleteConfig(ctx context.Context, configType model.AppConfigType, configKey string) error
	ListConfigs(ctx context.Context, configType model.AppConfigType) ([]*model.AppConfig, error)
	ListAllConfigs(ctx context.Context) ([]*model.AppConfig, error)

	// 事务支持
	WithTx(ctx context.Context, fn func(repo AppConfigRepository) error) error
}

// PostgresAppConfigRepository PostgreSQL 应用配置仓储实现
type PostgresAppConfigRepository struct {
	db *infrastructure.Postgres
	tx pgx.Tx
}

// NewAppConfigRepository 创建应用配置仓储
func NewAppConfigRepository(db *infrastructure.Postgres) AppConfigRepository {
	return &PostgresAppConfigRepository{db: db}
}

// NewAppConfigRepositoryWithTx 创建带事务的配置仓储
func NewAppConfigRepositoryWithTx(tx pgx.Tx) AppConfigRepository {
	return &PostgresAppConfigRepository{tx: tx}
}

func (r *PostgresAppConfigRepository) queryer() queryer {
	return newQueryer(r.db, r.tx)
}

// GetConfig 获取配置
func (r *PostgresAppConfigRepository) GetConfig(ctx context.Context, configType model.AppConfigType, configKey string) (*model.AppConfig, error) {
	q := r.queryer()
	query := `
		SELECT id, config_type, config_key, config_value, created_at, updated_at
		FROM app_configs WHERE config_type = $1 AND config_key = $2
	`
	var config model.AppConfig
	var configValueJSON []byte
	err := q.QueryRow(ctx, query, configType, configKey).Scan(
		&config.ID, &config.ConfigType, &config.ConfigKey,
		&configValueJSON, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // 配置不存在，返回 nil
		}
		return nil, err
	}

	// JSONB 驱动直接返回原始 JSON，服务层统一负责反序列化为具体配置类型。
	config.ConfigValue = configValueJSON
	return &config, nil
}

// SaveConfig 保存配置
func (r *PostgresAppConfigRepository) SaveConfig(ctx context.Context, config *model.AppConfig) error {
	q := r.queryer()
	configValueJSON, err := marshalConfigValue(config.ConfigValue)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO app_configs (id, config_type, config_key, config_value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (config_type, config_key) 
		DO UPDATE SET config_value = $4, updated_at = $6
	`
	_, err = q.Exec(ctx, query,
		config.ID, config.ConfigType, config.ConfigKey,
		configValueJSON, config.CreatedAt, config.UpdatedAt,
	)
	return err
}

// marshalConfigValue 保持 JSON 字节的原始语义，避免 sonic 将 []byte 编码成 base64 字符串。
// 服务层传入的结构体仍由 sonic 负责序列化。
func marshalConfigValue(value any) ([]byte, error) {
	if raw, ok := value.([]byte); ok {
		return raw, nil
	}
	data, err := sonic.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode config value: %w", err)
	}
	return data, nil
}

// DeleteConfig 删除配置
func (r *PostgresAppConfigRepository) DeleteConfig(ctx context.Context, configType model.AppConfigType, configKey string) error {
	q := r.queryer()
	query := `DELETE FROM app_configs WHERE config_type = $1 AND config_key = $2`
	_, err := q.Exec(ctx, query, configType, configKey)
	return err
}

// ListConfigs 获取指定类型的配置列表
func (r *PostgresAppConfigRepository) ListConfigs(ctx context.Context, configType model.AppConfigType) ([]*model.AppConfig, error) {
	q := r.queryer()
	query := `
		SELECT id, config_type, config_key, config_value, created_at, updated_at
		FROM app_configs WHERE config_type = $1
	`
	rows, err := q.Query(ctx, query, configType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*model.AppConfig
	for rows.Next() {
		var c model.AppConfig
		var configValueJSON []byte
		if err := rows.Scan(&c.ID, &c.ConfigType, &c.ConfigKey, &configValueJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.ConfigValue = configValueJSON
		configs = append(configs, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}

// ListAllConfigs 获取所有配置
func (r *PostgresAppConfigRepository) ListAllConfigs(ctx context.Context) ([]*model.AppConfig, error) {
	q := r.queryer()
	query := `
		SELECT id, config_type, config_key, config_value, created_at, updated_at
		FROM app_configs
	`
	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*model.AppConfig
	for rows.Next() {
		var c model.AppConfig
		var configValueJSON []byte
		if err := rows.Scan(&c.ID, &c.ConfigType, &c.ConfigKey, &configValueJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.ConfigValue = configValueJSON
		configs = append(configs, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}

// WithTx 在事务中执行操作
func (r *PostgresAppConfigRepository) WithTx(ctx context.Context, fn func(repo AppConfigRepository) error) error {
	if r.tx != nil {
		return fn(r)
	}
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			logRollbackFailure(ctx, rollbackTx(ctx, tx))
			panic(p)
		}
	}()
	if err := fn(&PostgresAppConfigRepository{db: r.db, tx: tx}); err != nil {
		return joinRollbackError(err, rollbackTx(ctx, tx))
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
