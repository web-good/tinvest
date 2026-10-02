package backtest

// Pure core of cmd/gapscreen: structural metrics that decide whether a ticker's 07:00 gap_fade
// entry is executable, and the gates over them. Profit is deliberately not measured — a ticker
// has a handful of gap signals a year, and the universe-wide profit check is cmd/gapwf's job.
// Spec: docs/superpowers/specs/2026-10-02-gap-fade-screener-design.md.

import (
	"fmt"
	"time"

	"tinvest/internal/domain/backtest"
)

// GapScreenOptions are the gate thresholds.
type GapScreenOptions struct {
	MinMorningDays      float64 // minimum share of weekday dates with a 07:00 bar, 0..1
	MinMorningTurnoverM float64 // minimum median 07:00-bar turnover, millions of RUB
	MaxTickPct          float64 // maximum two price ticks as % of the last close; 0 = off
	MinTurnoverM        float64 // minimum mean daily turnover, millions of RUB
}

// DefaultGapScreenOptions are the spec defaults: a 100k RUB position is 2% of a 5M 07:00 bar.
func DefaultGapScreenOptions() GapScreenOptions {
	return GapScreenOptions{MinMorningDays: 0.9, MinMorningTurnoverM: 5, MaxTickPct: 0.2, MinTurnoverM: 50}
}

// GapScreenInput is one ticker's data for ScreenGap.
type GapScreenInput struct {
	Ticker, Name      string
	Bars              []backtest.Candle // 30m, oldest-first
	Lot               int32
	MinPriceIncrement float64 // 0 when the API did not report one
	SignalsPerYear    float64 // from GapSignalsPerYear; informational
}

// GapScreenRow is one ticker's metrics and gate outcome.
type GapScreenRow struct {
	Ticker, Name     string
	MorningDays      float64 // share of weekday dates with a 07:00 bar
	MorningTurnoverM float64 // median 07:00-bar turnover, millions of RUB
	TurnoverM        float64 // mean daily turnover, millions of RUB
	TickPct          float64 // two ticks as % of the last close; 0 = unknown tick
	SignalsPerYear   float64
	Pass             bool
	Reasons          []string // one per failed gate
}

// GapMorningStats returns the share of weekday MSK dates that have a bar starting at 07:00 (the
// first continuous-trading bar, where gap_fade buys) and the median turnover of those bars in
// millions of RUB. Weekend sessions are ignored; a duplicate 07:00 bar on one date counts once.
func GapMorningStats(bars []backtest.Candle, lot int32) (days, turnoverM float64) {
	weekdays := map[string]bool{}
	seen := map[string]bool{}
	var morning []float64
	for _, b := range bars {
		tl := b.Time.In(gapLoc)
		if tl.Weekday() == time.Saturday || tl.Weekday() == time.Sunday {
			continue
		}
		day := tl.Format("2006-01-02")
		weekdays[day] = true
		if tl.Hour() != 7 || tl.Minute() != 0 || seen[day] {
			continue
		}
		seen[day] = true
		morning = append(morning, float64(b.Volume)*float64(lot)*b.Close/1e6)
	}
	if len(weekdays) == 0 {
		return 0, 0
	}
	return float64(len(morning)) / float64(len(weekdays)), medianF(morning)
}

// GapSignalsPerYear counts gap_fade trades over the window with the shared default parameters,
// run exactly as cmd/gapwf runs them (next-open entry), so the screener never re-implements the
// gap rule. Only the count is used — the run's profit stays out of the screen. Dividend ex-days
// are not filtered here.
func GapSignalsPerYear(ticker string, bars, daily []backtest.Candle, lot int32, months int) float64 {
	if months <= 0 {
		return 0
	}
	b := GapFadeLookupOrGeneric(ticker)
	cfg := backtest.Config{InitialCash: 1e7, Fraction: 1.0, Commission: 0.0005, Lot: lot, EntryAtNextOpen: true}
	res := backtest.Run(b.Build(b.DefaultParams()), bars, daily, nil, cfg)
	return float64(len(res.Trades)) / (float64(months) / 12)
}

// ApplyGapGates sets Pass and Reasons from the row's metrics. Equality passes every gate.
func ApplyGapGates(row GapScreenRow, opts GapScreenOptions) GapScreenRow {
	var reasons []string
	if row.MorningDays < opts.MinMorningDays {
		reasons = append(reasons, fmt.Sprintf("бар 07:00 в %.0f%% будних дней < %.0f%%", row.MorningDays*100, opts.MinMorningDays*100))
	}
	if row.MorningTurnoverM < opts.MinMorningTurnoverM {
		reasons = append(reasons, fmt.Sprintf("утренний оборот %.1f млн ₽ < %.1f", row.MorningTurnoverM, opts.MinMorningTurnoverM))
	}
	if opts.MaxTickPct > 0 && row.TickPct > opts.MaxTickPct {
		reasons = append(reasons, fmt.Sprintf("два шага %.2f%% > %.2f%%", row.TickPct, opts.MaxTickPct))
	}
	if row.TurnoverM < opts.MinTurnoverM {
		reasons = append(reasons, fmt.Sprintf("дневной оборот %.0f млн ₽ < %.0f", row.TurnoverM, opts.MinTurnoverM))
	}
	row.Reasons = reasons
	row.Pass = len(reasons) == 0
	return row
}

// ScreenGap measures one ticker and applies the gates.
func ScreenGap(in GapScreenInput, opts GapScreenOptions) GapScreenRow {
	row := GapScreenRow{Ticker: in.Ticker, Name: in.Name, SignalsPerYear: in.SignalsPerYear}
	row.MorningDays, row.MorningTurnoverM = GapMorningStats(in.Bars, in.Lot)
	row.TurnoverM = backtest.MeanDailyTurnoverM(in.Bars, in.Lot)
	if n := len(in.Bars); n > 0 && in.MinPriceIncrement > 0 && in.Bars[n-1].Close > 0 {
		row.TickPct = 2 * in.MinPriceIncrement / in.Bars[n-1].Close * 100
	}
	return ApplyGapGates(row, opts)
}
