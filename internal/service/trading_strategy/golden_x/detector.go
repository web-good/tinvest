package golden_x

import (
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/divergence"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/internal/service/trading_strategy/golden_x/percentile"
	"tinvest/pkg/indicators"
)

// Detect runs the full Golden X signal pipeline against an ordered weekly
// candle history for a single share. It is pure: no I/O, no time
// dependency, no telemetry. The caller controls slice composition:
// service.Trade includes the currently forming weekly candle so indicators
// react intra-week.
//
// Returns ErrAdaptiveInsufficientHistory or ErrInsufficientHistory when
// history is too short for the adaptive RSI window or the EMA200 trend
// filter respectively. Other errors are not currently surfaced.
func Detect(
	closed []*model.CandleItemTechAnalyse,
	rsiPeriod int,
	kind gxmodel.StrategyKind,
	useTrendFilter bool,
	settings gxmodel.Settings,
) (gxmodel.Signal, error) {
	lastRSI, rsiSeries, err := adaptiveRSIForShare(closed, rsiPeriod, settings.AdaptiveWindowMin, settings.AdaptiveWindowMax)
	if err != nil {
		return gxmodel.Signal{}, err
	}

	sig := gxmodel.Signal{
		RSI:            lastRSI,
		Thresholds:     percentile.AdaptiveThresholds(rsiSeries, settings.BuyGreen, settings.BuyYellow),
		SellThresholds: percentile.AdaptiveSellThresholds(rsiSeries, settings.SellYellow, settings.SellOrange, settings.SellRed),
	}

	if useTrendFilter {
		status, terr := trendStatusFromClosed(closed, settings.TrendEMAPeriod)
		if terr != nil {
			return gxmodel.Signal{}, terr
		}
		sig.TrendStatus = status
	}

	buyTier := percentile.TierFromAdaptive(lastRSI, sig.Thresholds.P5, sig.Thresholds.P15)
	sig.GreenBuy = buyTier == percentile.TierGreen
	sig.YellowBuy = buyTier == percentile.TierYellow

	if buyTier != percentile.TierNone {
		lows := lowsAlignedToRSI(closed, rsiPeriod, rsiSeries)
		if len(lows) > settings.DivergenceLookbackWeeks {
			lows = lows[len(lows)-settings.DivergenceLookbackWeeks:]
		}
		rsiTail := rsiSeries
		if len(rsiTail) > settings.DivergenceLookbackWeeks {
			rsiTail = rsiTail[len(rsiTail)-settings.DivergenceLookbackWeeks:]
		}
		sig.DivergenceOK = divergence.Bullish(lows, rsiTail, divergenceFractalK)

		volumes := make([]int64, len(closed))
		for i, c := range closed {
			volumes[i] = c.Volume
		}
		sig.VolumeOK = indicators.VolumeConfirmed(volumes, settings.VolumeSMALookback, settings.VolumeMultiplier)
	}

	return sig, nil
}
