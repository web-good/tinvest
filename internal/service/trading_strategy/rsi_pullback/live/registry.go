package live

import (
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lsngp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"
)

// paramsByTicker maps every rsi_pullback ticker the runner knows to its params. The
// configured universe (RSI_PULLBACK_TICKERS) selects which of these actually trade; every entry
// here now carries a calibrated literal, and none of the map tracks the baseline any more. Since
// 2026-08-17 the default universe is the WHOLE map: the owner put in the last three tickers that
// had a literal but no decision — RENI, NVTK and LSNGP. That decision does not upgrade their
// evidence, and the map is now the place where all ten risks live side by side. NVTK was
// calibrated 2026-08-16 and its nine themes missed the declared bar by the widest margin in this
// catalogue — one theme of nine above 1.5 pooled OOS PF (volume 1.674), entry 1.218 and trend
// 1.044. Its accepted point does measure 2.823 pooled on 93 trades with all four folds above 1.89,
// but that point was assembled by hand over the whole history and is not out-of-sample; read the
// package doc, starting with what the day gate does there. In exchange NVTK is the most liquid
// name here, the only one of the three added where execution risk does not stack on top of the
// statistical one. LSNGP was added and calibrated 2026-08-14: its themes missed the bar declared
// before the runs — every one of the nine cleared 1.5 pooled OOS PF, a first for this catalogue,
// but the leading axis was stable in only 2 folds of 4 on both key themes. Read its package doc,
// starting with the regime note: BOTH protocol windows rise (+46.7% / +11.2%), so nothing about
// that ticker has been tested against a falling market — the sharpest open question in the
// universe. FESH (2026-08-13), WUSH and LENT
// (both 2026-08-14) do have literals and the owner did put all three into the universe, in each
// case as a risk taken with eyes open rather than as a confirmation: for none of them does the
// standard §8 protocol confirm the instrument. T was calibrated in full on 2026-08-16, having
// traded live since 2026-08-05 on a literal that had no walk-forward at all; the protocol does
// not confirm it either (entry 0.864, trend 1.536 stable in 2 folds of 4), and the recalibration
// changed exactly two fields — the ATR trail is now armed at 0.5 daily ATR, which can only close
// a position EARLIER than the previous literal would, never later. RENI was recalibrated the same
// day and both of its known caveats are now closed: the target that disagreed with the
// walk-forward winner is accepted at 0.6 (1.5 never fired once in 36 months), and the fold that
// used to collapse to 0.336 now measures 1.351, because the trend filter was slowed to EMA 200.
// Of the three tickers added 2026-08-17 it is the closest to the bar — six themes of nine above
// 1.5, the mandatory pair cleared on PF (entry 1.955, trend 1.920) and failed only on the
// stability of its leading axis. WUSH missed the bar declared before its runs
// twice; LENT missed it too (entry 1.355, trend 1.777 with an unstable leading axis) and carries
// a second risk of a different nature — 38 mln RUB of median daily turnover, the thinnest of
// every ticker here. Every one of those literals was hand-picked over the whole history, which is
// not out-of-sample — see the package docs before touching any of them. UGLD was recalibrated
// 2026-08-17, the last prod ticker still running a literal from the narrow grids: three fields
// changed (RSIUpper 60 → 55, FreshDayATR 0.3 → 0, TPDailyATR 1.0 → 1.5), the accepted point
// measures pooled OOS PF 3.627 on 23 trades against 3.125 on 24 for the previous literal, and the
// declared bar is missed here too (entry 1.195/71, trend 3.780 on only 16 trades with a fold that
// has no trades at all). Two properties of that ticker are worth knowing before touching its
// numbers: the entry angle RSI(6)@15 is a PEAK, not a plateau (RSILower 20 drops the point to
// 1.580), and StopDailyATR is inert while the trail is armed — 0.5, 0.7, 1.0 and 1.3 measure
// byte-identical, because the 0.5-ATR trail binds from the first bar. It is also no longer the
// only prod ticker with the day gate's early branch on: that branch is now off everywhere.
var paramsByTicker = map[string]core.Params{
	ugld.Ticker:  ugld.DefaultParams(),
	tbank.Ticker: tbank.DefaultParams(),
	gazp.Ticker:  gazp.DefaultParams(),
	lent.Ticker:  lent.DefaultParams(),
	lsngp.Ticker: lsngp.DefaultParams(),
	nvtk.Ticker:  nvtk.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
	reni.Ticker:  reni.DefaultParams(),
	fesh.Ticker:  fesh.DefaultParams(),
	wush.Ticker:  wush.DefaultParams(),
}

// ParamsFor returns the params for a known ticker, ok=false otherwise.
func ParamsFor(ticker string) (core.Params, bool) {
	p, ok := paramsByTicker[ticker]
	return p, ok
}

// StrategyFor returns the strategy for a known ticker, ok=false otherwise.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
