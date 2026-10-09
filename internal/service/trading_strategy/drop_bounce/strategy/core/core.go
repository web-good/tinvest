// Package core implements drop_bounce: a long-only 15-minute bounce after a large red bar that
// closes oversold. A bar falling at least MinDropPct percent from its open to its close, with
// RSI(RSIPeriod) at its close below RSIEntry, signals a buy; the backtest fills it at the next
// bar's open (Config.EntryAtNextOpen). Signals are taken on Mon–Fri MSK bars starting at or after
// StartHour whose fill bar lies on the same date. Exits, in order: a stop StopPct below the fill
// price, a ProfitPct profit at the fill bar's close, the first red MACD histogram bar after a
// green one closed during the trade, RSI crossing RSIExit upward, the smoothed stochastic %K
// crossing StochExit upward. There is no end-of-day exit: positions carry overnight. Pure,
// stateless between bars and ticker-agnostic. Run with `-strategy drop_bounce -interval Minutes15`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// BarStep is the only timeframe drop_bounce runs on. It matters only for the entry window: the
// fill bar starts one step after the signal bar and must lie on the same MSK date.
const BarStep = 15 * time.Minute

// minLookback keeps 16 hours of 15m bars so the indicators converge and the MACD scan sees the
// holding bars of a typical trade.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them. A zero threshold disables its exit.
type Params struct {
	MinDropPct float64 // minimum (open−close)/open of the signal bar, percent (default 0.6)
	RSIPeriod  int     // RSI length for the entry and the RSI exit (default 6)
	RSIEntry   float64 // the signal bar's RSI must be below this (default 30)
	RSIExit    float64 // sell when RSI crosses this upward; 0 = off (default 70)
	// MACDExit 1 sells on the first red histogram bar (MACD − signal < 0) after a green one (> 0)
	// that closed while holding; bars before the entry do not count; 0 = off (default 1).
	MACDExit    int
	MACDFast    int     // default 9
	MACDSlow    int     // default 14
	MACDSignal  int     // default 3
	StochPeriod int     // stochastic %K period (default 14)
	StochSmooth int     // SMA smoothing of %K; 1 = raw %K (default 3)
	StochExit   float64 // sell when smoothed %K crosses this upward; 0 = off (default 80)
	ProfitPct   float64 // sell when the fill bar closes this percent above the fill; 0 = off (default 1)
	StopPct     float64 // stop this percent below the fill price; 0 = off (default 1)
	StartHour   int     // signal bars must start at/after StartHour:00 MSK (default 8)
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		MinDropPct:  0.6,
		RSIPeriod:   6,
		RSIEntry:    30,
		RSIExit:     70,
		MACDExit:    1,
		MACDFast:    9,
		MACDSlow:    14,
		MACDSignal:  3,
		StochPeriod: 14,
		StochSmooth: 3,
		StochExit:   80,
		ProfitPct:   1,
		StopPct:     1,
		StartHour:   8,
	}
}

// Strategy trades a single instrument with the drop_bounce rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback is enough history for every indicator to converge (Wilder RSI and the EMA-based MACD
// need several periods) plus the previous bar each cross compares against.
func (s *Strategy) Lookback() int {
	return max(minLookback, 4*s.p.RSIPeriod+2, 3*s.p.MACDSlow+s.p.MACDSignal+2, s.p.StochPeriod+s.p.StochSmooth+2)
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
	return dayKey(tl.Add(BarStep)) == dayKey(tl)
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys after a red bar falling at least MinDropPct percent whose close leaves RSI below
// RSIEntry. The signal carries no levels: the stop and the profit are measured from the fill.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n {
		return sig
	}
	// RSISeries is 0 up to index RSIPeriod−1; n > RSIPeriod keeps the warm-up out of the check.
	if p.MinDropPct <= 0 || p.RSIPeriod <= 0 || p.RSIEntry <= 0 || n <= p.RSIPeriod {
		return sig
	}
	i := n - 1
	if !s.inWindow(md.Times[i]) {
		return sig
	}
	o, c := md.Opens[i], md.Closes[i]
	if o <= 0 || c >= o {
		return sig
	}
	dropPct := (o - c) / o * 100
	// The tiny epsilon keeps a fall of exactly MinDropPct from failing on float rounding.
	if dropPct < p.MinDropPct-1e-9 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)[i]
	if rsi >= p.RSIEntry {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.EntryReason = fmt.Sprintf("DROP: open %.4f → close %.4f, падение %.2f%%, RSI(%d) %.1f < %.0f",
		o, c, dropPct, p.RSIPeriod, rsi, p.RSIEntry)
	return sig
}

// manage exits in priority order: the stop below the fill price (intrabar, filled by the engine at
// min(stop, open)), then at the bar's close the fill-bar profit, the MACD green→red flip, the RSI
// cross and the stochastic cross. There is no end-of-day exit.
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
	if s.p.StopPct > 0 {
		stop := entry * (1 - s.p.StopPct/100)
		// The tiny epsilon keeps a low of exactly the stop from slipping on float rounding.
		if md.Lows[i] <= stop+1e-9 {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f, −%.2f%%)", md.Lows[i], stop, entry, s.p.StopPct)
			return sig
		}
	}
	if s.p.ProfitPct > 0 && len(md.Times) == n && md.Times[i].Equal(pos.EntryTime) &&
		closeP >= entry*(1+s.p.ProfitPct/100)-1e-9 {
		return sell("PROFIT", fmt.Sprintf("бар входа закрылся на +%.2f%%", (closeP/entry-1)*100))
	}
	if s.p.MACDExit != 0 {
		if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); m != nil {
			h := make([]float64, n)
			for j := range h {
				h[j] = m[j] - sg[j]
			}
			if greenThenRed(h, md.Times, pos.EntryTime) {
				return sell("MACD", fmt.Sprintf("гистограмма %.4f — первый красный бар после зелёного", h[i]))
			}
		}
	}
	if s.p.RSIExit > 0 && s.p.RSIPeriod > 0 && n >= s.p.RSIPeriod+2 {
		r := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
		if crossUp(r[i-1], r[i], s.p.RSIExit) {
			return sell("RSI", fmt.Sprintf("RSI(%d) %.1f → %.1f пересёк %.0f вверх", s.p.RSIPeriod, r[i-1], r[i], s.p.RSIExit))
		}
	}
	if s.p.StochExit > 0 {
		_, k := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochPeriod, s.p.StochSmooth)
		if m := len(k); m >= 2 && crossUp(k[m-2], k[m-1], s.p.StochExit) {
			return sell("STOCH", fmt.Sprintf("%%K %.1f → %.1f пересёк %.0f вверх", k[m-2], k[m-1], s.p.StochExit))
		}
	}
	return sig
}

// greenThenRed reports whether the histogram's last bar is red (< 0) and the latest non-zero bar
// before it, among bars at or after entry, is green (> 0). A still-open position has not yet
// seen a red bar after a green one — it would have exited — so this is exactly "the first red
// after a green during the trade". Bars before entry never count; with an unknown entry or
// misaligned times the whole window counts.
func greenThenRed(h []float64, times []time.Time, entry time.Time) bool {
	last := len(h) - 1
	if last < 1 || h[last] >= 0 {
		return false
	}
	checkEntry := !entry.IsZero() && len(times) == len(h)
	for j := last - 1; j >= 0; j-- {
		if checkEntry && times[j].Before(entry) {
			return false
		}
		if h[j] > 0 {
			return true
		}
		if h[j] < 0 {
			return false
		}
	}
	return false
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
