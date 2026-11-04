# Final fix report: FVG retest-depth review

## Changes

- `docs/fvg_imbalance/strategy.md`: clarified that retest depth is reached by the candle minimum (wick), not its close.
- `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go`: corrected the `MinRetestDepthPct` comment to describe the wick minimum. The pre-existing `TakeProfitR: 1.5` working-tree change was left unstaged and is not part of this fix.
- `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`: changed the shallow first touch to low 101.5 for zone `[100, 102]`, fixed test depth at 50%, and asserted the snapshot ending at that first touch emits no entry. The later qualifying touch remains, while the overall result confirms the consumed zone does not produce an entry.

## Verification

- `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core` — passed.
- `go test -run '^$' ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp` — passed (package compiles; no tests run).
- `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp` — failed at existing `TestGAZPParams`: working-tree `TakeProfitR: 1.5` differs from expected `1`. That unrelated parameter change was preserved and excluded from the fix commit.
- `git diff --cached --check` — passed; no Markdown-specific checker was configured or run.

## Commit

`fix: clarify FVG retest depth measurement`. The unrelated GAZP parameter change was excluded.
