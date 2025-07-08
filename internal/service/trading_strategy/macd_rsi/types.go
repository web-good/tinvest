package rsi_trading

import (
	"context"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

var _ RsiTrading = (*service)(nil)

type RsiTrading interface {
	Trade(ctx context.Context, interval int) error
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, tgClient telegram.Client) RsiTrading {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
	}
}
