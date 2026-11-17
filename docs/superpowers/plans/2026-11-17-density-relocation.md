# Density relocation implementation plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Follow relocated density before entry, retain protection after entry, exit on genuine loss.
**Architecture:** Shared nearest-candidate selection; live feed tracks an anchor separately from fixed trade levels. Live engine cancels and reconciles before replacing under a stable ownership key. Paper engine tracks the same anchor using detector candidates from the same book.
**Tech Stack:** Go 1.25, existing broker and detector, no dependencies.
**Spec:** docs/superpowers/specs/2026-11-17-density-relocation-design.md

## Global constraints

- Preserve current working changes (percentage entry and 0.5% discovery).
- No real broker operations during development.
- First-entry confirmation, freshness, TTL, stop, and account coordination remain enforced.

## Review focus

- Fill racing cancellation must retain original protection and never buy twice.
- Unknown cancellation/placement must retain durable identities.
- New candidate must pass filters and belong to the current book and instrument.
- Repeated relocation must not extend entry TTL or change protection after entry.
- Shared ownership must retain old order IDs and survive restart.

## Tasks

- [x] Add failing live/paper tests for relocation up/down before/after fill, continued tracking and loss, and fill/cancel race.
- [x] Add shared `density.Relocation` selection and raw-candidate delivery alongside confirmed entry signals.
- [x] Add live watcher anchor, persisted replacement intent and stable ownership; cancel/reconcile before re-entry; revalidate and check funds under account lock.
- [x] Add paper repricing and fixed protection with relocation-aware loss monitoring.
- [x] Update mechanical docs; run focused tests, obtain independent review, fix findings, run `./bin/mage ci` and inspect diff.

Validation: focused relocation and notification tests passed; independent review
completed with its exit-reason finding fixed. Full `./bin/mage ci` passed
(lint, mock drift, and all tests with `-race`).
