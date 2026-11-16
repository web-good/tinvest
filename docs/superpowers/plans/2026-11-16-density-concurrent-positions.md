# Density concurrent positions implementation plan

> **For agentic workers:** Use superpowers:executing-plans for implementation; focused notification and shared-account tasks may be delegated under caveman:cavecrew.

**Goal:** Density trades multiple tickers, shares an instrument with candle strategies and reports confirmed fills in Telegram.

**Architecture:** Persist independent episodes by UID in one account journal. Isolate strategy inventory and order identities in the account coordinator and candle runner. Deliver journaled notification events outside the trading loop.

**Tech Stack:** Go, existing gRPC adapters, Telegram bot and append-only journal.

**Spec:** docs/superpowers/specs/2026-11-16-density-concurrent-positions-design.md (approved by user: «делай так»).

## Global constraints

- One density episode and one entry lot per UID; existing detector, stop and sizing parameters remain.
- Preserve existing workspace changes, journal UUIDs and unresolved ownership.
- Only own quantities may be sold and own stops cancelled; unknown outcomes remain unresolved.
- No live orders during development. Telegram failures never delay protection.

## Review focus

- Entry/cancellation races must retain fills and notification events.
- One failing episode must leave other episodes under supervision.
- A candle stop filling while density remains must settle the candle slice.
- Restart during journal/ownership handoff must not duplicate accounting or orders.
- Lost Telegram responses permit duplicate messages, never duplicate trades.

### Task 1: Multiple density episodes

Files: `density/live/{model,engine,feed,shared,participant,store,binding}.go` and focused tests.

- [x] Add failing two-instrument and independent-loss tests.
- [x] Replace single execution context with persistent per-UID episodes; migrate v1 state.
- [x] Track feed loss and polling independently; supervise all episodes before new entries.
- [x] Recover and stop all persisted instruments, including removed tickers.
- [x] Run density live tests with race detector.

### Task 2: Shared ticker ownership

Files: `livecore/accountcoord/`, `rsi_pullback/live/`, density shared reconciliation hooks.

- [x] Add failing density + candle reservations on one ticker and own-quantity exit tests.
- [x] Introduce separate density and candle claims, retaining unambiguous identity and migration.
- [x] Reconcile candle inventory from own order fills; ignore known density orders during stop recovery.
- [x] Allow density entry/exit beside known candle inventory and preserve unrelated holdings/orders.
- [x] Verify both entry orders, partial executions, unknown results, stop fills and restarts.

### Task 3: Fill notifications

Files: new `density/live/notifications.go`, notification tests, `service_provider/density_live.go` and live lifecycle integration.

- [x] Add failing tests for confirmed BUY/SELL events and durable retry.
- [x] Persist notification events atomically with fill state and close accounting.
- [x] Add separate delivery worker, acknowledgement and retry without blocking execution.
- [x] Wire existing density Telegram chat/topic and validate recipient in live mode.
- [x] Verify delayed fills, cancelled unfilled entries and worker shutdown.

### Task 4: Integration and verification

- [x] Update `docs/orderbook_density/live.md` to describe the final mechanics and migration.
- [x] Run affected race tests and `./bin/mage ci`; report any pre-existing unrelated failures.
- [x] Review final changes for cross-owner sales, stop cancellation and state replay defects.

## Verification evidence

- `go test -race ./...`: PASS across the module.
- `golangci-lint run ./...` (through `mage ci`): 0 issues.
- Mockery regeneration is stable: SHA-256 comparison of all 54 mock files before/after a further `mage mocks` invocation found no changes.
- Raw `mage ci` stops at its `git diff --exit-code -- **/mocks/**` check because the intentionally updated executor mock is uncommitted; this gate compares against the Git index, not against a pre-generation snapshot. No repository index changes or commits were made to bypass it. The full race suite was therefore also run directly and passed.
- Independent reviews identified and regression tests fixed shutdown/recovery notification delivery, journal frame size limit, non-FVG TP suppression, unknown order/stop entry checks and unexplained flat-portfolio ownership cleanup.
- Shared partial BUYs persist cumulative executions, protect fills before cancellation and reconcile stop children before resizing. Broker failures retain existing confirmed protection and report an error.
