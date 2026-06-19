# Reversion-fitness screener (`-volrank` v2)

**Date:** 2026-06-19
**Status:** approved (design), pending implementation plan
**Supersedes ranking logic of:** `2026-06-18-volatility-atr-screener-design.md`

## Problem

The current `-volrank` screener ranks the liquid RUB universe by **mean daily ATR%
alone**, using turnover only as a hard `-min-turnover` floor. Volume plays no part in
ranking, and there is no signal distinguishing *mean-reverting* volatility (good for the
reversion strategy) from *trending* volatility (a falling knife).

This gap is not theoretical: SFIN topped no volatility screen but was added and calibrated
on the reversion strategy, and failed — it was volatile but **trended down hard**, so the
oversold bounce the strategy bets on never came (pooled OOS PF 0.43). UGLD, which
oscillates, worked (PF 3.39). A screener meant to surface *reversion* candidates must
reward mean reversion, not raw volatility.

## Goal

Rank the universe by a composite **reversion-fitness score** combining three dimensions,
so the screener surfaces liquid stocks where **price moves well, volume is healthy, and
price mean-reverts** rather than trends.

## Metrics (per ticker, from the daily candle slice already fetched)

| Dimension | Metric | Source | Better when |
|---|---|---|---|
| Volatility | mean ATR% over the window | existing `VolMetrics` | higher |
| Liquidity | mean daily turnover (M₽) | existing `VolMetrics` | higher |
| Mean reversion | VR(2) — Lo-MacKinlay variance ratio | reuse `domain.VarianceRatio(rets, 2)` | **lower (<1)** |
| (informational) | Autocorr(1) | reuse `domain.Autocorr1(rets)` | more negative |
| (informational) | verdict | reuse `domain.MeanReversionVerdict(vr2)` | mean-reverting |

Returns come from `domain.SimpleReturns(closes)`; `VolMetrics` already builds `closes`.

## Composite score

A literal "volume ÷ volatility" ratio is rejected: it would *penalise* volatility, while
the user needs **both** high. Raw products are also meaningless across incomparable units
(ATR% ~1–5, turnover ~10–5000 M₽). Instead use **percentile-rank blending**, which is
unit-agnostic and robust to turnover's extreme right-skew (a few mega-caps would otherwise
dominate a z-score blend):

```
score = wVol·pct(ATR%) + wRev·pct(reversion) + wLiq·pct(turnover)
pct(x)        = fractional rank of x within the passed set, in [0,1]
pct(reversion)= fractional rank of (-VR2)  // lower VR2 ⇒ higher percentile
```

Default weights: `wVol=0.4, wRev=0.4, wLiq=0.2` (volatility and mean reversion are the
core of reversion fitness; liquidity is mostly a gate). Weights are CLI flags. The hard
`-min-turnover` floor stays — it decides "tradeable at all" before scoring.

## Hard trend-exclusion gate

Trending tickers are **excluded from the report entirely**, not merely down-ranked: a row
is dropped when `VR2 > maxVR` (default `1.05`, the existing `MeanReversionVerdict`
"trending" boundary) or when `VR2 <= 0` (undefined — too little history). This guarantees
the output contains only mean-reverting/neutral candidates. The threshold is a flag
(`-max-vr`, default 1.05). The gate runs alongside the liquidity/history filter, **before**
scoring, so percentiles are computed only over surviving (non-trending) rows.

Percentiles are computed over the **passed** set (rows that cleared the
liquidity/history/trend filter), so the score is relative to the screened universe.

## Report

Rank by `score` descending. Columns: `# | Тикер | Название | Score | Ср.ATR% | Тек.ATR% |
Тренд | VR(2) | Autocorr | Вердикт | Оборот М₽/день | Баров`. The header documents the
weights used and the formula.

## Architecture

- **`VolRow`** (`internal/service/backtest/volatility_screen.go`): add `VR2`, `Autocorr1`,
  `Verdict`, `Score` fields.
- **`VolMetrics`**: also return `vr2, autocorr1` computed from `closes` via the existing
  `domain` helpers. (Single responsibility kept: it computes per-ticker metrics; it does
  not score.)
- **`ScoreVolRows(rows []VolRow, wVol, wRev, wLiq float64)`**: new pure function. Needs the
  full passed set to compute percentiles; fills `Score` on each row. Lives beside the
  render code. Unit-testable in isolation.
- **`RenderVolatilityMarkdown`**: sort by `Score`; render the new columns; document weights.
- **`runVolRank`** (`cmd/backtest/main.go`): thread the weight flags + `-max-vr`, apply the
  trend gate while collecting rows, call `ScoreVolRows` after collecting, before render.
- **`VolMeta`**: carry the three weights + `maxVR` + a dropped-as-trending count for the header.

## Flags

- `-w-vol` (default 0.4), `-w-rev` (default 0.4), `-w-liq` (default 0.2) — composite weights.
- `-max-vr` (default 1.05) — hard trend-exclusion threshold on VR(2).
- Existing `-min-turnover`, `-atr-period`, `-top`, `-months` unchanged.

## Testing (`volatility_screen_test.go`)

- `VolMetrics` returns VR2<1 / negative autocorr for a synthetic oscillating series and
  VR2>1 for a synthetic trending series.
- `ScoreVolRows`: a deeper mean-reverting + volatile + liquid row outranks a weaker one;
  percentile blend is order-correct; weight changes shift order as expected; single-row and
  empty-input edge cases.
- Trend gate: rows with `VR2 > maxVR` or `VR2 <= 0` are excluded; a borderline `VR2 == maxVR`
  is kept; the dropped-as-trending count is reported.

## Out of scope (YAGNI)

- `-screen` mode (manual variance-ratio screener) is left untouched.
- No separate "volume activity / consistency" metric — mean turnover covers "volume moves
  well" for now.
- Trading engine and reversion strategy unchanged. The screener only proposes candidates;
  each still needs the staged walk-forward before going live.
