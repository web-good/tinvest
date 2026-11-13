# RNFT calibration outcome

## Decision

The locked candidate failed the reserved final-year quality gates. Do not write it to `strategy/rnft/rnft.go`; the ticker literal remains `core.DefaultParams()`.

## Locked candidate

Full nine-field reproduction object: `final-candidate.json`. It was locked before final-year inspection in `locked-candidate.md`.

Training 2023-11-12 22:32–2025-11-12 22:32: 37 closed trades (21 in 2023-11-12–2024-11-12; 16 in 2024-11-12–2025-11-12), net +7,437.07 RUB, PF 1.509, Max DD 5.98%, no open position. Selected branch: `UseStochExit=1`.

Final 2025-11-12 22:33–2026-11-12 22:33: 24 trades, net -1,882.39 RUB, PF 0.896, Max DD 10.04%, no open position. Trade-count floor passed, but net PnL, PF and drawdown gates failed.

## Original parameter comparison

Final-year original full set (`original.json`): 202 trades, net -11,179.94 RUB, PF 0.917, Max DD 42.01%, no open position. The candidate lost less in absolute terms, but did not meet the acceptance gates; this does not justify deployment.

## Reproduction reports

- Training control, original set: `reports/fvg_imbalance/RNFT/2026-11/control/RNFT_fvg_imbalance_Minutes30_20261112_222908.md`
- Stage 1 (all 350 rows): `reports/fvg_imbalance/RNFT/2026-11/stage-1/RNFT_fvg_imbalance_Minutes30_20261112_222922_calibration.md`
- Candidate annual and drawdown checks: `reports/fvg_imbalance/RNFT/2026-11/candidate-check-12/`, `candidate-check-13/`, `candidate-check-14/`, `candidate-check-15/`
- Stage 2, stochastic branch: `reports/fvg_imbalance/RNFT/2026-11/stage-2-candidate-14-stoch/`
- Stage 2, no-stochastic branch: `reports/fvg_imbalance/RNFT/2026-11/stage-2-candidate-14-no-stoch/`
- Stage-2 branch check runs: `reports/fvg_imbalance/RNFT/2026-11/stage-2-best-stoch/`, `stage-2-best-no-stoch/`
- Stage 3 joint and sensitivities: `reports/fvg_imbalance/RNFT/2026-11/stage-3-joint/`, `sensitivity-stochperiod/`, `sensitivity-htftrendema/`, `sensitivity-maxwaitbars/`
- Full candidate training rerun: `reports/fvg_imbalance/RNFT/2026-11/stage-3-finalist-training/`
- Reserved final runs: `reports/fvg_imbalance/RNFT/2026-11/final-candidate/`, `final-original/`
