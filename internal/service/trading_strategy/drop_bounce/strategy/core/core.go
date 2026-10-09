// Package core implements drop_bounce: a long-only intraday bounce after a sharp red 5-minute,
// 15-minute or hourly bar. When a bar falls at least MinDropPct percent from its open to its close, it signals a buy;
// the backtest fills it at the next bar's open (Config.EntryAtNextOpen). The target is TargetBody
// of the drop bar's body above its close, the stop StopBody of the body below it; both are frozen
// on the position. Signals are taken only on Mon–Fri MSK bars that start at or after StartHour
// and whose fill bar still starts before EODHour:EODMinute. Whatever is open at the first bar
// starting at or after EODHour:EODMinute — or on the first bar of a new date, or when bar times
// are missing — is sold at that bar's close, so no position survives the night. With TimeExit on,
// a position that reached neither level is also sold at the close of its TimeExitBars-th bar
// (the fill bar counts as the first). With DropSeries on, DropSeriesBars consecutive red bars of
// one date that fall MinDropPct..MaxDropPct together also signal. With ConfirmBar on, the entry
// waits for a green bar closing above the drop's close and fills at the open after it. TrendEMA and
// TrendDailyEMA optionally require the price above an EMA before the drop; SkipFromHour..SkipToHour
// optionally pauses signals for part of the day; PumpBars optionally skips a drop that follows a
// strong green bar, which it merely retraces. Pure, stateless
// between bars and ticker-agnostic. Run with `-strategy drop_bounce -interval Minutes5`,
// `-interval Minutes15` or `-interval Hour1`; a ticker keeps a separate Params literal per timeframe.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// DefaultBarStep is the timeframe NewWithParams assumes. The bar step matters only for the entry
// window: a signal bar's fill bar starts one step later, and that start must be before the EOD
// time. Every other rule counts bars.
const DefaultBarStep = 5 * time.Minute

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	MinDropPct float64 // minimum (open−close)/open of the signal bar, percent (default 1.0)
	MaxDropPct float64 // deeper bars are skipped; 0 = no cap (default 0)
	TargetBody float64 // target above the signal close as a share of the bar body (default 0.5)
	// TargetPct > 0 sets the target TargetPct percent above the signal close instead of
	// TargetBody of the body (default 0 = body target).
	TargetPct float64
	// MinTargetPct > 0 skips a signal whose target lies less than MinTargetPct percent above the
	// signal close: a short target cannot pay the round-trip commission (default 0 = off).
	MinTargetPct float64
	StopBody     float64 // stop below the signal close as a share of the bar body (default 0.5)
	StartHour    int     // signal bars must start at/after StartHour:00 MSK (default 8)
	EODHour      int     // the first bar starting at/after EODHour:EODMinute MSK forces the exit
	EODMinute    int     // (default 17:00)
	// TimeExit 1 closes a position that reached neither stop nor target at the close of its
	// TimeExitBars-th bar, counting the entry (fill) bar as the first; 0 = off (default 0, 3).
	TimeExit     int
	TimeExitBars int
	// DropSeries 1 also accepts DropSeriesBars consecutive red bars of one MSK date whose total
	// fall (first open → last close) lies within MinDropPct..MaxDropPct; the series body is
	// first open − last close. A single bar still signals on its own; 0 = off (default 0, 3).
	DropSeries     int
	DropSeriesBars int
	// ConfirmBar 1 waits for the bar after the drop (or the series): it must be green and close
	// above the drop's close, on the same MSK date, and the entry is the open after it. Levels stay
	// those of the drop; MinTargetPct is measured from the confirmation close (default 0 = off).
	ConfirmBar int
	// TrendEMA > 0 takes a signal only when the close of the bar before the drop (the first bar
	// of a series) is above EMA(TrendEMA) of the run's timeframe on that bar: the drop must hit
	// an uptrend. Too little history fails the signal (default 0 = off).
	TrendEMA int
	// TrendDailyEMA > 0 takes a signal only when the last completed daily close is above the daily
	// EMA(TrendDailyEMA). Too little daily history fails the signal (default 0 = off).
	TrendDailyEMA int
	// SkipFromHour < SkipToHour skips a drop bar starting in [SkipFromHour:00, SkipToHour:00) MSK,
	// a midday pause inside the signal hours; equal or reversed hours are off (default 0, 0).
	SkipFromHour int
	SkipToHour   int
	// PumpBars > 0 looks at the PumpBars bars before the drop (the first bar of a series) and
	// skips the signal when one of them is a strong green bar: the drop then only retraces a
	// pump instead of overshooting. A green bar is strong when its body (close−open)/open is at
	// least PumpPct percent, or at least PumpRatio of the drop's body; a zero threshold is off.
	// Bars missing from the start of the history are not checked (default 0 = off).
	PumpBars  int
	PumpPct   float64
	PumpRatio float64
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		MinDropPct:     1.0,
		MaxDropPct:     0,
		TargetBody:     0.5,
		StopBody:       0.5,
		StartHour:      8,
		EODHour:        17,
		EODMinute:      0,
		TimeExit:       0,
		TimeExitBars:   3,
		DropSeries:     0,
		DropSeriesBars: 3,
	}
}

// Strategy trades a single instrument with the drop_bounce rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
	step   time.Duration
}

// NewWithParams returns the strategy for a ticker with explicit params on DefaultBarStep bars.
func NewWithParams(ticker string, p Params) *Strategy {
	return NewWithStep(ticker, DefaultBarStep, p)
}

// NewWithStep returns the strategy for bars of the given duration; a non-positive step falls back
// to DefaultBarStep.
func NewWithStep(ticker string, step time.Duration, p Params) *Strategy {
	if step <= 0 {
		step = DefaultBarStep
	}
	return &Strategy{ticker: ticker, p: p, step: step}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback is two bars — the signal bar and the one before it, which the overnight guard needs —
// widened to count TimeExitBars bars since the entry plus one, and to hold a whole drop series
// plus the confirmation bar. With TrendEMA on it also holds twice the period before the drop:
// ema.Compute seeds on an SMA of the first period closes, and the second span lets it converge.
func (s *Strategy) Lookback() int {
	l := 2
	if s.timeExitOn() {
		l = max(l, s.p.TimeExitBars+1)
	}
	drop := 1
	if s.seriesOn() {
		drop = s.p.DropSeriesBars
	}
	if s.confirmOn() {
		drop++
	}
	if s.p.TrendEMA > 0 {
		drop += 2*s.p.TrendEMA + 1
	}
	if s.pumpOn() {
		drop += s.p.PumpBars
	}
	return max(l, drop)
}

func (s *Strategy) timeExitOn() bool { return s.p.TimeExit != 0 && s.p.TimeExitBars > 0 }

func (s *Strategy) seriesOn() bool { return s.p.DropSeries != 0 && s.p.DropSeriesBars > 1 }

func (s *Strategy) confirmOn() bool { return s.p.ConfirmBar != 0 }

func (s *Strategy) pumpOn() bool {
	return s.p.PumpBars > 0 && (s.p.PumpPct > 0 || s.p.PumpRatio > 0)
}

// barsHeld counts the window's bars starting at or after the entry, the fill bar included. A
// position entered before the window start counts the whole window, which Lookback makes longer
// than TimeExitBars, so the exit still fires. 0 when the entry time is unknown.
func barsHeld(times []time.Time, entry time.Time) int {
	if entry.IsZero() {
		return 0
	}
	n := 0
	for i := len(times) - 1; i >= 0 && !times[i].Before(entry); i-- {
		n++
	}
	return n
}

// mskLoc anchors every calendar rule to Moscow (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func isWeekend(tl time.Time) bool {
	wd := tl.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// dayKey is the MSK calendar date of t as yyyymmdd.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
}

func minuteOfDay(tl time.Time) int { return tl.Hour()*60 + tl.Minute() }

func (s *Strategy) eodMinute() int { return s.p.EODHour*60 + s.p.EODMinute }

// inWindow reports whether a signal on the bar starting at t may open a position: a weekday, at
// or after StartHour, and the fill bar (one step later) still before the EOD time.
func (s *Strategy) inWindow(t time.Time) bool {
	tl := t.In(mskLoc)
	if isWeekend(tl) || minuteOfDay(tl) < s.p.StartHour*60 {
		return false
	}
	fill := tl.Add(s.step)
	return dayKey(fill) == dayKey(tl) && minuteOfDay(fill) < s.eodMinute()
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys after a bar whose body falls MinDropPct..MaxDropPct percent or, with DropSeries on,
// after DropSeriesBars red bars that fall that much together. The single bar is checked first.
// With ConfirmBar on, the drop must end one bar earlier and the current bar confirm it.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n {
		return sig
	}
	if s.p.MinDropPct <= 0 || (s.p.TargetBody <= 0 && s.p.TargetPct <= 0) || s.p.StopBody <= 0 {
		return sig
	}
	i := n - 1
	if !s.inWindow(md.Times[i]) {
		return sig
	}
	last, suffix := i, ""
	if s.confirmOn() {
		if !s.confirms(md, i) {
			return sig
		}
		last, suffix = i-1, "+CONF"
	}
	if s.inSkip(md.Times[last]) {
		return sig
	}
	label, first := "DROP", last
	if !s.dropInRange(md.Opens[last], md.Closes[last]) {
		k := s.p.DropSeriesBars
		if !s.seriesOn() || !redSeries(md, last, k) || !s.dropInRange(md.Opens[last-k+1], md.Closes[last]) {
			return sig
		}
		label, first = fmt.Sprintf("DROP%d", k), last-k+1
	}
	if !s.inTrend(md, first) {
		return sig
	}
	label += suffix
	openP, closeP := md.Opens[first], md.Closes[last]
	body := openP - closeP
	if s.pumpBefore(md, first, body) {
		return sig
	}
	dropPct := body / openP * 100
	stop := closeP - s.p.StopBody*body
	if stop <= 0 {
		return sig
	}
	target := closeP + s.p.TargetBody*body
	if s.p.TargetPct > 0 {
		target = closeP * (1 + s.p.TargetPct/100)
	}
	// The floor is measured from the decision bar's close — the confirmation close when it is on —
	// as the nearest price the order can get. The tiny epsilon keeps a target of exactly
	// MinTargetPct from failing on float rounding.
	if from := md.Closes[i]; s.p.MinTargetPct > 0 && (target-from)/from*100 < s.p.MinTargetPct-1e-9 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.TakeProfit, sig.StopLoss = target, stop
	sig.EntryReason = fmt.Sprintf("%s: open %.4f → close %.4f, падение %.2f%%, тело %.4f, цель %.4f, стоп %.4f",
		label, openP, closeP, dropPct, body, target, stop)
	if last != i {
		sig.EntryReason += fmt.Sprintf(", подтверждение %.4f → %.4f", md.Opens[i], md.Closes[i])
	}
	return sig
}

// confirms reports whether bar i confirms a drop ending at bar i−1: green, closing above the
// drop's close, on the drop's MSK date, with the drop bar itself inside the signal hours.
func (s *Strategy) confirms(md strategy.MarketData, i int) bool {
	if i < 1 || md.Closes[i] <= md.Opens[i] || md.Closes[i] <= md.Closes[i-1] {
		return false
	}
	prev := md.Times[i-1]
	return dayKey(prev) == dayKey(md.Times[i]) && minuteOfDay(prev.In(mskLoc)) >= s.p.StartHour*60
}

// inSkip reports whether a drop bar starting at t falls in the optional SkipFromHour..SkipToHour
// pause.
func (s *Strategy) inSkip(t time.Time) bool {
	if s.p.SkipFromHour >= s.p.SkipToHour {
		return false
	}
	h := t.In(mskLoc).Hour()
	return h >= s.p.SkipFromHour && h < s.p.SkipToHour
}

// inTrend applies the optional trend filters to a drop starting at bar first. Each is checked
// only on a signal, so the EMAs cost nothing on quiet bars.
func (s *Strategy) inTrend(md strategy.MarketData, first int) bool {
	if n := s.p.TrendEMA; n > 0 {
		j := first - 1
		if j < n-1 {
			return false
		}
		e := ema.Compute(md.Closes[:j+1], n)[j]
		if e <= 0 || md.Closes[j] <= e {
			return false
		}
	}
	if n := s.p.TrendDailyEMA; n > 0 {
		d := md.DailyCloses
		if len(d) < n {
			return false
		}
		e := ema.Compute(d, n)[len(d)-1]
		if e <= 0 || d[len(d)-1] <= e {
			return false
		}
	}
	return true
}

// pumpBefore reports whether one of the PumpBars bars before bar first is a strong green bar:
// body at least PumpPct percent of its open, or at least PumpRatio of the drop body.
func (s *Strategy) pumpBefore(md strategy.MarketData, first int, dropBody float64) bool {
	if !s.pumpOn() {
		return false
	}
	for j := max(0, first-s.p.PumpBars); j < first; j++ {
		o, c := md.Opens[j], md.Closes[j]
		if o <= 0 || c <= o {
			continue
		}
		up := c - o
		// The tiny epsilons keep a body of exactly the threshold from slipping on float rounding.
		if s.p.PumpPct > 0 && up/o*100 >= s.p.PumpPct-1e-9 {
			return true
		}
		if s.p.PumpRatio > 0 && up >= s.p.PumpRatio*dropBody-1e-9 {
			return true
		}
	}
	return false
}

// dropInRange reports whether a fall from openP to closeP is within MinDropPct..MaxDropPct.
func (s *Strategy) dropInRange(openP, closeP float64) bool {
	if openP <= 0 || closeP >= openP {
		return false
	}
	dropPct := (openP - closeP) / openP * 100
	// The tiny epsilon keeps a fall of exactly MinDropPct from failing on float rounding.
	if dropPct < s.p.MinDropPct-1e-9 {
		return false
	}
	return s.p.MaxDropPct <= 0 || dropPct <= s.p.MaxDropPct
}

// redSeries reports whether the k bars ending at i are all red (close < open) and share i's MSK
// date.
func redSeries(md strategy.MarketData, i, k int) bool {
	if k < 2 || i-k+1 < 0 {
		return false
	}
	day := dayKey(md.Times[i])
	for j := i - k + 1; j <= i; j++ {
		if md.Closes[j] >= md.Opens[j] || dayKey(md.Times[j]) != day {
			return false
		}
	}
	return true
}

// manage exits in priority order: the frozen stop, the frozen target, the end of day, then the
// optional time exit. EOD and TIME both sell at the bar's close.
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
		return sig
	}
	if s.timeExitOn() && len(md.Times) == n {
		if held := barsHeld(md.Times, pos.EntryTime); held >= s.p.TimeExitBars {
			sig.Kind, sig.Reason = model.SignalSell, "TIME"
			sig.ExitReason = fmt.Sprintf("TIME: %d бар(а) без стопа и цели, выход по %.4f (вход %.4f)", held, closeP, pos.PurchasePrice)
		}
	}
	return sig
}

// eodReason explains why the position must close on the current bar, or returns "" when it may
// stay. The overnight guard fails closed: without aligned bar times the position is closed, and
// the first bar of a new MSK date always closes it.
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
	if minuteOfDay(now.In(mskLoc)) >= s.eodMinute() {
		return fmt.Sprintf("бар %s не раньше %02d:%02d MSK", now.In(mskLoc).Format("15:04"), s.p.EODHour, s.p.EODMinute)
	}
	return ""
}
