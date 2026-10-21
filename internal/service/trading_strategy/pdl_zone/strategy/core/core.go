// Package core implements pdl_zone: a long-only strategy at yesterday's low (PDL) on 5-minute
// bars. A setup arms on a bar that dips into the zone of interest above PDL (low ≤ PDL + ZoneATR
// daily ATRs, close ≥ PDL) or on a bar that closes below PDL (the break). The signal is the first
// closed bar with a green MACD histogram from the arming bar on, wherever the price is; the buy
// comes at the next bar's open on the same date, and only when the average volume from the arming
// bar to the signal is above VolumeMult × the average of the VolBars main-session bars before it.
//
// The stop is the lowest low of the StopBars bars before the signal, at least MinStopATR daily
// ATRs under the entry; the target is TargetR stop distances above the entry. The entry is the
// fill price (the next bar's open): the buy signal carries both levels off the signal close for
// sizing, and the position rebuilds them off the fill (exitLevels). Exits are the
// stop and the target only; the position carries overnight. Pure, stateless between bars and
// ticker-agnostic. Yesterday's high is not traded. Run with `-strategy pdl_zone -interval Minutes5`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// dayBars is the number of 5-minute bars from 07:00 to 23:50 (the last bar of the evening session)
// inclusive: a setup is never older than that.
const dayBars = 204

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	EntryZone  int     // 1 — a dip into the zone above PDL arms a setup
	EntryBreak int     // 1 — a close below PDL arms a setup (and turns an armed zone setup into a break)
	ZoneATR    float64 // the zone's top above PDL in daily ATR
	// BreakReclaim 1: a break setup signals only after a bar closes back above PDL (the first green
	// histogram bar from that reclaim bar on); 0 — the first green bar after the break
	BreakReclaim int

	MACDFast   int
	MACDSlow   int
	MACDSignal int

	// VolumeMult > 0: the average main-session volume from the arming bar to the signal must be
	// above VolumeMult × the average of the VolBars main-session bars before the arming bar; a
	// window without main-session bars fails closed; 0 — off (diagnostics only)
	VolumeMult float64
	VolBars    int

	StopBars   int     // the stop is the lowest low of this many bars before the signal bar
	MinStopATR float64 // the stop is at least this many daily ATRs under the entry
	TargetR    float64 // the target in stop distances above the entry
	// MinRoomATR > 0: the day must have this many daily ATRs of upside left at the signal — the
	// low of the signal's date so far plus one daily ATR, minus the signal close; 0 — off
	MinRoomATR float64
	// H1TrendEMA > 0: enter only while the last completed hourly close is above the EMA(N) of the
	// hourly closes (cmd/backtest loads Hour1 as the HTF series); 0 — off
	H1TrendEMA int
	// D1TrendEMA > 0: enter only while the last completed weekday daily close is above the EMA(N)
	// of the weekday daily closes; 0 — off
	D1TrendEMA int

	DailyATRPeriod int // Wilder period over completed weekday dailies
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		EntryZone:      1,
		EntryBreak:     1,
		ZoneATR:        0.2,
		MACDFast:       8,
		MACDSlow:       21,
		MACDSignal:     9,
		VolumeMult:     1.5,
		VolBars:        15,
		StopBars:       5,
		MinStopATR:     0.2,
		TargetR:        2,
		MinRoomATR:     0.7,
		DailyATRPeriod: 14,
	}
}

// Strategy trades a single instrument with the pdl_zone rules. Ticker-agnostic and pure.
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

// tf labels the bars in reasons ("5m").
func (s *Strategy) tf() string { return fmt.Sprintf("%dm", int(s.step.Minutes())) }

// macdWarm is the history the MACD histogram needs before it is trusted.
func (s *Strategy) macdWarm() int { return 3 * (s.p.MACDSlow + s.p.MACDSignal) }

// Lookback covers the MACD warm-up, the day of bars the setup replays and the VolBars
// main-session bars the volume baseline averages before an arming bar early in the day: whole
// days of bars, plus a weekend in between.
func (s *Strategy) Lookback() int {
	days := (max(s.p.VolBars, 1) + sessionBars - 1) / sessionBars
	return max(s.macdWarm(), (days+2)*dayBars) + dayBars + s.p.StopBars
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

// entryAllowed reports whether the current bar may buy at the next open: a weekday, and the buy
// bar starts on the signal's date.
func (s *Strategy) entryAllowed(md strategy.MarketData) bool {
	t := md.Times[len(md.Times)-1]
	return !isWeekend(t.In(mskLoc)) && dayKey(t.Add(s.step)) == dayKey(t)
}

// histogram is MACD − signal over the closes; nil when the history is too short to trust it.
func (s *Strategy) histogram(closes []float64) []float64 {
	if len(closes) < s.macdWarm() {
		return nil
	}
	m, sig := indicators.MACD(closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal)
	if m == nil {
		return nil
	}
	h := make([]float64, len(m))
	for i := range m {
		h[i] = m[i] - sig[i]
	}
	return h
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys the next open when the current bar is the signal of a setup whose kind is enabled,
// the volume filter passes and the stop sits under the close.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if s.p.EntryZone <= 0 && s.p.EntryBreak <= 0 || !s.entryAllowed(md) {
		return sig
	}
	ser, k := barsOf(md), n-1
	if k < s.p.StopBars || s.p.StopBars <= 0 {
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
	hist := s.histogram(ser.c)
	if hist == nil {
		return sig
	}
	st, ok := s.scan(ser, hist, k, pdl, pdl+s.p.ZoneATR*atr)
	if !ok {
		return sig
	}
	trendWhy, trendOK := s.trendOK(md)
	if !trendOK {
		return sig
	}
	room := roomLeft(ser, k, atr)
	if s.p.MinRoomATR > 0 && room < s.p.MinRoomATR {
		return sig
	}
	win, base, ok := s.volumeOK(ser, st.arm, k)
	if !ok {
		return sig
	}
	closeP := ser.c[k]
	low := ser.l[k-s.p.StopBars]
	for j := k - s.p.StopBars + 1; j < k; j++ {
		low = min(low, ser.l[j])
	}
	stop, stopWhy := low, fmt.Sprintf("стоп по лою %d свечей %.4f", s.p.StopBars, low)
	if floor := closeP - s.p.MinStopATR*atr; stop > floor {
		stop = floor
		stopWhy += fmt.Sprintf(", опущен до %.2f ATR: %.4f", s.p.MinStopATR, stop)
	}
	if stop <= 0 || stop >= closeP {
		return sig
	}
	target := closeP + s.p.TargetR*(closeP-stop)
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.TakeProfit, sig.ATR, sig.Level = stop, target, atr, pdl
	sig.FillBy = md.Times[k].Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s %s: взвод на свече %s, первая зелёная гистограмма MACD(%d,%d,%d) на свече %s, PDL %.4f, верх зоны %.4f, сигнал по close %.4f, дневной ATR %.4f, %s, цель %.2fR %.4f (стоп и цель пересчитываются от цены покупки)",
		st.name(), s.tf(), ser.t[st.arm].In(mskLoc).Format("15:04"), s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal,
		ser.t[k].In(mskLoc).Format("15:04"), pdl, pdl+s.p.ZoneATR*atr, closeP, atr, stopWhy, s.p.TargetR, target)
	if s.p.MinRoomATR > 0 {
		sig.EntryReason += fmt.Sprintf(", запас хода %.2f ATR ≥ %.2f", room, s.p.MinRoomATR)
	}
	if trendWhy != "" {
		sig.EntryReason += ", " + trendWhy
	}
	if st.kind == kindBreak && s.p.BreakReclaim > 0 {
		sig.EntryReason += ", после возврата закрытием над PDL"
	}
	if s.volumeOn() {
		sig.EntryReason += fmt.Sprintf(", средний объём от взвода %.0f > %.2f × среднее %d свечей %.0f", win, s.p.VolumeMult, s.p.VolBars, base)
	}
	return sig
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

// manage exits in priority order: the stop frozen at the signal, the target. The position
// carries overnight.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	stop, target, ok := s.exitLevels(pos)
	if !ok {
		return sig
	}
	low, high := md.Lows[n-1], md.Highs[n-1]
	if low <= stop {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
		return sig
	}
	if high >= target {
		sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", target
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цели %.4f (вход %.4f)", high, target, pos.PurchasePrice)
	}
	return sig
}

// exitLevels rebuilds the stop and the target off the fill price: the stop is the signal's stop
// (the StopBars low, or MinStopATR daily ATRs under the signal close when that low was nearer),
// lowered to MinStopATR daily ATRs (frozen at the signal) under the fill when the fill came lower;
// the target is TargetR stop distances above the fill. The stop matches the rule exactly except
// after a gap up over a signal whose low was nearer than MinStopATR: then it stays MinStopATR under
// the signal close, a little deeper than MinStopATR under the fill.
func (s *Strategy) exitLevels(pos *strategy.Position) (stop, target float64, ok bool) {
	f := pos.PurchasePrice
	if pos.StopLoss <= 0 || f <= 0 {
		return 0, 0, false
	}
	stop = pos.StopLoss
	if pos.EntryATR > 0 {
		stop = min(stop, f-s.p.MinStopATR*pos.EntryATR)
	}
	if stop <= 0 || stop >= f {
		return 0, 0, false
	}
	return stop, f + s.p.TargetR*(f-stop), true
}
