package backtest

import (
	"math"
	"sort"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
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
