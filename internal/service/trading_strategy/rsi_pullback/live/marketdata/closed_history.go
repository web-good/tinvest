package marketdata

import (
	"context"
	"fmt"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// ClosedHistory caches one instrument's M30 and daily bars for the closed-candle consumer.
// If the stream skips an interval, backfill from GetCandles before using its next close.
type ClosedHistory struct {
	client       candles.CandleClient
	instrumentID string
	lookback     int
	bars         []backtest.Candle
	daily        []backtest.Candle
	dailyDate    string
}

func NewClosedHistory(client candles.CandleClient, instrumentID string, lookback int) ClosedHistory {
	return ClosedHistory{client: client, instrumentID: instrumentID, lookback: lookback}
}

func (h *ClosedHistory) Warm(ctx context.Context, now time.Time) error {
	bars, err := fetchBars(ctx, h.client, h.instrumentID, enum.Minutes30, h.lookback, now)
	if err != nil {
		return err
	}
	h.bars = bars
	return h.loadDaily(ctx, now)
}

func (h *ClosedHistory) Assemble(ctx context.Context, bar backtest.Candle) (strategy.MarketData, error) {
	window := h.bars
	span := enum.Minutes30.ToTimeDuration()
	if n := len(window); n == 0 || !window[n-1].Time.Add(span).Equal(bar.Time) {
		// Query up to the opening of the event bar. If the endpoint includes that
		// timestamp, drop its copy in favor of the explicitly closed stream candle.
		var err error
		window, err = fetchBars(ctx, h.client, h.instrumentID, enum.Minutes30, h.lookback, bar.Time)
		if err != nil {
			return strategy.MarketData{}, err
		}
		for len(window) > 0 && !window[len(window)-1].Time.Before(bar.Time) {
			window = window[:len(window)-1]
		}
	}
	if len(window) < h.lookback-1 {
		return strategy.MarketData{}, fmt.Errorf("only %d previous M30 candles for lookback %d", len(window), h.lookback)
	}
	if err := h.loadDaily(ctx, bar.Time); err != nil {
		return strategy.MarketData{}, err
	}
	h.bars = append(window[len(window)-h.lookback+1:], bar)
	md := backtest.AssembleMarketData(h.bars, h.daily, nil, bar.Time)
	md.TodayHigh, md.TodayLow = backtest.TodayExtent(h.bars, len(h.bars)-1)
	return md, nil
}

func (h *ClosedHistory) loadDaily(ctx context.Context, at time.Time) error {
	date := at.In(mskZone).Format(time.DateOnly)
	if h.dailyDate == date {
		return nil
	}
	daily, err := fetchDaily(ctx, h.client, h.instrumentID, at)
	if err != nil {
		return err
	}
	h.daily, h.dailyDate = daily, date
	return nil
}
