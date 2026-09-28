package backtest

import (
	"fmt"
	"sort"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/domain/ema"
	zonecore "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Pinned zone-screener axes. Like the pullback screener, it answers "is this ticker
// worth calibrating", so it sweeps only the parameters whose useful range is
// instrument-specific. EMA 50 is on the axis because it was the lead lever on both
// DOMRF and AFKS. See docs/superpowers/specs/2026-09-28-rsi-zone-screener-design.md, §3.
var (
	zoneGridRSIPeriods = []int{4, 6}
	zoneGridRSILowers  = []float64{15, 25}
	zoneGridEMAPeriods = []int{50, 100, 200}
	zoneGridRSIUppers  = []float64{65, 75}
)

// ZoneGrid returns the 24 configurations every ticker is screened on, in a
// deterministic order. The stop is pinned at 1.0 daily ATR: it is a property of a
// tuned point, not of the instrument, and 1.0 was accepted on every calibrated ticker.
func ZoneGrid() []zonecore.Params {
	out := make([]zonecore.Params, 0,
		len(zoneGridRSIPeriods)*len(zoneGridRSILowers)*len(zoneGridEMAPeriods)*len(zoneGridRSIUppers))
	for _, period := range zoneGridRSIPeriods {
		for _, lower := range zoneGridRSILowers {
			for _, emaPeriod := range zoneGridEMAPeriods {
				for _, upper := range zoneGridRSIUppers {
					out = append(out, zonecore.Params{
						RSIPeriod:      period,
						RSILower:       lower,
						RSIUpper:       upper,
						EMAPeriod:      emaPeriod,
						DailyATRPeriod: 14,
						StopDailyATR:   1.0,
					})
				}
			}
		}
	}
	return out
}

// stressTrades returns copies of trades with the round trip made more expensive: one
// more commission on each side plus two price ticks (entry rounds up, exit rounds down).
// Recomputing from the recorded trades instead of rerunning the engine ignores the tiny
// sizing drift a poorer cash curve would cause; on a profit-factor ratio it is negligible.
// A non-positive tick (the API did not report one) stresses by commission only.
func stressTrades(trades []backtest.Trade, commission, tick float64) []backtest.Trade {
	if len(trades) == 0 {
		return nil
	}
	tick = max(tick, 0)
	out := make([]backtest.Trade, len(trades))
	for i, t := range trades {
		q := float64(t.Quantity)
		t.PnL -= commission*(t.EntryPrice+t.ExitPrice)*q + 2*tick*q
		out[i] = t
	}
	return out
}

// tickPct is the share of two price ticks in the current price, in percent. unknown is
// true when either input is non-positive: the gate lets such a row through and the
// report flags it rather than inventing a cost.
func tickPct(tick, lastClose float64) (pct float64, unknown bool) {
	if tick <= 0 || lastClose <= 0 {
		return 0, true
	}
	return 2 * tick / lastClose * 100, false
}

// HalfResult is the summed raw PnL of the trades entered in one calendar half-year.
type HalfResult struct {
	Label  string // "2026H1" (January–June) or "2026H2" (July–December), Moscow time
	PnL    float64
	Trades int
}

// halfLabel names the calendar half-year of t in Moscow time.
func halfLabel(t time.Time) string {
	tl := t.In(screenMSK)
	half := 1
	if tl.Month() >= time.July {
		half = 2
	}
	return fmt.Sprintf("%dH%d", tl.Year(), half)
}

// halfYearResults groups trades by the half-year of their ENTRY and sorts by label
// (four-digit years make the lexical order chronological).
func halfYearResults(trades []backtest.Trade) []HalfResult {
	idx := make(map[string]int)
	var out []HalfResult
	for _, t := range trades {
		label := halfLabel(t.EntryTime)
		i, ok := idx[label]
		if !ok {
			i = len(out)
			idx[label] = i
			out = append(out, HalfResult{Label: label})
		}
		out[i].PnL += t.PnL
		out[i].Trades++
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// halvesStats returns the share of profitable halves among the given ones and the share
// of the best half in the sum of profitable halves (1.0 = all profit from one half).
// hasProfit is false when no half made money; topShare is then 0 and meaningless.
func halvesStats(halves []HalfResult) (posShare, topShare float64, hasProfit bool) {
	if len(halves) == 0 {
		return 0, 0, false
	}
	var pos int
	var sum, best float64
	for _, h := range halves {
		if h.PnL <= 0 {
			continue
		}
		pos++
		sum += h.PnL
		best = max(best, h.PnL)
	}
	posShare = float64(pos) / float64(len(halves))
	if sum <= 0 {
		return posShare, 0, false
	}
	return posShare, best / sum, true
}

// slShare is the fraction of trades closed by the protective stop (exit reason "SL").
func slShare(trades []backtest.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	var n int
	for _, t := range trades {
		if t.Reason == "SL" {
			n++
		}
	}
	return float64(n) / float64(len(trades))
}

// medianHoldDays is the median trade duration in calendar days. Measured by wall clock,
// not BarsHeld: evening sessions and weekends make the bar count per day uneven.
func medianHoldDays(trades []backtest.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	days := make([]float64, 0, len(trades))
	for _, t := range trades {
		days = append(days, t.ExitTime.Sub(t.EntryTime).Hours()/24)
	}
	return medianF(days)
}

// ZoneConfigResult is one grid configuration's trade list on one ticker.
type ZoneConfigResult struct {
	Params zonecore.Params
	Trades []backtest.Trade
}

// ZoneRow is one ticker's zone-screening result. Medians run across grid configurations.
type ZoneRow struct {
	Ticker      string
	Name        string
	TurnoverM   float64 // mean daily turnover, millions of RUB
	DailyATRPct float64 // mean weekday daily ATR, percent of close
	Bars        int     // 30-minute bars replayed
	TickPct     float64 // two price ticks as a percentage of the last close
	TickUnknown bool    // the API gave no tick (or no price): TickPct is 0 and unmeasured

	PFMed       float64 // median raw train profit factor
	PFMedStress float64 // median train profit factor under stressed costs: the ranking key
	TradesMed   float64
	Plateau     float64 // share of configurations with PF >= PlateauPF at >= PlateauTrades trades
	Capped      int
	SilentCfg   int

	PFMedHO       float64 // holdout, never a ranking key
	PFMedStressHO float64
	TradesMedHO   float64

	HalvesPos float64 // median share of profitable half-years (trading configurations only)
	TopHalf   float64 // median share of the best half in summed profitable halves; 0 = none profitable
	SLShare   float64 // median share of stop exits (trading configurations only)
	HoldDays  float64 // median of per-configuration median holding time, calendar days

	AboveEMA50  float64 // share of train bars with close > EMA(50)
	AboveEMA200 float64 // share of train bars with close > EMA(200)

	Best       zonecore.Params // configuration with the highest raw train PF (reference only)
	BestPF     float64
	BestHalves []HalfResult // half-year breakdown of Best's train trades
	NoSignals  bool         // no configuration traded in either window
}

// AggregateZone reduces one ticker's per-configuration results to a report row. tick is
// the instrument's price increment used by the cost stress (0 = unknown).
func AggregateZone(ticker, name string, results []ZoneConfigResult, split time.Time, tick float64, opts ScreenOpts) ZoneRow {
	row := ZoneRow{Ticker: ticker, Name: name, NoSignals: true}
	n := len(results)
	pfs, pfsStress, counts := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	pfsHO, pfsStressHO, countsHO := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	var halvesPos, topHalf, sl, hold []float64
	var plateau int
	var haveBest bool
	var bestTrain []backtest.Trade

	for _, r := range results {
		train, holdout := splitTrades(r.Trades, split)

		pf, cnt := profitFactor(train)
		if cnt > 0 {
			row.NoSignals = false
		} else {
			row.SilentCfg++
		}
		// The first configuration claims Best unconditionally, so Best is always a real
		// grid entry even when every raw PF is 0 (see Aggregate in pullback_screen.go).
		if !haveBest || pf > row.BestPF {
			row.BestPF, row.Best, bestTrain = pf, r.Params, train
			haveBest = true
		}
		pf, capped := clampPF(pf, opts.PFCap)
		if capped {
			row.Capped++
		}
		if pf >= opts.PlateauPF && cnt >= opts.PlateauTrades {
			plateau++
		}
		pfs = append(pfs, pf)
		counts = append(counts, float64(cnt))

		pfS, _ := profitFactor(stressTrades(train, opts.Commission, tick))
		pfS, _ = clampPF(pfS, opts.PFCap)
		pfsStress = append(pfsStress, pfS)

		pfHO, cntHO := profitFactor(holdout)
		if cntHO > 0 {
			row.NoSignals = false
		}
		pfHO, _ = clampPF(pfHO, opts.PFCap)
		pfsHO = append(pfsHO, pfHO)
		countsHO = append(countsHO, float64(cntHO))
		pfSHO, _ := profitFactor(stressTrades(holdout, opts.Commission, tick))
		pfSHO, _ = clampPF(pfSHO, opts.PFCap)
		pfsStressHO = append(pfsStressHO, pfSHO)

		if cnt == 0 {
			continue // a silent configuration has no halves, exits or holding time to judge
		}
		pos, top, hasProfit := halvesStats(halfYearResults(train))
		halvesPos = append(halvesPos, pos)
		if hasProfit {
			topHalf = append(topHalf, top)
		}
		sl = append(sl, slShare(train))
		hold = append(hold, medianHoldDays(train))
	}

	row.BestPF, _ = clampPF(row.BestPF, opts.PFCap)
	row.BestHalves = halfYearResults(bestTrain)
	row.PFMed = medianF(pfs)
	row.PFMedStress = medianF(pfsStress)
	row.TradesMed = medianF(counts)
	row.PFMedHO = medianF(pfsHO)
	row.PFMedStressHO = medianF(pfsStressHO)
	row.TradesMedHO = medianF(countsHO)
	row.HalvesPos = medianF(halvesPos)
	row.TopHalf = medianF(topHalf)
	row.SLShare = medianF(sl)
	row.HoldDays = medianF(hold)
	if n > 0 {
		row.Plateau = float64(plateau) / float64(n)
	}
	return row
}

// ZoneTickerInput is everything ScreenZoneTicker needs about one instrument.
type ZoneTickerInput struct {
	Ticker            string
	Name              string
	Bars              []backtest.Candle // 30-minute bars over the whole window (train + holdout)
	Daily             []backtest.Candle // daily bars with a warm-up lead-in
	Lot               int32
	MinPriceIncrement float64 // 0 when unknown
}

// ScreenZoneTicker replays every grid configuration over one ticker and reduces the runs
// to a report row. The strategy is built directly with zonecore.NewWithParams and NOT
// through RSIZoneLookupOrGeneric: registered tickers carry calibrated literals, and
// grading them on those would make their rows incomparable with the rest.
func ScreenZoneTicker(in ZoneTickerInput, cfgs []zonecore.Params, split time.Time, opts ScreenOpts) ZoneRow {
	cfg := backtest.Config{
		InitialCash: opts.Cash,
		Fraction:    opts.Fraction,
		Commission:  opts.Commission,
		Lot:         in.Lot,
	}
	results := make([]ZoneConfigResult, 0, len(cfgs))
	for _, p := range cfgs {
		// rsi_zone needs no higher-timeframe series: htfCandles is nil.
		res := backtest.Run(zonecore.NewWithParams(in.Ticker, p), in.Bars, in.Daily, nil, cfg)
		results = append(results, ZoneConfigResult{Params: p, Trades: res.Trades})
	}
	row := AggregateZone(in.Ticker, in.Name, results, split, in.MinPriceIncrement, opts)
	row.Bars = len(in.Bars)
	row.TurnoverM = backtest.MeanDailyTurnoverM(in.Bars, in.Lot)
	row.DailyATRPct = MeanDailyATRPct(in.Daily, screenDailyATRPeriod)

	var lastClose float64
	if len(in.Bars) > 0 {
		lastClose = in.Bars[len(in.Bars)-1].Close
	}
	row.TickPct, row.TickUnknown = tickPct(in.MinPriceIncrement, lastClose)

	var train []backtest.Candle
	for _, b := range in.Bars {
		if b.Time.Before(split) {
			train = append(train, b)
		}
	}
	row.AboveEMA50 = aboveEMAShare(train, 50)
	row.AboveEMA200 = aboveEMAShare(train, 200)
	return row
}

// aboveEMAShare is the fraction of bars whose close sits strictly above EMA(period),
// computed with the same ema.Compute the strategy core uses. Warm-up bars (EMA == 0)
// are left out of the denominator.
func aboveEMAShare(bars []backtest.Candle, period int) float64 {
	if period <= 0 || len(bars) == 0 {
		return 0
	}
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	line := ema.Compute(closes, period)
	var above, n int
	for i, c := range closes {
		if line[i] <= 0 {
			continue
		}
		n++
		if c > line[i] {
			above++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(above) / float64(n)
}
