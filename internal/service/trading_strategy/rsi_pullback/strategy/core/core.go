// Package core implements a long-only multi-day RSI pullback strategy. When flat it buys the dip
// inside an uptrend: the fast EMA must sit above the slow one and a short RSI must cross DOWN
// through its lower band on the current bar. The stop and target are sized off the daily ATR at
// entry and frozen on the position; the trade is closed on the first of: the stop, the target, or
// RSI crossing UP through the upper band. There is no time stop and no end-of-day close — the
// position is held across nights and weekends until one of those three exits fires. The decision
// logic is pure, stateless between bars and ticker-agnostic. The reference timeframe is 30
// minutes. Run with `-strategy rsi_pullback -interval Minutes30`.
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
	DailyATRPeriod  int     // daily ATR length, over WEEKDAY completed dailies (grid; default 14)
	UseDayATRGate   int     // 1 arms the two-sided day gate; any other value disables it (grid; default 1)
	FreshDayATR     float64 // "day barely started": range so far <= FreshDayATR*dailyATR (grid; default 0.3)
	SpentDayATR     float64 // "day spent": range so far >= SpentDayATR*dailyATR (grid; default 0.8)
	StopDailyATR    float64 // stop = entry - StopDailyATR*dailyATR; 0 disables it (grid; never 0 in the grid)
	TPDailyATR      float64 // target = entry + TPDailyATR*dailyATR; 0 disables it (grid)
	SessionStartMin int     // entry window start, minutes from MSK midnight (420 = 07:00)
	SessionEndMin   int     // entry window end, minutes from MSK midnight (1020 = 17:00)
	UseVolume       int     // 1 arms the volume-background gate; any other value disables it (grid; default 1)
	VolBaseDays     int     // completed WEEKDAY days behind the baseline (grid; default 5)
	VolLookbackBars int     // how many recent weekday bars may open the gate (grid; default 3)
	VolMult         float64 // a bar opens the gate at volume >= VolMult * its slot baseline (grid; default 1.5)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:       4,
		RSILower:        15,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    1.0,
		TPDailyATR:      0.6,
		SessionStartMin: 420,
		SessionEndMin:   1020,
		UseVolume:       1,
		VolBaseDays:     5,
		VolLookbackBars: 3,
		VolMult:         1.5,
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
// as the seed span; the +20 covers the two-bar cross lookups. When the volume gate is armed the
// window must additionally hold VolBaseDays completed WEEKDAY days plus the current one, which
// on 30-minute bars dominates everything else. Sizing that purely off calendar days (VolBaseDays
// * maxBarsPerDay) undercounts: on MOEX 30-minute data roughly a third of all bars fall on a
// Saturday or Sunday (measured on GAZP), so a calendar window that size can contain fewer than
// VolBaseDays weekday days once the weekend bars inside it are discounted — the baseline then
// silently shrinks instead of failing loudly. Scaling by 7/5 accounts for the two weekend days
// riding along with every five weekday ones.
func (s *Strategy) Lookback() int {
	need := max(s.p.EMASlow, s.p.EMAFast, s.p.RSIPeriod)
	vol := 0
	if s.p.UseVolume == 1 && s.p.VolBaseDays > 0 {
		vol = (s.p.VolBaseDays + 1) * maxBarsPerDay * 7 / 5
	}
	return max(minLookback, 2*need+20, vol)
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

// weekdayDaily drops weekend (Sat/Sun MSK) bars from the daily series, keeping the three
// price slices aligned. MOEX runs weekend sessions and the candle cache contains them, but
// those bars are 3-4x narrower and 8-17x thinner than weekday ones: leaving them in
// understates the daily ATR by 9-16% (measured on GAZP/SBER/NVTK/RUAL), and that error would
// propagate into the stop, the target and both thresholds of the day gate at once. When times
// is empty or not aligned with the price slices there is nothing to filter by — the series is
// returned untouched rather than guessed at.
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
// The engine only ever exposes days that closed before the current bar, so no lookahead is
// possible here. Returns 0 whenever the data cannot support the calculation — the caller must
// then refuse the entry, because without it there is neither a stop nor a target.
func (s *Strategy) dailyATR(md strategy.MarketData) float64 {
	if s.p.DailyATRPeriod <= 0 {
		return 0
	}
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if len(c) < s.p.DailyATRPeriod+1 {
		return 0
	}
	return indicators.ATR(h, l, c, s.p.DailyATRPeriod)
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

// dayStateOK reports whether the current day is in one of the two states this strategy trades.
// Either the day has barely started — its range so far is within FreshDayATR of the daily ATR,
// so the whole move is still ahead — or the day is spent: the range has already reached
// SpentDayATR, the sell-off has happened and a further leg down is less likely. The band
// between the two is refused: there the day has moved meaningfully but is neither fresh nor
// exhausted. The gate skips itself — never blocks — when disabled, when both thresholds are
// non-positive, when the thresholds overlap (FreshDayATR >= SpentDayATR makes every day pass
// anyway) or when the data cannot support it.
func (s *Strategy) dayStateOK(md strategy.MarketData, atr float64) bool {
	if s.p.UseDayATRGate != 1 || atr <= 0 {
		return true
	}
	if md.TodayHigh <= 0 || md.TodayLow <= 0 || md.TodayHigh < md.TodayLow {
		return true
	}
	fresh, spent := s.p.FreshDayATR, s.p.SpentDayATR
	if fresh <= 0 && spent <= 0 {
		return true
	}
	used := md.TodayHigh - md.TodayLow
	return (fresh > 0 && used <= fresh*atr) || (spent > 0 && used >= spent*atr)
}

// maxBarsPerDay caps how many 30-minute bars a single calendar day can contribute (24h / 30m).
// It is a per-day bar count, not a per-day calendar allowance: Lookback multiplies it by a 7/5
// factor to also cover the weekend bars that ride along with every window of weekday days (see
// Lookback for why a plain calendar count silently undersizes the window).
const maxBarsPerDay = 48

// dayOf returns midnight of t's MSK calendar day — the grouping key for baseline days.
func dayOf(t time.Time) time.Time {
	tl := t.In(mskLoc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, mskLoc)
}

// slotOf returns the bar's intraday slot: minutes from MSK midnight. Bars sharing a slot are
// the same half-hour of the trading day across different days.
func slotOf(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Hour()*60 + tl.Minute()
}

// volumeBaseline builds the per-slot average volume over the last baseDays COMPLETED WEEKDAY
// days present in the window, plus a flat average over the same bars as a fallback for slots
// with fewer than two observations. The current day is excluded entirely, so the bars being
// judged never contaminate their own baseline and the baseline does not drift from bar to bar
// within a day. Weekend sessions are excluded: on MOEX they carry 8-17x less volume and would
// drag the baseline down, turning the gate into a free pass. Non-positive volumes are ignored.
// ok is false when no usable bar was found at all.
func volumeBaseline(vols []int64, times []time.Time, baseDays int) (bySlot map[int]float64, flat float64, ok bool) {
	n := len(vols)
	if n == 0 || len(times) != n || baseDays <= 0 {
		return nil, 0, false
	}
	current := dayOf(times[n-1])
	sums := make(map[int]float64)
	counts := make(map[int]int)
	var flatSum float64
	var flatCount int
	var lastDay time.Time
	days := 0
	for i := n - 1; i >= 0; i-- {
		t := times[i]
		if isWeekend(t.In(mskLoc)) {
			continue
		}
		d := dayOf(t)
		if !d.Before(current) {
			continue
		}
		if !d.Equal(lastDay) {
			if days == baseDays {
				break
			}
			days++
			lastDay = d
		}
		if vols[i] <= 0 {
			continue
		}
		sl := slotOf(t)
		sums[sl] += float64(vols[i])
		counts[sl]++
		flatSum += float64(vols[i])
		flatCount++
	}
	if flatCount == 0 {
		return nil, 0, false
	}
	bySlot = make(map[int]float64, len(sums))
	for sl, c := range counts {
		if c >= 2 {
			bySlot[sl] = sums[sl] / float64(c)
		}
	}
	return bySlot, flatSum / float64(flatCount), true
}

// volumeOK reports whether the recent tape is busier than usual FOR THIS TIME OF DAY: at least
// one of the last VolLookbackBars weekday bars must carry VolMult times the average volume of
// its own intraday slot. Comparing against a slot rather than a flat average matters because
// 30-minute volume is U-shaped — an opening bar dwarfs a midday one — so a flat baseline would
// measure the clock instead of the activity. The gate degrades to "allow" whenever it is
// disabled, misconfigured or unsupported by the data: missing volume must never block an entry.
func (s *Strategy) volumeOK(md strategy.MarketData) bool {
	if s.p.UseVolume != 1 || s.p.VolBaseDays <= 0 || s.p.VolLookbackBars <= 0 || s.p.VolMult <= 0 {
		return true
	}
	n := len(md.Volumes)
	if n == 0 || len(md.Times) != n {
		return true
	}
	bySlot, flat, ok := volumeBaseline(md.Volumes, md.Times, s.p.VolBaseDays)
	if !ok || flat <= 0 {
		return true
	}
	checked := 0
	for i := n - 1; i >= 0 && checked < s.p.VolLookbackBars; i-- {
		if isWeekend(md.Times[i].In(mskLoc)) {
			continue
		}
		if md.Volumes[i] <= 0 {
			// A non-positive volume is missing data, not a quiet bar: it must never consume a
			// slot in the lookback window (which would let a run of broken readings starve the
			// gate down to "no bar to judge" instead of reaching further back for a real one).
			continue
		}
		checked++
		base, hasSlot := bySlot[slotOf(md.Times[i])]
		if !hasSlot || base <= 0 {
			base = flat
		}
		if float64(md.Volumes[i]) >= base*s.p.VolMult {
			return true
		}
	}
	// No weekday bar to judge at all — allow, same as any other missing-data case.
	return checked == 0
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
// bounded: the exit is only delayed, SL/TP stay armed. Dropping the guard would instead
// let an RSISeries warm-up zero manufacture an exit out of nothing. Pinned by
// TestCrossHelpersBoundaries.
func crossedUp(series []float64, i int, level float64) bool {
	return i >= 1 && series[i-1] > 0 && series[i-1] <= level && series[i] > level
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
// while the fast EMA sits above the slow one, the day is either fresh or spent, and the tape
// is busy. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. entry window.
	if !s.inSession(s.barTime(md)) {
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
	// 4. the daily ATR is the unit of both the stop and the target: no ATR, no trade.
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	// 5. the day must be either fresh or spent.
	if !s.dayStateOK(md, atr) {
		return sig
	}
	// 6. the tape must be busier than usual for this time of day.
	if !s.volumeOK(md) {
		return sig
	}
	entry := md.Closes[i]
	var stop, target float64
	if s.p.StopDailyATR > 0 {
		stop = entry - s.p.StopDailyATR*atr
	}
	// A stop that lands at or below zero is not a floor, it is a naked long: entry minus the
	// stop distance ate through the whole price. manage() only ever checks pos.StopLoss > 0, so
	// a non-positive stop here would silently hold the position across nights and weekends with
	// no protective exit at all. Refuse the entry outright rather than let that through armed
	// with a target and an RSI exit only.
	if s.p.StopDailyATR > 0 && stop <= 0 {
		return sig
	}
	if s.p.TPDailyATR > 0 {
		target = entry + s.p.TPDailyATR*atr
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = target
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i], fast[i], slow[i], entry, stop, target, atr, md)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(rsiNow, fastNow, slowNow, entry, stop, target, atr float64, md strategy.MarketData) string {
	stopHow := "стоп выключен"
	if stop > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (−%.2f ATR)", stop, s.p.StopDailyATR)
	}
	tpHow := "цель выключена"
	if target > 0 {
		tpHow = fmt.Sprintf("цель %.4f (+%.2f ATR)", target, s.p.TPDailyATR)
	}
	dayHow := "гейт дня выключен"
	if s.p.UseDayATRGate == 1 && md.TodayHigh > 0 && md.TodayLow > 0 && md.TodayHigh >= md.TodayLow && atr > 0 {
		dayHow = fmt.Sprintf("день прошёл %.2f ATR", (md.TodayHigh-md.TodayLow)/atr)
	}
	return fmt.Sprintf(
		"RSI(%d) ушёл под %.0f (%.1f) на откате, EMA(%d) %.4f > EMA(%d) %.4f, %s (дневной ATR %.4f); вход %.4f, %s, %s",
		s.p.RSIPeriod, s.p.RSILower, rsiNow, s.p.EMAFast, fastNow, s.p.EMASlow, slowNow,
		dayHow, atr, entry, stopHow, tpHow,
	)
}

// manage handles an open long, exiting in precedence SL -> TP -> RSI. Both levels are read
// from the position (frozen at entry), never recomputed: the stop and target the trade was
// opened with are the ones it dies by. SL fills at the stop level and TP at the target (the
// engine handles that via model.IsStopReason and the "TP" reason); RSI fills at the bar close.
// There is no time stop and no end-of-day close — the position is held until one of the three
// exits fires, across nights and weekends.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	high, low, closeP := md.Highs[i], md.Lows[i], md.Closes[i]

	// 1. hard stop. It wins a same-bar tie with the target: the intrabar order of the two
	// touches is unknowable from OHLC, and assuming the worse of the two is the honest choice.
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}
	// 2. fixed target.
	if pos.TakeProfit > 0 && high >= pos.TakeProfit {
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.TakeProfit = pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цель %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
		return sig
	}
	// 3. RSI crosses UP through the upper band — the bounce reached overbought.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n && crossedUp(rsi, i, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) пересёк %.0f снизу вверх (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, rsi[i], closeP, pos.PurchasePrice)
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
	fmt.Fprintf(&sb, "сессия: вход разрешён? %v (бар %v)\n", s.inSession(barT), barT)

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

	atr := s.dailyATR(md)
	if atr > 0 {
		fmt.Fprintf(&sb, "дневной ATR(%d) по будням: %.4f\n", s.p.DailyATRPeriod, atr)
	} else {
		sb.WriteString("дневной ATR: не посчитан — вход невозможен\n")
	}

	switch {
	case s.p.UseDayATRGate != 1:
		sb.WriteString("состояние дня: гейт выключен (UseDayATRGate=0)\n")
	case atr <= 0 || md.TodayHigh <= 0 || md.TodayLow <= 0 || md.TodayHigh < md.TodayLow:
		sb.WriteString("состояние дня: нет данных, гейт пропускает\n")
	default:
		used := (md.TodayHigh - md.TodayLow) / atr
		fmt.Fprintf(&sb, "состояние дня: пройдено %.2f ATR (свежий ≤%.2f, исчерпан ≥%.2f); пройден? %v\n",
			used, s.p.FreshDayATR, s.p.SpentDayATR, s.dayStateOK(md, atr))
	}

	if s.p.UseVolume != 1 {
		sb.WriteString("фон объёмов: гейт выключен (UseVolume=0)\n")
	} else {
		fmt.Fprintf(&sb, "фон объёмов: хотя бы один из %d баров ≥ %.2f× своего слота за %d дней? %v\n",
			s.p.VolLookbackBars, s.p.VolMult, s.p.VolBaseDays, s.volumeOK(md))
	}

	if s.p.StopDailyATR > 0 && atr > 0 {
		fmt.Fprintf(&sb, "стоп: вход − %.2f×ATR (%.4f)\n", s.p.StopDailyATR, s.p.StopDailyATR*atr)
	} else {
		sb.WriteString("стоп: выключен\n")
	}
	if s.p.TPDailyATR > 0 && atr > 0 {
		fmt.Fprintf(&sb, "цель: вход + %.2f×ATR (%.4f)\n", s.p.TPDailyATR, s.p.TPDailyATR*atr)
	} else {
		sb.WriteString("цель: выключена\n")
	}
	return sb.String()
}
