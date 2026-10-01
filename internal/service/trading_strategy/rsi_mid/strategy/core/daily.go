package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// The calendar and daily-ATR helpers below are copied from the rsi_zone core on purpose: moving
// them into a shared package would touch production rsi_zone code and its golden tests for the
// sake of a hundred lines.

// mskLoc anchors every calendar rule (weekday checks) to Moscow (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// isWeekend reports whether tl (already in MSK) falls on a non-trading day.
func isWeekend(tl time.Time) bool {
	wd := tl.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// weekdayDaily drops weekend (Sat/Sun MSK) bars from the daily series, keeping the three price
// slices aligned. MOEX weekend sessions are several times narrower and thinner than weekday
// ones; leaving them in understates the daily ATR that sizes the stop. When times is empty or
// not aligned with the price slices the series is returned untouched rather than guessed at.
func weekdayDaily(highs, lows, closes []float64, times []time.Time) (h, l, c []float64) {
	n := len(closes)
	if n == 0 || len(highs) != n || len(lows) != n || len(times) != n {
		return highs, lows, closes
	}
	h = make([]float64, 0, n)
	l = make([]float64, 0, n)
	c = make([]float64, 0, n)
	for i := 0; i < n; i++ {
		if isWeekend(times[i].In(mskLoc)) {
			continue
		}
		h = append(h, highs[i])
		l = append(l, lows[i])
		c = append(c, closes[i])
	}
	return h, l, c
}

// dailyATR is the strategy's unit of risk: Wilder's ATR over COMPLETED weekday daily candles.
// The engine only exposes days that closed before the current bar, so no lookahead is possible.
// Returns 0 whenever the data cannot support the calculation — the caller must then refuse the
// entry, because without it there is no stop.
func (s *Strategy) dailyATR(md strategy.MarketData) float64 {
	if s.p.DailyATRPeriod <= 0 {
		return 0
	}
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if len(c) < s.p.DailyATRPeriod+1 || len(h) != len(c) || len(l) != len(c) {
		return 0
	}
	return indicators.ATR(h, l, c, s.p.DailyATRPeriod)
}

// tradingDay reports whether an entry may be opened on bar-time t: any bar of a Mon-Fri MSK
// day. A zero time skips the gate — never block on missing data.
func tradingDay(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	return !isWeekend(t.In(mskLoc))
}

// barTime returns the open-time of the latest bar, or the zero time when Times is absent or
// misaligned with Closes (so the weekday gate degrades instead of misfiring).
func barTime(md strategy.MarketData) time.Time {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n {
		return time.Time{}
	}
	return md.Times[n-1]
}

// crossedUp reports whether series crossed up through level between i-1 and i: it sat at or
// below the level and is now strictly above. period is the RSI length used to build series:
// RSISeries leaves indices below period as an unset zero rather than a genuine reading, so
// validity is gated on the INDEX (i-1 >= period), not on the VALUE — a genuine RSI of 0.00 is a
// real reading.
func crossedUp(series []float64, i, period int, level float64) bool {
	return i >= 1 && i < len(series) && i-1 >= period && series[i-1] <= level && series[i] > level
}

// trendUp reports whether the close sits strictly above a warmed EMA. ema.Compute zero-fills
// warm-up positions, so an unwarmed EMA must not pass as "price above zero".
func trendUp(closeP, emaNow float64) bool {
	return emaNow > 0 && closeP > emaNow
}

// StopLevel returns the protective stop for a position opened at entry with the daily ATR
// frozen at entry, or 0 when there is no stop: the stop is disabled, the ATR is unknown, or the
// level would land at or below zero (that is a naked long, not a floor).
func StopLevel(p Params, entry, entryATR float64) float64 {
	if p.StopDailyATR <= 0 || entryATR <= 0 {
		return 0
	}
	level := entry - p.StopDailyATR*entryATR
	if level <= 0 {
		return 0
	}
	return level
}
