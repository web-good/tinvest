package ethusdt

import "tinvest/internal/service/trading_strategy/bybit_trendline/strategy/core"

const Ticker = "ETHUSDT"

func DefaultParams() core.Params {
	return core.Params{TakeProfitR: 5, InPlayMinRangePct: 0.1, RequirePreviousDayHigh: true}
}
