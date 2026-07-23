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
	"sort"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// defaultBarSpanMin is the bar length in minutes assumed when the series carries no usable
// open-times (live paths without Times). The reference timeframe is 15 minutes, but the EOD
// gate reads the actual span from the data via barSpanMinutes.
const defaultBarSpanMin = 5

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
	SessionEndMin     int     // Mon-Thu session end, minutes from MSK midnight (1080 = 18:00)
	FridayEndMin      int     // Friday session end, minutes from MSK midnight (840 = 14:00)
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
// that still ends inside the session (or already sits outside it). A zero time degrades the
// EOD exit to a no-op.
func (s *Strategy) isDayEnd(t time.Time, spanMin int) bool {
	if t.IsZero() {
		return false
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return true
	}
	m := tl.Hour()*60 + tl.Minute()
	return m+spanMin >= s.sessionEndMin(tl)
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

// manage is implemented in Task 3.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal { return sig }
