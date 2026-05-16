package dto

// Thresholds are the per-share adaptive RSI percentile boundaries that drive
// the Golden X buy tiers (replaces the static 31/35/40 buckets).
type Thresholds struct {
	P5  float64
	P15 float64
}
