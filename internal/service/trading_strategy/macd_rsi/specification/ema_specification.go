package specification

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/model"
)

type EmaSpecification struct{}

func (s *EmaSpecification) IsSatisfiedBy(ema ema.ItemTechAnalyse, candle model.CandleItemTechAnalyse) bool {
	if ema.SignalLine.Units < candle.Close.Units {
		return true
	}

	return false
}
