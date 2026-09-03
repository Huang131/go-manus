package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bytedance/sonic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mooc-manus/go-manus/api/internal/infrastructure"
	"github.com/mooc-manus/go-manus/api/internal/model"
)

// SessionRepository 会话仓储接口
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByID(ctx context.Context, id string) (*model.Session, error)
	GetAll(ctx context.Context) ([]*model.Session, error)
	List(ctx context.Context, limit, offset int) ([]*model.Session, int, error)
	Update(ctx context.Context, session *model.Session) error
	Delete(ctx context.Context, id string) error

	// 事件操作
	AppendEvent(ctx context.Context, id string, event *model.Event) error

	// 内存操作
	GetMemory(ctx context.Context, id string, agentName string) (*model.Memory, error)
	SaveMemory(ctx context.Context, id string, agentName string, memory *model.Memory) error

	// 原子更新
	UpdateTitle(ctx context.Context, id string, title string) error
	UpdateLatestMessage(ctx context.Context, id string, message string) error
	UpdateStatus(ctx context.Context, id string, status model.SessionStatus) error
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

// NewSessionRepositoryWithTx 创建带事务的会话仓储
func NewSessionRepositoryWithTx(tx pgx.Tx) SessionRepository {
	return &PostgresSessionRepository{tx: tx}
}

func (r *PostgresSessionRepository) queryer(ctx context.Context) QueryContext {
	if r.tx != nil {
		return &txQueryContext{tx: r.tx, ctx: ctx}
	}
	return &poolQueryContext{pool: r.db.Pool}
}

// QueryContext 统一的查询接口
type QueryContext interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

type poolQueryContext struct {
	pool *pgxpool.Pool
}

func (p *poolQueryContext) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}
func (p *poolQueryContext) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}
func (p *poolQueryContext) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

type txQueryContext struct {
	tx  pgx.Tx
	ctx context.Context
}

func (t *txQueryContext) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}
func (t *txQueryContext) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}
func (t *txQueryContext) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

// Create 创建会话
func (r *PostgresSessionRepository) Create(ctx context.Context, session *model.Session) error {
	q := r.queryer(ctx)
	query := `
		INSERT INTO sessions (id, sandbox_id, task_id, title, unread_message_count, latest_message,
			latest_message_at, events, memories, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	events, _ := sonic.Marshal(session.Events)
	memories, _ := sonic.Marshal(session.Memories)
	_, err := q.Exec(ctx, query,
		session.ID, session.SandboxID, session.TaskID, session.Title,
		session.UnreadMessageCount, session.LatestMessage, session.LatestMessageAt,
		events, memories, session.Status, session.CreatedAt, session.UpdatedAt,
	)
	return err
}

// GetByID 根据ID获取会话 (全字段)
func (r *PostgresSessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
	q := r.queryer(ctx)
	query := `
		SELECT id, sandbox_id, task_id, title, unread_message_count, latest_message,
			latest_message_at, events, memories, status, created_at, updated_at
		FROM sessions WHERE id = $1
	`
	var s model.Session
	var eventsJSON, memoriesJSON []byte
	err := q.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.SandboxID, &s.TaskID, &s.Title, &s.UnreadMessageCount,
		&s.LatestMessage, &s.LatestMessageAt, &eventsJSON, &memoriesJSON,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := sonic.Unmarshal(eventsJSON, &s.Events); err != nil {
		return nil, err
	}
	if err := sonic.Unmarshal(memoriesJSON, &s.Memories); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetAll 获取所有会话 (按最新消息时间排序)
func (r *PostgresSessionRepository) GetAll(ctx context.Context) ([]*model.Session, error) {
	q := r.queryer(ctx)
	query := `
		SELECT id, sandbox_id, task_id, title, unread_message_count, latest_message,
			latest_message_at, events, memories, status, created_at, updated_at
		FROM sessions ORDER BY latest_message_at DESC NULLS LAST
	`
	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*model.Session
	for rows.Next() {
		var s model.Session
		var eventsJSON, memoriesJSON []byte
		if err := rows.Scan(
			&s.ID, &s.SandboxID, &s.TaskID, &s.Title, &s.UnreadMessageCount,
			&s.LatestMessage, &s.LatestMessageAt, &eventsJSON, &memoriesJSON,
			&s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sonic.Unmarshal(eventsJSON, &s.Events)
		sonic.Unmarshal(memoriesJSON, &s.Memories)
		sessions = append(sessions, &s)
	}
	return sessions, nil
}

// List 获取会话列表 (按最新消息时间排序)
func (r *PostgresSessionRepository) List(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	q := r.queryer(ctx)
	countQuery := `SELECT COUNT(*) FROM sessions`
	var total int
	if err := q.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, sandbox_id, task_id, title, unread_message_count, latest_message,
			latest_message_at, events, memories, status, created_at, updated_at
		FROM sessions ORDER BY latest_message_at DESC NULLS LAST LIMIT $1 OFFSET $2
	`
	rows, err := q.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*model.Session
	for rows.Next() {
		var s model.Session
		var eventsJSON, memoriesJSON []byte
		if err := rows.Scan(
			&s.ID, &s.SandboxID, &s.TaskID, &s.Title, &s.UnreadMessageCount,
			&s.LatestMessage, &s.LatestMessageAt, &eventsJSON, &memoriesJSON,
			&s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		sonic.Unmarshal(eventsJSON, &s.Events)
		sonic.Unmarshal(memoriesJSON, &s.Memories)
		sessions = append(sessions, &s)
	}
	return sessions, total, nil
}

// Update 更新会话 (全字段)
func (r *PostgresSessionRepository) Update(ctx context.Context, session *model.Session) error {
	q := r.queryer(ctx)
	query := `
		UPDATE sessions SET
			sandbox_id = $2, task_id = $3, title = $4, unread_message_count = $5,
			latest_message = $6, latest_message_at = $7, events = $8,
			memories = $9, status = $10, updated_at = $11
		WHERE id = $1
	`
	events, _ := sonic.Marshal(session.Events)
	memories, _ := sonic.Marshal(session.Memories)
	_, err := q.Exec(ctx, query,
		session.ID, session.SandboxID, session.TaskID, session.Title,
		session.UnreadMessageCount, session.LatestMessage, session.LatestMessageAt,
		events, memories, session.Status, session.UpdatedAt,
	)
	return err
}

// Delete 删除会话
func (r *PostgresSessionRepository) Delete(ctx context.Context, id string) error {
	q := r.queryer(ctx)
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := q.Exec(ctx, query, id)
	return err
}

// AppendEvent 追加事件并同步最新消息
func (r *PostgresSessionRepository) AppendEvent(ctx context.Context, id string, event *model.Event) error {
	q := r.queryer(ctx)

	// 回填事件 ID 和时间戳，确保存储到数据库的事件信息完整
	// （上游调用点只设置 Type/Data，之前导致 id 为空、created_at 为零值）
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// 从事件中提取最新消息内容，并判断是否为 assistant 回复
	// 只有 assistant 的新回复才计入未读数（用户自己发送的消息不算未读）
	var message string
	isAssistantReply := false
	if event.Type == model.EventTypeMessage {
		var msgEvent model.MessageEvent
		if err := sonic.Unmarshal(event.Data, &msgEvent); err == nil {
			message = msgEvent.Message
			isAssistantReply = msgEvent.Role == "assistant"
		}
	}

	unreadDelta := 0
	if isAssistantReply {
		unreadDelta = 1
	}

	timestamp := event.CreatedAt

	// 原子追加事件 + 更新最新消息 + 仅对 assistant 回复增加未读数
	query := `
		UPDATE sessions SET
			events = events || $2::jsonb,
			latest_message = CASE WHEN $3 != '' THEN $3 ELSE latest_message END,
			latest_message_at = CASE WHEN $3 != '' THEN $4 ELSE latest_message_at END,
			unread_message_count = unread_message_count + $5,
			updated_at = $4
		WHERE id = $1
	`
	_, err := q.Exec(ctx, query, id, event.ToJSON(), message, timestamp, unreadDelta)
	return err
}

// GetMemory 获取指定 Agent 的记忆
func (r *PostgresSessionRepository) GetMemory(ctx context.Context, id string, agentName string) (*model.Memory, error) {
	q := r.queryer(ctx)
	query := `SELECT memories->>$2 FROM sessions WHERE id = $1`
	var memoryJSON []byte
	err := q.QueryRow(ctx, query, id, agentName).Scan(&memoryJSON)
	if err == sql.ErrNoRows {
		return model.NewMemory(), nil
	}
	if err != nil {
		return nil, err
	}
	var m model.Memory
	if err := sonic.Unmarshal(memoryJSON, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SaveMemory 保存指定 Agent 的记忆
func (r *PostgresSessionRepository) SaveMemory(ctx context.Context, id string, agentName string, memory *model.Memory) error {
	q := r.queryer(ctx)
	memoryJSON, err := sonic.Marshal(memory)
	if err != nil {
		return err
	}
	query := `
		UPDATE sessions SET
			memories = JSONB_SET(COALESCE(memories, '{}'::jsonb), $2::text[], $3),
			updated_at = NOW()
		WHERE id = $1
	`
	_, err = q.Exec(ctx, query, id, "{"+agentName+"}", memoryJSON)
	return err
}

// UpdateTitle 更新会话标题
func (r *PostgresSessionRepository) UpdateTitle(ctx context.Context, id string, title string) error {
	q := r.queryer(ctx)
	query := `UPDATE sessions SET title = $2, updated_at = NOW() WHERE id = $1`
	_, err := q.Exec(ctx, query, id, title)
	return err
}

// UpdateLatestMessage 更新最新消息
func (r *PostgresSessionRepository) UpdateLatestMessage(ctx context.Context, id string, message string) error {
	q := r.queryer(ctx)
	query := `UPDATE sessions SET latest_message = $2, latest_message_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := q.Exec(ctx, query, id, message)
	return err
}

// UpdateStatus 更新会话状态
func (r *PostgresSessionRepository) UpdateStatus(ctx context.Context, id string, status model.SessionStatus) error {
	q := r.queryer(ctx)
	query := `UPDATE sessions SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := q.Exec(ctx, query, id, status)
	return err
}

// IncrementUnreadCount 原子增加未读数
func (r *PostgresSessionRepository) IncrementUnreadCount(ctx context.Context, id string) error {
	q := r.queryer(ctx)
	query := `UPDATE sessions SET unread_message_count = unread_message_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := q.Exec(ctx, query, id)
	return err
}

// DecrementUnreadCount 原子减少未读数 (最低为0)
func (r *PostgresSessionRepository) DecrementUnreadCount(ctx context.Context, id string) error {
	q := r.queryer(ctx)
	query := `
		UPDATE sessions SET unread_message_count = GREATEST(unread_message_count - 1, 0), updated_at = NOW()
		WHERE id = $1
	`
	_, err := q.Exec(ctx, query, id)
	return err
}

// SetUnreadCount 设置未读数
func (r *PostgresSessionRepository) SetUnreadCount(ctx context.Context, id string, count int) error {
	q := r.queryer(ctx)
	query := `UPDATE sessions SET unread_message_count = $2, updated_at = NOW() WHERE id = $1`
	_, err := q.Exec(ctx, query, id, count)
	return err
}

// WithTx 在事务中执行操作
func (r *PostgresSessionRepository) WithTx(ctx context.Context, fn func(repo SessionRepository) error) error {
	// 如果已经在事务中，直接执行
	if r.tx != nil {
		return fn(r)
	}

	// 开始新事务
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// 确保事务回滚
	var committed bool
	defer func() {
		if !committed {
			if p := recover(); p != nil {
				tx.Rollback(ctx)
				panic(p) // 重新抛出 panic
			}
			tx.Rollback(ctx) // 失败时回滚
		}
	}()

	// 执行操作
	err = fn(&PostgresSessionRepository{db: r.db, tx: tx})
	if err != nil {
		return err // 错误时 defer 会处理回滚
	}

	// 提交事务
	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit transaction: %w", commitErr)
	}
	committed = true
	return nil
}
