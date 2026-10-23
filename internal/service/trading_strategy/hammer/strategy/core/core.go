// Package core implements hammer: a long-only strategy on hourly or 30-minute bars. The signal is a
// hammer candle (isHammer: a non-zero body in the upper half, a lower tail of at least two bodies,
// an upper tail of at most one body)
// whose own close has RSI(RSIPeriod) below RSIZone and, with TrendEMA on, sits strictly above the
// EMA(TrendEMA) of the bar closes, on a volume above VolumeMult × the average of the VolBars
// main-session bars before it; the buy comes at the next bar's open on the same MSK weekday date.
// With ConfirmBars on, the buy waits for a green bar closing above the hammer's high within that
// many bars (confirmedHammer) and comes at the open after it.
//
// The stop is a fixed level StopBufferATR daily ATRs (Wilder DailyATRPeriod over completed weekday
// dailies) under the hammer's low; the target is TargetR risks above the fill (exitLevels; the buy
// signal carries it off the signal close for sizing). Exits in priority order: the stop, the target,
// the close of a red MACD(MACDFast, MACDSlow, MACDSignal) histogram bar after a green one since the
// hammer (macdTurnedRed). The first close since the entry with RSI above RSIHold exits at once when
// its histogram bar is not green (RSI); on a green one the target is off and the first red
// histogram bar from that close on exits. The position carries overnight and over
// weekends. Pure, stateless
// between bars and ticker- and timeframe-agnostic. Run with `-strategy hammer -interval Hour1` (or
// Minutes30).
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	RSIPeriod int     // RSI length over the bar closes
	RSIZone   float64 // the hammer bar's own RSI must be below this
	// MACD(MACDFast, MACDSlow, MACDSignal) of the bar closes: sell at the close of a red (< 0)
	// histogram bar once a green (> 0) one has printed since the hammer; MACDFast 0 — off
	MACDFast, MACDSlow, MACDSignal int
	// RSIHold > 0 (with the MACD exit on): the first close from the entry bar on with RSI(RSIPeriod)
	// above this sells at once when its histogram bar is not green (≤ 0, exit RSI); otherwise the
	// target no longer exits from the next bar on and the first red histogram bar from that close
	// on sells; 0 — off
	RSIHold float64
	// TrendEMA > 0: the hammer's close must be strictly above a warmed EMA of this period over the
	// bar closes; 0 — off
	TrendEMA int
	// VolumeMult > 0: the hammer bar's volume must be above this × the average of the VolBars
	// main-session bars before it; 0 — off
	VolumeMult float64
	VolBars    int

	// ConfirmBars > 0: no buy right after the hammer; the first of the next ConfirmBars bars that
	// closes green above the hammer's high is the signal (the buy at the open after it); a bar
	// under the hammer's low before that burns the hammer; 0 — buy right after the hammer
	ConfirmBars int

	// PDLZoneATR > 0: the hammer is looked for only at yesterday's low — its low lies within this
	// many daily ATRs of PDL, either side (atPDL); 0 — off
	PDLZoneATR float64

	StopBufferATR float64 // the stop is the hammer's low less this many daily ATRs (a fixed level)
	TargetR       float64 // the target is this × (fill − stop) above the fill; 0 — no target

	DailyATRPeriod int // Wilder period over completed weekday dailies
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      5,
		RSIZone:        30,
		MACDFast:       11,
		MACDSlow:       15,
		MACDSignal:     9,
		RSIHold:        60,
		TrendEMA:       200,
		VolumeMult:     1.5,
		VolBars:        20,
		StopBufferATR:  0.05,
		TargetR:        2,
		DailyATRPeriod: 14,
	}
}

// Strategy trades a single instrument with the hammer rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams returns the strategy for a ticker; step is the bar length (1h or 30m).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// tf labels the bars in reasons ("1h", "30m").
func (s *Strategy) tf() string {
	if s.step%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(s.step.Hours()))
	}
	return fmt.Sprintf("%dm", int(s.step.Minutes()))
}

// Lookback is enough bars for Wilder's RSI and the MACD to settle, for the trend EMA to recurse
// past its SMA seed (twice its period) and for VolBars main-session bars to fit past nights and
// weekends.
func (s *Strategy) Lookback() int {
	return max(20*s.p.RSIPeriod, 100, 2*s.p.TrendEMA+20, 5*s.p.VolBars+s.p.ConfirmBars, 10*(s.p.MACDSlow+s.p.MACDSignal))
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

// isHammer: a bar of either colour has a non-zero body (a doji with open = close is not a hammer)
// lying in the upper half of its range (the body's bottom at or above the middle) and a lower tail
// of at least two bodies; the upper tail (from the body's top to the high) may exist but is at most
// one body. The owner's definition since 2026-10-22; until then the body had to sit in the upper
// third, as in pdl_reclaim's isHammer. hammerEps absorbs float error on a tail of exactly two
// bodies or an upper tail of exactly one.
func isHammer(o, h, l, c float64) bool {
	rng := h - l
	if rng <= 0 {
		return false
	}
	bottom, top := min(o, c), max(o, c)
	body := top - bottom
	return body > 0 && bottom >= l+rng/2 && bottom-l+hammerEps >= 2*body && h-top <= body+hammerEps
}

// hammerEps is the float slack of isHammer's tail test, far below any price step.
const hammerEps = 1e-9

// The main session, MSK: the bars opening from 10:00 to before 19:00 (on 1h the 18:00 bar, on 30m
// the 18:30 one, holds the closing auction). The volume average skips the morning and evening
// sessions and weekends.
const (
	sessionFrom = 10 * 60
	sessionTo   = 19 * 60
)

// inSession reports whether t opens a weekday main-session bar.
func inSession(t time.Time) bool {
	tl := t.In(mskLoc)
	m := tl.Hour()*60 + tl.Minute()
	return !isWeekend(tl) && m >= sessionFrom && m < sessionTo
}

// volumeOK reports the volume of bar k, the average of the VolBars main-session bars before it
// and whether the filter passes: the volume is above VolumeMult × that average. Off (VolumeMult
// or VolBars ≤ 0) — always passes; without volumes or that many main-session bars it fails closed.
func (s *Strategy) volumeOK(md strategy.MarketData, k int) (vol, avg float64, ok bool) {
	if s.p.VolumeMult <= 0 || s.p.VolBars <= 0 {
		return 0, 0, true
	}
	if len(md.Volumes) != len(md.Closes) {
		return 0, 0, false
	}
	var sum int64
	got := 0
	for j := k - 1; j >= 0 && got < s.p.VolBars; j-- {
		if inSession(md.Times[j]) {
			sum += md.Volumes[j]
			got++
		}
	}
	if got < s.p.VolBars {
		return 0, 0, false
	}
	avg = float64(sum) / float64(got)
	vol = float64(md.Volumes[k])
	return vol, avg, avg > 0 && vol > s.p.VolumeMult*avg
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

// dailyATR returns Wilder's daily ATR over completed weekday dailies; ok is false when the dailies
// cannot support it.
func (s *Strategy) dailyATR(md strategy.MarketData) (atr float64, ok bool) {
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if s.p.DailyATRPeriod <= 0 || len(c) < s.p.DailyATRPeriod+1 || len(h) != len(c) || len(l) != len(c) {
		return 0, false
	}
	atr = indicators.ATR(h, l, c, s.p.DailyATRPeriod)
	return atr, atr > 0
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys the next open on a weekday whose next bar starts on the same date when the current
// bar is a hammer passing hammerGates or, with ConfirmBars on, confirms such a hammer
// (confirmedHammer). The stop sits under the hammer's low, the target is off the current close.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	k := n - 1
	t := md.Times[k]
	if isWeekend(t.In(mskLoc)) || dayKey(t.Add(s.step)) != dayKey(t) {
		return sig
	}
	ind := s.series(md)
	if !ind.atrOK {
		return sig
	}
	hb, why, ok := k, "", false
	if s.p.ConfirmBars > 0 {
		hb, why, ok = s.confirmedHammer(md, k, ind)
	} else {
		why, ok = s.hammerGates(md, k, ind)
	}
	if !ok {
		return sig
	}
	atr := ind.atr
	c, low := md.Closes[k], md.Lows[hb]
	stop := low - s.p.StopBufferATR*atr
	if stop <= 0 || stop >= c {
		return sig
	}
	target, targetWhy := 0.0, "цели нет"
	if s.p.TargetR > 0 {
		target = c + s.p.TargetR*(c-stop)
		targetWhy = fmt.Sprintf("цель %.2f R: %.4f", s.p.TargetR, target)
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.TakeProfit, sig.ATR = stop, target, atr
	sig.FillBy = t.Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s, дневной ATR %.4f, стоп под молотом %.4f − %.2f ATR: %.4f, %s (цель пересчитывается от цены покупки)",
		why, atr, low, s.p.StopBufferATR, stop, targetWhy)
	return sig
}

// series holds what the entry gates read: RSI(RSIPeriod), with TrendEMA on the trend EMA (nil when
// off or too short) and the daily ATR (atrOK false when the dailies cannot support it).
type series struct {
	rsi, trend []float64
	atr        float64
	atrOK      bool
}

func (s *Strategy) series(md strategy.MarketData) series {
	var ind series
	ind.atr, ind.atrOK = s.dailyATR(md)
	if s.p.RSIPeriod > 0 && len(md.Closes) > s.p.RSIPeriod {
		ind.rsi = indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	}
	if s.p.TrendEMA > 0 {
		ind.trend = ema.Compute(md.Closes, s.p.TrendEMA)
	}
	return ind
}

// hammerGates checks bar h as the signal hammer: the shape, its own RSI below RSIZone, the close
// strictly above a warmed trend EMA (TrendEMA on) and the raised volume (VolumeMult on). why
// describes the bar and the gates it passed.
func (s *Strategy) hammerGates(md strategy.MarketData, h int, ind series) (why string, ok bool) {
	o, hi, l, c := md.Opens[h], md.Highs[h], md.Lows[h], md.Closes[h]
	if !isHammer(o, hi, l, c) || ind.rsi == nil || h <= s.p.RSIPeriod {
		return "", false
	}
	rsi := ind.rsi[h]
	if rsi >= s.p.RSIZone {
		return "", false
	}
	trendWhy := ""
	if s.p.TrendEMA > 0 {
		e := ind.trend[h]
		// ema.Compute zero-fills the warm-up: an unwarmed EMA must not pass as "above zero"
		if e <= 0 || c <= e {
			return "", false
		}
		trendWhy = fmt.Sprintf(", close над EMA(%d) %.4f", s.p.TrendEMA, e)
	}
	vol, avg, volOK := s.volumeOK(md, h)
	if !volOK {
		return "", false
	}
	volWhy := ""
	if avg > 0 {
		volWhy = fmt.Sprintf(", объём %.0f > %.1f × среднего %.0f", vol, s.p.VolumeMult, avg)
	}
	levelWhy := ""
	if pdl, ok := s.atPDL(md, h, ind.atr); !ok {
		return "", false
	} else if s.p.PDLZoneATR > 0 {
		levelWhy = fmt.Sprintf(", low у вчерашнего минимума %.4f ± %.2f ATR", pdl, s.p.PDLZoneATR)
	}
	return fmt.Sprintf("молот %s на свече %s (o %.4f h %.4f l %.4f c %.4f), RSI(%d) %.1f < %.0f%s%s%s",
		s.tf(), md.Times[h].In(mskLoc).Format("01-02 15:04"), o, hi, l, c, s.p.RSIPeriod, rsi, s.p.RSIZone, trendWhy, volWhy, levelWhy), true
}

// confirmedHammer finds the hammer the current bar k confirms: k closes green above the high of a
// weekday hammer at most ConfirmBars bars back that passes hammerGates, and no bar between them
// confirmed it already (a green close above its high) or broke it (a low under its low). The
// latest such hammer wins.
func (s *Strategy) confirmedHammer(md strategy.MarketData, k int, ind series) (hb int, why string, ok bool) {
	o, c := md.Opens[k], md.Closes[k]
	if c <= o {
		return 0, "", false
	}
	for h := k - 1; h >= max(k-s.p.ConfirmBars, 0); h-- {
		if c <= md.Highs[h] || isWeekend(md.Times[h].In(mskLoc)) || !s.hammerOpen(md, h, k) {
			continue
		}
		if why, ok := s.hammerGates(md, h, ind); ok {
			return h, fmt.Sprintf("%s; подтверждение %s: зелёная, close %.4f > high молота %.4f",
				why, md.Times[k].In(mskLoc).Format("01-02 15:04"), c, md.Highs[h]), true
		}
	}
	return 0, "", false
}

// hammerOpen reports whether no bar strictly between the hammer h and k confirmed it (a green close
// above its high) or broke it (a low under its low).
func (s *Strategy) hammerOpen(md strategy.MarketData, h, k int) bool {
	for j := h + 1; j < k; j++ {
		if md.Lows[j] < md.Lows[h] || (md.Closes[j] > md.Opens[j] && md.Closes[j] > md.Highs[h]) {
			return false
		}
	}
	return true
}

// manage exits in priority order: the stop, the target while RSI has not closed above RSIHold, the
// close of the RSIHold bar when its histogram is not green, the close of a red MACD histogram bar
// after a green one since the hammer (or since the RSIHold bar). The position carries overnight.
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
	low, high, c := md.Lows[n-1], md.Highs[n-1], md.Closes[n-1]
	if low <= stop {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
		return sig
	}
	held := s.rsiHeld(md, n-2) // a cross on the current bar's close only follows its intrabar high
	if target > 0 && high >= target && held < 0 {
		sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", target
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цели %.4f (вход %.4f)", high, target, pos.PurchasePrice)
		return sig
	}
	if r, hist, ok := s.rsiExit(md); ok {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: свеча %s закрылась с RSI(%d) > %.0f, гистограмма MACD(%d,%d,%d) %.4f не зелёная, выход по %.4f (вход %.4f)",
			md.Times[r].In(mskLoc).Format("01-02 15:04"), s.p.RSIPeriod, s.p.RSIHold, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal, hist, c, pos.PurchasePrice)
		return sig
	}
	if hist, at, ok := s.macdTurnedRed(md); ok {
		sig.Kind, sig.Reason = model.SignalSell, "MACD"
		why := fmt.Sprintf("после зелёной %s, цель не взята", at.In(mskLoc).Format("01-02 15:04"))
		if s.rsiHeld(md, n-1) >= 0 {
			why = fmt.Sprintf("первая красная после RSI(%d) > %.0f на свече %s, цель снята", s.p.RSIPeriod, s.p.RSIHold, at.In(mskLoc).Format("01-02 15:04"))
		}
		sig.ExitReason = fmt.Sprintf("MACD: свеча %s гистограмма MACD(%d,%d,%d) %.4f < 0 %s, выход по %.4f (вход %.4f)",
			md.Times[n-1].In(mskLoc).Format("01-02 15:04"), s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal, hist, why, c, pos.PurchasePrice)
	}
	return sig
}

// entryIndex is the index of the position's entry bar: the first bar opening at or after its entry
// time (0 when the window starts later or the time is unset).
func entryIndex(md strategy.MarketData) int {
	for j := len(md.Times) - 1; j >= 0; j-- {
		if md.Times[j].Before(md.Position.EntryTime) {
			return j + 1
		}
	}
	return 0
}

// rsiHeld returns the first bar from the entry bar up to upto whose close has RSI(RSIPeriod) above
// RSIHold, or −1: none, RSIHold off, or the MACD exit off (without it a dropped target would leave
// only the stop).
func (s *Strategy) rsiHeld(md strategy.MarketData, upto int) int {
	if s.p.RSIHold <= 0 || s.p.RSIPeriod <= 0 || s.p.MACDFast <= 0 || len(md.Closes) <= s.p.RSIPeriod {
		return -1
	}
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	for j := max(entryIndex(md), s.p.RSIPeriod+1); j <= upto && j < len(rsi); j++ {
		if rsi[j] > s.p.RSIHold {
			return j
		}
	}
	return -1
}

// rsiExit reports whether the first close since the entry with RSI above RSIHold (rsiHeld) has a
// histogram bar that is not green (≤ 0): the bounce reached the upper zone without momentum under
// it, so the position sells at the current close. That bar is normally the current one; an earlier
// one (a position picked up late) sells at once too. It returns the RSIHold bar and its histogram.
func (s *Strategy) rsiExit(md strategy.MarketData) (r int, hist float64, ok bool) {
	r = s.rsiHeld(md, len(md.Closes)-1)
	if r < 0 {
		return 0, 0, false
	}
	h := s.macdHist(md.Closes)
	if r >= len(h) || h[r] > 0 {
		return 0, 0, false
	}
	return r, h[r], true
}

// macdHist returns the MACD histogram (MACD line − signal line) of the bar closes, nil when the
// exit is off or the history is too short.
func (s *Strategy) macdHist(closes []float64) []float64 {
	if s.p.MACDFast <= 0 {
		return nil
	}
	line, signal := indicators.MACD(closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal)
	if line == nil {
		return nil
	}
	hist := make([]float64, len(line))
	for i := range line {
		hist[i] = line[i] - signal[i]
	}
	return hist
}

// macdTurnedRed reports whether the last bar of a held position has a red (< 0) MACD histogram
// while a green (> 0) bar printed between the hammer (the bar before the entry) and the bar before
// the last: the bounce has faded without reaching the target. A histogram red since the hammer
// does not exit — after a fall it usually is, and the stop guards that stretch. Once RSI has closed
// above RSIHold (rsiHeld) no green bar is needed: the first red bar from that close on exits. It
// returns the last histogram value and the time of the latest green bar, or of the RSI bar.
func (s *Strategy) macdTurnedRed(md strategy.MarketData) (hist float64, at time.Time, ok bool) {
	h := s.macdHist(md.Closes)
	n := len(h)
	if n < 2 || len(md.Times) != n || h[n-1] >= 0 {
		return 0, time.Time{}, false
	}
	if r := s.rsiHeld(md, n-1); r >= 0 {
		return h[n-1], md.Times[r], true
	}
	for j := n - 2; j >= max(entryIndex(md)-1, 0); j-- {
		if h[j] > 0 {
			return h[n-1], md.Times[j], true
		}
	}
	return h[n-1], time.Time{}, false
}

// exitLevels returns the stop and the target of a position filled at f: the stop is the signal's
// level itself; the target, when the signal had one, is TargetR × (f − stop) above f (none once the
// fill is at or under the stop).
func (s *Strategy) exitLevels(pos *strategy.Position) (stop, target float64, ok bool) {
	f := pos.PurchasePrice
	if f <= 0 || pos.StopLoss <= 0 {
		return 0, 0, false
	}
	stop = pos.StopLoss
	if pos.TakeProfit > 0 && s.p.TargetR > 0 && f > stop {
		target = f + s.p.TargetR*(f-stop)
	}
	return stop, target, true
}
