# DENSITY Pinbar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Use closed minute pinbars to confirm DENSITY buys in live trading and Telegram simulation.

**Architecture:** Share candle validation and pending-signal tracking. Subscribe to completed exchange candles for both runners and consume each signal once through the existing order paths.

**Tech Stack:** Go, T-Invest gRPC.

**Spec:** [DENSITY entry on a minute pinbar](../specs/2026-11-18-density-pinbar-design.md)

- [x] Test candle geometry, next-minute expiry and the two-tick move in both directions. Implement the shared trigger and stream adapter.
- [x] Connect the live feed, final broker submission check and protection. Test the live order path.
- [x] Connect the paper runner used for Telegram alerts. Test entry at an ask beyond the original zone and the remaining exit rules.
- [x] Run the Go test suite and update the strategy mechanics reference.
