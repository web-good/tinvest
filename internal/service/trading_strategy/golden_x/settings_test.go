package golden_x

import (
	"testing"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	want := dto.Settings{
		BuyGreen: 5, BuyYellow: 15,
		SellYellow: 80, SellOrange: 90, SellRed: 95,
		ATRPeriod: 14, ATRMultiplierDividend: 2.0, ATRMultiplierGrowth: 1.5,
		VolumeSMALookback: 20, VolumeMultiplier: 1.5,
		TrendEMAPeriod: 200, AdaptiveWindowMin: 100, AdaptiveWindowMax: 200,
		DivergenceLookbackWeeks: 52,
	}
	if s != want {
		t.Fatalf("DefaultSettings drift:\n got %+v\nwant %+v", s, want)
	}
}
