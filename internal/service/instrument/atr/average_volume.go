package atr

import (
	"context"
	"fmt"
	"math"
	"time"
	"tinvest/internal/enum"
	"tinvest/internal/utils"
)

func (s *service) AverageVolume(ctx context.Context, instrumentUid string, interval enum.Interval, dateNow time.Time) (float64, error) {
	p := int32(20)
	candles, err := s.marketDataServiceClient.GetCandles(ctx, &instrumentUid, interval.ToNumberInvestApi(), utils.TimeStampPbGenerator(dateNow, -25, interval), utils.TimeStampPbGenerator(dateNow, -1, interval), &p, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get candles from MarketDataService: %w", err)
	}

	if len(candles) < 14 {
		return 0, fmt.Errorf("candles is too short")
	}

	j := 0
	volume := float64(0)

	for i := len(candles) - 1; j < 14; i-- {
		volume = volume + float64(candles[i].Volume)
		j++
	}

	return math.Round(((volume / 14) * 100) / 100), nil
}
