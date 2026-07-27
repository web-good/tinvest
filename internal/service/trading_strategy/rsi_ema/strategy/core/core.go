// Package core implements a long-only intraday RSI+EMA trend strategy. When flat it enters
// on RSI crossing up through its mid level (50) while the fast EMA sits above the slow EMA.
// It exits on any of: the fast EMA crossing below the slow EMA, RSI crossing UP through the
// upper level (70, taking profit into overbought), or RSI crossing DOWN through the mid level
// (50) — the last gated by a post-entry cooldown so the setup is not closed on the first
// pullback right after entry. An optional ATR stop (StopATR>0) and an end-of-day close cap the
// risk. The decision logic is pure, stateless between bars and ticker-agnostic. The reference
// timeframe is 15 minutes, but the rules are timeframe-agnostic: the EOD gate infers the bar
// span from the series, so other -interval values work as well. Run with
// `-strategy rsi_ema -interval Minutes15`.
package core

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// defaultBarSpanMin is the bar length in minutes assumed when the series carries no usable
// open-times (a dead fallback in practice: the backtest and -explain paths always populate
// Times, so barSpanMinutes infers the real span). It matches this strategy's 15-minute
// reference timeframe so the EOD gate degrades sanely rather than under-detecting the day-end
// bar; the actual span is read from the data via barSpanMinutes whenever Times is present.
const defaultBarSpanMin = 15

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod         int     // RSI length (grid; default 12)
	EMAFast           int     // fast EMA period (grid; default 10)
	EMASlow           int     // slow EMA period (grid; default 50)
	RSIMid            float64 // entry cross level and the RSI50 exit level (default 50)
	RSIUpper          float64 // RSI70 exit level, crossed UPWARD (default 70)
	EntryCooldownBars int     // bars after entry during which the RSI50 exit is ignored (grid)
	StopATR           float64 // stop = entry - StopATR*ATR; 0 disables the stop (grid)
	ATRPeriod         int     // ATR length; used only when StopATR>0
	SessionStartMin   int     // entry window start, minutes from MSK midnight (420 = 07:00)
	SessionEndMin     int     // Mon-Thu ENTRY cutoff, minutes from MSK midnight (1080 = 18:00)
	FridayEndMin      int     // Friday ENTRY cutoff, minutes from MSK midnight (840 = 14:00)
	DayEndMin         int     // day-end force-close boundary, minutes from MSK midnight (1380 = 23:00)

	EntryLookbackBars    int // fresh-entry window: bars before the cross to inspect (grid; default 5)
	EntryAboveMidLimit   int // reject entry when >= this many bars in the window are above RSIMid; <=0 disables (grid; default 0 = off)
	EntryMaxMidCrossings int // reject entry when the window holds >= this many RSIMid crossings; <=0 disables (grid; default 0 = off)

	UseVolume      int     // 1 arms the volume-regime entry gate; any other value disables it (grid; default 0 = off)
	VolShortPeriod int     // recent-activity window in bars, INCLUDING the entry bar (default 10)
	VolLongPeriod  int     // background window in bars; must exceed VolShortPeriod (default 50)
	VolMult        float64 // entry requires shortAvg >= longAvg*VolMult (grid; default 1.0)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:         12,
		EMAFast:           10,
		EMASlow:           50,
		RSIMid:            50,
		RSIUpper:          70,
		EntryCooldownBars: 1,
		StopATR:           0,
		ATRPeriod:         14,
		SessionStartMin:   420,
		SessionEndMin:     1080,
		FridayEndMin:      840,
		DayEndMin:         1380,

		EntryLookbackBars:    5,
		EntryAboveMidLimit:   0,
		EntryMaxMidCrossings: 0,

		UseVolume:      0,
		VolShortPeriod: 10,
		VolLongPeriod:  50,
		VolMult:        1.0,
	}
}

// Strategy trades a single instrument with the RSI+EMA rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to warm the slow EMA, the RSI and the ATR with margin.
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
// skips the gate — never block on missing data.
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

// isDayEnd reports whether the bar opening at t, spanning spanMin minutes, is the last one
// before the day-end force-close boundary (DayEndMin), decoupled from the entry cutoff so a
// position opened inside the entry window (≤ SessionEndMin) is still held and managed through
// the evening session up to DayEndMin. A zero time degrades the EOD exit to a no-op.
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

// barSpanMinutes infers the bar length from the series' own open-times: the MEDIAN gap
// between consecutive bars (robust to session/weekend jumps). Falls back to defaultBarSpanMin
// when Times is absent or too short.
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

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
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

// crossedUp reports whether series crossed strictly up through level between i-1 and i. The
// series[i-1] > 0 guard rejects RSISeries warm-up zeros reading as "below the level".
func crossedUp(series []float64, i int, level float64) bool {
	return i >= 1 && series[i-1] > 0 && series[i-1] < level && series[i] > level
}

// barsAboveMid counts, within the EntryLookbackBars bars immediately before bar i (indices
// i-EntryLookbackBars .. i-1, clamped at 0), how many had RSI strictly above RSIMid.
func (s *Strategy) barsAboveMid(rsi []float64, i int) int {
	start := i - s.p.EntryLookbackBars
	if start < 0 {
		start = 0
	}
	n := 0
	for j := start; j < i && j < len(rsi); j++ {
		if rsi[j] > s.p.RSIMid {
			n++
		}
	}
	return n
}

// midCrossings counts how many times RSI crossed RSIMid — in EITHER direction — between
// consecutive bars inside the EntryLookbackBars window before bar i (indices
// i-EntryLookbackBars .. i-1, clamped at 0). The entry cross itself (i-1 -> i) lies outside the
// window and is never counted, so the maximum is EntryLookbackBars-1. The rsi[j-1] > 0 guard
// rejects RSISeries warm-up zeros (same discipline as crossedUp/crossedDown); a bar sitting
// exactly at RSIMid is treated as "no crossing".
func (s *Strategy) midCrossings(rsi []float64, i int) int {
	start := i - s.p.EntryLookbackBars
	if start < 0 {
		start = 0
	}
	end := i
	if end > len(rsi) {
		end = len(rsi)
	}
	n := 0
	for j := start + 1; j < end; j++ {
		prev, cur := rsi[j-1], rsi[j]
		if prev <= 0 {
			continue
		}
		if (prev < s.p.RSIMid && cur > s.p.RSIMid) || (prev > s.p.RSIMid && cur < s.p.RSIMid) {
			n++
		}
	}
	return n
}

// limitLabel renders an optional integer gate limit for Explain: the number when the sub-filter
// is armed, "выключен" when it is off.
func limitLabel(v int) string {
	if v <= 0 {
		return "выключен"
	}
	return strconv.Itoa(v)
}

// freshByBarsAbove reports whether the bars-above-mid sub-filter allows the entry: fewer than
// EntryAboveMidLimit of the preceding bars sat above RSIMid. Rejects choppy re-entries where RSI
// only briefly dipped below the mid line after a sustained run above it. Off (always true) when
// EntryAboveMidLimit<=0.
func (s *Strategy) freshByBarsAbove(rsi []float64, i int) bool {
	return s.p.EntryAboveMidLimit <= 0 || s.barsAboveMid(rsi, i) < s.p.EntryAboveMidLimit
}

// freshByCrossings reports whether the chop sub-filter allows the entry: fewer than
// EntryMaxMidCrossings crossings of RSIMid inside the window. Rejects saws where RSI flips
// across the mid line every couple of bars. Off (always true) when EntryMaxMidCrossings<=0.
func (s *Strategy) freshByCrossings(rsi []float64, i int) bool {
	return s.p.EntryMaxMidCrossings <= 0 || s.midCrossings(rsi, i) < s.p.EntryMaxMidCrossings
}

// freshEntry reports whether the RSI cross at bar i is "fresh" — it must clear BOTH sub-filters.
// EntryLookbackBars<=0 disables the whole gate; each sub-filter also has its own off switch, and
// both are off in DefaultParams (the grid turns them on).
func (s *Strategy) freshEntry(rsi []float64, i int) bool {
	if s.p.EntryLookbackBars <= 0 {
		return true
	}
	return s.freshByBarsAbove(rsi, i) && s.freshByCrossings(rsi, i)
}

// avgVolumeLastN averages the volumes of the last n bars of vols, INCLUDING the final (entry)
// bar — unlike reversion's average, the entry bar's own volume is part of the "recent activity"
// this gate measures. When times is index-aligned to vols, weekend bars (Sat/Sun MSK) are
// dropped; when times is empty or misaligned, weekend exclusion is skipped. Non-positive volumes
// are ignored. ok is false when no sample survives — the caller must then skip the gate (never
// block an entry on missing data).
func avgVolumeLastN(vols []int64, times []time.Time, n int) (avg float64, ok bool) {
	if len(vols) == 0 || n <= 0 {
		return 0, false
	}
	lo := len(vols) - n
	if lo < 0 {
		lo = 0
	}
	haveTimes := len(times) == len(vols)
	var sum float64
	var count int
	for j := lo; j < len(vols); j++ {
		if haveTimes && isWeekend(times[j].In(mskLoc)) {
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

// volumeRegimeOK reports whether the volume background allows an entry: the recent average
// volume must hold at least VolMult times the longer background average, so entries into a dead
// tape are skipped. The gate degrades to "allow" whenever it is disabled (UseVolume != 1),
// misconfigured (non-positive or non-increasing windows) or unsupported by the data (no usable
// volumes) — a missing volume series must never block an entry.
func (s *Strategy) volumeRegimeOK(md strategy.MarketData) bool {
	if s.p.UseVolume != 1 || s.p.VolShortPeriod <= 0 || s.p.VolLongPeriod <= s.p.VolShortPeriod {
		return true
	}
	shortAvg, okShort := avgVolumeLastN(md.Volumes, md.Times, s.p.VolShortPeriod)
	longAvg, okLong := avgVolumeLastN(md.Volumes, md.Times, s.p.VolLongPeriod)
	if !okShort || !okLong || longAvg <= 0 {
		return true
	}
	return shortAvg >= longAvg*s.p.VolMult
}

// enter emits a long when RSI crosses up through RSIMid on the current bar while the fast EMA
// sits above the slow EMA. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. session window, and never on the day-end bar (manage() only runs from the NEXT bar,
	// so an entry on the day-end bar could not be EOD-closed on its own bar).
	t := s.barTime(md)
	if !s.inSession(t) || s.isDayEnd(t, barSpanMinutes(md.Times)) {
		return sig
	}
	i := n - 1
	// 2. RSI crosses up through the mid level on the current bar.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n || !crossedUp(rsi, i, s.p.RSIMid) {
		return sig
	}
	// 3. trend confirmation: fast EMA above slow EMA (both warmed).
	fast, slow, ok := s.emaPair(md.Closes)
	if !ok || fast[i] <= slow[i] {
		return sig
	}
	// 3.5 freshness filter: reject re-entries where RSI recently sat above the mid line
	// (short dip after a sustained run above 50 — chop around the mid, not a fresh reset).
	if !s.freshEntry(rsi, i) {
		return sig
	}
	// 3.6 volume regime: skip breakouts of the mid line on a dead tape (off by default; degrades
	// to "allow" whenever the volume data cannot support the comparison).
	if !s.volumeRegimeOK(md) {
		return sig
	}
	// 4. optional ATR stop.
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
		"RSI(%d) пересёк уровень %.0f снизу вверх (%.1f), EMA(%d) %.4f > EMA(%d) %.4f; вход %.4f, %s",
		s.p.RSIPeriod, s.p.RSIMid, rsiNow, s.p.EMAFast, fastNow, s.p.EMASlow, slowNow, entry, stopHow,
	)
}

// cooldownElapsed is returned by barsSinceEntry when the entry time is unknown, so a missing
// EntryTime never traps the position inside the cooldown.
const cooldownElapsed = math.MaxInt32

// barsSinceEntry counts bars from the position's entry to the current bar, purely from
// EntryTime, the current bar time and the data-inferred span. Positions never survive the
// EOD close, so the window is always within one session and the uniform span is exact.
func (s *Strategy) barsSinceEntry(md strategy.MarketData) int {
	pos := md.Position
	t := s.barTime(md)
	if pos == nil || pos.EntryTime.IsZero() || t.IsZero() {
		return cooldownElapsed
	}
	span := barSpanMinutes(md.Times)
	if span <= 0 {
		return cooldownElapsed
	}
	return int(math.Round(t.Sub(pos.EntryTime).Minutes() / float64(span)))
}

// crossedDown reports whether series crossed down through level between i-1 and i. The
// series[i-1] > 0 guard rejects RSISeries warm-up zeros.
func crossedDown(series []float64, i int, level float64) bool {
	return i >= 1 && series[i-1] > 0 && series[i-1] >= level && series[i] < level
}

// emaCrossDown reports whether the fast EMA crossed below the slow EMA between i-1 and i, with
// both prior values warmed (>0).
func emaCrossDown(fast, slow []float64, i int) bool {
	return i >= 1 && fast[i-1] > 0 && slow[i-1] > 0 && fast[i-1] >= slow[i-1] && fast[i] < slow[i]
}

// manage handles an open long, exiting in precedence SL -> EMAX -> RSI70 -> RSI50 -> EOD. SL
// is read from the position (frozen at entry), never recomputed. RSI70 fires on an UPWARD
// cross of RSIUpper (profit into overbought); RSI50 fires on a DOWNWARD cross of RSIMid and is
// the ONLY exit gated by the cooldown. EMAX/RSI70/RSI50/EOD fill at the bar close.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low := md.Lows[i]
	closeP := md.Closes[i]

	// 1. hard stop (always active).
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}

	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	fast, slow, ok := s.emaPair(md.Closes)

	// 2. EMA cross down (always active).
	if ok && emaCrossDown(fast, slow, i) {
		sig.Kind, sig.Reason = model.SignalSell, "EMAX"
		sig.ExitReason = fmt.Sprintf("EMAX: EMA(%d) пересекла EMA(%d) сверху вниз, выход по %.4f (вход %.4f)",
			s.p.EMAFast, s.p.EMASlow, closeP, pos.PurchasePrice)
		return sig
	}
	// 3. RSI cross UP through the upper level — profit into overbought (always active).
	if len(rsi) == n && crossedUp(rsi, i, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI70"
		sig.ExitReason = fmt.Sprintf("RSI70: RSI(%d) пересёк %.0f снизу вверх, выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, closeP, pos.PurchasePrice)
		return sig
	}
	// 4. RSI cross DOWN through the mid level (gated by the post-entry cooldown).
	if len(rsi) == n && crossedDown(rsi, i, s.p.RSIMid) && s.barsSinceEntry(md) >= s.p.EntryCooldownBars {
		sig.Kind, sig.Reason = model.SignalSell, "RSI50"
		sig.ExitReason = fmt.Sprintf("RSI50: RSI(%d) пересёк %.0f сверху вниз, выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIMid, closeP, pos.PurchasePrice)
		return sig
	}
	// 5. end of day (always active).
	if s.isDayEnd(s.barTime(md), barSpanMinutes(md.Times)) {
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
	fmt.Fprintf(&sb, "сессия: %v (бар %v); конец дня? %v\n", s.inSession(barT), barT, s.isDayEnd(barT, span))

	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n {
		fmt.Fprintf(&sb, "RSI(%d) пред %.1f тек %.1f; вход-крест вверх %.0f? %v; выход RSI70 вверх %.0f? %v; выход RSI50 вниз %.0f? %v\n",
			s.p.RSIPeriod, rsi[i-1], rsi[i], s.p.RSIMid, crossedUp(rsi, i, s.p.RSIMid),
			s.p.RSIUpper, crossedUp(rsi, i, s.p.RSIUpper), s.p.RSIMid, crossedDown(rsi, i, s.p.RSIMid))
	} else {
		sb.WriteString("RSI: недостаточно истории\n")
	}

	fast, slow, ok := s.emaPair(md.Closes)
	if ok {
		fmt.Fprintf(&sb, "EMA(%d) %.4f vs EMA(%d) %.4f: быстрая выше? %v; крест вниз? %v\n",
			s.p.EMAFast, fast[i], s.p.EMASlow, slow[i], fast[i] > slow[i], emaCrossDown(fast, slow, i))
	} else {
		sb.WriteString("EMA: не прогрето\n")
	}

	if len(rsi) == n {
		fmt.Fprintf(&sb, "фильтр свежести: баров выше %.0f в окне %d = %d (лимит %s); прошёл? %v\n",
			s.p.RSIMid, s.p.EntryLookbackBars, s.barsAboveMid(rsi, i),
			limitLabel(s.p.EntryAboveMidLimit), s.freshByBarsAbove(rsi, i))
		fmt.Fprintf(&sb, "фильтр пилы: пересечений %.0f в окне %d = %d (лимит %s); прошёл? %v\n",
			s.p.RSIMid, s.p.EntryLookbackBars, s.midCrossings(rsi, i),
			limitLabel(s.p.EntryMaxMidCrossings), s.freshByCrossings(rsi, i))
	}

	switch {
	case s.p.UseVolume != 1:
		sb.WriteString("фон объёмов: выключен (UseVolume=0)\n")
	default:
		shortAvg, okShort := avgVolumeLastN(md.Volumes, md.Times, s.p.VolShortPeriod)
		longAvg, okLong := avgVolumeLastN(md.Volumes, md.Times, s.p.VolLongPeriod)
		if okShort && okLong && longAvg > 0 {
			fmt.Fprintf(&sb, "фон объёмов: short(%d) %.0f vs long(%d) %.0f, отношение %.2f, порог %.2f; прошёл? %v\n",
				s.p.VolShortPeriod, shortAvg, s.p.VolLongPeriod, longAvg,
				shortAvg/longAvg, s.p.VolMult, s.volumeRegimeOK(md))
		} else {
			sb.WriteString("фон объёмов: нет данных → гейт пропущен\n")
		}
	}

	if s.p.StopATR > 0 {
		atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		fmt.Fprintf(&sb, "ATR(%d) %.4f; стоп %.4f (вход − %.2f×ATR)\n",
			s.p.ATRPeriod, atr, md.Closes[i]-s.p.StopATR*atr, s.p.StopATR)
	} else {
		sb.WriteString("стоп: выключен (StopATR=0)\n")
	}

	if md.Position != nil {
		bse := s.barsSinceEntry(md)
		fmt.Fprintf(&sb, "баров с входа %d; кулдаун RSI50 пройден (≥%d)? %v\n", bse, s.p.EntryCooldownBars, bse >= s.p.EntryCooldownBars)
	}
	return sb.String()
}
