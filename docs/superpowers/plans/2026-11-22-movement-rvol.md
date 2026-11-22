# Movement RVOL Implementation Plan

> For agentic workers: use superpowers:executing-plans or superpowers:subagent-driven-development.

Add market-specific RVOL thresholds to the existing Go movement screener. Use the baseline and output described in the [specification](../specs/2026-11-22-movement-rvol-design.md).

- [x] Update the Binance digest fixture to check real current and historical volume and the report threshold.
- [x] Read candle volumes in both feeds. Request daily history only for candidates. Cache historical half-hour totals and combine adjacent totals for hourly reports.
- [x] Wire independent defaults and env keys through app config to both markets. Verify the digest, missing and zero baselines, and the application packages.
