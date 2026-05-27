package golden_x

import (
	"context"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/service/trading_strategy/golden_x/factory"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type GoldenX interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type Option func(*service)

func WithSettings(s gxmodel.Settings) Option {
	return func(svc *service) {
		svc.settings = s
	}
}

type service struct {
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
	settings                    gxmodel.Settings
}

func NewService(
	marketDataServiceGrpcClient grpc.MarketDataServiceClient,
	tgClient telegram.Client,
	opts ...Option,
) *service {
	svc := &service{
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
		settings:                    factory.DefaultSettings(),
	}
	for _, o := range opts {
		o(svc)
	}
	return svc
}
