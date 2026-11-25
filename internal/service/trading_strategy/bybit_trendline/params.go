package bybit_trendline

import (
	"tinvest/internal/service/trading_strategy/bybit_trendline/strategy/core"
	"tinvest/internal/service/trading_strategy/bybit_trendline/strategy/ethusdt"
)

// ParamsFor selects parameters for a normalized Bybit spot symbol.
func ParamsFor(symbol string) core.Params {
	switch symbol {
	case ethusdt.Ticker:
		return ethusdt.DefaultParams()
	default:
		return core.Params{TakeProfitR: 2, InPlayMinRangePct: 1}
	}
}
