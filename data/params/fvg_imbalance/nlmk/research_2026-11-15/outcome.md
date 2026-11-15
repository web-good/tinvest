> Update: the user subsequently requested installation of this candidate despite the failed final validation. The literal now contains these parameters; the historical decision below records the original calibration outcome.

# NLMK — calibration rejected by final-year validation

The three-stage search completed, but the locked candidate fails the unchanged final-year quality criteria. NLMK Params() remains core.DefaultParams(); no strategy mechanics, registry, other tickers, or live workers changed. No second candidate was tried after the final year.

## Full locked candidate

```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.25,
  "MaxWaitBars": 45,
  "StochPeriod": 11,
  "UseStochExit": 0,
  "StopATR": 0.5,
  "TakeProfitATR": 1.5,
  "HTFTrendEMA": 0,
  "RequireEntryAboveFVGLower": 1
}
```

UseStochExit=0: stochastic remains part of ENTRY, exits use stop0.5 and target1.5 in daily ATR. Selection metric expectancy.

## Comparable full-engine runs

| Window / parameters | Closed trades | Net PnL RUB after costs | PF | Engine DD % | Expectancy RUB | Open |
|---|---|---|---|---|---|---|
| Train 2023-11-15/2025-11-15 — candidate | 28 | 32539.69 | 2.09404 | 7.73404 | 1162.13179 | False |
| Train — original | 29 | 11180.77 | 2.59822 | 4.14759 | 385.54395 | False |
| Final 2025-11-15/2026-11-15 — candidate | 20 | -1570.62 | 0.93160 | 15.48284 | -78.53105 | False |
| Final — original | 23 | -5832.86 | 0.65142 | 10.78180 | -253.60276 | False |

## Train annual entry-date cohorts

| Entry-date window | Candidate closed trades | Candidate closed net RUB | Candidate PF | Original closed trades | Original closed net RUB | Original PF |
|---|---|---|---|---|---|---|
| 2023-11-15/2024-11-15 | 18 | 19461.68 | 2.24823 | 18 | 2410.70 | 1.46503 |
| 2024-11-15/2025-11-15 | 10 | 13078.01 | 1.92415 | 11 | 8770.07 | 5.84044 |

Annual cohorts partition CLOSED trades from the same uninterrupted 24-month replay by entry time. These are training diagnostics, not independent annual reset runs. Annual Net is closed-cohort PnL. Full-run Net includes any open MTM (none for candidate/original in these runs). No cohort annual DD is inferred.

## Final decision

Candidate passes count (20 >=10) but fails Net>0, PF>1, DD<=10%. Versus original, final Net improves by 4262.24 RUB and PF improves, while drawdown worsens by 4.70 percentage points and trades drop23 to20. Final expectancy remains negative. Both finish flat. No average R calculated by engine; it is not invented.

## Search and stability

Entry350/350 (26 admissible), exits160/160, local36/36 (6 admissible), sensitivities11/11, bounded repeat36/36 (6 admissible). 593 grid replays plus original/baseline controls, then only locked candidate and original on final. Branch winner no-stoch had >5% expectancy lead at stage2. The single permitted repeat adopted MaxWaitBars45. No unexamined saved rows or keepTop pruning. Search is bounded, not global; StopATR0.5 is lower grid edge, next stop1 fails eligibility, second-year candidate count only10.

## History and execution

Exact fixed MSK windows from plan, completed Tinkoff candles from provider cache. Train snapshot:23838 M30 bars (2023-11-15 09:30 /2025-11-14 23:30);820 daily including warmup (2022-11-15 /2025-11-14). Final snapshot:13576 M30 bars (2025-11-15 02:00 /2026-11-14 23:30);644 daily including warmup (2024-11-15 /2026-11-14). Final daily November14 is Saturday and ignored by ATR; daily visibility follows production cursor. Calendar end is exclusive. Hashes/bounds in snapshot/manifest.json and final-snapshot/manifest.json.

Initial ordinary CLI failed instrument metadata refresh (unexpected EOF). After candidate lock, direct CandleProvider top-up using cached NLMK UID succeeded for both intervals with no logged chunk errors; history-topup.log retained. Cached lot=10, RUB100000, fraction1, risk-pct0, commission0.0005 per turnover each side. M30 next-open entry, production stop gap rule, exact target, completed daily ATR/EMA only, no forced liquidation.

Research adapter replay.go calls the unchanged production Binding/Run/Compute/report renderers. Final runs used full-object -params (no -calibrate) with fresh portfolios. Fixed-window adapter replaces wall-clock CLI so the reserved window is exact; no claim that moving-now CLI timestamps are identical. It consumes only the respective frozen snapshots. All grid rows retain full trade tables, exact cohort counts, OpenPosition/OpenBars. Stop-only rows are tabulated separately in stop-only-positions.md; residual open M30 bars measure holding duration and are not counted as closed trades.

## Reproduction

From repository root:

```bash
go run data/params/fvg_imbalance/nlmk/research_2026-11-15/replay.go \
  -final -snapshot reports/fvg_imbalance/NLMK/research_2026-11-15/final-snapshot \
  -params data/params/fvg_imbalance/nlmk/research_2026-11-15/final-candidate.json \
  -out /tmp/nlmk-final-reproduction
```

## Artifacts

- plan.md, original.json, baseline.json, stage-1-entry.json, stage-1-selected.json
- stage-2-candidate-{1..4}-{stoch,no-stoch}.json and branch selection notes
- stage-3-refinement.json, sensitivity-*.json, stage-3-repeat.json
- locked-candidate.md, final-candidate.json, locked-training-result.json, rejected-candidate.json
- Reports under reports/fvg_imbalance/NLMK/research_2026-11-15/: results.json and calibration.md per grid; final-candidate/report.md and final-original/report.md plus trades/equity CSVs; stop-only-positions.md. Reports/candle snapshots are gitignored locally, not lost or deleted.

## Verification

All593 grid rows checked against full closed trade tables for fixed annual counts, summed closed PnL and eligibility. Snapshot SHA256 verified. gofmt clean. Requested `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/... ./internal/service/backtest ./cmd/backtest` exits0. NLMK literal compared byte-for-byte with original-source.txt and unchanged.
