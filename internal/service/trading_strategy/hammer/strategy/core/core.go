// Package core implements hammer: a long-only strategy on hourly bars. The signal is a hammer
// candle (isHammer) whose own close has RSI(RSIPeriod) below RSIZone; the buy comes at the next
// bar's open on the same MSK weekday date.
//
// The stop is StopDailyATR daily ATRs (Wilder DailyATRPeriod over completed weekday dailies) under
// the entry; the target is TargetRangeMult × yesterday's daily range (high − low of the last
// completed weekday daily) above the entry. Both distances are frozen at the signal: the buy signal
// carries the levels off the signal close for sizing, and the position rebuilds them off the fill
// price (exitLevels). Exits in priority order: the stop, the target, a close with RSI above RSIExit.
// The position carries overnight and over weekends. Pure, stateless between bars and
// ticker-agnostic. Run with `-strategy hammer -interval Hour1`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	RSIPeriod int     // RSI length over the hourly closes
	RSIZone   float64 // the hammer bar's own RSI must be below this
	RSIExit   float64 // sell on a close whose RSI is above this; 0 — off

	StopDailyATR float64 // the stop distance under the entry in daily ATRs; ≤ 0 — no entries
	// TargetRangeMult > 0: the target is this share of yesterday's daily range above the entry;
	// 0 — no target
	TargetRangeMult float64

	DailyATRPeriod int // Wilder period over completed weekday dailies
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		RSIPeriod:       5,
		RSIZone:         30,
		RSIExit:         70,
		StopDailyATR:    0.5,
		TargetRangeMult: 0.5,
		DailyATRPeriod:  14,
	}
}

// Strategy trades a single instrument with the hammer rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams returns the strategy for a ticker; step is the bar length (1h).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback is enough hourly bars for Wilder's RSI to settle.
func (s *Strategy) Lookback() int { return max(20*s.p.RSIPeriod, 100) }

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

// isHammer: a bar of either colour has a body of at most a third of its range lying in the upper
// third — so the lower tail is at least two thirds of the range and at least twice the body.
// Copied from pdl_reclaim/strategy/core/setup.go:isHammer.
func isHammer(o, h, l, c float64) bool {
	rng := h - l
	if rng <= 0 {
		return false
	}
	return max(o, c)-min(o, c) <= rng/3 && min(o, c) >= l+rng*2/3
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

// dailyLevels returns Wilder's daily ATR and yesterday's range (high − low of the last completed
// weekday daily); ok is false when the dailies cannot support either.
func (s *Strategy) dailyLevels(md strategy.MarketData) (atr, rng float64, ok bool) {
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if s.p.DailyATRPeriod <= 0 || len(c) < s.p.DailyATRPeriod+1 || len(h) != len(c) || len(l) != len(c) {
		return 0, 0, false
	}
	atr = indicators.ATR(h, l, c, s.p.DailyATRPeriod)
	rng = h[len(h)-1] - l[len(l)-1]
	return atr, rng, atr > 0 && rng > 0
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys the next open when the current bar is a hammer with its own RSI below RSIZone, on a
// weekday whose next bar starts on the same date.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	k := n - 1
	t := md.Times[k]
	if s.p.StopDailyATR <= 0 || s.p.RSIPeriod <= 0 || k <= s.p.RSIPeriod ||
		isWeekend(t.In(mskLoc)) || dayKey(t.Add(s.step)) != dayKey(t) {
		return sig
	}
	o, h, l, c := md.Opens[k], md.Highs[k], md.Lows[k], md.Closes[k]
	if !isHammer(o, h, l, c) {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)[k]
	if rsi >= s.p.RSIZone {
		return sig
	}
	atr, rng, ok := s.dailyLevels(md)
	if !ok {
		return sig
	}
	stop := c - s.p.StopDailyATR*atr
	if stop <= 0 {
		return sig
	}
	target, targetWhy := 0.0, "цели нет"
	if s.p.TargetRangeMult > 0 {
		target = c + s.p.TargetRangeMult*rng
		targetWhy = fmt.Sprintf("цель %.2f вчерашнего диапазона %.4f: %.4f", s.p.TargetRangeMult, rng, target)
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.TakeProfit, sig.ATR = stop, target, atr
	sig.FillBy = t.Add(s.step)
	sig.EntryReason = fmt.Sprintf("молот 1h на свече %s (o %.4f h %.4f l %.4f c %.4f), RSI(%d) %.1f < %.0f, дневной ATR %.4f, стоп %.2f ATR %.4f, %s (стоп и цель пересчитываются от цены покупки)",
		t.In(mskLoc).Format("01-02 15:04"), o, h, l, c, s.p.RSIPeriod, rsi, s.p.RSIZone, atr, s.p.StopDailyATR, stop, targetWhy)
	return sig
}

// manage closes the position; filled in with the exits.
func (s *Strategy) manage(_ strategy.MarketData, sig model.Signal) model.Signal {
	return sig
}
