# Density Paper Trading Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Run a market-data-only forward test of density entries and report virtual results.

**Architecture:** Independent cmd/density-paper wires existing density market data to a serialized paper engine and durable JSONL/JSON reporting. Reuse the detector; no broker order interfaces or Telegram dependency in the command.

**Tech Stack:** Go 1.25, existing gRPC/protobuf and confita/godotenv dependencies, standard library reporting.

**Spec:** docs/superpowers/specs/2026-11-16-density-paper-design.md

## Global Constraints
- Never submit broker orders; real market data only.
- No touch fills, no fabricated execution during gaps, no resumption from old session files.
- At most ten instrument subscriptions to exchange trades.
- Existing unrelated work remains untouched; use clean linked tinvest-density worktree.

## Review Focus
- Latency, stale/out-of-order data and post-gap execution: test rejection and uncertainty reporting.
- Partial fills and insufficient depth: test capacity, residual inventory and commissions.
- Cash reservation across instruments: test no over-allocation or negative free cash.
- File errors/shutdown: stop on journal failure and retain unclosed inventory in final summary.
- Wrong tick, zero/NaN settings: reject before connecting; test flag/config validation.

### Task 1: Market metadata and deterministic simulator
**Files:** density/universe.go and stream.go, grpc/density_market_data.go and density_stream.go; new internal/service/screener/density/paper/{config,engine,execution,report}.go and tests.
**Interfaces:** Engine.OnBook(Book,time.Time), OnTrade(Event,time.Time), Observe(Signal,time.Time), Tick(time.Time), Gap(string,time.Time), Summary(time.Time). Prices in nanorubles internally; input uses density model.
- [x] Add failing metadata and engine tests: tick/timestamp propagation; no touch/pre-latency fills; density loss; partial fills; 2R exits; stop depth/fees; capital limits; invalid books/config; gaps.
- [x] Run go test on new package and adapter and observe failure.
- [x] Implement validated config, state machine and accounting; append every state change through injected journal writer.
- [x] Run focused tests with -race and confirm pass.

### Task 2: Standalone runner and persistent reports
**Files:** new paper/{runner,journal}.go, cmd/density-paper/main.go, corresponding tests and docs/orderbook_density/paper.md.
**Interfaces:** Run(context.Context, density.StreamSource, []density.Instrument, *Engine, density.Config) error; Journal.Record, WriteSummary. CLI accepts env-file/token environment, detection env and paper flags.
- [x] Add failing runner/journal tests: ack/status gates, reconnect gap, cancellation and final reporting, atomic summary and write failure.
- [x] Implement bounded live subscriptions, receive goroutine with cancellation, periodic timeout checks, reconnect and reports. Persist startup config, session events and final summary.
- [x] Document run command, tunables, report interpretation and model limitations.
- [x] Verify go test -race ./internal/service/screener/density/... ./pkg/client/grpc ./cmd/density-paper; build command, run --help and offline integration smoke test. Review whole diff; do not start a real daemon or create orders.

## Execution ledger
Native execution authorized by user's approval to implement the separate simulator. No extra approval round for the same work. Existing linked worktree is clean; parent checkout contains unrelated edits and is not used for implementation.

Task 1: complete. Tests first failed for missing engine and metadata; engine/adapter focused tests then passed.
Task 2: implemented. Runner/journal and command tests first failed for missing APIs, then passed. Final validation below.
Ruling: Runner struct rather than free Run function owns callbacks and retry settings, avoiding global report state.
Ruling: expected cancellation normalized before joining persistence errors; pre-gap book watermarks retained independently; depth sweep stops immediately on journal failure. Each has a failing-then-passing regression test.
Ruling: unknown valuations use null financial estimates; realized PnL remains readable. Initial CLI defaults are explicit hypotheses, not calibration. Pending cancellation and partial-entry remainder cancellation are immediate in v1 and documented as a potential optimistic bias; order activation and market exit latency are modeled.

Final verification (2026-11-16):
- Baseline density + gRPC tests passed before implementation.
- `go test -race -count=1 ./internal/service/screener/density/... ./pkg/client/grpc ./cmd/density-paper` passed (density 21s, paper/adapter/command about 1s each).
- Final `go test -race ./internal/service/screener/density/paper ./pkg/client/grpc ./cmd/density-paper` passed after lint fixes.
- `go vet` for affected packages and `git diff --check` passed.
- `./bin/golangci-lint run ./internal/service/screener/density/... ./pkg/client/grpc/... ./cmd/density-paper/...`: 0 issues.
- `go build ./internal/... ./pkg/... ./cmd/...` and standalone build/help passed.
- Offline journal integration verifies a completed two-lot round trip and RUB 11.584 net PnL at 0.1% per-side test commission.
- Fresh-context review found three important issues; regression tests reproduced and fixes resolved each: duplicate pre-gap execution, continued sweep after journal failure, and cancellation masking write failures.
- Two bounded real API smoke attempts with existing local credentials failed at Shares listing with DeadlineExceeded (25s and 60s request limit). No live session, actual trade or profitability result is claimed. HTTPS endpoint itself responded. No daemon left running, no order API used, no secrets printed.
- Changes left uncommitted in existing isolated feat/orderbook-density checkout; unrelated parent checkout unchanged.


Startup follow-up (2026-11-16):
- Instrumented read-only diagnostic compared ShareBy and Shares on one connection. Default Go 1.25 key exchange stalled in TLS authentication before any response headers, both methods failed in ~20s. With GODEBUG=tlsmlkem=0, ShareBy succeeded in ~7s; bulk Shares received headers but exceeded the bounded diagnostic deadline.
- Added opt-in `-tls-classic` (X25519/P-256, certificate verification and minimum TLS 1.2 retained) and targeted TQBR ShareBy lookup for explicit `-tickers`, avoiding the unnecessary bulk listing.
- Failing tests verified missing targeted lookup and TLS configuration, then passed after implementation. Added routing test that verifies explicit tickers never call bulk Shares.
- Live command with T,SBER and `-tls-classic` loaded both instruments, created a session in /tmp/tinvest-density-paper-smoke and emitted summary. Temporary diagnostic code removed. No broker order API involved.
