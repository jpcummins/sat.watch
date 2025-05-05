package api

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB defines the database operations needed by the API service.
type DB interface {
	Ping(ctx context.Context) error
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	// Add other methods if needed by other API functions
}

// pgxPoolWrapper adapts a *pgxpool.Pool to the DB interface.
type pgxPoolWrapper struct {
	*pgxpool.Pool
}

// Select implements the DB interface using pgxscan.
func (w *pgxPoolWrapper) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	// Use the embedded Pool for pgxscan
	return pgxscan.Select(ctx, w.Pool, dest, query, args...)
}
