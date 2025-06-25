package ema

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"time"
	domainema "tinvest/internal/domain/ema"
)

func (e *service) TechAnalyse(context context.Context, instrumentUid *string, interval int32, from time.Time, period int32) ([]domainema.ItemTechAnalyse, error) {
	limit := period * 4
	candles, err := e.marketDataServiceClient.GetCandles(context, instrumentUid, interval, timestamppb.New(from), timestamppb.New(time.Now()), &limit)

	if err != nil {
		return nil, fmt.Errorf("GetCandles error: %w", err)
	}

	ema := make([]domainema.ItemTechAnalyse, len(candles))

	if len(candles) == 0 {
		return ema, nil
	}

	var sum float64
	multiplier := 2.0 / float64((period)+1)
	//fracPart, intPart := SplitPrice(float64(candles[0].Close.Units))

	for i := 0; i < int(period) && i < len(candles); i++ {
		sum += float64(candles[i].Close.Units)
	}

	sma := sum / float64(min(period, int32(len(candles))))
	fracPart, intPart := SplitPrice(sma)
	ema[0] = domainema.ItemTechAnalyse{
		Date: candles[0].Time,
		SignalLine: domainema.Quotation{
			Units: fracPart,
			Nano:  intPart,
		},
	}

	for i := 1; i < len(candles); i++ {
		fracPart, intPart := SplitPrice(((float64(candles[i].Close.Units) - float64(ema[i-1].SignalLine.Units)) * multiplier) + float64(ema[i-1].SignalLine.Units))
		ema[i] = domainema.ItemTechAnalyse{
			Date: candles[i].Time,
			SignalLine: domainema.Quotation{
				Units: fracPart,
				Nano:  intPart,
			},
		}
	}

	return ema, nil
}

func SplitPrice(price float64) (int64, int32) {
	frac, intPart := math.Modf(price)

	return int64(frac), int32(math.Round(intPart * 1e9))
}
