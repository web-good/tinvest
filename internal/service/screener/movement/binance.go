package movement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var errRateLimited = errors.New("market data rate limited")

type binance struct {
	client *http.Client
	base   string
}

func NewBinance(minPct float64, send func(string) error) *Worker {
	return &Worker{"Binance Spot · USDT", binance{&http.Client{Timeout: 30 * time.Second}, "https://data-api.binance.vision/api/v3/"}, minPct, 120 * time.Millisecond, send}
}

func (b binance) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 429 || resp.StatusCode == 418 {
		return fmt.Errorf("%w: %s", errRateLimited, resp.Status)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("binance: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (b binance) instruments(ctx context.Context) ([]instrument, error) {
	var info struct {
		Symbols []struct {
			Symbol, Status, QuoteAsset string
			IsSpotTradingAllowed       bool
		}
	}
	if err := b.get(ctx, "exchangeInfo?permissions=SPOT&showPermissionSets=false", &info); err != nil {
		return nil, err
	}
	var out []instrument
	for _, s := range info.Symbols {
		if s.Status == "TRADING" && s.QuoteAsset == "USDT" && s.IsSpotTradingAllowed {
			out = append(out, instrument{s.Symbol, s.Symbol})
		}
	}
	return out, nil
}

func (b binance) candles(ctx context.Context, id string, start, end time.Time) ([]bar, error) {
	path := fmt.Sprintf("klines?symbol=%s&interval=5m&startTime=%d&endTime=%d&limit=12", url.QueryEscape(id), start.UnixMilli(), end.UnixMilli()-1)
	var rows [][]json.RawMessage
	if err := b.get(ctx, path, &rows); err != nil {
		return nil, err
	}
	var out []bar
	for _, row := range rows {
		if len(row) < 4 {
			return nil, errors.New("short Binance kline")
		}
		var ms int64
		if err := json.Unmarshal(row[0], &ms); err != nil {
			return nil, err
		}
		candle := bar{at: time.UnixMilli(ms)}
		for i, p := range []*float64{&candle.open, &candle.high, &candle.low} {
			var num json.Number
			if err := json.Unmarshal(row[i+1], &num); err != nil {
				return nil, err
			}
			v, err := num.Float64()
			if err != nil {
				return nil, err
			}
			*p = v
		}
		out = append(out, candle)
	}
	return out, nil
}
