package classifier

import (
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/internal/service/trading_strategy/golden_x/percentile"
)

// Classify buckets detected signals into buy and sell maps based on adaptive
// percentile tiers. It is pure: no I/O, no context.
func Classify(detected []gxmodel.DetectResult, in dto.Trade) gxmodel.TradeResult {
	result := gxmodel.TradeResult{
		BuyShares:       make(map[string]gxmodel.ShareResult),
		SellShares:      make(map[string]gxmodel.ShareResult),
		CappedBuyShares: make(map[string]gxmodel.ShareResult),
		Kind:            in.Kind,
	}
	for _, dr := range detected {
		sig := dr.Signal
		buyTier := percentile.TierFromAdaptive(sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15)
		sellTier := percentile.SellTierFromAdaptive(sig.RSI, sig.SellThresholds, in.Kind)

		sr := gxmodel.ShareResult{
			InstrumentName: dr.Share.Name,
			RSI:            sig.RSI,
			BuyTier:        buyTier,
			SellTier:       sellTier,
			Thresholds:     sig.Thresholds,
			SellThresholds: sig.SellThresholds,
			Sector:         dr.Share.Sector,
		}
		if in.UseTrendFilter {
			sr.TrendStatus = sig.TrendStatus
		}
		if sig.GreenBuy || sig.YellowBuy {
			sr.DivergenceOK = sig.DivergenceOK
			sr.VolumeOK = sig.VolumeOK
		}

		sr.Score = signalScore(sr)

		// Buy and sell zones are mutually exclusive — RSI can't be both < p15 and > p80.
		switch {
		case buyTier != gxmodel.TierNone:
			result.BuyShares[dr.Share.ID] = sr
		case sellTier != gxmodel.TierNone:
			result.SellShares[dr.Share.ID] = sr
		}
	}
	return result
}

// signalScore computes a composite buy-signal strength score for a ShareResult.
// Higher scores indicate stronger confluence of buy conditions.
//
//   - TierGreen buy tier:  +3
//   - TierYellow buy tier: +1
//   - TrendWith:           +2
//   - DivergenceOK:        +2
//   - VolumeOK:            +1
func signalScore(sr gxmodel.ShareResult) int {
	s := 0
	switch sr.BuyTier {
	case gxmodel.TierGreen:
		s += 3
	case gxmodel.TierYellow:
		s += 1
	}
	if sr.TrendStatus == gxmodel.TrendWith {
		s += 2
	}
	if sr.DivergenceOK {
		s += 2
	}
	if sr.VolumeOK {
		s += 1
	}
	return s
}
