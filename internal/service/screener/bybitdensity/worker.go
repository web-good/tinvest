// Package bybitdensity scans public Bybit books and runs a virtual trading session.
package bybitdensity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"tinvest/internal/service/screener/bybitdensity/feed"
	"tinvest/internal/service/screener/bybitdensity/journal"
	"tinvest/internal/service/screener/bybitdensity/model"

	"golang.org/x/sync/errgroup"
)

type marketData interface {
	m5Reader
	AverageVolume(context.Context, model.Key, time.Time, time.Time) (float64, error)
	Instruments(context.Context, model.Market, []string, int) ([]model.Instrument, error)
	Stream(context.Context, []model.Instrument, int, func(model.Event) error) error
}

type Worker struct {
	options options
}

// NewWorkerFromEnv loads the Bybit worker only when enabled.
func NewWorkerFromEnv() (*Worker, error) {
	value := strings.TrimSpace(os.Getenv(envPrefix + "ENABLED"))
	if value == "" {
		return nil, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("%sENABLED: %w", envPrefix, err)
	}
	if !enabled {
		return nil, nil
	}
	op, err := readOptions(nil, io.Discard, false)
	if err != nil {
		return nil, err
	}
	return &Worker{options: op}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	op := w.options
	market := feed.NewClient(op.restURL, op.wsURL)
	send, err := telegramSender(op.telegramToken)
	if err != nil {
		return err
	}
	screener, simulation := destinations(send, op.screener, op.simulation)
	return serve(ctx, market, op, screener, simulation)
}

func serve(ctx context.Context, market marketData, op options, screener, simulation sendMessage) (retErr error) {
	session, err := journal.Open(op.stateDir)
	if err != nil {
		return fmt.Errorf("open simulation state: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, session.Close()) }()
	slog.Info("Bybit Density worker started", "state_dir", op.stateDir)
	op.strategy.RequireAverageVolume = true
	op.strategy.FixedLimitEntry = true

	for {
		next := session.NextReset(time.Now())
		if next.IsZero() {
			return serveSession(ctx, market, op, session, screener, simulation)
		}
		runCtx, cancel := context.WithDeadline(ctx, next)
		err := serveSession(runCtx, market, op, session, screener, simulation)
		expired := runCtx.Err() == context.DeadlineExceeded
		cancel()
		if err != nil || !expired || ctx.Err() != nil {
			return err
		}
		if err := session.Reset(); err != nil {
			return fmt.Errorf("reset bybit simulation state: %w", err)
		}
		slog.Info("Bybit Density state cleared", "state_dir", op.stateDir)
	}
}

func serveSession(ctx context.Context, market marketData, op options, session *journal.Journal, screener, simulation sendMessage) error {
	hub := newAlertHub(op.strategy, session, op.spotProfiles)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return session.Deliver(ctx, simulation) })
	group.Go(func() error { return hub.sendSignals(ctx, screener) })
	group.Go(func() error { return hub.expire(ctx) })
	group.Go(func() error { return runSpot(ctx, market, op, hub) })
	return group.Wait()
}

func runSpot(ctx context.Context, market marketData, op options, hub *alertHub) error {
	const category = model.Spot
	symbols := slices.Sorted(maps.Keys(op.spotProfiles))
	if len(symbols) == 0 {
		return errors.New("no bybit spot profiles configured")
	}
	ctx, cancel := context.WithCancel(ctx)
	var refreshers sync.WaitGroup
	defer func() {
		cancel()
		refreshers.Wait()
	}()
	var instruments []model.Instrument
	var retry time.Duration
	for ctx.Err() == nil {
		if len(instruments) == 0 {
			var err error
			instruments, err = market.Instruments(ctx, category, symbols, len(symbols))
			if errors.Is(err, feed.ErrInvalidSymbols) {
				return fmt.Errorf("%s instrument configuration: %w", category, err)
			}
			if ctx.Err() != nil {
				return nil
			}
			if err != nil || len(instruments) == 0 {
				slog.Warn("Bybit instrument selection failed", "market", category, "error", err)
				retry = nextRetry(retry)
				if !waitRetry(ctx, retry) {
					return nil
				}
				continue
			}
			hub.add(instruments)
			refreshers.Go(func() { runInPlay(ctx, market, hub, instruments) })
			loaded := make(map[model.Key]time.Time, len(instruments))
			delay := refreshVolumes(ctx, market, hub, instruments, loaded)
			refreshers.Go(func() {
				for waitRetry(ctx, delay) {
					delay = refreshVolumes(ctx, market, hub, instruments, loaded)
				}
			})
			retry = 0
			slog.Info("Bybit instruments selected", "market", category, "count", len(instruments))
		}
		started := time.Now()
		var recordErr error
		var bookConnected bool
		err := market.Stream(ctx, instruments, op.depth, func(event model.Event) error {
			now := time.Now()
			if book := event.Book; !bookConnected && book != nil && book.Snapshot &&
				now.Sub(book.At).Abs() <= op.strategy.Freshness {
				bookConnected = true
				slog.Info("Bybit order book connected", "market", category, "symbol", event.Key.Symbol)
			}
			recordErr = hub.apply(event, now)
			return recordErr
		})
		hub.reset(instruments)
		if recordErr != nil {
			return fmt.Errorf("save simulated execution: %w", recordErr)
		}
		if ctx.Err() != nil {
			return nil
		}
		if time.Since(started) >= 30*time.Second {
			retry = 0
		}
		retry = nextRetry(retry)
		slog.Warn("Bybit stream disconnected", "market", category, "error", err, "retry", retry)
		if !waitRetry(ctx, retry) {
			return nil
		}
	}
	return nil
}

func nextRetry(previous time.Duration) time.Duration {
	if previous == 0 {
		return time.Second
	}
	return min(previous*2, 30*time.Second)
}

func waitRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
