package specification

import (
	"tinvest/internal/model"
)

const countGreenCandles int = 1

type GreenMacD struct {
}

func (s *GreenMacD) IsSatisfiedBy(itemTechAnalyse []*model.MacDItemTechAnalyse) bool {
	iterLen := 1
	j := 0
	countGreenIndicators := 0

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]

		if item.MacDLine.Units > item.SignalLine.Units && i > 0 {
			countGreenIndicators++
		}

		if item.MacDLine.Units == item.SignalLine.Units && i > 0 && item.MacDLine.Nano > item.SignalLine.Nano {
			countGreenIndicators++
		}

		j++
	}

	return countGreenIndicators == countGreenCandles
}
