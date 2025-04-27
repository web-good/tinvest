package app

import (
	"context"
	"fmt"
	_ "github.com/lib/pq"
	"tinvest/pkg/client/db/pg"
	"tinvest/pkg/closer"
	"tinvest/pkg/logger"
)

func (a *App) initDatabase(ctx context.Context) error {
	var err error
	a.db, err = pg.New(
		ctx,
		a.config.Storage.Dsn(),
	)

	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}

	closer.Add(func() error {
		err := a.db.Db().Close()

		if err != nil {
			return err
		}

		return nil
	})

	err = a.db.Db().Ping(ctx)

	if err != nil {
		return fmt.Errorf("ping to database error: %w", err)
	}

	logger.InfoContext(ctx, "Connecting to database")

	return nil
}
