package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/jackc/pgx/v5"
)

// runInTx 是所有仓储 WithTx 的统一实现，避免每个仓储复制粘贴事务样板。
//
// 参数：
//   - db：连接池，用于开启新事务（current 为 nil 时必须非 nil）
//   - current：当前仓储已绑定的事务；非 nil 表示已处于事务中，直接复用
//   - bind：拿到事务后构造绑定该事务的仓储实例（如 &PostgresSessionRepository{db: db, tx: tx}）
//   - fn：在事务上下文中执行的业务逻辑
//
// 语义：
//   - 已在事务中：直接用 bind(current) 执行 fn，支持嵌套/复用，不新开事务
//   - 否则：开启新事务，fn 成功则提交、失败则回滚
//   - panic 兜底：defer 中 recover 后回滚再重新 panic，防止事务泄漏
//   - committed/rolledBack 双标志确保不会对同一事务重复回滚
func runInTx[T any](
	ctx context.Context,
	db *infrastructure.Postgres,
	current pgx.Tx,
	bind func(tx pgx.Tx) T,
	fn func(repo T) error,
) error {
	// 已在事务中：直接复用当前事务执行
	if current != nil {
		return fn(bind(current))
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	var committed bool
	var rolledBack bool
	defer func() {
		if committed || rolledBack {
			return
		}
		if p := recover(); p != nil {
			logRollbackFailure(ctx, rollbackTx(ctx, tx))
			panic(p) // 重新抛出 panic
		}
		logRollbackFailure(ctx, rollbackTx(ctx, tx))
	}()

	if err := fn(bind(tx)); err != nil {
		rolledBack = true
		return joinRollbackError(err, rollbackTx(ctx, tx))
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit transaction: %w", commitErr)
	}
	committed = true
	return nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) error {
	// 回滚不能复用业务 ctx：走到回滚往往正是因为 ctx 已取消/超时，
	// 复用会让 Rollback 立即失败、事务悬挂到连接被池销毁。
	rctx := context.WithoutCancel(ctx)
	if err := tx.Rollback(rctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("rollback transaction: %w", err)
	}
	return nil
}

func joinRollbackError(operationErr, rollbackErr error) error {
	if rollbackErr == nil {
		return operationErr
	}
	if operationErr == nil {
		return rollbackErr
	}
	return errors.Join(operationErr, rollbackErr)
}

func logRollbackFailure(ctx context.Context, err error) {
	if err != nil {
		logger.ErrorContext(ctx, "transaction rollback failed", logger.Err(err))
	}
}
