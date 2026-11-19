# Bybit Density Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans for integration. Report test evidence for each independent component.

**Goal:** Run the standalone Bybit scanner and paper execution with independent Telegram destinations.

**Architecture:** A public V5 client produces normalized market events. The scanner tracks confirmed episodes and the strategy consumes fresh public data and emits simulated fills. The command wires both markets, Telegram and the local execution journal.

**Tech Stack:** Go 1.25, Bybit V5 REST and WebSocket, Telegram Bot API.

**Spec:** [Bybit Density design](../specs/2026-11-19-bybit-density-design.md)

## Global Constraints

The process owns its environment, storage and two chat IDs. Market identities include category and symbol. Trading activity stays in simulation.

## Review Focus

- A fresh delta after a quiet period restarts confirmation without discarding the local book.
- A connection reset invalidates current book eligibility until an authoritative snapshot.
- Spot and linear BTCUSDT cannot share episodes or paper positions.
- A partial exit cannot reuse volume unchanged by a later delta.
- A persisted but unacknowledged execution alert survives restart.

### 1. Public feeds

- [x] Add replay tests for snapshots, amount-zero deltas, invalid updates and distinct markets.
- [x] Implement category-specific metadata, ranking and V5 subscriptions in `internal/service/screener/bybitdensity/feed`.
- [x] Run targeted feed tests and a public read-only smoke check.

### 2. Detection and execution

- [x] Test both book sides, episode expiry, confirmed M1 entry, partial exit and fee-adjusted results.
- [x] Implement detector and simulation in `internal/service/screener/bybitdensity/strategy`.
- [x] Run targeted strategy tests.

### 3. Journal and command

- [x] Test orphaned sessions, restart delivery and routing with a fake destination.
- [x] Implement state and simulation outbox in `internal/service/screener/bybitdensity/journal`.
- [x] Wire market workers and independent chats in `cmd/bybit-density` and document startup.
- [x] Build the command and run targeted race tests, integration replay and repository checks.
