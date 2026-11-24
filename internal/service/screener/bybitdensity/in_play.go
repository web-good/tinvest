package bybitdensity

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/bybitdensity/strategy"
)

type m5Reader interface {
	SpotM5(context.Context, model.Key, time.Time, time.Time) ([]model.M5Candle, error)
}

const inPlayCandleCount = int(strategy.InPlayLookback / strategy.InPlayInterval)

func runInPlay(ctx context.Context, market m5Reader, hub *alertHub, instruments []model.Instrument) {
	filters := make(map[model.Key]strategy.InPlayFilter)
	for _, instrument := range instruments {
		if filter := hub.profiles[instrument.Key.Symbol].inPlay; filter.Enabled() {
			filters[instrument.Key] = filter
		}
	}
	if len(filters) == 0 {
		return
	}
	loaded := make(map[model.Key]time.Time, len(filters))
	for ctx.Err() == nil {
		closeTime := time.Now().Truncate(strategy.InPlayInterval)
		next := closeTime.Add(strategy.InPlayInterval)
		for key, filter := range filters {
			if loaded[key].Equal(closeTime) {
				continue
			}
			candles, err := market.SpotM5(ctx, key, closeTime.Add(-strategy.InPlayLookback), closeTime)
			if err == nil && len(candles) != inPlayCandleCount {
				err = fmt.Errorf("expected %d candles, got %d", inPlayCandleCount, len(candles))
			}
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.WarnContext(ctx, "Bybit in-play history failed", "symbol", key.Symbol, "error", err)
				if retryAt := time.Now().Add(15 * time.Second); retryAt.Before(next) {
					next = retryAt
				}
				continue
			}
			window := filter.Window(candles)
			hub.mu.Lock()
			hub.states[key].strategy.SetInPlayWindow(window, time.Now())
			hub.mu.Unlock()
			loaded[key] = closeTime
		}
		if !waitRetry(ctx, time.Until(next)) {
			return
		}
	}
}

func loadInPlayWindows(ctx context.Context, market m5Reader, key model.Key, filter strategy.InPlayFilter, from, end time.Time) ([]strategy.InPlayWindow, error) {
	if !filter.Enabled() {
		return nil, nil
	}
	candles, err := market.SpotM5(ctx, key, from.Add(-strategy.InPlayLookback), end.Add(-strategy.InPlayInterval))
	if err != nil {
		return nil, fmt.Errorf("in-play history for %s: %w", key.Symbol, err)
	}
	count := int(end.Sub(from) / strategy.InPlayInterval)
	windows := make([]strategy.InPlayWindow, count)
	closed := 0
	for i := range windows {
		at := from.Add(time.Duration(i) * strategy.InPlayInterval)
		for closed < len(candles) && !candles[closed].End.After(at) {
			closed++
		}
		windows[i].From = at
		if closed >= inPlayCandleCount && candles[closed-1].End.Equal(at) {
			windows[i] = filter.Window(candles[:closed])
		}
	}
	return windows, nil
}
