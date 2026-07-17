// Package core implements the SMC liquidity-sweep long-only swing strategy:
// a fractal swing-low is confirmed SwingK bars after its extreme; a bar
// piercing that level with its low and a close back above it within
// ReclaimBars is a stop-hunt — the reclaim bar buys at its close with a hard
// stop under the sweep extreme; exits are the intrabar stop, an R-multiple
// take-profit and a trading-day time-stop. Optional OB/FVG/discount filters
// are int toggles (0/1) so the calibration grid can sweep them. See
// docs/superpowers/specs/2026-07-17-smc-liquidity-sweep-design.md.
package core

import (
	"time"
)

// Fixed knobs — deliberately NOT part of Params: sweeping window/warm-up
// mechanics is how past strategies overfit.
const (
	atrPeriod        = 14 // stop unit
	levelWindowDays  = 10 // sliding window of distinct MSK days where levels live
	sessionOpenHour  = 10 // MSK: bars opening before 10:00 never enter
	eveningStartHour = 19 // MSK: bars opening at/after 19:00 (evening session) never enter
	// barsPerDayMax is the worst-case Hour1 bar count of one MSK day
	// (morning + main + evening sessions). Lookback must cover the whole
	// level window plus indicator warm-up.
	barsPerDayMax = 17
)

// Params are the SMC tunables. UseOB/UseFVG/UseDiscount are int toggles
// (grid values are numeric): 0 = off, 1 = on.
type Params struct {
	SwingK      int     // fractal wing: a swing-low needs SwingK strictly higher lows on each side
	ReclaimBars int     // max bars from pierce to reclaim close (0 = same-bar sweep only)
	Buffer      float64 // hard stop = sweepLow - Buffer*ATR(atrPeriod) at entry
	TPR         float64 // take-profit = entry + TPR*(entry - stop); <=0 disables
	MaxHoldDays int     // time-stop after this many distinct trading days; <=0 disables
	UseOB       int     // 1 = reclaim close must sit inside an unmitigated bullish order block
	UseFVG      int     // 1 = a bullish FVG must form between pierce and reclaim
	UseDiscount int     // 1 = entry close must be below the level-window range midpoint
}

// Strategy is the SMC rule bound to one ticker.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams builds the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p}
}

// Ticker returns the bound instrument ticker.
func (s *Strategy) Ticker() string { return s.ticker }

// Lookback returns the candle window the strategy needs: the full level
// window plus ATR warm-up.
func (s *Strategy) Lookback() int { return levelWindowDays*barsPerDayMax + atrPeriod + 2 }

// mskLoc anchors all session logic to the Moscow trading calendar (UTC
// fallback if the tz DB is absent), mirroring the backtest engine.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// isWeekend reports whether t falls on Saturday or Sunday in mskLoc.
func isWeekend(t time.Time) bool {
	wd := t.In(mskLoc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// sameMSKDay reports whether a and b share an MSK calendar day.
func sameMSKDay(a, b time.Time) bool {
	al, bl := a.In(mskLoc), b.In(mskLoc)
	return al.Year() == bl.Year() && al.YearDay() == bl.YearDay()
}

// startOfMSKDay returns midnight of t's MSK calendar day.
func startOfMSKDay(t time.Time) time.Time {
	tl := t.In(mskLoc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, mskLoc)
}

// windowStart returns the index of the oldest bar belonging to the last
// levelWindowDays distinct MSK days of times. Levels whose swing bar is
// older are forgotten.
func windowStart(times []time.Time) int {
	days := 0
	for i := len(times) - 1; i >= 0; i-- {
		if i == len(times)-1 || !sameMSKDay(times[i], times[i+1]) {
			days++
			if days > levelWindowDays {
				return i + 1
			}
		}
	}
	return 0
}

// tradingDaysSince counts distinct MSK days among times strictly after
// entry's day. Only days visible in the window count — safe while
// MaxHoldDays*barsPerDayMax stays well inside Lookback.
func tradingDaysSince(times []time.Time, entry time.Time) int {
	entryDay := startOfMSKDay(entry)
	days := 0
	var lastDay time.Time
	for _, t := range times {
		d := startOfMSKDay(t)
		if !d.After(entryDay) || d.Equal(lastDay) {
			continue
		}
		days++
		lastDay = d
	}
	return days
}
