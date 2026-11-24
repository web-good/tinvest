package bybitdensity

import (
	"context"
	"log/slog"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/volume"
)

func previousSpotVolumeWindow(now time.Time) (from, to, until time.Time) {
	session, _, until := volume.PreviousWeekday(now)
	day := session.In(moscowTime)
	from = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, moscowTime)
	return from, from.AddDate(0, 0, 1), until
}

func refreshVolumes(ctx context.Context, market marketData, hub *alertHub, instruments []model.Instrument, loaded map[model.Key]time.Time) time.Duration {
	now := time.Now()
	from, to, until := previousSpotVolumeWindow(now)
	delay := time.Until(until)
	for _, instrument := range instruments {
		if ctx.Err() != nil {
			break
		}
		key := instrument.Key
		if loaded[key].Equal(until) {
			continue
		}
		quantity, err := market.AverageVolume(ctx, key, from, to)
		if err != nil || quantity <= 0 {
			slog.WarnContext(ctx, "Bybit volume history failed", "symbol", key.Symbol, "error", err)
			delay = min(delay, time.Minute)
			continue
		}
		hub.mu.Lock()
		hub.states[key].strategy.SetAverageVolume(volume.Average{Quantity: quantity, Until: until})
		hub.mu.Unlock()
		loaded[key] = until
	}
	return min(delay, time.Until(until))
}
