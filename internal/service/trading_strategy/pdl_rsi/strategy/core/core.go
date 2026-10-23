// Package core implements pdl_rsi: a long-only strategy at yesterday's low (PDL) on 15-minute
// bars with an hourly RSI. A setup arms when the hourly RSI falls below RSIArm while the price is
// in the zone PDL..PDL + ZoneATR daily ATRs; the entry comes when the hourly RSI crosses back up
// through RSIEntry, filtered by the day's remaining room (MinRoomATR) and the volume (VolumeMult).
// The position is managed by a stop, a breakeven once the hourly RSI passes RSIBreakeven and an
// exit on the hourly RSI crossing RSIExit. Pure, stateless between bars and ticker-agnostic.
// Run with `-strategy pdl_rsi -interval Minutes15`.
package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// dayBars is the number of 15-minute bars from 07:00 to 23:45 (the last bar of the evening session)
// inclusive: a setup is never older than that.
const dayBars = 68

// holdDays is the weekdays of history the breakeven scan keeps for an open position.
const holdDays = 10

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	RSIPeriod int // Wilder period of the hourly RSI; 0 — no entries
	// RSIArm: a setup arms when the hourly RSI is below this
	RSIArm float64
	// RSIEntry: the entry is the hourly RSI crossing up through this
	RSIEntry float64
	// RSIGate: the hourly RSI must be below this at the entry; 0 — no gate
	RSIGate float64
	// RSIBreakeven: once the hourly RSI passes this the stop moves to the fill; 0 — no breakeven
	RSIBreakeven float64
	// RSIExit: the hourly RSI crossing up through this exits; 0 — no RSI exit
	RSIExit float64
	// ZoneATR is the zone's top above PDL in daily ATR
	ZoneATR float64

	// VolumeMult > 0: the entry window's average main-session volume must be above VolumeMult ×
	// the average of the VolBars main-session bars before it; 0 — volume off
	VolumeMult float64
	VolBars    int

	// MinRoomATR > 0: the day must have this many daily ATRs of upside left at the signal; 0 — room off
	MinRoomATR float64

	DailyATRPeriod int // Wilder period over completed weekday dailies
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      6,
		RSIArm:         30,
		RSIEntry:       30,
		RSIGate:        50,
		RSIBreakeven:   50,
		RSIExit:        70,
		ZoneATR:        0.4,
		VolumeMult:     1.5,
		VolBars:        20,
		MinRoomATR:     0.6,
		DailyATRPeriod: 14,
	}
}

// Strategy trades a single instrument with the pdl_rsi rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams returns the strategy for a ticker; step is the bar length (15m).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback covers holdDays weekdays plus a weekend for the breakeven scan, whole days for the
// VolBars main-session bars before a window that starts early in the day, the current day and
// the hourly RSI warm-up (four 15m bars per hour).
func (s *Strategy) Lookback() int {
	volDays := (max(s.p.VolBars, 1) + sessionBars - 1) / sessionBars
	return max(4*(4*s.p.RSIPeriod+2+64), (holdDays+volDays+3)*dayBars)
}

// mskLoc anchors every calendar rule to Moscow (UTC fallback).
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

// dayKey is the MSK calendar date of t as yyyymmdd.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
}

// minuteOf is the MSK minute of the day of t.
func minuteOf(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Hour()*60 + tl.Minute()
}

// series is the bars, oldest-first: opens, highs, lows, closes, volumes and start times. v is nil
// when the bars carry no volumes.
type series struct {
	o, h, l, c []float64
	v          []int64
	t          []time.Time
}

// barsOf is the engine's bars as a series (no copy).
func barsOf(md strategy.MarketData) series {
	return series{o: md.Opens, h: md.Highs, l: md.Lows, c: md.Closes, v: md.Volumes, t: md.Times}
}

// weekdayDaily drops weekend (Sat/Sun MSK) dailies, keeping the price slices aligned. When times
// is empty or misaligned the series is returned untouched rather than guessed at.
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

// pdl returns yesterday's low: the low of the last completed weekday daily.
func (s *Strategy) pdl(md strategy.MarketData) (float64, bool) {
	_, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if len(c) == 0 || len(l) != len(c) {
		return 0, false
	}
	return l[len(l)-1], l[len(l)-1] > 0
}

// dailyATR is Wilder's ATR over completed weekday dailies; 0 when the data cannot support it.
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

// entryAllowed reports whether the current bar may buy at the next open: a weekday, and the buy
// bar starts on the signal's date.
func (s *Strategy) entryAllowed(md strategy.MarketData) bool {
	t := md.Times[len(md.Times)-1]
	return !isWeekend(t.In(mskLoc)) && dayKey(t.Add(s.step)) == dayKey(t)
}

// roomLeft is the upside left in the day at bar k, in daily ATRs: the low of bar k's date up to
// and including k plus one ATR, minus the close of k.
func roomLeft(ser series, k int, atr float64) float64 {
	lo := ser.l[k]
	for j := k - 1; j >= 0 && dayKey(ser.t[j]) == dayKey(ser.t[k]); j-- {
		lo = min(lo, ser.l[j])
	}
	return (lo + atr - ser.c[k]) / atr
}

// Decide is a stub until the entry and exit rules land: it only echoes the ticker and price.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	return model.Signal{Ticker: s.ticker, Price: md.Price}
}
