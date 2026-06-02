package dto

import "tinvest/internal/enum"

// Trade carries per-run parameters for the scalping strategy.
type Trade struct {
	Interval  enum.Interval
	Scheduler string
}
