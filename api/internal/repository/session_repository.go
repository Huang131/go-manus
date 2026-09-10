package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/google/uuid"
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

	// 事件操作
	AppendEvent(ctx context.Context, id string, event *model.Event) error

	// 内存操作
	GetMemory(ctx context.Context, id string, agentName string) ([]llmcore.Message, error)
	SaveMemory(ctx context.Context, id string, agentName string, messages []llmcore.Message) error

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

func (r *PostgresSessionRepository) queryer() queryer {
	return newQueryer(r.db, r.tx)
}

// ErrSessionNotFound 会话写操作命中 0 行（不存在或已软删除）时返回，
// 避免 UPDATE 静默成功后调用方无法感知会话缺失。
var ErrSessionNotFound = errors.New("session not found")

// sessionSummaryColumns 是会话摘要查询的公共列（不含 events 大字段）。
const sessionSummaryColumns = `id, COALESCE(sandbox_id, ''), COALESCE(task_id, ''), title, unread_message_count, COALESCE(latest_message, ''),
			latest_message_at, status, created_at, updated_at`

// scanSessionSummary 扫描会话摘要行，供 GetAll / List 共用。
func scanSessionSummary(s rowScanner) (*model.Session, error) {
	var sess model.Session
	var latestMessageAt sql.NullTime
	if err := s.Scan(
		&sess.ID, &sess.SandboxID, &sess.TaskID, &sess.Title, &sess.UnreadMessageCount,
		&sess.LatestMessage, &latestMessageAt,
		&sess.Status, &sess.CreatedAt, &sess.UpdatedAt,
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
		return ErrSessionNotFound
	}
	return nil
}

// Create 创建会话
func (r *PostgresSessionRepository) Create(ctx context.Context, session *model.Session) error {
	q := r.queryer()
	query := `
		INSERT INTO sessions (id, sandbox_id, task_id, title, unread_message_count, latest_message,
			latest_message_at, events, memories, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '{}'::jsonb, $9, $10, $11)
	`
	events, err := marshalSessionEvents(session.Events)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, query,
		session.ID, session.SandboxID, session.TaskID, session.Title,
		session.UnreadMessageCount, session.LatestMessage, session.LatestMessageAt,
		events, session.Status, session.CreatedAt, session.UpdatedAt,
	)
	return err
}

// GetByID 根据ID获取会话 (全字段)
func (r *PostgresSessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
	q := r.queryer()
	query := `
		SELECT id, COALESCE(sandbox_id, ''), COALESCE(task_id, ''), title, unread_message_count, COALESCE(latest_message, ''),
			latest_message_at, events, status, created_at, updated_at
		FROM sessions WHERE id = $1 AND deleted_at IS NULL
	`
	var s model.Session
	var eventsJSON []byte
	var latestMessageAt sql.NullTime
	err := q.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.SandboxID, &s.TaskID, &s.Title, &s.UnreadMessageCount,
		&s.LatestMessage, &latestMessageAt, &eventsJSON,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
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
	if err := sonic.Unmarshal(eventsJSON, &s.Events); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetAll 获取所有会话 (按最新消息时间排序，排除已删除)
func (r *PostgresSessionRepository) GetAll(ctx context.Context) ([]*model.Session, error) {
	q := r.queryer()
	query := `
		SELECT ` + sessionSummaryColumns + `
		FROM sessions WHERE deleted_at IS NULL
		ORDER BY latest_message_at DESC NULLS LAST, created_at DESC, id DESC
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
		FROM sessions WHERE deleted_at IS NULL
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

// Update 更新会话 (全字段，memories 列由 SaveMemory 独立管理，此处不触碰)
func (r *PostgresSessionRepository) Update(ctx context.Context, session *model.Session) error {
	query := `
		UPDATE sessions SET
			sandbox_id = $2, task_id = $3, title = $4, unread_message_count = $5,
			latest_message = $6, latest_message_at = $7, events = $8,
			status = $9, updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL
	`
	events, err := marshalSessionEvents(session.Events)
	if err != nil {
		return err
	}
	return r.execSessionWrite(ctx, query,
		session.ID, session.SandboxID, session.TaskID, session.Title,
		session.UnreadMessageCount, session.LatestMessage, session.LatestMessageAt,
		events, session.Status, session.UpdatedAt,
	)
}

// marshalSessionEvents 为持久化错误补充字段上下文，避免非法 RawMessage 静默写入。
func marshalSessionEvents(events []model.Event) ([]byte, error) {
	data, err := sonic.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("encode session events: %w", err)
	}
	return data, nil
}

// Delete 删除会话（软删除）
// 设置 deleted_at 时间戳，会话关联的文件在过期后会被自动清理
func (r *PostgresSessionRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE sessions SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	return r.execSessionWrite(ctx, query, id)
}

// AppendEvent 追加事件并同步最新消息
func (r *PostgresSessionRepository) AppendEvent(ctx context.Context, id string, event *model.Event) error {
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
		WHERE id = $1 AND deleted_at IS NULL
	`
	return r.execSessionWrite(ctx, query, id, event.ToJSON(), message, timestamp, unreadDelta)
}

// GetMemory 获取指定 Agent 的记忆
func (r *PostgresSessionRepository) GetMemory(ctx context.Context, id string, agentName string) ([]llmcore.Message, error) {
	q := r.queryer()
	query := `SELECT memories->>$2 FROM sessions WHERE id = $1 AND deleted_at IS NULL`
	var memoryJSON []byte
	err := q.QueryRow(ctx, query, id, agentName).Scan(&memoryJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return []llmcore.Message{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(memoryJSON) == 0 {
		return []llmcore.Message{}, nil
	}
	var messages []llmcore.Message
	if err := sonic.Unmarshal(memoryJSON, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SaveMemory 保存指定 Agent 的记忆
func (r *PostgresSessionRepository) SaveMemory(ctx context.Context, id string, agentName string, messages []llmcore.Message) error {
	if messages == nil {
		messages = []llmcore.Message{}
	}
	memoryJSON, err := sonic.Marshal(messages)
	if err != nil {
		return err
	}
	query := `
		UPDATE sessions SET
			memories = JSONB_SET(COALESCE(memories, '{}'::jsonb), $2::text[], $3),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	return r.execSessionWrite(ctx, query, id, "{"+agentName+"}", memoryJSON)
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

// UpdateStatus 更新会话状态
func (r *PostgresSessionRepository) UpdateStatus(ctx context.Context, id string, status model.SessionStatus) error {
	query := `UPDATE sessions SET status = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	return r.execSessionWrite(ctx, query, id, status)
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
