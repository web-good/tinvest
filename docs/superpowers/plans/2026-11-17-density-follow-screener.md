# Density follow screener implementation plan

> **For agentic workers:** Use superpowers:executing-plans inline.

**Goal:** Permit entries for configured tickers plus tickers announced by the screener.
**Architecture:** Shared admission registry after successful alert delivery; expanded book-only runners for the scanner universe; existing entry/exit rules remain authoritative.
**Tech Stack:** Go, existing gRPC transport and paper/live engines.
**Spec:** docs/superpowers/specs/2026-11-17-density-follow-screener-design.md

## Constraints and review focus

- Preserve false/default mode and keep real orders disabled in prod.
- Failed alert delivery never admits an additional ticker; block unrelated tickers.
- Never subscribe trades for the whole 80-stock universe.
- Keep managing open positions regardless of admission changes.
- Respect simulated latency, visible size, commission and take activation.

## Tasks

- [x] Test and implement the env flag, shared ticker admission and sender success hook; wire both participants.
- [x] Test and implement expanded universe selection and book-only runners, retaining fixed mode.
- [x] Test and implement simulated book fills and take execution for expanded mode.
- [x] Update operational docs and prod config; run focused checks and mage ci.

## Validation

- Focused density/config/provider tests passed.
- `./bin/mage ci` passed: lint, mock drift, complete `go test -race ./...`.
- Independent review found no high/medium blockers, including the follow-up bound check.
- Additional regression: a coarse price tick never expands the percentage interest zone.
- Subscription universe remains fixed at startup, explicitly documented in the spec.
