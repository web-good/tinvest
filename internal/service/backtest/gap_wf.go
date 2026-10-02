package backtest

import (
	"fmt"
	"math"
	"sort"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
)

// Gate thresholds of the gap_fade pooled walk-forward (spec section 5).
const (
	gapGatePooledPF    = 1.5
	gapGateLast12PF    = 1.2
	gapGateTickerShare = 0.6
	gapGateStressedPF  = 1.0
)

// gapLoc anchors gap_fade calendar rules (ex-days, quarters, next-day exits) to Moscow.
var gapLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// GapTrade is one engine trade tagged with its ticker, so a pooled sample still breaks down
// per ticker.
type GapTrade struct {
	Ticker string
	backtest.Trade
}

// GapComboRun holds every trade one grid combination produced across the universe at one
// commission. Runs are index-aligned with the expanded grid.
type GapComboRun struct {
	Params any
	Trades []GapTrade
}

// GapFold is one fold of the pooled walk-forward. Combo is the index of the chosen run, or -1
// when no combination reached min-trades on the training window.
type GapFold struct {
	TrainFrom, TrainTo, TestFrom, TestTo time.Time
	Combo                                int
	TrainPF                              float64
	TrainTrades                          int
	OOS                                  []GapTrade
}

// GapGroup is one row of a breakdown: trades, PF and the sum of per-trade returns (fractions).
type GapGroup struct {
	Key    string
	Trades int
	PF     float64
	SumPct float64
}

// GapComboRank is one combination's full-window score.
type GapComboRank struct {
	Combo                            int
	PF                               float64
	Trades                           int
	ProfitableTickers, TradedTickers int
}

// GapCheck is one gate condition.
type GapCheck struct {
	Name      string
	Value     float64
	Threshold float64
	Pass      bool
}

// GapVerdict is the gate outcome: every check must pass.
type GapVerdict struct {
	Checks []GapCheck
	Pass   bool
}

// ExpandGapGrid parses a one-phase grid and expands it over core.DefaultParams. The pooled
// walk-forward picks one combination per fold from a single cartesian grid, so staged
// (multi-phase) grids are rejected rather than silently flattened.
func ExpandGapGrid(raw []byte) ([]any, error) {
	phases, err := ParsePhases(raw)
	if err != nil {
		return nil, err
	}
	if len(phases) != 1 {
		return nil, fmt.Errorf("backtest: gap_fade grid must have exactly one phase, got %d", len(phases))
	}
	return expandGrid(core.DefaultParams(), phases[0].Grid)
}

// TagGapTrades attaches the ticker to engine trades.
func TagGapTrades(ticker string, trades []backtest.Trade) []GapTrade {
	out := make([]GapTrade, len(trades))
	for i, t := range trades {
		out[i] = GapTrade{Ticker: ticker, Trade: t}
	}
	return out
}

func gapDay(t time.Time) string { return t.In(gapLoc).Format("2006-01-02") }

// DividendExDays maps each dividend's ex-day — the first weekday strictly after its last buy
// date, taken as an MSK calendar date — keyed "2006-01-02". Zero dates are skipped.
func DividendExDays(lastBuy []time.Time) map[string]bool {
	out := make(map[string]bool, len(lastBuy))
	for _, lb := range lastBuy {
		if lb.IsZero() {
			continue
		}
		l := lb.In(gapLoc)
		d := time.Date(l.Year(), l.Month(), l.Day(), 12, 0, 0, 0, gapLoc).AddDate(0, 0, 1)
		for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			d = d.AddDate(0, 0, 1)
		}
		out[d.Format("2006-01-02")] = true
	}
	return out
}

// DropExDayTrades removes trades entered on an ex-day — the structural dividend gap does not
// close and would pose as a signal. It also returns the distinct ex-days that were hit, sorted.
func DropExDayTrades(trades []GapTrade, exDays map[string]bool) (kept []GapTrade, droppedDays []string) {
	hit := map[string]bool{}
	kept = make([]GapTrade, 0, len(trades))
	for _, t := range trades {
		if d := gapDay(t.EntryTime); exDays[d] {
			hit[d] = true
			continue
		}
		kept = append(kept, t)
	}
	for d := range hit {
		droppedDays = append(droppedDays, d)
	}
	sort.Strings(droppedDays)
	return kept, droppedDays
}

// GapPF is the profit factor over per-trade returns (Trade.PnLPct): every trade weighs the same
// whatever its ticker's price or the compounded account size. No losses: +Inf when there is any
// profit, 0 otherwise.
func GapPF(trades []GapTrade) float64 {
	var gp, gl float64
	for _, t := range trades {
		if t.PnLPct > 0 {
			gp += t.PnLPct
		} else {
			gl -= t.PnLPct
		}
	}
	if gl == 0 {
		if gp > 0 {
			return math.Inf(1)
		}
		return 0
	}
	return gp / gl
}

// gapTradesIn keeps trades entered in [from, to).
func gapTradesIn(trades []GapTrade, from, to time.Time) []GapTrade {
	var out []GapTrade
	for _, t := range trades {
		if !t.EntryTime.Before(from) && t.EntryTime.Before(to) {
			out = append(out, t)
		}
	}
	return out
}

// GapTradesFrom keeps trades entered at or after from.
func GapTradesFrom(trades []GapTrade, from time.Time) []GapTrade {
	var out []GapTrade
	for _, t := range trades {
		if !t.EntryTime.Before(from) {
			out = append(out, t)
		}
	}
	return out
}

// RunGapWalkForward picks, per fold, the run with the best pooled training PF among runs with at
// least minTrades training trades (ties: more trades, then grid order) and collects its trades
// in the test window. Slicing full-window runs by entry date is exact for an intraday strategy:
// no trade carries state across days.
func RunGapWalkForward(runs []GapComboRun, from, to time.Time, trainMonths, testMonths, minTrades int) ([]GapFold, error) {
	windows, err := walkForwardFolds(from, to, trainMonths, testMonths)
	if err != nil {
		return nil, err
	}
	folds := make([]GapFold, 0, len(windows))
	for _, w := range windows {
		f := GapFold{TrainFrom: w.trainFrom, TrainTo: w.trainTo, TestFrom: w.testFrom, TestTo: w.testTo, Combo: -1}
		for k, r := range runs {
			train := gapTradesIn(r.Trades, w.trainFrom, w.trainTo)
			if len(train) < minTrades {
				continue
			}
			pf := GapPF(train)
			if f.Combo < 0 || pf > f.TrainPF || (pf == f.TrainPF && len(train) > f.TrainTrades) {
				f.Combo, f.TrainPF, f.TrainTrades = k, pf, len(train)
			}
		}
		if f.Combo >= 0 {
			f.OOS = gapTradesIn(runs[f.Combo].Trades, w.testFrom, w.testTo)
		}
		folds = append(folds, f)
	}
	return folds, nil
}

// PooledGapOOS concatenates the folds' out-of-sample trades.
func PooledGapOOS(folds []GapFold) []GapTrade {
	var out []GapTrade
	for _, f := range folds {
		out = append(out, f.OOS...)
	}
	return out
}

// SelectedGapCombos is the set of run indices some fold chose.
func SelectedGapCombos(folds []GapFold) map[int]bool {
	out := map[int]bool{}
	for _, f := range folds {
		if f.Combo >= 0 {
			out[f.Combo] = true
		}
	}
	return out
}

// ReplayGapFolds re-reads every fold's test window from alt — the same grid run at another
// commission — with the combination the fold already chose: it measures how fragile the chosen
// parameters are, it does not re-fit.
func ReplayGapFolds(folds []GapFold, alt []GapComboRun) []GapTrade {
	var out []GapTrade
	for _, f := range folds {
		if f.Combo < 0 || f.Combo >= len(alt) {
			continue
		}
		out = append(out, gapTradesIn(alt[f.Combo].Trades, f.TestFrom, f.TestTo)...)
	}
	return out
}

// gapGroupBy breaks trades down by key, rows sorted by key.
func gapGroupBy(trades []GapTrade, key func(GapTrade) string) []GapGroup {
	byKey := map[string][]GapTrade{}
	for _, t := range trades {
		k := key(t)
		byKey[k] = append(byKey[k], t)
	}
	out := make([]GapGroup, 0, len(byKey))
	for k, ts := range byKey {
		g := GapGroup{Key: k, Trades: len(ts), PF: GapPF(ts)}
		for _, t := range ts {
			g.SumPct += t.PnLPct
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func GapByTicker(trades []GapTrade) []GapGroup {
	return gapGroupBy(trades, func(t GapTrade) string { return t.Ticker })
}

func GapByYear(trades []GapTrade) []GapGroup {
	return gapGroupBy(trades, func(t GapTrade) string { return t.EntryTime.In(gapLoc).Format("2006") })
}

func GapByQuarter(trades []GapTrade) []GapGroup {
	return gapGroupBy(trades, func(t GapTrade) string {
		e := t.EntryTime.In(gapLoc)
		return fmt.Sprintf("%dQ%d", e.Year(), (int(e.Month())-1)/3+1)
	})
}

func GapByReason(trades []GapTrade) []GapGroup {
	return gapGroupBy(trades, func(t GapTrade) string { return t.Reason })
}

// GapNextDayEOD counts EOD exits that happened on a later date than the entry — days whose data
// had no bar at the EOD time, closed by the overnight guard.
func GapNextDayEOD(trades []GapTrade) int {
	n := 0
	for _, t := range trades {
		if t.Reason == "EOD" && gapDay(t.ExitTime) != gapDay(t.EntryTime) {
			n++
		}
	}
	return n
}

// profitableTickers counts tickers whose summed returns are positive, out of tickers with trades.
func profitableTickers(trades []GapTrade) (profitable, traded int) {
	for _, g := range GapByTicker(trades) {
		traded++
		if g.SumPct > 0 {
			profitable++
		}
	}
	return profitable, traded
}

// RankGapCombos scores every run on the full window (runs with fewer than minTrades trades are
// left out), best PF first; ties: more trades, then grid order. top <= 0 keeps all.
func RankGapCombos(runs []GapComboRun, minTrades, top int) []GapComboRank {
	var out []GapComboRank
	for k, r := range runs {
		if len(r.Trades) < minTrades {
			continue
		}
		p, n := profitableTickers(r.Trades)
		out = append(out, GapComboRank{Combo: k, PF: GapPF(r.Trades), Trades: len(r.Trades), ProfitableTickers: p, TradedTickers: n})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PF != out[j].PF {
			return out[i].PF > out[j].PF
		}
		return out[i].Trades > out[j].Trades
	})
	if top > 0 && len(out) > top {
		out = out[:top]
	}
	return out
}

// GapGate applies the spec's four conditions: pooled OOS PF, pooled OOS PF since last12From,
// the share of profitable tickers among tickers with OOS trades, and the pooled OOS PF of the
// stressed-cost replay.
func GapGate(oos, stressed []GapTrade, last12From time.Time) GapVerdict {
	p, n := profitableTickers(oos)
	share := 0.0
	if n > 0 {
		share = float64(p) / float64(n)
	}
	checks := []GapCheck{
		{Name: "pooled OOS PF", Value: GapPF(oos), Threshold: gapGatePooledPF},
		{Name: "pooled OOS PF, последние 12 месяцев", Value: GapPF(GapTradesFrom(oos, last12From)), Threshold: gapGateLast12PF},
		{Name: "доля прибыльных тикеров в OOS", Value: share, Threshold: gapGateTickerShare},
		{Name: "pooled OOS PF при 0.2% на сторону", Value: GapPF(stressed), Threshold: gapGateStressedPF},
	}
	v := GapVerdict{Pass: true}
	for _, c := range checks {
		// Inclusive at the threshold; the epsilon keeps a PF that is the threshold up to
		// summation rounding from failing.
		c.Pass = c.Value >= c.Threshold-1e-9
		v.Pass = v.Pass && c.Pass
		v.Checks = append(v.Checks, c)
	}
	return v
}
