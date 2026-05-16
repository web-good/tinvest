package dto

// SellThresholds are the per-share adaptive RSI percentile boundaries that
// drive the Golden X sell tiers. Gold uses all three (p80/p90/p95 → 🟠/🔴/🚨).
// Growth uses only P90 (single 🔴 tier — sharp exit).
type SellThresholds struct {
	P80 float64
	P90 float64
	P95 float64
}
