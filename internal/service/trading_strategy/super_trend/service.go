package super_trend

import (
	"context"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

var _ SuperTrend = (*service)(nil)

type SuperTrend interface {
	Trade(ctx context.Context) error
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, tgClient telegram.Client) SuperTrend {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
	}
}
