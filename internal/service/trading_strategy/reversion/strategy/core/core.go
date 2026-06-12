// Package core implements a long-only mean-reversion strategy driven purely by
// RSI on the daily timeframe. It buys when RSI crosses the oversold zone and exits
// when RSI crosses the overbought zone; the exact moment (entering vs exiting each
// zone) is configurable per side. An optional trend filter restricts buys to a
// confirmed uptrend. The protective stop is a daily-ATR stop frozen at entry. The
// decision logic is pure and ticker-agnostic; per-share packages supply ticker +
// Params. Run it with `-interval Day1`.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Trigger modes for EntryMode (oversold zone) and ExitMode (overbought zone).
// The semantics are shared: 0 fires when price enters the zone, 1 when it exits.
const (
	triggerEnterZone = 0 // RSI crosses into the zone
	triggerExitZone  = 1 // RSI crosses out of the zone
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	UseTrend      int     // 1 = require uptrend before buying; 0 = ignore trend
	FastEMA       int     // fast regime EMA (e.g. 50)
	SlowEMA       int     // slow regime EMA + price floor (e.g. 200)
	RSIPeriod     int     // RSI length; required (>0)
	RSIOversold   float64 // oversold zone boundary (entry side)
	RSIOverbought float64 // overbought zone boundary (exit side)
	EntryMode     int     // 0 = buy when RSI enters oversold; 1 = when it exits
	ExitMode      int     // 0 = sell when RSI enters overbought; 1 = when it exits
	ATRPeriod     int     // ATR length for the stop
	ATRMult       float64 // stop = entry - ATRMult*ATR; must be > 0
}

// Strategy trades a single instrument with the mean-reversion rules. Ticker-agnostic
// and pure: decide() is a function of its input. Not safe for concurrent use.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the reversion strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to feed the hungriest consumer.
func (s *Strategy) Lookback() int {
	m := s.p.SlowEMA
	for _, c := range []int{
		s.p.FastEMA,
		s.p.RSIPeriod + 1,
		s.p.ATRPeriod + 1,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core.
type decideInput struct {
	price   float64
	atr     float64
	emaFast float64
	emaSlow float64
	rsiNow  float64
	rsiPrev float64
	barLow  float64
	pos     *strategy.Position
}

// Decide computes every indicator from md and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := s.decide(s.buildInput(md))
	sig.Ticker = s.ticker
	return sig
}

// buildInput computes every indicator from md and packs them for the pure core.
func (s *Strategy) buildInput(md strategy.MarketData) decideInput {
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)

	emaFast, emaSlow := 0.0, 0.0
	if e := ema.Compute(md.Closes, s.p.FastEMA); len(e) > 0 {
		emaFast = e[len(e)-1]
	}
	if e := ema.Compute(md.Closes, s.p.SlowEMA); len(e) > 0 {
		emaSlow = e[len(e)-1]
	}

	var rsiNow, rsiPrev float64
	if s.p.RSIPeriod > 0 {
		if r := indicators.RSISeries(md.Closes, s.p.RSIPeriod); len(r) >= 2 {
			rsiNow, rsiPrev = r[len(r)-1], r[len(r)-2]
		}
	}

	var barLow float64
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	return decideInput{
		price:   md.Price,
		atr:     atr,
		emaFast: emaFast,
		emaSlow: emaSlow,
		rsiNow:  rsiNow,
		rsiPrev: rsiPrev,
		barLow:  barLow,
		pos:     md.Position,
	}
}

// crossUp reports an up-cross of level: prev at/below, now above.
func crossUp(prev, now, level float64) bool { return prev <= level && now > level }

// crossDown reports a down-cross of level: prev at/above, now below.
func crossDown(prev, now, level float64) bool { return prev >= level && now < level }

// entryFired reports whether the RSI entry trigger fires, honouring EntryMode.
// enter zone: RSI crosses DOWN through oversold. exit zone: crosses UP through it.
func (s *Strategy) entryFired(in decideInput) bool {
	if s.p.RSIPeriod <= 0 {
		return false
	}
	if s.p.EntryMode == triggerEnterZone {
		return crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	}
	return crossUp(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
}

// exitFired reports whether the RSI exit trigger fires, honouring ExitMode.
// enter zone: RSI crosses UP through overbought. exit zone: crosses DOWN through it.
func (s *Strategy) exitFired(in decideInput) bool {
	if s.p.RSIPeriod <= 0 {
		return false
	}
	if s.p.ExitMode == triggerEnterZone {
		return crossUp(in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	}
	return crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
}

// uptrend reports the regime gate: fast EMA above slow EMA and price above the slow EMA.
func uptrend(in decideInput) bool {
	return in.emaFast > in.emaSlow && in.emaSlow > 0 && in.price > in.emaSlow
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price}

	if in.pos != nil {
		return s.manage(in, sig)
	}

	// 1. Optional trend filter.
	if s.p.UseTrend == 1 && !uptrend(in) {
		return sig
	}
	// 2. RSI entry trigger.
	if !s.entryFired(in) {
		return sig
	}
	// 3. ATR stop is mandatory and must size a positive risk.
	if s.p.ATRMult <= 0 || in.atr <= 0 {
		return sig
	}
	stop := in.price - s.p.ATRMult*in.atr
	risk := in.price - stop
	if risk <= 0 {
		return sig
	}

	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = in.atr
	sig.RSI = in.rsiNow
	sig.EntryReason = s.entryReason(in, stop, risk)
	return sig
}

// zoneWord renders the trigger mode for a zone in human terms.
func zoneWord(mode int) string {
	if mode == triggerEnterZone {
		return "вход в зону"
	}
	return "выход из зоны"
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput, stop, risk float64) string {
	trend := "выкл"
	if s.p.UseTrend == 1 {
		trend = fmt.Sprintf("EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}
	return fmt.Sprintf(
		"Тренд: %s; RSI(%d) %s перепроданности %.0f (%.2f→%.2f); SL=%.4f (−%.2g×ATR %.4f, риск %.4f)",
		trend,
		s.p.RSIPeriod, zoneWord(s.p.EntryMode), s.p.RSIOversold, in.rsiPrev, in.rsiNow,
		stop, s.p.ATRMult, in.atr, risk,
	)
}

// manage handles an open long: the frozen ATR stop first, then the RSI exit trigger.
// Protective stops are checked first so the worst case for the position wins ties.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	hardSL := in.pos.StopLoss
	sig.StopLoss = hardSL
	sig.RSI = in.rsiNow

	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (зафиксирован на входе)", in.barLow, hardSL)
	case s.exitFired(in):
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: %.2f → %.2f, %s перекупленности %.0f", in.rsiPrev, in.rsiNow, zoneWord(s.p.ExitMode), s.p.RSIOverbought)
	}
	return sig
}

// Explain re-runs the entry gates over md and reports each gate's value and verdict
// (✓ pass / ✗ block) in entry order, stopping at the first blocker. Diagnostic only.
func (s *Strategy) Explain(md strategy.MarketData) string {
	in := s.buildInput(md)

	if in.pos != nil {
		return "позиция уже открыта — вход не рассматривается"
	}

	var b strings.Builder
	pass := func(format string, args ...any) { fmt.Fprintf(&b, "✓ "+format+"\n", args...) }
	block := func(format string, args ...any) string {
		fmt.Fprintf(&b, "✗ "+format+"\n", args...)
		fmt.Fprintf(&b, "→ ВХОДА НЕТ: заблокировал этот фильтр")
		return b.String()
	}

	// 1. Optional trend filter.
	if s.p.UseTrend == 1 {
		if !uptrend(in) {
			return block("Тренд: нужно EMA%d > EMA%d и close > EMA%d (EMA%d=%.4f, EMA%d=%.4f, close=%.4f)",
				s.p.FastEMA, s.p.SlowEMA, s.p.SlowEMA, s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price)
		}
		pass("Тренд↑: EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d", s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}

	// 2. RSI entry trigger.
	if !s.entryFired(in) {
		return block("RSI(%d): нет события (%s перепроданности %.0f), %.2f→%.2f",
			s.p.RSIPeriod, zoneWord(s.p.EntryMode), s.p.RSIOversold, in.rsiPrev, in.rsiNow)
	}
	pass("RSI(%d): %s перепроданности %.0f (%.2f→%.2f)", s.p.RSIPeriod, zoneWord(s.p.EntryMode), s.p.RSIOversold, in.rsiPrev, in.rsiNow)

	// 3. ATR stop.
	if s.p.ATRMult <= 0 {
		return block("Стоп: ATRMult=%.2g ≤ 0 — защита не задана", s.p.ATRMult)
	}
	if in.atr <= 0 {
		return block("Стоп: ATR=%.4f ≤ 0 — нельзя рассчитать стоп", in.atr)
	}
	stop := in.price - s.p.ATRMult*in.atr
	pass("Стоп: SL=%.4f (−%.2g×ATR %.4f)", stop, s.p.ATRMult, in.atr)

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
