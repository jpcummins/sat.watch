package api

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	Ping(ctx context.Context) error
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Select(ctx context.Context, dest any, query string, args ...any) error
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type pgxPoolWrapper struct {
	*pgxpool.Pool
}

func (w *pgxPoolWrapper) Select(ctx context.Context, dest any, query string, args ...any) error {
	return pgxscan.Select(ctx, w.Pool, dest, query, args...)
}
