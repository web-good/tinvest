package specification

import (
	"tinvest/internal/model"
)

type RsiSpecification struct{}

func (s *RsiSpecification) IsSatisfiedBy(itemTechAnalyse []*model.RsiItemTechAnalyse) bool {
	iterLen := 4

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	j := 0

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i-1]

		if prevItem.SignalLine.Units < int64(50) && item.SignalLine.Units >= int64(50) {
			return true
		}

		j++
	}

	return false
}
