package repository

import (
	"context"
	"errors"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/jackc/pgx/v5"
)

// FileRepository 文件仓储接口
type FileRepository interface {
	Create(ctx context.Context, file *model.File) error
	GetByID(ctx context.Context, id string) (*model.File, error)
	// 仅返回属于指定会话的文件，避免跨会话引用文件。
	GetBySessionAndID(ctx context.Context, sessionID, id string) (*model.File, error)
	// 根据 session_id + filepath 查重
	GetBySessionAndFilepath(ctx context.Context, sessionID, filepath string) (*model.File, error)
	// 查找会话内同名的最近文件（用于上传幂等去重）
	GetBySessionAndFilename(ctx context.Context, sessionID, filename string) (*model.File, error)
	// 按 (session_id, sha256) 查找同内容文件（内容级去重）
	GetBySessionAndHash(ctx context.Context, sessionID, sha256 string) (*model.File, error)
	Update(ctx context.Context, file *model.File) error
	Delete(ctx context.Context, id string) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
	ListBySessionID(ctx context.Context, sessionID string) ([]*model.File, error)

	// 过期文件管理
	GetExpiredFiles(ctx context.Context, expireDuration string, limit int64) ([]*model.File, error)
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

func (r *PostgresFileRepository) queryer() queryer {
	return newQueryer(r.db, r.tx)
}

// fileColumns 是 files 表的标准查询列，集中定义避免各方法重复列举。
const fileColumns = `id, session_id, filename, filepath, key, extension, mime_type, size, sha256, created_at`

// scanFile 把一行结果映射为 *model.File，供 QueryRow 与 Query 行迭代共用。
func scanFile(s rowScanner) (*model.File, error) {
	var f model.File
	if err := s.Scan(&f.ID, &f.SessionID, &f.Filename, &f.Filepath, &f.Key, &f.Extension, &f.MimeType, &f.Size, &f.Sha256, &f.CreatedAt); err != nil {
		return nil, err
	}
	return &f, nil
}

// Create 创建文件记录
func (r *PostgresFileRepository) Create(ctx context.Context, file *model.File) error {
	q := r.queryer()
	query := `
		INSERT INTO files (id, session_id, filename, filepath, key, extension, mime_type, size, sha256, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := q.Exec(ctx, query,
		file.ID, file.SessionID, file.Filename, file.Filepath,
		file.Key, file.Extension, file.MimeType, file.Size, file.Sha256, file.CreatedAt,
	)
	return err
}

// GetBySessionAndFilepath 根据 session_id + filepath 查重（替代旧 sessions.files JSONB 的 GetFileByPath）
func (r *PostgresFileRepository) GetBySessionAndFilepath(ctx context.Context, sessionID, filepath string) (*model.File, error) {
	q := r.queryer()
	query := `SELECT ` + fileColumns + ` FROM files WHERE session_id = $1 AND filepath = $2 LIMIT 1`
	file, err := scanFile(q.QueryRow(ctx, query, sessionID, filepath))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return file, nil
}

// GetBySessionAndFilename 查找会话内同名的最近文件（用于上传幂等去重）
func (r *PostgresFileRepository) GetBySessionAndFilename(ctx context.Context, sessionID, filename string) (*model.File, error) {
	q := r.queryer()
	query := `SELECT ` + fileColumns + ` FROM files WHERE session_id = $1 AND filename = $2 ORDER BY created_at DESC LIMIT 1`
	file, err := scanFile(q.QueryRow(ctx, query, sessionID, filename))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return file, nil
}

// GetBySessionAndHash 按 (session_id, sha256) 查找同内容文件（内容级去重）
func (r *PostgresFileRepository) GetBySessionAndHash(ctx context.Context, sessionID, sha256 string) (*model.File, error) {
	q := r.queryer()
	query := `SELECT ` + fileColumns + ` FROM files WHERE session_id = $1 AND sha256 = $2 ORDER BY created_at DESC LIMIT 1`
	file, err := scanFile(q.QueryRow(ctx, query, sessionID, sha256))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return file, nil
}

// GetByID 根据ID获取文件
func (r *PostgresFileRepository) GetByID(ctx context.Context, id string) (*model.File, error) {
	q := r.queryer()
	query := `SELECT ` + fileColumns + ` FROM files WHERE id = $1`
	file, err := scanFile(q.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return file, nil
}

// GetBySessionAndID 根据会话和文件 ID 查询文件，所有权约束在 SQL 边界执行。
func (r *PostgresFileRepository) GetBySessionAndID(ctx context.Context, sessionID, id string) (*model.File, error) {
	q := r.queryer()
	query := `SELECT ` + fileColumns + ` FROM files WHERE session_id = $1 AND id = $2`
	file, err := scanFile(q.QueryRow(ctx, query, sessionID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Update 更新文件记录（filename/filepath）。
// 目标不存在时返回 pgx.ErrNoRows，避免对不存在的 id 静默报告成功。
func (r *PostgresFileRepository) Update(ctx context.Context, file *model.File) error {
	q := r.queryer()
	query := `
		UPDATE files SET filename = $2, filepath = $3 WHERE id = $1
	`
	tag, err := q.Exec(ctx, query, file.ID, file.Filename, file.Filepath)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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
	query := `SELECT ` + fileColumns + ` FROM files WHERE session_id = $1 ORDER BY created_at DESC`
	rows, err := q.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanFile)
}

// WithTx 在事务中执行操作
func (r *PostgresFileRepository) WithTx(ctx context.Context, fn func(repo FileRepository) error) error {
	return runInTx(ctx, r.db, r.tx, func(tx pgx.Tx) FileRepository {
		return &PostgresFileRepository{db: r.db, tx: tx}
	}, fn)
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
		SELECT f.id, f.session_id, f.filename, f.filepath, f.key, f.extension, f.mime_type, f.size, f.sha256, f.created_at
		FROM files f
		LEFT JOIN sessions s ON f.session_id = s.id
		WHERE f.created_at < NOW() - $1::interval
		  AND (f.session_id IS NULL OR s.deleted_at IS NOT NULL)
		ORDER BY f.created_at ASC
		LIMIT $2
	`

	rows, err := q.Query(ctx, query, expireDuration, limit)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanFile)
}

// GetFilesBySessionIDs 根据会话ID列表获取文件
func (r *PostgresFileRepository) GetFilesBySessionIDs(ctx context.Context, sessionIDs []string) ([]*model.File, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}

	q := r.queryer()
	query := `SELECT ` + fileColumns + ` FROM files WHERE session_id = ANY($1) ORDER BY created_at DESC`
	rows, err := q.Query(ctx, query, sessionIDs)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanFile)
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
