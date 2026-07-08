package ema

import (
	"context"
	"time"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/model"

	"github.com/golang/protobuf/ptypes/timestamp"
)

type Instrument interface {
	TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, to time.Time, period int) ([]domainema.ItemTechAnalyse, error)
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
