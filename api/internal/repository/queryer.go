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

// rowScanner 抽象 pgx.Row（单行）与 pgx.Rows（多行迭代）共有的 Scan 能力，
// 让各仓储的 scanXxx 映射函数既能用于 QueryRow 也能用于 Query 的行迭代。
type rowScanner interface {
	Scan(dest ...any) error
}

// collectRows 遍历 pgx.Rows，用 scan 逐行构造 T，收敛各仓储重复的
// “Query → defer Close → for Next{scan;append} → rows.Err()” 样板。
//
// scan 通常直接传仓储的 scanXxx（如 scanFile），其入参为 rowScanner，
// pgx.Rows 天然满足该接口。返回的切片在无数据时为 nil。
func collectRows[T any](rows pgx.Rows, scan func(rowScanner) (T, error)) ([]T, error) {
	defer rows.Close()
	var out []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
