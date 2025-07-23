package macd_rsi

import (
	"context"
	"tinvest/internal/domain/atr"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type MacdRsi interface {
	Trade(ctx context.Context, in dto.Trade) error
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
