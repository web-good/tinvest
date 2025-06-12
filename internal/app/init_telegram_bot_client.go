package app

import (
	"context"
	"tinvest/pkg/logger"
)

func (a *App) initTelegramBotClient(ctx context.Context) error {
	logger.InfoContext(ctx, "Start telegram bot client")
	_, err := a.sp.GetTelegramBotClient(a.config.TelegramClient.Token, a.config.TelegramClient.ChatID)

	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "Started telegram bot client")

	return nil
}
