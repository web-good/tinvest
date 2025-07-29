package macd_rsi

import (
	"context"
	"time"
	"tinvest/internal/domain/atr"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type MacdRsi interface {
	Trade(ctx context.Context, in dto.Trade) error
	TakeProfit(ctx context.Context, in dto.TakeProfit) error
}

type atrInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string) (atr.ItemTechAnalyse, error)
}

type emaInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	atrInstrument               atrInstrument
	tgClient                    telegram.Client
	ema                         emaInstrument
	usersServiceClient          grpc.UsersServiceClient
	operationsServiceClient     grpc.OperationsServiceClient
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, atrInstrument atrInstrument, tgClient telegram.Client, emaInstrument emaInstrument, userServiceClient grpc.UsersServiceClient, operationServiceClient grpc.OperationsServiceClient) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		atrInstrument:               atrInstrument,
		tgClient:                    tgClient,
		ema:                         emaInstrument,
		usersServiceClient:          userServiceClient,
		operationsServiceClient:     operationServiceClient,
	}
}
