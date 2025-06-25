package ema

import (
	"context"
	"github.com/golang/protobuf/ptypes/timestamp"
	"time"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/model"
)

type Instrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, period int32) ([]domainema.ItemTechAnalyse, error)
}

type MarketDataClient interface {
	GetCandles(context context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32) ([]*model.CandleItemTechAnalyse, error)
}

type service struct {
	marketDataServiceClient MarketDataClient
}

func NewEma(marketDataServiceClient MarketDataClient) *service {
	return &service{
		marketDataServiceClient: marketDataServiceClient,
	}
}
