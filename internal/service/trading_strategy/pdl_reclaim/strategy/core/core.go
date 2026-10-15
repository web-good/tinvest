// Package core implements pdl_reclaim: a long-only intraday strategy on 5-minute bars around
// yesterday's range. Yesterday's low (PDL) is support, yesterday's high (PDH) the target. It buys
// the next bar's open after a green bar that dipped below PDL and closed back above it (entry A),
// or after a bullish engulfing whose red bar closed below PDL (entry B). Exits, in priority order:
// a stop StopDailyATR daily ATRs below the fill, PDH, RSI(RSIPeriod) at or above RSIExit, the
// first red MACD histogram bar after a green one inside the trade, and the end of day. Pure,
// stateless between bars and ticker-agnostic. Run with `-strategy pdl_reclaim -interval Minutes5`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// dayBars is the number of 5-minute bars from 07:00 to the 18:30 exit bar inclusive: an open
// position is never older than that, so the window always reaches back to its entry bar.
const dayBars = 139

// minWarmup keeps the indicator warm-up sane when the periods are tiny.
const minWarmup = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	EntryA          int     // 1 — false break of PDL (green bar, low < PDL < close) enters
	EntryB          int     // 1 — bullish engulfing under PDL enters
	StopDailyATR    float64 // stop below the fill in daily ATR; 0 — no entries (no risk unit)
	DailyATRPeriod  int     // Wilder period over completed weekday dailies
	TakeProfitPDH   int     // 1 — target at yesterday's high; 0 — no target
	RSIPeriod       int     // 5m RSI for the exit; 0 — RSI exit off
	RSIExit         float64 // RSI at/above this sells; 0 — RSI exit off
	MACDFast        int     // 5m MACD for the exit; invalid periods — MACD exit off
	MACDSlow        int
	MACDSignal      int
	EntryFromHour   int // the signal bar starts at/after EntryFrom MSK
	EntryFromMinute int
	EntryToHour     int // the buy bar starts before EntryTo MSK, on the signal's date
	EntryToMinute   int
	ExitHour        int // the first bar starting at/after Exit MSK sells at its close
	ExitMinute      int
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		EntryA:          1,
		EntryB:          1,
		StopDailyATR:    0.5,
		DailyATRPeriod:  14,
		TakeProfitPDH:   1,
		RSIPeriod:       6,
		RSIExit:         70,
		MACDFast:        12,
		MACDSlow:        26,
		MACDSignal:      9,
		EntryFromHour:   7,
		EntryFromMinute: 0,
		EntryToHour:     18,
		EntryToMinute:   0,
		ExitHour:        18,
		ExitMinute:      30,
	}
}

// Strategy trades a single instrument with the pdl_reclaim rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams returns the strategy for a ticker; step is the bar length (5m).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback warms MACD and RSI before the oldest possible entry bar of a held position.
func (s *Strategy) Lookback() int {
	return max(minWarmup, 3*(s.p.MACDSlow+s.p.MACDSignal), 4*s.p.RSIPeriod+2) + dayBars
}

// mskLoc anchors every calendar rule to Moscow (UTC fallback).
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

// dayKey is the MSK calendar date of t as yyyymmdd.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
}

// minuteOf is the MSK minute of the day of t.
func minuteOf(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Hour()*60 + tl.Minute()
}

// weekdayDaily drops weekend (Sat/Sun MSK) dailies, keeping the price slices aligned. When times
// is empty or misaligned the series is returned untouched rather than guessed at.
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

// levels returns yesterday's low and high: the last completed weekday daily.
func (s *Strategy) levels(md strategy.MarketData) (pdl, pdh float64, ok bool) {
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	n := len(c)
	if n == 0 || len(h) != n || len(l) != n {
		return 0, 0, false
	}
	return l[n-1], h[n-1], l[n-1] > 0 && h[n-1] >= l[n-1]
}

// dailyATR is Wilder's ATR over completed weekday dailies; 0 when the data cannot support it.
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

// inEntryWindow reports whether a signal on the bar starting at t may buy: a weekday, the bar
// starts at/after EntryFrom, and the buy bar (t+step) starts before EntryTo on the same date.
func (s *Strategy) inEntryWindow(t time.Time) bool {
	if isWeekend(t.In(mskLoc)) || minuteOf(t) < s.p.EntryFromHour*60+s.p.EntryFromMinute {
		return false
	}
	fill := t.Add(s.step)
	return dayKey(fill) == dayKey(t) && minuteOf(fill) < s.p.EntryToHour*60+s.p.EntryToMinute
}

// isA: bar i is green, dipped below PDL and closed back above it.
func isA(md strategy.MarketData, i int, pdl float64) bool {
	return md.Closes[i] > md.Opens[i] && md.Lows[i] < pdl && md.Closes[i] > pdl
}

// isB: bar i-1 is red and closed below PDL; bar i, on the same date, is green, closes above the
// red bar's open and trades below its low.
func isB(md strategy.MarketData, i int, pdl float64) bool {
	if i < 1 || dayKey(md.Times[i]) != dayKey(md.Times[i-1]) {
		return false
	}
	o0, c0, l0 := md.Opens[i-1], md.Closes[i-1], md.Lows[i-1]
	o1, c1, l1 := md.Opens[i], md.Closes[i], md.Lows[i]
	return c0 < o0 && c0 < pdl && c1 > o1 && c1 > o0 && l1 < l0
}

// pattern names the entry bar i matches ("A" wins over "B"), or "" for none.
func (s *Strategy) pattern(md strategy.MarketData, i int, pdl float64) string {
	switch {
	case s.p.EntryA > 0 && isA(md, i, pdl):
		return "A"
	case s.p.EntryB > 0 && isB(md, i, pdl):
		return "B"
	}
	return ""
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys the next bar's open on entry A or B. The cheap bar gates run before the daily ATR.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if s.p.StopDailyATR <= 0 {
		return sig
	}
	i := n - 1
	if !s.inEntryWindow(md.Times[i]) {
		return sig
	}
	pdl, pdh, ok := s.levels(md)
	if !ok {
		return sig
	}
	kind := s.pattern(md, i, pdl)
	if kind == "" {
		return sig
	}
	closeP := md.Closes[i]
	if s.p.TakeProfitPDH > 0 && closeP >= pdh {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	stop := closeP - s.p.StopDailyATR*atr
	if stop <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.ATR, sig.Level = stop, atr, pdl
	if s.p.TakeProfitPDH > 0 {
		sig.TakeProfit = pdh
	}
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s: PDL %.4f, PDH %.4f, сигнал по close %.4f, дневной ATR %.4f, стоп %.4f",
		kind, pdl, pdh, closeP, atr, stop)
	return sig
}

// manage exits in priority order: stop, target, RSI, MACD, end of day.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low, high, closeP := md.Lows[i], md.Highs[i], md.Closes[i]
	if stop := s.stop(md, pos); stop > 0 && low <= stop {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
		return sig
	}
	if pos.TakeProfit > 0 && high >= pos.TakeProfit {
		sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ PDH %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
		return sig
	}
	if s.p.RSIPeriod > 0 && s.p.RSIExit > 0 && n > s.p.RSIPeriod {
		if rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)[i]; rsi >= s.p.RSIExit {
			sig.Kind, sig.Reason = model.SignalSell, "RSI"
			sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) %.1f ≥ %.0f, выход по %.4f (вход %.4f)", s.p.RSIPeriod, rsi, s.p.RSIExit, closeP, pos.PurchasePrice)
			return sig
		}
	}
	if hist := histogram(md.Closes, s.p); hist != nil && len(md.Times) == n {
		if macdExit(hist, entryIndex(md.Times, pos.EntryTime), i) {
			sig.Kind, sig.Reason = model.SignalSell, "MACD"
			sig.ExitReason = fmt.Sprintf("MACD: гистограмма %.4f красная после зелёной в сделке, выход по %.4f (вход %.4f)", hist[i], closeP, pos.PurchasePrice)
			return sig
		}
	}
	if why := s.eodReason(md, pos); why != "" {
		sig.Kind, sig.Reason = model.SignalSell, "EOD"
		sig.ExitReason = fmt.Sprintf("EOD: %s, выход по %.4f (вход %.4f)", why, closeP, pos.PurchasePrice)
	}
	return sig
}

// stop is StopDailyATR daily ATRs below the fill, with the ATR frozen at entry (today's when the
// position carries none). 0 — no stop.
func (s *Strategy) stop(md strategy.MarketData, pos *strategy.Position) float64 {
	if s.p.StopDailyATR <= 0 {
		return 0
	}
	atr := pos.EntryATR
	if atr <= 0 {
		atr = s.dailyATR(md)
	}
	if atr <= 0 {
		return 0
	}
	return max(pos.PurchasePrice-s.p.StopDailyATR*atr, 0)
}

// histogram is MACD line minus signal line over closes; nil for invalid periods or short history.
func histogram(closes []float64, p Params) []float64 {
	m, sg := indicators.MACD(closes, p.MACDFast, p.MACDSlow, p.MACDSignal)
	if m == nil {
		return nil
	}
	h := make([]float64, len(m))
	for k := range m {
		h[k] = m[k] - sg[k]
	}
	return h
}

// entryIndex is the index of the bar that opened at entry, or -1 when entry is unset or not in
// the window.
func entryIndex(times []time.Time, entry time.Time) int {
	if entry.IsZero() {
		return -1
	}
	for j := len(times) - 1; j >= 0; j-- {
		if times[j].Equal(entry) {
			return j
		}
		if times[j].Before(entry) {
			return -1
		}
	}
	return -1
}

// macdExit: the histogram is red on bar i and was green (≥ 0) on some bar from the entry bar e
// up to i-1.
func macdExit(hist []float64, e, i int) bool {
	if e < 0 || e >= i || hist[i] >= 0 {
		return false
	}
	for k := e; k < i; k++ {
		if hist[k] >= 0 {
			return true
		}
	}
	return false
}

// eodReason explains why the position must close on the current bar, or returns "" when it may
// stay. It fails closed: no bar times, the first bar of a new MSK date, a bar of a later date
// than the entry, or a bar starting at/after Exit MSK.
func (s *Strategy) eodReason(md strategy.MarketData, pos *strategy.Position) string {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n {
		return "нет времени свечей — выход по безопасности"
	}
	now := md.Times[n-1]
	if dayKey(now) != dayKey(md.Times[n-2]) {
		return "первая свеча новой даты — позиция не переносится через ночь"
	}
	if !pos.EntryTime.IsZero() && dayKey(now) != dayKey(pos.EntryTime) {
		return "свеча следующего дня — позиция не переносится через ночь"
	}
	if minuteOf(now) >= s.p.ExitHour*60+s.p.ExitMinute {
		return fmt.Sprintf("свеча %s не раньше %02d:%02d MSK", now.In(mskLoc).Format("15:04"), s.p.ExitHour, s.p.ExitMinute)
	}
	return ""
}
