package dto

// Run is one scheduled invocation of the rsi_pullback runner. Unlike reversion there is no
// Mode: entries and exits are both evaluated on every 30-minute bar, so the runner makes a
// single pass — exactly one Decide per bar per ticker, as in the backtest engine.
type Run struct {
	Scheduler string
}
