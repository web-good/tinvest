package pg

import (
	"context"
	"github.com/jmoiron/sqlx"
	"tinvest/pkg/client/db"
)

type client struct {
	db db.Db
}

func New(ctx context.Context, dsn string) (db.Client, error) {
	dbc, err := sqlx.ConnectContext(ctx, "postgres", dsn)

	if err != nil {
		return nil, err
	}

	return &client{
		db: &pg{connect: dbc, transactor: NewTransactionManager(dbc)},
	}, nil
}

func (c *client) Db() db.Db {
	return c.db
}
