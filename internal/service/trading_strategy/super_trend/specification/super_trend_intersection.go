package specification

import (
	domainema "tinvest/internal/domain/ema"
)

type SuperTrendIntersection struct{}

type item struct {
	iFastEma  domainema.ItemTechAnalyse
	iTradeEma domainema.ItemTechAnalyse
}

func (s *SuperTrendIntersection) IsSatisfiedBy(fastEma []domainema.ItemTechAnalyse, tradeEma []domainema.ItemTechAnalyse) bool {
	iterLen := 1

	if len(fastEma) <= iterLen || len(tradeEma) <= iterLen {
		return false
	}

	var (
		itemTechAnalyse item
		//prevItemTechAnalyse item
	)
	itemFlagTrade := false
	//itemPrevFlagTrade := false
	j := 0
	tradeI := len(tradeEma)

	for fastI := len(fastEma) - 1; j < iterLen; fastI-- {
		tradeI = tradeI - 1
		itemTechAnalyse.iFastEma = fastEma[fastI]
		itemTechAnalyse.iTradeEma = tradeEma[tradeI]
		//prevItemTechAnalyse.iFastEma = fastEma[fastI-1]
		//prevItemTechAnalyse.iTradeEma = tradeEma[tradeI-1]

		if itemTechAnalyse.iFastEma.SignalLine.Units > itemTechAnalyse.iTradeEma.SignalLine.Units {
			itemFlagTrade = true
		}
		if itemTechAnalyse.iFastEma.SignalLine.Units == itemTechAnalyse.iTradeEma.SignalLine.Units &&
			itemTechAnalyse.iFastEma.SignalLine.Nano > itemTechAnalyse.iTradeEma.SignalLine.Nano {
			itemFlagTrade = true
		}

		/*if itemFlagTrade {
			if prevItemTechAnalyse.iFastEma.SignalLine.Units == prevItemTechAnalyse.iTradeEma.SignalLine.Units &&
				prevItemTechAnalyse.iFastEma.SignalLine.Nano < prevItemTechAnalyse.iTradeEma.SignalLine.Nano {
				itemPrevFlagTrade = true
			}

			if prevItemTechAnalyse.iFastEma.SignalLine.Units < prevItemTechAnalyse.iTradeEma.SignalLine.Units {
				itemPrevFlagTrade = true
			}
		}
		*/
		if itemFlagTrade == true {
			return true
		}

		j++
	}

	return false
}
