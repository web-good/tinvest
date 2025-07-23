package specification

import (
	"tinvest/internal/model"
)

type RsiProfit struct{}

func (s *RsiProfit) IsSatisfiedBy(itemTechAnalyse []*model.RsiItemTechAnalyse) bool {
	iterLen := 1

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	j := 0

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i-1]

		if prevItem.SignalLine.Units < int64(70) && item.SignalLine.Units >= int64(70) {
			return true
		}

		j++
	}

	return false
}
