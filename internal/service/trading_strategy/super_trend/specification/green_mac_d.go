package specification

import (
	"fmt"
	"tinvest/internal/model"
)

const countGreenCandles int = 2

type GreenMacD struct {
	countGreenCandles int
}

func (s *GreenMacD) IsSatisfiedBy(itemTechAnalyse []*model.MacDItemTechAnalyse) bool {
	iterLen := 2
	j := 0
	countGreenIndicators := 0

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]
		fmt.Println("rrrrrrrrrrrr", item.Date, item.MacDLine)
		if item.MacDLine.Units > item.SignalLine.Units && i > 0 {
			countGreenIndicators++
		}

		if item.MacDLine.Units == item.SignalLine.Units && i > 0 && item.MacDLine.Nano > item.SignalLine.Nano {
			countGreenIndicators++
		}

		j++
	}

	if countGreenIndicators == countGreenCandles {
		return true
	}

	return false
}
