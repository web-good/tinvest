// Package core implements a long-only hourly mean-reversion strategy. It buys sharp
// RSI drawdowns inside a confirmed uptrend (fast EMA above slow EMA, and price above
// the slow EMA) and exits on the bounce (RSI crossing up through an overbought level),
// a hard percentage stop frozen at entry, or a time-stop when the bounce never comes.
// Volume must be above its recent average; an optional Stochastic oversold gate can be
// switched on. The decision logic is pure and ticker-agnostic; per-share packages
// supply ticker + Params.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// EntryMode values for Params.EntryMode.
const (
	entryConfirmed = 0 // RSI crosses up through RSIOversold (bounce confirmed)
	entryKnife     = 1 // RSI crosses down through RSIOversold (catch the falling knife)
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	FastEMA       int     // fast regime EMA (e.g. 50)
	SlowEMA       int     // slow regime EMA + price floor (e.g. 200)
	RSIPeriod     int     // RSI length (e.g. 6); required (>0)
	RSIOversold   float64 // dip-trigger level (e.g. 40)
	RSIOverbought float64 // exit level (e.g. 70)
	EntryMode     int     // 0 = confirmed up-cross, 1 = knife down-cross
	VolLookback   int     // SMA window for the volume baseline
	VolMultiplier float64 // last volume must exceed VolMultiplier*SMA(volume)
	UseStoch      int     // 1 = require Stochastic oversold confirmation; 0 = skip
	StochPeriod   int     // %K lookback
	StochSmooth   int     // %D smoothing of %K
	StochOversold float64 // %K oversold threshold (e.g. 20)
	StopLossPct   float64 // hard stop = entry*(1-StopLossPct); must be > 0
	MaxHoldBars   int     // time-stop bar count; <= 0 disables
	ATRPeriod     int     // ATR length — display only, never gates logic
}

// Strategy trades a single instrument with the mean-reversion rules. Ticker-agnostic.
// It carries barsInPosition as mutable state in the impure shell; the pure decide()
// core stays a function of its input. Not safe for concurrent use; the backtest and
// live runners drive Decide sequentially, one bar at a time.
type Strategy struct {
	ticker         string
	p              Params
	barsInPosition int // bars elapsed since entry; reset to 0 while flat
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
		s.p.VolLookback + 1,
		s.p.ATRPeriod + 1,
		s.p.StochPeriod + s.p.StochSmooth,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core.
type decideInput struct {
	price      float64
	atr        float64 // display only
	emaFast    float64
	emaSlow    float64
	rsiNow     float64
	rsiPrev    float64
	stochK     float64 // computed only when UseStoch == 1
	stochValid bool    // true only when there is enough history for a real %K
	volumeOK   bool
	barLow     float64
	pos        *strategy.Position
	barsInPos  int
}

// Decide computes every indicator from md, advances the position-age counter, and
// delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position != nil {
		s.barsInPosition++
	} else {
		s.barsInPosition = 0
	}
	in := s.buildInput(md)
	in.barsInPos = s.barsInPosition
	sig := s.decide(in)
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

	stochK, stochValid := 0.0, false
	if s.p.UseStoch == 1 && s.p.StochPeriod > 0 && len(md.Closes) >= s.p.StochPeriod {
		stochK, _ = indicators.Stochastic(md.Highs, md.Lows, md.Closes, s.p.StochPeriod, s.p.StochSmooth)
		stochValid = true
	}

	var barLow float64
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	return decideInput{
		price:      md.Price,
		atr:        atr,
		emaFast:    emaFast,
		emaSlow:    emaSlow,
		rsiNow:     rsiNow,
		rsiPrev:    rsiPrev,
		stochK:     stochK,
		stochValid: stochValid,
		volumeOK:   indicators.VolumeConfirmed(md.Volumes, s.p.VolLookback, s.p.VolMultiplier),
		barLow:     barLow,
		pos:        md.Position,
	}
}

// dipFired reports whether the RSI dip trigger fires on this bar, honouring EntryMode.
// confirmed: RSI crosses up through RSIOversold (the up-cross implies a prior dip).
// knife: RSI crosses down through RSIOversold.
func (s *Strategy) dipFired(in decideInput) bool {
	if s.p.RSIPeriod <= 0 {
		return false
	}
	if s.p.EntryMode == entryKnife {
		return in.rsiPrev >= s.p.RSIOversold && in.rsiNow < s.p.RSIOversold
	}
	return in.rsiPrev <= s.p.RSIOversold && in.rsiNow > s.p.RSIOversold
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

	// 1. Regime: only buy dips inside a confirmed uptrend.
	if !uptrend(in) {
		return sig
	}
	// 2. Dip trigger.
	if !s.dipFired(in) {
		return sig
	}
	// 3. Volume.
	if !in.volumeOK {
		return sig
	}
	// 4. Optional Stochastic oversold confirmation. A 0 from insufficient history is
	// NOT oversold — require a valid reading so UseStoch never silently no-ops during
	// the indicator's warm-up window.
	if s.p.UseStoch == 1 && (!in.stochValid || in.stochK >= s.p.StochOversold) {
		return sig
	}
	// 5. Protective-stop sanity: a hard stop is mandatory.
	if s.p.StopLossPct <= 0 {
		return sig
	}
	stop := in.price * (1 - s.p.StopLossPct)
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
	mode := "кросс вверх через"
	if s.p.EntryMode == entryKnife {
		mode = "кросс вниз через"
	}
	stoch := "выкл"
	if s.p.UseStoch == 1 {
		stoch = fmt.Sprintf("%%K %.1f < %.0f", in.stochK, s.p.StochOversold)
	}
	return fmt.Sprintf(
		"Тренд↑ (EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d); RSI(%d) %s %.0f (%.2f→%.2f); объём > %.2g×ср(%d); стохастик: %s; SL=%.4f (−%.2g%%, −%.4f)",
		s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA,
		s.p.RSIPeriod, mode, s.p.RSIOversold, in.rsiPrev, in.rsiNow,
		s.p.VolMultiplier, s.p.VolLookback,
		stoch,
		stop, s.p.StopLossPct*100, risk,
	)
}

// manage handles an open long: frozen hard stop, a time-stop, or an RSI-overbought
// exit. Protective stops are checked first so the worst case for the position wins
// ties on a bar.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	hardSL := in.pos.StopLoss
	sig.StopLoss = hardSL
	sig.RSI = in.rsiNow

	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (зафиксирован на входе)", in.barLow, hardSL)
	case s.p.MaxHoldBars > 0 && in.barsInPos >= s.p.MaxHoldBars:
		sig.Kind, sig.Reason = model.SignalSell, "TIME"
		sig.ExitReason = fmt.Sprintf("TIME: %d бар(ов) в позиции ≥ %d — отскок не пришёл", in.barsInPos, s.p.MaxHoldBars)
	case s.p.RSIPeriod > 0 && in.rsiPrev < s.p.RSIOverbought && in.rsiNow >= s.p.RSIOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: %.2f → %.2f, пересёк %.2g вверх (отскок завершён)", in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	}
	return sig
}

// Explain re-runs the entry gates over md and reports each gate's value and verdict
// (✓ pass / ✗ block) in entry order, stopping at the first blocker — the same
// short-circuit order decide uses. Diagnostic only; never part of the trading path;
// never mutates barsInPosition.
func (s *Strategy) Explain(md strategy.MarketData) string {
	in := s.buildInput(md)
	in.barsInPos = s.barsInPosition

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

	// 1. Regime.
	if !uptrend(in) {
		return block("Тренд: нужно EMA%d > EMA%d и close > EMA%d (EMA%d=%.4f, EMA%d=%.4f, close=%.4f)",
			s.p.FastEMA, s.p.SlowEMA, s.p.SlowEMA, s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price)
	}
	pass("Тренд↑: EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d", s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)

	// 2. Dip trigger.
	mode := "кросс вверх через"
	if s.p.EntryMode == entryKnife {
		mode = "кросс вниз через"
	}
	if !s.dipFired(in) {
		return block("RSI(%d): нет события (%s %.0f), %.2f→%.2f", s.p.RSIPeriod, mode, s.p.RSIOversold, in.rsiPrev, in.rsiNow)
	}
	pass("RSI(%d): сработал триггер просадки (%s %.0f, %.2f→%.2f)", s.p.RSIPeriod, mode, s.p.RSIOversold, in.rsiPrev, in.rsiNow)

	// 3. Volume.
	if !in.volumeOK {
		return block("Объём: ниже %.2g×ср(%d)", s.p.VolMultiplier, s.p.VolLookback)
	}
	pass("Объём: выше %.2g×ср(%d)", s.p.VolMultiplier, s.p.VolLookback)

	// 4. Optional Stochastic. Mirror decide: a 0 from insufficient history is not oversold.
	if s.p.UseStoch == 1 {
		if !in.stochValid {
			return block("Стохастик: недостаточно истории для %%K (нужно ≥ %d баров)", s.p.StochPeriod)
		}
		if in.stochK >= s.p.StochOversold {
			return block("Стохастик: %%K %.1f ≥ %.0f (не в перепроданности)", in.stochK, s.p.StochOversold)
		}
		pass("Стохастик: %%K %.1f < %.0f", in.stochK, s.p.StochOversold)
	}

	// 5. Protective stop.
	if s.p.StopLossPct <= 0 {
		return block("Стоп: StopLossPct=%.2g ≤ 0 — защита не задана", s.p.StopLossPct)
	}
	stop := in.price * (1 - s.p.StopLossPct)
	pass("Стоп: SL=%.4f (−%.2g%%)", stop, s.p.StopLossPct*100)

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
