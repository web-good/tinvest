package detector

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/model"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/internal/utils"
	"tinvest/pkg/indicators"
)

// adaptiveRSIForShare computes the share's last weekly RSI and the
// trimmed historical RSI slice used for percentile calculations. Returns
// ErrAdaptiveInsufficientHistory if fewer than minWin RSI values are
// available; trims the head to maxWin if longer. Threshold computation
// (adaptiveThresholds / adaptiveSellThresholds) is the caller's responsibility.
func adaptiveRSIForShare(candles []*model.CandleItemTechAnalyse, rsiPeriod, minWin, maxWin int) (float64, []float64, error) {
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	full := indicators.RSISeries(closes, rsiPeriod)
	if len(full) <= rsiPeriod {
		return 0, nil, ErrAdaptiveInsufficientHistory
	}
	rsi := full[rsiPeriod:]
	if len(rsi) < minWin {
		return 0, nil, ErrAdaptiveInsufficientHistory
	}
	if len(rsi) > maxWin {
		rsi = rsi[len(rsi)-maxWin:]
	}
	return rsi[len(rsi)-1], rsi, nil
}

// lowsAlignedToRSI extracts Low values from candles and trims the head
// so the returned slice aligns 1-to-1 with the rsiSeries returned by
// adaptiveRSIForShare. The result has the same length as rsiSeries (or
// shorter, if there are fewer candles available — defensive).
func lowsAlignedToRSI(candles []*model.CandleItemTechAnalyse, rsiPeriod int, rsiSeries []float64) []float64 {
	// adaptiveRSIForShare drops rsiPeriod warmup candles, then potentially
	// keeps only the last settings.AdaptiveWindowMax. Mirror the same trim here.
	start := rsiPeriod
	if len(candles)-rsiPeriod > len(rsiSeries) {
		start = len(candles) - len(rsiSeries)
	}
	if start < 0 {
		start = 0
	}
	out := make([]float64, 0, len(candles)-start)
	for i := start; i < len(candles); i++ {
		out = append(out, utils.CombinePrice(candles[i].Low.Units, candles[i].Low.Nano))
	}
	return out
}

// trendStatusFromClosed computes the trend status from an ordered weekly
// candle slice: it compares the last close against an EMA of the given
// period over all closes. Returns ErrInsufficientHistory if the slice is
// shorter than period.
func trendStatusFromClosed(closed []*model.CandleItemTechAnalyse, period int) (gxmodel.TrendStatus, error) {
	if len(closed) < period {
		return gxmodel.TrendUnknown, ErrInsufficientHistory
	}
	closes := make([]float64, len(closed))
	for i, c := range closed {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	emaValues := ema.Compute(closes, period)
	lastClose := closes[len(closes)-1]
	lastEMA := emaValues[len(emaValues)-1]
	if lastClose > lastEMA {
		return gxmodel.TrendWith, nil
	}
	return gxmodel.TrendAgainst, nil
}
