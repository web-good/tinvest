// Package core implements pdl_reclaim: a long-only strategy around yesterday's range on
// 15-minute bars. Yesterday's low (PDL) and high (PDH) are the levels. Two entries:
//   - at PDL the reclaim: a green bar closes above PDL after a red bar of the same date closed
//     below it, with no close above PDL in between; the buy comes at the next bar's open right
//     after the reclaim bar;
//   - at PDH the breakout: the first green close above PDH, on volume above VolumeMult × the
//     average volume of the VolBars main-session bars before it (10:00–18:30 MSK, weekdays); the
//     buy comes at the next bar's open once the HoldBars bars after the breakout bar, on its date,
//     all closed at or above PDH.
//
// The target is the setup level plus half of yesterday's range: the middle of the range at PDL,
// PDH + half the range at PDH. Exits, in priority order: a stop StopDailyATR daily ATRs below the
// fill and the target; the position carries overnight until one of them. Pure, stateless between
// bars and ticker-agnostic. Run with `-strategy pdl_reclaim -interval Minutes15`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// dayBars is the number of 15-minute bars from 07:00 to 23:45 (the last bar of the evening session)
// inclusive: a setup is never older than that.
const dayBars = 68

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	EntryPDL int // 1 — the reclaim of PDL (green close above it after a red close below it) arms an entry
	EntryPDH int // 1 — the breakout of PDH (first green close above it on raised volume) arms an entry
	// HoldBars: the PDH breakout buys once the HoldBars bars after it all closed at or above PDH; a
	// close below PDH among them cancels it; 0 — buy right after the breakout bar; < 0 — no PDH entries
	HoldBars        int
	StopDailyATR    float64 // stop below the fill in daily ATR; 0 — no entries (no risk unit)
	DailyATRPeriod  int     // Wilder period over completed weekday dailies
	EntryFromHour   int     // the buy-trigger bar starts at/after EntryFrom MSK
	EntryFromMinute int
	EntryToHour     int // the buy bar starts before EntryTo MSK, on the trigger's date
	EntryToMinute   int
	// VolumeMult > 0: the PDH breakout bar's volume must be above VolumeMult × the average volume of
	// the VolBars main-session bars (10:00–18:30 MSK, weekdays) before it; 0 — off
	VolumeMult float64
	VolBars    int
	// MaxDayRangeATR > 0: no entry when today's range (high − low of the trigger's date up to and
	// including the trigger bar) is at or above MaxDayRangeATR daily ATRs — the day has already
	// spent its move; 0 — off
	MaxDayRangeATR float64
	// IndexTrendEMA > 0: enter only while the market index (IMOEX, set with WithIndex) is falling —
	// its EMA(IndexTrendEMA) over completed dailies below the value indexSlopeDays dailies earlier;
	// 0 — off
	IndexTrendEMA int
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		EntryPDL:        1,
		EntryPDH:        1,
		HoldBars:        1,
		StopDailyATR:    0.5,
		DailyATRPeriod:  14,
		EntryFromHour:   7,
		EntryFromMinute: 0,
		EntryToHour:     18,
		EntryToMinute:   0,
		VolumeMult:      1.5,
		VolBars:         20,
	}
}

// Strategy trades a single instrument with the pdl_reclaim rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
	idx    index
}

// NewWithParams returns the strategy for a ticker; step is the bar length (15m).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// tf labels the bars in reasons ("15m").
func (s *Strategy) tf() string { return fmt.Sprintf("%dm", int(s.step.Minutes())) }

// Lookback covers the bars of the day the setups look back over and the VolBars main-session bars
// the volume filter averages before a breakout early in the day: whole days of bars, plus a
// weekend in between.
func (s *Strategy) Lookback() int {
	warm := 0
	if s.volumeOn() {
		days := (s.p.VolBars + sessionBars - 1) / sessionBars
		warm = max(warm, (days+2)*dayBars)
	}
	return warm + dayBars
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

// series is the bars, oldest-first: opens, highs, lows, closes, volumes and start times. v is nil
// when the bars carry no volumes.
type series struct {
	o, h, l, c []float64
	v          []int64
	t          []time.Time
}

// barsOf is the engine's bars as a series (no copy).
func barsOf(md strategy.MarketData) series {
	return series{o: md.Opens, h: md.Highs, l: md.Lows, c: md.Closes, v: md.Volumes, t: md.Times}
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

// entryAllowed reports whether the current bar may buy at the next open: a weekday, the bar
// starts at/after EntryFrom, and the buy bar starts before EntryTo on the trigger's date.
func (s *Strategy) entryAllowed(md strategy.MarketData) bool {
	t := md.Times[len(md.Times)-1]
	if isWeekend(t.In(mskLoc)) || minuteOf(t) < s.entryFrom() {
		return false
	}
	fill := t.Add(s.step)
	return dayKey(fill) == dayKey(t) && minuteOf(fill) < s.p.EntryToHour*60+s.p.EntryToMinute
}

// entryFrom is the MSK minute of the day from which a bar may trigger a buy.
func (s *Strategy) entryFrom() int { return s.p.EntryFromHour*60 + s.p.EntryFromMinute }

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys the next open when the current bar completes the hold of a PDH breakout or is a PDL
// reclaim. The cheap bar gates run before the daily ATR.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if s.p.StopDailyATR <= 0 || !s.entryAllowed(md) {
		return sig
	}
	ser, k := barsOf(md), n-1
	pdl, pdh, ok := s.levels(md)
	if !ok {
		return sig
	}
	st, why, ok := s.trigger(ser, k, pdl, pdh)
	if !ok {
		return sig
	}
	closeP := ser.c[k]
	target := targetOf(st, pdl, pdh)
	if closeP >= target {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	spent := dayRange(ser, k) / atr
	if !s.roomOK(spent) {
		return sig
	}
	indexWhy, indexOK := s.indexOK(md.Times[k])
	if !indexOK {
		return sig
	}
	stop := closeP - s.p.StopDailyATR*atr
	if stop <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.ATR, sig.Level = stop, atr, st.level
	sig.TakeProfit = target
	sig.FillBy = md.Times[k].Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s %s: %s на свече %s, %s, PDL %.4f, PDH %.4f, сигнал по close %.4f, цель %.4f, дневной ATR %.4f, стоп %.4f",
		st.name, s.tf(), st.what(), ser.t[st.bar].In(mskLoc).Format("15:04"), why, pdl, pdh, closeP, target, atr, stop)
	if st.name == "PDH" && s.volumeOn() {
		vol, avg, _ := s.volumeOK(ser, st.bar)
		sig.EntryReason += fmt.Sprintf(", объём пробоя %.0f > %.2f × среднее %d дневных свечей %.0f", vol, s.p.VolumeMult, s.p.VolBars, avg)
	}
	if s.p.MaxDayRangeATR > 0 {
		sig.EntryReason += fmt.Sprintf(", диапазон дня %.2f ATR < %.2f", spent, s.p.MaxDayRangeATR)
	}
	if indexWhy != "" {
		sig.EntryReason += ", " + indexWhy
	}
	return sig
}

// roomOK reports whether today's range so far, spent daily ATRs, leaves the day room to move.
func (s *Strategy) roomOK(spent float64) bool {
	return s.p.MaxDayRangeATR <= 0 || spent < s.p.MaxDayRangeATR
}

// dayRange is high − low over the bars of bar k's date up to and including k.
func dayRange(ser series, k int) float64 {
	hi, lo := ser.h[k], ser.l[k]
	for j := k - 1; j >= 0 && dayKey(ser.t[j]) == dayKey(ser.t[k]); j-- {
		hi, lo = max(hi, ser.h[j]), min(lo, ser.l[j])
	}
	return hi - lo
}

// targetOf is the target of setup st: its level plus half of yesterday's range — the middle of
// the range for the PDL reclaim, PDH + half the range for the PDH breakout.
func targetOf(st setup, pdl, pdh float64) float64 { return st.level + (pdh-pdl)/2 }

// manage exits in priority order: stop, target. The position carries overnight.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low, high := md.Lows[i], md.Highs[i]
	if stop := s.stop(md, pos); stop > 0 && low <= stop {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
		return sig
	}
	if pos.TakeProfit > 0 && high >= pos.TakeProfit {
		sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цели %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
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
