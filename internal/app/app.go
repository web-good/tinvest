package app

import (
	"context"
	"log/slog"
	"sync"
	"tinvest/internal/config"
	"tinvest/internal/enum"
	newsscheduler "tinvest/internal/service/news/scheduler"
	goldenx "tinvest/internal/service/trading_strategy/golden_x/dto"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/internal/service/trading_strategy/golden_x/scheduler"
	reversiondto "tinvest/internal/service/trading_strategy/reversion/live/dto"
	reversionscheduler "tinvest/internal/service/trading_strategy/reversion/live/scheduler"
	rsipullbackdto "tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	rsipullbackscheduler "tinvest/internal/service/trading_strategy/rsi_pullback/live/scheduler"
	"tinvest/internal/service_provider"
	"tinvest/pkg/closer"
	"tinvest/pkg/logger"
)

type App struct {
	config     *config.Config
	sp         *service_provider.ServiceProvider
	collection *Collection
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
		a.initCollection,
		a.initServiceProvider,
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
	wg.Add(3)
	go func() {
		defer wg.Done()
		listener, err := a.sp.GetTelegramCommands()
		if err != nil {
			logger.ErrorContext(ctx, "telegram commands init failed", err.Error())
			return
		}
		logger.InfoContext(ctx, "telegram commands listener started")
		listener.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		svc, err := a.sp.GetNewsService()
		if err != nil {
			logger.ErrorContext(ctx, "news service init failed", err.Error())
			return
		}
		// Dev: один немедленный прогон для ручной проверки дайджеста.
		if err := svc.Run(ctx); err != nil {
			logger.ErrorContext(ctx, "news digest run failed", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := rsipullbackscheduler.NewSchedulerService(a.sp.GetRSIPullbackLiveService()).Run(
			ctx,
			rsipullbackdto.Run{Scheduler: a.config.RSIPullback.Schedule},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker RSI Pullback", err.Error())
		}
	}()

	wg.Wait()
}

func (a *App) runProd(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(7)
	go func() {
		defer wg.Done()
		listener, err := a.sp.GetTelegramCommands()
		if err != nil {
			logger.ErrorContext(ctx, "telegram commands init failed", err.Error())
			return
		}
		logger.InfoContext(ctx, "telegram commands listener started")
		listener.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		svc, err := a.sp.GetNewsService()
		if err != nil {
			logger.ErrorContext(ctx, "news service init failed", err.Error())
			return
		}
		// 5-я минута часа: дайджест собирает полный прошедший час.
		if err := newsscheduler.NewSchedulerService(svc).Run(ctx, "5 * * * *"); err != nil {
			logger.ErrorContext(ctx, "Error in worker News digest", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := scheduler.NewSchedulerService(a.sp.GetGoldenXTradingService()).Trade(
			ctx,
			goldenx.Trade{
				Kind:           gxmodel.StrategyKindDividend,
				Interval:       enum.Week1,
				Scheduler:      "0 */5 * * *",
				ShareList:      *a.collection.GoldInstruments,
				UseTrendFilter: true,
			},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker golden X strategy ShareTip:1", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := scheduler.NewSchedulerService(a.sp.GetGoldenXTradingService()).Trade(
			ctx,
			goldenx.Trade{
				Kind:           gxmodel.StrategyKindGrowth,
				Interval:       enum.Week1,
				Scheduler:      "0 */5 * * *",
				ShareList:      *a.collection.GrowthShare,
				UseTrendFilter: true,
			},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker golden X strategy ShareTip:2", err.Error())
		}
	}()

	go func() {
		defer wg.Done()
		err := reversionscheduler.NewSchedulerService(a.sp.GetReversionLiveService()).Run(
			ctx,
			reversiondto.Run{Scheduler: "0 7-23 * * 1-5", Mode: reversiondto.ModeBuy},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Reversion buy", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := reversionscheduler.NewSchedulerService(a.sp.GetReversionLiveService()).Run(
			ctx,
			reversiondto.Run{Scheduler: "0 7-23,0 * * *", Mode: reversiondto.ModeManage},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Reversion manage", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := rsipullbackscheduler.NewSchedulerService(a.sp.GetRSIPullbackLiveService()).Run(
			ctx,
			rsipullbackdto.Run{Scheduler: a.config.RSIPullback.Schedule},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker RSI Pullback", err.Error())
		}
	}()
	wg.Wait()
}
