package model

type Settings struct {
	BuyGreen  float64
	BuyYellow float64

	SellYellow float64
	SellOrange float64
	SellRed    float64

	VolumeSMALookback int
	VolumeMultiplier  float64

	TrendEMAPeriod          int
	AdaptiveWindowMin       int
	AdaptiveWindowMax       int
	DivergenceLookbackWeeks int
}
