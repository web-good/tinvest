package app

import (
	"context"
	"log/slog"
	"sync"
	"tinvest/internal/config"
	st "tinvest/internal/service/trading_strategy/super_trend/scheduler"
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

	if a.config.AppEnv == "prod" {
		a.runProd(ctx)

		return nil
	}

	a.runDev(ctx)

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

func (a *App) runDev(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetSuperTrendTradingService().Trade(ctx)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker super trend", err.Error())
		}
	}()
	/*wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetMacdRsiTradingService().Trade(ctx)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi", err.Error())
		}
	}()*/
	wg.Wait()
}

func (a *App) runProd(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		sh := st.NewSchedulerService(a.sp.GetSuperTrendTradingService())
		err := sh.Trade(ctx)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker super trend", err.Error())
		}
	}()
	/*wg.Add(1)
	go func() {
		defer wg.Done()
		sh := mdrs.NewSchedulerService(a.sp.GetMacdRsiTradingService())
		err := sh.Trade(ctx)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi", err.Error())
		}
	}()*/
	wg.Wait()
}
