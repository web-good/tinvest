// Package core implements drop_bounce: a long-only bounce after a large red bar that
// closes oversold. A bar falling at least MinDropPct percent from its open to its close, with
// RSI(RSIPeriod) at its close below RSIEntry, signals a buy. With GreenWaitBars the buy waits for
// the first green bar at most GreenWaitBars bars after the signal on the same date (only non-green
// bars between them); without it the signal bar itself triggers. The backtest fills at the next
// bar's open after the triggering bar (Config.EntryAtNextOpen). Signals are taken on Mon–Fri MSK
// bars starting at or after StartHour whose fill bar lies on the same date; an optional day-range
// gate requires the day's high−low up to the triggering bar to stay within MaxDayRangeATR daily
// ATRs, and an optional volume gate requires the signal bar's volume to reach VolMult times the
// mean of the VolBars bars before it. The daily ATR (Wilder, DailyATRPeriod, completed weekday dailies) is taken at the
// signal and frozen for the trade. Exits, in order: a stop StopDailyATR daily ATRs below the fill
// price, a ProfitPct profit at the fill bar's close, RSI crossing RSIExit upward, the smoothed
// stochastic %K crossing StochExit upward; with ExitInProfit either cross at or below the entry is
// deferred to the first close above it. There is no end-of-day exit: positions carry
// overnight. Pure, stateless between bars and ticker-agnostic. The bar step (5m, 15m, 30m or 1h)
// is fixed at construction. Run with `-strategy drop_bounce -interval Minutes5|Minutes15|Minutes30|Hour1`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// deferredLookback keeps 200 bars while a deferred indicator exit can be armed, so the scan for the cross
// still sees it on a trade that has waited for profit for a while (about 17 hours of 5m).
const deferredLookback = 200

// minLookback keeps 64 bars (5h20m of 5m, 16 hours of 15m, about four sessions of 1h) so the
// indicators converge.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them. A zero threshold disables its exit.
type Params struct {
	MinDropPct float64 // minimum (open−close)/open of the signal bar, percent (default 0.6)
	RSIPeriod  int     // RSI length for the entry and the RSI exit (default 6)
	RSIEntry   float64 // the signal bar's RSI must be below this (default 30)
	RSIExit    float64 // sell when RSI crosses this upward; 0 = off (default 70)
	// ExitInProfit 1 defers an RSI or stochastic exit cross that closes at or below the entry: the
	// trade then sells at the first close above the entry; 0 sells at the cross whatever the price
	// (default 1).
	ExitInProfit int
	StochPeriod  int     // stochastic %K period (default 14)
	StochSmooth  int     // SMA smoothing of %K; 1 = raw %K (default 3)
	StochExit    float64 // sell when smoothed %K crosses this upward; 0 = off (default 80)
	ProfitPct    float64 // sell when the fill bar closes this percent above the fill; 0 = off (default 1)
	// StopDailyATR is the stop distance below the fill price in daily ATRs frozen at the signal;
	// 0 = off (default 0.5).
	StopDailyATR float64
	// DailyATRPeriod is the Wilder ATR length over completed weekday dailies (default 14).
	DailyATRPeriod int
	// MaxDayRangeATR caps the day's high−low up to and including the triggering bar (the green
	// one with GreenWaitBars), in daily ATRs:
	// a day that has already travelled further is refused; 0 = off (default 0).
	MaxDayRangeATR float64
	// VolMult requires the signal bar's volume to reach VolMult times the mean volume of the
	// VolBars bars before it; 0 = off (default 0).
	VolMult   float64
	VolBars   int // bars behind the volume mean (default 20)
	StartHour int // signal bars must start at/after StartHour:00 MSK (default 8)
	// GreenWaitBars waits for the first green bar after the signal: the entry fills at the open
	// after a green bar that comes at most GreenWaitBars bars after the signal on the same MSK
	// date, with only non-green bars between them; 0 = off, the fill follows the signal bar
	// (default 3).
	GreenWaitBars int
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		MinDropPct:     0.6,
		RSIPeriod:      6,
		RSIEntry:       30,
		RSIExit:        70,
		ExitInProfit:   1,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      80,
		ProfitPct:      1,
		StopDailyATR:   0.5,
		DailyATRPeriod: 14,
		VolBars:        20,
		StartHour:      8,
		GreenWaitBars:  3,
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

// Lookback is enough history for every indicator to converge (Wilder RSI needs several periods)
// plus the previous bar each cross compares against, and the bars behind the volume mean, all
// taken back from a signal up to GreenWaitBars bars before the current one.
func (s *Strategy) Lookback() int {
	w := max(s.p.GreenWaitBars, 0)
	n := max(minLookback, 4*s.p.RSIPeriod+2+w, s.p.StochPeriod+s.p.StochSmooth+2, s.p.VolBars+1+w)
	if s.p.ExitInProfit != 0 && (s.p.RSIExit > 0 || s.p.StochExit > 0) {
		n = max(n, deferredLookback)
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

// dayKey is the MSK calendar date of t as yyyymmdd.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
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
		if wd := times[i].In(mskLoc).Weekday(); wd == time.Saturday || wd == time.Sunday {
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

// volumeRatio returns the volume of the signal bar j over the mean volume of the VolBars bars
// before it, zero-volume bars left out of the mean. ok is false when the data cannot support the
// ratio — no or misaligned volumes, fewer than VolBars bars of history, nothing to average — and
// the caller then lets the signal through: missing volume must never block an entry.
func (s *Strategy) volumeRatio(md strategy.MarketData, j int) (ratio float64, ok bool) {
	n := len(md.Volumes)
	if s.p.VolBars <= 0 || n != len(md.Closes) || j >= n || j < s.p.VolBars {
		return 0, false
	}
	var sum float64
	cnt := 0
	for _, v := range md.Volumes[j-s.p.VolBars : j] {
		if v > 0 {
			sum += float64(v)
			cnt++
		}
	}
	if cnt == 0 {
		return 0, false
	}
	return float64(md.Volumes[j]) / (sum / float64(cnt)), true
}

// inWindow reports whether a signal on the bar starting at t may open a position: a weekday, at
// or after StartHour, and the fill bar (one step later) on the same MSK date.
func (s *Strategy) inWindow(t time.Time) bool {
	tl := t.In(mskLoc)
	if wd := tl.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	if tl.Hour() < s.p.StartHour {
		return false
	}
	return dayKey(tl.Add(s.step)) == dayKey(tl)
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// isDrop reports whether bar j is a signal: red, falling at least MinDropPct percent, with RSI
// below RSIEntry at its close. rsi is RSISeries over md.Closes.
func (s *Strategy) isDrop(md strategy.MarketData, rsi []float64, j int) bool {
	o, c := md.Opens[j], md.Closes[j]
	// RSISeries is 0 up to index RSIPeriod−1; j ≥ RSIPeriod keeps the warm-up out of the check.
	if j < s.p.RSIPeriod || o <= 0 || c >= o {
		return false
	}
	// The tiny epsilon keeps a fall of exactly MinDropPct from failing on float rounding.
	return (o-c)/o*100 >= s.p.MinDropPct-1e-9 && rsi[j] < s.p.RSIEntry
}

// greenSignal returns the signal bar the current bar confirms, or −1: the current bar must be
// green, and the signal must sit at most GreenWaitBars bars before it on the same MSK date, at or
// after StartHour, with only non-green bars between them — so the current bar is the first green.
func (s *Strategy) greenSignal(md strategy.MarketData, rsi []float64) int {
	i := len(md.Closes) - 1
	if md.Opens[i] <= 0 || md.Closes[i] <= md.Opens[i] {
		return -1
	}
	day := dayKey(md.Times[i])
	for j := i - 1; j >= 0 && i-j <= s.p.GreenWaitBars; j-- {
		if dayKey(md.Times[j]) != day || md.Times[j].In(mskLoc).Hour() < s.p.StartHour || md.Closes[j] > md.Opens[j] {
			return -1
		}
		if s.isDrop(md, rsi, j) {
			return j
		}
	}
	return -1
}

// enter buys after a red bar falling at least MinDropPct percent whose close leaves RSI below
// RSIEntry — at once, or with GreenWaitBars after the first green bar that follows it. The signal
// carries no levels: the stop and the profit are measured from the fill. It carries the daily ATR
// for the engine to freeze; when the stop or the day-range gate needs that ATR and it is unknown,
// there is no entry.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n {
		return sig
	}
	if p.MinDropPct <= 0 || p.RSIPeriod <= 0 || p.RSIEntry <= 0 || n <= p.RSIPeriod {
		return sig
	}
	i := n - 1
	if !s.inWindow(md.Times[i]) {
		return sig
	}
	rsis := indicators.RSISeries(md.Closes, p.RSIPeriod)
	j, green := i, ""
	if p.GreenWaitBars > 0 {
		if j = s.greenSignal(md, rsis); j < 0 {
			return sig
		}
		green = fmt.Sprintf(", первая зелёная через %d бар(а): open %.4f → close %.4f", i-j, md.Opens[i], md.Closes[i])
	} else if !s.isDrop(md, rsis, i) {
		return sig
	}
	o, c, rsi := md.Opens[j], md.Closes[j], rsis[j]
	dropPct := (o - c) / o * 100
	atr := s.dailyATR(md)
	if (p.StopDailyATR > 0 || p.MaxDayRangeATR > 0) && atr <= 0 {
		return sig
	}
	// An armed stop measured from the last close must land above zero, or the long is naked.
	if p.StopDailyATR > 0 && md.Closes[i]-p.StopDailyATR*atr <= 0 {
		return sig
	}
	dayRange := ""
	// The gate never blocks on a missing day extent.
	if p.MaxDayRangeATR > 0 && md.TodayHigh > 0 && md.TodayLow > 0 && md.TodayHigh >= md.TodayLow {
		used := (md.TodayHigh - md.TodayLow) / atr
		// The tiny epsilon keeps a range of exactly the cap from failing on float rounding.
		if used > p.MaxDayRangeATR+1e-9 {
			return sig
		}
		dayRange = fmt.Sprintf(", размах дня %.2f ATR ≤ %.2f", used, p.MaxDayRangeATR)
	}
	volume := ""
	if p.VolMult > 0 {
		if r, ok := s.volumeRatio(md, j); ok {
			// The tiny epsilon keeps a volume of exactly the cap from failing on float rounding.
			if r < p.VolMult-1e-9 {
				return sig
			}
			volume = fmt.Sprintf(", объём %d = %.2f× среднего за %d свечей", md.Volumes[j], r, p.VolBars)
		}
	}
	sig.Kind = model.SignalBuy
	sig.ATR = atr
	// The fill must be the very next slot: if that candle is missing from the data, the engine
	// drops the signal rather than buying at a later session's open.
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("DROP: open %.4f → close %.4f, падение %.2f%%, RSI(%d) %.1f < %.0f%s%s%s, дневной ATR %.4f",
		o, c, dropPct, p.RSIPeriod, rsi, p.RSIEntry, volume, green, dayRange, atr)
	return sig
}

// manage exits in priority order: the stop StopDailyATR daily ATRs below the fill price (the ATR
// frozen at entry, today's ATR when the position carries none; intrabar, filled by the engine at
// min(stop, open)), then at the bar's close the fill-bar profit, the RSI cross and the stochastic cross. There is no end-of-day exit.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n || pos.PurchasePrice <= 0 {
		return sig
	}
	i := n - 1
	entry, closeP := pos.PurchasePrice, md.Closes[i]
	sell := func(reason, why string) model.Signal {
		sig.Kind, sig.Reason = model.SignalSell, reason
		sig.ExitReason = fmt.Sprintf("%s: %s, выход по %.4f (вход %.4f)", reason, why, closeP, entry)
		return sig
	}
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
	if s.p.ProfitPct > 0 && len(md.Times) == n && md.Times[i].Equal(pos.EntryTime) &&
		closeP >= entry*(1+s.p.ProfitPct/100)-1e-9 {
		return sell("PROFIT", fmt.Sprintf("бар входа закрылся на +%.2f%%", (closeP/entry-1)*100))
	}
	// With ExitInProfit an indicator cross that closes at or below the entry does not sell: it
	// arms the exit, which then fires at the first close above the entry.
	deferred := s.p.ExitInProfit != 0
	var rsi, stoch []float64 // valid values only, aligned so the last one is the current bar
	if s.p.RSIExit > 0 && s.p.RSIPeriod > 0 && n >= s.p.RSIPeriod+2 {
		rsi = indicators.RSISeries(md.Closes, s.p.RSIPeriod)[s.p.RSIPeriod:]
		if m := len(rsi); crossUp(rsi[m-2], rsi[m-1], s.p.RSIExit) && (!deferred || closeP > entry) {
			return sell("RSI", fmt.Sprintf("RSI(%d) %.1f → %.1f пересёк %.0f вверх", s.p.RSIPeriod, rsi[m-2], rsi[m-1], s.p.RSIExit))
		}
	}
	if s.p.StochExit > 0 {
		_, stoch = indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochPeriod, s.p.StochSmooth)
		if m := len(stoch); m >= 2 && crossUp(stoch[m-2], stoch[m-1], s.p.StochExit) && (!deferred || closeP > entry) {
			return sell("STOCH", fmt.Sprintf("%%K %.1f → %.1f пересёк %.0f вверх", stoch[m-2], stoch[m-1], s.p.StochExit))
		}
	}
	if deferred && closeP > entry {
		when := func(j int) string {
			if len(md.Times) != n {
				return ""
			}
			return " " + md.Times[j].In(mskLoc).Format("02.01 15:04")
		}
		if j := crossInTrade(rsi, n, md.Times, pos.EntryTime, s.p.RSIExit); j >= 0 {
			return sell("RSI", fmt.Sprintf("RSI(%d) пересёк %.0f вверх на баре%s в минусе, выход в плюс",
				s.p.RSIPeriod, s.p.RSIExit, when(j)))
		}
		if j := crossInTrade(stoch, n, md.Times, pos.EntryTime, s.p.StochExit); j >= 0 {
			return sell("STOCH", fmt.Sprintf("%%K пересёк %.0f вверх на баре%s в минусе, выход в плюс", s.p.StochExit, when(j)))
		}
	}
	return sig
}

// crossInTrade returns the bar index of the latest cross of level upward before the current bar
// and at or after entry, or −1. v holds only valid indicator values and is aligned to the end of
// the window of n bars (its last value is the current bar), so value k sits on bar n−len(v)+k.
// A still-open position that saw a cross above its entry would have sold, so any cross found here
// closed at or below the entry — the deferred exit is armed. With an unknown entry or bar times
// not aligned to the n bars the whole series counts.
func crossInTrade(v []float64, n int, times []time.Time, entry time.Time, level float64) int {
	off := n - len(v)
	checkEntry := !entry.IsZero() && len(times) == n
	for k := len(v) - 2; k >= 1; k-- {
		if checkEntry && times[off+k].Before(entry) {
			return -1
		}
		if crossUp(v[k-1], v[k], level) {
			return off + k
		}
	}
	return -1
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
