package bybitdensity

type spotProfile struct {
	zoneWidthUSDT float64
	minNotional   float64
}

// The profile list is also the subscription list for the spot worker.
func defaultSpotProfiles() map[string]spotProfile {
	return map[string]spotProfile{
		"BTCUSDT": {zoneWidthUSDT: 10, minNotional: 1_000_000},
		"ETHUSDT": {zoneWidthUSDT: 1, minNotional: 800_000},
		"SOLUSDT": {zoneWidthUSDT: 0.1, minNotional: 400_000},
	}
}
