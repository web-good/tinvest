package down_pump

import (
	"context"
	"tinvest/internal/service/trading_strategy/down_pump/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type DownPump interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, tgClient telegram.Client) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
	}
}
