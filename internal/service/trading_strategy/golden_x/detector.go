package golden_x

import (
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/utils"
	"tinvest/pkg/indicators"
)

// Detect runs the full Golden X signal pipeline against an already-closed
// weekly candle history for a single share. It is pure: no I/O, no time
// dependency, no telemetry. Callers (service.Trade and backtest.Replay)
// are responsible for trimming to closed weeks before invoking.
//
// Returns ErrAdaptiveInsufficientHistory or ErrInsufficientHistory when
// history is too short for the adaptive RSI window or the EMA200 trend
// filter respectively. Other errors are not currently surfaced.
func Detect(
	closed []*model.CandleItemTechAnalyse,
	rsiPeriod int,
	kind dto.StrategyKind,
	useTrendFilter bool,
	settings dto.Settings,
) (dto.Signal, error) {
	lastRSI, rsiSeries, err := adaptiveRSIForShare(closed, rsiPeriod, settings.AdaptiveWindowMin, settings.AdaptiveWindowMax)
	if err != nil {
		return dto.Signal{}, err
	}

	sig := dto.Signal{
		RSI:            lastRSI,
		Thresholds:     adaptiveThresholds(rsiSeries, settings.BuyGreen, settings.BuyYellow),
		SellThresholds: adaptiveSellThresholds(rsiSeries, settings.SellYellow, settings.SellOrange, settings.SellRed),
	}

	if useTrendFilter {
		status, terr := trendStatusFromClosed(closed, settings.TrendEMAPeriod)
		if terr != nil {
			return dto.Signal{}, terr
		}
		sig.TrendStatus = status
	}

	buyTier := tierFromAdaptive(lastRSI, sig.Thresholds.P5, sig.Thresholds.P15)
	sig.GreenBuy = buyTier == tierGreen
	sig.YellowBuy = buyTier == tierYellow

	if buyTier != tierNone {
		lows := lowsAlignedToRSI(closed, rsiPeriod, rsiSeries)
		if len(lows) > settings.DivergenceLookbackWeeks {
			lows = lows[len(lows)-settings.DivergenceLookbackWeeks:]
		}
		rsiTail := rsiSeries
		if len(rsiTail) > settings.DivergenceLookbackWeeks {
			rsiTail = rsiTail[len(rsiTail)-settings.DivergenceLookbackWeeks:]
		}
		sig.DivergenceOK = bullishDivergence(lows, rsiTail, divergenceFractalK)

		volumes := make([]int64, len(closed))
		for i, c := range closed {
			volumes[i] = c.Volume
		}
		sig.VolumeOK = indicators.VolumeConfirmed(volumes, settings.VolumeSMALookback, settings.VolumeMultiplier)

		highs := make([]float64, len(closed))
		lowsF := make([]float64, len(closed))
		closes := make([]float64, len(closed))
		for i, c := range closed {
			highs[i] = utils.CombinePrice(c.High.Units, c.High.Nano)
			lowsF[i] = utils.CombinePrice(c.Low.Units, c.Low.Nano)
			closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
		}
		if atrValue := indicators.ATR(highs, lowsF, closes, settings.ATRPeriod); atrValue > 0 {
			lastClose := closes[len(closes)-1]
			sig.LastClose = lastClose
			sig.Stop = stopFromATR(lastClose, atrValue, kForKind(kind, settings))
		} else {
			sig.LastClose = closes[len(closes)-1]
		}
	} else if len(closed) > 0 {
		last := closed[len(closed)-1]
		sig.LastClose = utils.CombinePrice(last.Close.Units, last.Close.Nano)
	}

	return sig, nil
}
