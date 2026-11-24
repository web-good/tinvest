package bybitdensity

import "tinvest/internal/service/screener/bybitdensity/strategy"

type spotProfile struct {
	zoneWidthUSDT        float64
	m5VolumeMultiplier   float64
	zoneFilter           strategy.ZoneFilter
	inPlay               strategy.InPlayFilter
	bounceConfirmSeconds int
}

// The profile list is also the subscription list for the spot worker.
func defaultSpotProfiles() map[string]spotProfile {
	return map[string]spotProfile{
		"BTCUSDT": {zoneWidthUSDT: 10, m5VolumeMultiplier: 1},
		"ETHUSDT": {zoneWidthUSDT: 1, m5VolumeMultiplier: 1},
		"SOLUSDT": {zoneWidthUSDT: 0.001, zoneFilter: strategy.ZoneFilter{MinNotional: 2500, NeighborMultiple: 2}, bounceConfirmSeconds: 2},
		"PIEVERSEUSDT": {
			zoneWidthUSDT: 0.001, zoneFilter: strategy.ZoneFilter{MinNotional: 4000, NeighborMultiple: 4},
			inPlay: strategy.InPlayFilter{MinRange: .01, MinTurnoverMultiple: 2}, bounceConfirmSeconds: 1,
		},
	}
}
