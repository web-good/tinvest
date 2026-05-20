package dto

// Settings carries all tunable strategy knobs for Golden X. Each field has a
// well-defined default in golden_x.DefaultSettings(). Names use role-based
// terminology, not literal-value terminology, so the meaning survives retuning.
type Settings struct {
	// Buy-tier percentiles (RSI < BuyGreen → green; < BuyYellow → yellow).
	BuyGreen  float64 // default 5
	BuyYellow float64 // default 15

	// Sell-tier percentiles. Growth uses only SellOrange; Dividend uses all three.
	SellYellow float64 // default 80 (Dividend only)
	SellOrange float64 // default 90 (both kinds)
	SellRed    float64 // default 95 (Dividend only)

	// ATR-based stop.
	ATRPeriod             int     // default 14
	ATRMultiplierDividend float64 // default 2.0
	ATRMultiplierGrowth   float64 // default 1.5

	// Volume-confirmation indicator (last weekly volume > Multiplier × SMA of preceding Lookback weeks).
	VolumeSMALookback int     // default 20
	VolumeMultiplier  float64 // default 1.5

	// History windows.
	TrendEMAPeriod          int // default 200 (EMA200 W trend filter for Growth)
	AdaptiveWindowMin       int // default 100 (minimum closed-week RSI samples for adaptive tiers)
	AdaptiveWindowMax       int // default 200 (cap on closed-week RSI samples kept for percentiles)
	DivergenceLookbackWeeks int // default 52  (bullish-divergence pivot search horizon)
}
