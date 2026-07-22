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
	"fmt"
	"strings"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
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

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig) // implemented in Task 3
	}
	return s.enter(md, sig)
}

// enter emits a long when the current bar carries a below-zero MACD bullish cross that
// confirms a recent RSI cross up out of the oversold zone, and the trigger's stop level
// has held since. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. session window (skipped when Times is absent or misaligned).
	if !s.inSession(s.barTime(md)) {
		return sig
	}
	// 2. MACD bullish cross on the current bar, both lines below zero.
	macd, signal := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal)
	if len(macd) != n || len(signal) != n {
		return sig
	}
	i := n - 1
	if !(macd[i-1] <= signal[i-1] && macd[i] > signal[i]) {
		return sig
	}
	if macd[i] >= 0 || signal[i] >= 0 {
		return sig
	}
	// 3. the RSI trigger that this cross confirms.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		return sig
	}
	trig, ok := s.lastRSITrigger(rsi, n)
	if !ok {
		return sig
	}
	// 4. ATR unit for the stop buffer and the risk sanity bounds.
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	if atr <= 0 {
		return sig
	}
	stop := md.Lows[trig] - s.p.StopBufferATR*atr
	// 5. the trigger's stop level must have held since the trigger bar.
	if !s.triggerAlive(md, trig, n, stop) {
		return sig
	}
	// 6. risk sanity: reject degenerate and abnormally wide stops.
	entry := md.Closes[i]
	risk := entry - stop
	if risk < s.p.MinRiskATR*atr || risk > s.p.MaxRiskATR*atr {
		return sig
	}

	tp := entry + s.p.RR*risk
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = tp
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.Level = md.Lows[trig]
	sig.EntryReason = s.entryReason(i-trig, rsi[i], stop, entry, tp, atr)
	return sig
}

// lastRSITrigger returns the most recent bar in [n-1-MACDConfirmBars, n-1] on which RSI
// crossed up through the oversold level and closed above it. The rsi[j-1] > 0 guard is
// load-bearing: RSISeries fills warm-up positions with 0, which would otherwise read as
// "was inside the oversold zone" and manufacture a phantom trigger.
func (s *Strategy) lastRSITrigger(rsi []float64, n int) (int, bool) {
	lo := n - 1 - s.p.MACDConfirmBars
	if lo < 1 {
		lo = 1
	}
	for j := n - 1; j >= lo; j-- {
		if rsi[j-1] > 0 && rsi[j-1] < s.p.RSIOversold && rsi[j] > s.p.RSIOversold {
			return j, true
		}
	}
	return 0, false
}

// triggerAlive reports whether the stop level has held on every bar after the trigger and
// the entry close still sits above it. A bar that already traded through the stop means
// the setup was stopped out before it could be entered.
func (s *Strategy) triggerAlive(md strategy.MarketData, t, n int, stop float64) bool {
	for j := t + 1; j <= n-1; j++ {
		if md.Lows[j] <= stop {
			return false
		}
	}
	return md.Closes[n-1] > stop
}

// entryReason renders the rationale shown in the trade journal. barsAgo is the distance
// from the RSI trigger bar to the entry bar (0 = same bar).
func (s *Strategy) entryReason(barsAgo int, rsiNow, stop, entry, tp, atr float64) string {
	return fmt.Sprintf(
		"RSI(%d) вышел вверх из зоны %.0f (сейчас %.1f), через %d бар(ов) MACD(%d,%d,%d) пересёкся ниже нуля; вход %.4f, стоп %.4f (лоу свечи кросса, буфер %.2f×ATR), тейк %.4f (RR=%.2f), ATR=%.4f",
		s.p.RSIPeriod, s.p.RSIOversold, rsiNow, barsAgo, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal,
		entry, stop, s.p.StopBufferATR, tp, s.p.RR, atr,
	)
}

// manage handles an open long, exiting in precedence SL -> TP -> stochastic -> EOD. SL and
// TP are read from the position (frozen at entry by the engine), never recomputed from
// Params. The stochastic and EOD exits fill at the bar close — their reasons are
// deliberately absent from model.IsStopReason.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Lows)
	if pos == nil || n == 0 || len(md.Highs) != n || len(md.Closes) != n {
		return sig
	}
	low, high := md.Lows[n-1], md.Highs[n-1]

	switch {
	case pos.StopLoss > 0 && low <= pos.StopLoss:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
	case pos.TakeProfit > 0 && high >= pos.TakeProfit:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.TakeProfit = pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ тейк %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
	case s.p.EnableStochExit == 1 && s.stochExit(md):
		sig.Kind, sig.Reason = model.SignalSell, "STOCH"
		sig.ExitReason = fmt.Sprintf("STOCH: %%K вышел вниз из зоны %.0f, выход по закрытию %.4f (вход %.4f)",
			s.p.StochOverbought, md.Closes[n-1], pos.PurchasePrice)
	case s.isDayEnd(s.barTime(md)):
		sig.Kind, sig.Reason = model.SignalSell, "EOD"
		sig.ExitReason = fmt.Sprintf("EOD: закрытие на конец торгового дня по %.4f (вход %.4f)",
			md.Closes[n-1], pos.PurchasePrice)
	}
	return sig
}

// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace
// (-explain). It recomputes the same values enter() does and reports pass/fail per gate.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return "недостаточно свечей\n"
	}
	fmt.Fprintf(&sb, "сессия: %v (бар %v)\n", s.inSession(s.barTime(md)), s.barTime(md))

	macd, signal := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal)
	if len(macd) != n || len(signal) != n {
		sb.WriteString("MACD: недостаточно истории\n")
		return sb.String()
	}
	i := n - 1
	crossed := macd[i-1] <= signal[i-1] && macd[i] > signal[i]
	fmt.Fprintf(&sb, "MACD(%d,%d,%d): линия %.5f сигнал %.5f, пересечение вверх? %v; обе ниже нуля? %v\n",
		s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal, macd[i], signal[i], crossed, macd[i] < 0 && signal[i] < 0)

	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		sb.WriteString("RSI: недостаточно истории\n")
		return sb.String()
	}
	fmt.Fprintf(&sb, "RSI(%d) сейчас %.1f, зона %.0f\n", s.p.RSIPeriod, rsi[i], s.p.RSIOversold)
	trig, ok := s.lastRSITrigger(rsi, n)
	if !ok {
		fmt.Fprintf(&sb, "RSI-триггер в окне %d бар(ов): нет\n", s.p.MACDConfirmBars)
		return sb.String()
	}
	fmt.Fprintf(&sb, "RSI-триггер: бар -%d (RSI %.1f -> %.1f)\n", i-trig, rsi[trig-1], rsi[trig])

	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	fmt.Fprintf(&sb, "ATR(%d): %.4f\n", s.p.ATRPeriod, atr)
	if atr <= 0 {
		return sb.String()
	}
	stop := md.Lows[trig] - s.p.StopBufferATR*atr
	fmt.Fprintf(&sb, "стоп %.4f (лоу свечи кросса %.4f - %.2f×ATR); уровень удержан? %v\n",
		stop, md.Lows[trig], s.p.StopBufferATR, s.triggerAlive(md, trig, n, stop))

	risk := md.Closes[i] - stop
	fmt.Fprintf(&sb, "риск %.4f в границах [%.4f..%.4f]? %v; тейк %.4f (RR=%.2f)\n",
		risk, s.p.MinRiskATR*atr, s.p.MaxRiskATR*atr,
		risk >= s.p.MinRiskATR*atr && risk <= s.p.MaxRiskATR*atr,
		md.Closes[i]+s.p.RR*risk, s.p.RR)

	if s.p.EnableStochExit == 1 {
		fmt.Fprintf(&sb, "выход по стохастику(%d,%d) на этом баре? %v\n", s.p.StochK, s.p.StochD, s.stochExit(md))
	}
	fmt.Fprintf(&sb, "конец дня? %v\n", s.isDayEnd(s.barTime(md)))
	return sb.String()
}

// stochExit reports whether %K left the overbought zone downward on the current bar.
// StochasticSeries is right-aligned and shorter than the price series, so the two latest
// values are read from the tail — never by bar index.
func (s *Strategy) stochExit(md strategy.MarketData) bool {
	ks, _ := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochK, s.p.StochD)
	if len(ks) < 2 {
		return false
	}
	prev, now := ks[len(ks)-2], ks[len(ks)-1]
	return prev >= s.p.StochOverbought && now < s.p.StochOverbought
}
