package dto

// Stop is the ATR-based stop suggestion attached to a buy-tier row.
// The zero value means "not computed" — the notification renderer treats it
// as nothing, the same way it treats empty Thresholds and SellThresholds.
type Stop struct {
	Price       float64
	DistancePct float64
}
