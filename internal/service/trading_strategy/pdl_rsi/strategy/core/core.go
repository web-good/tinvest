// Package core implements pdl_rsi: a long-only strategy at yesterday's low (PDL) on 15-minute
// bars with an hourly RSI. Hours are built from the 15m bars; one RSI period (RSIPeriod) drives
// both the hourly and the 15m RSI.
//
// A completed hour with the hourly RSI below RSIArm arms a setup. The signal is a closed 15m bar
// on which the 15m RSI crosses RSIEntry upward, provided its low lies within PDL ± ZoneATR daily
// ATRs, the hourly RSI has not crossed RSIGate upward since arming, the volume and room filters
// pass and the buy bar (the next 15m open) falls on the same weekday date. The stop is the lowest
// low of the search window's 15m bars with RSI below RSIEntry and the signal bar.
//
// The search window starts at the first 15m bar of the arming hour and ends at the signal; the
// signal comes only after the arming hour completed. While a setup is alive further hours below
// RSIArm do not re-arm it, so the window stays put. A setup burns on the hourly RSI crossing
// RSIGate up, or on its signal cross (15m RSI up through RSIEntry with the low in the zone)
// whether it buys or fails a filter; a cross with the low outside the zone keeps it. Arming and
// the signal share one MSK date: a setup never outlives its date. The core is stateless: every
// bar replays the automaton over the bars of its date.
//
// Once a completed hour after the fill crosses RSIBreakeven upward the stop moves to the fill
// (exit reason TRAIL). The RSI exit fires on the bar completing an hour whose RSI crosses RSIExit
// upward. The position carries overnight.
//
// A weekday exchange holiday shows up as dealer-only bars from about 02:00 MSK, while an exchange
// weekday starts at 06:45: a weekday date whose first bar starts before 06:45 is a dealer date
// (dealerDate). It never buys, and its bars stay out of the volume window and baseline.
// Pure, stateless between bars and ticker-agnostic.
// Run with `-strategy pdl_rsi -interval Minutes15`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// dayBars is the number of 15-minute bars of an exchange weekday, from 06:45 to 23:45 (the last bar
// of the evening session) inclusive: a setup is never older than that.
const dayBars = 69

// longDayBars bounds the 15-minute bars of a weekend or dealer date: from 02:00 to 23:45 inclusive.
const longDayBars = 88

// holdDays is the weekdays of history the breakeven scan keeps for an open position; they span
// holdWeekendDays weekend days.
const (
	holdDays        = 10
	holdWeekendDays = 4
)

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	// RSIPeriod is the Wilder period of both the hourly and the 15m RSI; 0 — no entries
	RSIPeriod int
	// RSIArm: a completed hour with the hourly RSI below this arms a setup
	RSIArm float64
	// RSIEntry: the signal is the 15m RSI crossing up through this
	RSIEntry float64
	// RSIGate: the hourly RSI must not have crossed this upward since arming; 0 — no gate
	RSIGate float64
	// RSIBreakeven: once the hourly RSI passes this the stop moves to the fill; 0 — no breakeven
	RSIBreakeven float64
	// RSIExit: the hourly RSI crossing up through this exits; 0 — no RSI exit
	RSIExit float64
	// ZoneATR is the half-width of the zone in daily ATR: the signal bar's low must lie in
	// PDL ± ZoneATR × daily ATR
	ZoneATR float64

	// VolumeMult > 0: the entry window's average main-session volume must be above VolumeMult ×
	// the average of the VolBars main-session bars before it; 0 — volume off
	VolumeMult float64
	VolBars    int

	// MinRoomATR > 0: the day must have this many daily ATRs of upside left at the signal; 0 — room off
	MinRoomATR float64

	DailyATRPeriod int // Wilder period over completed weekday dailies
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      6,
		RSIArm:         30,
		RSIEntry:       30,
		RSIGate:        50,
		RSIBreakeven:   50,
		RSIExit:        70,
		ZoneATR:        0.4,
		VolumeMult:     1.5,
		VolBars:        20,
		MinRoomATR:     0.6,
		DailyATRPeriod: 14,
	}
}

// Strategy trades a single instrument with the pdl_rsi rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams returns the strategy for a ticker; step is the bar length (15m).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback covers holdDays weekdays plus the two weekends they span (holdWeekendDays days of up to
// longDayBars bars) for the breakeven scan, whole days for the VolBars main-session bars before a
// window that starts early in the day, the current day and the hourly RSI warm-up (four 15m bars
// per hour).
func (s *Strategy) Lookback() int {
	volDays := (max(s.p.VolBars, 1) + sessionBars - 1) / sessionBars
	return max(4*(4*s.p.RSIPeriod+2+64), (holdDays+volDays+1)*dayBars+holdWeekendDays*longDayBars)
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

// exchangeOpen is the MSK minute the first bar of an exchange weekday starts at (06:45).
const exchangeOpen = 6*60 + 45

// dealerDate reports whether bar k falls on a weekday exchange holiday: a weekday date whose first
// bar in times starts before exchangeOpen (dealer-only prints from about 02:00, no daily candle).
// Only bars at or before k are read. The oldest date may start mid-day in the window: it counts as
// an exchange date unless its first visible bar is before exchangeOpen. Weekends are never dealer
// dates: they are handled as weekends.
func dealerDate(times []time.Time, k int) bool {
	if isWeekend(times[k].In(mskLoc)) {
		return false
	}
	date, first := dayKey(times[k]), k
	for first > 0 && dayKey(times[first-1]) == date {
		first--
	}
	return minuteOf(times[first]) < exchangeOpen
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

// pdl returns yesterday's low: the low of the last completed weekday daily.
func (s *Strategy) pdl(md strategy.MarketData) (float64, bool) {
	_, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if len(c) == 0 || len(l) != len(c) {
		return 0, false
	}
	return l[len(l)-1], l[len(l)-1] > 0
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

// entryAllowed reports whether the current bar may buy at the next open: a weekday exchange date
// (not a dealer date), and the buy bar starts on the signal's date.
func (s *Strategy) entryAllowed(md strategy.MarketData) bool {
	k := len(md.Times) - 1
	t := md.Times[k]
	return !isWeekend(t.In(mskLoc)) && dayKey(t.Add(s.step)) == dayKey(t) && !dealerDate(md.Times, k)
}

// roomLeft is the upside left in the day at bar k, in daily ATRs: the low of bar k's date up to
// and including k plus one ATR, minus the close of k.
func roomLeft(ser series, k int, atr float64) float64 {
	lo := ser.l[k]
	for j := k - 1; j >= 0 && dayKey(ser.t[j]) == dayKey(ser.t[k]); j-- {
		lo = min(lo, ser.l[j])
	}
	return (lo + atr - ser.c[k]) / atr
}

// Decide buys the next open on a signal, or manages the open position.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys the next open when the current bar is the signal of a setup (scan) and the buy bar,
// room, volume and stop checks pass. A failed check leaves the setup burnt: the replay already
// spent it on this bar.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if s.p.RSIPeriod <= 0 || !s.entryAllowed(md) {
		return sig
	}
	pdl, ok := s.pdl(md)
	if !ok {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	ser, k := barsOf(md), n-1
	hs := s.hourBars(md)
	rh := indicators.RSISeries(hourCloses(hs), s.p.RSIPeriod)
	r15 := indicators.RSISeries(ser.c, s.p.RSIPeriod)
	zoneLo, zoneHi := pdl-s.p.ZoneATR*atr, pdl+s.p.ZoneATR*atr
	st, ok := s.scan(ser, hs, rh, r15, k, zoneLo, zoneHi)
	if !ok {
		return sig
	}
	room := roomLeft(ser, k, atr)
	if s.p.MinRoomATR > 0 && room < s.p.MinRoomATR {
		return sig
	}
	win, base, ok := s.volumeOK(ser, st.from, k)
	if !ok {
		return sig
	}
	closeP := ser.c[k]
	sb := stopBar(ser, r15, st, s.p.RSIEntry)
	stop := ser.l[sb]
	if stop <= 0 || stop >= closeP {
		return sig
	}
	armHour := s.armHour(hs, st)
	hm := func(j int) string { return ser.t[j].In(mskLoc).Format("15:04") }
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.TakeProfit, sig.ATR, sig.Level = stop, 0, atr, pdl
	sig.FillBy = md.Times[k].Add(s.step)
	sig.EntryReason = fmt.Sprintf("взвод: час %s RSI(%d) %.2f < %.0f (закрыт на свече %s); крест RSI(%d) 15m %.2f → %.2f через %.0f на свече %s, лой %.4f в зоне PDL %.4f ± %.2f ATR [%.4f; %.4f]; close %.4f, дневной ATR %.4f; стоп %.4f по лою свечи %s",
		hs[armHour].start.In(mskLoc).Format("15:04"), s.p.RSIPeriod, rh[armHour], s.p.RSIArm, hm(st.arm),
		s.p.RSIPeriod, r15[k-1], r15[k], s.p.RSIEntry, hm(k), ser.l[k], pdl, s.p.ZoneATR, zoneLo, zoneHi,
		closeP, atr, stop, hm(sb))
	if s.p.MinRoomATR > 0 {
		sig.EntryReason += fmt.Sprintf(", запас хода %.2f ATR ≥ %.2f", room, s.p.MinRoomATR)
	}
	if s.volumeOn() {
		sig.EntryReason += fmt.Sprintf(", средний объём от %s %.0f > %.2f × среднее %d свечей %.0f",
			hm(st.from), win, s.p.VolumeMult, s.p.VolBars, base)
	}
	return sig
}

// armHour is the index in hs of the hour that armed st: the one starting at the window's first
// bar and completed at the arming bar.
func (s *Strategy) armHour(hs []hourBar, st setup) int {
	for h := len(hs) - 1; h >= 0; h-- {
		if hs[h].first == st.from && hs[h].done == st.arm {
			return h
		}
	}
	return 0
}
