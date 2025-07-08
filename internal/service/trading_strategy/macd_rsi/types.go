package macd_rsi

import (
	"context"
	"tinvest/internal/domain/atr"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type MacdRsi interface {
	Trade(ctx context.Context) error
}

type atrInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string) (atr.ItemTechAnalyse, error)
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	atrInstrument               atrInstrument
	tgClient                    telegram.Client
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, atrInstrument atrInstrument, tgClient telegram.Client) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		atrInstrument:               atrInstrument,
		tgClient:                    tgClient,
	}
}
