package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

// ConfigQueryContext 配置查询接口
type ConfigQueryContext interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (r *PostgresAppConfigRepository) queryer() ConfigQueryContext {
	if r.tx != nil {
		return &configTxQueryContext{tx: r.tx}
	}
	return &configPoolQueryContext{pool: r.db.Pool}
}

type configPoolQueryContext struct {
	pool *pgxpool.Pool
}

func (p *configPoolQueryContext) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}
func (p *configPoolQueryContext) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}
func (p *configPoolQueryContext) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

type configTxQueryContext struct {
	tx pgx.Tx
}

func (t *configTxQueryContext) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}
func (t *configTxQueryContext) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}
func (t *configTxQueryContext) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
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

	// 统一规范化为实际 JSON 字节，兼容历史遗留的 base64 编码与多重序列化数据
	config.ConfigValue = normalizeConfigValue(configValueJSON)
	return &config, nil
}

// normalizeConfigValue 将存储的 config_value 规范化为实际 JSON 字节。
// 兼容两种历史数据格式：
//   - 直接存 JSON 对象：{"key":"value"} => 原样返回
//   - 存 base64 编码的 JSON 字符串："eyJi..." => 解码后返回 JSON 字节
func normalizeConfigValue(raw []byte) []byte {
	var v interface{}
	if err := sonic.Unmarshal(raw, &v); err != nil {
		return raw // 非 JSON，原样返回
	}

	if str, ok := v.(string); ok {
		if decoded, err := base64.StdEncoding.DecodeString(str); err == nil && sonic.Valid(decoded) {
			return decoded
		}
		return []byte(str)
	}

	return raw
}

// SaveConfig 保存配置
func (r *PostgresAppConfigRepository) SaveConfig(ctx context.Context, config *model.AppConfig) error {
	q := r.queryer()
	configValueJSON, err := sonic.Marshal(config.ConfigValue)
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
		if err := sonic.Unmarshal(configValueJSON, &c.ConfigValue); err != nil {
			return nil, err
		}
		configs = append(configs, &c)
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
		if err := sonic.Unmarshal(configValueJSON, &c.ConfigValue); err != nil {
			return nil, err
		}
		configs = append(configs, &c)
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
			tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err := fn(&PostgresAppConfigRepository{db: r.db, tx: tx}); err != nil {
		tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
