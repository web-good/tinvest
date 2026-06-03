package dto

// Trade carries per-run parameters for the scalping strategy.
type Trade struct {
	Scheduler string
	// SellOnly turns the run into an out-of-schedule exit watcher: it processes
	// only instruments currently held and emits only Sell signals.
	SellOnly bool
}
