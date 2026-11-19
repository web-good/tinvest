package bybitdensity

import (
	"context"
	"log/slog"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/volume"
)

func refreshVolumes(ctx context.Context, market marketData, hub *alertHub, instruments []model.Instrument, loaded map[model.Key]time.Time) time.Duration {
	from, to, until := volume.PreviousWeekday(time.Now())
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
		if err != nil {
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
