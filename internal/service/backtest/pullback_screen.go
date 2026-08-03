package backtest

import (
	"math"
	"sort"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/pkg/indicators"
)

// Pinned screener axes. The screener answers "is this ticker worth calibrating",
// so it sweeps only the parameters whose useful range is instrument-specific and
// pins everything else to one value shared by every ticker. See
// docs/superpowers/specs/2026-08-03-rsi-pullback-screener-design.md, section 3.
var (
	gridRSIPeriods = []int{4, 6}
	gridRSILowers  = []float64{10, 15, 20}
	gridEMASlows   = []int{100, 150}
	gridTPDailyATR = []float64{1.0, 1.5}
)

// PullbackGrid returns the 24 configurations every ticker is screened on, in a
// deterministic order. The volume gate is pinned OFF: it cuts trade count harder
// than any other filter, and the screening stage needs a sample, not a filter.
// Trailing is pinned OFF because it is a property of a tuned configuration, not
// of the instrument.
func PullbackGrid() []core.Params {
	out := make([]core.Params, 0, len(gridRSIPeriods)*len(gridRSILowers)*len(gridEMASlows)*len(gridTPDailyATR))
	for _, rsiPeriod := range gridRSIPeriods {
		for _, rsiLower := range gridRSILowers {
			for _, emaSlow := range gridEMASlows {
				for _, tp := range gridTPDailyATR {
					out = append(out, core.Params{
						RSIPeriod:       rsiPeriod,
						RSILower:        rsiLower,
						RSIUpper:        60,
						EMAFast:         20,
						EMASlow:         emaSlow,
						DailyATRPeriod:  14,
						UseDayATRGate:   1,
						FreshDayATR:     0.3,
						SpentDayATR:     0.8,
						StopDailyATR:    0.5,
						TPDailyATR:      tp,
						UseVolume:       0,
						VolBaseDays:     5,
						VolLookbackBars: 3,
						VolMult:         1.2,
						UseRSIExit:      1,
						UseTrail:        0,
						TrailDailyATR:   0,
					})
				}
			}
		}
	}
	return out
}

// profitFactor is gross profit over gross loss on an arbitrary SUBSET of trades.
// It deliberately differs from ComputeMetrics on one point: with no losing trade
// it returns +Inf rather than gross profit. The screener takes a MEDIAN across 24
// configurations, and a currency amount masquerading as a ratio would poison it;
// the caller clamps the infinity with clampPF instead.
func profitFactor(trades []backtest.Trade) (float64, int) {
	var gross, loss float64
	for _, t := range trades {
		if t.PnL >= 0 {
			gross += t.PnL
			continue
		}
		loss += -t.PnL
	}
	switch {
	case len(trades) == 0:
		return 0, 0
	case loss == 0 && gross > 0:
		return math.Inf(1), len(trades)
	case loss == 0:
		return 0, len(trades)
	}
	return gross / loss, len(trades)
}

// splitTrades cuts a trade list into the selection window and the holdout by ENTRY
// time: a trade opened before the split belongs to train even if it closed after it,
// because the entry is the decision the screener is grading. A trade entered exactly
// at the split goes to the holdout.
func splitTrades(trades []backtest.Trade, split time.Time) (train, holdout []backtest.Trade) {
	for _, t := range trades {
		if t.EntryTime.Before(split) {
			train = append(train, t)
			continue
		}
		holdout = append(holdout, t)
	}
	return train, holdout
}

// medianF is the median of vals; it copies before sorting so callers keep their order.
func medianF(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := append([]float64(nil), vals...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}

// clampPF caps a profit factor for ranking purposes and reports whether it bit.
// A limit of zero or less disables clamping. (The parameter is named `limit`, not
// `cap`, to avoid shadowing the builtin.)
func clampPF(pf, limit float64) (float64, bool) {
	if limit <= 0 {
		return pf, false
	}
	if pf > limit {
		return limit, true
	}
	return pf, false
}

// ConfigResult is one grid configuration's trade list on one ticker.
type ConfigResult struct {
	Params core.Params
	Trades []backtest.Trade
}

// ScreenOpts are the knobs of one screening run.
type ScreenOpts struct {
	PFCap         float64 // ranking cap on profit factor; 0 disables clamping
	PlateauPF     float64 // a configuration joins the plateau at this profit factor
	PlateauTrades int     // ...and at this many trades
	Cash          float64 // mock portfolio starting cash
	Fraction      float64 // fraction of cash per entry
	Commission    float64 // commission as a fraction of turnover, per side
}

// DefaultScreenOpts are the screener's defaults; the CLI overrides them by flag.
func DefaultScreenOpts() ScreenOpts {
	return ScreenOpts{
		PFCap:         10,
		PlateauPF:     1.3,
		PlateauTrades: 10,
		Cash:          100000,
		Fraction:      1.0,
		Commission:    0.0005,
	}
}

// PullbackRow is one ticker's screening result.
type PullbackRow struct {
	Ticker      string
	Name        string
	TurnoverM   float64 // mean daily turnover, millions of RUB (filled by ScreenTicker)
	DailyATRPct float64 // mean weekday daily ATR as a percentage of close (filled by ScreenTicker)
	Bars        int     // 30-minute bars the run replayed (filled by ScreenTicker)

	PFMed     float64 // MEDIAN profit factor across the grid on the train window: the ranking key
	TradesMed float64 // median trade count on the train window
	Plateau   float64 // share of configurations clearing PlateauPF at PlateauTrades trades
	Capped    int     // configurations whose train profit factor hit PFCap
	SilentCfg int     // configurations with zero train trades; each contributes PF=0 to PFMed

	PFMedHO     float64 // median profit factor on the holdout window: a red flag, never a ranking key
	TradesMedHO float64

	Best      core.Params // configuration with the highest train profit factor (reference only)
	BestPF    float64
	NoSignals bool // every configuration produced zero trades: profit factor does not exist
}

// Aggregate reduces one ticker's per-configuration results to a report row. The
// ranking key is the MEDIAN profit factor, never the best one: across 271 tickers
// and 24 configurations the maximum is a lottery, while the median asks whether the
// strategy works across a band of parameters.
func Aggregate(ticker, name string, results []ConfigResult, split time.Time, opts ScreenOpts) PullbackRow {
	row := PullbackRow{Ticker: ticker, Name: name, NoSignals: true}
	pfs := make([]float64, 0, len(results))
	counts := make([]float64, 0, len(results))
	pfsHO := make([]float64, 0, len(results))
	countsHO := make([]float64, 0, len(results))
	var plateau int
	var haveBest bool // tracks whether row.Best has ever been assigned a real grid entry

	for _, r := range results {
		train, holdout := splitTrades(r.Trades, split)

		pf, n := profitFactor(train)
		if n > 0 {
			row.NoSignals = false
		} else {
			row.SilentCfg++
		}
		// The first configuration always claims Best regardless of its own PF: a strict
		// "pf > row.BestPF" starting from the zero PullbackRow would never assign Best at
		// all when every configuration's raw PF is exactly 0 (e.g. all-losing train
		// windows), leaving row.Best at the zero-value core.Params{} — which the report
		// renders as "RSI 0/0, EMA 0/0, TP 0.0", a value that reads as a real (and wrong)
		// configuration rather than "no winner". Best must reference an actual grid entry
		// whenever results is non-empty.
		if !haveBest || pf > row.BestPF {
			row.BestPF, row.Best = pf, r.Params
			haveBest = true
		}
		pf, capped := clampPF(pf, opts.PFCap)
		if capped {
			row.Capped++
		}
		if pf >= opts.PlateauPF && n >= opts.PlateauTrades {
			plateau++
		}
		pfs = append(pfs, pf)
		counts = append(counts, float64(n))

		pfHO, nHO := profitFactor(holdout)
		if nHO > 0 {
			row.NoSignals = false
		}
		pfHO, _ = clampPF(pfHO, opts.PFCap)
		pfsHO = append(pfsHO, pfHO)
		countsHO = append(countsHO, float64(nHO))
	}

	row.BestPF, _ = clampPF(row.BestPF, opts.PFCap)
	row.PFMed = medianF(pfs)
	row.TradesMed = medianF(counts)
	row.PFMedHO = medianF(pfsHO)
	row.TradesMedHO = medianF(countsHO)
	if len(results) > 0 {
		row.Plateau = float64(plateau) / float64(len(results))
	}
	return row
}

// screenMSK anchors the weekday rule to Moscow, matching the strategy core.
var screenMSK = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// MeanDailyATRPct is the mean of ATR(period)/close over WEEKDAY daily candles, in
// percent. Weekend bars are dropped for the same reason the strategy drops them:
// MOEX weekend sessions are 3-4x narrower, and leaving them in understates the daily
// ATR by 9-16% (docs/rsi_pullback/strategy.md, section 5). Returns 0 when the series
// cannot support the calculation.
func MeanDailyATRPct(daily []backtest.Candle, period int) float64 {
	if period <= 0 {
		return 0
	}
	highs := make([]float64, 0, len(daily))
	lows := make([]float64, 0, len(daily))
	closes := make([]float64, 0, len(daily))
	for _, c := range daily {
		switch c.Time.In(screenMSK).Weekday() {
		case time.Saturday, time.Sunday:
			continue
		}
		highs = append(highs, c.High)
		lows = append(lows, c.Low)
		closes = append(closes, c.Close)
	}
	if len(closes) < period+1 {
		return 0
	}
	atr := indicators.ATRSeries(highs, lows, closes, period)
	var sum float64
	var n int
	for i := range atr {
		if atr[i] <= 0 || closes[i] <= 0 {
			continue
		}
		sum += atr[i] / closes[i]
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n) * 100
}

// screenDailyATRPeriod matches the strategy's fixed DailyATRPeriod: the ATR% column
// must be measured with the same ruler the strategy sizes its stop with.
const screenDailyATRPeriod = 14

// ScreenTicker replays every grid configuration over one ticker and reduces the runs
// to a report row. The strategy is built directly with core.NewWithParams and NOT
// through RSIPullbackLookupOrGeneric: registered tickers (GAZP, T, UGLD) carry their
// own calibrated literals, and grading them on those would make their rows
// incomparable with the other 268.
func ScreenTicker(ticker, name string, bars, daily []backtest.Candle, lot int32,
	cfgs []core.Params, split time.Time, opts ScreenOpts,
) PullbackRow {
	cfg := backtest.Config{
		InitialCash: opts.Cash,
		Fraction:    opts.Fraction,
		Commission:  opts.Commission,
		Lot:         lot,
	}
	results := make([]ConfigResult, 0, len(cfgs))
	for _, p := range cfgs {
		// rsi_pullback needs no higher-timeframe series: htfCandles is nil.
		res := backtest.Run(core.NewWithParams(ticker, p), bars, daily, nil, cfg)
		results = append(results, ConfigResult{Params: p, Trades: res.Trades})
	}
	row := Aggregate(ticker, name, results, split, opts)
	row.Bars = len(bars)
	row.TurnoverM = backtest.MeanDailyTurnoverM(bars, lot)
	row.DailyATRPct = MeanDailyATRPct(daily, screenDailyATRPeriod)
	return row
}
