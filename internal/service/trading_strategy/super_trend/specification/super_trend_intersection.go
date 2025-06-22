package specification

import (
	"time"
	"tinvest/internal/model"
)

type SuperTrendIntersection struct{}

type item struct {
	iFastEma  model.EmaItemTechAnalyse
	iTradeEma model.EmaItemTechAnalyse
}

func (s *SuperTrendIntersection) IsSatisfiedBy(fastEma []*model.EmaItemTechAnalyse, tradeEma []*model.EmaItemTechAnalyse) bool {
	iterLen := 2
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
	itemTechAnalyse.iFastEma = *fastEma[len(fastEma)-1]
	itemTechAnalyse.iTradeEma = *tradeEma[len(fastEma)-1]
	prevItemTechAnalyse.iFastEma = *fastEma[len(fastEma)-2]
	prevItemTechAnalyse.iTradeEma = *tradeEma[len(fastEma)-2]

	if itemTechAnalyse.iFastEma.Date.Hour() != timeNow.Hour() ||
		itemTechAnalyse.iFastEma.Date.Hour() != timeNow.Hour() ||
		prevItemTechAnalyse.iFastEma.Date.Hour() != timeNow.Add(-1*time.Hour).Hour() ||
		prevItemTechAnalyse.iTradeEma.Date.Hour() != timeNow.Add(-1*time.Hour).Hour() {
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
		return true
	}

	return false
}
