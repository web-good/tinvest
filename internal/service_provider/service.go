package service_provider

import (
	"tinvest/internal/service/instrument/atr"
	"tinvest/internal/service/instrument/ema"
	"tinvest/internal/service/notification/purchase_shares"
	"tinvest/internal/service/trading_strategy/ema200"
	"tinvest/internal/service/trading_strategy/macd_rsi"
	"tinvest/internal/service/trading_strategy/super_trend"
)

type service struct {
	purchaseSharesService    purchase_shares.PurchaseShares
	macdRsiTradingService    macd_rsi.MacdRsi
	superTrendTradingService super_trend.SuperTrend
	ema200                   ema200.Ema200
	emaInstrument            ema.Instrument
	atrInstrument            atr.Instrument
}

func (*ServiceProvider) GetPurchaseSharesService() purchase_shares.PurchaseShares {
	if serviceProvider.service.purchaseSharesService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.purchaseSharesService = purchase_shares.NewService(grpcClient.InstrumentsServiceClient(), grpcClient.MarketDataServiceClient())
	}

	return serviceProvider.service.purchaseSharesService
}

func (*ServiceProvider) GetMacdRsiTradingService() macd_rsi.MacdRsi {
	if serviceProvider.service.macdRsiTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.macdRsiTradingService = macd_rsi.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			serviceProvider.Atr(),
			tgClient,
		)
	}

	return serviceProvider.service.macdRsiTradingService
}

func (*ServiceProvider) GetSuperTrendTradingService() super_trend.SuperTrend {
	if serviceProvider.service.superTrendTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.superTrendTradingService = super_trend.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			serviceProvider.Ema(),
			serviceProvider.Atr(),
			tgClient,
		)
	}

	return serviceProvider.service.superTrendTradingService
}

func (*ServiceProvider) Get200EmaService() ema200.Ema200 {
	if serviceProvider.service.ema200 == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.ema200 = ema200.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			serviceProvider.Ema(),
			serviceProvider.Atr(),
			tgClient,
		)
	}

	return serviceProvider.service.ema200
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
