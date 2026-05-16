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
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
	state                       *alertState
}

func NewService(
	instrumentsServiceClient grpc.InstrumentsServiceClient,
	marketDataServiceGrpcClient grpc.MarketDataServiceClient,
	tgClient telegram.Client,
) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
		state:                       newAlertState(),
	}
}
