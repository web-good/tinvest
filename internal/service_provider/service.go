package service_provider

import (
	"tinvest/internal/service/instrument/atr"
	"tinvest/internal/service/instrument/ema"
	"tinvest/internal/service/instrument/macd"
	"tinvest/internal/service/instrument/rsi"
	"tinvest/internal/service/instrument/volatility"
	"tinvest/internal/service/notification/purchase_shares"
	"tinvest/internal/service/portfolio/analyze"
	"tinvest/internal/service/portfolio/yield"
	"tinvest/internal/service/trading_strategy/bonds"
	"tinvest/internal/service/trading_strategy/golden_x"
	"tinvest/internal/service/trading_strategy/reversion/live"
	"tinvest/internal/service/trading_strategy/scalping"
)

type service struct {
	purchaseSharesService  purchase_shares.PurchaseShares
	scalpingTradingService scalping.Scalping
	reversionLiveService   live.Service
	goldenXTradingService  golden_x.GoldenX
	bondsTradingService    bonds.Bonds
	emaInstrument          ema.Instrument
	atrInstrument          atr.Instrument
	rsiInstrument          rsi.Instrument
	MACDInstrument         macd.Instrument
	volatilityInstrument   volatility.Instrument
	analyze                analyze.Analyze
	portfolioYield         yield.Yield
}

func (*ServiceProvider) GetPurchaseSharesService() purchase_shares.PurchaseShares {
	if serviceProvider.service.purchaseSharesService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.purchaseSharesService = purchase_shares.NewService(grpcClient.InstrumentsServiceClient(), grpcClient.MarketDataServiceClient())
	}

	return serviceProvider.service.purchaseSharesService
}

func (*ServiceProvider) GetBondsTradingService() bonds.Bonds {
	if serviceProvider.service.bondsTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.bondsTradingService = bonds.NewService(
			serviceProvider.RSI(),
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			tgClient,
		)
	}

	return serviceProvider.service.bondsTradingService
}

func (*ServiceProvider) GetGoldenXTradingService() golden_x.GoldenX {
	if serviceProvider.service.goldenXTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.goldenXTradingService = golden_x.NewService(
			grpcClient.MarketDataServiceClient(),
			tgClient,
		)
	}

	return serviceProvider.service.goldenXTradingService
}

func (*ServiceProvider) Ema() ema.Instrument {
	if serviceProvider.service.emaInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.emaInstrument = ema.NewEma(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.emaInstrument
}

func (*ServiceProvider) Atr() atr.Instrument {
	if serviceProvider.service.atrInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.atrInstrument = atr.NewAtr(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.atrInstrument
}

func (*ServiceProvider) MACD() macd.Instrument {
	if serviceProvider.service.MACDInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.MACDInstrument = macd.New(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.MACDInstrument
}

func (*ServiceProvider) RSI() rsi.Instrument {
	if serviceProvider.service.rsiInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.rsiInstrument = rsi.New(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.rsiInstrument
}

func (*ServiceProvider) Volatility() volatility.Instrument {
	if serviceProvider.service.volatilityInstrument == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.volatilityInstrument = volatility.New(
			grpcClient.MarketDataServiceClient(),
		)
	}

	return serviceProvider.service.volatilityInstrument
}

func (*ServiceProvider) GetAnalyze() analyze.Analyze {
	if serviceProvider.service.analyze == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.analyze = analyze.NewService(
			grpcClient.OperationsServiceClient(),
			grpcClient.UserServiceClient(),
			tgClient,
			grpcClient.InstrumentsServiceClient(),
		)
	}

	return serviceProvider.service.analyze
}

func (*ServiceProvider) GetPortfolioYield() yield.Yield {
	if serviceProvider.service.portfolioYield == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.portfolioYield = yield.NewService(
			grpcClient.OperationsServiceClient(),
			grpcClient.UserServiceClient(),
			tgClient,
			serviceProvider.appConfig.PortfolioYield,
		)
	}

	return serviceProvider.service.portfolioYield
}

func (*ServiceProvider) GetScalpingTradingService() scalping.Scalping {
	if serviceProvider.service.scalpingTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.scalpingTradingService = scalping.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			tgClient,
			serviceProvider.appConfig.Scalping.AccountID,
		)
	}

	return serviceProvider.service.scalpingTradingService
}

func (*ServiceProvider) GetReversionLiveService() live.Service {
	if serviceProvider.service.reversionLiveService == nil {
		grpcClient, _ := serviceProvider.GetReversionGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.reversionLiveService = live.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			grpcClient.OrdersServiceClient(),
			grpcClient.StopOrdersServiceClient(),
			tgClient,
			serviceProvider.appConfig.Reversion,
		)
	}

	return serviceProvider.service.reversionLiveService
}
