package strategy

import (
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	InPlayInterval  = 5 * time.Minute
	inPlayRecent    = 6
	inPlayReference = 288
	inPlayHistory   = inPlayRecent + inPlayReference
	InPlayLookback  = inPlayHistory * InPlayInterval
)

type InPlayFilter struct {
	MinRange, MinTurnoverMultiple float64
}

func (f InPlayFilter) Enabled() bool {
	return f.MinRange > 0 && f.MinTurnoverMultiple > 0
}

// InPlayWindow is valid for the five minutes following the latest candle close.
type InPlayWindow struct {
	From   time.Time
	Active bool
}

func (w InPlayWindow) allows(at time.Time) bool {
	return w.Active && !at.Before(w.From) && at.Before(w.From.Add(InPlayInterval))
}

// Window evaluates the latest six closes against the preceding 24 hours.
func (f InPlayFilter) Window(candles []model.M5Candle) InPlayWindow {
	if !f.Enabled() || len(candles) < inPlayHistory {
		return InPlayWindow{}
	}
	candles = candles[len(candles)-inPlayHistory:]
	window := InPlayWindow{From: candles[len(candles)-1].End}
	var rangeSum, recentTurnover, referenceTurnover float64
	for i, candle := range candles {
		if !candle.End.Equal(window.From.Add(time.Duration(i+1-inPlayHistory) * InPlayInterval)) {
			return window
		}
		if i < inPlayReference {
			referenceTurnover += candle.Turnover
			continue
		}
		if candle.Open <= 0 {
			return window
		}
		rangeSum += (candle.High - candle.Low) / candle.Open
		recentTurnover += candle.Turnover
	}
	referencePerWindow := referenceTurnover / (inPlayReference / inPlayRecent)
	const roundingTolerance = 1e-12
	window.Active = referencePerWindow > 0 && rangeSum/inPlayRecent+roundingTolerance >= f.MinRange &&
		recentTurnover/referencePerWindow+roundingTolerance >= f.MinTurnoverMultiple
	return window
}
