package density

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"
)

type Sender interface {
	Send(context.Context, Signal) error
}
type Service struct {
	cfg             Config
	source          UniverseSource
	stream          StreamSource
	sender          Sender
	historyInterval time.Duration
	refreshInterval time.Duration
	refreshRetry    time.Duration
	sent            map[key]time.Time
}

func NewService(cfg Config, source UniverseSource, stream StreamSource, sender Sender, historyInterval time.Duration) *Service {
	return &Service{cfg: cfg, source: source, stream: stream, sender: sender, historyInterval: historyInterval, refreshInterval: 24 * time.Hour, refreshRetry: time.Minute, sent: make(map[key]time.Time)}
}
func (s *Service) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "density screener started")
	var universe []Instrument
	var refreshAt time.Time
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !time.Now().Before(refreshAt) {
			updated, err := RefreshUniverse(ctx, s.source, s.cfg, s.historyInterval)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err == nil {
				universe = updated
				slog.InfoContext(ctx, "density universe refreshed", "instruments", len(universe))
				refreshAt = time.Now().Add(s.refreshInterval)
			} else {
				slog.WarnContext(ctx, "density universe refresh failed", "error", err)
				refreshAt = time.Now().Add(s.refreshRetry)
			}
		}
		if len(universe) == 0 {
			if err := waitFor(ctx, time.Until(refreshAt)); err != nil {
				return err
			}
			continue
		}
		connectionCtx, cancel := context.WithDeadline(ctx, refreshAt)
		err := runConnection(connectionCtx, s, universe)
		expired := connectionCtx.Err() != nil
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if expired || err == nil {
			backoff = time.Second
			continue
		}
		slog.WarnContext(ctx, "density stream disconnected", "error", err)
		delay := backoff/2 + time.Duration(rand.Int64N(int64(backoff/2)+1))
		if time.Until(refreshAt) < delay {
			delay = time.Until(refreshAt)
		}
		if err := waitFor(ctx, delay); err != nil {
			return err
		}
		backoff *= 2
		if backoff > time.Minute {
			backoff = time.Minute
		}
	}
}
func waitFor(ctx context.Context, delay time.Duration) error {
	if delay < 0 {
		delay = 0
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
