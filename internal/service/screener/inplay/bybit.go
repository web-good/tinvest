package inplay

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"slices"
	"strings"
	"time"

	"tinvest/internal/service/screener/bybitdensity/feed"
	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/pkg/indicators"
)

type bybitMarket interface {
	SpotTurnovers(context.Context) ([]model.Instrument, error)
	AvailableSpotBars(context.Context, model.Key, time.Duration, time.Time, time.Time) ([]feed.SpotBar, error)
}

type BybitWorker struct {
	market bybitMarket
	send   func(string) error
}

func NewBybit(send func(string) error) *BybitWorker {
	return &BybitWorker{market: feed.NewClient("", ""), send: send}
}

func (w *BybitWorker) Run(ctx context.Context) error {
	for {
		end := time.Now().UTC().Truncate(time.Hour).Add(time.Hour)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(end.Add(5 * time.Second))):
		}
		if err := w.scan(ctx, end); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "inplay Bybit scan failed", "error", err)
		}
	}
}

func bybitNATR(bars []feed.SpotBar, end time.Time) (float64, bool) {
	const period = 14
	if len(bars) <= period {
		return 0, false
	}
	highs := make([]float64, len(bars))
	lows := make([]float64, len(bars))
	closes := make([]float64, len(bars))
	for i, b := range bars {
		expected := end.Add(-time.Duration(len(bars)-1-i) * step)
		if !b.End.Equal(expected) {
			return 0, false
		}
		highs[i], lows[i], closes[i] = b.High, b.Low, b.Close
	}
	return 100 * indicators.ATR(highs, lows, closes, period) / closes[len(closes)-1], true
}

func (w *BybitWorker) scan(ctx context.Context, end time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Minute)
	defer cancel()
	instruments, err := w.market.SpotTurnovers(ctx)
	if err != nil {
		return err
	}
	type row struct {
		symbol         string
		turnover, natr float64
	}
	var rows []row
	var firstErr error
	var failed, checked int
	for _, inst := range instruments {
		if inst.Turnover <= 80_000_000 {
			continue
		}
		bars, err := w.market.AvailableSpotBars(ctx, inst.Key, step, end.Add(-24*time.Hour), end)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", inst.Key.Symbol, err)
			}
			failed++
			continue
		}
		natr, ok := bybitNATR(bars, end)
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: insufficient continuous M5 history", inst.Key.Symbol)
			}
			failed++
			continue
		}
		checked++
		// Decimal OHLC prices can lose precision when high and low are subtracted.
		const epsilon = 1e-12
		if natr >= 0.4-epsilon && natr <= 1.7+epsilon {
			rows = append(rows, row{inst.Key.Symbol, inst.Turnover, natr})
		}
	}
	if checked == 0 && firstErr != nil {
		return firstErr
	}
	slices.SortFunc(rows, func(a, b row) int {
		if order := cmp.Compare(b.turnover, a.turnover); order != 0 {
			return order
		}
		return cmp.Compare(a.symbol, b.symbol)
	})
	header := fmt.Sprintf("inplay · Bybit Spot · USDT · %s МСК\nОборот 24ч > 80 млн USDT · NATR(14) M5 0.4–1.7%%", end.In(time.FixedZone("MSK", 3*3600)).Format("15:04"))
	var msg strings.Builder
	msg.WriteString(header)
	for _, r := range rows {
		line := fmt.Sprintf("\n%s    %.2f млн USDT    NATR %.2f%%", html.EscapeString(r.symbol), r.turnover/1e6, r.natr)
		if msg.Len()+len(line) > 4000 {
			if err := w.send(msg.String()); err != nil {
				return errors.Join(firstErr, err)
			}
			msg.Reset()
			msg.WriteString(header)
		}
		msg.WriteString(line)
	}
	if len(rows) == 0 {
		msg.WriteString("\nСреди проверенных пар совпадений нет.")
	}
	if failed > 0 {
		fmt.Fprintf(&msg, "\nБез данных: %d", failed)
	}
	return errors.Join(firstErr, w.send(msg.String()))
}
