package inplay

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

func NewBinance(minPct, minRVOL float64, send func(string) error) *Worker {
	return &Worker{market: "Binance Spot · USDT", feed: binance{&http.Client{Timeout: 30 * time.Second}, "https://data-api.binance.vision/api/v3/"}, minPct: minPct, minRVOL: minRVOL, delay: 120 * time.Millisecond, send: send}
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
	path := fmt.Sprintf("klines?symbol=%s&interval=5m&startTime=%d&endTime=%d&limit=%d", url.QueryEscape(id), start.UnixMilli(), end.UnixMilli()-1, int(end.Sub(start)/step))
	var rows [][]json.RawMessage
	if err := b.get(ctx, path, &rows); err != nil {
		return nil, err
	}
	var out []bar
	for _, row := range rows {
		if len(row) < 6 {
			return nil, errors.New("short Binance kline")
		}
		var ms int64
		if err := json.Unmarshal(row[0], &ms); err != nil {
			return nil, err
		}
		candle := bar{at: time.UnixMilli(ms)}
		for _, field := range []struct {
			index int
			dst   *float64
		}{
			{1, &candle.open},
			{2, &candle.high},
			{3, &candle.low},
			{5, &candle.volume},
		} {
			var num json.Number
			if err := json.Unmarshal(row[field.index], &num); err != nil {
				return nil, err
			}
			v, err := num.Float64()
			if err != nil {
				return nil, err
			}
			*field.dst = v
		}
		out = append(out, candle)
	}
	return out, nil
}
