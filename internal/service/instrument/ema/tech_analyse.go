package ema

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"time"
	"tinvest/internal/domain/ema"
)

func (e *service) TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, period int) ([]ema.ItemTechAnalyse, error) {
	limit := int32(period * 2)
	candles, err := e.marketDataServiceClient.GetCandles(context, instrumentUid, interval, timestamppb.New(from), timestamppb.New(time.Now()), &limit, true)

	if err != nil {
		return nil, fmt.Errorf("GetCandles error: %w", err)
	}

	if len(candles) < period {
		return nil, fmt.Errorf("not enough candles")
	}

	emas := make([]float64, len(candles))
	emaR := make([]ema.ItemTechAnalyse, len(candles))

	var sum float64

	for i := 0; i < period; i++ {
		sum += float64(candles[i].Close.Units)
	}

	emas[period-1] = sum / float64(period)
	emaR[period-1].SignalLine.Units, emaR[period-1].SignalLine.Nano = splitPrice(emas[period-1])
	multiplier := 2.0 / float64(period+1)

	for k := period; k < len(candles); k++ {
		emas[k] = (float64(candles[k].Close.Units)-emas[k-1])*multiplier + emas[k-1]
		emaR[k].SignalLine.Units, emaR[k].SignalLine.Nano = splitPrice(emas[k])
		emaR[k].Date = candles[k].Time
	}

	return emaR, nil
}

func splitPrice(price float64) (int64, int32) {
	frac, intPart := math.Modf(price)

	return int64(frac), int32(math.Round(intPart * 1e9))
}
