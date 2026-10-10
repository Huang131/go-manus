package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/bytedance/sonic"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrActiveRunExists 表示该 Session 已有 pending/running/waiting_input/cancelling Run。
var ErrActiveRunExists = errors.New("session already has an active run")

// RunRepository 按 Run 聚合持久化执行事实与长期消息。
//
// 它只负责事务、约束和条件更新；不启动 Engine、不发布 Redis 事件，也不维护 Session 摘要。
type RunRepository interface {
	CreateWithInitialMessage(ctx context.Context, run *model.Run, initial *model.RunMessage) (created *model.Run, inserted bool, err error)
	GetByID(ctx context.Context, id string) (*model.Run, error)
	GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error)
	ListMessages(ctx context.Context, runID string) ([]*model.RunMessage, error)
	TransitionStatus(ctx context.Context, id string, from []model.RunStatus, to model.RunStatus) (bool, error)
	EnterWaitingInput(ctx context.Context, runID string, expectedRevision int, snapshot model.RunExecutionSnapshot, question *model.RunMessage) (bool, error)
	ResumeWaitingInput(ctx context.Context, runID string, expectedRevision int, answer *model.RunMessage) (bool, error)
	FinishTerminal(ctx context.Context, runID string, terminal TerminalTransition, final *model.RunMessage) (bool, error)
	WithTx(ctx context.Context, fn func(repo RunRepository) error) error
}

// TerminalTransition 描述一次终态收敛的完整写入内容。
// 状态、错误信息与最终消息必须在一个事务内提交，避免出现终态无回复或回复无终态。
type TerminalTransition struct {
	Status       model.RunStatus
	Snapshot     model.RunExecutionSnapshot
	ErrorCode    string
	ErrorMessage string
}

// PostgresRunRepository 是 RunRepository 的 PostgreSQL 实现。
type PostgresRunRepository struct {
	db *infrastructure.Postgres
	tx pgx.Tx
}

// NewRunRepository 创建 Run 聚合仓储。
func NewRunRepository(db *infrastructure.Postgres) RunRepository {
	return &PostgresRunRepository{db: db}
}

func (r *PostgresRunRepository) queryer() queryer {
	return newQueryer(r.db, r.tx)
}

const runColumns = `id, session_id, status, idempotency_key, settings_snapshot, prompt_hash,
	execution_snapshot, snapshot_revision, waiting_message_id, error_code, error_message,
	started_at, finished_at, created_at, updated_at`

const runMessageColumns = `id, session_id, run_id, idempotency_key, reply_to_message_id, role, content, attachments, created_at`

// CreateWithInitialMessage 原子创建 pending Run 和首条用户输入。
//
// 同一个 session_id + idempotency_key 的客户端重试返回首个已创建的 Run，
// 不重复写入初始消息。其他唯一约束冲突（例如同一 Session 已有活跃 Run）照常返回错误。
func (r *PostgresRunRepository) CreateWithInitialMessage(ctx context.Context, run *model.Run, initial *model.RunMessage) (created *model.Run, inserted bool, err error) {
	if err := validateNewRun(run, initial); err != nil {
		return nil, false, err
	}
	err = r.WithTx(ctx, func(repo RunRepository) error {
		bound := repo.(*PostgresRunRepository)
		if err := bound.insertRun(ctx, run); err != nil {
			return fmt.Errorf("insert run: %w", err)
		}
		if err := bound.insertMessage(ctx, initial); err != nil {
			return fmt.Errorf("insert initial message: %w", err)
		}
		return nil
	})
	if err == nil {
		return run, true, nil
	}
	if !isUniqueConstraint(err, "uq_runs_session_idempotency") {
		if isUniqueConstraint(err, "idx_runs_one_active_per_session") {
			return nil, false, ErrActiveRunExists
		}
		return nil, false, err
	}

	existing, lookupErr := r.runBySessionAndIdempotencyKey(ctx, run.SessionID, run.IdempotencyKey)
	if lookupErr != nil {
		return nil, false, lookupErr
	}
	if existing == nil {
		return nil, false, fmt.Errorf("load idempotent run after unique constraint: %w", err)
	}
	return existing, false, nil
}

func isUniqueConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

func validateNewRun(run *model.Run, initial *model.RunMessage) error {
	if run == nil || initial == nil {
		return fmt.Errorf("run and initial message are required")
	}
	if run.ID == "" || run.SessionID == "" || run.IdempotencyKey == "" {
		return fmt.Errorf("run id, session id and idempotency key are required")
	}
	if run.Status != model.RunStatusPending {
		return fmt.Errorf("new run status must be %q", model.RunStatusPending)
	}
	if run.PromptHash == "" {
		return fmt.Errorf("prompt hash is required")
	}
	if err := run.SettingsSnapshot.Validate(); err != nil {
		return fmt.Errorf("validate settings snapshot: %w", err)
	}
	if initial.ID == "" || initial.RunID != run.ID || initial.SessionID != run.SessionID {
		return fmt.Errorf("initial message must belong to the created run and session")
	}
	if initial.IdempotencyKey != run.IdempotencyKey {
		return fmt.Errorf("initial message must reuse run idempotency key")
	}
	if initial.Role != model.RoleUser {
		return fmt.Errorf("initial message role must be %q", model.RoleUser)
	}
	return nil
}

func (r *PostgresRunRepository) insertRun(ctx context.Context, run *model.Run) error {
	settingsJSON, err := sonic.Marshal(run.SettingsSnapshot)
	if err != nil {
		return fmt.Errorf("encode settings snapshot: %w", err)
	}
	snapshotJSON, err := sonic.Marshal(run.ExecutionSnapshot)
	if err != nil {
		return fmt.Errorf("encode execution snapshot: %w", err)
	}
	_, err = r.queryer().Exec(ctx, `
		INSERT INTO runs (id, session_id, status, idempotency_key, settings_snapshot, prompt_hash,
			execution_snapshot, snapshot_revision, waiting_message_id, error_code, error_message,
			started_at, finished_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), $12, $13, $14, $15)
	`, run.ID, run.SessionID, run.Status, run.IdempotencyKey, settingsJSON, run.PromptHash,
		snapshotJSON, run.SnapshotRevision, run.WaitingMessageID, run.ErrorCode, run.ErrorMessage,
		run.StartedAt, run.FinishedAt, run.CreatedAt, run.UpdatedAt,
	)
	return err
}

func (r *PostgresRunRepository) insertMessage(ctx context.Context, message *model.RunMessage) error {
	if err := validateMessage(message); err != nil {
		return err
	}
	attachments, err := sonic.Marshal(message.Attachments)
	if err != nil {
		return fmt.Errorf("encode message attachments: %w", err)
	}
	_, err = r.queryer().Exec(ctx, `
		INSERT INTO messages (id, session_id, run_id, idempotency_key, reply_to_message_id, role, content, attachments, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, $9)
	`, message.ID, message.SessionID, message.RunID, message.IdempotencyKey, message.ReplyToMessageID,
		message.Role, message.Content, attachments, message.CreatedAt,
	)
	return err
}

func validateMessage(message *model.RunMessage) error {
	if message == nil {
		return fmt.Errorf("message is required")
	}
	if message.ID == "" || message.SessionID == "" || message.RunID == "" {
		return fmt.Errorf("message id, session id and run id are required")
	}
	if message.Role != model.RoleUser && message.Role != model.RoleAssistant {
		return fmt.Errorf("unsupported persistent message role %q", message.Role)
	}
	return nil
}

// GetByID 返回 Run；未找到时返回 (nil, nil)，与 SessionRepository 保持一致。
func (r *PostgresRunRepository) GetByID(ctx context.Context, id string) (*model.Run, error) {
	run, err := scanRun(r.queryer().QueryRow(ctx, `SELECT `+runColumns+` FROM runs WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return run, nil
}

// GetActiveBySessionID 返回会话当前唯一活跃 Run；无活跃 Run 时返回 nil,nil。
func (r *PostgresRunRepository) GetActiveBySessionID(ctx context.Context, sessionID string) (*model.Run, error) {
	run, err := scanRun(r.queryer().QueryRow(ctx, `
		SELECT `+runColumns+` FROM runs
		WHERE session_id = $1 AND status IN ('pending', 'running', 'waiting_input', 'cancelling')
		ORDER BY created_at DESC, id DESC LIMIT 1
	`, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (r *PostgresRunRepository) runBySessionAndIdempotencyKey(ctx context.Context, sessionID, idempotencyKey string) (*model.Run, error) {
	run, err := scanRun(r.queryer().QueryRow(ctx, `
		SELECT `+runColumns+`
		FROM runs
		WHERE session_id = $1 AND idempotency_key = $2
	`, sessionID, idempotencyKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return run, nil
}

func scanRun(s rowScanner) (*model.Run, error) {
	var run model.Run
	var settingsJSON, snapshotJSON []byte
	var waitingMessageID, errorCode, errorMessage sql.NullString
	var startedAt, finishedAt sql.NullTime
	if err := s.Scan(
		&run.ID, &run.SessionID, &run.Status, &run.IdempotencyKey, &settingsJSON, &run.PromptHash,
		&snapshotJSON, &run.SnapshotRevision, &waitingMessageID, &errorCode, &errorMessage,
		&startedAt, &finishedAt, &run.CreatedAt, &run.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := sonic.Unmarshal(settingsJSON, &run.SettingsSnapshot); err != nil {
		return nil, fmt.Errorf("decode settings snapshot: %w", err)
	}
	if err := sonic.Unmarshal(snapshotJSON, &run.ExecutionSnapshot); err != nil {
		return nil, fmt.Errorf("decode execution snapshot: %w", err)
	}
	if waitingMessageID.Valid {
		run.WaitingMessageID = waitingMessageID.String
	}
	if errorCode.Valid {
		run.ErrorCode = errorCode.String
	}
	if errorMessage.Valid {
		run.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	return &run, nil
}

// ListMessages 按稳定的创建时间和 ID 顺序读取 Run 消息。
func (r *PostgresRunRepository) ListMessages(ctx context.Context, runID string) ([]*model.RunMessage, error) {
	rows, err := r.queryer().Query(ctx, `
		SELECT `+runMessageColumns+`
		FROM messages
		WHERE run_id = $1
		ORDER BY created_at ASC, id ASC
	`, runID)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanRunMessage)
}

func scanRunMessage(s rowScanner) (*model.RunMessage, error) {
	var message model.RunMessage
	var idempotencyKey, replyToMessageID sql.NullString
	var attachments []byte
	if err := s.Scan(
		&message.ID, &message.SessionID, &message.RunID, &idempotencyKey, &replyToMessageID,
		&message.Role, &message.Content, &attachments, &message.CreatedAt,
	); err != nil {
		return nil, err
	}
	if err := sonic.Unmarshal(attachments, &message.Attachments); err != nil {
		return nil, fmt.Errorf("decode message attachments: %w", err)
	}
	if idempotencyKey.Valid {
		message.IdempotencyKey = idempotencyKey.String
	}
	if replyToMessageID.Valid {
		message.ReplyToMessageID = replyToMessageID.String
	}
	return &message, nil
}

// TransitionStatus 使用状态条件更新，避免迟到写入覆盖新的状态。
func (r *PostgresRunRepository) TransitionStatus(ctx context.Context, id string, from []model.RunStatus, to model.RunStatus) (bool, error) {
	if len(from) == 0 {
		return false, fmt.Errorf("source statuses are required")
	}
	for _, source := range from {
		if !source.CanTransitionTo(to) {
			return false, fmt.Errorf("invalid run transition %q -> %q", source, to)
		}
	}
	statuses := make([]string, len(from))
	for i, status := range from {
		statuses[i] = string(status)
	}
	tag, err := r.queryer().Exec(ctx, `
		UPDATE runs
		SET status = $3::varchar,
			started_at = CASE WHEN $3::varchar = 'running' AND started_at IS NULL THEN CURRENT_TIMESTAMP(0) ELSE started_at END,
			finished_at = CASE WHEN $3::varchar IN ('succeeded', 'failed', 'cancelled', 'interrupted') THEN CURRENT_TIMESTAMP(0) ELSE finished_at END,
			updated_at = CURRENT_TIMESTAMP(0)
		WHERE id = $1 AND status = ANY($2)
	`, id, statuses, to)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// EnterWaitingInput 原子持久化可恢复快照、问题消息和 waiting_input 状态。
func (r *PostgresRunRepository) EnterWaitingInput(ctx context.Context, runID string, expectedRevision int, snapshot model.RunExecutionSnapshot, question *model.RunMessage) (bool, error) {
	if err := snapshot.ValidateWaitingInput(); err != nil {
		return false, err
	}
	if snapshot.SnapshotRevision != expectedRevision+1 {
		return false, fmt.Errorf("snapshot revision %d must follow expected revision %d", snapshot.SnapshotRevision, expectedRevision)
	}
	if err := validateMessage(question); err != nil {
		return false, err
	}
	if question.RunID != runID || question.Role != model.RoleAssistant || question.ID != snapshot.WaitingCheckpoint.QuestionMessageID {
		return false, fmt.Errorf("waiting question must be the snapshot assistant question for this run")
	}

	changed := false
	err := r.WithTx(ctx, func(repo RunRepository) error {
		bound := repo.(*PostgresRunRepository)
		snapshotJSON, err := sonic.Marshal(snapshot.Clone())
		if err != nil {
			return fmt.Errorf("encode execution snapshot: %w", err)
		}
		var sessionID string
		err = bound.queryer().QueryRow(ctx, `
			UPDATE runs
			SET status = $2, execution_snapshot = $3, snapshot_revision = $4, waiting_message_id = $5, updated_at = CURRENT_TIMESTAMP(0)
			WHERE id = $1 AND status = 'running' AND snapshot_revision = $6
			RETURNING session_id
		`, runID, model.RunStatusWaitingInput, snapshotJSON, snapshot.SnapshotRevision, question.ID, expectedRevision).Scan(&sessionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if question.SessionID != sessionID {
			return fmt.Errorf("waiting question session does not match run session")
		}
		if err := bound.insertMessage(ctx, question); err != nil {
			return fmt.Errorf("insert waiting question: %w", err)
		}
		changed = true
		return nil
	})
	return changed, err
}

// ResumeWaitingInput 原子写入用户回答并从 waiting_input 返回 running。
// 对相同 Run 幂等键的重试返回成功，不重复写入消息。
func (r *PostgresRunRepository) ResumeWaitingInput(ctx context.Context, runID string, expectedRevision int, answer *model.RunMessage) (bool, error) {
	if err := validateMessage(answer); err != nil {
		return false, err
	}
	if answer.RunID != runID || answer.Role != model.RoleUser || answer.IdempotencyKey == "" || answer.ReplyToMessageID == "" {
		return false, fmt.Errorf("resumed input must be an idempotent user reply for this run")
	}

	changed := false
	err := r.WithTx(ctx, func(repo RunRepository) error {
		bound := repo.(*PostgresRunRepository)
		existing, err := bound.messageByIdempotencyKey(ctx, runID, answer.IdempotencyKey)
		if err != nil {
			return err
		}
		if existing != nil {
			if !sameClientInput(existing, answer) {
				return fmt.Errorf("idempotency key is already bound to different input")
			}
			run, err := bound.GetByID(ctx, runID)
			if err != nil {
				return err
			}
			if run != nil && run.Status == model.RunStatusRunning && run.SnapshotRevision == expectedRevision {
				changed = true
			}
			return nil
		}

		var sessionID string
		err = bound.queryer().QueryRow(ctx, `
			UPDATE runs
			SET status = $2, waiting_message_id = NULL, updated_at = CURRENT_TIMESTAMP(0)
			WHERE id = $1 AND status = 'waiting_input' AND snapshot_revision = $3 AND waiting_message_id = $4
			RETURNING session_id
		`, runID, model.RunStatusRunning, expectedRevision, answer.ReplyToMessageID).Scan(&sessionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if sessionID != answer.SessionID {
			return fmt.Errorf("resumed input session does not match run session")
		}
		if err := bound.insertMessage(ctx, answer); err != nil {
			return fmt.Errorf("insert resumed input: %w", err)
		}
		changed = true
		return nil
	})
	return changed, err
}

// FinishTerminal 原子写入 Run 终态和可选的最终助手消息。
// 条件更新阻止取消后的迟到成功/失败覆盖；消息插入失败会回滚状态迁移。
func (r *PostgresRunRepository) FinishTerminal(ctx context.Context, runID string, terminal TerminalTransition, final *model.RunMessage) (bool, error) {
	if !terminal.Status.IsTerminal() {
		return false, fmt.Errorf("terminal status required, got %q", terminal.Status)
	}
	if final != nil && (final.RunID != runID || final.Role != model.RoleAssistant) {
		return false, fmt.Errorf("final message must be an assistant message for this run")
	}

	changed := false
	err := r.WithTx(ctx, func(repo RunRepository) error {
		bound := repo.(*PostgresRunRepository)
		var from []string
		switch terminal.Status {
		case model.RunStatusSucceeded, model.RunStatusFailed:
			from = []string{"running"}
		case model.RunStatusCancelled:
			from = []string{"cancelling"}
		case model.RunStatusInterrupted:
			from = []string{"pending", "running"}
		default:
			return fmt.Errorf("unsupported terminal status %q", terminal.Status)
		}
		snapshotJSON, err := sonic.Marshal(terminal.Snapshot.Clone())
		if err != nil {
			return fmt.Errorf("encode terminal execution snapshot: %w", err)
		}
		var sessionID string
		err = bound.queryer().QueryRow(ctx, `
			UPDATE runs
			SET status = $2, execution_snapshot = $3, snapshot_revision = $4,
				error_code = NULLIF($5, ''), error_message = NULLIF($6, ''),
				finished_at = CURRENT_TIMESTAMP(0), updated_at = CURRENT_TIMESTAMP(0)
			WHERE id = $1 AND status = ANY($7::varchar[])
			RETURNING session_id
		`, runID, terminal.Status, snapshotJSON, terminal.Snapshot.SnapshotRevision, terminal.ErrorCode, terminal.ErrorMessage, from).Scan(&sessionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if final != nil {
			if final.SessionID != sessionID {
				return fmt.Errorf("final message session does not match run session")
			}
			if err := bound.insertMessage(ctx, final); err != nil {
				return fmt.Errorf("insert final message: %w", err)
			}
		}
		changed = true
		return nil
	})
	return changed, err
}

func (r *PostgresRunRepository) messageByIdempotencyKey(ctx context.Context, runID, idempotencyKey string) (*model.RunMessage, error) {
	message, err := scanRunMessage(r.queryer().QueryRow(ctx, `
		SELECT `+runMessageColumns+` FROM messages
		WHERE run_id = $1 AND idempotency_key = $2
	`, runID, idempotencyKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return message, nil
}

func sameClientInput(existing, incoming *model.RunMessage) bool {
	return existing.SessionID == incoming.SessionID &&
		existing.RunID == incoming.RunID &&
		existing.ReplyToMessageID == incoming.ReplyToMessageID &&
		existing.Role == incoming.Role &&
		existing.Content == incoming.Content &&
		reflect.DeepEqual(existing.Attachments, incoming.Attachments)
}

// WithTx 在同一个 Run 聚合事务中复用 Repository 实例。
func (r *PostgresRunRepository) WithTx(ctx context.Context, fn func(repo RunRepository) error) error {
	return runInTx(ctx, r.db, r.tx, func(tx pgx.Tx) RunRepository {
		return &PostgresRunRepository{db: r.db, tx: tx}
	}, fn)
}
