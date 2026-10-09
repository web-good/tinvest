// Package gazp supplies the ticker and the 15m drop_bounce Params for GAZP (ПАО «Газпром»).
//
// Заведён 2026-10-09 по просьбе владельца до калибровки: литерал повторяет дефолты ядра числами,
// чтобы бэктест GAZP не уезжал вместе с правкой core.DefaultParams. Калибровка не делалась —
// после неё сюда пишутся замеры, а в data/params/drop_bounce/gazp/ — сетки с _comment.
package gazp

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "GAZP"

// DefaultParams returns the GAZP literal for Minutes15.
func DefaultParams() core.Params {
	return core.Params{
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
