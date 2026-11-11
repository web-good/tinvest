#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../../../.."
go run ./cmd/backtest -ticker LKOH -strategy fvg_imbalance -interval Minutes30 \
  -params data/params/fvg_imbalance/lkoh/baseline.json \
  -out reports/LKOH/fvg_imbalance/baseline -months 18 -holdout-months 3 \
  -commission 0.0005 -fraction 1 -risk-pct 0 -cpu 2
go run ./cmd/backtest -ticker LKOH -strategy fvg_imbalance -interval Minutes30 \
  -calibrate data/params/fvg_imbalance/lkoh/ordered_grid.json \
  -out reports/LKOH/fvg_imbalance/ordered_wf -months 18 -holdout-months 3 \
  -train-months 9 -test-months 3 -metric expectancy -min-trades 9 \
  -commission 0.0005 -fraction 1 -risk-pct 0 -cpu 2
