package backtest

import (
	"fmt"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	pdlrsicore "tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// PDLRSIStochMode is one of the two pdl_rsi entry modes every ticker is screened in.
type PDLRSIStochMode int

const (
	PDLRSIStochOff   PDLRSIStochMode = iota // no stochastic: the RSI signal hour is the trigger
	PDLRSIStochLevel                        // the RSI and %K (below 20) low on the same hour
)

func (m PDLRSIStochMode) String() string {
	switch m {
	case PDLRSIStochOff:
		return "без стох."
	case PDLRSIStochLevel:
		return "уровень"
	}
	return fmt.Sprintf("mode(%d)", int(m))
}

// Pinned pdl_rsi screener axes: the two entry modes (each calibrated on its own by the skill),
// the hourly RSI length and the stop, at the ends of their calibration ranges.
var (
	pdlRSIGridModes      = []PDLRSIStochMode{PDLRSIStochOff, PDLRSIStochLevel}
	pdlRSIGridRSIPeriods = []int{4, 6}
	pdlRSIGridStops      = []float64{0.5, 1.0}
)

// PDLRSIModeOf returns the entry mode a params value runs in.
func PDLRSIModeOf(p pdlrsicore.Params) PDLRSIStochMode {
	if p.StochK == 0 || p.StochSmooth == 0 {
		return PDLRSIStochOff
	}
	return PDLRSIStochLevel
}

// PDLRSIGrid returns the 8 configurations every ticker is screened on, in a deterministic order
// (mode outermost, StopATR innermost). The rest — RSIArm 30, RSIFloor 22,
// EMA(200), target 2R, RSI exit 70, %K(14,1) under 20 on the RSI hour — are the core defaults. pdl_rsi runs on
// Hour1 only.
func PDLRSIGrid(interval enum.Interval) ([]pdlrsicore.Params, error) {
	if interval != enum.Hour1 {
		return nil, fmt.Errorf("pdl_rsi screener runs on Hour1 only, not %s", interval)
	}
	out := make([]pdlrsicore.Params, 0, len(pdlRSIGridModes)*len(pdlRSIGridRSIPeriods)*len(pdlRSIGridStops))
	for _, mode := range pdlRSIGridModes {
		for _, rsi := range pdlRSIGridRSIPeriods {
			for _, stop := range pdlRSIGridStops {
				p := pdlrsicore.DefaultParams()
				if mode == PDLRSIStochOff {
					p.StochK = 0
				}
				p.HourRSIPeriod, p.StopATR = rsi, stop
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// PDLRSIConfigResult is one grid configuration's trade list on one ticker.
type PDLRSIConfigResult struct {
	Params pdlrsicore.Params
	Trades []backtest.Trade
}

// PDLRSIModeStat is one entry mode's medians across its configurations on the train window.
type PDLRSIModeStat struct {
	PFMedStress float64
	TradesMed   float64
}

// PDLRSIRow is one ticker's pdl_rsi screening result. Medians run across grid configurations, on
// the train window unless the field says holdout.
type PDLRSIRow struct {
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
	TPShare   float64 // median share of target exits
	RSIShare  float64 // median share of RSI exits
	HoldDays  float64 // median of per-configuration median holding time, calendar days

	Modes [2]PDLRSIModeStat // per entry mode, indexed by PDLRSIStochMode

	Best       pdlrsicore.Params // configuration with the highest raw train PF (reference only)
	BestPF     float64
	BestHalves []HalfResult // half-year breakdown of Best's train trades
	NoSignals  bool         // no configuration traded in either window
	// Pregated: turnover failed on daily candles before the grid ran; PF/trade fields are unmeasured.
	Pregated bool
}

// AggregatePDLRSI reduces one ticker's per-configuration results to a report row. tick is the
// instrument's price increment used by the cost stress (0 = unknown). It mirrors AggregateDrop and
// adds the target exit share and the per-mode medians.
func AggregatePDLRSI(ticker, name string, results []PDLRSIConfigResult, split time.Time, tick float64, opts ScreenOpts) PDLRSIRow {
	row := PDLRSIRow{Ticker: ticker, Name: name, NoSignals: true}
	n := len(results)
	pfs, pfsStress, counts := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	pfsHO, pfsStressHO, countsHO := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	var halvesPos, topHalf, sl, tp, rsi, hold []float64
	var modePF, modeTrades [3][]float64
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
		mode := PDLRSIModeOf(r.Params)
		modePF[mode] = append(modePF[mode], pfS)
		modeTrades[mode] = append(modeTrades[mode], float64(cnt))

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
		tp = append(tp, exitShare(train, "TP"))
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
	row.TPShare = medianF(tp)
	row.RSIShare = medianF(rsi)
	row.HoldDays = medianF(hold)
	for m := range row.Modes {
		row.Modes[m] = PDLRSIModeStat{PFMedStress: medianF(modePF[m]), TradesMed: medianF(modeTrades[m])}
	}
	if n > 0 {
		row.Plateau = float64(plateau) / float64(n)
	}
	return row
}

// PDLRSITickerInput is everything ScreenPDLRSITicker needs about one instrument.
type PDLRSITickerInput struct {
	Ticker            string
	Name              string
	Interval          enum.Interval     // timeframe of Bars; sets the strategy's bar step
	Bars              []backtest.Candle // bars over the whole window (train + holdout)
	Daily             []backtest.Candle // daily bars with a warm-up lead-in (ATR)
	Lot               int32
	MinPriceIncrement float64 // 0 when unknown
}

// ScreenPDLRSITicker replays every grid configuration over one ticker and reduces the runs to a
// report row. The strategy is built directly with pdlrsicore.NewWithParams and NOT through
// PDLRSILookupOrGeneric: registered tickers carry calibrated literals, and grading them on those
// would make their rows incomparable with the rest. Entries fill at the next bar's open, as
// cmd/backtest runs pdl_rsi.
func ScreenPDLRSITicker(in PDLRSITickerInput, cfgs []pdlrsicore.Params, split time.Time, opts ScreenOpts) PDLRSIRow {
	cfg := backtest.Config{
		InitialCash:     opts.Cash,
		Fraction:        opts.Fraction,
		Commission:      opts.Commission,
		Lot:             in.Lot,
		EntryAtNextOpen: true,
	}
	step := in.Interval.ToTimeDuration()
	results := make([]PDLRSIConfigResult, 0, len(cfgs))
	for _, p := range cfgs {
		// pdl_rsi needs no higher-timeframe series: htfCandles is nil.
		res := backtest.Run(pdlrsicore.NewWithParams(in.Ticker, step, p), in.Bars, in.Daily, nil, cfg)
		results = append(results, PDLRSIConfigResult{Params: p, Trades: res.Trades})
	}
	row := AggregatePDLRSI(in.Ticker, in.Name, results, split, in.MinPriceIncrement, opts)
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

// PDLRSIPregateRow builds the row of a ticker rejected on daily turnover before any intraday load
// or grid replay, exactly as DropPregateRow does for drop_bounce.
func PDLRSIPregateRow(ticker, name string, daily []backtest.Candle, from, to time.Time, lot int32, minPriceIncrement float64) PDLRSIRow {
	d := DropPregateRow(ticker, name, daily, from, to, lot, minPriceIncrement)
	return PDLRSIRow{
		Ticker: d.Ticker, Name: d.Name, Pregated: true,
		TurnoverM: d.TurnoverM, DailyATRPct: d.DailyATRPct, TickPct: d.TickPct, TickUnknown: d.TickUnknown,
	}
}
