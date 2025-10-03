package specification

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/model"
	"tinvest/internal/utils"
)

type Ema struct{}

func (s *Ema) IsSatisfiedBy(ema ema.ItemTechAnalyse, candle model.CandleItemTechAnalyse) bool {
	if utils.CombinePrice(ema.SignalLine.Units, ema.SignalLine.Nano) <
		utils.CombinePrice(candle.Open.Units, candle.Open.Nano) {
		return true
	}

	return false
}
