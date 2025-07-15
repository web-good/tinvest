package ema200

import (
	"context"
	"time"
	"tinvest/internal/domain/atr"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/ema200/dto/input"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type Ema200 interface {
	Trade(ctx context.Context, dto input.Trade) error
}

type emaInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type atrInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string) (atr.ItemTechAnalyse, error)
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	ema                         emaInstrument
	atr                         atrInstrument
	tgClient                    telegram.Client
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, emaInstrument emaInstrument, atrInstrument atrInstrument, tgClient telegram.Client) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		ema:                         emaInstrument,
		atr:                         atrInstrument,
		tgClient:                    tgClient,
	}
}
