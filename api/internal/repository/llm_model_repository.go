package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
)

// LLMModelRepository 多模型仓储
type LLMModelRepository interface {
	Create(ctx context.Context, m *model.LLMModel) error
	Update(ctx context.Context, m *model.LLMModel) error
	UpdateRuntimeHealth(ctx context.Context, id string, health model.RuntimeHealth) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.LLMModel, error)
	GetDefault(ctx context.Context) (*model.LLMModel, error)
	GetFirstEnabled(ctx context.Context) (*model.LLMModel, error)
	List(ctx context.Context) ([]*model.LLMModel, error)
	ClearDefault(ctx context.Context, tx pgx.Tx) error
	SetDefault(ctx context.Context, tx pgx.Tx, id string) error
	WithTx(ctx context.Context, fn func(repo LLMModelRepository) error) error
}

// PostgresLLMModelRepository PostgreSQL 实现
type PostgresLLMModelRepository struct {
	db *infrastructure.Postgres
	tx pgx.Tx
}

// NewLLMModelRepository 创建多模型仓储
func NewLLMModelRepository(db *infrastructure.Postgres) LLMModelRepository {
	return &PostgresLLMModelRepository{db: db}
}

// NewLLMModelRepositoryWithTx 创建带事务的仓储
func NewLLMModelRepositoryWithTx(tx pgx.Tx) LLMModelRepository {
	return &PostgresLLMModelRepository{tx: tx}
}

// llmModelQueryer 统一 tx / pool
type llmModelQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (r *PostgresLLMModelRepository) queryer() llmModelQueryer {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

// 阶段 0 新增：capabilities/request_policy/cost_policy/runtime_health 四个 JSONB 列
const llmModelColumns = `id, name, provider, base_url, api_key, model_name,
	temperature, max_tokens, tags, is_default, is_enabled, sort_order,
	capabilities, request_policy, cost_policy, runtime_health,
	created_at, updated_at`

func scanLLMModel(row pgx.Row, m *model.LLMModel) error {
	var tagsJSON, capJSON, reqJSON, costJSON, healthJSON []byte
	if err := row.Scan(
		&m.ID, &m.Name, &m.Provider, &m.BaseURL, &m.APIKey, &m.ModelName,
		&m.Temperature, &m.MaxTokens, &tagsJSON, &m.IsDefault, &m.IsEnabled,
		&m.SortOrder, &capJSON, &reqJSON, &costJSON, &healthJSON,
		&m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return err
	}
	if len(tagsJSON) > 0 {
		if err := sonic.Unmarshal(tagsJSON, &m.Tags); err != nil {
			return fmt.Errorf("decode model tags: %w", err)
		}
	}
	if m.Tags == nil {
		m.Tags = []string{}
	}
	if len(capJSON) > 0 {
		if err := sonic.Unmarshal(capJSON, &m.Capabilities); err != nil {
			return fmt.Errorf("decode model capabilities: %w", err)
		}
	}
	if len(reqJSON) > 0 {
		if err := sonic.Unmarshal(reqJSON, &m.RequestPolicy); err != nil {
			return fmt.Errorf("decode model request policy: %w", err)
		}
	}
	if len(costJSON) > 0 {
		if err := sonic.Unmarshal(costJSON, &m.CostPolicy); err != nil {
			return fmt.Errorf("decode model cost policy: %w", err)
		}
	}
	if len(healthJSON) > 0 {
		if err := sonic.Unmarshal(healthJSON, &m.RuntimeHealth); err != nil {
			return fmt.Errorf("decode model runtime health: %w", err)
		}
	}
	return nil
}

// capabilitiesToJSON / requestPolicyToJSON / costPolicyToJSON / runtimeHealthToJSON
// 用法：写入 DB 前调用，返回 []byte 给 pgx
func modelJSON(field string, value any) ([]byte, error) {
	b, err := sonic.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode model %s: %w", field, err)
	}
	return b, nil
}

// runtimeHealthToJSON 保留仓储内部测试和已有调用的便捷封装。
func runtimeHealthToJSON(h model.RuntimeHealth) ([]byte, error) {
	return modelJSON("runtime health", h)
}

func modelJSONFields(m *model.LLMModel) ([][]byte, error) {
	values := []struct {
		name  string
		value any
	}{
		{"tags", m.Tags}, {"capabilities", m.Capabilities},
		{"request policy", m.RequestPolicy}, {"cost policy", m.CostPolicy},
		{"runtime health", m.RuntimeHealth},
	}
	encoded := make([][]byte, 0, len(values))
	for _, value := range values {
		b, err := modelJSON(value.name, value.value)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, b)
	}
	return encoded, nil
}

// Create 新增
func (r *PostgresLLMModelRepository) Create(ctx context.Context, m *model.LLMModel) error {
	fields, err := modelJSONFields(m)
	if err != nil {
		return err
	}
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	_, err = r.queryer().Exec(ctx, `
		INSERT INTO llm_models (`+llmModelColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`,
		m.ID, m.Name, m.Provider, m.BaseURL, m.APIKey, m.ModelName,
		m.Temperature, m.MaxTokens, fields[0], m.IsDefault, m.IsEnabled,
		m.SortOrder,
		fields[1], fields[2], fields[3], fields[4],
		m.CreatedAt, m.UpdatedAt,
	)
	return err
}

// Update 更新
func (r *PostgresLLMModelRepository) Update(ctx context.Context, m *model.LLMModel) error {
	fields, err := modelJSONFields(m)
	if err != nil {
		return err
	}
	m.UpdatedAt = time.Now()
	_, err = r.queryer().Exec(ctx, `
		UPDATE llm_models SET
			name = $2, provider = $3, base_url = $4, api_key = $5, model_name = $6,
			temperature = $7, max_tokens = $8, tags = $9, is_default = $10,
			is_enabled = $11, sort_order = $12,
			capabilities = $13, request_policy = $14, cost_policy = $15, runtime_health = $16,
			updated_at = $17
	WHERE id = $1
	`,
		m.ID, m.Name, m.Provider, m.BaseURL, m.APIKey, m.ModelName,
		m.Temperature, m.MaxTokens, fields[0], m.IsDefault, m.IsEnabled,
		m.SortOrder,
		fields[1], fields[2], fields[3], fields[4],
		m.UpdatedAt,
	)
	return err
}

// UpdateRuntimeHealth 只更新运行时健康快照，避免健康打点覆盖模型配置。
func (r *PostgresLLMModelRepository) UpdateRuntimeHealth(ctx context.Context, id string, health model.RuntimeHealth) error {
	healthJSON, err := modelJSON("runtime health", health)
	if err != nil {
		return err
	}
	_, err = r.queryer().Exec(ctx, `
		UPDATE llm_models
		SET runtime_health = $2, updated_at = $3
		WHERE id = $1
	`, id, healthJSON, time.Now())
	return err
}

// Delete 删除
func (r *PostgresLLMModelRepository) Delete(ctx context.Context, id string) error {
	_, err := r.queryer().Exec(ctx, `DELETE FROM llm_models WHERE id = $1`, id)
	return err
}

// GetByID 查单条
func (r *PostgresLLMModelRepository) GetByID(ctx context.Context, id string) (*model.LLMModel, error) {
	var m model.LLMModel
	row := r.queryer().QueryRow(ctx, `SELECT `+llmModelColumns+` FROM llm_models WHERE id = $1`, id)
	if err := scanLLMModel(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// GetDefault 查 default 模型
func (r *PostgresLLMModelRepository) GetDefault(ctx context.Context) (*model.LLMModel, error) {
	var m model.LLMModel
	row := r.queryer().QueryRow(ctx,
		`SELECT `+llmModelColumns+` FROM llm_models WHERE is_default = TRUE LIMIT 1`)
	if err := scanLLMModel(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// GetFirstEnabled 第一个启用的模型（降级用）
func (r *PostgresLLMModelRepository) GetFirstEnabled(ctx context.Context) (*model.LLMModel, error) {
	var m model.LLMModel
	row := r.queryer().QueryRow(ctx,
		`SELECT `+llmModelColumns+` FROM llm_models WHERE is_enabled = TRUE
		 ORDER BY sort_order ASC, created_at ASC LIMIT 1`)
	if err := scanLLMModel(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// List 列出所有
func (r *PostgresLLMModelRepository) List(ctx context.Context) ([]*model.LLMModel, error) {
	rows, err := r.queryer().Query(ctx,
		`SELECT `+llmModelColumns+` FROM llm_models ORDER BY is_default DESC, sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.LLMModel
	for rows.Next() {
		var m model.LLMModel
		if err := scanLLMModel(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ClearDefault 清空所有 default（事务内调用，tx 可传 nil 用 pool）
func (r *PostgresLLMModelRepository) ClearDefault(ctx context.Context, tx pgx.Tx) error {
	q := llmModelQueryer(tx)
	if q == nil {
		q = r.queryer()
	}
	_, err := q.Exec(ctx, `UPDATE llm_models SET is_default = FALSE, updated_at = $1 WHERE is_default = TRUE`, time.Now())
	return err
}

// SetDefault 把指定 id 设为 default（事务内调用，tx 可传 nil 用 pool）
func (r *PostgresLLMModelRepository) SetDefault(ctx context.Context, tx pgx.Tx, id string) error {
	q := llmModelQueryer(tx)
	if q == nil {
		q = r.queryer()
	}
	_, err := q.Exec(ctx, `UPDATE llm_models SET is_default = TRUE, updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

// WithTx 事务
func (r *PostgresLLMModelRepository) WithTx(ctx context.Context, fn func(repo LLMModelRepository) error) error {
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
	if err := fn(&PostgresLLMModelRepository{db: r.db, tx: tx}); err != nil {
		return joinRollbackError(err, rollbackTx(ctx, tx))
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
