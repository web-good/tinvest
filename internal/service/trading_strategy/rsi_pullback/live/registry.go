package live

import (
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/aqua"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/astr"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/banep"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/cnru"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/elfv"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/irkt"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ivat"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lsngp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/magn"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mvid"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nkhp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sfin"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sngsp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sofl"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/spbe"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svav"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svcb"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tgka"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/vsmo"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/x5"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ydex"
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
// package doc, starting with what the day gate does there. In exchange NVTK is the most liquid of
// the three added that day (RENI and LSNGP trade far thinner), the only one of the three where
// execution risk does not stack on top of the statistical one. LSNGP was added and calibrated 2026-08-14: its themes missed the bar declared
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
// this map. Against that, liquidity is not a risk factor either: 519 mln RUB of median daily
// turnover with a p10 of 206 mln — below T (4961), YDEX (2826), NVTK (2475) and GAZP (587) in this
// universe, but still four times the screener's own 50 mln gate even on its tenth percentile — so
// execution risk does not stack on top of the statistical one, unlike IVAT and LENT. The risk that
// the runs cannot close is dividend gaps: SIBN pays regularly, the strategy
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
//
// YDEX was calibrated 2026-08-26 on an ADAPTED schedule, the same shape as IVAT (25/9/4), and is
// the seventeenth entry here. The adaptation is forced by a corporate event, not by short IPO
// history: YNDX (Yandex N.V.) trading was halted 2024-06-14 and the redomiciled YDEX (МКПАО
// «Яндекс») started trading only 2024-07-24 — a 40-day hole in the 30-minute series with a price
// jump 4007 -> 4542 — so the calculation window covers only the life of the current security,
// 2024-07-25 … 2026-08-25 (25.0 months), and the half of the old window that belonged to the
// foreign-domiciled paper is dropped whole rather than reweighted. The theme procedure was
// CANONICAL: core defaults sat inside the working zone, so all ten themes ran over them with no
// anchor, unlike BSPB. It MISSES the declared bar — say plainly: NOT TAKEN — in the same shape
// ELFV showed and the second time this catalogue has seen it: BOTH key themes clear profit factor
// and BOTH fail only the stability half. entry measures pooled OOS PF 1.916 on 52 trades with its
// leading axis RSILower stable in 2 folds of 4 (10/25/35/25); trend measures pooled OOS PF 1.642
// on 71 trades with EMASlow stable in 2 folds of 4 (200/50/200/70). The control baseline on the
// same 25-month window is the second strongest of the TEN recorded in the catalogue, after DIAS
// (153 trades, PF 1.822): core.DefaultParams() measures 108 trades, PF 1.778, ahead of LSNGP
// 1.554, ELFV 1.546, IVAT 1.432, LENT 1.417, SVAV 1.368, BSPB 1.114, SIBN 1.027 and NVTK 0.984.
// The accepted point measures
// pooled OOS PF 3.014 on 77 trades with all FOUR folds profitable (2.915/17, 1.849/20, 4.920/26,
// 2.765/14); under doubled costs (-commission 0.001) it still measures PF 2.167, four-for-four
// above one. Liquidity removes the execution risk carried by ELFV, DIAS and LENT entirely: median
// daily turnover is 2826 mln RUB with only 0.2% of days shorter than 20 bars. What remains instead
// is a history risk new in shape rather than degree: the security has traded under this ticker for
// under two years, and its record holds almost no corporate events to test against.
//
// ASTR was calibrated 2026-08-31 on a schedule adapted twice. The instrument IPO'd 2023-10-13 and
// has only 34.5 months of history, so the standard §8 protocol (36/12/6) cannot produce four
// folds; the first attempt, -months 34 -train-months 10 -test-months 6, was itself broken by
// month-end overflow in cmd/backtest's AddDate arithmetic and a live run on the 31st collapsed it
// to THREE folds, degenerating the plan's stability rule. The corrected schedule is -months 34
// -train-months 9 -test-months 6 — four folds with a month of slack, no longer date-dependent. The
// owner kept ASTR's grids maximally wide, the same decision as BANEP, at a declared price: a wider
// axis raises the overfitting risk on the shortened nine-month training window, so per-fold
// stability is read more strictly. It MISSES the declared bar: entry measures pooled OOS PF 1.231
// on 70 trades, failing both PF and stability (leading axis RSILower stable in only 2 folds of 4);
// trend measures 1.141 on 93 trades, failing PF but clearing stability (EMASlow/EMAFast stable in
// 3 folds of 4). The ticker enters production anyway under the standing rule, the bar failure
// recorded plainly rather than softened. The accepted point measures pooled OOS PF 2.740 on 91
// trades, inflated by a disclosed fold-3 outlier (18.197 on 18 trades, below the 20-trade
// threshold); it departs from core defaults on exactly three fields — EMAFast 5, EMASlow 30,
// StopDailyATR 1.3 — the only ones three folds of four agreed on. The control baseline (34 months,
// core defaults) measures PF 1.093, second-to-last in the catalogue ahead of only SIBN and NVTK;
// all ten themes ran canonically, with no anchor. Liquidity removes execution risk (turnover p10 85
// mln RUB against the universe's 50 mln gate; only 1.5% of days shorter than 20 bars) and dividend
// risk is effectively absent — exactly one gap worse than -3% in 34 months, and it was a
// market-wide drop, not an ex-date, so unlike BANEP no ex-date calendar is needed. What remains is
// the heaviest regime in this catalogue (window -57.7%, max drawdown -82.9%, one growing half-year
// of six) and a wide-stop trap: at StopDailyATR 1.3 the SL-exit share is 1.4% (2 of 146) against
// 27.2% (43 of 158) at the default 0.5, trade count almost unchanged — most of the stop's PF gain
// is the loss escaping into the RSI exit, not real protection.
//
// SNGSP was calibrated 2026-09-01 on the canonical 36/12/6 schedule — the first of the last six
// tickers that needed no adaptation, and the owner kept its grids maximally wide anyway, the same
// decision as BANEP and ASTR. All ten themes ran canonically, no anchor. Both key themes MISS the
// declared bar: entry measures pooled OOS PF 1.216 on 69 trades, failing both PF and stability
// (leading axis RSILower stable at only 20/30/30/25 across folds); trend measures 1.415 on 151
// trades, failing PF but clearing stability (EMASlow stable at 50 in 3 folds of 4). The ticker
// enters production anyway under the standing rule. The accepted point measures pooled OOS PF 1.363
// on 69 trades (win rate 75.36%): folds 1.718/17, 2.870/11 and 2.095/23 are profitable, but fold 4 —
// the most recent — is not (0.597/18). Under doubled costs the point measures PF 1.007 on the same
// 69 trades: a 0.7% margin above break-even, the thinnest in the catalogue. The control baseline
// (core defaults, 183 trades) measures PF 1.535, fourth in the catalogue after YDEX 1.778, LSNGP
// 1.554 and ELFV 1.546. Liquidity and execution risk are both removed: SNGSP has the best turnover
// in the catalogue (median 1299 mln RUB, only 0.8% of days shorter than 20 bars) and the cheapest
// price-step share (0.0096% of price). What remains is a dividend risk, not a regime risk — the
// regime is the softest in the catalogue and all six half-years of the baseline are profitable — but
// the July ex-date gap runs to -14.06% (2025) and -9.86% (2024), about 5.4 daily ATR, wider than any
// stop or trail the point carries; and a wide-stop trap that the point escapes only because its
// trail (0.5 daily ATR) already sits at the stop level from entry, making StopDailyATR inert at any
// value at or above 0.5.
//
// NKHP was calibrated 2026-09-02 on the canonical 36/12/6 schedule, all ten themes canonical with no
// anchor, and the owner kept its grids maximally wide, the same decision as BANEP, ASTR, SNGSP and
// HEAD. The point is verified TWICE: the canonical run plus a mandatory control run on 24/12/3,
// because two structural boundaries sit inside the window — the session widened on 2024-09-26 (19
// half-hour weekday bars became 34) and liquidity fell by an order of magnitude (median daily
// turnover 153.0 mln RUB in 2023 against 14.4 mln in 2026). Both key themes MISS the declared bar:
// entry measures pooled OOS PF 1.612 on 112 trades, clearing PF but failing stability (leading axis
// RSILower at 50/30/25/25, at most two folds agreeing); trend measures 1.318 on 102 trades, failing
// both, its leading axis EMASlow returning four different values in four folds. The ticker enters
// production anyway under the standing rule. The accepted point departs from core defaults on
// exactly two fields — UseVolume 1 and StopDailyATR 1.0 — and measures pooled OOS PF 2.678 on 93
// trades with all four folds profitable and none degenerate, the cleanest fold table in the
// catalogue; the control 24/12/3 run holds at 1.642 on 46 trades, so the result is not an artefact
// of the paper's first, 19-bar regime. NKHP is the FIRST AND ONLY ticker whose volume gate wins a
// free choice in all four folds (+0.32 PF in both screen rows): on the thinnest paper of the
// universe a volume spike separates real participation from a single order. The control baseline
// (core defaults, 150 trades) measures PF 1.697, second in the catalogue behind YDEX 1.778, and all
// six of its half-years are profitable — the only such regime in the catalogue despite the
// instrument losing 84.0% over the window. Gap exposure is zero: the point held no position through
// any of the four large overnight drops and entered on none of those days. What remains is the
// heaviest set of execution risks in the live universe: the THINNEST LIQUIDITY of all — median
// weekday turnover 14.4 mln RUB in 2026 against the universe's 50 mln gate, and a fresh single-ticker
// screener run reports 46 mln, below that gate, because the 36-month average is inflated by 2023; a
// ROUND-TRIP COST OF 0.20-0.26% against the 0.1% the engine models (price step 0.5 RUB at a price of
// 391 RUB), so the cost model here is optimistic rather than conservative, which is why the point is
// also scored under tripled costs (PF 2.083); 13.8% of the point's exits land in the WEEKEND SESSION,
// whose median turnover is 1.09 mln RUB and whose slippage the engine does not model at all; and a
// wide-stop trap the point does NOT escape — the stop axis is live (the trail is off, so nothing
// overlaps it) but rises monotonically 2.446 -> 2.678 -> 2.927 across stops 0.7/1.0/1.3 at an
// unchanged 93 trades, and on the full history the SL-exit share falls from the baseline's 20.7% to
// 6.9% while overnight holds rise from 51.3% to 56.0% and weekend exits from 10.0% to 13.8%: 14 of
// the 22 losing trades close on the RSI exit, so the loss moved rather than shrank and the risk is
// amplified, not reduced.
//
// SOFL was calibrated 2026-09-03 on a SECOND round: the first round's point (canonical 36/12/6)
// scored pooled OOS PF 1.808 on 54 trades but failed the control 24/12/3 run at 0.848, and the
// owner ordered a recalibration on a fresh 24-month window (24/12/3 as the primary scheme, 36/12/6
// as the control) rather than accept the point. The procedure is HYBRID WITH AN ANCHOR, the reason
// stated up front rather than discovered after the fact: core defaults sit in a dead zone on SOFL
// (PF 0.987, a loss), so the early screen/entry/trend themes still run over core defaults for a
// bar-comparable verdict, while the five late themes run over the entry theme's winner and the
// trend theme's core defaults instead (trend never reached a majority) — their numbers are
// conditional and not comparable line by line with the rest of the catalogue.
// Both key themes MISS the declared bar on the second round: entry measures pooled OOS PF 0.416 on
// 27 trades, every one of its four folds unprofitable; trend measures 1.229 on 52 trades, its
// leading axis EMASlow returning two different values across folds. The ticker enters production
// anyway under the standing rule. The accepted point clears BOTH risk gates the owner declared
// before the runs — gate A (effective protection min(StopDailyATR, TrailDailyATR) survives at
// least 30% of weekdays: this point's 0.5 ATR stop survives 91.8%, nowhere near the 1.0 ceiling)
// and gate B (drawdown at most 1.3x the baseline's on the calibration window: 3.69% against a
// 18.15% ceiling) — and the pooled numbers back it: 1.453 on 27 trades on the primary 24/12/3
// scheme, 2.106 on 60 trades with all four folds profitable on the 36/12/6 control. Three risks are
// accepted with eyes open. First, liquidity has fallen for three straight years — 314 -> 208 -> 124
// -> 62 mln RUB of median weekday turnover by 2026 — leaving only a 1.2x margin over the
// universe's own 50 mln gate, an execution risk calibration cannot remove. Second, the instrument's
// whole window is down 67.1% with NOT ONE rising half-year, so the point has been tested only
// against decline; the first rising half-year triggers an out-of-cycle recalibration. Third, the
// weekend session carries a median turnover of just 6 mln RUB, and the engine models none of the
// slippage that implies for trades that exit there.
// TGKA joins twenty-third on 2026-09-03 as the universe's ONLY ticker trading the core defaults, and
// that is a decision made on numbers rather than a forgotten literal. Its calibration ran three
// times and lost to those same defaults every time: the first round's point measured 1.789 on 156
// trades on its own 36/12/6 scheme but collapsed to 0.899 on the 24/12/3 control, where the plain
// defaults hold 2.518; the second round, recalculated on the homogeneous window B, assembled a point
// so narrow it took six trades in three years and left ONE trade in the OOS pool; the third round
// kept the last six months as a blind holdout, voted only on folds 1-3, and its point returned
// PF 0.815 with net -1 070 there against 3.709 and +20 223 for the defaults. The mechanism is worth
// naming because it is new to the catalogue: every theme finds, in isolation, a value that beats the
// default on its training window, yet the chosen values COMPOSE into a configuration that loses out
// of sample — a defect of the majority rule, not a property of the instrument. There is nothing to
// improve here because the defaults already sit at the maximum of three separate axes on TGKA
// (RSILower 30, FreshDayATR 0, the day gate armed), and they deliver the catalogue's best baseline:
// PF 1.967 over 167 trades in 36 months, 2.518 over 131 in 24 months, 3.709 on the blind holdout.
// Overfitting is impossible by construction — the defaults were chosen across the whole catalogue,
// never against this ticker — and the named exception to the universe guard lives in
// registry_test.go (baselineByDesignTickers). The accepted risk is execution, and it is heavy:
// median weekday turnover has fallen 106.8 -> 44.1 -> 27.7 -> 26.0 mln RUB by 2026, leaving 22.4 mln
// over the last twelve months against the universe's own 50 mln gate, which the screener cleared
// only on a 36-month MEAN inflated by 2023; the weekend session adds 254 days at a 1.9 mln median
// the engine models no slippage for. Costs are not the problem: a 0.000002 RUB tick on a 0.006036
// RUB price is a 0.066% round trip against the 0.1% modelled, and the baseline stays profitable even
// at four times that. Review trigger: median turnover below 15 mln RUB removes the ticker from the
// universe out of cycle; a rise above 50 mln, or the instrument's first sustained rising half-year,
// calls for a fresh calibration.
//
// VSMO (VSMPA-AVISMA) joined on 2026-09-04 as the twenty-third ticker, and its accepted point is the
// closest to the core baseline in the whole catalogue: only two of eighteen fields differ, EMAFast
// 10 -> 5 (theme trend_low, three folds of four) and TPDailyATR 0.6 -> 0.3 (theme risk, three folds
// of four). The bar was missed for the twelfth time running — entry pooled OOS 1.133 with the
// leading axis split 10/15/15/30, trend took the PF criterion (1.684 over 115 trades) but split its
// leading axis across four different values in four folds. What the point does have is an unusually
// clean risk profile: pooled OOS 1.814 over 116 trades with all four folds profitable, 3.558 on the
// 24/12/3 control, and a full-window max drawdown of 4.92% — BELOW the 5.49% baseline rather than
// merely under the 7.14% gate B ceiling. Every one of the five third-contour numbers improved
// (SL exits 22.2% -> 18.1%, holding median 8 -> 6 bars, overnights 46.5% -> 39.6%, weekend-session
// exits 6.2% -> 5.4%).
//
// Two things about VSMO are worth knowing before touching its params. First, COSTS ARE THE ONE PLACE
// WHERE THE MODEL IS OPTIMISTIC rather than conservative: a 20 RUB tick on a 22-24k RUB price is a
// 0.13-0.18% round trip against the 0.1% modelled, which is why the stop condition carried a fourth
// clause for this ticker — the point had to stay above 1.0 at a doubled round trip, and it does with
// room (1.563). Second, RISK GATE A WAS NOT APPLIED ONLY BECAUSE THE RISK THEME TIED: all four folds
// voted for a stop of 1.3-1.5 ATR, levels reachable in 21.2% and 13.0% of weekdays against the 30%
// floor, and the 2/2 split is what left the stop at the default 0.5 (85.1%). Accepted risks:
// liquidity median 27.2 mln RUB over twelve months against the screener's 50 mln universe gate
// (review trigger: below 15 mln removes the ticker out of cycle), a weekend session with a 1.97 mln
// RUB median turnover the engine models no slippage for, and a regime of -53.0% over the window with
// a single rising half-year out of six — recalibrate out of cycle at the first sustained rising one.
//
// SPBE (SPB Exchange) joined on 2026-09-05 as the twenty-fifth ticker. The bar was missed for the
// fourteenth time running: entry pooled OOS 1.222 over 66 trades with the leading axis split
// 45/10/20/10 (both wins for the lower edge landed on the thinnest folds, 4 and 10 OOS trades), and
// canonical trend 1.180 over 83 trades — that theme did take criterion B with EMASlow 50 in three
// folds of four. The stop condition never fired: pooled OOS 1.777 over 64 trades on 36/12/6 against
// a 1.176/87 baseline, 1.617 over 30 trades on the 24/12/3 control against 1.156, and 1.593 at a
// doubled round trip (1.424 at a tripled one). The accepted point differs from the core baseline in
// three decisions, each carried by three folds of four: EMASlow 100 -> 50 (theme trend), UseVolume
// 0 -> 1 (theme screen), and a trailing stop switched on at 0.7 daily ATR (theme trail).
//
// BOTH RISK GATES PASSED, AND GATE A IS THE ONLY ONE THAT MATTERS HERE. Effective protection is
// min(stop 0.5, trail 0.7) = 0.5, reachable in 90.5% of the window's weekdays against the 30% floor,
// and the package test computes that minimum so a later widening of either field cannot slip past
// the ceiling (the AFKS lesson). Gate B passed on drawdown, 17.11% against a 24.83% baseline ceiling
// applied WITHOUT the 1.3 multiplier — but on this ticker gate B is blind by construction: probes at
// the accepted fields show the wide-stop trap raising PF from 1.575 to 3.140 while the trade pool
// stands still (106 -> 103), SL exits fall to zero, and MAX DRAWDOWN IMPROVES to 11.30%. A drawdown
// ceiling cannot catch a trap that lowers drawdown; only survivability can. The risk theme voted
// 1.3/1.5/1.5/1.3 — every fold above the ceiling — and only its 2/2 tie left the stop at 0.5. The
// price of that refusal is stated plainly: 2.952 pooled OOS for the theme against 1.176 baseline.
//
// Accepted risks. Costs: a 0.1 RUB tick is a 0.088% round trip at the twelve-month median price
// (227.80 RUB) and 0.140% at the current one (142.50 RUB) against the 0.1% modelled — review trigger:
// a price below 100 RUB pushes the round trip past 0.2% and calls for re-checking the fourth stop
// clause out of cycle. Liquidity is not a constraint: 164.9 mln RUB median turnover over twelve
// months, three times the screener's 50 mln universe gate, plus the catalogue's liveliest weekend
// session at a 27.85 mln RUB median the engine models no slippage for. The regime is the ragged
// extreme of the catalogue — half-years of -52.2% and +181.5% out of six, a 70.4% instrument
// drawdown inside the window; review trigger: two consecutive half-years with instrument amplitude
// under 15% call for a fresh calibration out of cycle. The issuer carries US blocking sanctions from
// November 2023, so headline moves exceed the daily ATR and the only defence in the core is the
// spent-day gate; review trigger: ANY change in sanction or listing status means recalibrate or drop
// the ticker. The screener's prior sat below the direct measurement (26th place, PFmed 1.37, holdout
// 0.90 on six trades), the same disagreement seen on ELFV. The session widened from 28 to 34
// half-hour bars inside the window, which is why the 24/12/3 control run is mandatory here.
//
// MVID (ПАО «М.видео») joined on 2026-09-07 as the twenty-sixth ticker, and it is the second in the
// catalogue after SVAV to take the declared bar WHOLE: canonical entry measures pooled OOS PF 1.754
// on 53 trades with RSILower stable at 10 in 3 folds of 4 and RSIPeriod stable at 3 in all 4 of 4,
// and canonical trend measures 1.543 on 82 trades with EMASlow stable at 50 in 3 of 4. Read that
// majority with care: on BOTH themes the 3-of-4 verdict rests on the thinner, older folds while the
// largest and most recent fold dissents on both — the bar is taken, not settled.
//
// The accepted point itself hit the catalogue's second stop-condition failure by RARITY, the same
// shape as UWGN: pooled OOS PF 1.697 on the canonical 36/12/6 schedule against the required 20
// trades, the pool holds only 15 — two losses, and two of the four folds carry no losing trade at
// all, so their fold PF is arithmetic on almost nothing, not a measurement. The 24/12/3 control
// reads 1.484 on 10 trades and is NOT an independent check: its test window is the same
// 2025-09-07..2026-09-07 stretch as folds 3 and 4 of the canonical run, the same ten trades read
// twice. The other four clauses did pass, each against its own baseline: clause 1 (canonical
// 36/12/6) 1.697 against a 1.326/100 baseline; clause 3 (24/12/3 control, the same non-independent
// 1.484/10 above) against a 1.469/56 baseline; clause 4 (36/12/6 at a doubled round trip,
// `-commission 0.001`) 1.404 against a 1.176 baseline; clause 5 (36/12/6 at MVID's real 0.217%
// round trip, `-commission 0.0011`) 1.348 against a 1.148 baseline — the clause this stop condition
// was written for. MVID trades live BY THE OWNER'S DECISION of 2026-09-07 (design §5.8.1), taken
// AFTER being shown the fired clause and the UWGN precedent it echoes — not because the stop
// condition passed whole.
//
// Both risk gates passed. Gate A: effective protection min(StopDailyATR 0.5, TrailDailyATR 0.5) =
// 0.5 daily ATR survives 86.2% of weekdays, nowhere near the 1.0 ceiling (36.0% survivability)
// reached by no accepted field. Gate B passed with a factor-of-two margin — max drawdown 5.64%
// against a HARD 11.10% ceiling, the baseline's own max DD with no 1.3x multiplier: MVID's baseline
// drawdown is among the highest measured for gate B, and the wide-stop trap begins inside the zone
// gate A allows (0.7 daily ATR), so trading drawdown for return was judged unsafe here, the same
// reasoning as SPBE's hard ceiling.
//
// The fifth clause exists because the cost model is optimistic 2.2x on this ticker: a 0.05 RUB
// price step is a 0.217% round trip at the last price (46.25 RUB), 0.153% at the twelve-month
// median (65.55 RUB), 0.104% at the window median (96.1 RUB), against the 0.1% the engine models.
//
// Four risks are accepted with eyes open, each carrying the review trigger declared for it. First,
// the round trip grows as price falls, since it is inversely proportional to price — review
// trigger: price below 40 RUB, re-check costs and recalibrate if needed. Second, turnover has
// fallen one-directionally and now sits below the screener's own universe gate: 41.40 mln RUB
// median over twelve months, 32.64 mln over six, against the screener's 50 mln floor — review
// trigger: median weekday turnover below 20 mln RUB over six months. Third, the instrument is in a
// one-directional decline, down 76.8% over the whole window (198.5 -> 46.25 RUB) with only two of
// six half-years rising; the strategy stays profitable through the decline, but the issuer's own
// credit risk (leverage, repeat recapitalisations) is not hedged by any parameter here. Fourth, the
// weekend session is thin — 3.25 mln RUB median turnover, two orders of magnitude below weekdays —
// and the point's share of exits landing there rose to 11.5% (3 of 26 trades on the full window)
// against the baseline's 6.7%: fewer such exits in absolute terms (3 against 10) but a larger share
// of a much smaller pool, a realised slippage risk.
//
// On top of those, four risks specific to the stop-condition failure are carried forward from the
// owner's decision (design §5.8.1). The 15-trade OOS pool spans three years — about one trade per
// ten weeks — and supports no confirmed statistical claim: two of the four folds have no losing
// trade at all, so their fold PF is division by near-zero. The passed clauses 1, 3, 4 and 5 inherit
// that same weakness; clause 5 in particular must NOT be quoted as proof this point survives real
// costs, only that it did not fail on this thin pool. The rarity itself is created by the
// COMBINATION RSIPeriod 3 x RSILower 10, not by either field alone: RSIPeriod 2 at the same
// RSILower gives a 93-trade pool at PF 1.406, and RSILower 15 at the same RSIPeriod gives 47 trades
// at PF 2.106 — both dominate the accepted point on PF and pool size at once, and both are the
// natural starting point for a second calibration round on the SOFL precedent (a fresh window, not
// a denser grid on the same one). Finally, the live runner will see this rarity as months of
// silence from the ticker — expected behaviour on MVID's accepted point, not a runner fault.
// CNRU (МКПАО «ЦИАН») joined on 2026-09-10 as the twenty-seventh ticker, and it is the SECOND entry
// in this map that trades the CORE DEFAULTS by design, after TGKA — but for the opposite reason.
// TGKA's calibration produced points and they lost to the defaults on all three slices. CNRU's
// calibration produced NO POINT AT ALL that differs from the defaults: of eighteen fields, only
// five drew a fold majority of 3-of-4, and three of those (UseDayATRGate 4-of-4, UseVolume 3-of-4,
// UseRSIExit 3-of-4) simply confirmed the default. The fourth, StopDailyATR at 2.0, was rejected by
// risk gate A (2.8% weekday survivability against a 30% floor), and the gate's own ceiling of 1.0
// was then rejected by anatomy: it carries the wide-stop trap signature — the SL exit share falls
// from 21.3% to 7.4% while holding time and overnight share both rise, meaning profit is bought by
// letting losers sit to the RSI exit instead of stopping out. The fifth, VolMult at 2.5, is INERT,
// because the volume gate is off (UseVolume 0 won 3-of-4) and the multiplier never enters the
// calculation: a neighbour probe at VolMult 1.2 reproduced the report byte for byte apart from the
// parameter line itself. So the accepted point is behaviourally identical to core.DefaultParams(),
// there is no literal to record, and CNRU is listed by name in baselineByDesignTickers.
//
// The numbers the defaults produce here are the reason the ticker is worth trading at all: pooled
// OOS PF 2.001 on 123 trades over 36/12/6 with FOUR profitable folds out of four and none degenerate
// (the thinnest holds 21 trades against the 20 floor), 1.739 on the 24/12/3 control, 1.715 at a
// doubled round trip and 1.455 at a 0.3% one, max drawdown 9.15%, and every calendar year of the
// window profitable (+2 528 / +22 942 / +45 937 / +12 489 RUB). Both risk gates pass: gate A with a
// wide margin (effective protection is StopDailyATR 0.5, surviving 88.5% of weekdays, since the
// trail stays off), gate B exactly at its hard ceiling — the point's max drawdown IS the baseline's,
// down to the same 12 081.22 RUB episode. The declared bar was NOT taken: both canonical key themes
// cleared the PF criterion (entry 1.884 on 82 trades, trend 1.771 on 121) and both failed the
// stability criterion, entry worst of all — RSILower voted 24, 15, 40, 30, four different values
// across four folds, spread over the full width of the grid. The 22-26 entry plateau measured before
// the grids were written was confirmed by exactly one fold of four.
//
// Read the 24/12/3 control with care rather than as a second confirmation: its pool is 57 trades
// across four folds of which fold 2 is degenerate to the point of meaninglessness (PF 5039 on eight
// trades with no losing trade and 0.00% drawdown, which is a metric artefact, not a result) and
// fold 3 is outright unprofitable (0.755, -2.47%). Three of its four folds sit below the 20-trade
// floor. The pool clears clause 3 of the stop condition, the fold-level picture does not support it.
//
// Five risks are accepted with eyes open. First and without precedent in this catalogue, trading in
// the instrument was HALTED for 57 days inside the window, 2025-02-05 to 2025-04-03, for the CIAN ->
// МКПАО «ЦИАН» redomiciliation. There was no split and no exchange ratio, so the series needs no
// adjustment, but a multi-day strategy with no time stop can be caught holding through such a halt:
// an exchange stop order does not execute in a suspended instrument and the runner has no exit
// mechanism at all. Both the baseline and the point measure ZERO exposure through the hole — the
// last position before it closed on 2025-02-05 itself and the first after it opened on 2025-04-03 —
// but that is history, not a mechanism, and the redomiciliation was recent enough that further
// corporate events are plausible. Review trigger: any announced trading suspension means flatten the
// ticker by hand before it starts. Second, the window contains the largest dividend gap in the
// catalogue, -16.18% on 2025-12-12 (a ~104 RUB special dividend at a ~640 RUB price), against 48.5%
// of the point's trades held overnight; exposure through it is zero, again by luck, and the company
// began paying only after redomiciliation, so cut-off dates will recur. Third, the wide-stop trap
// begins at the DEFAULT stop of 0.5 — the trade pool freezes at 164-162 all the way from 0.5 to 2.0
// while PF climbs 1.814 -> 2.763 — so gate A is structurally powerless against it here and the whole
// trap lies inside the zone gate A allows; that is why gate B was taken HARD (<= 9.15%, no 1.3x
// multiplier), and it is what rejected the 0.6 and 0.7 nodes (9.85% and 10.66%). Fourth, the trading
// session widened from 19 to 34 half-hour bars inside the window (2024H2), which is why the 24/12/3
// control on the homogeneous later stretch is mandatory here. Fifth, the weekend session is thin —
// 3.92 mln RUB median turnover — and 6.1% of exits land there, the same share as the baseline, a
// standing slippage risk rather than a realised one.
//
// Two review triggers are recorded explicitly. Median weekday turnover below 50 mln RUB over six
// months (the screener's own universe gate) means re-check the ticker: it currently reads 102.91 mln
// over six months and 118.49 over twelve, and liquidity is RISING, the first such case in the
// catalogue. Price below 200 RUB means re-check costs: the 0.2 RUB price step is a 0.064% round trip
// at 627 RUB — the cost model is PESSIMISTIC by a factor of 1.6 here, the opposite of MVID — but
// below 200 RUB the same step would exceed the 0.2% level that clause 4 of the stop condition tests.
//
// AQUA (ПАО «Совкомфлот») joined on 2026-09-10 as the twenty-eighth ticker, on a SECOND round, and
// it carries the SMALLEST literal in this map: one field. Its point is core.DefaultParams() with
// RSIUpper=45 instead of 70 — an early RSI exit and nothing else. That single field is the whole
// edge, which is why the parity check of Task 18 mattered more here than anywhere: a live-vs-backtest
// gap on the exit would not degrade the ticker, it would delete it. The defaults themselves lose on
// this instrument (136 trades, PF 0.931, three calendar years of four in the red), so the "trade the
// core defaults" route of TGKA and CNRU was closed from the start.
//
// The first round is part of the record because it was WRONG about which field carries the edge. It
// built a point on a deep entry (RSILower=10, a majority of 3-of-4 folds, accepted mechanically) and
// failed clause 3 of the stop condition (pooled OOS PF 0.786 on a 15-trade pool over 24/12/3). The
// second round, run on the homogeneous 24-month window the owner prescribed, put narrow grids around
// the first round's own fold votes: the deep entry collapsed outright (pooled OOS 0.214 on eight
// trades, no value drawing a majority) and returned to the default 30, while the RSIUpper=45 ×
// StopDailyATR pair that the first round had DEFERRED won 45 unanimously across all four folds.
//
// Risk gate B did real work here and its verdict must be read exactly. The second round's majority
// point turned the day gate OFF (4-of-4 folds) and drew 416 trades in two years at PF 1.225 — with a
// max drawdown of 14.18% against the 10% ceiling. Of seven measured neighbours exactly one clears the
// ceiling: putting the day gate back (the core default) gives 122 trades, PF 1.433 and 6.30%
// drawdown. So the accepted point stands on a value its own calibration rejected UNANIMOUSLY; the
// substitution is allowed by the rule declared before the runs, and the trade it refuses is explicit
// — the off-gate variant earns more (+19.52% against +11.75%) by buying it with a drawdown half again
// over the risk ceiling.
//
// What the point measures: 122 trades over 24 months (about five a month), PF 1.433, max DD 6.30%,
// expectancy 96.31 RUB; pooled OOS PF 1.487 on 124 trades over 36/12/6 with FOUR profitable folds of
// four, 1.619 on the 24/12/3 control, and all three calendar years of the window profitable (+1 315 /
// +9 997 / +437 RUB). The declared bar was not taken in the first round (entry 1.002, trend 0.773).
//
// Four risks are accepted with eyes open. First, clause 4 of the stop condition passes by eight
// thousandths — pooled OOS PF 1.008 at a doubled round trip, with two folds of four unprofitable and
// expectancy 1.96 RUB per trade. It survives only because the real round trip is small: the 0.1 RUB
// price step at ~327 RUB is 0.061%, so the tested 0.2% is over three times the live cost. Second,
// liquidity is FALLING — median weekday turnover 61.48 / 47.85 / 45.13 mln RUB over 36 / 24 / 12
// months — and the 24- and 12-month horizons both sit BELOW the screener's own 50 mln universe gate.
// Third, the instrument's regime is falling: buy&hold -67.46% over the window, one rising half-year
// out of seven, so the point has never been tested on a rising market in this name. Fourth, the
// session widened twice inside the 36-month window (19 -> 29 -> 34 half-hour bars), which is why the
// second round was run on the homogeneous 24-month stretch in the first place.
//
// Review triggers. Median weekday turnover below 50 mln RUB over six months means re-check the
// ticker — it is already below that line on two longer horizons. Price below 165 RUB means re-check
// costs: at that level the 0.1 RUB step reaches the 0.12% round trip where the thin margin of clause
// 4 would be gone. And because the whole literal is one exit field, any change to core's RSI exit
// semantics is a change to this ticker's only mechanism.
//
// MAGN (ПАО «Магнитогорский металлургический комбинат») joined on 2026-09-11 as the twenty-ninth
// ticker, on a FIRST round in which not one of the four stop-condition clauses fired — pooled OOS PF
// 2.275 on 66 trades over 36/12/6 with four profitable folds, 2.927 over the 24/12/3 control, 1.893
// at a doubled round trip, and ZERO unprofitable calendar years. On the full window the point draws
// 89 trades at PF 2.418 with a 5.36% max drawdown — against the core defaults' 139 trades, PF 1.054
// and 11.57%: profit factor up 2.3x while drawdown HALVED, which is the opposite of the wide-stop
// trap and rare in this map.
//
// This is the FIRST ticker here with the volume gate ARMED, and its window sits on the axis minimum:
// UseVolume=1 with VolLookbackBars=1, one bar compared against its slot baseline. The gate is what
// buys the edge — switching it off at the point costs 0.76 PF (2.418 -> 1.659) and raises drawdown
// (5.36% -> 6.80%) while adding 56 trades, i.e. the gate removes losers, not volume. Because the gate
// computes its baseline from daily turnover, a live-vs-backtest gap there would not degrade the
// ticker gradually, it would trade a different instrument.
//
// Two of the five literal fields deserve naming. EMASlow=50 won UNANIMOUSLY in all four folds of the
// trend theme — the only unanimous field of this calibration — and the point deliberately does NOT
// sit on the in-sample peak of that axis (the neighbour 40 measures 2.916 against 2.418), which is
// read as absence of fitting rather than an assembly error. StopDailyATR=0.7 came from risk gate A,
// NOT from a vote: the risk theme's majority was 1.5 (fold 1 even 2.0), where the stop stops firing
// altogether — 137 trades at every node from 0.7 to 2.0 while the SL share falls 13.1% -> 1.5% -> 0%
// and drawdown grows to 11.83%. The gate cost 0.373 PF on the full window and bought back a third of
// the drawdown. Worth remembering: the 1.5 node would have PASSED risk gate B (11.19% against the
// 11.57% ceiling), so only gate A separates this ticker from the trap.
//
// What the calibration did NOT find is equally part of the record. The two-lever pair (trend x
// volume) looked like the best prior in the catalogue for eight tickers — every one of fifteen probed
// pairs above 1.57 in-sample, the best 18x1.8 at 2.284/70 with half the baseline drawdown — and it
// did not survive walk-forward: pooled OOS 1.674 on 69 trades, below the trend_hump theme, three
// folds of four between 1.15 and 1.47, the whole result made by the fourth fold, and the pair's
// choice wandering across the axis (EMASlow 18/40/40/12, VolMult tied 2-2). The declared bar was
// therefore not taken: the trend theme took both criteria, the entry theme failed both (pooled OOS
// 0.973, RSILower 35/45/10/35).
//
// Risks accepted with eyes open. First, the volume gate is armed by a 3-of-4 majority from the screen
// theme whose own pooled OOS is 0.862, and no theme calibrated the gate's SHAPE (VolMult and
// VolBaseDays both stay at core defaults) — the gate's value rests on the plateau probe and the
// two-dimensional measurement, not on the walk-forward of the theme that measures it. Second,
// VolLookbackBars=1 is the physical floor of the axis, so no beyond-the-edge probe exists and the
// gate is maximally sensitive to a single volume spike. Third, per-fold OOS pools are 15-18 trades
// (the pool of 66 clears the threshold, the individual folds do not), and on the three-month scheme
// two folds degenerate to four trades each. Fourth, the instrument's regime is a fall with a reversal
// at the end: buy&hold -56.42% over the window, three rising half-years of seven, and the strongest
// of them is the last — the same half-year that carries fold 4. Fifth, the weekend session is thin
// (2.11 mln RUB median turnover, the thinnest among recent candidates) and seven of the point's 89
// exits land in it, a slippage risk the cost model does not cover. Sixth, the half-hour series starts
// 2023-09-11, one day AFTER the window opens: there is no history left to extend backwards, and the
// session widened once inside the window (29 -> 34 bars at the 2024H2 boundary).
//
// Review triggers. Median weekday turnover below 50 mln RUB over six months means re-check the
// ticker — today it is 128.99 mln and RISING on all four horizons, the healthiest liquidity profile
// of the recent intake. Any change to core's volume-gate semantics (slot baselines, weekday
// selection, the lookback window) is a change to this ticker's main mechanism, unlike anywhere else
// in this map. And if the daily-ATR stop ever reaches 1.0 here by any route, that is the trap, not an
// improvement — the measurement above says the stop simply stops existing.
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
	ydex.Ticker:  ydex.DefaultParams(),
	banep.Ticker: banep.DefaultParams(),
	astr.Ticker:  astr.DefaultParams(),
	sngsp.Ticker: sngsp.DefaultParams(),
	nkhp.Ticker:  nkhp.DefaultParams(),
	sofl.Ticker:  sofl.DefaultParams(),
	tgka.Ticker:  tgka.DefaultParams(),
	vsmo.Ticker:  vsmo.DefaultParams(),
	spbe.Ticker:  spbe.DefaultParams(),
	mvid.Ticker:  mvid.DefaultParams(),
	cnru.Ticker:  cnru.DefaultParams(),
	aqua.Ticker:  aqua.DefaultParams(),
	magn.Ticker:  magn.DefaultParams(),
	irkt.Ticker:  irkt.DefaultParams(),
	x5.Ticker:    x5.DefaultParams(),
	sfin.Ticker:  sfin.DefaultParams(),
	svcb.Ticker:  svcb.DefaultParams(),
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
