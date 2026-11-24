package feed

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/volume"
)

// AverageVolume reads spot five-minute candle volumes in the base asset.
func (c *Client) AverageVolume(ctx context.Context, key model.Key, from, to time.Time) (float64, error) {
	if key.Market != model.Spot {
		return 0, fmt.Errorf("volume history requires a spot instrument")
	}
	candleCount := int(to.Sub(from) / volume.CandleInterval)
	if candleCount < 1 || candleCount > 1000 {
		return 0, fmt.Errorf("invalid volume history range for %s", key.Symbol)
	}
	var result klineResponse
	query := url.Values{
		"category": {string(key.Market)}, "symbol": {key.Symbol}, "interval": {"5"},
		"limit": {strconv.Itoa(candleCount)},
		"start": {strconv.FormatInt(from.UnixMilli(), 10)}, "end": {strconv.FormatInt(to.UnixMilli()-1, 10)},
	}
	if err := c.get(ctx, "/v5/market/kline", query, &result); err != nil {
		return 0, err
	}
	if result.Category != key.Market || result.Symbol != key.Symbol {
		return 0, fmt.Errorf("mismatched kline response for %s", key.Symbol)
	}
	samples := volume.Samples{From: from, To: to}
	const candlesPerDay = int(24 * time.Hour / volume.CandleInterval)
	if candleCount == candlesPerDay && len(result.List) != candlesPerDay {
		return 0, fmt.Errorf("incomplete daily volume history for %s", key.Symbol)
	}
	for i, candle := range result.List {
		if len(candle) < 7 {
			return 0, fmt.Errorf("invalid kline for %s", key.Symbol)
		}
		at, timeErr := strconv.ParseInt(candle[0], 10, 64)
		qty, qtyErr := nonnegative(candle[5])
		if timeErr != nil || qtyErr != nil {
			return 0, fmt.Errorf("invalid kline volume for %s", key.Symbol)
		}
		if candleCount == candlesPerDay && at != to.Add(-time.Duration(i+1)*volume.CandleInterval).UnixMilli() {
			return 0, fmt.Errorf("missing or unordered daily volume candle for %s", key.Symbol)
		}
		samples.Add(time.UnixMilli(at), qty)
	}
	return samples.Mean(), nil
}
