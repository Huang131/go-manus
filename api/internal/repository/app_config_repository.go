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

func (r *PostgresAppConfigRepository) queryer() queryer {
	return newQueryer(r.db, r.tx)
}

// appConfigColumns 是 app_configs 表的标准查询列，集中定义避免各方法重复列举。
const appConfigColumns = `id, config_type, config_key, config_value, created_at, updated_at`

// scanAppConfig 把一行结果映射为 *model.AppConfig，供 QueryRow 与 Query 行迭代共用。
// JSONB 驱动直接返回原始 JSON，服务层统一负责反序列化为具体配置类型。
func scanAppConfig(s rowScanner) (*model.AppConfig, error) {
	var c model.AppConfig
	var configValueJSON []byte
	if err := s.Scan(&c.ID, &c.ConfigType, &c.ConfigKey, &configValueJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.ConfigValue = configValueJSON
	return &c, nil
}

// GetConfig 获取配置
func (r *PostgresAppConfigRepository) GetConfig(ctx context.Context, configType model.AppConfigType, configKey string) (*model.AppConfig, error) {
	q := r.queryer()
	query := `SELECT ` + appConfigColumns + ` FROM app_configs WHERE config_type = $1 AND config_key = $2`
	config, err := scanAppConfig(q.QueryRow(ctx, query, configType, configKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // 配置不存在，返回 nil
	}
	if err != nil {
		return nil, err
	}
	return config, nil
}

// SaveConfig 保存配置
func (r *PostgresAppConfigRepository) SaveConfig(ctx context.Context, config *model.AppConfig) error {
	q := r.queryer()
	configValueJSON, err := marshalConfigValue(config.ConfigValue)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO app_configs (` + appConfigColumns + `)
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
	query := `SELECT ` + appConfigColumns + ` FROM app_configs WHERE config_type = $1`
	rows, err := q.Query(ctx, query, configType)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanAppConfig)
}

// ListAllConfigs 获取所有配置
func (r *PostgresAppConfigRepository) ListAllConfigs(ctx context.Context) ([]*model.AppConfig, error) {
	q := r.queryer()
	query := `SELECT ` + appConfigColumns + ` FROM app_configs`
	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanAppConfig)
}

// WithTx 在事务中执行操作
func (r *PostgresAppConfigRepository) WithTx(ctx context.Context, fn func(repo AppConfigRepository) error) error {
	return runInTx(ctx, r.db, r.tx, func(tx pgx.Tx) AppConfigRepository {
		return &PostgresAppConfigRepository{db: r.db, tx: tx}
	}, fn)
}
