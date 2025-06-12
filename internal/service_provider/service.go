package service_provider

import (
	"tinvest/internal/service/notification/purchase_shares"
	"tinvest/internal/service/trading_strategy/rsi_trading"
)

type service struct {
	purchaseSharesService purchase_shares.PurchaseShares
	rsiTradingService     rsi_trading.RsiTrading
}

func (*ServiceProvider) GetPurchaseSharesService(address string, token string) purchase_shares.PurchaseShares {
	if serviceProvider.service.purchaseSharesService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient(address, token)
		serviceProvider.service.purchaseSharesService = purchase_shares.NewService(grpcClient.InstrumentsServiceClient(), grpcClient.MarketDataServiceClient())
	}

	return serviceProvider.service.purchaseSharesService
}

func (*ServiceProvider) GetRsiTradingService(host string, tBankToken string, chatId int64, tgToken string) rsi_trading.RsiTrading {
	if serviceProvider.service.rsiTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient(host, tBankToken)
		tgClient, _ := serviceProvider.GetTelegramBotClient(tgToken, chatId)
		serviceProvider.service.rsiTradingService = rsi_trading.NewService(grpcClient.InstrumentsServiceClient(), grpcClient.MarketDataServiceClient(), tgClient)
	}

	return serviceProvider.service.rsiTradingService
}
