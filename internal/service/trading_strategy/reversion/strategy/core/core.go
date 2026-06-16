// Package core implements a long-only mean-reversion strategy on the daily timeframe,
// driven by the agreement of two oscillators: RSI and the Stochastic %D line. It buys
// when one oscillator is already inside its oversold zone and the other crosses into it.
// It exits an open long on one of four signals: an overbought take-profit when both RSI
// and Stochastic %D are simultaneously in their overbought zones (OB, gated by UseOverbought);
// RSI crossing the 50 line downward (primary momentum fade); a middle exit selected by the
// UseATRStop flag — either RSI breaking back down through the oversold zone (RSIOS, failed
// bounce) or price falling below the ATR stop PurchasePrice − StopATRMult×EntryATR with
// EntryATR frozen at entry (ATRSL); and a bearish EMA cross (FastEMA below SlowEMA) as a
// regime-break backstop. There is no protective stop unless UseATRStop=1. An optional trend filter restricts buys to
// a confirmed uptrend. An optional volume filter (UseVolume) additionally blocks a buy when the
// entry bar's volume is below the average of the preceding VolAvgPeriod bars (weekend Sat/Sun
// bars excluded), scaled by VolMult. The decision logic is pure and
// ticker-agnostic; per-share packages supply ticker + Params. Run with `-interval Day1`.
package core

import (
	"fmt"
	"strings"
	"time"

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
	UseTrend        int     // 1 = require uptrend before buying; 0 = ignore trend
	FastEMA         int     // fast regime EMA (e.g. 50); also the bearish-cross exit fast line
	SlowEMA         int     // slow regime EMA + price floor (e.g. 200); bearish-cross exit slow line
	RSIPeriod       int     // RSI length; required (>0)
	RSIOversold     float64 // RSI oversold zone (entry side)
	StochKPeriod    int     // Stochastic %K lookback; required (>0)
	StochDSmooth    int     // Stochastic %D smoothing; required (>0); 1 = raw %K
	StochOversold   float64 // Stochastic oversold zone (entry side)
	UseATRStop      int     // 0 = RSIOS exit (RSI breaks oversold zone down); 1 = ATRSL exit (price below entry by the daily ATR)
	ATRPeriod       int     // daily ATR length; consulted only when UseATRStop=1
	StopATRMult     float64 // ATRSL distance: stop = PurchasePrice - StopATRMult*EntryATR (default 1.0)
	UseVolume       int     // 0 = no volume filter; 1 = block entries below the average bar volume
	VolAvgPeriod    int     // preceding-bar window for the average-volume baseline; consulted only when UseVolume=1
	VolMult         float64 // entry requires entryVolume >= avg*VolMult (default 1.0)
	UseOverbought   int     // 1 = exit when RSI and Stoch are simultaneously overbought; 0 = off
	RSIOverbought   float64 // RSI overbought zone for the OB exit (default 70); consulted only when UseOverbought=1
	StochOverbought float64 // Stoch %D overbought zone for the OB exit (default 80); consulted only when UseOverbought=1
	HTFTrendEMA     int     // EMA period on the 4H timeframe for the higher-timeframe trend filter; 0 = off
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
	if s.p.UseVolume == 1 && s.p.VolAvgPeriod > 0 {
		cands = append(cands, s.p.VolAvgPeriod+1)
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
	entryVol    float64 // entry (latest) bar's volume; 0 unless UseVolume=1 and a baseline was computed
	avgVol      float64 // average volume of the preceding VolAvgPeriod bars (weekends excluded); 0 unless gate active
	volOK       bool    // true when the volume baseline could be computed; false -> gate is skipped
	htfClose    float64 // last completed 4H close (0 unless HTFTrendEMA>0 and enough data)
	htfEMA      float64 // EMA(HTFCloses, HTFTrendEMA), latest value; 0 unless gate active
	htfOK       bool    // true when the 4H EMA is warmed; false -> higher-timeframe trend not confirmed
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

	var entryVol, avgVol float64
	volOK := false
	if s.p.UseVolume == 1 && s.p.VolAvgPeriod > 0 {
		if n := len(md.Volumes); n > 0 && md.Volumes[n-1] > 0 {
			if a, ok := averageVolumeExcludingWeekends(md.Volumes, md.Times, s.p.VolAvgPeriod); ok {
				entryVol = float64(md.Volumes[n-1])
				avgVol = a
				volOK = true
			}
		}
	}

	var htfClose, htfEMA float64
	htfOK := false
	if s.p.HTFTrendEMA > 0 && len(md.HTFCloses) >= s.p.HTFTrendEMA {
		if e := ema.Compute(md.HTFCloses, s.p.HTFTrendEMA); len(e) > 0 {
			// Prices are positive, so a real EMA is never 0; a 0 means "not warmed"
			// (same warm-up discipline as lastTwoEMA).
			if v := e[len(e)-1]; v != 0 {
				htfEMA = v
				htfClose = md.HTFCloses[len(md.HTFCloses)-1]
				htfOK = true
			}
		}
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
		entryVol:    entryVol,
		avgVol:      avgVol,
		volOK:       volOK,
		htfClose:    htfClose,
		htfEMA:      htfEMA,
		htfOK:       htfOK,
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

// mskLoc anchors weekend detection to the Moscow trading calendar (UTC fallback if the
// tz DB is absent), mirroring the backtest engine. Weekend trading sessions (Sat/Sun) on
// MOEX trade at much lower volume and are excluded from the average-volume baseline.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// isWeekend reports whether t falls on Saturday or Sunday in mskLoc.
func isWeekend(t time.Time) bool {
	wd := t.In(mskLoc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// averageVolumeExcludingWeekends averages the volumes of the `period` bars that PRECEDE
// the final (entry) bar of vols. The entry bar is never part of its own average. When
// times is supplied and index-aligned to vols, weekend bars (Sat/Sun MSK) are dropped;
// when times is empty or misaligned, weekend exclusion is skipped (all preceding bars
// count). Non-positive volumes are ignored. ok is false when no sample survives — the
// caller must then skip the gate (never block an entry on missing data).
func averageVolumeExcludingWeekends(vols []int64, times []time.Time, period int) (avg float64, ok bool) {
	n := len(vols)
	if n < 2 || period <= 0 {
		return 0, false
	}
	lo := n - 1 - period // window = the `period` bars before the entry bar: [lo, n-1)
	if lo < 0 {
		lo = 0
	}
	haveTimes := len(times) == n
	var sum float64
	var count int
	for j := lo; j < n-1; j++ {
		if haveTimes && isWeekend(times[j]) {
			continue
		}
		if vols[j] <= 0 {
			continue
		}
		sum += float64(vols[j])
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

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
	// 3. Optional volume filter: block a buy on a below-average-volume bar.
	if s.p.UseVolume == 1 && in.volOK && in.entryVol < in.avgVol*s.p.VolMult {
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
// ATR stop below. It exits on one of four signals, evaluated in precedence order (all
// fills at close):
//   - OB: RSI and Stochastic %D simultaneously in their overbought zones — take-profit
//     (gated by UseOverbought=1). Highest precedence.
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
	case s.p.UseOverbought == 1 && in.rsiOK && in.stochOK &&
		in.rsiNow >= s.p.RSIOverbought && in.stochNow >= s.p.StochOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "OB"
		sig.ExitReason = fmt.Sprintf("OB: RSI %.2f ≥ %.0f и Stoch %.2f ≥ %.0f — обе зоны перекупленности",
			in.rsiNow, s.p.RSIOverbought, in.stochNow, s.p.StochOverbought)
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
	return s.explainFrom(s.buildInput(md))
}

func (s *Strategy) explainFrom(in decideInput) string {
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

	// 3. Optional volume filter.
	if s.p.UseVolume == 1 {
		switch {
		case in.volOK && in.entryVol < in.avgVol*s.p.VolMult:
			return block("Объём: бар входа %.0f < порога %.0f (среднее %.0f × %.2g, бары выходных исключены)",
				in.entryVol, in.avgVol*s.p.VolMult, in.avgVol, s.p.VolMult)
		case in.volOK:
			pass("Объём: бар входа %.0f ≥ порога %.0f (среднее %.0f × %.2g)",
				in.entryVol, in.avgVol*s.p.VolMult, in.avgVol, s.p.VolMult)
		default:
			pass("Объём: фильтр включён, но базу не посчитать (нет данных) — пропуск")
		}
	}

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
