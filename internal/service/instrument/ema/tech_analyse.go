package ema

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/utils"
)

func (e *service) TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, to time.Time, period int) ([]domainema.ItemTechAnalyse, error) {
	limit := int32(period * 2)
	candles, err := e.marketDataServiceClient.GetCandles(context, instrumentUid, interval, timestamppb.New(from), timestamppb.New(to), &limit, true)

	if err != nil {
		return nil, fmt.Errorf("GetCandles error: %w", err)
	}

	if len(candles) < period {
		return nil, fmt.Errorf("not enough candles")
	}

	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = float64(c.Close.Units)
	}

	emas := domainema.Compute(closes, period)
	emaR := make([]domainema.ItemTechAnalyse, len(candles))

	for k := period - 1; k < len(candles); k++ {
		emaR[k].SignalLine.Units, emaR[k].SignalLine.Nano = utils.SplitPrice(emas[k])
		if k >= period {
			emaR[k].Date = candles[k].Time
		}
	}

	return emaR, nil
}
