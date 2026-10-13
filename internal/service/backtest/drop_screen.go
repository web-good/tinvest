package backtest

import (
	"fmt"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	dropcore "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Pinned drop-screener axes. Like the zone screener, it answers "is this ticker worth
// calibrating", so it sweeps only the axes whose useful range depends on the paper: the volume
// threshold (how often a ticker spikes) and the two opposite exit shapes. See
// docs/superpowers/specs/2026-10-12-drop-bounce-screener-design.md, §4.
var (
	dropGridVolsByInterval = map[enum.Interval][]float64{
		enum.Minutes15: {0.05, 0.1, 0.2, 0.3},
		enum.Minutes30: {0.1, 0.2, 0.3, 0.5},
		enum.Hour1:     {0.15, 0.25, 0.4, 0.6},
	}
	dropGridRSIPeriods = []int{4, 6}
	dropGridRSIExits   = []float64{55, 75}
	dropGridGreenVols  = []float64{0, 1.5}
)

// dropGridVols returns the VolDailyMult axis of a timeframe; ok is false for an unscreened one.
func dropGridVols(interval enum.Interval) ([]float64, bool) {
	v, ok := dropGridVolsByInterval[interval]
	return v, ok
}

// DropGrid returns the 32 configurations every ticker is screened on for one timeframe, in a
// deterministic order (VolDailyMult outermost, GreenVolExit innermost). The stop is pinned at
// 1.0 daily ATR(10): a property of a tuned point, not of the paper, and the value walk-forward
// folds picked on GAZP and T. Only Minutes15, Minutes30 and Hour1 are screened, matching the
// timeframes drop-bounce-calibrate calibrates.
func DropGrid(interval enum.Interval) ([]dropcore.Params, error) {
	vols, ok := dropGridVols(interval)
	if !ok {
		return nil, fmt.Errorf("drop screener runs on Minutes15, Minutes30 or Hour1, not %s", interval)
	}
	out := make([]dropcore.Params, 0, len(vols)*len(dropGridRSIPeriods)*len(dropGridRSIExits)*len(dropGridGreenVols))
	for _, vol := range vols {
		for _, period := range dropGridRSIPeriods {
			for _, exit := range dropGridRSIExits {
				for _, gv := range dropGridGreenVols {
					out = append(out, dropcore.Params{
						VolDailyMult:   vol,
						VolAvgPeriod:   10,
						RSIPeriod:      period,
						RSIExit:        exit,
						StopDailyATR:   1.0,
						DailyATRPeriod: 10,
						StartHour:      7,
						GreenVolExit:   gv,
					})
				}
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
