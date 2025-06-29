package ema

import (
	"context"
	"github.com/golang/protobuf/ptypes/timestamp"
	"time"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/model"
)

type Instrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type marketDataClient interface {
	GetCandles(context context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*model.CandleItemTechAnalyse, error)
}

type service struct {
	marketDataServiceClient marketDataClient
}

func NewEma(marketDataServiceClient marketDataClient) *service {
	return &service{
		marketDataServiceClient: marketDataServiceClient,
	}
}
