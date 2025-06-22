package service_provider

import (
	"tinvest/internal/service/notification/purchase_shares"
	"tinvest/internal/service/trading_strategy/rsi_trading"
	"tinvest/internal/service/trading_strategy/super_trend"
)

type service struct {
	purchaseSharesService    purchase_shares.PurchaseShares
	rsiTradingService        rsi_trading.RsiTrading
	superTrendTradingService super_trend.SuperTrend
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
			tgClient,
		)
	}

	return serviceProvider.service.superTrendTradingService
}
