package live

import (
	"strings"

	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
	"tinvest/internal/service/trading_strategy/gap_fade/strategy/shared"
)

// ParamsFor — общий литерал для любого непустого тикера. Калибровки по тикерам у gap_fade нет;
// вход ограничивает вселенная GAP_FADE_TICKERS, а позиция тикера, убранного из неё, должна
// дойти до выхода, поэтому реестр её не отвергает.
func ParamsFor(ticker string) (core.Params, bool) {
	if strings.TrimSpace(ticker) == "" {
		return core.Params{}, false
	}
	return shared.Params(), true
}

// StrategyFor — ядро тикера с общим литералом.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := ParamsFor(ticker)
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
