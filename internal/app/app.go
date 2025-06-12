package app

import (
	"context"
	"log/slog"
	"sync"
	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/rsi_trading/scheduler"
	"tinvest/internal/service_provider"
	"tinvest/pkg/closer"
	"tinvest/pkg/logger"
)

type App struct {
	config *config.Config
	sp     *service_provider.ServiceProvider
}

func InitApp(ctx context.Context) (app *App, err error) {
	app = &App{}
	app.initLogger()
	logger.Info("init t-invest application")
	err = app.initializationLoop(ctx)

	if err != nil {
		return
	}

	logger.Info("app started", slog.String("APP_ENV", app.config.AppEnv))

	return
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		closer.CloseAll()
		closer.Wait()
	}()

	logger.Info("starting App", slog.String("APP_ENV", a.config.AppEnv))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		rsiResult := scheduler.NewSchedulerService(
			a.sp.GetRsiTradingService(
				a.config.GrpcClient.AddressProd,
				a.config.GrpcClient.TokenProd,
				a.config.TelegramClient.ChatID,
				a.config.TelegramClient.Token,
			),
		)
		err := rsiResult.Trade(ctx, 1)

		if err != nil {
			logger.ErrorContext(ctx, "Ошибка при работе воркера MacD Rsi", err.Error())
		}
	}()

	wg.Wait()

	return nil
}

func (a *App) initializationLoop(ctx context.Context) (err error) {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initServiceProvider,
		//	a.initDatabase,
		a.initGrpcClient,
		a.initTelegramBotClient,
	}
	err = nil

	for _, f := range inits {
		err = f(ctx)

		if err != nil {
			logger.ErrorContext(ctx, err.Error(), err)

			return
		}
	}

	return
}
