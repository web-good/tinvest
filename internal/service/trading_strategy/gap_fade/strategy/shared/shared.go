// Package shared holds the single gap_fade parameter set that live trades on every ticker of
// GAP_FADE_TICKERS. There is no per-ticker calibration: a ticker sees a handful of gaps a year,
// so a per-ticker profit factor is noise; the set is validated once, pooled, by cmd/gapwf.
//
// Choice (2026-10-02, cmd/gapwf on the 46-ticker cmd/gapscreen universe): the walk-forward folds
// pick GapATR 0.5 or 0.7 with TargetFill 0.75 and StopDailyATR 1.0. GapATR 0.7 is taken because
// it makes the same sum of returns with less than half the trades, which leaves a wider margin
// for real morning costs — the main live risk of this strategy.
package shared

import "tinvest/internal/service/trading_strategy/gap_fade/strategy/core"

// Params returns the live parameter set.
func Params() core.Params {
	return core.Params{
		GapATR:         0.7,
		MaxGapATR:      3.0,
		EntryBar:       0,
		TargetFill:     0.75,
		StopDailyATR:   1.0,
		DailyATRPeriod: 14,
		EODHour:        18,
		EODMinute:      30,
	}
}
