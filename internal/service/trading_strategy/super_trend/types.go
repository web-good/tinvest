package super_trend

import (
	"context"
	"time"
	"tinvest/internal/domain/atr"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/enum"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

var _ SuperTrend = (*service)(nil)

type SuperTrend interface {
	Trade(ctx context.Context) error
	TakeProfit(ctx context.Context) error
}

type emaInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, to time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type atrInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval enum.Interval) (atr.ItemTechAnalyse, error)
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	ema                         emaInstrument
	atr                         atrInstrument
	tgClient                    telegram.Client
	usersServiceClient          grpc.UsersServiceClient
	operationsServiceClient     grpc.OperationsServiceClient
}

func NewService(
	instrumentsServiceClient grpc.InstrumentsServiceClient,
	marketDataServiceGrpcClient grpc.MarketDataServiceClient,
	emaInstrument emaInstrument,
	atrInstrument atrInstrument,
	tgClient telegram.Client,
	usersServiceClient grpc.UsersServiceClient,
	operationsServiceClient grpc.OperationsServiceClient,
) SuperTrend {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		ema:                         emaInstrument,
		atr:                         atrInstrument,
		tgClient:                    tgClient,
		usersServiceClient:          usersServiceClient,
		operationsServiceClient:     operationsServiceClient,
	}
}
