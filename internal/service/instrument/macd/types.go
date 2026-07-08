package macd

import (
	"context"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Instrument interface {
	CalculateMACD(context context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, DateTo *timestamppb.Timestamp, fast int32, slow int32, signal int32) ([]*domain.MACDItemTechAnalyse, error)
}

type marketDataClient interface {
	GetCandles(context context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*model.CandleItemTechAnalyse, error)
}

type service struct {
	marketDataServiceClient marketDataClient
}

func New(marketDataServiceClient marketDataClient) *service {
	return &service{
		marketDataServiceClient: marketDataServiceClient,
	}
}
