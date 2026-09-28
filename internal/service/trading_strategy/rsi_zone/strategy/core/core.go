// Package core implements a long-only multi-day RSI zone strategy. When flat it buys a short RSI
// crossing DOWN through its lower critical band on the current bar while the close sits above a
// single trend EMA. It sells when the same RSI crosses UP through its upper critical band, or
// when the protective stop — sized in daily ATR at entry and frozen on the position — is
// touched. There is no target, no trail, no time stop and no end-of-day close: the position is
// held across nights and weekends until one of the two exits fires. The decision logic is pure,
// stateless between bars and ticker-agnostic. The reference timeframe is 30 minutes. Run with
// `-strategy rsi_zone -interval Minutes30`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback floors the candle window at roughly one trading week of 30-minute bars, so the
// RSI always sees enough history even with a short trend EMA.
const minLookback = 120

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod      int     // RSI length (grid; default 4)
	RSILower       float64 // lower critical band; a DOWNWARD cross of it is the entry (grid; default 25)
	RSIUpper       float64 // upper critical band; an UPWARD cross of it is the exit (grid; default 75)
	EMAPeriod      int     // trend EMA period; the entry needs close > EMA (grid; default 200)
	DailyATRPeriod int     // daily ATR length, over WEEKDAY completed dailies (fixed; default 14)
	StopDailyATR   float64 // stop = entry - StopDailyATR*dailyATR; 0 disables it (grid; never 0 in the grid)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      200,
		DailyATRPeriod: 14,
		StopDailyATR:   1.0,
	}
}

// Strategy trades a single instrument with the RSI zone rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window the engine feeds Decide on every bar. ema.Compute seeds on
// an SMA over the first `period` closes, so a window shorter than the period yields an all-zero
// series that silently fails the trend gate for the whole run. Doubling the largest period
// leaves as many recursion steps as the seed span; the +20 covers the two-bar cross lookups.
func (s *Strategy) Lookback() int {
	return max(minLookback, 2*max(s.p.EMAPeriod, s.p.RSIPeriod)+20)
}

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
// not aligned with the price slices there is nothing to filter by — the series is returned
// untouched rather than guessed at.
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
func (s *Strategy) tradingDay(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	return !isWeekend(t.In(mskLoc))
}

// barTime returns the open-time of the latest bar, or the zero time when Times is absent or
// misaligned with Closes (so the weekday gate degrades instead of misfiring).
func (s *Strategy) barTime(md strategy.MarketData) time.Time {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n {
		return time.Time{}
	}
	return md.Times[n-1]
}

// crossedDown reports whether series crossed down through level between i-1 and i: it sat at or
// above the level and is now strictly below. The series[i-1] > 0 guard rejects RSISeries warm-up
// zeros reading as "below the level".
func crossedDown(series []float64, i int, level float64) bool {
	return i >= 1 && i < len(series) && series[i-1] > 0 && series[i-1] >= level && series[i] < level
}

// crossedUp reports whether series crossed up through level between i-1 and i: it sat at or
// below the level and is now strictly above. The series[i-1] > 0 guard keeps an RSISeries
// warm-up zero from manufacturing an exit out of nothing.
func crossedUp(series []float64, i int, level float64) bool {
	return i >= 1 && i < len(series) && series[i-1] > 0 && series[i-1] <= level && series[i] > level
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

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return sig
	}
	return s.enter(md, sig)
}

// enter emits a long when a short RSI crosses DOWN through its lower band on the current bar
// while the close sits above the trend EMA on a weekday and the daily ATR is known. Everything
// is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n || s.p.RSIPeriod <= 0 || s.p.EMAPeriod <= 0 {
		return sig
	}
	// 1. weekday: any time of a trading day will do, weekends will not.
	if !s.tradingDay(s.barTime(md)) {
		return sig
	}
	i := n - 1
	// 2. RSI crosses down through the lower band on the current bar.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n || !crossedDown(rsi, i, s.p.RSILower) {
		return sig
	}
	// 3. trend: close above a warmed EMA.
	trend := ema.Compute(md.Closes, s.p.EMAPeriod)
	if len(trend) != n || !trendUp(md.Closes[i], trend[i]) {
		return sig
	}
	// 4. the daily ATR is the unit of the stop: no ATR, no trade.
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	// 5. an armed stop must land above zero, otherwise the long would be unprotected.
	entry := md.Closes[i]
	stop := StopLevel(s.p, entry, atr)
	if s.p.StopDailyATR > 0 && stop <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i], trend[i], entry, stop, atr)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(rsiNow, emaNow, entry, stop, atr float64) string {
	stopHow := "стоп выключен"
	if stop > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (−%.2f ATR)", stop, s.p.StopDailyATR)
	}
	return fmt.Sprintf(
		"RSI(%d) ушёл под %.0f (%.1f), close %.4f > EMA(%d) %.4f, дневной ATR %.4f; вход %.4f, %s",
		s.p.RSIPeriod, s.p.RSILower, rsiNow, entry, s.p.EMAPeriod, emaNow, atr, entry, stopHow,
	)
}
