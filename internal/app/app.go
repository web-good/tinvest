package app

import (
	"context"
	"log/slog"
	"sync"
	"tinvest/internal/config"
	"tinvest/internal/enum"
	analyzescheduler "tinvest/internal/service/portfolio/analyze/scheduler"
	yieldscheduler "tinvest/internal/service/portfolio/yield/scheduler"
	bondsscheduler "tinvest/internal/service/trading_strategy/bonds/scheduler"
	goldenx "tinvest/internal/service/trading_strategy/golden_x/dto"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/internal/service/trading_strategy/golden_x/scheduler"
	scalpingdto "tinvest/internal/service/trading_strategy/scalping/dto"
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
	/*wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetSuperTrendTradingService().Trade(ctx)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker super trend", err.Error())
		}
	}()*/

	/*	go func() {
			defer wg.Done()
			err := a.sp.GetSuperTrendTradingService().TakeProfit(ctx)

			if err != nil {
				logger.ErrorContext(ctx, "Error in worker super trend", err.Error())
			}
		}()

		/*wg.Add(1)
		go func() {
			defer wg.Done()
			err := a.sp.Get200EmaService().Trade(ctx, input.Trade{
				Interval: input.Hour1,
			})

			if err != nil {
				logger.ErrorContext(ctx, "Error in worker super trend", err.Error())
			}
		}()*/
	/*wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetMacdRsiTradingService().Trade(ctx, dto.Trade{AtrInterval: enum.Day1, Interval: enum.Hour1, Scheduler: "*35 * * * *"})

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi", err.Error())
		}
	}()*/
	/*
		wg.Add(1)
		go func() {
			defer wg.Done()
			sh := mr.NewSchedulerService(a.sp.GetMacdRsiTradingService())
			err := sh.TakeProfit(ctx, dto.TakeProfit{Interval: enum.Hour1, ATRInterval: enum.Day1, Scheduler: ""})

			if err != nil {
				logger.ErrorContext(ctx, "Error in worker macd rsi 1H take profit", err.Error())
			}
		}()
	*/

	/*wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetScalpingRsiTradingService().Trade(ctx, scalpinrsi.Trade{AtrInterval: enum.Day1, Interval: enum.Hour1, Scheduler: "*35 * * * *"})

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi", err.Error())
		}
	}()
	*/
	/*wg.Add(1)
	go func() {
		defer wg.Done()

		err := a.sp.GetBondsTradingService().Trade(
			ctx,
		)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker golden X strategy", err.Error())
		}
	}()
	/*wg.Add(1)
	go func() {
		defer wg.Done()

		err := a.sp.GetAnalyze().BondsPortfolio(
			ctx,
			a.config.TelegramClient.ChatID[0],
		)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker golden X strategy")
		}
	}()
	*/
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetPortfolioYield().PortfolioYieldYTD(ctx, a.config.TelegramClient.ChatID[0])
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Portfolio Yield YTD", err.Error())
		}
	}()
	/*
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := a.sp.GetGoldenXTradingService().Trade(
				ctx,
				goldenx.Trade{
					Kind:           gxmodel.StrategyKindGrowth,
					Interval:       enum.Week1,
					Scheduler:      "* * * * *",
					ShareList:      *a.collection.GrowthShare,
					UseTrendFilter: true,
				},
			)

			if err != nil {
				logger.ErrorContext(ctx, "Error in worker golden X strategy")
			}
		}()
	*/
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetScalpingTradingService().Trade(ctx, scalpingdto.Trade{
			Scheduler: "0 8-23 * * 1-5",
		})
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Scalping", err.Error())
		}
	}()
	wg.Wait()
}

func (a *App) runProd(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(6)
	/*go func() {
		defer wg.Done()
		sh := mr.NewSchedulerService(a.sp.GetMacdRsiTradingService())
		err := sh.Trade(ctx, dto.Trade{AtrInterval: enum.Day1, Interval: enum.Hour1, Scheduler: "5 8-23 * * 1-5"})

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi 1H", err.Error())
		}
	}()*/

	/*go func() {
	defer wg.Done()
	sh := mr.NewSchedulerService(a.sp.GetMacdRsiTradingService())
	err := sh.TakeProfit(ctx, dto.TakeProfit{Interval: enum.Hour1, ATRInterval: enum.Day1, Scheduler: "*/ //2 8-23 * * *"})

	/*if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi 1H take profit", err.Error())
		}
	}()*/
	go func() {
		defer wg.Done()
		err := bondsscheduler.NewScheduler(a.sp.GetBondsTradingService()).Trade(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker golden X strategy ShareTip:2", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := analyzescheduler.NewScheduler(a.sp.GetAnalyze()).BondsPortfolio(ctx, a.config.TelegramClient.ChatID[0])
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Bonds Portfolio Analyze", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := yieldscheduler.NewScheduler(a.sp.GetPortfolioYield()).PortfolioYieldYTD(ctx, a.config.TelegramClient.ChatID[0])
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Portfolio Yield YTD", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		tgBot, tgErr := a.sp.GetTelegramBotClient()
		if tgErr != nil {
			logger.ErrorContext(ctx, "telegram bot init failed for golden X dividend", tgErr.Error())
			return
		}
		err := scheduler.NewSchedulerService(a.sp.GetGoldenXTradingService(), tgBot).Trade(
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
		tgBot, tgErr := a.sp.GetTelegramBotClient()
		if tgErr != nil {
			logger.ErrorContext(ctx, "telegram bot init failed for golden X growth", tgErr.Error())
			return
		}
		err := scheduler.NewSchedulerService(a.sp.GetGoldenXTradingService(), tgBot).Trade(
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
	/*go func() {
		defer wg.Done()
		err := scalpingscheduler.NewSchedulerService(a.sp.GetScalpingTradingService()).Trade(
			ctx,
			scalpingdto.Trade{
				Scheduler: "1 8-23 * * 1-5",
			},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Scalping", err.Error())
		}
	}()*/
	wg.Wait()
}
