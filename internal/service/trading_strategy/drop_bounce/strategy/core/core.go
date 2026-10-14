// Package core implements drop_bounce: a long-only 1h buy of an oversold RSI inside an uptrend
// within two daily buy windows. On the close of a bar whose fill bar (one step later, the next
// bar's open — Config.EntryAtNextOpen) falls on a Mon–Fri MSK weekday, on the same date, and starts
// inside the morning [MorningFrom, MorningTo) or the evening [EveningFrom, EveningTo) window
// (default 07:00–11:00 and 17:00–24:00 MSK), it buys when (1) the close is strictly above
// EMA(EMAPeriod); (2) RSI(RSIPeriod) is below RSIZone (ZONE), or has just crossed RSIZone upward
// and is not above RSICap (CROSS). The daily ATR (Wilder, DailyATRPeriod, completed weekday
// dailies) is frozen for the trade. Exits, in order: a stop StopDailyATR daily ATRs below the fill
// price; then at the bar's close RSI(RSIExitPeriod) crossing RSIExit upward whatever the price
// (reason RSI) — at any hour, the windows bound only entries; then an optional time exit
// (StaleBars, off by default) at the close of a position held StaleBars bars whose highs never
// reached StaleATR daily ATRs above the fill (reason STALE). No end-of-day exit: positions carry
// overnight and over weekends. Pure, stateless between bars and ticker-agnostic. Run with
// `-strategy drop_bounce -interval Hour1`. Owner rewrite 2026-10-14: the day-range gate, the MACD
// exit and the Friday rules were dropped.
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
	RSIPeriod int     // RSI length of the entry (default 6); 0 = no entries
	RSIZone   float64 // oversold level: ZONE below it, CROSS through it upward (default 30)
	RSICap    float64 // a CROSS counts only if RSI on the cross bar is ≤ this (default 50)
	EMAPeriod int     // trend EMA on the bar closes (default 200); 0 = no entries
	// StopDailyATR is the stop distance below the fill price in daily ATRs frozen at the signal;
	// 0 = off (default 1).
	StopDailyATR float64
	// DailyATRPeriod is the Wilder ATR length over completed weekday dailies (default 14).
	DailyATRPeriod int
	// RSIExitPeriod is the RSI length of the exit, separate from the entry's (default 10).
	RSIExitPeriod int
	// RSIExit is the level whose upward cross by RSI(RSIExitPeriod) sells at the bar's close;
	// RSIExit or RSIExitPeriod 0 = the RSI exit is off, only the stop (and STALE) remain (default 70).
	RSIExit float64
	// MorningFrom/MorningTo and EveningFrom/EveningTo are the buy windows: the fill bar must start
	// at an MSK hour h with From ≤ h < To in either window (default 7–11 and 17–24, owner
	// 2026-10-14). From ≥ To leaves that window empty.
	MorningFrom int
	MorningTo   int
	EveningFrom int
	EveningTo   int
	// StaleBars is the time exit: once the position has been held this many bars (the fill bar
	// counts) and no bar's high since the fill reached entry + StaleATR daily ATRs, it sells at the
	// bar's close with reason STALE; 0 = off (default 0, owner 2026-10-14).
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
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		RSIExitPeriod:  10,
		RSIExit:        70,
		MorningFrom:    7,
		MorningTo:      11,
		EveningFrom:    17,
		EveningTo:      24,
		StaleBars:      0,
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

// Lookback is enough history for the EMA (three lengths: the SMA seed has faded) and both Wilder
// RSIs to converge, plus the previous bar the crosses compare against.
func (s *Strategy) Lookback() int {
	return max(minLookback, 3*s.p.EMAPeriod, 4*s.p.RSIPeriod+2, 4*s.p.RSIExitPeriod+2)
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

// inWindow reports whether a signal on the bar starting at t may open a position: the fill bar
// (one step later) lies on the same MSK date, on a weekday, and starts inside the morning or the
// evening buy window.
func (s *Strategy) inWindow(t time.Time) bool {
	fill := t.Add(s.step)
	if isWeekend(fill) || dayKey(fill) != dayKey(t) {
		return false
	}
	return s.inBuyHours(fill.In(mskLoc).Hour())
}

// inBuyHours reports whether a fill at MSK hour h falls into either buy window.
func (s *Strategy) inBuyHours(h int) bool {
	p := s.p
	return (p.MorningFrom <= h && h < p.MorningTo) || (p.EveningFrom <= h && h < p.EveningTo)
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys an oversold RSI above the trend EMA inside a buy window. The signal carries no
// levels: the stop is measured from the fill. It carries the daily ATR for the engine to freeze;
// without a daily ATR there is no entry, since the stop and the time exit are measured in it.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if p.RSIPeriod <= 0 || p.EMAPeriod <= 0 || len(md.Times) != n {
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
	if atr <= 0 {
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
	sig.EntryReason = fmt.Sprintf("%s: RSI(%d) %.1f → %.1f (зона %.0f, планка %.0f), close %.4f > EMA(%d) %.4f, покупка в %s (ATR %.4f)",
		branch, p.RSIPeriod, prev, cur, p.RSIZone, p.RSICap, c, p.EMAPeriod, trend,
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
// min(stop, open)), then at the bar's close the RSI exit (manageRSI), then the time exit
// (manageStale). Exits run at any hour; there is no end-of-day exit.
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
	if sig = s.manageRSI(md, sig); sig.Kind == model.SignalSell {
		return sig
	}
	return s.manageStale(md, sig)
}

// manageRSI sells at the bar's close when RSI(RSIExitPeriod) crosses RSIExit upward, whatever the
// price. A cross on the fill bar counts (its close is a decision bar).
func (s *Strategy) manageRSI(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	// RSISeries is 0 up to index RSIExitPeriod−1, so the cross needs bars RSIExitPeriod+1 and later.
	if p.RSIExit <= 0 || p.RSIExitPeriod <= 0 || n < p.RSIExitPeriod+2 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIExitPeriod)
	i := n - 1
	if prev, cur := rsi[i-1], rsi[i]; crossUp(prev, cur, p.RSIExit) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) %.1f → %.1f пересёк %.0f вверх, выход по %.4f (вход %.4f)",
			p.RSIExitPeriod, prev, cur, p.RSIExit, md.Closes[i], md.Position.PurchasePrice)
	}
	return sig
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

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
