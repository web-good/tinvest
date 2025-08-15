package macd_rsi

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/domain/atr"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type MacdRsi interface {
	Trade(ctx context.Context, in dto.Trade) error
	TakeProfit(ctx context.Context, in dto.TakeProfit) error
	BackTest(ctx context.Context, in dto.BackTest) error
}

type atrInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval enum.Interval) (atr.ItemTechAnalyse, error)
	AverageVolume(ctx context.Context, instrumentUid string, interval enum.Interval) (float64, error)
}

type emaInstrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, to time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type macdInstrument interface {
	CalculateMACD(context context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, DateTo *timestamppb.Timestamp, fast int32, slow int32, signal int32) ([]*domain.MACDItemTechAnalyse, error)
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	atrInstrument               atrInstrument
	tgClient                    telegram.Client
	ema                         emaInstrument
	macd                        macdInstrument
	usersServiceClient          grpc.UsersServiceClient
	operationsServiceClient     grpc.OperationsServiceClient
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, atrInstrument atrInstrument, tgClient telegram.Client, emaInstrument emaInstrument, macd macdInstrument, userServiceClient grpc.UsersServiceClient, operationServiceClient grpc.OperationsServiceClient) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		atrInstrument:               atrInstrument,
		tgClient:                    tgClient,
		ema:                         emaInstrument,
		usersServiceClient:          userServiceClient,
		operationsServiceClient:     operationServiceClient,
		macd:                        macd,
	}
}
