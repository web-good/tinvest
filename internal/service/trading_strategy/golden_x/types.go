package golden_x

import (
	"context"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type GoldenX interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type service struct {
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
}

func NewService(
	marketDataServiceGrpcClient grpc.MarketDataServiceClient,
	tgClient telegram.Client,
) *service {
	return &service{
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
	}
}
