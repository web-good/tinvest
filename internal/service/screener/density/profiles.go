package density

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type SupportSource interface {
	SupportCandles(context.Context, string, int) ([]SupportCandle, error)
}

func waitHistory(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// RefreshProfiles never returns partially updated histories.
func RefreshProfiles(ctx context.Context, source UniverseSource, instruments []Instrument, cfg Config, interval time.Duration, now time.Time) ([]Instrument, error) {
	result := append([]Instrument(nil), instruments...)
	history, ok := source.(SupportSource)
	if cfg.Support.Enabled && !ok {
		return nil, fmt.Errorf("density support history source is required")
	}
	for n := range result {
		result[n].ProfileAt = now
		result[n].Supports = nil
		if !cfg.Support.Enabled {
			continue
		}
		if err := waitHistory(ctx, interval); err != nil {
			return nil, err
		}
		candles, err := history.SupportCandles(ctx, result[n].ID, cfg.Support.LookbackDays)
		if err != nil {
			return nil, fmt.Errorf("support history %s: %w", result[n].Ticker, err)
		}
		result[n].Supports = FindSupports(candles, cfg.Support, now)
	}
	return result, nil
}

// StreamProfiles refreshes entry histories without blocking market data or exits.
// Failures leave existing profiles to expire; retries do not extend their lifetime.
func StreamProfiles(ctx context.Context, source UniverseSource, cfg Config, interval time.Duration, initial []Instrument, warn func(error)) <-chan []Instrument {
	if source == nil || (cfg.TurnoverFraction == 0 && !cfg.Support.Enabled) {
		return nil
	}
	updates := make(chan []Instrument, 1)
	delay := ProfileLifetime
	now := time.Now()
	for _, i := range initial {
		if !i.ProfileAt.IsZero() {
			delay = min(delay, i.ProfileAt.Add(ProfileLifetime).Sub(now))
		}
	}
	go func() {
		defer close(updates)
		timer := time.NewTimer(max(delay, time.Millisecond))
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
			profiles, err := RefreshUniverse(ctx, source, cfg, interval)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				if warn != nil {
					warn(err)
				} else {
					slog.WarnContext(ctx, "density history refresh failed", "error", err)
				}
				timer.Reset(time.Minute)
				continue
			}
			select {
			case updates <- profiles:
			case <-ctx.Done():
				return
			}
			timer.Reset(ProfileLifetime)
		}
	}()
	return updates
}
