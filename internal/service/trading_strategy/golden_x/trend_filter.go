package golden_x

import (
	"errors"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/utils"
)

// trendEMAPeriod is the lookback for the higher-trend EMA filter (EMA200 W).
const trendEMAPeriod = 200

// ErrInsufficientHistory is returned when a share does not have enough closed
// weekly candles to compute the EMA200 filter (fresh IPOs).
var ErrInsufficientHistory = errors.New("trend filter: insufficient candle history")

// computeEMA returns a sliding EMA of the given period over closes.
// Result has the same length as closes; positions before period-1 are zero.
// The seed is an SMA of the first `period` values, matching the formula used
// across the codebase (see internal/service/instrument/ema/tech_analyse.go).
func computeEMA(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if len(closes) < period {
		return out
	}

	var sum float64
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	out[period-1] = sum / float64(period)
	multiplier := 2.0 / float64(period+1)

	for k := period; k < len(closes); k++ {
		out[k] = (closes[k]-out[k-1])*multiplier + out[k-1]
	}
	return out
}

// trendStatusFromCandles computes the trend status using the last closed
// weekly candle (Monday 00:00 in loc as the week boundary) and an EMA of the
// given period over closed-week closes. Returns ErrInsufficientHistory when
// fewer than `period` closed weekly candles are available.
func trendStatusFromCandles(candles []*model.CandleItemTechAnalyse, period int, now time.Time, loc *time.Location) (dto.TrendStatus, error) {
	return trendStatusFromClosed(closedWeeklyCandles(candles, now, loc), period)
}

// trendStatusFromClosed is the closed-candle-aware variant used by Detect
// (which receives already-trimmed candles). Equivalent to
// trendStatusFromCandles but skips the closedWeeklyCandles filter.
func trendStatusFromClosed(closed []*model.CandleItemTechAnalyse, period int) (dto.TrendStatus, error) {
	if len(closed) < period {
		return dto.TrendUnknown, ErrInsufficientHistory
	}
	closes := make([]float64, len(closed))
	for i, c := range closed {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	ema := computeEMA(closes, period)
	lastClose := closes[len(closes)-1]
	lastEMA := ema[len(ema)-1]
	if lastClose > lastEMA {
		return dto.TrendWith, nil
	}
	return dto.TrendAgainst, nil
}

// closedWeeklyCandles returns the input slice filtered to candles whose Time
// is strictly before the current week (Monday 00:00 in loc), preserving order.
// nil entries are skipped.
func closedWeeklyCandles(candles []*model.CandleItemTechAnalyse, now time.Time, loc *time.Location) []*model.CandleItemTechAnalyse {
	currentWeekStart := startOfWeek(now, loc)
	out := make([]*model.CandleItemTechAnalyse, 0, len(candles))
	for _, c := range candles {
		if c == nil {
			continue
		}
		if c.Time.Before(currentWeekStart) {
			out = append(out, c)
		}
	}
	return out
}
