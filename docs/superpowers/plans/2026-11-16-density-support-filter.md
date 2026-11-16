# Density Support Filter Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans; implementation is authorized by the user in the current session.

**Goal:** Apply turnover-normalized density size and historical support to notifications and new entries.
**Architecture:** Load immutable instrument history profiles; apply them in the shared detector. Refresh live profiles asynchronously without interrupting position management.
**Tech Stack:** Go, existing gRPC market data, existing tests.
**Spec:** docs/superpowers/specs/2026-11-16-density-support-filter-design.md

## Global Constraints
- Preserve existing uncommitted changes; do not commit or restart services.
- No historical order-book collection, no new dependencies, no real orders during verification.
- Use completed H1 candles over 60 calendar days; profile validity 24 hours.

## Review Focus
- Missing or expired histories suppress new signals without affecting existing position management.
- Independent support touches; broken supports and unconfirmed pivots are excluded.
- Turnover uses lots correctly and minimum RUB remains effective.
- Screener/live/paper all install the same instrument profiles.
- Slow history requests do not block live stream processing.

### Task 1: Pure support algorithm and shared detector filter
**Files:** density/support.go, support_test.go, model.go, universe.go, detector.go, detector_test.go, notification.go.
**Interfaces:** SupportConfig, SupportCandle, Support; FindSupports; Detector.SetInstruments([]Instrument).
- [x] Write failing support and detector tests for grouping, bounce, break, incomplete candles, thresholds and expiration.
- [x] Implement pure support calculation, instrument profile fields and detector gating.
- [x] Run go test ./internal/service/screener/density.

### Task 2: History loading and live refresh
**Files:** density/refresh.go, profiles.go, connection.go; grpc/density_market_data.go; live/runner.go, participant.go; paper/runner.go.
**Interfaces:** SupportSource.SupportCandles(context.Context,string,int); RefreshProfiles; StreamProfiles.
- [x] Add failing history and adapter tests for completed H1 range, partial errors and expiration.
- [x] Load profiles after selecting liquid instruments and refresh in background in live/paper.
- [x] Install profiles in all detector consumers, preserving positions and feed lifecycle.
- [x] Run go test -race ./internal/service/screener/density/... ./pkg/client/grpc ./cmd/density-paper.

### Task 3: Configuration, activation, docs and verification
**Files:** internal/config/density*, internal/service_provider/density*, cmd/density-paper/main.go, env/prod.env, docs/orderbook_density/{screener,live,paper}.md.
- [x] Add failing validation/env tests, wire common settings to all consumers.
- [x] Activate prod with turnover fraction .005, support enabled; preserve user's max distance .004.
- [x] Document parameters, limits and profile refresh semantics.
- [x] Run relevant lint, full race tests and supported build targets; review only this task's changes.

## Verification

- Baseline density/live/paper/config/provider/gRPC/command tests passed before edits.
- New support/profile/config tests failed before implementation, then passed.
- Shared detector → live feed → engine tests verify actual buy requests are blocked by size/support filters.
- Slow history refresh test verifies position exits continue while history RPC is blocked.
- `go test -race ./...`: passed.
- `go build ./internal/... ./pkg/... ./cmd/...`: passed.
- Scoped golangci-lint: 0 issues.
- Independent read-only reviewer: no actionable correctness findings.
- `./bin/mage ci` cannot pass mock drift: existing uncommitted
  `internal/service/trading_strategy/livecore/executor/mocks/mock_OrdersClient.go`
  differs from HEAD (148 existing added lines). Regeneration leaves the same diff;
  this unrelated work was preserved.
- No commit, service restart or real broker orders performed.
