package specification

import (
	domainema "tinvest/internal/domain/ema"
)

type Over200ema struct{}

type item struct {
	iFastEma  domainema.ItemTechAnalyse
	iTradeEma domainema.ItemTechAnalyse
}

func (s *Over200ema) IsSatisfiedBy(fastEma []domainema.ItemTechAnalyse, tradeEma []domainema.ItemTechAnalyse) bool {
	iterLen := 1

	if len(fastEma) <= iterLen || len(tradeEma) <= iterLen {
		return false
	}

	var (
		itemTechAnalyse item
	)
	itemFlagTrade := false
	j := 0
	tradeI := len(tradeEma)

	for fastI := len(fastEma) - 1; j < iterLen; fastI-- {
		tradeI = tradeI - 1
		itemTechAnalyse.iFastEma = fastEma[fastI]

		if itemTechAnalyse.iFastEma.SignalLine.Units > itemTechAnalyse.iTradeEma.SignalLine.Units {
			itemFlagTrade = true
		}
		if itemTechAnalyse.iFastEma.SignalLine.Units == itemTechAnalyse.iTradeEma.SignalLine.Units &&
			itemTechAnalyse.iFastEma.SignalLine.Nano > itemTechAnalyse.iTradeEma.SignalLine.Nano {
			itemFlagTrade = true
		}

		if itemFlagTrade == true {
			return true
		}

		j++
	}

	return false
}
