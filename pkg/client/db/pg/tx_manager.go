package pg

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"tinvest/pkg/client/db"
)

const (
	TxKey = "tx"
)

type txManager struct {
	connect *sqlx.DB
}

func NewTransactionManager(connect *sqlx.DB) db.TxManager {
	return &txManager{connect: connect}
}

func (m *txManager) ReadCommitted(ctx context.Context, f db.Handler) error {
	txOpts := &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}

	return m.transaction(ctx, txOpts, f)
}

func (m *txManager) transaction(ctx context.Context, txOptions *sql.TxOptions, fn db.Handler) error {
	var err error
	tx, ok := ctx.Value(TxKey).(*sqlx.Tx)

	if ok {
		return fn(ctx)
	}

	tx, err = m.connect.BeginTxx(ctx, txOptions)

	if err != nil {
		return fmt.Errorf("can't begin transaction: %w", err)
	}

	ctx = context.WithValue(ctx, TxKey, tx)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}

		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrapf(err, "errRollback: %v", errRollback)
			}

			return
		}

		if nil == err {
			err = tx.Commit()
			if err != nil {
				err = errors.Wrap(err, "tx commit failed")
			}
		}
	}()

	if err = fn(ctx); err != nil {
		err = errors.Wrap(err, "failed executing code in side transaction")
	}

	return err
}
