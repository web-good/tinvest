package app

import (
	"context"

	"tinvest/internal/service/notification/errorlog"
	"tinvest/pkg/closer"
	"tinvest/pkg/logger"
)

// initErrorLogSink подключает дублирование ERROR-логов в тему General.
// Шаг обязан идти после initTelegramBotClient: sink нужен готовый бот.
func (a *App) initErrorLogSink(ctx context.Context) error {
	if a.config.TelegramClient.Token == "" || a.config.TelegramClient.GroupChatID == 0 {
		logger.Warn("error log telegram sink disabled: TELEGRAM/TELEGRAM_GROUP_CHAT_ID are not set")

		return nil
	}

	sender, err := a.sp.GetErrorLogSender()
	if err != nil {
		return err
	}

	sink := errorlog.New(sender, a.config.AppEnv)
	sinkCtx, cancel := context.WithCancel(ctx)
	closer.Add(func() error {
		cancel()

		return nil
	})

	go sink.Run(sinkCtx)
	logger.SetErrorSink(sink)
	logger.InfoContext(ctx, "error log telegram sink enabled")

	return nil
}
