#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../../../.."
ticker="${1:-LKOH}"
pause_seconds="${PAUSE_SECONDS:-60}"
for branch in stoch_exit no_stoch_exit; do
  if [[ "$branch" == no_stoch_exit ]]; then
    echo "Пауза ${pause_seconds} секунд перед второй веткой..."
    sleep "$pause_seconds"
  fi
  grid=data/params/fvg_imbalance/gazp/grid.json
  if [[ "$branch" == no_stoch_exit ]]; then
    grid=data/params/fvg_imbalance/gazp/grid_no_stoch_exit.json
  fi
  go run ./cmd/backtest \
    -ticker "$ticker" -strategy fvg_imbalance -interval Minutes30 \
    -calibrate "$grid" \
    -out "reports/$ticker/fvg_imbalance/$branch" \
    -months 18 -holdout-months 3 -train-months 9 -test-months 3 \
    -metric profit_factor -min-trades 9 -cpu 2
done
