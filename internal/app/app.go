package app

import (
	"context"
	"log/slog"
	"sync"
	"tinvest/internal/config"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	mr "tinvest/internal/service/trading_strategy/macd_rsi/scheduler"
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
	/*loc, _ := time.LoadLocation("Europe/Moscow")
	a.sp.GetMacdRsiTradingService().BackTest(ctx, dto.BackTest{
		AtrInterval: enum.Day1,
		Interval:    enum.Hour1,
		//	DateFrom: time.Date(2025, time.July, 18, 9, 20, 0, 0, time.UTC),
		DateFrom: time.Now().AddDate(0, -1, 0).In(loc),
		DateTo:   time.Now().AddDate(0, 0, 0).In(loc),
		InstrumentID: []string{
			//"e6123145-9665-43e0-8413-cd61b8aa9b13",//сбер
			//"87db07bc-0e02-4e29-90bb-05e8ef791d7b",//Тбанк
			//	"ab1f751e-15b2-4c74-802c-1b3e8638c394", //софт
			"7de75794-a27f-4d81-a39b-492345813822", //яндекс
			"02cfdf61-6298-4c0f-a9ca-9cabc82afaf3", //лукойл
			"eb4ba863-e85f-4f80-8c29-f2627938ee58", //мечел
		},
	})*/
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
	wg.Add(1)
	/*go func() {
		defer wg.Done()
		err := a.sp.GetMacdRsiTradingService().Trade(ctx, dto.Trade{AtrInterval: enum.Day1, Interval: enum.Hour1, Scheduler: "*35 * * * *"})

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi", err.Error())
		}
	}()*/

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetMacdRsiTradingService().TakeProfit(ctx, dto.TakeProfit{Interval: enum.Hour1, ATRInterval: enum.Day1})

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi", err.Error())
		}
	}()
	wg.Wait()
}

func (a *App) runProd(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		sh := mr.NewSchedulerService(a.sp.GetMacdRsiTradingService())
		err := sh.Trade(ctx, dto.Trade{AtrInterval: enum.Day1, Interval: enum.Hour1, Scheduler: "5 8-23 * * 1-5"})

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi 1H", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		sh := mr.NewSchedulerService(a.sp.GetMacdRsiTradingService())
		err := sh.TakeProfit(ctx, dto.TakeProfit{Interval: enum.Hour1, ATRInterval: enum.Day1, Scheduler: "*/2 * * * *"})

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker macd rsi 1H take profit", err.Error())
		}
	}()
	wg.Wait()
}
