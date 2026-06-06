package rusal

import (
	"tinvest/internal/service/trading_strategy/levels/strategy/core"
)

// Ticker is the instrument this config targets.
const Ticker = "RUAL"

// DefaultParams returns RUAL params for the levels strategy. All values are
// starting points pending calibration against RUAL history.
func DefaultParams() core.Params {
	return core.Params{
		ProfileWindow:     120,   // ~one intraday session of bars for the profile
		ProfileBins:       40,    // profile resolution
		HVNFactor:         1.5,   // HVN when a bin holds >= 1.5x the mean bin volume
		ATRPeriod:         14,    // ATR for stops/room/anti-churn
		LevelTolATR:       0.5,   // a touch counts within 0.5 ATR of the level
		RoomATR:           2.0,   // need >= 2 ATR of runway to resistance
		MaxExtensionATR:   1.0,   // skip entries > 1 ATR above support (chasing)
		SLMult:            1.0,   // hard stop 1 ATR below support
		TrailMult:         2.5,   // chandelier trail at 2.5 ATR below the recent high
		ChandelierWindow:  20,    // recent-high window for the trail
		TrailArmATR:       1.0,   // arm trail after +1 ATR profit (kills bar-1 stop-outs)
		BreakoutLookback:  10,    // bars back to classify a retest vs a bounce
		TrendFilterPeriod: 50,    // daily EMA for the HTF downtrend filter
		TrendSlopeTol:     0.0,   // any falling EMA with price below it counts as downtrend
		ConfirmCloseFrac:  0.5,   // require the bar to close in its upper half
		MinATRFrac:        0.003, // skip sub-0.3%-ATR setups (anti-churn)
		MinRR:             1.5,   // skip setups whose target is < 1.5x the risk
	}
}

// New returns the levels strategy bound to RUAL with its default params.
func New() *core.Strategy { return core.NewWithParams(Ticker, DefaultParams()) }
