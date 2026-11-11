# FVG Imbalance: optional stochastic exit

## Intent
Make the existing stochastic position exit independently configurable for `fvg_imbalance`, while preserving current behavior by default. Stop-loss and take-profit exits remain active and keep their current priority.

## Scope and constraints
- Add integer parameter `UseStochExit`, accepting only `0` or `1`.
- Default `UseStochExit` to `1` so current configurations retain their behavior.
- When set to `0`, suppress only the `%K` upward-crossing exit above 80.
- Keep stop-loss first and take-profit second; stochastic exit remains third when enabled.
- Keep the entry stochastic logic unchanged.
- Expose the parameter through ticker parameter definitions and calibration grids, and document it.

## Approaches
- Add an integer switch defaulting to `1` (chosen): fits the existing reflection-friendly parameter/grid convention and preserves current behavior.
- Default the switch to `0`: rejected because existing configurations would change behavior without being edited.
- Remove the exit or encode it indirectly through another parameter: rejected because this request is for an independent, explicit option.

## Implementation decisions
Add `UseStochExit` to core params, default and validation; guard only the existing stochastic sell condition. Propagate the parameter through GAZP/DOMRF ticker defaults and grids so calibration can compare enabled and disabled behavior. Update the strategy mechanics documentation. Take-profit and stop handling remain untouched.

## Out of scope
Changing stochastic thresholds, entry rules, stop/target execution, or calibrating ticker values.
