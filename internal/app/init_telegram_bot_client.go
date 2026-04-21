package app

import (
	"context"

	"tinvest/pkg/logger"
)

func (a *App) initTelegramBotClient(ctx context.Context) error {
	logger.InfoContext(ctx, "Start telegram bot client")
	if a.config.AppEnv == "dev" {
		_, err := a.sp.GetTelegramBotClient()
		if err != nil {
			return err
		}
	} else {
		_, err := a.sp.GetTelegramBotClient()
		if err != nil {
			return err
		}
	}

	logger.InfoContext(ctx, "Started telegram bot client")

	return nil
}
