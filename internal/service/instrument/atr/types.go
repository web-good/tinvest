package atr

import (
	"context"
	"time"
	"tinvest/internal/domain/atr"
	"tinvest/internal/enum"
	"tinvest/internal/model"

	"github.com/golang/protobuf/ptypes/timestamp"
)

type Instrument interface {
	TechAnalyse(context context.Context, instrumentUID *string, interval enum.Interval, dateNow time.Time) (atr.ItemTechAnalyse, error)
	AverageVolume(ctx context.Context, instrumentUID string, interval enum.Interval, dateNow time.Time) (float64, error)
}

type MarketDataClient interface {
	GetCandles(context context.Context, instrumentUID *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*model.CandleItemTechAnalyse, error)
}

type service struct {
	period                  int64
	marketDataServiceClient MarketDataClient
}

func NewAtr(marketDataServiceClient MarketDataClient) *service {
	return &service{
		period:                  14,
		marketDataServiceClient: marketDataServiceClient,
	}
}
