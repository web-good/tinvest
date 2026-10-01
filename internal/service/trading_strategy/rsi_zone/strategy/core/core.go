// Package core implements a long-only multi-day RSI zone strategy. When flat it buys a short RSI
// crossing DOWN through its lower critical band on the current bar while the close sits above a
// single trend EMA. It sells when the same RSI crosses UP through its upper critical band, or
// when the protective stop — sized in daily ATR at entry and frozen on the position — is
// touched. Optionally (StuckExitBars > 0) it also sells when RSI has sat below the lower band on
// every one of the first StuckExitBars bars after the entry: the bounce never started. There is
// no target, no trail and no end-of-day close: the position is held across nights and weekends
// until an exit fires. The decision logic is pure, stateless between bars and ticker-agnostic. The reference timeframe is 30 minutes. Run with
// `-strategy rsi_zone -interval Minutes30`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback floors the candle window at roughly one trading week of 30-minute bars, so the
// RSI always sees enough history even with a short trend EMA.
const minLookback = 120

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod      int     // RSI length (grid; default 4)
	RSILower       float64 // lower critical band; a DOWNWARD cross of it is the entry (grid; default 25)
	RSIUpper       float64 // upper critical band; an UPWARD cross of it is the exit (grid; default 75)
	EMAPeriod      int     // trend EMA period; the entry needs close > EMA (grid; default 200)
	DailyATRPeriod int     // daily ATR length, over WEEKDAY completed dailies (fixed; default 14)
	StopDailyATR   float64 // stop = entry - StopDailyATR*dailyATR; 0 disables it (grid; never 0 in the grid)
	StuckExitBars  int     // exit when RSI stayed below RSILower on this many bars after the entry bar without ever leaving the zone; 0 disables it (grid: stuck)

	// Stochastic confirmation gate (theme stoch). With UseStoch=1 the entry needs RSI and
	// Stoch %D to BOTH read oversold within the last ZoneWindowBars bars, and one of them to
	// cross down into its zone on the current bar. A zero knob resolves to its default (see
	// stochConfig), so a ticker literal written before the gate switches it on with UseStoch
	// alone; DefaultParams leaves them all zero and the gate off.
	UseStoch       int     // 0 = off; any non-zero value = require the stochastic confirmation (grid: stoch, 1)
	StochKPeriod   int     // %K lookback; 0 -> 14; negative refuses every entry (grid: stoch)
	StochDSmooth   int     // %D smoothing, 1 = raw %K; 0 -> 3; negative refuses every entry (grid: stoch)
	StochLower     float64 // Stoch %D lower critical band; 0 -> 20 (grid: stoch)
	ZoneWindowBars int     // bars, current one included, in which the other oscillator may have been in its zone; < 1 -> 1 (grid: stoch)
}

// Defaults the stochastic gate falls back to when its knobs are left at zero.
const (
	defaultStochKPeriod = 14
	defaultStochDSmooth = 3
	defaultStochLower   = 20.0
)

// stochConfig resolves the stochastic gate's knobs: a zero period or band falls back to its
// default and a window below one bar becomes one bar. Negative periods pass through untouched
// so stochDSeries refuses them — a misconfiguration must block entries, not be papered over.
func stochConfig(p Params) (k, d int, lower float64, window int) {
	k, d, lower, window = p.StochKPeriod, p.StochDSmooth, p.StochLower, p.ZoneWindowBars
	if k == 0 {
		k = defaultStochKPeriod
	}
	if d == 0 {
		d = defaultStochDSmooth
	}
	if lower == 0 {
		lower = defaultStochLower
	}
	if window < 1 {
		window = 1
	}
	return k, d, lower, window
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      200,
		DailyATRPeriod: 14,
		StopDailyATR:   1.0,
	}
}

// Strategy trades a single instrument with the RSI zone rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window the engine feeds Decide on every bar. ema.Compute seeds on
// an SMA over the first `period` closes, so a window shorter than the period yields an all-zero
// series that silently fails the trend gate for the whole run. Doubling the largest period
// leaves as many recursion steps as the seed span; the +20 covers the two-bar cross lookups.
// With the stochastic gate on, the %K+%D warm-up joins the largest period and the confirmation
// window is added on top; with it off the window is exactly what it was before the gate existed.
func (s *Strategy) Lookback() int {
	span := max(s.p.EMAPeriod, s.p.RSIPeriod)
	if s.p.UseStoch == 0 {
		return max(minLookback, 2*span+20)
	}
	k, d, _, window := stochConfig(s.p)
	return max(minLookback, 2*max(span, k+d)+window+20)
}

// mskLoc anchors every calendar rule (weekday checks) to Moscow (UTC fallback).
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

// weekdayDaily drops weekend (Sat/Sun MSK) bars from the daily series, keeping the three price
// slices aligned. MOEX weekend sessions are several times narrower and thinner than weekday
// ones; leaving them in understates the daily ATR that sizes the stop. When times is empty or
// not aligned with the price slices there is nothing to filter by — the series is returned
// untouched rather than guessed at.
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

// dailyATR is the strategy's unit of risk: Wilder's ATR over COMPLETED weekday daily candles.
// The engine only exposes days that closed before the current bar, so no lookahead is possible.
// Returns 0 whenever the data cannot support the calculation — the caller must then refuse the
// entry, because without it there is no stop.
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

// tradingDay reports whether an entry may be opened on bar-time t: any bar of a Mon-Fri MSK
// day. A zero time skips the gate — never block on missing data.
func (s *Strategy) tradingDay(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	return !isWeekend(t.In(mskLoc))
}

// barTime returns the open-time of the latest bar, or the zero time when Times is absent or
// misaligned with Closes (so the weekday gate degrades instead of misfiring).
func (s *Strategy) barTime(md strategy.MarketData) time.Time {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n {
		return time.Time{}
	}
	return md.Times[n-1]
}

// crossedDown reports whether series crossed down through level between i-1 and i: it sat at or
// above the level and is now strictly below. period is the RSI length used to build series:
// RSISeries (pkg/indicators/rsi.go) leaves indices below period as an unset zero rather than a
// genuine reading, so validity is gated on the INDEX (i-1 >= period), not on the VALUE. Gating
// on the value would reject a genuine RSI of exactly 0.00 — which Wilder's formula produces
// whenever avgGain is 0, i.e. after a long enough run of falling bars — as if it were still
// warm-up, silently eating a real cross on the very next bar.
func crossedDown(series []float64, i, period int, level float64) bool {
	return i >= 1 && i < len(series) && i-1 >= period && series[i-1] >= level && series[i] < level
}

// crossedUp reports whether series crossed up through level between i-1 and i: it sat at or
// below the level and is now strictly above. See crossedDown for why validity is gated on the
// index rather than the value: a genuine post-warm-up RSI of exactly 0.00 must still count as a
// real prior reading, so a sharp bounce past the upper band on the next bar is not missed.
func crossedUp(series []float64, i, period int, level float64) bool {
	return i >= 1 && i < len(series) && i-1 >= period && series[i-1] <= level && series[i] > level
}

// stochDSeries returns Stochastic %D laid over the full bar index: series[b] is the %D reading
// of bar b, and warm = k+d-2 is the first bar that has one (earlier slots are an unset zero,
// the same convention RSISeries uses). indicators.StochasticSeries returns a shorter,
// right-aligned slice; its j-th value belongs to bar j+k+d-2. ok is false when the periods are
// not positive or history is too short to produce a single %D value.
func stochDSeries(highs, lows, closes []float64, k, d int) (series []float64, warm int, ok bool) {
	if k <= 0 || d <= 0 {
		return nil, 0, false
	}
	_, ds := indicators.StochasticSeries(highs, lows, closes, k, d)
	n := len(closes)
	warm = k + d - 2
	if len(ds) == 0 || warm+len(ds) != n {
		return nil, 0, false
	}
	series = make([]float64, n)
	copy(series[warm:], ds)
	return series, warm, true
}

// oscillator is one confirmation series with its first genuine index and its lower band.
type oscillator struct {
	series []float64
	warm   int
	level  float64
}

// zoneHit records which oscillator crossed on the current bar and where the other one was seen
// in its zone.
type zoneHit struct {
	by      string // "RSI" or "Stoch": the oscillator that crossed down on the current bar
	otherAt int    // the most recent bar of the window where the other oscillator was in its zone
}

// seenInZone returns the most recent bar of [i-window+1, i] (clipped at 0) where o reads
// strictly below its band on a warmed index, or -1. Validity is gated on the index, as in
// crossedDown: a genuine 0.00 after warm-up counts, an unset warm-up zero does not.
func seenInZone(o oscillator, i, window int) int {
	for b := i; b >= 0 && b > i-window; b-- {
		if b >= o.warm && b < len(o.series) && o.series[b] < o.level {
			return b
		}
	}
	return -1
}

// zoneTrigger is the symmetric confirmation: an entry fires when RSI crosses down through its
// band on bar i and Stoch was in its zone somewhere in the window, or the other way round. When
// both cross on the same bar the RSI trigger is reported. crossedDown's period argument is the
// oscillator's warm index, so the previous bar must itself be a genuine reading.
func zoneTrigger(rsi, stoch oscillator, i, window int) (zoneHit, bool) {
	if crossedDown(rsi.series, i, rsi.warm, rsi.level) {
		if b := seenInZone(stoch, i, window); b >= 0 {
			return zoneHit{by: "RSI", otherAt: b}, true
		}
	}
	if crossedDown(stoch.series, i, stoch.warm, stoch.level) {
		if b := seenInZone(rsi, i, window); b >= 0 {
			return zoneHit{by: "Stoch", otherAt: b}, true
		}
	}
	return zoneHit{}, false
}

// trendUp reports whether the close sits strictly above a warmed EMA. ema.Compute zero-fills
// warm-up positions, so an unwarmed EMA must not pass as "price above zero".
func trendUp(closeP, emaNow float64) bool {
	return emaNow > 0 && closeP > emaNow
}

// StopLevel returns the protective stop for a position opened at entry with the daily ATR
// frozen at entry, or 0 when there is no stop: the stop is disabled, the ATR is unknown, or the
// level would land at or below zero (that is a naked long, not a floor).
func StopLevel(p Params, entry, entryATR float64) float64 {
	if p.StopDailyATR <= 0 || entryATR <= 0 {
		return 0
	}
	level := entry - p.StopDailyATR*entryATR
	if level <= 0 {
		return 0
	}
	return level
}

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter emits a long when a short RSI crosses DOWN through its lower band on the current bar
// while the close sits above the trend EMA on a weekday and the daily ATR is known. Everything
// is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n || s.p.RSIPeriod <= 0 || s.p.EMAPeriod <= 0 {
		return sig
	}
	// 1. weekday: any time of a trading day will do, weekends will not.
	if !s.tradingDay(s.barTime(md)) {
		return sig
	}
	i := n - 1
	// 2. the trigger. Gate off: RSI crosses down through the lower band on the current bar.
	// Gate on: RSI or Stoch %D crosses down into its zone now while the other one was in its
	// zone somewhere in the last ZoneWindowBars bars.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		return sig
	}
	var (
		hit   zoneHit
		stoch []float64
	)
	if s.p.UseStoch != 0 {
		k, d, lower, window := stochConfig(s.p)
		st, warm, ok := stochDSeries(md.Highs, md.Lows, md.Closes, k, d)
		if !ok {
			return sig
		}
		h, fired := zoneTrigger(
			oscillator{series: rsi, warm: s.p.RSIPeriod, level: s.p.RSILower},
			oscillator{series: st, warm: warm, level: lower},
			i, window,
		)
		if !fired {
			return sig
		}
		hit, stoch = h, st
	} else if !crossedDown(rsi, i, s.p.RSIPeriod, s.p.RSILower) {
		return sig
	}
	// 3. trend: close above a warmed EMA.
	trend := ema.Compute(md.Closes, s.p.EMAPeriod)
	if len(trend) != n || !trendUp(md.Closes[i], trend[i]) {
		return sig
	}
	// 4. the daily ATR is the unit of the stop: no ATR, no trade.
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	// 5. an armed stop must land above zero, otherwise the long would be unprotected.
	entry := md.Closes[i]
	stop := StopLevel(s.p, entry, atr)
	if s.p.StopDailyATR > 0 && stop <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i], trend[i], entry, stop, atr)
	if s.p.UseStoch != 0 {
		sig.EntryReason += s.stochReason(hit, i, rsi[i], stoch[i])
	}
	return sig
}

// stochReason is the journal tail of a stochastic-confirmed entry: which oscillator crossed,
// how many bars back the other one was in its zone, and both readings on the entry bar.
func (s *Strategy) stochReason(hit zoneHit, i int, rsiNow, stochNow float64) string {
	k, d, lower, window := stochConfig(s.p)
	other := "Stoch"
	if hit.by == "Stoch" {
		other = "RSI"
	}
	return fmt.Sprintf("; подтверждение: крест дал %s, %s в зоне %d бар(ов) назад (окно %d); RSI %.1f (зона <%.0f), Stoch%%D(%d,%d) %.1f (зона <%.0f)",
		hit.by, other, i-hit.otherAt, window, rsiNow, s.p.RSILower, k, d, stochNow, lower)
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(rsiNow, emaNow, entry, stop, atr float64) string {
	stopHow := "стоп выключен"
	if stop > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (−%.2f ATR)", stop, s.p.StopDailyATR)
	}
	return fmt.Sprintf(
		"RSI(%d) ушёл под %.0f (%.1f), close %.4f > EMA(%d) %.4f, дневной ATR %.4f; вход %.4f, %s",
		s.p.RSIPeriod, s.p.RSILower, rsiNow, entry, s.p.EMAPeriod, emaNow, atr, entry, stopHow,
	)
}

// manage handles an open long. It exits on one of two signals, evaluated in precedence order
// SL → RSI. The stop triggers INTRABAR (the bar's low touching the level), because a real stop
// order fills as soon as price trades through it; the engine prices that fill via
// model.IsStopReason. It wins a same-bar tie with the RSI exit: the intrabar order is unknowable
// from OHLC, and assuming the worse outcome is the honest choice. The stop level is rebuilt from
// the entry price and the daily ATR frozen at entry, never from the current ATR. The RSI exit
// fills at the bar close. There is no time stop and no end-of-day close.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low, closeP := md.Lows[i], md.Closes[i]

	// 1. protective stop, frozen at entry.
	if level := StopLevel(s.p, pos.PurchasePrice, pos.EntryATR); level > 0 && low <= level {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = level
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, level, pos.PurchasePrice)
		return sig
	}
	// 2. RSI crosses UP through the upper band — the bounce reached the upper critical zone.
	if s.p.RSIPeriod <= 0 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		return sig
	}
	if crossedUp(rsi, i, s.p.RSIPeriod, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) пересёк %.0f снизу вверх (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, rsi[i], closeP, pos.PurchasePrice)
		return sig
	}
	// 3. optional: RSI never left the lower zone for StuckExitBars bars after the entry bar.
	if bars := s.stuckBars(md, rsi); s.p.StuckExitBars > 0 && bars >= s.p.StuckExitBars {
		sig.Kind, sig.Reason = model.SignalSell, "STUCK"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("STUCK: RSI(%d) %d бар(ов) после входа ниже %.0f (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, bars, s.p.RSILower, rsi[i], closeP, pos.PurchasePrice)
	}
	return sig
}

// stuckBars counts the bars after the entry bar on which RSI read strictly below RSILower,
// provided it did so on EVERY one of them; a single bar at or above the band (or an unwarmed
// reading) disarms the stuck exit for the rest of the trade and yields 0. The entry bar is the
// last bar that opened at or before Position.EntryTime and is not counted itself. Without
// EntryTime or aligned Times, or when the entry bar has already left the candle window, there is
// no anchor and the result is 0: the window (>= minLookback bars) dwarfs any StuckExitBars, so a
// position that old either exited long ago or was disarmed by a rise the window may no longer show.
func (s *Strategy) stuckBars(md strategy.MarketData, rsi []float64) int {
	pos, n := md.Position, len(md.Closes)
	if s.p.StuckExitBars <= 0 || pos == nil || pos.EntryTime.IsZero() || len(md.Times) != n || len(rsi) != n {
		return 0
	}
	entryIdx := -1
	for b := n - 1; b >= 0; b-- {
		if !md.Times[b].After(pos.EntryTime) {
			entryIdx = b
			break
		}
	}
	if entryIdx < 0 {
		return 0
	}
	for b := entryIdx + 1; b < n; b++ {
		if b < s.p.RSIPeriod || rsi[b] >= s.p.RSILower {
			return 0
		}
	}
	return n - 1 - entryIdx
}
