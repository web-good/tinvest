package bonds

import (
	"context"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Bonds interface {
	Trade(ctx context.Context, tg telegram.Client) error
}

type rsiInstrument interface {
	CalculateRSI(context context.Context, instrumentUID string, interval enum.Interval, dateFrom *timestamppb.Timestamp, DateTo *timestamppb.Timestamp, period int32) ([]*domain.RSIItemTechAnalyse, error)
}

type service struct {
	rsi                         rsiInstrument
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
}

func NewService(rsi rsiInstrument, instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient) *service {
	return &service{
		rsi:                         rsi,
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
	}
}
