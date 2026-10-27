package backtest

import (
	"fmt"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	dropcore "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Pinned drop-screener axes. Like the zone screener, it answers "is this ticker worth
// calibrating", so it sweeps the calibration axes at their ends: the entry RSI length, the exit
// RSI length and the stop (rules of 2026-10-14: hourly signal on 15m bars, 15m MACD confirmation).
var (
	dropGridRSIPeriods     = []int{4, 6}
	dropGridRSIExitPeriods = []int{8, 12}
	dropGridStops          = []float64{0.5, 1.0}
)

// DropGrid returns the 8 configurations every ticker is screened on, in a deterministic order
// (RSIPeriod outermost, StopDailyATR innermost). The rest — the RSI zone 30/50, EMA(200), the exit
// level 70, MACD(12,26,9) and the confirmation window — are the core defaults. drop_bounce runs on
// Minutes15 only.
func DropGrid(interval enum.Interval) ([]dropcore.Params, error) {
	if interval != enum.Minutes15 {
		return nil, fmt.Errorf("drop screener runs on Minutes15 only, not %s", interval)
	}
	out := make([]dropcore.Params, 0, len(dropGridRSIPeriods)*len(dropGridRSIExitPeriods)*len(dropGridStops))
	for _, rsi := range dropGridRSIPeriods {
		for _, exit := range dropGridRSIExitPeriods {
			for _, stop := range dropGridStops {
				p := dropcore.DefaultParams()
				p.RSIPeriod, p.RSIExitPeriod, p.StopDailyATR = rsi, exit, stop
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// exitShare is the fraction of trades closed with the given exit reason code.
func exitShare(trades []backtest.Trade, reason string) float64 {
	if len(trades) == 0 {
		return 0
	}
	var n int
	for _, t := range trades {
		if t.Reason == reason {
			n++
		}
	}
	return float64(n) / float64(len(trades))
}

// DropConfigResult is one grid configuration's trade list on one ticker.
type DropConfigResult struct {
	Params dropcore.Params
	Trades []backtest.Trade
}

// DropRow is one ticker's drop-screening result. Medians run across grid configurations, on the
// train window unless the field says holdout.
type DropRow struct {
	Ticker      string
	Name        string
	TurnoverM   float64 // mean daily turnover, millions of RUB
	DailyATRPct float64 // mean weekday daily ATR, percent of close
	Bars        int     // bars replayed on the screened timeframe
	TickPct     float64 // two price ticks as a percentage of the last close
	TickUnknown bool    // the API gave no tick (or no price): TickPct is 0 and unmeasured

	PFMed       float64 // median raw train profit factor
	PFMedStress float64 // median train profit factor under stressed costs: the ranking key
	TradesMed   float64
	Plateau     float64 // share of configurations with PF >= PlateauPF at >= PlateauTrades trades
	Capped      int     // configurations whose raw train PF hit the cap
	SilentCfg   int     // configurations with no train trade

	PFMedHO       float64 // holdout, never a ranking key
	PFMedStressHO float64
	TradesMedHO   float64

	HalvesPos float64 // median share of profitable half-years (trading configurations only)
	TopHalf   float64 // median share of the best half in summed profitable halves; 0 = none profitable
	SLShare   float64 // median share of stop exits (trading configurations only)
	RSIShare  float64 // median share of RSI exits
	HoldDays  float64 // median of per-configuration median holding time, calendar days

	Best       dropcore.Params // configuration with the highest raw train PF (reference only)
	BestPF     float64
	BestHalves []HalfResult // half-year breakdown of Best's train trades
	NoSignals  bool         // no configuration traded in either window
	// Pregated: turnover failed on daily candles before the grid ran; PF/trade fields are unmeasured.
	Pregated bool
}

// AggregateDrop reduces one ticker's per-configuration results to a report row. tick is the
// instrument's price increment used by the cost stress (0 = unknown). It mirrors AggregateZone
// and adds the RSI exit share.
func AggregateDrop(ticker, name string, results []DropConfigResult, split time.Time, tick float64, opts ScreenOpts) DropRow {
	row := DropRow{Ticker: ticker, Name: name, NoSignals: true}
	n := len(results)
	pfs, pfsStress, counts := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	pfsHO, pfsStressHO, countsHO := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	var halvesPos, topHalf, sl, rsi, hold []float64
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
		// The first configuration claims Best unconditionally, so Best is always a real grid
		// entry even when every raw PF is 0.
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
		sl = append(sl, exitShare(train, "SL"))
		rsi = append(rsi, exitShare(train, "RSI"))
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
	row.RSIShare = medianF(rsi)
	row.HoldDays = medianF(hold)
	if n > 0 {
		row.Plateau = float64(plateau) / float64(n)
	}
	return row
}

// DropTickerInput is everything ScreenDropTicker needs about one instrument.
type DropTickerInput struct {
	Ticker            string
	Name              string
	Interval          enum.Interval     // timeframe of Bars; sets the strategy's bar step
	Bars              []backtest.Candle // bars over the whole window (train + holdout)
	Daily             []backtest.Candle // daily bars with a warm-up lead-in (ATR)
	Lot               int32
	MinPriceIncrement float64 // 0 when unknown
}

// ScreenDropTicker replays every grid configuration over one ticker and reduces the runs to a
// report row. The strategy is built directly with dropcore.NewWithParams and NOT through
// DropBounceLookupOrGeneric: registered tickers carry calibrated literals, and grading them on
// those would make their rows incomparable with the rest. Entries fill at the next bar's open,
// as cmd/backtest runs drop_bounce.
func ScreenDropTicker(in DropTickerInput, cfgs []dropcore.Params, split time.Time, opts ScreenOpts) DropRow {
	cfg := backtest.Config{
		InitialCash:     opts.Cash,
		Fraction:        opts.Fraction,
		Commission:      opts.Commission,
		Lot:             in.Lot,
		EntryAtNextOpen: true,
	}
	step := in.Interval.ToTimeDuration()
	results := make([]DropConfigResult, 0, len(cfgs))
	for _, p := range cfgs {
		// drop_bounce needs no higher-timeframe series: htfCandles is nil.
		res := backtest.Run(dropcore.NewWithParams(in.Ticker, step, p), in.Bars, in.Daily, nil, cfg)
		results = append(results, DropConfigResult{Params: p, Trades: res.Trades})
	}
	row := AggregateDrop(in.Ticker, in.Name, results, split, in.MinPriceIncrement, opts)
	row.Bars = len(in.Bars)
	row.TurnoverM = backtest.MeanDailyTurnoverM(in.Bars, in.Lot)
	row.DailyATRPct = MeanDailyATRPct(in.Daily, screenDailyATRPeriod)

	var lastClose float64
	if len(in.Bars) > 0 {
		lastClose = in.Bars[len(in.Bars)-1].Close
	}
	row.TickPct, row.TickUnknown = tickPct(in.MinPriceIncrement, lastClose)
	return row
}

// DropPregateRow builds the row of a ticker rejected on daily turnover before any intraday load or
// grid replay. Turnover is averaged over the daily candles inside [from, to] only (the lead-in year
// is excluded); ATR runs over the whole daily slice and the tick comes from its last close, as in
// ScreenDropTicker. The caller compares TurnoverM with the gate.
func DropPregateRow(ticker, name string, daily []backtest.Candle, from, to time.Time, lot int32, minPriceIncrement float64) DropRow {
	window := make([]backtest.Candle, 0, len(daily))
	for _, c := range daily {
		if !c.Time.Before(from) && !c.Time.After(to) {
			window = append(window, c)
		}
	}
	row := DropRow{
		Ticker:      ticker,
		Name:        name,
		Pregated:    true,
		TurnoverM:   backtest.MeanDailyTurnoverM(window, lot),
		DailyATRPct: MeanDailyATRPct(daily, screenDailyATRPeriod),
	}
	var lastClose float64
	if len(daily) > 0 {
		lastClose = daily[len(daily)-1].Close
	}
	row.TickPct, row.TickUnknown = tickPct(minPriceIncrement, lastClose)
	return row
}
