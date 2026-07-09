package specification

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/model"
	"tinvest/internal/utils"
)

type EmaSpecification struct{}

func (s *EmaSpecification) IsSatisfiedBy(ema ema.ItemTechAnalyse, candle model.CandleItemTechAnalyse) bool {
	return utils.CombinePrice(ema.SignalLine.Units, ema.SignalLine.Nano) <
		utils.CombinePrice(candle.Open.Units, candle.Open.Nano)
}
