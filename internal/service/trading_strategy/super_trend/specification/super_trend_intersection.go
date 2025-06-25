package specification

import (
	"fmt"
	"time"
	domainema "tinvest/internal/domain/ema"
)

type SuperTrendIntersection struct{}

type item struct {
	iFastEma  domainema.ItemTechAnalyse
	iTradeEma domainema.ItemTechAnalyse
}

func (s *SuperTrendIntersection) IsSatisfiedBy(fastEma []domainema.ItemTechAnalyse, tradeEma []domainema.ItemTechAnalyse) bool {
	iterLen := 3
	timeNow := time.Now()

	if len(fastEma) <= iterLen || len(tradeEma) <= iterLen {
		return false
	}

	var (
		itemTechAnalyse     item
		prevItemTechAnalyse item
	)
	itemFlagTrade := false
	itemPrevFlagTrade := false
	j := 0

	for i := len(fastEma) - 1; j < iterLen; i-- {
		itemTechAnalyse.iFastEma = fastEma[i]
		itemTechAnalyse.iTradeEma = tradeEma[i]
		prevItemTechAnalyse.iFastEma = fastEma[i-1]
		prevItemTechAnalyse.iTradeEma = tradeEma[i-1]

		if i == len(fastEma)-1 && (itemTechAnalyse.iFastEma.Date.Hour() != timeNow.Hour() ||
			itemTechAnalyse.iTradeEma.Date.Hour() != timeNow.Hour() ||
			prevItemTechAnalyse.iFastEma.Date.Hour() != timeNow.Add(-1*time.Hour).Hour() ||
			prevItemTechAnalyse.iTradeEma.Date.Hour() != timeNow.Add(-1*time.Hour).Hour()) {
			return false
		}

		if itemTechAnalyse.iFastEma.SignalLine.Units > itemTechAnalyse.iTradeEma.SignalLine.Units {
			itemFlagTrade = true
		}

		if prevItemTechAnalyse.iFastEma.SignalLine.Units < prevItemTechAnalyse.iTradeEma.SignalLine.Units {
			itemPrevFlagTrade = true
		}

		if itemTechAnalyse.iFastEma.SignalLine.Units == itemTechAnalyse.iTradeEma.SignalLine.Units &&
			itemTechAnalyse.iFastEma.SignalLine.Nano > itemTechAnalyse.iTradeEma.SignalLine.Nano {
			itemFlagTrade = true
		}

		if prevItemTechAnalyse.iFastEma.SignalLine.Units == prevItemTechAnalyse.iTradeEma.SignalLine.Units &&
			prevItemTechAnalyse.iFastEma.SignalLine.Nano < prevItemTechAnalyse.iTradeEma.SignalLine.Nano {
			itemFlagTrade = true
		}

		if itemPrevFlagTrade == true && itemFlagTrade == true {
			fmt.Println(itemTechAnalyse, prevItemTechAnalyse)
			return true
		}

		j++
	}

	return false
}
