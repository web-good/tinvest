// Package core implements gap_fade: a long-only intraday fade of a morning gap down on
// 30-minute bars. When the day's first bar opens at least GapATR daily ATRs below the previous
// bar's close (the last trade before the session, weekend sessions included) and bar number
// EntryBar of the day still closes below that previous close, it buys that close. The target is
// a TargetFill share of the remaining gap, the stop StopDailyATR daily ATRs below the entry; both
// are frozen on the position. Whatever is still open at the first bar starting at or after
// EODHour:EODMinute MSK — or on the first bar of a new date, or on any bar when bar times are
// missing — is sold at that bar's close, so no position survives the night. Entries only on Mon–Fri MSK. Pure, stateless between bars and
// ticker-agnostic. Run with `-strategy gap_fade -interval Minutes30`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback is two full trading days of 30-minute bars with evening sessions: always enough
// to see the previous day's last bar behind a small EntryBar.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	GapATR         float64 // minimum gap depth in daily ATR (grid; default 0.5)
	MaxGapATR      float64 // deeper gaps are events, not noise — skipped; 0 = no cap (default 3.0)
	EntryBar       int     // 0-based bar of the MSK day whose close is the entry (grid; default 0)
	TargetFill     float64 // share of the gap left at entry that the target closes (grid; default 1.0)
	StopDailyATR   float64 // stop distance below entry in daily ATR (grid; default 0.5)
	DailyATRPeriod int     // Wilder period over completed weekday dailies (default 14)
	EODHour        int     // the first bar starting at/after EODHour:EODMinute MSK forces the exit
	EODMinute      int     // (default 18:30 — the last main-session bar)
}

// DefaultParams returns the probe's configuration.
func DefaultParams() Params {
	return Params{
		GapATR:         0.5,
		MaxGapATR:      3.0,
		EntryBar:       0,
		TargetFill:     1.0,
		StopDailyATR:   0.5,
		DailyATRPeriod: 14,
		EODHour:        18,
		EODMinute:      30,
	}
}

// Strategy trades a single instrument with the gap fade rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback must reach the bar before the day's first bar behind bar number EntryBar.
func (s *Strategy) Lookback() int { return max(minLookback, s.p.EntryBar+2) }

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

// dayKey is the MSK calendar date of t as yyyymmdd: cheap to compare and allocation-free, which
// matters because it runs on every bar of every grid combination.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
}

// weekdayDaily drops weekend (Sat/Sun MSK) bars from the daily series, keeping the three price
// slices aligned. Weekend sessions are narrow and thin and would understate the daily ATR. When
// times is empty or misaligned the series is returned untouched rather than guessed at.
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

// dailyATR is Wilder's ATR over COMPLETED weekday dailies (the engine exposes only days closed
// before the current bar). Returns 0 when the data cannot support it; the caller then refuses
// the entry, because without it there is no stop.
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

// beforeEOD reports whether bar-time t starts strictly before EODHour:EODMinute MSK.
func (s *Strategy) beforeEOD(t time.Time) bool {
	tl := t.In(mskLoc)
	return tl.Hour()*60+tl.Minute() < s.p.EODHour*60+s.p.EODMinute
}

// entryBarStart reports whether bar i is bar number k (0-based) of its MSK day and returns the
// index of the day's first bar. Times are oldest-first, so checking the two ends is enough: bar
// i-k shares i's date and the bar before it does not. A day whose start is not inside the window
// (no bar before it) is rejected — prevClose would be a guess.
func entryBarStart(times []time.Time, i, k int) (int, bool) {
	first := i - k
	if k < 0 || first < 1 {
		return 0, false
	}
	day := dayKey(times[i])
	if dayKey(times[first]) != day || dayKey(times[first-1]) == day {
		return 0, false
	}
	return first, true
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys bar number EntryBar of a weekday when the day opened at least GapATR daily ATRs
// below the previous close and the bar still closes below it. The cheap bar/price gates run
// before the daily ATR, so the ATR is computed about once a day instead of on every bar.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n {
		return sig
	}
	if s.p.StopDailyATR <= 0 || s.p.TargetFill <= 0 {
		return sig
	}
	i := n - 1
	now := md.Times[i]
	if isWeekend(now.In(mskLoc)) || !s.beforeEOD(now) {
		return sig
	}
	first, ok := entryBarStart(md.Times, i, s.p.EntryBar)
	if !ok {
		return sig
	}
	prevClose, openDay, closeP := md.Closes[first-1], md.Opens[first], md.Closes[i]
	if openDay >= prevClose || closeP >= prevClose {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	gap := (openDay - prevClose) / atr
	if gap > -s.p.GapATR {
		return sig
	}
	if s.p.MaxGapATR > 0 && gap < -s.p.MaxGapATR {
		return sig
	}
	stop := closeP - s.p.StopDailyATR*atr
	if stop <= 0 {
		return sig
	}
	target := closeP + s.p.TargetFill*(prevClose-closeP)
	sig.Kind = model.SignalBuy
	sig.TakeProfit, sig.StopLoss, sig.ATR = target, stop, atr
	sig.EntryReason = fmt.Sprintf("GAP: open %.4f ниже close %.4f на %.2f ATR (дневной ATR %.4f), сигнал по close %.4f бара %d дня, цель %.4f, стоп %.4f",
		openDay, prevClose, -gap, atr, closeP, s.p.EntryBar, target, stop)
	return sig
}

// manage exits in priority order: the frozen stop, the frozen target, then the end of day.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low, high, closeP := md.Lows[i], md.Highs[i], md.Closes[i]
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}
	if pos.TakeProfit > 0 && high >= pos.TakeProfit {
		sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цель %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
		return sig
	}
	if why := s.eodReason(md, pos); why != "" {
		sig.Kind, sig.Reason = model.SignalSell, "EOD"
		sig.ExitReason = fmt.Sprintf("EOD: %s, выход по %.4f (вход %.4f)", why, closeP, pos.PurchasePrice)
	}
	return sig
}

// eodReason explains why the position must close on the current bar, or returns "" when it may
// stay. The overnight guard fails closed: without aligned bar times the position is closed, and
// the first bar of a new MSK date always closes it — a gap_fade position is never legitimately
// open there, whatever EntryTime says (it may be unset). A bar of a later date than a known
// entry closes it too; together they cover days whose data has no bar at the EOD time.
func (s *Strategy) eodReason(md strategy.MarketData, pos *strategy.Position) string {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n {
		return "нет времени баров — выход по безопасности"
	}
	now := md.Times[n-1]
	if dayKey(now) != dayKey(md.Times[n-2]) {
		return "первый бар новой даты — позиция не переносится через ночь"
	}
	if !pos.EntryTime.IsZero() && dayKey(now) != dayKey(pos.EntryTime) {
		return "бар следующего дня — позиция не переносится через ночь"
	}
	if !s.beforeEOD(now) {
		return fmt.Sprintf("бар %s не раньше %02d:%02d MSK", now.In(mskLoc).Format("15:04"), s.p.EODHour, s.p.EODMinute)
	}
	return ""
}
