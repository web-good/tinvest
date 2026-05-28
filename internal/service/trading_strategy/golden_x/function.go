package golden_x

import (
	"tinvest/internal/model"
)

// compactCandles drops nil entries from candles, preserving order.
// It is the only sanitization the prod path needs before handing the slice
// to detector.Detect — the current forming candle (IsComplete=false) is
// intentionally kept so indicators reflect intra-week price movement.
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
