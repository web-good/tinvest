package golden_x

import (
	"errors"

	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/utils"
)

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

// compactCandles drops nil entries from candles, preserving order.
// It is the only sanitization the prod path needs before handing the slice
// to Detect — the current forming candle (IsComplete=false) is intentionally
// kept so indicators reflect intra-week price movement.
func compactCandles(candles []*model.CandleItemTechAnalyse) []*model.CandleItemTechAnalyse {
	out := make([]*model.CandleItemTechAnalyse, 0, len(candles))
	for _, c := range candles {
		if c == nil {
			continue
		}
		out = append(out, c)
	}
	return out
}
