// Package core implements drop_bounce: a long-only buy of an oversold hourly RSI inside an hourly
// uptrend, timed by the 30-minute MACD. It runs on 30m bars and builds the hourly series itself
// from them (hourBars): an hour is complete at the close of its :30 bar, or at the first bar of a
// later hour when that bar is missing. When an hour completes with (1) its close strictly above
// EMA(EMAPeriod) of hourly closes and (2) the hourly RSI(RSIPeriod) below RSIZone (ZONE) or just
// crossed RSIZone upward and not above RSICap (CROSS), the entry is armed for ConfirmBars 30m bars
// starting with the bar that completed the hour. It buys at the next 30m bar's open
// (Config.EntryAtNextOpen) on the first armed bar whose MACD(MACDFast, MACDSlow, MACDSignal)
// histogram on 30m closes is green (≥ 0): the arming bar itself if already green, otherwise the
// red→green flip. The fill bar must be a Mon–Fri MSK bar on the signal bar's date, on Friday
// starting before FridayTo (default 11:00); there are no other time limits. The daily ATR (Wilder, DailyATRPeriod, completed weekday dailies) is frozen for
// the trade. Exits, in order: a stop StopDailyATR daily ATRs below the fill price (intrabar on 30m);
// then, on the 30m bar that completes an hour held in the position, the hourly RSI(RSIExitPeriod)
// crossing RSIExit upward (reason RSI); then an optional time exit (StaleBars 30m bars, off by
// default; reason STALE). No end-of-day exit: positions carry overnight and over weekends. Pure,
// stateless between bars and ticker-agnostic. Run with `-strategy drop_bounce -interval Minutes30`.
// Owner rewrite 2026-10-14: hourly signal, 30-minute MACD confirmation, weekdays only, Friday
// buys before 11:00.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback keeps 64 hourly bars so the RSI converges even with a short EMA.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	RSIPeriod int     // hourly RSI length of the entry (default 6); 0 = no entries
	RSIZone   float64 // oversold level: ZONE below it, CROSS through it upward (default 30)
	RSICap    float64 // a CROSS counts only if RSI on the cross hour is ≤ this (default 50)
	// ZoneEntry 1 lets the ZONE branch (RSI still below RSIZone) arm entries; 0 leaves only CROSS,
	// the buy after the RSI turned up (default 1).
	ZoneEntry int
	EMAPeriod int // trend EMA on hourly closes (default 200); 0 = no entries
	// StopDailyATR is the stop distance below the fill price in daily ATRs frozen at the signal;
	// 0 = off (default 1).
	StopDailyATR float64
	// BreakevenATR raises the stop once a completed bar since the fill had its high at entry +
	// BreakevenATR daily ATRs: to the entry price, or, with TrailATR > 0, to the best such high −
	// TrailATR ATRs when that is higher (reason BE / TRAIL); 0 = off (default 0.5, owner 2026-10-14).
	BreakevenATR float64
	// TrailATR is the trailing distance below the best high after BreakevenATR fired; 0 = the
	// stop stays at the entry price (default 0).
	TrailATR float64
	// DailyATRPeriod is the Wilder ATR length over completed weekday dailies (default 14).
	DailyATRPeriod int
	// RSIExitPeriod is the hourly RSI length of the exit, separate from the entry's (default 10).
	RSIExitPeriod int
	// RSIExit is the level whose upward cross by the hourly RSI(RSIExitPeriod) sells; RSIExit or
	// RSIExitPeriod 0 = the RSI exit is off, only the stop (and STALE) remain (default 70).
	RSIExit float64
	// MACDFast, MACDSlow and MACDSignal are the MACD(12,26,9) periods on 30m closes whose histogram
	// (MACD line − signal line) confirms the entry. Invalid periods (fast ≥ slow, a non-positive
	// one) leave no histogram: then nothing confirms and there are no entries.
	MACDFast   int
	MACDSlow   int
	MACDSignal int
	// ConfirmBars is how many 30m bars, starting with the one that completed the signal hour, the
	// entry waits for a green histogram (default 3 = an hour and a half); 0 = no entries.
	ConfirmBars int
	// FridayTo closes Friday buys: on Friday the fill bar must start before FridayTo:00 MSK
	// (default 11, owner 2026-10-14); 24 = no Friday limit, 0 = no Friday buys.
	FridayTo int
	// StaleBars is the time exit: once the position has been held this many 30m bars (the fill bar
	// counts) and no bar's high since the fill reached entry + StaleATR daily ATRs, it sells at the
	// bar's close with reason STALE; 0 = off (default 0).
	StaleBars int
	// StaleATR is the favourable move in daily ATRs (frozen at entry) that keeps a position from the
	// STALE exit (default 0.3; 0 means any high above the fill keeps it).
	StaleATR float64
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   1,
		BreakevenATR:   0.5,
		TrailATR:       0,
		DailyATRPeriod: 14,
		RSIExitPeriod:  10,
		RSIExit:        70,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		ConfirmBars:    3,
		FridayTo:       11,
		StaleBars:      0,
		StaleATR:       0.3,
	}
}

// Strategy trades a single instrument with the drop_bounce rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	// step is the bar timeframe (30m in production; tests and the screener pass it). It bounds the
	// fill: the fill bar starts one step after the signal bar, must lie on the same MSK date, and
	// bounds Signal.FillBy.
	step time.Duration
	p    Params
}

// NewWithParams returns the strategy for a ticker on bars of the given step with explicit params.
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// barsPerHour is how many bars of the strategy's step make an hour (2 on 30m), at least 1.
func (s *Strategy) barsPerHour() int {
	if s.step <= 0 || s.step >= time.Hour {
		return 1
	}
	return int(time.Hour / s.step)
}

// Lookback is enough 30m bars for the hourly EMA (three lengths) and both hourly Wilder RSIs to
// converge, plus the 30m MACD (three slow+signal lengths) and the confirmation window.
func (s *Strategy) Lookback() int {
	p := s.p
	hours := max(minLookback, 3*p.EMAPeriod, 4*p.RSIPeriod+2, 4*p.RSIExitPeriod+2)
	return max(s.barsPerHour()*(hours+1), 3*(p.MACDSlow+p.MACDSignal)+p.ConfirmBars+1)
}

// hourBar is one hour aggregated from the strategy's bars.
type hourBar struct {
	start time.Time // the hour's start
	close float64   // the close of its last bar
	last  int       // index of its last bar in the window
	done  int       // index of the bar at whose close the hour is known complete
}

// hourBars aggregates the window into completed hours, oldest-first. An hour is complete at the
// close of the bar that ends it (start + step reaches the next hour), or, when that bar is missing,
// at the first bar of a later hour. The still-forming hour is left out.
func (s *Strategy) hourBars(md strategy.MarketData) []hourBar {
	n := len(md.Closes)
	out := make([]hourBar, 0, n/s.barsPerHour()+1)
	var cur *hourBar
	for k := range n {
		h := md.Times[k].Truncate(time.Hour)
		if cur != nil && !h.Equal(cur.start) {
			cur.done = k // the hour lacked its closing bar: known complete at this later bar
			out = append(out, *cur)
			cur = nil
		}
		if cur == nil {
			cur = &hourBar{start: h}
		}
		cur.close, cur.last = md.Closes[k], k
		if !md.Times[k].Add(s.step).Before(h.Add(time.Hour)) {
			cur.done = k
			out = append(out, *cur)
			cur = nil
		}
	}
	return out
}

func hourCloses(hs []hourBar) []float64 {
	out := make([]float64, len(hs))
	for k, h := range hs {
		out[k] = h.close
	}
	return out
}

// mskLoc anchors every calendar rule to Moscow (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// dayKey is the MSK calendar date of t as yyyymmdd.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
}

// isWeekend reports whether t falls on Saturday or Sunday MSK.
func isWeekend(t time.Time) bool {
	wd := t.In(mskLoc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// weekdayDaily drops weekend (Sat/Sun MSK) bars from the daily series, keeping the three price
// slices aligned: MOEX weekend sessions are several times narrower and thinner than weekday ones
// and would understate the ATR. Without aligned times there is nothing to filter by and the
// series is returned untouched.
func weekdayDaily(highs, lows, closes []float64, times []time.Time) (h, l, c []float64) {
	n := len(closes)
	if n == 0 || len(highs) != n || len(lows) != n || len(times) != n {
		return highs, lows, closes
	}
	h, l, c = make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	for i := range n {
		if isWeekend(times[i]) {
			continue
		}
		h, l, c = append(h, highs[i]), append(l, lows[i]), append(c, closes[i])
	}
	return h, l, c
}

// dailyATR is Wilder's ATR over the completed weekday dailies the engine exposes (no lookahead),
// or 0 when the data cannot support it.
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

// isWeekdayFill reports whether a signal on the bar starting at t may fill: the fill bar (one step
// later) lies on the same MSK date, on a weekday and, on Friday, starts before FridayTo:00.
func (s *Strategy) isWeekdayFill(t time.Time) bool {
	fill := t.Add(s.step)
	return !isWeekend(fill) && dayKey(fill) == dayKey(t) && s.fridayOK(fill)
}

// fridayOK reports whether a fill bar starting at t passes the Friday cutoff FridayTo.
func (s *Strategy) fridayOK(t time.Time) bool {
	tl := t.In(mskLoc)
	return tl.Weekday() != time.Friday || tl.Hour() < s.p.FridayTo
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// hourSignal is the hourly entry condition on hour k of hs: the branch ("ZONE"/"CROSS") or "".
func (s *Strategy) hourSignal(closes, trend, rsi []float64, k int) string {
	p := s.p
	if k < p.RSIPeriod+1 || !trendUp(closes[k], trend[k]) {
		return ""
	}
	switch {
	case rsi[k] < p.RSIZone:
		if p.ZoneEntry == 0 {
			return ""
		}
		return "ZONE"
	case crossUp(rsi[k-1], rsi[k], p.RSIZone) && rsi[k] <= p.RSICap:
		return "CROSS"
	}
	return ""
}

// armedBy returns the hour that arms an entry on the current bar i — the latest completed hour with
// the entry condition whose confirmation window [done, done+ConfirmBars) holds i — or −1.
func (s *Strategy) armedBy(hs []hourBar, closes, trend, rsi []float64, i int) int {
	for k := len(hs) - 1; k >= 0 && i-hs[k].done < s.p.ConfirmBars; k-- {
		if hs[k].done <= i && s.hourSignal(closes, trend, rsi, k) != "" {
			return k
		}
	}
	return -1
}

// confirmed reports whether the 30m histogram confirms on bar i of a window armed at bar a: green
// on i, and i is the arming bar or the histogram was red on the bar before.
func confirmed(hist []float64, a, i int) bool {
	return hist != nil && !isRed(hist[i]) && (i == a || isRed(hist[i-1]))
}

// enter buys at the next bar's open once an armed hourly signal is confirmed by a green 30m MACD
// histogram. The signal carries no levels: the stop is measured from the fill. It carries the
// daily ATR for the engine to freeze; without a daily ATR there is no entry.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if p.RSIPeriod <= 0 || p.EMAPeriod <= 0 || p.ConfirmBars <= 0 || n < 2 || len(md.Times) != n {
		return sig
	}
	i := n - 1
	if !s.isWeekdayFill(md.Times[i]) {
		return sig
	}
	hs := s.hourBars(md)
	if len(hs) < p.RSIPeriod+2 {
		return sig
	}
	closes := hourCloses(hs)
	trend := ema.Compute(closes, p.EMAPeriod)
	rsi := indicators.RSISeries(closes, p.RSIPeriod)
	k := s.armedBy(hs, closes, trend, rsi, i)
	if k < 0 {
		return sig
	}
	hist := macdHistogram(md.Closes, p)
	if !confirmed(hist, hs[k].done, i) {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	c := md.Closes[i]
	if p.StopDailyATR > 0 && c-p.StopDailyATR*atr <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.ATR = atr
	// The fill must be the very next slot: if that candle is missing from the data, the engine
	// drops the signal rather than buying at a later session's open.
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s: час %s RSI(%d) %.1f → %.1f (зона %.0f, планка %.0f), close %.4f > EMA(%d) %.4f; MACD 30m %.4f зелёная через %d бар(а), покупка в %s (ATR %.4f)",
		s.hourSignal(closes, trend, rsi, k), hs[k].start.In(mskLoc).Format("01-02 15:04"), p.RSIPeriod, rsi[k-1], rsi[k],
		p.RSIZone, p.RSICap, closes[k], p.EMAPeriod, trend[k], hist[i], i-hs[k].done,
		sig.FillBy.In(mskLoc).Format("15:04"), atr)
	return sig
}

// trendUp reports whether the close sits strictly above a warmed EMA. ema.Compute zero-fills
// warm-up positions, so an unwarmed EMA must not pass as "price above zero".
func trendUp(closeP, emaNow float64) bool {
	return emaNow > 0 && closeP > emaNow
}

// manage exits in priority order: the stop StopDailyATR daily ATRs below the fill price (the ATR
// frozen at entry, today's ATR when the position carries none; intrabar, filled by the engine at
// min(stop, open)), then the hourly RSI exit (manageRSI), then the time exit (manageStale). There
// is no end-of-day exit.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Lows) != n || pos.PurchasePrice <= 0 {
		return sig
	}
	i := n - 1
	entry := pos.PurchasePrice
	atr := pos.EntryATR
	if atr <= 0 {
		atr = s.dailyATR(md)
	}
	if s.p.StopDailyATR > 0 {
		// The tiny epsilon keeps a low of exactly the stop from slipping on float rounding.
		if stop := entry - s.p.StopDailyATR*atr; atr > 0 && stop > 0 && md.Lows[i] <= stop+1e-9 {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f, −%.2f ATR %.4f = −%.2f%%)",
				md.Lows[i], stop, entry, s.p.StopDailyATR, atr, (1-stop/entry)*100)
			return sig
		}
	}
	if sig = s.manageBreakeven(md, sig, atr); sig.Kind == model.SignalSell {
		return sig
	}
	if sig = s.manageRSI(md, sig); sig.Kind == model.SignalSell {
		return sig
	}
	return s.manageStale(md, sig)
}

// manageBreakeven is the raised stop: once a completed bar since the fill (fill..i−1, so the current
// bar's own high never arms its own stop) reached entry + BreakevenATR ATRs, the stop is the entry
// price, or best high − TrailATR ATRs when TrailATR > 0 and that is higher. Intrabar on bar i; the
// engine fills at min(stop, open). Without a known fill bar in the window it does not fire.
func (s *Strategy) manageBreakeven(md strategy.MarketData, sig model.Signal, atr float64) model.Signal {
	p := s.p
	fill := fillIndex(md)
	i := len(md.Closes) - 1
	if p.BreakevenATR <= 0 || atr <= 0 || fill < 0 || i <= fill || len(md.Highs) != len(md.Closes) {
		return sig
	}
	entry := md.Position.PurchasePrice
	best := md.Highs[fill]
	for _, h := range md.Highs[fill+1 : i] {
		best = max(best, h)
	}
	if best < entry+p.BreakevenATR*atr-1e-9 {
		return sig
	}
	stop, reason := entry, "BE"
	if p.TrailATR > 0 {
		if t := best - p.TrailATR*atr; t > entry {
			stop, reason = t, "TRAIL"
		}
	}
	if md.Lows[i] > stop+1e-9 {
		return sig
	}
	sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, reason, stop
	sig.ExitReason = fmt.Sprintf("%s: low %.4f ≤ стоп %.4f (вход %.4f, лучший high %.4f ≥ вход + %.2f ATR %.4f)",
		reason, md.Lows[i], stop, entry, best, p.BreakevenATR, atr)
	return sig
}

// manageRSI sells at the close of the bar that completes an hour when the hourly RSI(RSIExitPeriod)
// crossed RSIExit upward on that hour, whatever the price. The hour must end at or after the entry
// (its last bar starts at or after EntryTime; a zero EntryTime passes), so a cross the position
// was bought after does not sell it.
func (s *Strategy) manageRSI(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if p.RSIExit <= 0 || p.RSIExitPeriod <= 0 || len(md.Times) != n {
		return sig
	}
	hs := s.hourBars(md)
	k := len(hs) - 1
	if k < p.RSIExitPeriod+1 || hs[k].done != n-1 || md.Times[hs[k].last].Before(md.Position.EntryTime) {
		return sig
	}
	rsi := indicators.RSISeries(hourCloses(hs), p.RSIExitPeriod)
	if prev, cur := rsi[k-1], rsi[k]; crossUp(prev, cur, p.RSIExit) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: час %s RSI(%d) %.1f → %.1f пересёк %.0f вверх, выход по %.4f (вход %.4f)",
			hs[k].start.In(mskLoc).Format("01-02 15:04"), p.RSIExitPeriod, prev, cur, p.RSIExit, md.Closes[n-1], md.Position.PurchasePrice)
	}
	return sig
}

// macdHistogram is MACD line − signal line over closes, or nil when the periods are invalid or the
// history is too short.
func macdHistogram(closes []float64, p Params) []float64 {
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

// isRed reports whether a histogram value is red: strictly below zero, so exactly 0 is green.
func isRed(h float64) bool { return h < 0 }

// fillIndex is the index of the first bar at or after the entry time, or −1 when the entry time is
// unknown (zero, no aligned times) or before the visible window (the fill bar is not in it).
func fillIndex(md strategy.MarketData) int {
	pos := md.Position
	if pos.EntryTime.IsZero() || len(md.Times) != len(md.Closes) || len(md.Times) == 0 ||
		md.Times[0].After(pos.EntryTime) {
		return -1
	}
	for k, t := range md.Times {
		if !t.Before(pos.EntryTime) {
			return k
		}
	}
	return -1
}

// manageStale is the time exit at the bar's close: StaleBars bars held (the fill bar counts) and no
// high since the fill reached entry + StaleATR daily ATRs (frozen at entry, today's when the
// position carries none). Without a known fill bar in the window it does not fire.
func (s *Strategy) manageStale(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	if p.StaleBars <= 0 || len(md.Highs) != len(md.Closes) {
		return sig
	}
	fill := fillIndex(md)
	i := len(md.Closes) - 1
	if fill < 0 || i-fill+1 < p.StaleBars {
		return sig
	}
	pos := md.Position
	atr := pos.EntryATR
	if atr <= 0 {
		atr = s.dailyATR(md)
	}
	if atr <= 0 {
		return sig
	}
	best := md.Highs[fill]
	for _, h := range md.Highs[fill+1:] {
		best = max(best, h)
	}
	target := pos.PurchasePrice + p.StaleATR*atr
	if best >= target {
		return sig
	}
	sig.Kind, sig.Reason = model.SignalSell, "STALE"
	sig.ExitReason = fmt.Sprintf("STALE: %d баров, лучший high %.4f < вход %.4f + %.2f ATR %.4f = %.4f, выход по %.4f",
		i-fill+1, best, pos.PurchasePrice, p.StaleATR, atr, target, md.Closes[i])
	return sig
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
