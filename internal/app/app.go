package app

import (
	"context"
	"log/slog"
	"sync"
	"tinvest/internal/config"
	"tinvest/internal/enum"
	goldenx "tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/service/trading_strategy/golden_x/scheduler"
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		tgBot, _ := a.sp.GetTelegramBotClient()
		err := scheduler.NewSchedulerService(a.sp.GetGoldenXTradingService(), tgBot).Trade(
			ctx,
			goldenx.Trade{
				Interval:  enum.Week1,
				Scheduler: "*/2 * * * *",
				ShareList: []goldenx.Share{
					{ID: "962e2a95-02a9-4171-abd7-aa198dbe643a", RSILength: 12, Name: "Газпром", AverageDevident: 13.1},
					{ID: "a797f14a-8513-4b84-b15e-a3b98dc4cc00", RSILength: 14, Name: "Сургутнефтегаз - прив", AverageDevident: 5.4},
					{ID: "efdb54d3-2f92-44da-b7a3-8849e96039f6", RSILength: 11, Name: "Татнефть - прив", AverageDevident: 58.1},
					{ID: "fd417230-19cf-4e7b-9623-f7c9ca18ec6b", RSILength: 9, Name: "Роснефть", AverageDevident: 30.2},
					{ID: "02cfdf61-6298-4c0f-a9ca-9cabc82afaf3", RSILength: 10, Name: "Лукойл", AverageDevident: 559},
					{ID: "c190ff1f-1447-4227-b543-316332699ca5", RSILength: 11, Name: "Сбер Банк - прив", AverageDevident: 18.4},
					{ID: "e1b089f3-9bf1-44c3-897f-25e9f591bebc", RSILength: 14, Name: "Ростелеком - прив", AverageDevident: 5.3},
					{ID: "cd8063ad-73ad-4b31-bd0d-93138d9e99a2", RSILength: 13, Name: "МТС", AverageDevident: 31.4},
					{ID: "fa6aae10-b8d5-48c8-bbfd-d320d925d096", RSILength: 13, Name: "Северсталь", AverageDevident: 161.3},
					{ID: "161eb0d0-aaac-4451-b374-f5d0eeb1b508", RSILength: 13, Name: "НЛМК", AverageDevident: 23.7},
					{ID: "7132b1c9-ee26-4464-b5b5-1046264b61d9", RSILength: 15, Name: "ММК", AverageDevident: 4.2},
					{ID: "9978b56f-782a-4a80-a4b1-a48cbecfd194", RSILength: 14, Name: "ФосАгро", AverageDevident: 484},
				},
			},
		)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker golden X strategy", err.Error())
		}
	}()

	wg.Wait()
}

func (a *App) runProd(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(1)
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
		tgBot, _ := a.sp.GetTelegramBotClient()
		err := scheduler.NewSchedulerService(a.sp.GetGoldenXTradingService(), tgBot).Trade(
			ctx,
			goldenx.Trade{
				Interval:  enum.Week1,
				Scheduler: "0 */2 * * *",
				ShareList: []goldenx.Share{
					{ID: "962e2a95-02a9-4171-abd7-aa198dbe643a", RSILength: 12, Name: "Газпром", AverageDevident: 13.1},
					{ID: "a797f14a-8513-4b84-b15e-a3b98dc4cc00", RSILength: 14, Name: "Сургутнефтегаз - прив", AverageDevident: 5.4},
					{ID: "efdb54d3-2f92-44da-b7a3-8849e96039f6", RSILength: 11, Name: "Татнефть - прив", AverageDevident: 58.1},
					{ID: "fd417230-19cf-4e7b-9623-f7c9ca18ec6b", RSILength: 9, Name: "Роснефть", AverageDevident: 30.2},
					{ID: "02cfdf61-6298-4c0f-a9ca-9cabc82afaf3", RSILength: 10, Name: "Лукойл", AverageDevident: 559},
					{ID: "c190ff1f-1447-4227-b543-316332699ca5", RSILength: 11, Name: "Сбер Банк - прив", AverageDevident: 18.4},
					{ID: "e1b089f3-9bf1-44c3-897f-25e9f591bebc", RSILength: 14, Name: "Ростелеком - прив", AverageDevident: 5.3},
					{ID: "cd8063ad-73ad-4b31-bd0d-93138d9e99a2", RSILength: 13, Name: "МТС", AverageDevident: 31.4},
					{ID: "fa6aae10-b8d5-48c8-bbfd-d320d925d096", RSILength: 13, Name: "Северсталь", AverageDevident: 161.3},
					{ID: "161eb0d0-aaac-4451-b374-f5d0eeb1b508", RSILength: 13, Name: "НЛМК", AverageDevident: 23.7},
					{ID: "7132b1c9-ee26-4464-b5b5-1046264b61d9", RSILength: 15, Name: "ММК", AverageDevident: 4.2},
					{ID: "9978b56f-782a-4a80-a4b1-a48cbecfd194", RSILength: 14, Name: "ФосАгро", AverageDevident: 484},
				},
			},
		)

		if err != nil {
			logger.ErrorContext(ctx, "Error in worker golden X strategy", err.Error())
		}
	}()
	wg.Wait()
}
