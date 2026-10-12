// Package core implements drop_bounce: a long-only bounce after a red bar that trades a sizeable
// share of an average day's volume. A bar closing below its open, whose volume reaches
// VolDailyMult times the average weekday daily volume (Wilder over VolAvgPeriod completed
// Mon–Fri dailies, zero-volume days left out), signals a buy; the backtest fills at the next
// bar's open (Config.EntryAtNextOpen). Signals are taken on Mon–Fri MSK bars starting at or after
// StartHour whose fill bar lies on the same date. The daily ATR (Wilder, DailyATRPeriod,
// completed weekday dailies) is taken at the signal and frozen for the trade. Exits, in order: a
// stop StopDailyATR daily ATRs below the fill price, then RSI(RSIPeriod) crossing RSIExit upward
// whatever the price. There is no end-of-day exit: positions carry overnight. Pure, stateless
// between bars and ticker-agnostic. The bar step (5m, 15m, 30m or 1h) is fixed at construction.
// Run with `-strategy drop_bounce -interval Minutes5|Minutes15|Minutes30|Hour1`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback keeps 64 bars (16 hours of 15m) so the RSI converges.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	// VolDailyMult is the share of the average weekday daily volume the signal bar must trade
	// (default 0.4); 0 = no entries.
	VolDailyMult float64
	// VolAvgPeriod is the Wilder length of the average over completed weekday dailies with
	// nonzero volume (default 14).
	VolAvgPeriod int
	RSIPeriod    int     // RSI length of the exit (default 6)
	RSIExit      float64 // sell when RSI crosses this upward; 0 = off (default 70)
	// StopDailyATR is the stop distance below the fill price in daily ATRs frozen at the signal;
	// 0 = off (default 1).
	StopDailyATR float64
	// DailyATRPeriod is the Wilder ATR length over completed weekday dailies (default 14).
	DailyATRPeriod int
	StartHour      int // signal bars must start at/after StartHour:00 MSK (default 8)
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		VolDailyMult:   0.4,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        70,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		StartHour:      8,
	}
}

// Strategy trades a single instrument with the drop_bounce rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	// step is the bar timeframe. It matters only for the entry: the fill bar starts one step after
	// the signal bar, must lie on the same MSK date, and bounds Signal.FillBy.
	step time.Duration
	p    Params
}

// NewWithParams returns the strategy for a ticker on bars of the given step with explicit params.
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback is enough history for the Wilder RSI to converge plus the previous bar its cross
// compares against.
func (s *Strategy) Lookback() int { return max(minLookback, 4*s.p.RSIPeriod+2) }

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

// avgDailyVolume is the Wilder average of the completed weekday daily volumes: weekend sessions
// and zero-volume days are left out (both would drag the yardstick down and fire the threshold on
// nothing), the first VolAvgPeriod values seed it as a plain mean, each later one moves it by
// (v − avg)/VolAvgPeriod. It is 0 — unknown — when the volumes are missing or not aligned with
// DailyTimes, or fewer than VolAvgPeriod days are left.
func (s *Strategy) avgDailyVolume(md strategy.MarketData) float64 {
	n := s.p.VolAvgPeriod
	if n <= 0 || len(md.DailyVolumes) == 0 || len(md.DailyVolumes) != len(md.DailyTimes) {
		return 0
	}
	vols := make([]float64, 0, len(md.DailyVolumes))
	for i, v := range md.DailyVolumes {
		if v > 0 && !isWeekend(md.DailyTimes[i]) {
			vols = append(vols, float64(v))
		}
	}
	if len(vols) < n {
		return 0
	}
	var avg float64
	for _, v := range vols[:n] {
		avg += v
	}
	avg /= float64(n)
	for _, v := range vols[n:] {
		avg += (v - avg) / float64(n)
	}
	return avg
}

// inWindow reports whether a signal on the bar starting at t may open a position: a weekday, at
// or after StartHour, and the fill bar (one step later) on the same MSK date.
func (s *Strategy) inWindow(t time.Time) bool {
	if isWeekend(t) || t.In(mskLoc).Hour() < s.p.StartHour {
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

// enter buys after a red bar whose volume reaches VolDailyMult average days. The signal carries no
// levels: the stop is measured from the fill. It carries the daily ATR for the engine to freeze.
// Without a known average daily volume there is no entry — the volume is the signal itself — and
// neither is there one when the stop needs an unknown ATR or would land at or below zero.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n || len(md.Volumes) != n || p.VolDailyMult <= 0 {
		return sig
	}
	i := n - 1
	o, c := md.Opens[i], md.Closes[i]
	if !s.inWindow(md.Times[i]) || o <= 0 || c >= o {
		return sig
	}
	avg := s.avgDailyVolume(md)
	if avg <= 0 {
		return sig
	}
	ratio := float64(md.Volumes[i]) / avg
	// The tiny epsilon keeps a volume of exactly the threshold from failing on float rounding.
	if ratio < p.VolDailyMult-1e-9 {
		return sig
	}
	atr := s.dailyATR(md)
	if p.StopDailyATR > 0 && (atr <= 0 || c-p.StopDailyATR*atr <= 0) {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.ATR = atr
	// The fill must be the very next slot: if that candle is missing from the data, the engine
	// drops the signal rather than buying at a later session's open.
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("VOLUME: open %.4f → close %.4f, объём %d = %.2f× среднего дня (%.0f за %d дн., порог %.2f), дневной ATR %.4f",
		o, c, md.Volumes[i], ratio, avg, p.VolAvgPeriod, p.VolDailyMult, atr)
	return sig
}

// manage exits in priority order: the stop StopDailyATR daily ATRs below the fill price (the ATR
// frozen at entry, today's ATR when the position carries none; intrabar, filled by the engine at
// min(stop, open)), then at the bar's close RSI(RSIPeriod) crossing RSIExit upward, whatever the
// price. There is no end-of-day exit.
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
	// RSISeries is 0 up to index RSIPeriod−1, so the cross needs bars RSIPeriod and later.
	if s.p.RSIExit > 0 && s.p.RSIPeriod > 0 && n >= s.p.RSIPeriod+2 {
		rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
		if prev, cur := rsi[i-1], rsi[i]; crossUp(prev, cur, s.p.RSIExit) {
			sig.Kind, sig.Reason = model.SignalSell, "RSI"
			sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) %.1f → %.1f пересёк %.0f вверх, выход по %.4f (вход %.4f)",
				s.p.RSIPeriod, prev, cur, s.p.RSIExit, md.Closes[i], entry)
		}
	}
	return sig
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
