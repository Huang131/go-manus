package repository

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// queryer 抽象连接池和事务共有的数据库操作。
// pgx.Tx 与 pgxpool.Pool 都满足该接口，避免各仓储重复编写转发包装。
type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// newQueryer 根据仓储状态选择事务或连接池作为执行者。
func newQueryer(db *infrastructure.Postgres, tx pgx.Tx) queryer {
	if tx != nil {
		return tx
	}
	return db.Pool
}
