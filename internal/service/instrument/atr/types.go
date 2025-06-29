package atr

import (
	"context"
	"github.com/golang/protobuf/ptypes/timestamp"
	"tinvest/internal/domain/atr"
	"tinvest/internal/model"
)

type Instrument interface {
	TechAnalyse(context context.Context, instrumentUid *string) (atr.ItemTechAnalyse, error)
}

type MarketDataClient interface {
	GetCandles(context context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*model.CandleItemTechAnalyse, error)
}

type service struct {
	period                  int64
	marketDataServiceClient MarketDataClient
}

func NewAtr(marketDataServiceClient MarketDataClient) *service {
	return &service{
		period:                  15,
		marketDataServiceClient: marketDataServiceClient,
	}
}
