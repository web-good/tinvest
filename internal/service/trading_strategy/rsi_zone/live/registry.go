// Package live подключает rsi_zone к живому раннеру счёта rsi_pullback
// (rsi_pullback/live) адаптером livecore/adapter.Strategy. Механика — docs/rsi_zone/live.md.
package live

import (
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/aflt"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/astr"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/bane"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/baza"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/fixr"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/ivat"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/lent"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/moex"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/msng"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/mtss"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/ragr"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/reni"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/smlt"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/svav"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/svcb"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/tatnp"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/trmk"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/trnfp"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/unac"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/upro"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/vsmo"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/wush"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/ydex"
)

// paramsByTicker — тикеры, которые раннер знает. SBER сюда не заводится: калибровка планку
// не взяла. Торгуют только перечисленные в RSI_ZONE_TICKERS.
var paramsByTicker = map[string]core.Params{
	afks.Ticker:  afks.DefaultParams(),
	aflt.Ticker:  aflt.DefaultParams(),
	msng.Ticker:  msng.DefaultParams(),
	bane.Ticker:  bane.DefaultParams(),
	baza.Ticker:  baza.DefaultParams(),
	dias.Ticker:  dias.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
	ivat.Ticker:  ivat.DefaultParams(),
	lent.Ticker:  lent.DefaultParams(),
	mdmg.Ticker:  mdmg.DefaultParams(),
	ragr.Ticker:  ragr.DefaultParams(),
	upro.Ticker:  upro.DefaultParams(),
	mtss.Ticker:  mtss.DefaultParams(),
	vsmo.Ticker:  vsmo.DefaultParams(),
	astr.Ticker:  astr.DefaultParams(),
	svav.Ticker:  svav.DefaultParams(),
	wush.Ticker:  wush.DefaultParams(),
	trnfp.Ticker: trnfp.DefaultParams(),
	ydex.Ticker:  ydex.DefaultParams(),
	smlt.Ticker:  smlt.DefaultParams(),
	tatnp.Ticker: tatnp.DefaultParams(),
	reni.Ticker:  reni.DefaultParams(),
	fixr.Ticker:  fixr.DefaultParams(),
	moex.Ticker:  moex.DefaultParams(),
	trmk.Ticker:  trmk.DefaultParams(),
	unac.Ticker:  unac.DefaultParams(),
	svcb.Ticker:  svcb.DefaultParams(),
}

// ParamsFor возвращает параметры тикера; false — тикер не зарегистрирован.
func ParamsFor(ticker string) (core.Params, bool) {
	p, ok := paramsByTicker[ticker]
	return p, ok
}

// StrategyFor строит ядро стратегии для тикера; false — тикер не зарегистрирован.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
