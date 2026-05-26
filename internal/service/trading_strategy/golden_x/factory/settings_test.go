package factory

import (
	"testing"

	"tinvest/internal/service/trading_strategy/golden_x/model"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	want := model.Settings{
		BuyGreen: 5, BuyYellow: 15,
		SellYellow: 80, SellOrange: 90, SellRed: 95,
		VolumeSMALookback: 20, VolumeMultiplier: 1.5,
		TrendEMAPeriod: 200, AdaptiveWindowMin: 100, AdaptiveWindowMax: 200,
		DivergenceLookbackWeeks: 52,
	}
	if s != want {
		t.Fatalf("DefaultSettings drift:\n got %+v\nwant %+v", s, want)
	}
}
