package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// DefaultSettings returns the strategy knobs in use across the codebase prior
// to D2. Behavioral parity is the explicit contract: calling Detect with
// DefaultSettings() must produce byte-identical output to the pre-D2 code.
func DefaultSettings() dto.Settings {
	return dto.Settings{
		BuyGreen:                5,
		BuyYellow:               15,
		SellYellow:              80,
		SellOrange:              90,
		SellRed:                 95,
		ATRPeriod:               14,
		ATRMultiplierDividend:   2.0,
		ATRMultiplierGrowth:     1.5,
		VolumeSMALookback:       20,
		VolumeMultiplier:        1.5,
		TrendEMAPeriod:          200,
		AdaptiveWindowMin:       100,
		AdaptiveWindowMax:       200,
		DivergenceLookbackWeeks: 52,
	}
}
