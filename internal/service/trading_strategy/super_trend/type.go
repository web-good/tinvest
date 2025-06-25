package super_trend

import (
	"context"
	"time"
	domainema "tinvest/internal/domain/ema"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

var _ SuperTrend = (*service)(nil)

type SuperTrend interface {
	Trade(ctx context.Context) error
}

type emaInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, period int32) ([]domainema.ItemTechAnalyse, error)
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	emaData                     emaInstrument
	tgClient                    telegram.Client
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, emaData emaInstrument, tgClient telegram.Client) SuperTrend {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		emaData:                     emaData,
		tgClient:                    tgClient,
	}
}
