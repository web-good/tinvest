package scalping_rsi

import (
	"context"
	"time"
	"tinvest/internal/domain"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/scalping_rsi/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type ScalpingRsi interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type emaInstrument interface {
	TechAnalyse(context context.Context, instrumentUID *string, interval int32, from time.Time, to time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type rsiInstrument interface {
	CalculateRSI(context context.Context, instrumentUID string, interval enum.Interval, dateFrom *timestamppb.Timestamp, DateTo *timestamppb.Timestamp, period int32) ([]*domain.RSIItemTechAnalyse, error)
}

type service struct {
	ema                         emaInstrument
	rsi                         rsiInstrument
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
}

func NewService(ema emaInstrument, rsi rsiInstrument, instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, tgClient telegram.Client) *service {
	return &service{
		ema:                         ema,
		rsi:                         rsi,
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
	}
}
