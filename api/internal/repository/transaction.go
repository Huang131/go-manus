package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/jackc/pgx/v5"
)

func rollbackTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
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
