# AFKS in FVG live monitoring

## Intent

Add AFKS to the existing FVG Imbalance live runner so it is monitored and processed alongside the configured tickers. The user chose monitoring only; order execution remains disabled.

## Scope

- Register AFKS in the live adapter's ticker-to-params mapping using `afks.Params()`.
- Append AFKS to `FVG_IMBALANCE_TICKERS` in `env/prod.env`.
- Keep `FVG_IMBALANCE_TRADE_ENABLED=false` and use the current AFKS literal (`core.DefaultParams()`). The calibrated candidate failed the reserved final-year quality limits and is not selected.
- Add AFKS to the live adapter registry test.

## Approaches considered

1. **Monitoring only with current literal (selected):** add AFKS to the live map and configured ticker list, leaving global trade execution disabled. This implements the requested runner inclusion without enabling orders or deploying the failed calibration candidate.
2. Use the research candidate in live. Rejected because it failed the reserved final-year check.
3. Enable order execution. Rejected for this change because the flag applies to all FVG tickers, not only AFKS.

## Constraints and non-goals

- Do not change strategy mechanics, AFKS params, or other ticker mappings.
- Do not turn on trade execution or alter sizing.
- This change only makes AFKS available to the existing live runner's monitoring path.

## Implementation calls

- Use the existing `params(ticker)` dispatch in `live/strategy.go`; no new routing layer is needed.
- Preserve the current `FVG_IMBALANCE_TRADE_ENABLED=false` setting.
