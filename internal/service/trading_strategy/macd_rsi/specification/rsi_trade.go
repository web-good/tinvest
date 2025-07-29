package specification

import (
	"tinvest/internal/model"
)

type RsiTrade struct {
	Value int64
}

func (s *RsiTrade) IsSatisfiedBy(itemTechAnalyse []*model.RsiItemTechAnalyse) bool {
	iterLen := 1
	j := 0
	itemTechAnalyse = itemTechAnalyse[:len(itemTechAnalyse)-1]

	if len(itemTechAnalyse) == 0 || len(itemTechAnalyse) < iterLen+1 {
		return false
	}

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i-1]

		if item.SignalLine.Units < s.Value && prevItem.SignalLine.Units >= s.Value {
			return true
		}

		j++
	}

	return false
}
