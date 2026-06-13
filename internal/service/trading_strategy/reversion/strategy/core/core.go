// Package core implements a long-only mean-reversion strategy on the daily timeframe,
// driven by the agreement of two oscillators: RSI and the Stochastic %D line. It buys
// when one oscillator is already inside its oversold zone and the other crosses into it.
// It exits an open long on one of three signals: RSI crossing the 50 line downward
// (primary momentum fade), RSI breaking back down through the oversold zone from above
// (failed-bounce / RSI-oversold breakdown), or a bearish EMA cross (FastEMA dropping
// below SlowEMA) as a regime-break backstop. There is no protective stop. An optional
// trend filter restricts buys to a confirmed uptrend. The decision logic is pure and
// ticker-agnostic; per-share packages supply ticker + Params. Run with `-interval Day1`.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// rsiExitLevel is the fixed RSI midline: an open long exits when RSI crosses it downward.
const rsiExitLevel = 50.0

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	UseTrend      int     // 1 = require uptrend before buying; 0 = ignore trend
	FastEMA       int     // fast regime EMA (e.g. 50); also the bearish-cross exit fast line
	SlowEMA       int     // slow regime EMA + price floor (e.g. 200); bearish-cross exit slow line
	RSIPeriod     int     // RSI length; required (>0)
	RSIOversold   float64 // RSI oversold zone (entry side)
	StochKPeriod  int     // Stochastic %K lookback; required (>0)
	StochDSmooth  int     // Stochastic %D smoothing; required (>0); 1 = raw %K
	StochOversold float64 // Stochastic oversold zone (entry side)
	UseATRStop    int     // 0 = RSIOS exit (RSI breaks oversold zone down); 1 = ATRSL exit (price below entry by the daily ATR)
	ATRPeriod     int     // daily ATR length; consulted only when UseATRStop=1
	StopATRMult   float64 // ATRSL distance: stop = PurchasePrice - StopATRMult*EntryATR (default 1.0)
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
	cands := []int{
		s.p.FastEMA,
		s.p.RSIPeriod + 1,
		s.p.StochKPeriod + s.p.StochDSmooth + 1,
	}
	if s.p.UseATRStop == 1 && s.p.ATRPeriod > 0 {
		cands = append(cands, s.p.ATRPeriod+1)
	}
	for _, c := range cands {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core. stochNow/Prev
// are the %D line (smoothed %K). emaFastPrev/emaSlowPrev are the previous-bar EMA values,
// needed to detect the bearish cross. rsiOK/stochOK report whether each oscillator produced
// a valid two-bar reading; when false the (now/prev) values are warm-up sentinels (0) and
// must NOT be treated as a real in-zone reading. emaOK reports whether both EMA readings are
// warmed (non-zero); the EMAX exit gates on it so the warm-up zero sentinel cannot fake a
// fresh cross — mirroring the rsiOK/stochOK discipline.
type decideInput struct {
	price       float64
	emaFast     float64
	emaFastPrev float64
	emaSlow     float64
	emaSlowPrev float64
	emaOK       bool
	rsiNow      float64
	rsiPrev     float64
	rsiOK       bool
	stochNow    float64
	stochPrev   float64
	stochOK     bool
	atr         float64 // daily ATR over the window (0 unless UseATRStop=1 and ATRPeriod>0); stamped onto sig.ATR at entry to freeze EntryATR
	pos         *strategy.Position
}

// Decide computes every indicator from md and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := s.decide(s.buildInput(md))
	sig.Ticker = s.ticker
	return sig
}

// buildInput computes every indicator from md and packs them for the pure core.
func (s *Strategy) buildInput(md strategy.MarketData) decideInput {
	emaFast, emaFastPrev, fastOK := lastTwoEMA(md.Closes, s.p.FastEMA)
	emaSlow, emaSlowPrev, slowOK := lastTwoEMA(md.Closes, s.p.SlowEMA)

	var rsiNow, rsiPrev float64
	rsiOK := false
	if s.p.RSIPeriod > 0 {
		if r := indicators.RSISeries(md.Closes, s.p.RSIPeriod); len(r) >= 2 {
			rsiNow, rsiPrev = r[len(r)-1], r[len(r)-2]
			rsiOK = true
		}
	}

	var stochNow, stochPrev float64
	stochOK := false
	if s.p.StochKPeriod > 0 && s.p.StochDSmooth > 0 {
		if _, d := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochKPeriod, s.p.StochDSmooth); len(d) >= 2 {
			stochNow, stochPrev = d[len(d)-1], d[len(d)-2]
			stochOK = true
		}
	}

	var atr float64
	if s.p.UseATRStop == 1 && s.p.ATRPeriod > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	}

	return decideInput{
		price:       md.Price,
		emaFast:     emaFast,
		emaFastPrev: emaFastPrev,
		emaSlow:     emaSlow,
		emaSlowPrev: emaSlowPrev,
		emaOK:       fastOK && slowOK,
		rsiNow:      rsiNow,
		rsiPrev:     rsiPrev,
		rsiOK:       rsiOK,
		stochNow:    stochNow,
		stochPrev:   stochPrev,
		stochOK:     stochOK,
		atr:         atr,
		pos:         md.Position,
	}
}

// lastTwoEMA returns the latest and previous EMA values for the period, plus ok=true only
// when BOTH are genuinely warmed. ema.Compute returns a slice the same length as closes
// with leading zeros before index period-1, so at the bar where the EMA first becomes
// valid the previous value is still the zero sentinel. Prices are positive, so a real EMA
// is never 0; a 0 reliably means "not warmed". ok=false (with prev still returned) lets the
// caller suppress a spurious cross at the warm-up boundary.
func lastTwoEMA(closes []float64, period int) (now, prev float64, ok bool) {
	e := ema.Compute(closes, period)
	if len(e) < 2 {
		return 0, 0, false
	}
	now, prev = e[len(e)-1], e[len(e)-2]
	return now, prev, now != 0 && prev != 0
}

// crossDown reports a down-cross of level: prev at/above, now below.
func crossDown(prev, now, level float64) bool { return prev >= level && now < level }

// indicatorsReady reports that both oscillators produced valid two-bar readings. A warm-up
// sentinel (rsiOK/stochOK false, values 0) must never count as an in-zone reading, or the
// dual confirmation silently degrades to a single-oscillator gate.
func indicatorsReady(in decideInput) bool {
	return in.rsiOK && in.stochOK
}

// entryFired reports the dual oversold confirmation: one oscillator crosses DOWN into its
// oversold zone while the other is already inside its oversold zone. Simultaneous entry
// (both cross the same bar) satisfies this because the "already inside" test reads now.
func (s *Strategy) entryFired(in decideInput) bool {
	if !indicatorsReady(in) {
		return false
	}
	rsiCrossIn := crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	stochCrossIn := crossDown(in.stochPrev, in.stochNow, s.p.StochOversold)
	rsiIn := in.rsiNow < s.p.RSIOversold
	stochIn := in.stochNow < s.p.StochOversold
	return (rsiCrossIn && stochIn) || (stochCrossIn && rsiIn)
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

	sig.Kind = model.SignalBuy
	sig.RSI = in.rsiNow
	sig.EntryReason = s.entryReason(in)
	sig.ATR = in.atr
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput) string {
	trend := "выкл"
	if s.p.UseTrend == 1 {
		trend = fmt.Sprintf("EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}
	return fmt.Sprintf(
		"Тренд: %s; двойное подтверждение перепроданности: RSI(%d) %.2f→%.2f (зона <%.0f) + Stoch%%D(%d,%d) %.2f→%.2f (зона <%.0f)",
		trend,
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold,
		s.p.StochKPeriod, s.p.StochDSmooth, in.stochPrev, in.stochNow, s.p.StochOversold,
	)
}

// manage handles an open long. There is no protective price stop other than the optional
// ATR stop below. It exits on one of three signals, evaluated in precedence order (all
// fills at close):
//   - RSI50: RSI crosses the 50 midline downward — primary momentum fade.
//   - middle branch, selected by UseATRStop:
//     UseATRStop==0 -> RSIOS: RSI breaks back down through the oversold zone from above
//     (failed-bounce breakdown); fires when RSI was at/above RSIOversold last bar and
//     is now below it.
//     UseATRStop==1 -> ATRSL: price has fallen to/below PurchasePrice - StopATRMult*EntryATR,
//     where EntryATR is the daily ATR frozen at entry. Guarded by EntryATR>0 and
//     StopATRMult>0 so it stays inert in live trading (EntryATR not persisted) and on
//     a misconfigured zero multiplier.
//   - EMAX: FastEMA drops below SlowEMA (bearish EMA cross) — regime-break backstop.
//
// When multiple signals fire on the same bar the first in the list wins; the fill price
// (close) is identical either way.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	sig.RSI = in.rsiNow

	switch {
	case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, rsiExitLevel):
		sig.Kind, sig.Reason = model.SignalSell, "RSI50"
		sig.ExitReason = fmt.Sprintf("RSI50: RSI %.2f→%.2f пересёк 50 сверху вниз", in.rsiPrev, in.rsiNow)
	case s.p.UseATRStop == 0 && in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold):
		sig.Kind, sig.Reason = model.SignalSell, "RSIOS"
		sig.ExitReason = fmt.Sprintf("RSIOS: RSI %.2f→%.2f пробил зону перепроданности %.0f сверху вниз",
			in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	case s.p.UseATRStop == 1 && in.pos.EntryATR > 0 && s.p.StopATRMult > 0 &&
		in.price <= in.pos.PurchasePrice-s.p.StopATRMult*in.pos.EntryATR:
		stop := in.pos.PurchasePrice - s.p.StopATRMult*in.pos.EntryATR
		sig.Kind, sig.Reason = model.SignalSell, "ATRSL"
		sig.ExitReason = fmt.Sprintf("ATRSL: цена %.4f ≤ вход %.4f − %.2g×ATR %.4f (порог %.4f)",
			in.price, in.pos.PurchasePrice, s.p.StopATRMult, in.pos.EntryATR, stop)
	case in.emaOK && crossDown(in.emaFastPrev-in.emaSlowPrev, in.emaFast-in.emaSlow, 0):
		sig.Kind, sig.Reason = model.SignalSell, "EMAX"
		sig.ExitReason = fmt.Sprintf("EMAX: FastEMA%d %.4f ушла под SlowEMA%d %.4f (медвежий кросс)",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow)
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

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
