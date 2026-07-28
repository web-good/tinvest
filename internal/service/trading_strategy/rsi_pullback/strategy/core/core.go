// Package core implements a long-only intraday RSI pullback strategy. When flat it buys the dip
// inside an uptrend: the fast EMA must sit above the slow one and a short RSI must cross DOWN
// through its lower band on the current bar. The position is closed on the first of: the ATR
// stop, RSI crossing UP through the upper band, a time stop measured in bars, or the day-end
// force close — positions never survive into the next day. The decision logic is pure, stateless
// between bars and ticker-agnostic. The reference timeframe is 30 minutes; the EOD gate infers
// the bar span from the series, so other -interval values work as well. Run with
// `-strategy rsi_pullback -interval Minutes30`.
package core

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// defaultBarSpanMin is the bar length in minutes assumed when the series carries no usable
// open-times (a dead fallback in practice: the backtest and -explain paths always populate
// Times, so barSpanMinutes infers the real span). It matches this strategy's 30-minute
// reference timeframe so the EOD gate degrades sanely rather than under-detecting the day-end
// bar.
const defaultBarSpanMin = 30

// minLookback floors the candle window at roughly one trading week of 30-minute bars, so the
// session and RSI gates always see enough history even with short indicator periods.
const minLookback = 120

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod       int     // RSI length (grid; default 4)
	RSILower        float64 // lower band; a DOWNWARD cross of it is the entry (grid; default 15)
	RSIUpper        float64 // upper band; an UPWARD cross of it is the exit (grid; default 70)
	EMAFast         int     // fast EMA period (grid; default 10)
	EMASlow         int     // slow EMA period (grid; default 100)
	StopATR         float64 // stop = entry - StopATR*ATR; 0 disables the stop (grid; never 0 in the grid)
	ATRPeriod       int     // ATR length; used only when StopATR>0
	MaxHoldBars     int     // time stop in bars; 0 disables it (grid)
	SessionStartMin int     // entry window start, minutes from MSK midnight (420 = 07:00)
	SessionEndMin   int     // entry window end, minutes from MSK midnight (1020 = 17:00)
	DayEndMin       int     // day-end force-close boundary, minutes from MSK midnight (1380 = 23:00)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:       4,
		RSILower:        15,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		StopATR:         1.2,
		ATRPeriod:       14,
		MaxHoldBars:     8,
		SessionStartMin: 420,
		SessionEndMin:   1020,
		DayEndMin:       1380,
	}
}

// Strategy trades a single instrument with the RSI pullback rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window the engine feeds Decide on every bar. It must cover the
// hungriest indicator with room to converge: ema.Compute seeds on an SMA over the first
// `period` closes, so a window of exactly `period` bars yields a bare seed — and a window
// SHORTER than the period yields an all-zero series, which silently fails the trend gate for
// the whole run instead of erroring. Doubling the largest period leaves as many recursion steps
// as the seed span; the +20 covers the two-bar cross lookups and the ATR's extra bar.
func (s *Strategy) Lookback() int {
	need := max(s.p.EMASlow, s.p.EMAFast, s.p.RSIPeriod, s.p.ATRPeriod)
	return max(minLookback, 2*need+20)
}

// mskLoc anchors the session windows to the Moscow calendar (UTC fallback).
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

// inSession reports whether bar-time t falls inside the entry window in MSK. A zero time skips
// the gate — never block on missing data.
func (s *Strategy) inSession(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return false
	}
	m := tl.Hour()*60 + tl.Minute()
	return m >= s.p.SessionStartMin && m < s.p.SessionEndMin
}

// isDayEnd reports whether the bar opening at t, spanning spanMin minutes, is the last one
// before the day-end force-close boundary (DayEndMin). It is decoupled from the entry cutoff so
// a position opened inside the entry window is still managed through the evening session up to
// DayEndMin. A zero time degrades the EOD exit to a no-op.
func (s *Strategy) isDayEnd(t time.Time, spanMin int) bool {
	if t.IsZero() {
		return false
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return true
	}
	m := tl.Hour()*60 + tl.Minute()
	return m+spanMin >= s.p.DayEndMin
}

// crossedIntoNewDay reports whether the current bar belongs to a different MSK calendar date
// than the position's entry. It backstops isDayEnd, which can only fire on a bar that actually
// exists: on a truncated session (data gap, halt, short trading day) the series simply has no
// bar reaching DayEndMin, and without this check a position would ride overnight — contradicting
// the strategy's "never hold into the next day" invariant. Unknown times keep it silent, exactly
// as the time stop does with a missing EntryTime.
func crossedIntoNewDay(barT, entryT time.Time) bool {
	if barT.IsZero() || entryT.IsZero() {
		return false
	}
	by, bm, bd := barT.In(mskLoc).Date()
	ey, em, ed := entryT.In(mskLoc).Date()
	return by != ey || bm != em || bd != ed
}

// barSpanMinutes infers the bar length from the series' own open-times: the MEDIAN gap between
// consecutive bars (robust to session and weekend jumps). Falls back to defaultBarSpanMin when
// Times is absent or too short.
func barSpanMinutes(times []time.Time) int {
	if len(times) < 2 {
		return defaultBarSpanMin
	}
	gaps := make([]int, 0, len(times)-1)
	for i := 1; i < len(times); i++ {
		if d := int(times[i].Sub(times[i-1]) / time.Minute); d > 0 {
			gaps = append(gaps, d)
		}
	}
	if len(gaps) == 0 {
		return defaultBarSpanMin
	}
	sort.Ints(gaps)
	return gaps[len(gaps)/2]
}

// barTime returns the open-time of the latest bar, or the zero time when Times is absent or
// misaligned with Closes (so time-based gates degrade instead of misfiring).
func (s *Strategy) barTime(md strategy.MarketData) time.Time {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n {
		return time.Time{}
	}
	return md.Times[n-1]
}

// emaPair computes the fast and slow EMA series (index-aligned to closes) and reports ok=true
// only when both are warmed at the last bar. ema.Compute zero-fills warm-up positions, so an
// unwarmed value would compare as a spurious 0.
func (s *Strategy) emaPair(closes []float64) (fast, slow []float64, ok bool) {
	fast = ema.Compute(closes, s.p.EMAFast)
	slow = ema.Compute(closes, s.p.EMASlow)
	i := len(closes) - 1
	if i < 0 || len(fast) != len(closes) || len(slow) != len(closes) {
		return nil, nil, false
	}
	return fast, slow, fast[i] > 0 && slow[i] > 0
}

// crossedDown reports whether series crossed down through level between i-1 and i: it sat at or
// above the level and is now strictly below. The series[i-1] > 0 guard rejects RSISeries warm-up
// zeros reading as "below the level".
func crossedDown(series []float64, i int, level float64) bool {
	return i >= 1 && series[i-1] > 0 && series[i-1] >= level && series[i] < level
}

// crossedUp reports whether series crossed up through level between i-1 and i: it sat at or
// below the level and is now strictly above. Mirrors crossedDown, so a bar sitting exactly ON
// the level is treated as "not yet crossed" in both directions.
//
// The series[i-1] > 0 guard is deliberately kept even though it is NOT inert here the way it is
// in crossedDown: Wilder's avgGain sits at exactly 0 while every bar since the seed is a loss
// (pkg/indicators/rsi.go), so an RSI of 0 can be a real reading, and this guard suppresses a
// cross that starts from it. That asymmetry is intentional and safe. The suppressed case needs
// an up-bar worth several average losses right after an unbroken losing streak while a long is
// open — a position that would almost certainly have hit its stop first — and the cost is
// bounded: the exit is only delayed, SL/TIME/EOD stay armed. Dropping the guard would instead
// let an RSISeries warm-up zero manufacture an exit out of nothing. Pinned by
// TestCrossHelpersBoundaries.
func crossedUp(series []float64, i int, level float64) bool {
	return i >= 1 && series[i-1] > 0 && series[i-1] <= level && series[i] > level
}

// Decide routes to entry (flat) or position management (open). The bar span is inferred once
// here and threaded down: barSpanMinutes sorts a scratch slice, and the calibration sweeps this
// path millions of times.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	span := barSpanMinutes(md.Times)
	if md.Position != nil {
		return s.manage(md, sig, span)
	}
	return s.enter(md, sig, span)
}

// enter emits a long when a short RSI crosses DOWN through its lower band on the current bar
// while the fast EMA sits above the slow one. Everything is recomputed from md — no state
// survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal, span int) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. session window, and never on the day-end bar (manage() only runs from the NEXT bar, so
	// an entry on the day-end bar could not be EOD-closed on its own bar).
	t := s.barTime(md)
	if !s.inSession(t) || s.isDayEnd(t, span) {
		return sig
	}
	i := n - 1
	// 2. RSI crosses down through the lower band on the current bar.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n || !crossedDown(rsi, i, s.p.RSILower) {
		return sig
	}
	// 3. trend confirmation: fast EMA above slow EMA (both warmed).
	fast, slow, ok := s.emaPair(md.Closes)
	if !ok || fast[i] <= slow[i] {
		return sig
	}
	// 4. optional ATR stop; a non-positive ATR means the data cannot support the stop, and an
	// entry without its planned protection is refused.
	entry := md.Closes[i]
	var stop, atr float64
	if s.p.StopATR > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		if atr <= 0 {
			return sig
		}
		stop = entry - s.p.StopATR*atr
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i], fast[i], slow[i], entry, stop, atr)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(rsiNow, fastNow, slowNow, entry, stop, atr float64) string {
	stopHow := "стоп выключен (StopATR=0)"
	if s.p.StopATR > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (вход − %.2f×ATR, ATR=%.4f)", stop, s.p.StopATR, atr)
	}
	return fmt.Sprintf(
		"RSI(%d) ушёл под %.0f (%.1f) на откате, EMA(%d) %.4f > EMA(%d) %.4f; вход %.4f, %s",
		s.p.RSIPeriod, s.p.RSILower, rsiNow, s.p.EMAFast, fastNow, s.p.EMASlow, slowNow, entry, stopHow,
	)
}

// holdUnknown is returned by barsHeld when the position's entry time is unknown. The time stop
// treats it as "do not fire": a missing EntryTime must never close a position by itself.
const holdUnknown = -1

// barsHeld counts bars from the position's entry to the current bar, purely from EntryTime, the
// current bar time and the data-inferred span. A position is force-closed at the day end, so in
// the normal case entry and current bar sit in the same session and the uniform span is exact.
// The one exception is a truncated session, where the position survives to the first bar of the
// next trading day (crossedIntoNewDay closes it there): that single count spans the overnight
// gap and therefore OVERSTATES the hold. The error is one-sided and harmless — it can only make
// an armed time stop fire on the very bar the EOD backstop would have closed anyway.
func (s *Strategy) barsHeld(md strategy.MarketData, span int) int {
	pos := md.Position
	t := s.barTime(md)
	if pos == nil || pos.EntryTime.IsZero() || t.IsZero() || span <= 0 {
		return holdUnknown
	}
	return int(math.Round(t.Sub(pos.EntryTime).Minutes() / float64(span)))
}

// manage handles an open long, exiting in precedence SL -> RSI -> TIME -> EOD. SL is read from
// the position (frozen at entry), never recomputed: the stop the trade was opened with is the
// stop it dies by. RSI fires on an UPWARD cross of RSIUpper — taking the bounce into overbought.
// RSI/TIME/EOD fill at the bar close; SL fills at the stop level (the engine handles that via
// model.IsStopReason).
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal, span int) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low := md.Lows[i]
	closeP := md.Closes[i]

	// 1. hard stop (always active, wins any same-bar tie).
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}
	// 2. RSI crosses UP through the upper band — the bounce reached overbought.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n && crossedUp(rsi, i, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) пересёк %.0f снизу вверх (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, rsi[i], closeP, pos.PurchasePrice)
		return sig
	}
	// 3. time stop: the setup had its chance and did not deliver.
	if held := s.barsHeld(md, span); s.p.MaxHoldBars > 0 && held != holdUnknown && held >= s.p.MaxHoldBars {
		sig.Kind, sig.Reason = model.SignalSell, "TIME"
		sig.ExitReason = fmt.Sprintf("TIME: удержано %d баров ≥ %d, выход по %.4f (вход %.4f)",
			held, s.p.MaxHoldBars, closeP, pos.PurchasePrice)
		return sig
	}
	// 4. end of day (always active): either this is the last bar before DayEndMin, or the session
	// was truncated and we are already looking at a bar from a later MSK day.
	barT := s.barTime(md)
	if s.isDayEnd(barT, span) || crossedIntoNewDay(barT, pos.EntryTime) {
		sig.Kind, sig.Reason = model.SignalSell, "EOD"
		sig.ExitReason = fmt.Sprintf("EOD: закрытие на конец дня по %.4f (вход %.4f)", closeP, pos.PurchasePrice)
	}
	return sig
}

// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace
// (-explain). It recomputes the same values enter()/manage() do and reports each gate.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		sb.WriteString("недостаточно свечей\n")
		return sb.String()
	}
	i := n - 1
	barT := s.barTime(md)
	span := barSpanMinutes(md.Times)
	fmt.Fprintf(&sb, "сессия: вход разрешён? %v (бар %v); конец дня? %v\n",
		s.inSession(barT), barT, s.isDayEnd(barT, span))

	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n {
		fmt.Fprintf(&sb, "RSI(%d) пред %.1f тек %.1f; вход-крест вниз через %.0f? %v; выход-крест вверх через %.0f? %v\n",
			s.p.RSIPeriod, rsi[i-1], rsi[i], s.p.RSILower, crossedDown(rsi, i, s.p.RSILower),
			s.p.RSIUpper, crossedUp(rsi, i, s.p.RSIUpper))
	} else {
		sb.WriteString("RSI: недостаточно истории\n")
	}

	if fast, slow, ok := s.emaPair(md.Closes); ok {
		fmt.Fprintf(&sb, "EMA(%d) %.4f vs EMA(%d) %.4f: тренд вверх? %v\n",
			s.p.EMAFast, fast[i], s.p.EMASlow, slow[i], fast[i] > slow[i])
	} else {
		sb.WriteString("EMA: не прогрето\n")
	}

	if s.p.StopATR > 0 {
		atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		fmt.Fprintf(&sb, "стоп: вход − %.2f×ATR (ATR=%.4f)\n", s.p.StopATR, atr)
	} else {
		sb.WriteString("стоп: выключен (StopATR=0)\n")
	}

	if md.Position == nil {
		fmt.Fprintf(&sb, "удержание: позиции нет; тайм-стоп %s\n", holdLabel(s.p.MaxHoldBars))
	} else {
		fmt.Fprintf(&sb, "удержание: %s; тайм-стоп %s\n",
			heldLabel(s.barsHeld(md, span)), holdLabel(s.p.MaxHoldBars))
	}
	return sb.String()
}

// holdLabel renders the time stop for Explain: the bar count when armed, "выключен" when off.
func holdLabel(v int) string {
	if v <= 0 {
		return "выключен"
	}
	return fmt.Sprintf("%d баров", v)
}

// heldLabel renders the measured hold for Explain, keeping the holdUnknown sentinel out of the
// diagnostics the owner reads (it would otherwise print as "-1 баров").
func heldLabel(v int) string {
	if v == holdUnknown {
		return "неизвестно"
	}
	return fmt.Sprintf("%d баров", v)
}
