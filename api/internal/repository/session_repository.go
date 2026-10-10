package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/jackc/pgx/v5"
)

// SessionRepository 会话仓储接口
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByID(ctx context.Context, id string) (*model.Session, error)
	GetAll(ctx context.Context) ([]*model.Session, error)
	List(ctx context.Context, limit, offset int) ([]*model.Session, int, error)
	Update(ctx context.Context, session *model.Session) error
	Delete(ctx context.Context, id string) error

	// 原子更新
	UpdateTitle(ctx context.Context, id string, title string) error
	UpdateLatestMessage(ctx context.Context, id string, message string) error
	IncrementUnreadCount(ctx context.Context, id string) error
	DecrementUnreadCount(ctx context.Context, id string) error
	SetUnreadCount(ctx context.Context, id string, count int) error

	// 事务支持
	WithTx(ctx context.Context, fn func(repo SessionRepository) error) error
}

// PostgresSessionRepository PostgreSQL 会话仓储实现
type PostgresSessionRepository struct {
	db *infrastructure.Postgres
	tx pgx.Tx
}

// NewSessionRepository 创建会话仓储
func NewSessionRepository(db *infrastructure.Postgres) SessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) queryer() queryer {
	return newQueryer(r.db, r.tx)
}

// sessionSummaryColumns 是会话摘要查询的公共列，并投影当前活跃 Run 状态。
const sessionSummaryColumns = `s.id, COALESCE(s.sandbox_id, ''), s.title, s.unread_message_count, COALESCE(s.latest_message, ''),
			s.latest_message_at, COALESCE(active_run.status, ''), s.created_at, s.updated_at`

// scanSessionSummary 扫描会话摘要行，供 GetAll / List 共用。
func scanSessionSummary(s rowScanner) (*model.Session, error) {
	var sess model.Session
	var latestMessageAt sql.NullTime
	if err := s.Scan(
		&sess.ID, &sess.SandboxID, &sess.Title, &sess.UnreadMessageCount,
		&sess.LatestMessage, &latestMessageAt,
		&sess.ActiveRunStatus, &sess.CreatedAt, &sess.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if latestMessageAt.Valid {
		sess.LatestMessageAt = &latestMessageAt.Time
	}
	return &sess, nil
}

// execSessionWrite 执行会话写操作并校验 RowsAffected，命中 0 行返回 ErrSessionNotFound。
func (r *PostgresSessionRepository) execSessionWrite(ctx context.Context, query string, args ...any) error {
	tag, err := r.queryer().Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrSessionNotFound
	}
	return nil
}

// Create 创建会话
func (r *PostgresSessionRepository) Create(ctx context.Context, session *model.Session) error {
	q := r.queryer()
	query := `
		INSERT INTO sessions (id, sandbox_id, title, unread_message_count, latest_message,
			latest_message_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := q.Exec(ctx, query,
		session.ID, session.SandboxID, session.Title,
		session.UnreadMessageCount, session.LatestMessage, session.LatestMessageAt,
		session.CreatedAt, session.UpdatedAt,
	)
	return err
}

// GetByID 根据ID获取会话 (全字段)
func (r *PostgresSessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
	q := r.queryer()
	query := `
		SELECT ` + sessionSummaryColumns + `
		FROM sessions s
		LEFT JOIN LATERAL (
			SELECT status FROM runs
			WHERE session_id = s.id
				AND status IN ('pending', 'running', 'waiting_input', 'cancelling')
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) active_run ON true
		WHERE s.id = $1 AND s.deleted_at IS NULL
	`
	var s model.Session
	var latestMessageAt sql.NullTime
	err := q.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.SandboxID, &s.Title, &s.UnreadMessageCount,
		&s.LatestMessage, &latestMessageAt, &s.ActiveRunStatus,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if latestMessageAt.Valid {
		s.LatestMessageAt = &latestMessageAt.Time
	}
	return &s, nil
}

// GetAll 获取所有会话 (按最新消息时间排序，排除已删除)
func (r *PostgresSessionRepository) GetAll(ctx context.Context) ([]*model.Session, error) {
	q := r.queryer()
	// 上限保护：GetAll 服务于 SSE 会话流推送，无界全量会在会话数增长后拖垮轮询
	query := `
		SELECT ` + sessionSummaryColumns + `
		FROM sessions s
		LEFT JOIN LATERAL (
			SELECT status FROM runs
			WHERE session_id = s.id
				AND status IN ('pending', 'running', 'waiting_input', 'cancelling')
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) active_run ON true
		WHERE s.deleted_at IS NULL
		ORDER BY latest_message_at DESC NULLS LAST, created_at DESC, id DESC
		LIMIT 500
	`
	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanSessionSummary)
}

// List 获取会话列表 (按最新消息时间排序，排除已删除)
func (r *PostgresSessionRepository) List(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	q := r.queryer()
	countQuery := `SELECT COUNT(*) FROM sessions WHERE deleted_at IS NULL`
	var total int
	if err := q.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT ` + sessionSummaryColumns + `
		FROM sessions s
		LEFT JOIN LATERAL (
			SELECT status FROM runs
			WHERE session_id = s.id
				AND status IN ('pending', 'running', 'waiting_input', 'cancelling')
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) active_run ON true
		WHERE s.deleted_at IS NULL
		ORDER BY latest_message_at DESC NULLS LAST, created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := q.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	sessions, err := collectRows(rows, scanSessionSummary)
	if err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}

// Update 更新会话全字段。
func (r *PostgresSessionRepository) Update(ctx context.Context, session *model.Session) error {
	query := `
		UPDATE sessions SET
			sandbox_id = $2, title = $3, unread_message_count = $4,
			latest_message = $5, latest_message_at = $6, updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL
	`
	return r.execSessionWrite(ctx, query,
		session.ID, session.SandboxID, session.Title,
		session.UnreadMessageCount, session.LatestMessage, session.LatestMessageAt,
		session.UpdatedAt,
	)
}

// Delete 删除会话（软删除）
// 设置 deleted_at 时间戳，会话关联的文件在过期后会被自动清理
func (r *PostgresSessionRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE sessions SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	return r.execSessionWrite(ctx, query, id)
}

// UpdateTitle 更新会话标题
func (r *PostgresSessionRepository) UpdateTitle(ctx context.Context, id string, title string) error {
	query := `UPDATE sessions SET title = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	return r.execSessionWrite(ctx, query, id, title)
}

// UpdateLatestMessage 更新最新消息
func (r *PostgresSessionRepository) UpdateLatestMessage(ctx context.Context, id string, message string) error {
	query := `UPDATE sessions SET latest_message = $2, latest_message_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	return r.execSessionWrite(ctx, query, id, message)
}

// IncrementUnreadCount 原子增加未读数
func (r *PostgresSessionRepository) IncrementUnreadCount(ctx context.Context, id string) error {
	query := `UPDATE sessions SET unread_message_count = unread_message_count + 1, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	return r.execSessionWrite(ctx, query, id)
}

// DecrementUnreadCount 原子减少未读数 (最低为0)
func (r *PostgresSessionRepository) DecrementUnreadCount(ctx context.Context, id string) error {
	query := `
		UPDATE sessions SET unread_message_count = GREATEST(unread_message_count - 1, 0), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	return r.execSessionWrite(ctx, query, id)
}

// SetUnreadCount 设置未读数
func (r *PostgresSessionRepository) SetUnreadCount(ctx context.Context, id string, count int) error {
	query := `UPDATE sessions SET unread_message_count = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	return r.execSessionWrite(ctx, query, id, count)
}

// WithTx 在事务中执行操作
func (r *PostgresSessionRepository) WithTx(ctx context.Context, fn func(repo SessionRepository) error) error {
	return runInTx(ctx, r.db, r.tx, func(tx pgx.Tx) SessionRepository {
		return &PostgresSessionRepository{db: r.db, tx: tx}
	}, fn)
}
