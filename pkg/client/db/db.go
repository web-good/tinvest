package db

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Handler func(ctx context.Context) error

type Client interface {
	Db() Db
}

type Db interface {
	Ping(context.Context) error
	Close() error
	Query
	Transaction() TxManager
}

type Ping interface {
	Ping(ctx context.Context) error
}

type Query interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type TxManager interface {
	ReadCommitted(ctx context.Context, f Handler) error
}
