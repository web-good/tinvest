package app

import (
	"context"
	"fmt"
	_ "github.com/lib/pq"
	"tinvest/pkg/logger"
)

func (a *App) initDatabase(ctx context.Context) error {
	logger.InfoContext(ctx, "Start connect to database")
	fmt.Println(a.config)
	_, err := a.sp.GetDbClient(a.config.Storage.Dsn())

	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "Connecting to database")

	return nil
}
