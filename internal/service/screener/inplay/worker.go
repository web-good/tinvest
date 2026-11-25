package inplay

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"
)

const step = 5 * time.Minute

type bar struct {
	at                      time.Time
	open, high, low, volume float64
}

type instrument struct{ symbol, id string }

type source interface {
	instruments(context.Context) ([]instrument, error)
	candles(context.Context, string, time.Time, time.Time) ([]bar, error)
}

type result struct {
	symbol string
	mean   float64
	hits   int
	volume float64
	rvol   float64
}

func summarize(bars []bar, end time.Time, n int, minPct float64) (result, bool) {
	start := end.Add(-time.Duration(n) * step)
	var r result
	seen := 0
	for _, b := range bars {
		if b.at.Before(start) || !b.at.Before(end) {
			continue
		}
		if !b.at.Equal(start.Add(time.Duration(seen)*step)) || !(b.low > 0 && b.high >= b.open && b.open >= b.low) {
			return result{}, false
		}
		if b.volume < 0 || math.IsNaN(b.volume) || math.IsInf(b.volume, 0) {
			return result{}, false
		}
		r.volume += b.volume
		pct := (b.high - b.low) / b.open * 100
		r.mean += pct
		if pct+1e-12 >= minPct {
			r.hits++
		}
		seen++
	}
	if seen != n {
		return result{}, false
	}
	r.mean /= float64(n)
	return r, !math.IsInf(r.mean, 0) && !math.IsInf(r.volume, 0)
}

type Worker struct {
	market  string
	feed    source
	minPct  float64
	minRVOL float64
	history volumeHistory
	delay   time.Duration
	send    func(string) error
}

func (w *Worker) Run(ctx context.Context) error {
	if w.minPct < 0 || math.IsNaN(w.minPct) || math.IsInf(w.minPct, 0) {
		return errors.New("invalid inplay threshold")
	}
	if w.minRVOL <= 0 || math.IsNaN(w.minRVOL) || math.IsInf(w.minRVOL, 0) {
		return errors.New("invalid relative volume threshold")
	}
	for {
		end := time.Now().UTC().Truncate(30 * time.Minute).Add(30 * time.Minute)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(end.Add(5 * time.Second))):
		}
		if err := w.scan(ctx, end); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "inplay scan failed", "market", w.market, "error", err)
		}
	}
}

func (w *Worker) scan(ctx context.Context, end time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Minute)
	defer cancel()
	instruments, err := w.feed.instruments(ctx)
	if err != nil {
		return err
	}
	windows := []int{6}
	if end.Minute() == 0 {
		windows = append(windows, 12)
	}
	start := end.Add(-step * time.Duration(windows[len(windows)-1]))
	ranks := make(map[int][]result)
	valid := make(map[int]int)
	failed := 0
	var firstErr error
	pace := time.NewTicker(w.delay)
	defer pace.Stop()
	if w.history == nil {
		w.history = make(volumeHistory)
	}
	cutoff := start.UTC().Truncate(24*time.Hour).AddDate(0, 0, -historyLookbackDays)
	for key := range w.history {
		if key.day.Before(cutoff) {
			delete(w.history, key)
		}
	}
	fetch := func(id string, start, end time.Time) ([]bar, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pace.C:
			return w.feed.candles(ctx, id, start, end)
		}
	}
	recordFailure := func(symbol string, err error) {
		failed++
		if firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", symbol, err)
		}
	}
	for _, inst := range instruments {
		bars, err := fetch(inst.id, start, end)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, errRateLimited) {
				return err
			}
			recordFailure(inst.symbol, err)
			continue
		}
		for _, n := range windows {
			r, ok := summarize(bars, end, n, w.minPct)
			if !ok {
				continue
			}
			valid[n]++
			if r.mean+1e-12 < w.minPct {
				continue
			}
			baseline, err := w.history.baseline(fetch, inst.id, end, n)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, errRateLimited) {
					return err
				}
				recordFailure(inst.symbol, err)
				continue
			}
			if baseline <= 0 {
				continue
			}
			r.rvol = r.volume / baseline
			if !math.IsInf(r.rvol, 0) && r.rvol >= w.minRVOL {
				r.symbol = inst.symbol
				ranks[n] = append(ranks[n], r)
			}
		}
	}
	var errs []error
	if firstErr != nil {
		errs = append(errs, fmt.Errorf("%d instruments failed: %w", failed, firstErr))
	}
	for _, n := range windows {
		if valid[n] == 0 {
			continue
		}
		rows := ranks[n]
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].mean == rows[j].mean {
				return rows[i].symbol < rows[j].symbol
			}
			return rows[i].mean > rows[j].mean
		})
		var msg strings.Builder
		fmt.Fprintf(&msg, "inplay · %s · %d мин до %s МСК\nСредний размах 5м · свечей ≥ %g%% · RVOL ≥ %g×", w.market, n*5, end.In(time.FixedZone("MSK", 3*3600)).Format("15:04"), w.minPct, w.minRVOL)
		for _, r := range rows[:min(10, len(rows))] {
			fmt.Fprintf(&msg, "\n%s    %.2f%%    %d/%d    RVOL %.2f×", html.EscapeString(r.symbol), r.mean, r.hits, n, r.rvol)
		}
		if len(rows) == 0 {
			msg.WriteString("\nИнструментов выше или на уровне порога нет.")
		}
		if failed > 0 {
			fmt.Fprintf(&msg, "\nОшибки получения данных: %d", failed)
		}
		errs = append(errs, w.send(msg.String()))
	}
	return errors.Join(errs...)
}
