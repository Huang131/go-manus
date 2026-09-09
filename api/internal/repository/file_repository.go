package repository

import (
	"context"
	"errors"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FileRepository 文件仓储接口
type FileRepository interface {
	Create(ctx context.Context, file *model.File) error
	GetByID(ctx context.Context, id string) (*model.File, error)
	// GetBySessionAndFilepath 根据 session_id + filepath 查重（替代旧 sessions.files JSONB 的路径去重逻辑）
	GetBySessionAndFilepath(ctx context.Context, sessionID, filepath string) (*model.File, error)
	Update(ctx context.Context, file *model.File) error
	Delete(ctx context.Context, id string) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
	ListBySessionID(ctx context.Context, sessionID string) ([]*model.File, error)

	// 过期文件管理
	GetExpiredFiles(ctx context.Context, expireDuration string, limit int64) ([]*model.File, error)
	CountExpiredFiles(ctx context.Context, expireDuration string) (int64, error)
	DeleteByIDs(ctx context.Context, ids []string) (int64, error)

	// 批量操作
	GetFilesBySessionIDs(ctx context.Context, sessionIDs []string) ([]*model.File, error)

	// 事务支持
	WithTx(ctx context.Context, fn func(repo FileRepository) error) error
}

// PostgresFileRepository PostgreSQL 文件仓储实现
type PostgresFileRepository struct {
	db *infrastructure.Postgres
	tx pgx.Tx
}

// NewFileRepository 创建文件仓储
func NewFileRepository(db *infrastructure.Postgres) FileRepository {
	return &PostgresFileRepository{db: db}
}

// NewFileRepositoryWithTx 创建带事务的文件仓储
func NewFileRepositoryWithTx(tx pgx.Tx) FileRepository {
	return &PostgresFileRepository{tx: tx}
}

func (r *PostgresFileRepository) queryer() QueryContext {
	if r.tx != nil {
		return &fileTxQueryContext{tx: r.tx}
	}
	return &filePoolQueryContext{pool: r.db.Pool}
}

type filePoolQueryContext struct {
	pool *pgxpool.Pool
}

func (p *filePoolQueryContext) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}
func (p *filePoolQueryContext) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}
func (p *filePoolQueryContext) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

type fileTxQueryContext struct {
	tx pgx.Tx
}

func (t *fileTxQueryContext) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}
func (t *fileTxQueryContext) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}
func (t *fileTxQueryContext) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

// Create 创建文件记录
func (r *PostgresFileRepository) Create(ctx context.Context, file *model.File) error {
	q := r.queryer()
	query := `
		INSERT INTO files (id, session_id, filename, filepath, key, extension, mime_type, size, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := q.Exec(ctx, query,
		file.ID, file.SessionID, file.Filename, file.Filepath,
		file.Key, file.Extension, file.MimeType, file.Size, file.CreatedAt,
	)
	return err
}

// GetBySessionAndFilepath 根据 session_id + filepath 查重（替代旧 sessions.files JSONB 的 GetFileByPath）
func (r *PostgresFileRepository) GetBySessionAndFilepath(ctx context.Context, sessionID, filepath string) (*model.File, error) {
	q := r.queryer()
	query := `
		SELECT id, session_id, filename, filepath, key, extension, mime_type, size, created_at
		FROM files WHERE session_id = $1 AND filepath = $2 LIMIT 1
	`
	var file model.File
	err := q.QueryRow(ctx, query, sessionID, filepath).Scan(
		&file.ID, &file.SessionID, &file.Filename, &file.Filepath,
		&file.Key, &file.Extension, &file.MimeType, &file.Size, &file.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// GetByID 根据ID获取文件
func (r *PostgresFileRepository) GetByID(ctx context.Context, id string) (*model.File, error) {
	q := r.queryer()
	query := `
		SELECT id, session_id, filename, filepath, key, extension, mime_type, size, created_at
		FROM files WHERE id = $1
	`
	var file model.File
	err := q.QueryRow(ctx, query, id).Scan(
		&file.ID, &file.SessionID, &file.Filename, &file.Filepath,
		&file.Key, &file.Extension, &file.MimeType, &file.Size, &file.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// Update 更新文件记录
func (r *PostgresFileRepository) Update(ctx context.Context, file *model.File) error {
	q := r.queryer()
	query := `
		UPDATE files SET filename = $2, filepath = $3 WHERE id = $1
	`
	_, err := q.Exec(ctx, query, file.ID, file.Filename, file.Filepath)
	return err
}

// Delete 删除文件记录
func (r *PostgresFileRepository) Delete(ctx context.Context, id string) error {
	q := r.queryer()
	query := `DELETE FROM files WHERE id = $1`
	_, err := q.Exec(ctx, query, id)
	return err
}

// DeleteBySessionID 根据会话ID删除所有文件记录
func (r *PostgresFileRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	q := r.queryer()
	query := `DELETE FROM files WHERE session_id = $1`
	_, err := q.Exec(ctx, query, sessionID)
	return err
}

// ListBySessionID 根据会话ID获取文件列表
func (r *PostgresFileRepository) ListBySessionID(ctx context.Context, sessionID string) ([]*model.File, error) {
	q := r.queryer()
	query := `
		SELECT id, session_id, filename, filepath, key, extension, mime_type, size, created_at
		FROM files WHERE session_id = $1 ORDER BY created_at DESC
	`
	rows, err := q.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.SessionID, &f.Filename, &f.Filepath, &f.Key, &f.Extension, &f.MimeType, &f.Size, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// WithTx 在事务中执行操作
func (r *PostgresFileRepository) WithTx(ctx context.Context, fn func(repo FileRepository) error) error {
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
	if err := fn(&PostgresFileRepository{db: r.db, tx: tx}); err != nil {
		tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// GetExpiredFiles 获取过期文件列表
// 只清理孤立文件（无关联会话或关联会话已删除）：
// 1. session_id IS NULL：未关联会话的文件
// 2. session.deleted_at IS NOT NULL：关联的会话已被软删除
//
// 不会清理关联活跃会话的文件，保证"会话文件与会话同寿命"
func (r *PostgresFileRepository) GetExpiredFiles(ctx context.Context, expireDuration string, limit int64) ([]*model.File, error) {
	q := r.queryer()
	query := `
		SELECT f.id, f.session_id, f.filename, f.filepath, f.key, f.extension, f.mime_type, f.size, f.created_at
		FROM files f
		LEFT JOIN sessions s ON f.session_id = s.id
		WHERE f.created_at < NOW() - INTERVAL $1
		  AND (f.session_id IS NULL OR s.deleted_at IS NOT NULL)
		ORDER BY f.created_at ASC
		LIMIT $2
	`

	rows, err := q.Query(ctx, query, expireDuration, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.SessionID, &f.Filename, &f.Filepath, &f.Key, &f.Extension, &f.MimeType, &f.Size, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// GetFilesBySessionIDs 根据会话ID列表获取文件
func (r *PostgresFileRepository) GetFilesBySessionIDs(ctx context.Context, sessionIDs []string) ([]*model.File, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}

	q := r.queryer()
	query := `
		SELECT id, session_id, filename, filepath, key, extension, mime_type, size, created_at
		FROM files
		WHERE session_id = ANY($1)
		ORDER BY created_at DESC
	`

	rows, err := q.Query(ctx, query, sessionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.SessionID, &f.Filename, &f.Filepath, &f.Key, &f.Extension, &f.MimeType, &f.Size, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// CountExpiredFiles 统计可清理的过期文件数量
// 只统计孤立文件（无关联会话或关联会话已删除）
func (r *PostgresFileRepository) CountExpiredFiles(ctx context.Context, expireDuration string) (int64, error) {
	q := r.queryer()
	query := `
		SELECT COUNT(*) FROM files f
		LEFT JOIN sessions s ON f.session_id = s.id
		WHERE f.created_at < NOW() - INTERVAL $1
		  AND (f.session_id IS NULL OR s.deleted_at IS NOT NULL)
	`

	var count int64
	err := q.QueryRow(ctx, query, expireDuration).Scan(&count)
	return count, err
}

// DeleteByIDs 根据 ID 列表批量删除文件
func (r *PostgresFileRepository) DeleteByIDs(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	q := r.queryer()
	query := `DELETE FROM files WHERE id = ANY($1)`
	result, err := q.Exec(ctx, query, ids)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
