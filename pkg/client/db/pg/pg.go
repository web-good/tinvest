package pg

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strconv"
	"strings"
	"tinvest/pkg/client/db"
	"tinvest/pkg/logger"
)

type pg struct {
	connect    *sqlx.DB
	transactor db.TxManager
}

func (p *pg) Ping(ctx context.Context) error {
	return p.connect.PingContext(ctx)
}

func (p *pg) Close() error {
	err := p.connect.Close()
	if err != nil {
		return err
	}

	return nil
}

func (p *pg) Transaction() db.TxManager {
	return p.transactor
}

func (p *pg) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	p.logQuery(ctx, query, args...)
	tx, ok := ctx.Value(TxKey).(*sqlx.Tx)

	if ok {
		return tx.QueryContext(ctx, query, args...)
	}

	return p.connect.QueryContext(ctx, query, args...)
}
func (p *pg) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	p.logQuery(ctx, query, args...)
	tx, ok := ctx.Value(TxKey).(*sqlx.Tx)

	if ok {
		return tx.QueryxContext(ctx, query, args...)
	}

	return p.connect.QueryxContext(ctx, query, args...)
}

func (p *pg) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	p.logQuery(ctx, query, args...)
	tx, ok := ctx.Value(TxKey).(*sqlx.Tx)

	if ok {
		return tx.QueryRowxContext(ctx, query, args...)
	}

	return p.connect.QueryRowxContext(ctx, query, args...)
}

func (p *pg) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	p.logQuery(ctx, query, args...)
	tx, ok := ctx.Value(TxKey).(*sqlx.Tx)

	if ok {
		return tx.ExecContext(ctx, query, args...)
	}

	return p.connect.ExecContext(ctx, query, args...)
}

func (p *pg) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	p.logQuery(ctx, query, args...)
	tx, ok := ctx.Value(TxKey).(*sqlx.Tx)

	if ok {
		return tx.QueryRowContext(ctx, query, args...)
	}

	return p.connect.QueryRowContext(ctx, query, args...)
}

func (p *pg) logQuery(ctx context.Context, q string, args ...interface{}) {
	logger.DebugContext(ctx, pretty(q, "$", args...))
}

func pretty(query string, placeholder string, args ...any) string {
	for i, param := range args {
		var value string
		switch v := param.(type) {
		case string:
			value = fmt.Sprintf("%q", v)
		case []byte:
			value = fmt.Sprintf("%q", string(v))
		default:
			value = fmt.Sprintf("%v", v)
		}

		query = strings.Replace(query, fmt.Sprintf("%s%s", placeholder, strconv.Itoa(i+1)), value, -1)
	}

	query = strings.ReplaceAll(query, "\t", "")
	query = strings.ReplaceAll(query, "\n", " ")

	return strings.TrimSpace(query)
}
