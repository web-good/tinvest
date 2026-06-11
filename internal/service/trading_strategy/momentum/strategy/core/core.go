// Package core implements a long-only hourly trend-momentum strategy. It enters on
// a confluence of two independent bullish signals — a MACD bullish cross (optionally
// only below zero) and an RSI cross up through a configurable level — that occur
// within SignalValidBars bars of each other, while price is above a long EMA
// (uptrend) and volume is above its recent average. Exits on a frozen structural ATR
// stop, a fixed reward-multiple take-profit, an optional chandelier trail, an optional
// bearish MACD cross, or an RSI overbought cross-down. The decision logic is pure and
// ticker-agnostic; per-share packages supply ticker + Params.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// signalSaturate seeds and caps the per-signal "bars since the event" counters so
// they never overflow and never pair before their first occurrence: a fresh
// strategy starts with both signals "infinitely old", so no confluence can fire
// until each signal actually happens.
const signalSaturate = 1 << 30

// adxPeriod / adxThreshold define the regime gate: entry is allowed only when the
// hourly ADX(adxPeriod) is at least adxThreshold, i.e. the instrument is trending
// rather than ranging. The threshold is a fixed in-package constant — deliberately
// NOT a Params field — so grid calibration can never tune it per ticker. This makes
// the basket walk-forward an honest test of whether filtering chop yields real edge.
// Ablation: set adxThreshold to 0 to disable the gate.
const (
	adxPeriod    = 14
	adxThreshold = 25.0
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1)
// so reflection grid calibration can sweep them.
type Params struct {
	EMAPeriod         int     // long EMA trend filter (hourly)
	MACDFast          int     // MACD fast EMA period
	MACDSlow          int     // MACD slow EMA period
	MACDSignal        int     // MACD signal EMA period
	MACDBelowZeroOnly int     // 1 = the MACD cross counts only when macd<0; 0 = any bullish cross
	VolLookback       int     // SMA window for the volume baseline
	VolMultiplier     float64 // last volume must exceed VolMultiplier*SMA(volume)
	ATRPeriod         int     // hourly ATR period (stops)
	SwingLowWindow    int     // bars scanned for the structural low anchoring the stop
	SLMult            float64 // stop = swingLow - SLMult*ATR
	TakeProfitRR      float64 // TP = entry + TakeProfitRR*(entry-stop)
	MinRR             float64 // reject entry if (TP-price) < MinRR*risk; <=0 disables
	UseTrail          int     // 1 = trail instead of fixed TP
	UseMACDExit       int     // 1 = exit when MACD crosses below its signal line
	RSIPeriod         int     // RSI length; required (>0) — feeds both the entry cross and the overbought exit
	RSICrossLevel     float64 // entry RSI level: the cross up through this level arms the RSI signal
	RSIOverbought     float64 // RSI overbought line for the exit (e.g. 70)
	TrailMult         float64 // chandelier = recentHigh(ChandelierWindow) - TrailMult*ATR
	ChandelierWindow  int     // window for the chandelier high
	TrailArmATR       float64 // trail arms after MaxFavorable >= entry + TrailArmATR*EntryATR
	SignalValidBars   int     // max gap in bars between the MACD cross and the RSI cross; 0 = same bar
}

// Strategy trades a single instrument with the momentum rules. Ticker-agnostic.
// It carries the two "bars since the event" signal counters as mutable state in
// the impure shell; the pure decide() core stays a function of its input. Not safe
// for concurrent use; the backtest and live runners drive Decide sequentially, one
// bar at a time.
type Strategy struct {
	ticker             string
	p                  Params
	barsSinceMACDCross int // bars since the last qualifying MACD bullish cross
	barsSinceRSICross  int // bars since RSI last crossed up through RSICrossLevel
}

// NewWithParams returns the momentum strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p, barsSinceMACDCross: signalSaturate, barsSinceRSICross: signalSaturate}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to feed the hungriest consumer.
func (s *Strategy) Lookback() int {
	m := s.p.EMAPeriod
	for _, c := range []int{
		s.p.MACDSlow + s.p.MACDSignal,
		s.p.VolLookback + 1,
		s.p.ATRPeriod + 1,
		s.p.SwingLowWindow,
		s.p.ChandelierWindow,
		s.p.RSIPeriod + 1,
		2*adxPeriod + 1,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core.
type decideInput struct {
	price         float64
	atr           float64
	emaTrend      float64
	adx           float64 // hourly ADX(adxPeriod); the regime gate requires adx >= adxThreshold
	macdNow       float64
	macdFired     bool    // a qualifying MACD bullish cross happened on this bar (honours MACDBelowZeroOnly)
	macdCrossDown bool    // MACD line just crossed below its signal line (bearish exit)
	rsiNow        float64 // latest RSI value (0 when history is insufficient)
	rsiPrev       float64 // previous-bar RSI value
	rsiFired      bool    // RSI crossed up through RSICrossLevel on this bar
	volumeOK      bool
	barHigh       float64
	barLow        float64
	recentLow     float64
	recentHigh    float64
	// signal ages copied from the shell (already advanced for this bar).
	barsSinceMACDCross int
	barsSinceRSICross  int
	pos                *strategy.Position
}

// Decide computes every indicator from md, packs them, advances the signal
// counters, and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	in := s.buildInput(md)
	s.advanceSignals(in)
	in.barsSinceMACDCross = s.barsSinceMACDCross
	in.barsSinceRSICross = s.barsSinceRSICross
	sig := s.decide(in)
	sig.Ticker = s.ticker
	return sig
}

// advanceSignals updates the two "bars since the event" counters in the impure
// shell. A counter resets to 0 on the bar its signal fires and otherwise increments
// (saturating). Market events do not depend on position state, so this runs every
// bar regardless of whether a position is open. Called once per bar from Decide,
// before decide reads the counters. Explain never mutates this state.
func (s *Strategy) advanceSignals(in decideInput) {
	if in.macdFired {
		s.barsSinceMACDCross = 0
	} else if s.barsSinceMACDCross < signalSaturate {
		s.barsSinceMACDCross++
	}
	if in.rsiFired {
		s.barsSinceRSICross = 0
	} else if s.barsSinceRSICross < signalSaturate {
		s.barsSinceRSICross++
	}
}

// buildInput computes every indicator from md and packs them for the pure core.
func (s *Strategy) buildInput(md strategy.MarketData) decideInput {
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	adx, _, _ := indicators.ADX(md.Highs, md.Lows, md.Closes, adxPeriod)

	emaTrend := 0.0
	if e := ema.Compute(md.Closes, s.p.EMAPeriod); len(e) > 0 {
		emaTrend = e[len(e)-1]
	}

	macdNow, crossUp, crossDown := 0.0, false, false
	if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); len(m) >= 2 {
		prevDiff := m[len(m)-2] - sg[len(sg)-2]
		currDiff := m[len(m)-1] - sg[len(sg)-1]
		macdNow = m[len(m)-1]
		crossUp = prevDiff <= 0 && currDiff > 0
		crossDown = prevDiff >= 0 && currDiff < 0
	}
	macdFired := crossUp && (s.p.MACDBelowZeroOnly == 0 || macdNow < 0)

	var rsiNow, rsiPrev float64
	if s.p.RSIPeriod > 0 {
		if r := indicators.RSISeries(md.Closes, s.p.RSIPeriod); len(r) >= 2 {
			rsiNow, rsiPrev = r[len(r)-1], r[len(r)-2]
		}
	}
	rsiFired := s.p.RSIPeriod > 0 && rsiPrev <= s.p.RSICrossLevel && rsiNow > s.p.RSICrossLevel

	var barHigh, barLow float64
	if n := len(md.Highs); n > 0 {
		barHigh = md.Highs[n-1]
	}
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	return decideInput{
		price:         md.Price,
		atr:           atr,
		emaTrend:      emaTrend,
		adx:           adx,
		macdNow:       macdNow,
		macdFired:     macdFired,
		macdCrossDown: crossDown,
		rsiNow:        rsiNow,
		rsiPrev:       rsiPrev,
		rsiFired:      rsiFired,
		volumeOK:      indicators.VolumeConfirmed(md.Volumes, s.p.VolLookback, s.p.VolMultiplier),
		barHigh:       barHigh,
		barLow:        barLow,
		recentLow:     recentLow(md.Lows, s.p.SwingLowWindow),
		recentHigh:    recentHigh(md.Highs, s.p.ChandelierWindow),
		pos:           md.Position,
	}
}

// confluence reports whether the MACD↔RSI pairing fires on this bar: a fresh
// qualifying signal on the current bar whose partner fired no more than
// SignalValidBars bars ago. It is an edge trigger (requires a fresh event now), so
// it fires exactly once — on the bar of the second signal of the pair. macdAge/rsiAge
// fold a current-bar fire to age 0 so the test is correct whether or not the shell
// counters were advanced for this bar (decide passes advanced counters; Explain may
// run standalone).
func confluence(in decideInput, window int) bool {
	macdAge, rsiAge := in.barsSinceMACDCross, in.barsSinceRSICross
	if in.macdFired {
		macdAge = 0
	}
	if in.rsiFired {
		rsiAge = 0
	}
	return (in.macdFired && rsiAge <= window) || (in.rsiFired && macdAge <= window)
}

// Explain re-runs the entry gates over md and reports each gate's value and
// verdict (✓ pass / ✗ block) in entry order, stopping the chain at the first
// blocker — the same short-circuit order decide uses. Diagnostic only; never
// part of the trading path.
func (s *Strategy) Explain(md strategy.MarketData) string {
	in := s.buildInput(md)
	in.barsSinceMACDCross = s.barsSinceMACDCross
	in.barsSinceRSICross = s.barsSinceRSICross

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

	// Effective signal ages: a current-bar fire reads as age 0 even if Decide
	// hasn't advanced the shell counters for this bar.
	macdAge, rsiAge := in.barsSinceMACDCross, in.barsSinceRSICross
	if in.macdFired {
		macdAge = 0
	}
	if in.rsiFired {
		rsiAge = 0
	}

	// 1. MACD↔RSI confluence (the entry trigger).
	if !confluence(in, s.p.SignalValidBars) {
		return block("Confluence: нет свежей пары MACD↔RSI (MACD %d бар(ов) назад, RSI %d бар(ов) назад, окно %d)",
			macdAge, rsiAge, s.p.SignalValidBars)
	}
	pass("Confluence: MACD %d бар(ов) назад, RSI %d бар(ов) назад (окно %d)", macdAge, rsiAge, s.p.SignalValidBars)

	// 2. Uptrend.
	if !(in.emaTrend > 0 && in.price > in.emaTrend) {
		return block("Тренд: close %.4f ≤ EMA%d %.4f (нужно выше)", in.price, s.p.EMAPeriod, in.emaTrend)
	}
	pass("Тренд: close %.4f > EMA%d %.4f", in.price, s.p.EMAPeriod, in.emaTrend)

	// 3. Volume.
	if !in.volumeOK {
		return block("Объём: ниже %.2g×ср(%d) — подтверждения нет", s.p.VolMultiplier, s.p.VolLookback)
	}
	pass("Объём: выше %.2g×ср(%d)", s.p.VolMultiplier, s.p.VolLookback)

	// 4. Risk / RR sanity.
	stop := in.recentLow - s.p.SLMult*in.atr
	risk := in.price - stop
	if risk <= 0 {
		return block("Риск: SL=%.4f ≥ цены %.4f (risk ≤ 0)", stop, in.price)
	}
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.TakeProfitRR > 0 && s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return block("RR: цель %.4f даёт %.2gR < MinRR %.2g", target, (target-in.price)/risk, s.p.MinRR)
	}
	pass("Риск: SL=%.4f, TP=%.4f, %.2gR", stop, target, (target-in.price)/risk)

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price}

	if in.pos != nil {
		return s.manage(in, sig)
	}

	// Regime gate: only trade when the instrument is trending (ADX >= threshold),
	// never in chop. Fixed threshold, identical for every ticker, never calibrated.
	if in.adx < adxThreshold {
		return sig
	}

	// Entry trigger: a fresh MACD↔RSI confluence on this bar. Edge-triggered, so it
	// fires once on the bar of the second signal and does not re-fire while flat.
	if !confluence(in, s.p.SignalValidBars) {
		return sig
	}

	// Trend.
	if !(in.emaTrend > 0 && in.price > in.emaTrend) {
		return sig
	}

	// Volume.
	if !in.volumeOK {
		return sig
	}

	stop := in.recentLow - s.p.SLMult*in.atr
	risk := in.price - stop
	if risk <= 0 {
		return sig
	}
	target := in.price + s.p.TakeProfitRR*risk
	// MinRR only gates the fixed-TP reward. With no fixed TP (TakeProfitRR<=0) the
	// trade is managed by trail/MACD/RSI exits, so the RR filter does not apply.
	if s.p.TakeProfitRR > 0 && s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return sig
	}

	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = target
	sig.ATR = in.atr
	sig.EntryReason = s.entryReason(in, stop, target, risk)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput, stop, target, risk float64) string {
	zero := "над нулём"
	if in.macdNow < 0 {
		zero = "под нулём"
	}
	macdAge, rsiAge := in.barsSinceMACDCross, in.barsSinceRSICross
	if in.macdFired {
		macdAge = 0
	}
	if in.rsiFired {
		rsiAge = 0
	}
	return fmt.Sprintf(
		"Тренд↑ (close %.4f > EMA%d %.4f); MACD бычий кросс (%s, %.4f) %d бар(ов) назад; RSI пересёк %.0f↑ (%.2f→%.2f) %d бар(ов) назад, зазор ≤ %d; объём > %.2g×ср(%d); SL=%.4f (-%.4f); TP=%.4f (+%.4f, %.2gR)",
		in.price, s.p.EMAPeriod, in.emaTrend, zero, in.macdNow, macdAge,
		s.p.RSICrossLevel, in.rsiPrev, in.rsiNow, rsiAge, s.p.SignalValidBars,
		s.p.VolMultiplier, s.p.VolLookback,
		stop, risk, target, target-in.price, s.p.TakeProfitRR,
	)
}

// manage handles an open long: frozen hard stop, fixed take-profit (reconstructed
// from the frozen entry stop), or an optional armed chandelier trail.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	entry := in.pos.PurchasePrice
	hardSL := in.pos.StopLoss
	// TP is reconstructed from the frozen entry stop (Position carries no target):
	// risk and stop are both fixed at entry, so this is deterministic.
	risk := entry - hardSL
	tp := entry + s.p.TakeProfitRR*risk
	chandelier := in.recentHigh - s.p.TrailMult*in.atr
	trailArmed := s.p.TrailArmATR <= 0 || in.pos.MaxFavorablePrice >= entry+s.p.TrailArmATR*in.pos.EntryATR

	sig.StopLoss = hardSL
	if s.p.TakeProfitRR > 0 {
		sig.TakeProfit = tp
	}

	// Exit on the first trigger; protective/intrabar stops are checked first so the
	// worst case for the position wins ties on a bar. MACD/RSI are close-confirmed.
	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (зафиксирован на входе)", in.barLow, hardSL)
	case s.p.UseTrail == 1 && trailArmed && in.barLow <= chandelier:
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		sig.ExitReason = fmt.Sprintf("TRAIL: low %.4f ≤ шанделье %.4f (recentHigh %.4f − %.2g×ATR %.4f)",
			in.barLow, chandelier, in.recentHigh, s.p.TrailMult, in.atr)
	case s.p.TakeProfitRR > 0 && in.barHigh >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цель %.4f (%.2gR)", in.barHigh, tp, s.p.TakeProfitRR)
	case s.p.UseMACDExit == 1 && in.macdCrossDown:
		sig.Kind, sig.Reason = model.SignalSell, "MACD"
		sig.ExitReason = fmt.Sprintf("MACD: медвежий кросс сигнальной линии (MACD=%.4f)", in.macdNow)
	case s.p.RSIPeriod > 0 && in.rsiPrev > s.p.RSIOverbought && in.rsiNow <= s.p.RSIOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: %.2f → %.2f, пересёк границу %.2g сверху вниз",
			in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	}
	return sig
}

// recentLow returns the lowest low over the last window bars (all if fewer);
// a non-positive window is clamped to the last bar.
func recentLow(lows []float64, window int) float64 {
	n := len(lows)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	if start > n-1 {
		start = n - 1
	}
	l := lows[start]
	for i := start + 1; i < n; i++ {
		if lows[i] < l {
			l = lows[i]
		}
	}
	return l
}

// recentHigh returns the highest high over the last window bars (all if fewer);
// a non-positive window is clamped to the last bar.
func recentHigh(highs []float64, window int) float64 {
	n := len(highs)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	if start > n-1 {
		start = n - 1
	}
	h := highs[start]
	for i := start + 1; i < n; i++ {
		if highs[i] > h {
			h = highs[i]
		}
	}
	return h
}
