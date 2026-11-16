# RSI broker take-profit implementation plan

> **For agentic workers:** Execute inline using superpowers:executing-plans; preserve the user's current feature branch and workspace.

**Goal:** Add broker TP to RSI Pullback and RSI Zone without removing existing exits.
**Architecture:** Keep separate RSI adapter placement methods, reuse account-level paired-order reconciliation, and add Zone's frozen ATR target.
**Tech Stack:** Go, existing Invest gRPC clients and testify mocks.
**Spec:** `docs/superpowers/specs/2026-11-16-rsi-broker-take-profit-design.md`.

## Global constraints

- Zone live literals use TPDailyATR 1.5; Pullback literals retain current values.
- Keep existing exits, state JSON compatibility, dry-run, unrelated strategies and FVG behavior.
- No deployment, broker calls, commits or branch switches are needed.

## Review focus

- Existing Zone state without a target must gain a frozen target.
- A triggered TP with a stale portfolio must never lead to another sale.
- Failure to cancel a sibling must preserve durable state and block another sale.
- Unknown submission results and partial fills must retain their reconciliation keys.
- Broker tick rounding and actual entry fills must determine the placed prices.

### Task 1: Zone target and reconstruction

- [x] Add failing core tests for target formula, wick-only TP, disabled TP and SL precedence.
- [x] Add TPDailyATR to Zone Params, populate entry target/distance and add TP exit.
- [x] Set live ticker literals to 1.5; keep historical explicit zero-TP fixtures valid.
- [x] Test and implement reconstruction from entry price/ATR and old-state target initialization.
- [x] Run Zone core/live and backtest tests.

### Task 2: Paired broker orders for RSI

- [x] Add failing runner tests for entry TP placement and cancellation before indicator exit.
- [x] Route RSI slots through existing paired reconciliation, TP placement and durable submission handling.
- [x] Cover restart, external cancellation, stale portfolio, partial fills and cancel failures.
- [x] Preserve legacy state fields and unrelated strategies; run runner/livecore tests.

### Task 3: Documentation and verification

- [x] Update strategy/live mechanics for both RSI strategies.
- [x] Review the diff and run `./bin/mage ci`; report actual results and material limits.

## Execution notes

- Implemented in the existing feature branch, without commits or deployment.
- Added `adapter.BrokerTakeProfit`; placement methods live in each strategy package.
- Existing ticker grids/reports remain historical; Zone literals explicitly enable 1.5 ATR.
- Independent review found an unfinished BUY path outside shared mode; added a failing
  regression and fixed durable reconciliation before TP placement. Reviewer rechecked the fix.
- Focused suites pass. Final verification: `./bin/mage ci` (lint, mock drift, race tests).
