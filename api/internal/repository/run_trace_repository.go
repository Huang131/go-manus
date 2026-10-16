package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/jackc/pgx/v5"
)

// RunTraceRepository 存取 Run 的有限 UI 回放事件。
type RunTraceRepository interface {
	Append(ctx context.Context, runID string, event *model.RunTraceEvent) error
	ListByRunIDs(ctx context.Context, runIDs []string) (map[string][]*model.RunTraceEvent, error)
	ListTraceEventsByRunIDs(ctx context.Context, runIDs []string) (map[string][]*model.RunTraceEvent, error)
}

type postgresRunTraceRepository struct {
	db *infrastructure.Postgres
}

// NewRunTraceRepository 创建 PostgreSQL Run 追踪仓储。
func NewRunTraceRepository(db *infrastructure.Postgres) RunTraceRepository {
	return &postgresRunTraceRepository{db: db}
}

// Append 为同一 Run 分配严格递增的序号后写入追踪事件。
// 锁定父 Run 行可在尚无 trace 行时仍串行化首条事件，避免 MAX(sequence) 的并发竞态。
func (r *postgresRunTraceRepository) Append(ctx context.Context, runID string, event *model.RunTraceEvent) error {
	if r == nil || r.db == nil || r.db.Pool == nil {
		return errors.New("run trace repository is not initialized")
	}
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id is required")
	}
	if event == nil || !model.IsPersistableRunTraceEvent(event.Type) {
		return errors.New("persistable run trace event is required")
	}
	if !json.Valid(event.Data) {
		return errors.New("run trace event data must be valid json")
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin append run trace event: %w", err)
	}
	defer func() {
		if err := tx.Rollback(context.WithoutCancel(ctx)); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			// Commit 后的 ErrTxClosed 是预期状态；其他错误由原始操作决定返回值。
		}
	}()

	var lockedRunID string
	if err := tx.QueryRow(ctx, `SELECT id FROM runs WHERE id = $1 FOR UPDATE`, runID).Scan(&lockedRunID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("run %q not found", runID)
		}
		return fmt.Errorf("lock run for trace event: %w", err)
	}
	var sequence int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM run_trace_events WHERE run_id = $1`, runID).Scan(&sequence); err != nil {
		return fmt.Errorf("next run trace sequence: %w", err)
	}
	createdAt := event.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO run_trace_events (run_id, sequence, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, runID, sequence, event.Type, []byte(event.Data), createdAt); err != nil {
		return fmt.Errorf("insert run trace event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit run trace event: %w", err)
	}
	return nil
}

// ListByRunIDs 一次读取多个 Run 的追踪事件，并以 Run ID 分组保留写入顺序。
func (r *postgresRunTraceRepository) ListByRunIDs(ctx context.Context, runIDs []string) (map[string][]*model.RunTraceEvent, error) {
	result := make(map[string][]*model.RunTraceEvent)
	if r == nil || r.db == nil || r.db.Pool == nil {
		return nil, errors.New("run trace repository is not initialized")
	}
	ids := uniqueNonEmptyStrings(runIDs)
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.db.Pool.Query(ctx, `
		SELECT run_id, sequence, event_type, payload, created_at
		FROM run_trace_events
		WHERE run_id = ANY($1)
		ORDER BY run_id ASC, sequence ASC
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("list run trace events: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		event := &model.RunTraceEvent{}
		if err := rows.Scan(&event.RunID, &event.Sequence, &event.Type, &event.Data, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan run trace event: %w", err)
		}
		result[event.RunID] = append(result[event.RunID], event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run trace events: %w", err)
	}
	return result, nil
}

// ListTraceEventsByRunIDs 适配应用层的历史回放读取边界，复用同一批量查询实现。
func (r *postgresRunTraceRepository) ListTraceEventsByRunIDs(ctx context.Context, runIDs []string) (map[string][]*model.RunTraceEvent, error) {
	return r.ListByRunIDs(ctx, runIDs)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
