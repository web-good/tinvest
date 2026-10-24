// Package core implements pdl_rsi: a long-only strategy on hourly bars
// (`-strategy pdl_rsi -interval Hour1`).
//
// The RSI signal is a closed hour on which the hourly RSI(HourRSIPeriod) crosses RSIArm downward:
// the previous hour of the same MSK date closed with the RSI at or above RSIArm, this one below it.
// The trigger is the first hour within StochBars hours after the RSI signal on which the slow
// stochastic %K (StochK raw, smoothed over StochSmooth) crosses its %D (StochD) upward with %K below
// StochCrossMax. It buys the next hour's open when the trigger closes strictly above the
// EMA(TrendEMA) of the hourly closes, the RSI stayed at or above RSIFloor from the signal through
// the trigger, the trigger hour's RSI is at or below RSITriggerMax, the
// optional filters pass (the volume on the RSI hour, the room on the trigger) and the buy bar falls
// on the same weekday exchange date.
//
// The stop sits StopATR daily ATRs (frozen at the signal) under the fill, the target TargetR stop
// distances above it; the signal carries both off the trigger close for sizing. The RSI exit fires
// on the close of an hour whose RSI crosses RSIExit upward, the optional stochastic exit on the close
// of an hour whose %K crosses %D down above StochExitAbove. The position carries overnight.
//
// A weekday exchange holiday shows up as dealer-only bars from 02:00 MSK, while an exchange
// weekday starts at 06:00: a weekday date whose first bar starts before 06:00 is a dealer date
// (dealerDate). It never buys, and its bars stay out of the volume baseline.
// Pure, stateless between bars and ticker-agnostic.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// dayBars is the number of hourly bars of an exchange weekday, from 06:00 to 23:00 inclusive.
const dayBars = 18

// longDayBars bounds the hourly bars of a weekend or dealer date: from 02:00 to 23:00 inclusive.
const longDayBars = 22

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	// HourRSIPeriod is the Wilder period of the hourly RSI; 0 — no entries
	HourRSIPeriod int
	// RSIArm: the RSI signal is the hourly RSI crossing down through this
	RSIArm float64
	// RSIFloor > 0: no entry when the hourly RSI dropped below this on any hour from the RSI signal
	// through the trigger; 0 — off
	RSIFloor float64
	// RSITriggerMax > 0: the hourly RSI of the trigger hour must be at or below this; 0 — off
	RSITriggerMax float64
	// RSIExit: the hourly RSI crossing up through this exits; 0 — no RSI exit
	RSIExit float64
	// StochExitAbove > 0: the stochastic %K crossing %D down with %K (and so %D) above this exits at
	// the hour's close; 0 — off
	StochExitAbove float64

	// The slow stochastic: %K is the raw StochK-bar %K smoothed over StochSmooth bars, %D its
	// StochD-bar average. The trigger is %K crossing %D up within StochBars hours after the RSI
	// signal (the signal hour itself excluded); any of them 0 — no entries
	StochK      int
	StochSmooth int
	StochD      int
	StochBars   int
	// StochCrossMax > 0: the cross counts only with %K (and so %D) below this on the trigger hour;
	// a cross at or above it is no trigger; 0 — any level
	StochCrossMax float64

	// VolumeMult > 0: the RSI signal hour (a main-session hour) must trade above VolumeMult × the
	// average of the VolBars main-session hours before it; 0 — volume off
	VolumeMult float64
	VolBars    int

	// MinRoomATR > 0: the day must have this many daily ATRs of upside left at the trigger; 0 — room off
	MinRoomATR float64

	// StopATR: the stop sits this many daily ATRs under the fill; 0 — no entries
	StopATR float64
	// MaxStopPct > 0 caps the stop distance at this percent of the price, whatever the ATR; 0 — off
	MaxStopPct float64
	// TargetR: the target sits this many stop distances above the fill; 0 — no target
	TargetR float64

	// TrendEMA > 0: the trigger hour must close strictly above the warmed EMA of this period over
	// the hourly closes; 0 — trend off
	TrendEMA int

	DailyATRPeriod int // Wilder period over completed weekday dailies
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       22,
		RSITriggerMax:  50,
		RSIExit:        70,
		StochK:         5,
		StochSmooth:    3,
		StochD:         3,
		StochBars:      4,
		StochCrossMax:  40,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.5,
		TargetR:        2,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
}

// Strategy trades a single instrument with the pdl_rsi rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams returns the strategy for a ticker; step is the bar length (1h).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback covers the VolBars main-session hours before an RSI hour up to StochBars hours back
// (whole days plus a weekend of up to longDayBars bars a day), the current day, the RSI and
// stochastic warm-up and twice the trend EMA period.
func (s *Strategy) Lookback() int {
	volDays := (max(s.p.VolBars, 1) + sessionHours - 1) / sessionHours
	return max(4*s.p.HourRSIPeriod+66+s.p.StochBars, (volDays+2)*dayBars+2*longDayBars,
		2*s.p.TrendEMA+20)
}

// trendAt is the EMA(TrendEMA) of the closes at bar k; false while the EMA is not warm
// (ema.Compute zero-fills the first TrendEMA-1 values).
func (s *Strategy) trendAt(closes []float64, k int) (float64, bool) {
	if k+1 < s.p.TrendEMA {
		return 0, false
	}
	e := ema.Compute(closes[:k+1], s.p.TrendEMA)[k]
	return e, e > 0
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

// exchangeOpen is the MSK minute the first hourly bar of an exchange weekday starts at (06:00).
const exchangeOpen = 6 * 60

// dealerDate reports whether bar k falls on a weekday exchange holiday: a weekday date whose first
// bar in times starts before exchangeOpen (dealer-only prints from 02:00, no daily candle).
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

// enter buys the next open when the current hour is a trigger (the first stochastic cross up within
// StochBars hours after the RSI crossing RSIArm down) and the trend, RSI cap, room, volume and
// stop checks pass.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if s.p.HourRSIPeriod <= 0 || s.p.StochBars <= 0 || !s.entryAllowed(md) {
		return sig
	}
	ser, k := barsOf(md), n-1
	sk, sd, first := s.stochKD(ser)
	if sk == nil || k-1 < first || !s.stochTrigger(sk, sd, k) {
		return sig
	}
	rsi := indicators.RSISeries(ser.c, s.p.HourRSIPeriod)
	j := s.rsiSignal(ser, rsi, k)
	if j < 0 {
		return sig
	}
	for m := max(j+1, first+1); m < k; m++ {
		if s.stochTrigger(sk, sd, m) {
			return sig // not the first trigger cross after the RSI signal
		}
	}
	if s.p.RSITriggerMax > 0 && rsi[k] > s.p.RSITriggerMax {
		return sig
	}
	low := rsi[j]
	for m := j + 1; m <= k; m++ {
		low = min(low, rsi[m])
	}
	if s.p.RSIFloor > 0 && low < s.p.RSIFloor {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	closeP := ser.c[k]
	var trend float64
	if s.p.TrendEMA > 0 {
		var ok bool
		if trend, ok = s.trendAt(ser.c, k); !ok || closeP <= trend {
			return sig
		}
	}
	room := roomLeft(ser, k, atr)
	if s.p.MinRoomATR > 0 && room < s.p.MinRoomATR {
		return sig
	}
	win, base, ok := s.volumeOK(ser, j)
	if !ok {
		return sig
	}
	stop := closeP - s.stopDistance(closeP, atr)
	if stop <= 0 || stop >= closeP {
		return sig
	}
	target := 0.0
	if s.p.TargetR > 0 {
		target = closeP + s.p.TargetR*(closeP-stop)
	}
	hm := func(i int) string { return ser.t[i].In(mskLoc).Format("15:04") }
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.TakeProfit, sig.ATR = stop, target, atr
	sig.FillBy = md.Times[k].Add(s.step)
	sig.EntryReason = fmt.Sprintf("крест часового RSI(%d) %.2f → %.2f через %.0f вниз на часе %s; стохастик(%d,%d,%d) %%K %.2f → %.2f пересёк %%D %.2f → %.2f вверх на часе %s (через %d ч)%s, RSI %.2f; close %.4f, дневной ATR %.4f; стоп %.4f = close − %.2f ATR",
		s.p.HourRSIPeriod, rsi[j-1], rsi[j], s.p.RSIArm, hm(j),
		s.p.StochK, s.p.StochSmooth, s.p.StochD, sk[k-1], sk[k], sd[k-1], sd[k], hm(k), k-j, s.stochWhy(), rsi[k],
		closeP, atr, stop, s.p.StopATR)
	if s.stopCapped(closeP, atr) {
		sig.EntryReason += fmt.Sprintf(" (урезан до %.2f%% цены)", s.p.MaxStopPct)
	}
	if target > 0 {
		sig.EntryReason += fmt.Sprintf(", цель %.4f = close + %.2f R", target, s.p.TargetR)
	}
	if s.p.RSITriggerMax > 0 {
		sig.EntryReason += fmt.Sprintf(" (RSI ≤ %.0f)", s.p.RSITriggerMax)
	}
	if s.p.RSIFloor > 0 {
		sig.EntryReason += fmt.Sprintf(", мин. RSI от сигнала %.2f ≥ %.0f", low, s.p.RSIFloor)
	}
	if s.p.TrendEMA > 0 {
		sig.EntryReason += fmt.Sprintf(", над EMA(%d) %.4f", s.p.TrendEMA, trend)
	}
	if s.p.MinRoomATR > 0 {
		sig.EntryReason += fmt.Sprintf(", запас хода %.2f ATR ≥ %.2f", room, s.p.MinRoomATR)
	}
	if s.volumeOn() {
		sig.EntryReason += fmt.Sprintf(", объём часа RSI %.0f > %.2f × среднее %d часов %.0f",
			win, s.p.VolumeMult, s.p.VolBars, base)
	}
	return sig
}

// stopDistance is the stop distance under price: StopATR daily ATRs, capped at MaxStopPct percent
// of price when MaxStopPct > 0.
func (s *Strategy) stopDistance(price, atr float64) float64 {
	d := s.p.StopATR * atr
	if s.stopCapped(price, atr) {
		d = s.p.MaxStopPct / 100 * price
	}
	return d
}

// stopCapped reports whether MaxStopPct cuts the StopATR distance at price.
func (s *Strategy) stopCapped(price, atr float64) bool {
	return s.p.MaxStopPct > 0 && s.p.MaxStopPct/100*price < s.p.StopATR*atr
}

// rsiSignal is the latest hour j within StochBars hours before the trigger k (k−StochBars ≤ j < k)
// on which the RSI crosses RSIArm down from an hour of the same MSK date; -1 when none. Hours start
// at HourRSIPeriod + 1, so both RSIs of the cross are real values.
func (s *Strategy) rsiSignal(ser series, rsi []float64, k int) int {
	for j := k - 1; j >= k-s.p.StochBars && j > s.p.HourRSIPeriod; j-- {
		if dayKey(ser.t[j-1]) == dayKey(ser.t[j]) && crossDown(rsi[j-1], rsi[j], s.p.RSIArm) {
			return j
		}
	}
	return -1
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }

// crossDown reports whether a series moved from at or above level to below it.
func crossDown(prev, cur, level float64) bool { return prev >= level && cur < level }
