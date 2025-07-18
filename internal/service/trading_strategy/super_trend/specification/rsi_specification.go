package specification

import (
	"tinvest/internal/model"
)

type RsiSpecification struct{}

func (s *RsiSpecification) IsSatisfiedBy(itemTechAnalyse []*model.RsiItemTechAnalyse) bool {
	iterLen := 1

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	j := 0

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i-1]

		if prevItem.SignalLine.Units > int64(30) && item.SignalLine.Units <= int64(30) {
			return true
		}

		//if prevItem.SignalLine.Units < int64(55) && item.SignalLine.Units >= int64(55) {
		//	return true
		//}

		j++
	}

	return false
}
