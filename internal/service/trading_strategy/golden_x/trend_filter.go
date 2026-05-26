package golden_x

import (
	"errors"

	"tinvest/internal/domain/ema"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/utils"
)

// ErrInsufficientHistory is returned when a share does not have enough closed
// weekly candles to compute the EMA200 filter (fresh IPOs).
var ErrInsufficientHistory = errors.New("trend filter: insufficient candle history")

// trendStatusFromClosed computes the trend status from an ordered weekly
// candle slice: it compares the last close against an EMA of the given
// period over all closes. Returns ErrInsufficientHistory if the slice is
// shorter than period.
func trendStatusFromClosed(closed []*model.CandleItemTechAnalyse, period int) (dto.TrendStatus, error) {
	if len(closed) < period {
		return dto.TrendUnknown, ErrInsufficientHistory
	}
	closes := make([]float64, len(closed))
	for i, c := range closed {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	emaValues := ema.Compute(closes, period)
	lastClose := closes[len(closes)-1]
	lastEMA := emaValues[len(emaValues)-1]
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
