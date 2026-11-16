# RSI_PULLBACK TP comparison Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement task-by-task.

**Goal:** Compare delayed and touch TP fills on current RSI_PULLBACK literals and cached market candles.
**Architecture:** Keep strategy signals intact; share TP pricing between Run and Trace. Expose execution assumptions in CLI and reports; provide a repeatable offline comparison.
**Tech Stack:** Go, existing JSON candle cache and backtest reports.
**Spec:** docs/superpowers/specs/2026-11-15-rsi-pullback-take-profit-design.md

## Global Constraints

RSI_PULLBACK only; no live or RSI_ZONE changes. Zero settings preserve historical fills. No recalibration. Report approximation and data limitations explicitly.

## Review Focus

Invalid/NaN options must fail before I/O; exact fills and gap fills preserve defaults; non-TP exits stay unchanged; Trace and Run agree; incomplete cache coverage is reported.

### Task 1: Shared TP pricing

Files: internal/domain/backtest/{types.go,engine.go,tp_execution.go,tp_execution_test.go}.
Interface: Config.TPExecution TPExecution; TPExecution{Mode string, SlippageBPS float64}; Validate(strategyName string) error; private price(signal, candle) float64.
- [x] Write table tests: 105 target/106 high/102 close; gap open 107; exact target; 10 bps; zero target; commissions; non-TP isolation; Trace/Run consistency.
- [x] Observe failures with go test ./internal/domain/backtest.
- [x] Implement shared pricing and validation; invoke it in Run and Trace without altering signal priority.
- [x] Run domain tests and existing RSI_PULLBACK core tests.

### Task 2: CLI and report assumptions

Files: cmd/backtest/{main.go,tp_execution_test.go}; internal/domain/backtest/{types.go,report.go}.
- [x] Test invalid mode, negative/NaN/infinite/out-of-range bps, foreign strategy; report assumptions.
- [x] Add -tp-fill touch|bar-close and -tp-slippage-bps; validate before run's I/O; pass optional TPExecution to run preserving existing callers.
- [x] Preserve assumptions in report Meta, independently of strategy Params.
- [x] Run go test ./cmd/backtest ./internal/domain/backtest.

### Task 3: Offline experiment

Files: cmd/tpcompare/{main.go,main_test.go}; reports/rsi_pullback_tp/ (ignored artifacts).
Interface: offline CLI reads data/candles, live registry literals, explicit common end date, 24 months; writes per-ticker mode/scenario reports and aggregate comparison.
- [x] Test input window filtering and reject unknown tickers/missing daily history.
- [x] Build offline runner with current live ParamsFor and mode/slippage scenarios 0/5/10 bps; record lot-size assumption, coverage and input hashes.
- [x] Run fixed-window experiment, inspect outputs, summarize per-ticker improvements/regressions without treating independent ticker accounts as portfolio returns.
- [x] Run focused race tests, lint and a fresh-context code review; address material findings.

## Execution ledger

Baseline: go test ./internal/domain/backtest ./cmd/backtest passed before edits.
Ruling: implement inline to avoid extra approval rounds after explicit design approval. Use separate worktree /tmp/tinvest-tp-comparison; leave original live configuration intact.

Completion: focused RED/GREEN tests observed; go test -race for affected packages passed; ./bin/mage ci passed including full lint, race suite and mock drift; fresh reviewer found no remaining Important/Critical issues. Daily warmup rejection added after review; experiment repeated. No live changes.
