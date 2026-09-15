package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/jackc/pgx/v5"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
)

// LLMModelRepository 多模型仓储
type LLMModelRepository interface {
	Create(ctx context.Context, m *model.LLMModel) error
	Update(ctx context.Context, m *model.LLMModel) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.LLMModel, error)
	GetDefault(ctx context.Context) (*model.LLMModel, error)
	GetFirstEnabled(ctx context.Context) (*model.LLMModel, error)
	List(ctx context.Context) ([]*model.LLMModel, error)
	ClearDefault(ctx context.Context) error
	SetDefault(ctx context.Context, id string) error
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

func (r *PostgresLLMModelRepository) queryer() queryer {
	return newQueryer(r.db, r.tx)
}

// capabilities/request_policy/cost_policy 三个 JSONB 列；运行时健康只在内存维护，不落库。
const llmModelColumns = `id, name, provider, base_url, api_key, model_name,
	temperature, max_tokens, tags, is_default, is_enabled, sort_order,
	capabilities, request_policy, cost_policy,
	created_at, updated_at`

func scanLLMModel(row rowScanner, m *model.LLMModel) error {
	var tagsJSON, capJSON, reqJSON, costJSON []byte
	if err := row.Scan(
		&m.ID, &m.Name, &m.Provider, &m.BaseURL, &m.APIKey, &m.ModelName,
		&m.Temperature, &m.MaxTokens, &tagsJSON, &m.IsDefault, &m.IsEnabled,
		&m.SortOrder, &capJSON, &reqJSON, &costJSON,
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
	return nil
}

// modelJSON 把模型的一个 JSONB 字段编码为 []byte（写入 DB 前调用）。
func modelJSON(field string, value any) ([]byte, error) {
	b, err := sonic.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode model %s: %w", field, err)
	}
	return b, nil
}

// modelJSONColumns 模型全部 JSONB 列的编码结果。
// 具名字段与 SQL 占位符一一对应，避免魔法下标错位（下标错位编译期不报错）。
type modelJSONColumns struct {
	tags          []byte
	capabilities  []byte
	requestPolicy []byte
	costPolicy    []byte
}

func encodeModelJSONColumns(m *model.LLMModel) (modelJSONColumns, error) {
	var c modelJSONColumns
	for _, f := range []struct {
		name  string
		value any
		dst   *[]byte
	}{
		{"tags", m.Tags, &c.tags},
		{"capabilities", m.Capabilities, &c.capabilities},
		{"request policy", m.RequestPolicy, &c.requestPolicy},
		{"cost policy", m.CostPolicy, &c.costPolicy},
	} {
		b, err := modelJSON(f.name, f.value)
		if err != nil {
			return modelJSONColumns{}, err
		}
		*f.dst = b
	}
	return c, nil
}

// Create 新增
func (r *PostgresLLMModelRepository) Create(ctx context.Context, m *model.LLMModel) error {
	f, err := encodeModelJSONColumns(m)
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
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`,
		m.ID, m.Name, m.Provider, m.BaseURL, m.APIKey, m.ModelName,
		m.Temperature, m.MaxTokens, f.tags, m.IsDefault, m.IsEnabled,
		m.SortOrder,
		f.capabilities, f.requestPolicy, f.costPolicy,
		m.CreatedAt, m.UpdatedAt,
	)
	return err
}

// Update 更新；id 不存在（校验后被并发删除）时返回 pgx.ErrNoRows。
func (r *PostgresLLMModelRepository) Update(ctx context.Context, m *model.LLMModel) error {
	f, err := encodeModelJSONColumns(m)
	if err != nil {
		return err
	}
	m.UpdatedAt = time.Now()
	tag, err := r.queryer().Exec(ctx, `
		UPDATE llm_models SET
			name = $2, provider = $3, base_url = $4, api_key = $5, model_name = $6,
			temperature = $7, max_tokens = $8, tags = $9, is_default = $10,
			is_enabled = $11, sort_order = $12,
			capabilities = $13, request_policy = $14, cost_policy = $15,
			updated_at = $16
	WHERE id = $1
	`,
		m.ID, m.Name, m.Provider, m.BaseURL, m.APIKey, m.ModelName,
		m.Temperature, m.MaxTokens, f.tags, m.IsDefault, m.IsEnabled,
		m.SortOrder,
		f.capabilities, f.requestPolicy, f.costPolicy,
		m.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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
	return collectRows(rows, func(row rowScanner) (*model.LLMModel, error) {
		var m model.LLMModel
		if err := scanLLMModel(row, &m); err != nil {
			return nil, err
		}
		return &m, nil
	})
}

// ClearDefault 清空所有 default。
func (r *PostgresLLMModelRepository) ClearDefault(ctx context.Context) error {
	_, err := r.queryer().Exec(ctx, `UPDATE llm_models SET is_default = FALSE, updated_at = $1 WHERE is_default = TRUE`, time.Now())
	return err
}

// SetDefault 把指定 id 设为 default。
func (r *PostgresLLMModelRepository) SetDefault(ctx context.Context, id string) error {
	_, err := r.queryer().Exec(ctx, `UPDATE llm_models SET is_default = TRUE, updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

// WithTx 事务
func (r *PostgresLLMModelRepository) WithTx(ctx context.Context, fn func(repo LLMModelRepository) error) error {
	return runInTx(ctx, r.db, r.tx, func(tx pgx.Tx) LLMModelRepository {
		return &PostgresLLMModelRepository{db: r.db, tx: tx}
	}, fn)
}
