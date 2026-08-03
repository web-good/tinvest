package backtest

import (
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
