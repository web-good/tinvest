package live

import (
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/elfv"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ivat"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lsngp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svav"
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
// IVAT was added and calibrated 2026-08-18 and it is the weakest evidence in this map, on three
// counts that stack. First, the RUNS ARE NOT THE STANDARD PROTOCOL: the instrument IPO'd in June
// 2024 and has 26.0 months of history, so §8 (-months 36 -train-months 12 -test-months 6) cannot
// produce four folds at all; the owner adapted the scheme to -months 25 -train-months 9
// -test-months 4, which makes IVAT's numbers incomparable line by line with everything above.
// Second, BOTH key themes miss the bar declared before the runs, and entry misses it by the
// widest margin this catalogue has seen — pooled OOS PF 0.979 on 47 trades, the first time a
// ticker's entry theme has gone below one; trend measures 1.282 with its leading axis stable in
// 2 folds of 4 and reversing mid-history (EMASlow 200/170/50/50). Third, the accepted point does
// measure pooled 1.988 on 55 trades, but 85% of that comes from ONE WEEK: the four largest trades
// of the entire history are 21-27 July 2026, and on the OOS of the first three folds — a full
// year, 42 trades — the same configuration measures 1.209. Two properties are worth knowing
// before touching its numbers. The regime is the mirror image of LSNGP: not one rising half-year
// in the whole history (train −45.6%, holdout −58.8%, peak-to-trough −87.6%), so nothing here has
// been tested against a RISING market, and the trend filter is the most closed in the catalogue
// (open 29.4-36.0% of bars against 54-60% on LSNGP). And liquidity stacks an execution risk on
// top of the statistical one: 43 mln RUB of median daily turnover with a p10 of 7 mln puts half
// of all days below the screener's own 50 mln gate — LENT territory. The literal itself departs
// from the theme leaderboards on two fields on purpose (EMASlow 70 where trend gave no answer,
// StopDailyATR 0.7 where the theme picked the widest edge of the axis); the package doc measures
// both.
// SVAV was added and calibrated 2026-08-21 on the standard protocol (-months 36 -train-months 12
// -test-months 6): unlike IVAT and DOMRF, its history is exactly 36.0 months, so its numbers are
// comparable line by line with the rest of the catalogue rather than adapted to a shorter window.
// It is the FIRST ticker in this catalogue to take the declared bar whole: entry pooled OOS PF
// 2.139 on 47 trades with the leading axis RSILower stable in 3 folds of 4, and trend pooled OOS
// PF 2.343 on 104 trades with the leading axis EMASlow stable in 3 folds of 4 — both key themes
// clear 1.5 with the required axis stability, a first for this map. The regime it was tested
// against is hostile: both protocol windows fall, the whole history is -58.9%, peak-to-trough
// -83.4%, and only one of six half-years rises, by +0.8% — the long result here is not flattered
// by the market. Daily ATR(14) medians 4.38% of price, the second-highest in the catalogue after
// FESH, so a 0.5 ATR stop moves 2.2% of price at Fraction=1. Median daily turnover is 68 mln RUB,
// ahead of IVAT, LENT and LSNGP. The package doc carries two further caveats worth reading before
// touching the numbers: the entry theme's fold 3 is a thin-sample overfit (in-sample 7.893 vs OOS
// 0.568 on 4 trades), and the volume theme never reaches 3-of-4 axis stability at all.
// SIBN was added and calibrated 2026-08-21 on the standard protocol, right after SVAV, and it is
// the exact opposite case: SVAV took the declared bar whole, SIBN misses BOTH halves on BOTH key
// themes and by the widest margin here. entry measures pooled OOS PF 1.378 on 50 trades with its
// leading axis RSILower stable in 1 fold of 4 (35/30/15/20 — four different values), trend
// measures 0.847 on 94 trades with EMASlow stable in 0 folds of 4 (100/120/50/200), the first
// zero-stability leading axis in this catalogue. Not one of the nine themes reaches 1.5 and three
// of them are outright unprofitable. The prior said as much before the runs and was written down
// then: the screener's PFmed HO column reads 0.20 on 9 trades, the worst of the rating's first
// nine, and the control baseline run over 36 months measures PF 1.027 on 129 trades — the
// instrument trades flat on core defaults. What changed the picture is that those defaults sit
// outside the zone where the strategy works here (the NVTK precedent): the accepted point, built
// by point runs over that zone, measures pooled OOS PF 2.258 on 89 trades with three folds above
// 3.0, no degenerate fold, and a result spread over 47 weeks rather than concentrated in one (the
// IVAT failure mode was checked for explicitly and is absent — best week 17.9%, all five
// half-years positive). That point is still hand-picked over the whole history and is NOT
// out-of-sample. Two properties are worth knowing before touching its numbers. Daily ATR(14)
// medians 2.52% of price — the LOWEST in this catalogue — so the 0.7 ATR stop moves 1.76% of price
// at Fraction=1, and a round of costs eats 5.7% of the risk instead of the 2-3% typical higher up
// this map. Against that, liquidity is the BEST here by a wide margin: 519 mln RUB of median daily
// turnover with a p10 of 206 mln, four times the screener's own 50 mln gate even on its tenth
// percentile, so execution risk does not stack on top of the statistical one — unlike IVAT and
// LENT. The risk that the runs cannot close is dividend gaps: SIBN pays regularly, the strategy
// holds overnight, and five gaps in the window opened worse than -3% (the deepest -8.49%, or -4.69
// daily ATR). On such a morning the stop is a wish, not a floor, and there is no parameter in the
// strategy against it.
//
// ELFV was calibrated 2026-08-22 on the standard schedule and is the fourteenth entry here. It
// misses the declared bar, but in a shape this catalogue had not seen: BOTH key themes clear 1.5
// on profit factor and BOTH fail the stability half. entry measures pooled OOS PF 1.575 on 106
// trades with RSILower stable in 1 fold of 4 (30/20/25/35); trend measures 1.545 on 147 trades
// with EMASlow stable in 0 folds of 4 (200/120/150/50). All ten themes land above 1.38 and none is
// unprofitable — the flattest theme table here — yet no axis is stable in more than 2 folds of 4.
// This ticker also settles the question left open after SIBN. It carried the STRONGEST prior of
// every candidate, written down before the runs: the screener's PFmed HO reads 1.26 and the
// control baseline over 36 months measures PF 1.546 on 181 trades. The prior neither softened the
// bar nor predicted the outcome — the protocol failed exactly as it failed on tickers with a weak
// prior. Read that together with SIBN (baseline 1.027, same verdict): the screener columns and the
// baseline run predict the protocol NEITHER from below NOR from above. The accepted point measures
// pooled OOS PF 2.863 on 68 trades with ALL FOUR folds profitable — including the fourth, declared
// hard in advance because it covers a half-year down 30.5% — and no degenerate fold; best week
// 28.3% of the result, all five half-years positive. That point is still hand-picked over the
// whole history and is NOT out-of-sample.
//
// The risk on ELFV is EXECUTION, not the prior. Its price step is 0.0002 RUB — 0.039% of the
// 0.5134 median price — and the backtest fills at bar close without modelling the spread, so the
// real round of costs sits closer to 0.2% than to the 0.1% modelled. The sensitivity is measured:
// the accepted point drops from pooled OOS PF 2.863 to 2.346 when the round doubles, and its
// fourth fold falls below 1.0 (1.250 -> 0.944); on the earlier VolMult 2.0 probe the same doubling
// reads 2.003 -> 1.668 -> 1.381 at rounds of 0.1% / 0.2% / 0.3%. Live results will land between
// those two numbers, nearer the lower one on thin days. Liquidity is the WORST in this map: median
// daily turnover 25.1 mln RUB with a p10 of 6.4 mln, so on one weekday in ten the whole book turns
// less than seven million — size the live position against p10, not the median. The regime is the
// harshest here as well: the window is down 49.5% end to end, the holdout half-year down 32.5%,
// and only two of six half-years rose at all, so nothing about this ticker was flattered by a
// rising market. Bar coverage is ragged — 18.9% of weekdays carry fewer than twenty bars and 3.4%
// of bars have zero range — which is why VolLookbackBars was swept for the first time in this
// catalogue (theme cal_vol_window.json) and why its default of 3 was kept: the axis is alive but
// its maximum sits exactly on the default. The one risk this ticker does NOT carry is dividend
// gaps: EL5-Energo has not paid since 2020, and over 36 months exactly one gap opened worse than
// -3% (-0.95 daily ATR) against five on SIBN. It is the only name in the universe where holding
// overnight carries no ex-dividend risk.
//
// DIAS was calibrated 2026-08-22 on an ADAPTED schedule: the instrument IPO'd 2024-02-13 and has
// 30.3 months of history, so the standard §8 protocol (-months 36 -train-months 12 -test-months 6)
// cannot produce four folds; the runs use -months 30 -train-months 10 -test-months 5 instead — four
// folds of train 10 + OOS 5 months back to back (10 + 4x5 = 30), the same kind of adaptation as
// IVAT (25/9/4). DIAS's numbers are comparable to IVAT's but NOT line by line with the rest of the
// catalogue, which runs the full 36/12/6. It MISSES the declared bar, in a shape between ELFV and
// SIBN: entry fails BOTH criteria, like SIBN — pooled OOS PF 1.213, below the 1.5 bar, with its
// leading axis RSILower stable in 2 folds of 4, below the required 3 — while trend clears only
// profit factor (pooled OOS PF 1.848) with EMASlow stable in 2 folds of 4, the same partial failure
// both ELFV themes showed. It carried the SECOND-STRONGEST prior in this catalogue, written down
// before the runs, and the prior did not save the bar here either, same as ELFV: the screener's
// PFmed reads 1.62 and PFmed HO 3.35, and the control baseline over 30 months measures
// core.DefaultParams() at 153 trades, PF 1.822. The ticker is added anyway under the owner's rule
// announced BEFORE the runs, and the plan's stop condition (pooled OOS PF below 1.0, or under 20
// trades over the adapted 30-month window) did not fire: the accepted point measures pooled OOS PF
// 2.361 on 98 trades with all FOUR folds profitable. The regime is the sharpest caveat here, and it
// cuts the opposite way from the usual worry: ALL SIX half-years of the window fall (-6.7% / -26.7%
// / -18.5% / -38.2% / -4.2% / -29.4%), the whole window -79.0%, peak-to-trough -85.7% — the point is
// profitable in every one of the six, but the configuration has not been tested against a rising
// market AT ALL, the mirror image of LSNGP's risk. Liquidity is thin: median daily turnover 42.6
// mln RUB with a p10 of 13.6 mln, in IVAT/LENT territory. The one execution property that IS better
// than ELFV here is the price step: 0.5 RUB is 0.0165% of the 3033.5 median price, against ELFV's
// 0.039% — 2.4x cheaper. A procedural note for the catalogue: the daily cache was silently broken —
// DIAS_Day1.json held only 600 candles with a full missing half-year, 2024-06-24…2024-12-23 (143
// weekdays present in the 30-minute series but absent from the daily one) — fixed by a full
// -refresh before the first measurement, which brought the daily series to 745 candles. Lesson for
// future tickers: check BOTH series for continuity, not just history depth.
//
// BSPB was calibrated 2026-08-24 on the STANDARD schedule (36/12/6, like SVAV, SIBN and ELFV;
// adapted schedules were only needed for IVAT and DIAS, whose history is too short) and is the
// sixteenth entry here. History margin is ZERO, the same as SVAV: the 30-minute series starts
// exactly 36 months before the right edge of the window. It MISSES the declared bar in a shape
// this catalogue had not seen — both key themes land below ONE, not just below 1.5: entry
// measures pooled OOS PF 0.726 on 69 trades with RSILower stable in 2 folds of 4, trend measures
// 0.980 on 95 trades with EMASlow stable in 3 folds of 4 (the one criterion this ticker does
// clear). Of the four tickers whose theme numbers are still preserved in their own grids — DIAS
// (1.213 / 1.848), ELFV (1.575 / 1.545), SIBN (1.378 / 0.847), IVAT (0.979 / 1.282) — none of
// them had BOTH key themes fall below one; the comparison is limited to those four, not to all
// fifteen predecessors, since earlier tickers' theme numbers were not kept in their grids. The
// accepted point measures pooled OOS PF 2.336 on 47 trades with all FOUR folds profitable
// (3.562/11, 2.241/14, 1.585/15, 3.177/7), but that four-for-four record is not unique to the
// point — four of BSPB's ten themes achieve it too, and the nearest, cal_exit, measures 2.261 on
// the same 47 trades, only 3.3% below the point. Under doubled costs (-commission 0.001) the
// point measures PF 1.829, a 21.7% loss — more sensitive than both DIAS (19.7%) and ELFV (18.1%),
// but of the same order. The theme procedure here is HYBRID and its cost is recorded plainly: the
// anchor for BSPB's seven late themes was assembled by probing whole winning fold tuples over the
// FULL history, i.e. with lookahead, so those seven themes' leaderboards are conditional on top
// of the usual point caveat and are not comparable line by line with the rest of the catalogue.
// The main live risk
// is EX-DIVIDEND GAPS, and it is operational rather than statistical or execution-shaped like its
// neighbours (DIAS's regime risk, ELFV's execution risk): seven gaps worse than -3% over 36
// months, the worst -8.44%, on a regular May/October cycle — unlike the rest of the catalogue the
// dates are known in advance, so the risk is manageable by schedule rather than by a parameter.
// Liquidity is the BEST of any candidate measured so far: median daily turnover 333 mln RUB, tick
// size 0.01 RUB is 0.0029% of the median price — the cheapest step among those measured (DIAS
// 0.0165%, ELFV 0.039%; the rest of the catalogue is unmeasured), real round cost ≈0.106% against
// the 0.1% modelled — and daily ATR(14) medians 2.53% of price, narrow but not the narrowest:
// SIBN (2.52%) and DOMRF (2.02%) sit below it.
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
	ivat.Ticker:  ivat.DefaultParams(),
	svav.Ticker:  svav.DefaultParams(),
	sibn.Ticker:  sibn.DefaultParams(),
	elfv.Ticker:  elfv.DefaultParams(),
	dias.Ticker:  dias.DefaultParams(),
	bspb.Ticker:  bspb.DefaultParams(),
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
