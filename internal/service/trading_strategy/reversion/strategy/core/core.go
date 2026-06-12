// Package core implements a long-only mean-reversion strategy on the daily timeframe,
// driven by the agreement of two oscillators: RSI and the Stochastic %D line. It buys
// when one oscillator is already inside its oversold zone and the other crosses into it,
// and exits (XOVER) when one is already overbought and the other crosses up into the
// overbought zone. The protective ATR stop is frozen at entry and checked first. An
// optional trend filter restricts buys to a confirmed uptrend. The decision logic is pure
// and ticker-agnostic; per-share packages supply ticker + Params. Run with `-interval Day1`.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	UseTrend        int     // 1 = require uptrend before buying; 0 = ignore trend
	FastEMA         int     // fast regime EMA (e.g. 50)
	SlowEMA         int     // slow regime EMA + price floor (e.g. 200)
	RSIPeriod       int     // RSI length; required (>0)
	RSIOversold     float64 // RSI oversold zone (entry side)
	RSIOverbought   float64 // RSI overbought zone (exit side)
	StochKPeriod    int     // Stochastic %K lookback; required (>0)
	StochDSmooth    int     // Stochastic %D smoothing; required (>0); 1 = raw %K
	StochOversold   float64 // Stochastic oversold zone (entry side)
	StochOverbought float64 // Stochastic overbought zone (exit side)
	ATRPeriod       int     // ATR length for the stop
	ATRMult         float64 // stop = entry - ATRMult*ATR; must be > 0
}

// Strategy trades a single instrument with the dual-confirmation rules. Ticker-agnostic
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
		s.p.StochKPeriod + s.p.StochDSmooth + 1,
		s.p.ATRPeriod + 1,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core. stochNow/Prev
// are the %D line (smoothed %K).
type decideInput struct {
	price     float64
	atr       float64
	emaFast   float64
	emaSlow   float64
	rsiNow    float64
	rsiPrev   float64
	stochNow  float64
	stochPrev float64
	barLow    float64
	pos       *strategy.Position
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

	var stochNow, stochPrev float64
	if s.p.StochKPeriod > 0 && s.p.StochDSmooth > 0 {
		if _, d := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochKPeriod, s.p.StochDSmooth); len(d) >= 2 {
			stochNow, stochPrev = d[len(d)-1], d[len(d)-2]
		}
	}

	var barLow float64
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	return decideInput{
		price:     md.Price,
		atr:       atr,
		emaFast:   emaFast,
		emaSlow:   emaSlow,
		rsiNow:    rsiNow,
		rsiPrev:   rsiPrev,
		stochNow:  stochNow,
		stochPrev: stochPrev,
		barLow:    barLow,
		pos:       md.Position,
	}
}

// crossUp reports an up-cross of level: prev at/below, now above.
func crossUp(prev, now, level float64) bool { return prev <= level && now > level }

// crossDown reports a down-cross of level: prev at/above, now below.
func crossDown(prev, now, level float64) bool { return prev >= level && now < level }

// indicatorsReady reports that both oscillators are configured (valid readings possible).
func (s *Strategy) indicatorsReady() bool {
	return s.p.RSIPeriod > 0 && s.p.StochKPeriod > 0 && s.p.StochDSmooth > 0
}

// entryFired reports the dual oversold confirmation: one oscillator crosses DOWN into its
// oversold zone while the other is already inside its oversold zone. Simultaneous entry
// (both cross the same bar) satisfies this because the "already inside" test reads now.
func (s *Strategy) entryFired(in decideInput) bool {
	if !s.indicatorsReady() {
		return false
	}
	rsiCrossIn := crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	stochCrossIn := crossDown(in.stochPrev, in.stochNow, s.p.StochOversold)
	rsiIn := in.rsiNow < s.p.RSIOversold
	stochIn := in.stochNow < s.p.StochOversold
	return (rsiCrossIn && stochIn) || (stochCrossIn && rsiIn)
}

// exitFired reports the dual overbought confirmation: one oscillator crosses UP into its
// overbought zone while the other is already above its overbought zone.
func (s *Strategy) exitFired(in decideInput) bool {
	if !s.indicatorsReady() {
		return false
	}
	rsiCrossUp := crossUp(in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	stochCrossUp := crossUp(in.stochPrev, in.stochNow, s.p.StochOverbought)
	rsiHigh := in.rsiNow > s.p.RSIOverbought
	stochHigh := in.stochNow > s.p.StochOverbought
	return (rsiCrossUp && stochHigh) || (stochCrossUp && rsiHigh)
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
	// 2. Dual oversold confirmation.
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

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput, stop, risk float64) string {
	trend := "выкл"
	if s.p.UseTrend == 1 {
		trend = fmt.Sprintf("EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}
	return fmt.Sprintf(
		"Тренд: %s; двойное подтверждение перепроданности: RSI(%d) %.2f→%.2f (зона <%.0f) + Stoch%%D(%d,%d) %.2f→%.2f (зона <%.0f); SL=%.4f (−%.2g×ATR %.4f, риск %.4f)",
		trend,
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold,
		s.p.StochKPeriod, s.p.StochDSmooth, in.stochPrev, in.stochNow, s.p.StochOversold,
		stop, s.p.ATRMult, in.atr, risk,
	)
}

// manage handles an open long: the frozen ATR stop first, then the dual overbought exit.
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
		sig.Kind, sig.Reason = model.SignalSell, "XOVER"
		sig.ExitReason = fmt.Sprintf(
			"XOVER: RSI %.2f→%.2f (зона >%.0f) + Stoch%%D %.2f→%.2f (зона >%.0f) — двойное подтверждение перекупленности",
			in.rsiPrev, in.rsiNow, s.p.RSIOverbought, in.stochPrev, in.stochNow, s.p.StochOverbought)
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

	// 2. Dual oversold confirmation.
	if !s.entryFired(in) {
		return block("Двойное подтверждение: нет (RSI(%d) %.2f→%.2f зона<%.0f; Stoch%%D %.2f→%.2f зона<%.0f) — нужен кросс одного в зону при другом уже в зоне",
			s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold, in.stochPrev, in.stochNow, s.p.StochOversold)
	}
	pass("Двойное подтверждение: RSI(%d) %.2f→%.2f + Stoch%%D %.2f→%.2f в зоне перепроданности",
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, in.stochPrev, in.stochNow)

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
