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
	var result struct {
		marketResult[[]string]
		Symbol string `json:"symbol"`
	}
	query := url.Values{
		"category": {string(key.Market)}, "symbol": {key.Symbol}, "interval": {"5"},
		"start": {strconv.FormatInt(from.UnixMilli(), 10)}, "end": {strconv.FormatInt(to.UnixMilli()-1, 10)},
	}
	if err := c.get(ctx, "/v5/market/kline", query, &result); err != nil {
		return 0, err
	}
	if result.Category != key.Market || result.Symbol != key.Symbol {
		return 0, fmt.Errorf("mismatched kline response for %s", key.Symbol)
	}
	samples := volume.Samples{From: from, To: to}
	for _, candle := range result.List {
		if len(candle) < 7 {
			return 0, fmt.Errorf("invalid kline for %s", key.Symbol)
		}
		at, timeErr := strconv.ParseInt(candle[0], 10, 64)
		qty, qtyErr := strconv.ParseFloat(candle[5], 64)
		if timeErr != nil || qtyErr != nil {
			return 0, fmt.Errorf("invalid kline volume for %s", key.Symbol)
		}
		samples.Add(time.UnixMilli(at), qty)
	}
	return samples.Mean(), nil
}
