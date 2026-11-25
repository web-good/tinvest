package core

// Params holds instrument-specific parameters for the trendline strategy.
type Params struct {
	// TakeProfitR sets the target distance in multiples of the initial entry-to-stop risk.
	TakeProfitR float64
	// InPlayMinRangePct is the minimum mean high-low range of the last 12 M5 bars, in percent.
	InPlayMinRangePct float64
	// RequirePreviousDayHigh gates entry on the previous Moscow calendar day's high.
	RequirePreviousDayHigh bool
}
