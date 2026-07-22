// Package core implements a long-only 5-minute RSI+MACD scalping strategy. When flat it
// looks for a fast RSI crossing up out of its oversold zone, confirmed within a few bars
// by a MACD(3,6,9) bullish line cross that happens BELOW zero. The stop is frozen at the
// low of the RSI-cross candle; the take-profit is RR times the entry risk. An optional
// stochastic exit closes the position when %K leaves the overbought zone downward, and
// every position is force-closed at the end of the trading day. The decision logic is
// pure, stateless between bars and ticker-agnostic. Run with
// `-strategy scalping_rsimacd -interval Minutes5`.
package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// barSpanMin is the bar length in minutes. The strategy is defined on 5-minute candles;
// the EOD gate uses it to close on the LAST bar that still ends inside the session.
const barSpanMin = 5

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	RSIPeriod       int     // fast RSI length (grid 3/4/5)
	RSIOversold     float64 // lower critical zone (grid 20/25/30)
	MACDFast        int     // MACD fast EMA (fixed 3)
	MACDSlow        int     // MACD slow EMA (fixed 6)
	MACDSignal      int     // MACD signal EMA (fixed 9)
	MACDConfirmBars int     // MACD cross accepted on the RSI bar or the next N bars (grid 2/3/4)
	ATRPeriod       int     // ATR length; the unit of the risk sanity bounds
	StopBufferATR   float64 // stop = low(RSI bar) - StopBufferATR*ATR (grid 0/0.1)
	RR              float64 // take-profit = entry + RR*(entry-stop) (grid 2/2.5/3)
	MinRiskATR      float64 // reject entries whose risk < MinRiskATR*ATR
	MaxRiskATR      float64 // reject entries whose risk > MaxRiskATR*ATR
	EnableStochExit int     // 1 = stochastic exit active; 0 = SL/TP/EOD only (grid 0/1)
	StochK          int     // stochastic %K period (fixed 14)
	StochD          int     // stochastic %D smoothing (fixed 3)
	StochOverbought float64 // upper critical zone for the exit (fixed 80)
	SessionStartMin int     // entry window start, minutes from MSK midnight (480 = 08:00)
	SessionEndMin   int     // Mon-Thu session end, minutes from MSK midnight (1020 = 17:00)
	FridayEndMin    int     // Friday session end, minutes from MSK midnight (840 = 14:00)
}

// DefaultParams returns the spec's baseline; the swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:       3,
		RSIOversold:     30,
		MACDFast:        3,
		MACDSlow:        6,
		MACDSignal:      9,
		MACDConfirmBars: 3,
		ATRPeriod:       14,
		StopBufferATR:   0,
		RR:              2.0,
		MinRiskATR:      0.1,
		MaxRiskATR:      3.0,
		EnableStochExit: 1,
		StochK:          14,
		StochD:          3,
		StochOverbought: 80,
		SessionStartMin: 480,
		SessionEndMin:   1020,
		FridayEndMin:    840,
	}
}

// Strategy trades a single instrument with the RSI+MACD rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to warm every consumer with margin: Wilder RSI
// (period 3-5), MACD(3,6,9), the stochastic (14+3) and the confirmation window.
func (s *Strategy) Lookback() int { return 120 }

// mskLoc anchors the session windows to the Moscow calendar (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// sessionEndMin returns the session end for the weekday of tl (Friday closes early).
func (s *Strategy) sessionEndMin(tl time.Time) int {
	if tl.Weekday() == time.Friday {
		return s.p.FridayEndMin
	}
	return s.p.SessionEndMin
}

// isWeekend reports whether tl (already in MSK) falls on a non-trading day.
func isWeekend(tl time.Time) bool {
	wd := tl.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// inSession reports whether bar-time t falls inside the entry window in MSK. A zero time
// (live paths without Times) skips the gate — never block on missing data.
func (s *Strategy) inSession(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return false
	}
	m := tl.Hour()*60 + tl.Minute()
	return m >= s.p.SessionStartMin && m < s.sessionEndMin(tl)
}

// isDayEnd reports whether the bar opening at t is the last one that still ends inside
// the session (or already sits outside it). A zero time degrades the EOD exit to a no-op.
func (s *Strategy) isDayEnd(t time.Time) bool {
	if t.IsZero() {
		return false
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return true
	}
	m := tl.Hour()*60 + tl.Minute()
	return m+barSpanMin >= s.sessionEndMin(tl)
}

// barTime returns the open-time of the latest bar, or the zero time when Times is absent
// or misaligned with Closes (so the time-based gates degrade instead of misfiring).
func (s *Strategy) barTime(md strategy.MarketData) time.Time {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n {
		return time.Time{}
	}
	return md.Times[n-1]
}
