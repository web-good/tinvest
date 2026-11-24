# Bybit Density In Play Implementation Plan

**Goal** — Restrict new density alerts and entries for PIEVERSEUSDT to volatile M5 windows with elevated USDT turnover.

**Architecture** — Bybit REST supplies closed spot M5 bars. Live and replay apply the same calculation, with replay selecting each window by event time.

**Tech Stack** — Go and Bybit V5 public REST.

**Spec** — [Bybit Density In Play](../specs/2026-11-24-bybit-density-in-play-design.md)

## Calculation and feed

- [x] Test threshold equality, a missing candle, insufficient turnover, and insufficient range in `strategy/in_play_test.go`.
- [x] Implement `model.M5Candle`, `strategy.InPlayFilter.Window` and the spot M5 reader, reusing the existing paginated candle fetch in `feed/candles.go`.
- [x] Verify the reader preserves missing bars, maps turnover to USDT, and lets the shared calculation reject incomplete windows.

## Live and replay

- [x] Test alert expiry and replay cancellation without dropping open positions.
- [x] Enable the filter in `spot.go` for PIEVERSEUSDT. Refresh the closed window through the alert hub and reject entries when the window is missing, stale, or inactive.
- [x] Load M5 history once for the replay and update the window before processing the corresponding event. Include active windows and thresholds in the report.
- [x] Document the behavior and run the Bybit tests, broader suite and a cached PIEVERSEUSDT replay.
