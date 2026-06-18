// Package ugld supplies the ticker and reversion Params for UGLD (Yuzhuralzoloto / ЮГК).
package ugld

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "UGLD"

// DefaultParams returns UGLD's reversion parameters.
//
// 2026-06-18: UNCALIBRATED baseline. UGLD topped the daily-ATR volatility screen
// (mean ATR% ~4.4 over 6 months, deep liquidity), which makes it a natural
// mean-reversion candidate, so it is registered as a first-class reversion ticker.
// These are neutral starting params (mirroring the generic baseline); calibrate via
//
//	go run ./cmd/backtest -ticker UGLD -strategy reversion -interval Hour1 \
//	  -calibrate data/params/ugld/reversion_grid.json -train-months 12 -test-months 6 \
//	  -months 36 -metric profit_factor
//
// then hardcode the walk-forward winner here. UseRSI50 stays ON (always-on
// momentum-fade exit, per the project-wide convention).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
		UseRSI50:     1,
		UseBreakeven: 0, BreakevenArmATR: 1.0,
	}
}
