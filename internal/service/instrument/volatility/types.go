package volatility

import (
	"context"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Instrument interface {
	CalculateVolatility(context context.Context, instrumentUID string, interval enum.Interval, dateFrom *timestamppb.Timestamp, dateTo *timestamppb.Timestamp, length int32) (*domain.VolatilityItemTechAnalyse, error)
}

type marketDataClient interface {
	GetCandles(context context.Context, instrumentUID *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*model.CandleItemTechAnalyse, error)
}

type service struct {
	marketDataServiceClient marketDataClient
}

func New(marketDataServiceClient marketDataClient) *service {
	return &service{
		marketDataServiceClient: marketDataServiceClient,
	}
}
