# Density Simulation Alerts Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans. User authorized inline implementation and chose alerts at virtual fill.

**Goal:** Run paper density through the main application when real trading is off.
**Architecture:** Reuse paper engine/runner; add a simulation service and durable asynchronous notification outbox; branch dependency wiring before real account creation.
**Tech Stack:** Go, existing density market data, paper engine, Telegram.
**Spec:** docs/superpowers/specs/2026-11-16-density-simulation-alerts-design.md

## Global Constraints
- Preserve existing dirty work; no commits, restarts or real orders.
- Simulate only, one lot, same detector and ticker allowlist as live.

## Review Focus
- Entry placement is not an execution alert.
- Real broker/account methods must not be constructed in simulation.
- Slow/failed Telegram cannot block price processing; messages survive restart.
- Mode switch must not conceal unfinished real episodes.
- Price-only alerts do not depend on any balance, size, fee or artificial latency settings.

### Task 1: Simulation execution and delivery
**Files:** paper/config.go, paper/engine.go; simulation/{service,outbox,notification}.go and tests; live/participant.go, live/binding.go and tests.
**Interfaces:** Participant.RunPaper callback; paper.Config.MaxEntryLots; simulation.Service.Run; Outbox.Record/Run; FormatFill.
- [x] Add failing tests for one-lot fills, fill-only notification, durable retry, mode routing and real state guard.
- [x] Implement simulation service and asynchronous outbox using market-data interfaces only.
- [x] Run go test -race ./internal/service/screener/density/...

### Task 2: Wiring, configuration and verification
**Files:** config/density_live*, service_provider/density_live*, env/prod.env, docs/orderbook_density/{live,paper}.md.
- [x] Add failing mode wiring and no-real-account tests.
- [x] Route paper mode before real account dependency creation; configure prod simulation with comments.
- [x] Update docs and verify scoped lint, full race tests and supported build targets.
- [x] Obtain independent focused review; preserve known unrelated mock-drift changes.

## Final scope and verification

- User clarified: only ticker and hypothetical buy/sell price; no position size, cash, commission or latency env.
- Removed temporary DENSITY_PAPER_* fields and env entries; simulation ignores capital, uses zero fees/latency and one internal execution unit.
- `go test -race ./...`: passed.
- `go build ./internal/... ./pkg/... ./cmd/...`: passed.
- Scoped golangci-lint: 0 issues.
- Additional slow-delivery race test: passed; network wait does not block recording next execution.
- Reviewer findings fixed: fill + alert snapshot committed atomically before diagnostic journal; state guard scans known real bindings even without account configuration. Regression tests passed.
- Independent final review: no remaining actionable findings.
- Existing unrelated mock diff remains untouched; no commit/restart/real order/Telegram send performed.
