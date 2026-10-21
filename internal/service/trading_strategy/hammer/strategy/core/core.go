// Package core implements hammer: a long-only strategy on hourly or 30-minute bars. The signal is a hammer
// candle (isHammer) whose own close has RSI(RSIPeriod) below RSIZone and, with TrendEMA on, sits
// strictly above the EMA(TrendEMA) of the bar closes; the buy comes at the next bar's open on
// the same MSK weekday date.
//
// The stop is StopDailyATR daily ATRs (Wilder DailyATRPeriod over completed weekday dailies) under
// the entry; the target is TargetRangeMult × yesterday's daily range (high − low of the last
// completed weekday daily) above the entry. Both distances are frozen at the signal: the buy signal
// carries the levels off the signal close for sizing, and the position rebuilds them off the fill
// price (exitLevels). Exits in priority order: the stop, the target, a close with RSI above RSIExit.
// The position carries overnight and over weekends. Pure, stateless between bars and
// ticker- and timeframe-agnostic. Run with `-strategy hammer -interval Hour1` (or Minutes30).
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	RSIPeriod int     // RSI length over the bar closes
	RSIZone   float64 // the hammer bar's own RSI must be below this
	RSIExit   float64 // sell on a close whose RSI is above this; 0 — off
	// TrendEMA > 0: the hammer's close must be strictly above a warmed EMA of this period over the
	// bar closes; 0 — off
	TrendEMA int

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
		TrendEMA:        200,
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

// NewWithParams returns the strategy for a ticker; step is the bar length (1h or 30m).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// tf labels the bars in reasons ("1h", "30m").
func (s *Strategy) tf() string {
	if s.step%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(s.step.Hours()))
	}
	return fmt.Sprintf("%dm", int(s.step.Minutes()))
}

// Lookback is enough bars for Wilder's RSI to settle and for the trend EMA to recurse past
// its SMA seed (twice its period).
func (s *Strategy) Lookback() int { return max(20*s.p.RSIPeriod, 100, 2*s.p.TrendEMA+20) }

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
	trendWhy := ""
	if s.p.TrendEMA > 0 {
		e := ema.Compute(md.Closes, s.p.TrendEMA)[k]
		// ema.Compute zero-fills the warm-up: an unwarmed EMA must not pass as "above zero"
		if e <= 0 || c <= e {
			return sig
		}
		trendWhy = fmt.Sprintf(", close над EMA(%d) %.4f", s.p.TrendEMA, e)
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
	sig.EntryReason = fmt.Sprintf("молот %s на свече %s (o %.4f h %.4f l %.4f c %.4f), RSI(%d) %.1f < %.0f%s, дневной ATR %.4f, стоп %.2f ATR %.4f, %s (стоп и цель пересчитываются от цены покупки)",
		s.tf(), t.In(mskLoc).Format("01-02 15:04"), o, h, l, c, s.p.RSIPeriod, rsi, s.p.RSIZone, trendWhy, atr, s.p.StopDailyATR, stop, targetWhy)
	return sig
}

// manage exits in priority order: the stop, the target, a close with RSI above RSIExit. The
// position carries overnight.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	stop, target, ok := s.exitLevels(pos)
	if !ok {
		return sig
	}
	low, high, c := md.Lows[n-1], md.Highs[n-1], md.Closes[n-1]
	if low <= stop {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
		return sig
	}
	if target > 0 && high >= target {
		sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", target
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цели %.4f (вход %.4f)", high, target, pos.PurchasePrice)
		return sig
	}
	if s.p.RSIExit > 0 && s.p.RSIPeriod > 0 && n > s.p.RSIPeriod {
		if rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)[n-1]; rsi > s.p.RSIExit {
			sig.Kind, sig.Reason = model.SignalSell, "RSI"
			sig.ExitReason = fmt.Sprintf("RSI: свеча %s RSI(%d) %.1f > %.0f, выход по %.4f (вход %.4f)",
				md.Times[n-1].In(mskLoc).Format("01-02 15:04"), s.p.RSIPeriod, rsi, s.p.RSIExit, c, pos.PurchasePrice)
		}
	}
	return sig
}

// exitLevels rebuilds the stop and the target off the fill price f: the stop is StopDailyATR daily
// ATRs (frozen at the signal) under f; the target is f plus the target distance frozen at the
// signal, recovered from the signal's levels — (TakeProfit − StopLoss) less the stop distance —
// since both were laid off the same signal close. A zero TakeProfit means no target (target 0).
func (s *Strategy) exitLevels(pos *strategy.Position) (stop, target float64, ok bool) {
	f := pos.PurchasePrice
	d := s.p.StopDailyATR * pos.EntryATR
	if f <= 0 || d <= 0 || pos.StopLoss <= 0 {
		return 0, 0, false
	}
	stop = f - d
	if stop <= 0 {
		return 0, 0, false
	}
	if pos.TakeProfit > 0 {
		target = f + (pos.TakeProfit - pos.StopLoss - d)
	}
	return stop, target, true
}
