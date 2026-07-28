// Package core implements a long-only intraday mean-reversion strategy anchored to the
// session VWAP. When flat it buys a close that has fallen EntryK sigmas below the running
// session VWAP — provided the move is large enough in percent to clear round-trip costs, the
// bar did not close on its low, and the daily trend is up. The target is the VWAP itself. The
// decision logic is pure, stateless between bars and ticker-agnostic. The reference timeframe
// is 30 minutes. Run with `-strategy vwap_rev -interval Minutes30`.
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
// open-times. It matches this strategy's 30-minute reference timeframe; the real span is read
// from the data via barSpanMinutes whenever Times is present.
const defaultBarSpanMin = 30

// minLookback is the floor on the candle window handed to Decide. The session VWAP anchor is
// only correct when the current day's opening bar sits inside the window, and the indicator
// discards the window's first session outright, so the window must cover at least two full
// sessions. 300 bars clears that on 15-minute bars and coarser; this strategy is not meant for
// anything finer.
const minLookback = 300

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	EntryK          float64 // entry requires Close <= VWAP - EntryK*sigma (grid; default 1.5)
	MaxDevK         float64 // reject when Close < VWAP - MaxDevK*sigma; <=0 disables (grid; default 4)
	MinEdgePct      float64 // entry requires (VWAP-Close)/Close*100 >= this, IN PERCENT (grid; default 0.35)
	MinBarsFromOpen int     // entry requires barsFromOpen >= this (grid; default 6)
	MinClosePos     float64 // entry requires Close >= Low + MinClosePos*(High-Low), a 0..1 fraction; <=0 disables (grid; default 0.33)
	UseDailyTrend   int     // 1 arms the daily EMA trend gate; any other value disables it (grid; default 1)
	DailyEMAPeriod  int     // daily EMA period for the trend gate (grid; default 50)
	StopATR         float64 // stop = entry - StopATR*ATR; 0 disables the stop (grid; default 1.2)
	ATRPeriod       int     // ATR length; used only when StopATR>0
	MaxHoldBars     int     // TIME exit after this many bars held; <=0 disables (grid; default 8)
	SessionStartMin int     // entry window start, minutes from MSK midnight (420 = 07:00)
	SessionEndMin   int     // Mon-Thu entry cutoff, minutes from MSK midnight (1080 = 18:00)
	FridayEndMin    int     // Friday entry cutoff, minutes from MSK midnight (840 = 14:00)
	DayEndMin       int     // day-end force-close boundary, minutes from MSK midnight (1380 = 23:00)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		EntryK:          1.5,
		MaxDevK:         4.0,
		MinEdgePct:      0.35,
		MinBarsFromOpen: 6,
		MinClosePos:     0.33,
		UseDailyTrend:   1,
		DailyEMAPeriod:  50,
		StopATR:         1.2,
		ATRPeriod:       14,
		MaxHoldBars:     8,
		SessionStartMin: 420,
		SessionEndMin:   1080,
		FridayEndMin:    840,
		DayEndMin:       1380,
	}
}

// Strategy trades a single instrument with the VWAP-reversion rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window the engine feeds Decide on every bar. See minLookback for
// why two full sessions are the binding requirement; the ATR term only matters if someone
// calibrates an unusually long ATR.
func (s *Strategy) Lookback() int {
	return max(minLookback, 2*s.p.ATRPeriod+20)
}

// mskLoc anchors the session windows and the VWAP anchor to the Moscow calendar (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// sessionEndMin returns the entry cutoff for the weekday of tl (Friday closes early).
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
// before the day-end force-close boundary. A zero time degrades the EOD exit to a no-op.
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
// between consecutive bars (robust to session and weekend jumps).
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

// sessionVWAP recomputes the session anchor from md. Returns ok=false whenever the anchor
// cannot be trusted at the last bar.
func (s *Strategy) sessionVWAP(md strategy.MarketData) (vwap, sigma []float64, bfo []int, ok bool) {
	n := len(md.Closes)
	vwap, sigma, bfo = indicators.SessionVWAP(md.Highs, md.Lows, md.Closes, md.Volumes, md.Times, mskLoc)
	if vwap == nil || len(vwap) != n {
		return nil, nil, nil, false
	}
	return vwap, sigma, bfo, true
}

// closePosOK reports whether the bar closed clear of its own low — a falling-knife filter. A
// zero-range bar carries no position information and is allowed through. Off when
// MinClosePos<=0.
func (s *Strategy) closePosOK(high, low, closeP float64) bool {
	if s.p.MinClosePos <= 0 {
		return true
	}
	rng := high - low
	if rng <= 0 {
		return true
	}
	return closeP >= low+s.p.MinClosePos*rng
}

// dailyTrendOK reports whether the daily trend allows the entry. Unlike most gates in this
// repo it FAILS CLOSED: with the gate armed and no usable daily series, the entry is rejected
// rather than allowed. Buying a deep intraday dip without knowing the daily trend is exactly
// the trade this strategy must not take.
func (s *Strategy) dailyTrendOK(dailyCloses []float64, price float64) bool {
	if s.p.UseDailyTrend != 1 {
		return true
	}
	if s.p.DailyEMAPeriod <= 0 || len(dailyCloses) < s.p.DailyEMAPeriod {
		return false
	}
	e := ema.Compute(dailyCloses, s.p.DailyEMAPeriod)
	if len(e) == 0 {
		return false
	}
	last := e[len(e)-1]
	return last > 0 && price > last
}

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter emits a long when the close sits deep enough below the session VWAP, the move is
// worth more than the round-trip cost, the bar did not close on its low and the daily trend
// is up. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. entry window, and never on the day-end bar (manage() only runs from the NEXT bar, so
	// an entry on the day-end bar could not be EOD-closed on its own bar).
	t := s.barTime(md)
	if !s.inSession(t) || s.isDayEnd(t, barSpanMinutes(md.Times)) {
		return sig
	}
	i := n - 1
	// 2. session anchor must be warmed and the session complete inside the window.
	vwap, sigma, bfo, ok := s.sessionVWAP(md)
	if !ok || vwap[i] <= 0 || sigma[i] <= 0 || bfo[i] < s.p.MinBarsFromOpen {
		return sig
	}
	closeP := md.Closes[i]
	if closeP <= 0 {
		return sig
	}
	dev := vwap[i] - closeP
	// 3. deep enough below the anchor, but not a collapse.
	if dev < s.p.EntryK*sigma[i] {
		return sig
	}
	if s.p.MaxDevK > 0 && dev > s.p.MaxDevK*sigma[i] {
		return sig
	}
	// 4. the move to the target must clear round-trip costs.
	if dev/closeP*100 < s.p.MinEdgePct {
		return sig
	}
	// 5. not a falling knife.
	if !s.closePosOK(md.Highs[i], md.Lows[i], closeP) {
		return sig
	}
	// 6. daily trend (fails closed).
	if !s.dailyTrendOK(md.DailyCloses, closeP) {
		return sig
	}
	// 7. optional ATR stop.
	var stop, atr float64
	if s.p.StopATR > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		if atr <= 0 {
			return sig
		}
		stop = closeP - s.p.StopATR*atr
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = vwap[i]
	sig.ATR = atr
	sig.EntryReason = s.entryReason(vwap[i], sigma[i], closeP, stop, atr)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(vwapNow, sigmaNow, entry, stop, atr float64) string {
	stopHow := "стоп выключен (StopATR=0)"
	if s.p.StopATR > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (вход − %.2f×ATR, ATR=%.4f)", stop, s.p.StopATR, atr)
	}
	dev := vwapNow - entry
	return fmt.Sprintf(
		"цена %.4f ниже VWAP %.4f на %.4f (%.2f×σ, σ=%.4f; %.2f%% хода до цели); %s",
		entry, vwapNow, dev, dev/sigmaNow, sigmaNow, dev/entry*100, stopHow,
	)
}

// barsHeld counts bars from the position's entry to the current bar, purely from EntryTime,
// the current bar time and the data-inferred span. Returns -1 when either time is unknown, so
// the TIME exit degrades to "do not fire" instead of closing the position on its first managed
// bar. Positions never survive the EOD close, so the window is always inside one session and
// the uniform span is exact.
func (s *Strategy) barsHeld(md strategy.MarketData) int {
	pos := md.Position
	t := s.barTime(md)
	if pos == nil || pos.EntryTime.IsZero() || t.IsZero() {
		return -1
	}
	span := barSpanMinutes(md.Times)
	if span <= 0 {
		return -1
	}
	return int(math.Round(t.Sub(pos.EntryTime).Minutes() / float64(span)))
}

// manage handles an open long, exiting in precedence SL -> TP -> TIME -> EOD.
//
// SL is read from the position (frozen at entry) and is checked BEFORE the target: the
// intrabar order of the two is unknown, so the worst case is assumed. The target is the
// PREVIOUS bar's session VWAP — the level a resting limit order could have carried into this
// bar; using the current bar's VWAP would be look-ahead. SL fills at the stop (gap-adjusted by
// the engine), TP at max(target, open), TIME and EOD at the bar close.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	high, low, closeP := md.Highs[i], md.Lows[i], md.Closes[i]

	// 1. hard stop (always active, checked first by design).
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}
	// 2. target — the previous bar's VWAP, reachable intrabar.
	if vwap, _, _, ok := s.sessionVWAP(md); ok && vwap[i-1] > 0 && high >= vwap[i-1] {
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.TakeProfit = vwap[i-1]
		sig.ExitReason = fmt.Sprintf("TP: high %.4f достиг VWAP предыдущего бара %.4f (вход %.4f)",
			high, vwap[i-1], pos.PurchasePrice)
		return sig
	}
	// 3. time stop: a reversion that has not happened by now probably will not.
	if s.p.MaxHoldBars > 0 {
		if held := s.barsHeld(md); held >= 0 && held >= s.p.MaxHoldBars {
			sig.Kind, sig.Reason = model.SignalSell, "TIME"
			sig.ExitReason = fmt.Sprintf("TIME: удержание %d баров ≥ %d, выход по %.4f (вход %.4f)",
				held, s.p.MaxHoldBars, closeP, pos.PurchasePrice)
			return sig
		}
	}
	// 4. end of day (always active).
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
	fmt.Fprintf(&sb, "сессия: вход разрешён? %v (бар %v); конец дня? %v\n",
		s.inSession(barT), barT, s.isDayEnd(barT, span))

	vwap, sigma, bfo, ok := s.sessionVWAP(md)
	if !ok || vwap[i] <= 0 || sigma[i] <= 0 {
		sb.WriteString("VWAP: не прогрет (нет времён, нет объёма или окно невалидно)\n")
		return sb.String()
	}
	closeP := md.Closes[i]
	dev := vwap[i] - closeP
	fmt.Fprintf(&sb, "бар в сессии №%d (нужно ≥ %d); VWAP %.4f, σ %.4f\n",
		bfo[i], s.p.MinBarsFromOpen, vwap[i], sigma[i])
	fmt.Fprintf(&sb, "отклонение вниз %.4f = %.2f×σ; порог входа %.2f×σ? %v; не глубже %.2f×σ? %v\n",
		dev, dev/sigma[i], s.p.EntryK, dev >= s.p.EntryK*sigma[i],
		s.p.MaxDevK, s.p.MaxDevK <= 0 || dev <= s.p.MaxDevK*sigma[i])
	fmt.Fprintf(&sb, "ход до цели %.2f%%; MinEdgePct %.2f%%? %v\n",
		dev/closeP*100, s.p.MinEdgePct, dev/closeP*100 >= s.p.MinEdgePct)
	fmt.Fprintf(&sb, "закрытие не в нижней части бара (порог %.2f)? %v\n",
		s.p.MinClosePos, s.closePosOK(md.Highs[i], md.Lows[i], closeP))
	if s.p.UseDailyTrend != 1 {
		sb.WriteString("дневной тренд: гейт выключен (UseDailyTrend=0)\n")
	} else {
		fmt.Fprintf(&sb, "дневной тренд: цена выше EMA(%d) по дневкам (дней %d)? %v\n",
			s.p.DailyEMAPeriod, len(md.DailyCloses), s.dailyTrendOK(md.DailyCloses, closeP))
	}
	if s.p.StopATR > 0 {
		atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		fmt.Fprintf(&sb, "ATR(%d) %.4f; стоп %.4f (вход − %.2f×ATR)\n",
			s.p.ATRPeriod, atr, closeP-s.p.StopATR*atr, s.p.StopATR)
	} else {
		sb.WriteString("стоп: выключен (StopATR=0)\n")
	}
	if md.Position != nil {
		fmt.Fprintf(&sb, "позиция открыта: удержано баров %d (лимит %d); цель = VWAP пред. бара %.4f\n",
			s.barsHeld(md), s.p.MaxHoldBars, vwap[i-1])
	}
	return sb.String()
}
