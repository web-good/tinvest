package golden_x

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type GoldenX interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type rsiInstrument interface {
	CalculateRSI(context context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, DateTo *timestamppb.Timestamp, period int32) ([]*domain.RSIItemTechAnalyse, error)
}

type service struct {
	rsi                         rsiInstrument
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
}

func NewService(rsi rsiInstrument, instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, tgClient telegram.Client) *service {
	return &service{
		rsi:                         rsi,
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
	}
}
