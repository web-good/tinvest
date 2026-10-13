// Package core implements drop_bounce: a long-only 1h buy of an oversold RSI inside an uptrend
// while the day still has range left. On the close of a Mon–Fri MSK bar starting at or after
// StartHour (on Friday before 12:00) whose fill bar lies on the same date, it buys when (1) the close is strictly above
// EMA(EMAPeriod); (2) RSI(RSIPeriod) is below RSIZone (ZONE), or has just crossed RSIZone upward
// and is not above RSICap (CROSS); (3) the day's range so far, TodayHigh − TodayLow, is below
// DaySpentATR daily ATRs (Wilder, DailyATRPeriod, completed weekday dailies). The backtest fills
// at the next bar's open (Config.EntryAtNextOpen); the daily ATR is frozen for the trade. Exits,
// in order: a stop StopDailyATR daily ATRs below the fill price; then RSI(RSIPeriod) crossing
// RSIExit upward whatever the price, which sells at once on a red MACD histogram and on a green one
// arms an exit on the histogram's next green→red flip; on Friday RSI at or above RSIExit sells at
// once whatever the histogram (reason FRI). A time exit (StaleBars, on by default)
// sells at the close of a position held StaleBars bars whose highs never reached StaleATR daily
// ATRs above the fill (reason STALE). No end-of-day exit: positions carry overnight. Pure, stateless between bars and ticker-agnostic. Run with `-strategy drop_bounce -interval Hour1`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback keeps 64 bars so the RSI converges even with a short EMA.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	RSIPeriod int     // RSI length of the entry and the exit (default 6); 0 = no entries
	RSIZone   float64 // oversold level: ZONE below it, CROSS through it upward (default 30)
	RSICap    float64 // a CROSS counts only if RSI on the cross bar is ≤ this (default 50)
	EMAPeriod int     // trend EMA on the bar closes (default 200); 0 = no entries
	// DaySpentATR is the share of the daily ATR the day's range may have used before entry: entry
	// needs TodayHigh − TodayLow < DaySpentATR · dailyATR (default 0.7); 0 = no entries.
	DaySpentATR float64
	// StopDailyATR is the stop distance below the fill price in daily ATRs frozen at the signal;
	// 0 = off (default 1).
	StopDailyATR float64
	// DailyATRPeriod is the Wilder ATR length over completed weekday dailies (default 14).
	DailyATRPeriod int
	// RSIExit is the RSI level whose upward cross triggers the exit logic: on a red MACD histogram
	// the position is sold at once, on a green one the exit is armed for the histogram's next
	// green→red flip; 0 = RSI and MACD exits off, only the stop remains (default 70).
	RSIExit   float64
	StartHour int // signal bars must start at/after StartHour:00 MSK (default 7)
	// MACDFast, MACDSlow and MACDSignal are the MACD(12,26,9) periods whose histogram
	// (MACD line − signal line) colours the RSI cross and times the armed exit. Invalid periods
	// (fast ≥ slow, a non-positive one) leave no histogram: it then counts as red.
	MACDFast   int
	MACDSlow   int
	MACDSignal int
	// StaleBars is the time exit: once the position has been held this many bars (the fill bar
	// counts) and no bar's high since the fill reached entry + StaleATR daily ATRs, it sells at the
	// bar's close with reason STALE; 0 = off (default 10, owner 2026-10-13).
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
		EMAPeriod:      200,
		DaySpentATR:    0.7,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		RSIExit:        70,
		StartHour:      7,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		StaleBars:      10,
		StaleATR:       0.3,
	}
}

// Strategy trades a single instrument with the drop_bounce rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	// step is the bar timeframe (1h in production; tests and the screener pass it). It bounds the
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

// Lookback is enough history for the EMA (three lengths: the SMA seed has faded), the Wilder RSI
// and the MACD (three slow+signal lengths) to converge, plus the previous bar the cross compares
// against.
func (s *Strategy) Lookback() int {
	return max(minLookback, 3*s.p.EMAPeriod, 4*s.p.RSIPeriod+2, 3*(s.p.MACDSlow+s.p.MACDSignal))
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

// fridayLastEntryHour closes Friday entries: a Friday signal bar must start before 12:00 MSK, so
// the latest Friday buy is at the 12:00 bar's open (owner, 2026-10-13).
const fridayLastEntryHour = 12

// isFriday reports whether t falls on Friday MSK.
func isFriday(t time.Time) bool { return t.In(mskLoc).Weekday() == time.Friday }

// inWindow reports whether a signal on the bar starting at t may open a position: a weekday, at
// or after StartHour (on Friday before fridayLastEntryHour), and the fill bar (one step later) on
// the same MSK date.
func (s *Strategy) inWindow(t time.Time) bool {
	h := t.In(mskLoc).Hour()
	if isWeekend(t) || h < s.p.StartHour || (isFriday(t) && h >= fridayLastEntryHour) {
		return false
	}
	return dayKey(t.Add(s.step)) == dayKey(t)
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys an oversold RSI above the trend EMA while the day still has range left. The signal
// carries no levels: the stop is measured from the fill. It carries the daily ATR for the engine
// to freeze. Without a daily ATR or a sane day extent there is no entry — the day gate needs both.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if p.RSIPeriod <= 0 || p.EMAPeriod <= 0 || p.DaySpentATR <= 0 || len(md.Times) != n {
		return sig
	}
	// RSISeries is 0 up to index RSIPeriod−1: the cross needs bars RSIPeriod and later, and a
	// warm-up zero must never read as "below the zone".
	if n < p.RSIPeriod+2 {
		return sig
	}
	i := n - 1
	c := md.Closes[i]
	if !s.inWindow(md.Times[i]) {
		return sig
	}
	trend := ema.Compute(md.Closes, p.EMAPeriod)[i]
	if !trendUp(c, trend) {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)
	prev, cur := rsi[i-1], rsi[i]
	var branch string
	switch {
	case cur < p.RSIZone:
		branch = "ZONE"
	case crossUp(prev, cur, p.RSIZone) && cur <= p.RSICap:
		branch = "CROSS"
	default:
		return sig
	}
	atr := s.dailyATR(md)
	hi, lo := md.TodayHigh, md.TodayLow
	if atr <= 0 || hi <= 0 || lo <= 0 || hi < lo {
		return sig
	}
	used := hi - lo
	if used >= p.DaySpentATR*atr {
		return sig
	}
	if p.StopDailyATR > 0 && c-p.StopDailyATR*atr <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.ATR = atr
	// The fill must be the very next slot: if that candle is missing from the data, the engine
	// drops the signal rather than buying at a later session's open.
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s: RSI(%d) %.1f → %.1f (зона %.0f, планка %.0f), close %.4f > EMA(%d) %.4f, ход дня %.4f = %.2f ATR < %.2f (ATR %.4f)",
		branch, p.RSIPeriod, prev, cur, p.RSIZone, p.RSICap, c, p.EMAPeriod, trend, used, used/atr, p.DaySpentATR, atr)
	return sig
}

// trendUp reports whether the close sits strictly above a warmed EMA. ema.Compute zero-fills
// warm-up positions, so an unwarmed EMA must not pass as "price above zero".
func trendUp(closeP, emaNow float64) bool {
	return emaNow > 0 && closeP > emaNow
}

// manage exits in priority order: the stop StopDailyATR daily ATRs below the fill price (the ATR
// frozen at entry, today's ATR when the position carries none; intrabar, filled by the engine at
// min(stop, open)), then at the bar's close the RSI/MACD exit (manageRSIMACD), then the time exit
// (manageStale). There is no end-of-day exit.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Lows) != n || pos.PurchasePrice <= 0 {
		return sig
	}
	i := n - 1
	entry := pos.PurchasePrice
	if s.p.StopDailyATR > 0 {
		atr := pos.EntryATR
		if atr <= 0 {
			atr = s.dailyATR(md)
		}
		// The tiny epsilon keeps a low of exactly the stop from slipping on float rounding.
		if stop := entry - s.p.StopDailyATR*atr; atr > 0 && stop > 0 && md.Lows[i] <= stop+1e-9 {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f, −%.2f ATR %.4f = −%.2f%%)",
				md.Lows[i], stop, entry, s.p.StopDailyATR, atr, (1-stop/entry)*100)
			return sig
		}
	}
	if s.p.RSIExit > 0 && s.p.RSIPeriod > 0 {
		if sig = s.manageRSIMACD(md, sig); sig.Kind == model.SignalSell {
			return sig
		}
	}
	return s.manageStale(md, sig)
}

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

// manageRSIMACD is the exit on the bar's close once the stop has not fired. RSI(RSIPeriod) crossing
// RSIExit upward sells at once when the MACD histogram on that bar is red; on a green histogram it
// only arms the exit, and the next green→red flip of the histogram sells with reason MACD. Arming is
// recomputed from history on every call: a cross at bar j ≥ the fill bar with hist[j] ≥ 0 and j < i
// arms. A cross on the fill bar counts. An EntryTime before the first bar (or zero) scans from the
// first bar where the cross is evaluable. Without a histogram (invalid periods, too short history)
// it counts as red: the cross sells at once and MACD never arms. Arming is a function of the visible
// window: a position held more than Lookback bars past its arming cross loses it, and a scan that
// starts at the window's first bar reads MACD close to its warm-up.
func (s *Strategy) manageRSIMACD(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	pos := md.Position
	n := len(md.Closes)
	i := n - 1
	// RSISeries is 0 up to index RSIPeriod−1, so the cross needs bars RSIPeriod+1 and later.
	if n < p.RSIPeriod+2 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)
	hist := macdHistogram(md.Closes, p)
	red := func(k int) bool { return hist == nil || isRed(hist[k]) }
	entry := pos.PurchasePrice
	if prev, cur := rsi[i-1], rsi[i]; crossUp(prev, cur, p.RSIExit) && red(i) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		histText := "нет (считается красной)"
		if hist != nil {
			histText = fmt.Sprintf("%.4f < 0", hist[i])
		}
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) %.1f → %.1f пересёк %.0f вверх, гистограмма MACD %s, выход по %.4f (вход %.4f)",
			p.RSIPeriod, prev, cur, p.RSIExit, histText, md.Closes[i], entry)
		return sig
	}
	// On Friday RSI at or above RSIExit sells at once whatever the histogram: no wait for a red
	// flip into the weekend. Entries need RSI ≤ RSICap, so RSI up here means a cross after entry.
	if len(md.Times) == n && isFriday(md.Times[i]) && rsi[i] >= p.RSIExit {
		sig.Kind, sig.Reason = model.SignalSell, "FRI"
		sig.ExitReason = fmt.Sprintf("FRI: пятница, RSI(%d) %.1f ≥ %.0f — выход без ожидания MACD по %.4f (вход %.4f)",
			p.RSIPeriod, rsi[i], p.RSIExit, md.Closes[i], entry)
		return sig
	}
	if hist == nil || i < 1 || isRed(hist[i-1]) || !isRed(hist[i]) {
		return sig
	}
	from := p.RSIPeriod + 1
	if len(md.Times) == n && !pos.EntryTime.IsZero() {
		fill := n // no bar at or after the entry time: nothing to scan
		for k, t := range md.Times {
			if !t.Before(pos.EntryTime) {
				fill = k
				break
			}
		}
		from = max(from, fill)
	}
	for j := from; j < i; j++ {
		if crossUp(rsi[j-1], rsi[j], p.RSIExit) && !red(j) {
			sig.Kind, sig.Reason = model.SignalSell, "MACD"
			armBar := fmt.Sprintf("индекс %d", j) // Times may be empty (live), then only the index is known
			if j < len(md.Times) {
				armBar = md.Times[j].In(mskLoc).Format("2006-01-02 15:04")
			}
			sig.ExitReason = fmt.Sprintf("MACD: гистограмма %.4f → %.4f перекрасилась в красную после RSI ≥ %.0f (бар %s), выход по %.4f (вход %.4f)",
				hist[i-1], hist[i], p.RSIExit, armBar, md.Closes[i], entry)
			return sig
		}
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

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
