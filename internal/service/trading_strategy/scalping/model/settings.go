package model

import "tinvest/internal/enum"

// Settings holds the runner-level knobs for the scalping strategy.
// Per-share signal parameters live inside each strategy.
type Settings struct {
	Interval enum.Interval // timeframe
}

// DefaultSettings returns the hourly default.
func DefaultSettings() Settings {
	return Settings{
		Interval: enum.Hour1,
	}
}
