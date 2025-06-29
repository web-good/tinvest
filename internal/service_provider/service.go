package service_provider

import (
	"tinvest/internal/service/instrument/atr"
	"tinvest/internal/service/instrument/ema"
	"tinvest/internal/service/notification/purchase_shares"
	"tinvest/internal/service/trading_strategy/rsi_trading"
	"tinvest/internal/service/trading_strategy/super_trend"
)

type service struct {
	purchaseSharesService    purchase_shares.PurchaseShares
	rsiTradingService        rsi_trading.RsiTrading
	superTrendTradingService super_trend.SuperTrend
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

func (*ServiceProvider) GetRsiTradingService() rsi_trading.RsiTrading {
	if serviceProvider.service.rsiTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.rsiTradingService = rsi_trading.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			tgClient,
		)
	}

	return serviceProvider.service.rsiTradingService
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
