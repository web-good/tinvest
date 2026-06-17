# Reversion strategy: risk sizing, regime gate, exit asymmetry, screening

**Date:** 2026-06-17
**Branch:** feat/reversion-rsi-dip
**Status:** design approved, pending implementation plan

## Motivation

Walk-forward (expectancy ranking, 12mo train / 6mo OOS, Hour1) across 8 tickers
showed the reversion strategy is, as a basket, near break-even with a few
positives (NVTK PF 2.40, GAZP 1.18, PLZL 1.16) and several losers (AFKS 0.73,
MDMG 0.73). Diagnosis from the reports + `core.go`:

1. **No protective price stop** when `UseATRStop=0` (the NVTK default). A losing
   long rides down to a bearish EMA cross (EMAX) or a failed-bounce RSIOS. Tail
   losses (PLZL worst trade −5419 vs best +4006) erase many small wins.
2. **All-in fixed-fraction sizing** (`Fraction=1.0`): the tail hits full size on
   the most volatile names.
3. **No regime discrimination**: mean-reversion-long gets run over in trending /
   down regimes (fold 5, 2H2025, was negative for almost every ticker). The EMA
   trend filter gates up-vs-down, not range-vs-trend.
4. **Instrument selection by P&L** invites survivorship bias on ~33–69 OOS trades.
5. **Exit asymmetry**: `RSI50` cross-down fires early and caps winners while
   losers run far — "cut winners, let losers run."

Timeframe is **Hour1 by design** (confirmed) — not in scope to change.

## Scope

**Backtest research only.** Live trading does not persist `EntryATR`, so all
ATR-based mechanics (catastrophic stop, risk sizing, trailing) are inert live and
are deliberately NOT wired into live position state in this work. A later effort
can persist entry-time ATR if/when an edge is confirmed.

All new behaviour is **opt-in**; defaults preserve current behaviour and keep
existing tests green.

## Components

### A. Risk-based sizing + catastrophic stop (point 2)

A single config flag turns on the whole block. The catastrophic stop is the
sizing anchor: stop distance determines position size so each trade risks a
fixed fraction of equity.

**New strategy param (`core.Params`):**
- `CatStopATRMult float64` — catastrophic stop distance in EntryATR multiples
  (e.g. 2.5). Stop price = `EntryPrice − CatStopATRMult × EntryATR`.
  Always active when risk-sizing is on; calibration cannot disable it (protective
  logic must not be a toggle).

**New engine config (`domain.Config`) + CLI flag:**
- `RiskFractionPct float64` (CLI `-risk-pct`, default `0`). `0` = legacy
  fixed-`Fraction` sizing (full backward compatibility). `>0` activates
  risk-based sizing AND the catastrophic stop exit.

**Signal carries the stop:**
- `model.Signal.StopPrice float64`, set by the core on a `Buy` to
  `EntryPrice − CatStopATRMult × EntryATR`. `0` when risk-sizing is off.

**Engine sizing (`portfolio.open`):**
- When `RiskFractionPct>0` and `sig.StopPrice>0`:
  `riskCapital = RiskFractionPct/100 × equity`;
  `perShareRisk = EntryPrice − StopPrice`;
  `targetShares = riskCapital / perShareRisk`;
  `lots = floor(targetShares / Lot)`; cost capped by available cash (fall back to
  the cash-affordable lot count if the risk-target exceeds cash).
- When `RiskFractionPct==0`: unchanged `Fraction × cash` path.

**Exit (`manage`):** new `CatSL` exit — sell when `price ≤ StopPrice`
(`= PurchasePrice − CatStopATRMult × EntryATR`, EntryATR frozen at entry). Active
only when risk-sizing is on.

**Coexistence with the existing optional `UseATRStop` (ATRSL):** both stops are
kept. CatSL is the WIDE always-on backstop and sizing anchor (~2.5 ATR); the
existing ATRSL is the TIGHT optional middle exit (~0.8–1.2 ATR) and fires first
when enabled. Sizing always anchors to CatSL (always defined), independent of
`UseATRStop`.

**ATR availability:** `buildInput` must compute `atr` whenever risk-sizing is on
(currently only when `UseATRStop==1 || UseBreakeven==1`). Extend that condition.

### B. ADX regime gate (point 3)

Trade reversion only in range / mean-reverting regimes; stand aside in trends.

**New indicator:** `pkg/indicators` ADX (Wilder's DM/ATR smoothing). Pure,
table-tested. Inputs `highs, lows, closes, period` → ADX series.

**New strategy params:**
- `UseRegime int` (default `0`).
- `ADXPeriod int` (e.g. 14).
- `ADXMax float64` — enter only when `ADX < ADXMax`.

**Gate (`decide`):** after the trend filter, before the dual-oversold check: when
`UseRegime==1`, block the entry unless ADX is warmed AND `ADX < ADXMax`. An
un-warmed ADX blocks (same protective discipline as the HTF gate). `Lookback`
gains a `~2×ADXPeriod` candidate. `buildInput` computes ADX only when
`UseRegime==1`. `Explain` reports the gate's value/verdict.

### C. ATR trailing stop + optional RSI50 (point 5)

Let winners run once the catastrophic stop caps the downside.

**New strategy params:**
- `UseTrail int` (default `0`), `TrailATRMult float64` — trail distance in
  EntryATR multiples below the running max.
- `UseRSI50 int` (default `1` — preserves current always-on behaviour).

**Exits (`manage`):**
- New `TRAIL` exit: when `UseTrail==1` and `EntryATR>0`, sell when
  `price ≤ MaxFavorablePrice − TrailATRMult × EntryATR` (reuses the monotonic
  `Position.MaxFavorablePrice` already maintained for breakeven).
- Existing `RSI50` case wrapped in `if UseRSI50==1`.

**New exit precedence:**
`OB → CatSL → TRAIL → RSI50(opt) → BE(opt) → middle RSIOS/ATRSL(opt) → EMAX`.

### D. Variance-ratio screener (point 4, independent module)

Pre-screen the universe by a measurable mean-reversion property instead of by
realized P&L (avoids survivorship bias in instrument selection).

**Pure stat:** `internal/domain/backtest/variance_ratio.go`:
`VR(returns, q) = Var(q-bar returns) / (q × Var(1-bar returns))` for
`q ∈ {2, 4, 8}`, plus lag-1 autocorrelation. `VR<1` = mean-reverting, `VR>1` =
trending. Pure, table-tested, no I/O.

**Mode:** `-screen <tickers csv>` flag in `cmd/backtest`, reusing the existing
gRPC client and `CandleProvider` (no duplicated wiring). Output: a ranked
Markdown/console table — ticker, VR(2/4/8), lag-1 autocorr, verdict
(mean-reverting / trending / neutral). `-screen` is mutually exclusive with
`-calibrate` / `-basket` / `-explain`.

## Calibration impact

New knobs expand the parameter space, but the trimmed-grid philosophy holds:
**sweep only `CatStopATRMult`, `ADXMax`, `TrailATRMult`; fix the rest** at
sensible defaults. `RiskFractionPct` is a risk-budget choice (a CLI flag), not an
edge parameter — never swept. Per-ticker grids stay small so params can stabilize
across folds.

## Testing

Table-driven unit tests mirroring `core_test.go`:
- `pkg/indicators` ADX (known-series fixtures).
- `variance_ratio.go` (synthetic mean-reverting vs trending series).
- `portfolio.open` risk-based sizing (tight vs wide stop → share count; cash cap).
- `manage` exits: `CatSL`, `TRAIL`, `RSI50` gated off, precedence ordering.
- `decide` regime gate: blocks when `ADX ≥ ADXMax` / un-warmed; passes when below.
- Backward-compat: `RiskFractionPct=0` path and existing defaults unchanged.

Validation (user-run): re-run walk-forward on NVTK / GAZP / PLZL with the new
mechanics on a trimmed grid; compare pooled OOS and per-fold consistency +
parameter stability vs the 2026-06-17 expectancy baseline.

## Backward compatibility

Defaults: `RiskFractionPct=0`, `UseRegime=0`, `UseTrail=0`, `UseRSI50=1`; CatSL
active only when risk-sizing is on. All current tests and report runs are
unaffected; every new mechanic is opt-in.

## Out of scope

- Live-trading wiring of ATR-based mechanics (no `EntryATR`/regime persistence).
- Changing the Hour1 timeframe.
- Unrelated refactors of the engine or report renderers.
