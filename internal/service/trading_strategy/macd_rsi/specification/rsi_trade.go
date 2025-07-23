package specification

import (
	"tinvest/internal/model"
)

type RsiTrade struct {
	SearchArea int
	Value      int64
}

func (s *RsiTrade) IsSatisfiedBy(itemTechAnalyse []*model.RsiItemTechAnalyse) bool {
	if len(itemTechAnalyse) <= s.SearchArea {
		return false
	}

	j := 0

	for i := len(itemTechAnalyse) - 1; j < s.SearchArea; i-- {
		item := itemTechAnalyse[i]

		if item.SignalLine.Units >= int64(50) {
			return true
		}

		j++
	}

	return false
}
