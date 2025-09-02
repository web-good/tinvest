package specification

import (
	"tinvest/internal/domain"
)

type Macd struct {
}

func (s *Macd) IsSatisfiedBy(itemTechAnalyse []*domain.MACDItemTechAnalyse) bool {
	for i := len(itemTechAnalyse) - 1; i >= 1; i-- {
		item := itemTechAnalyse[i]
		//	prevItem := itemTechAnalyse[i-1]

		if item == nil {
			//|| prevItem == nil
			continue
		}

		if item.IsCross == false || item.Diff > 0 {
			continue
		}

		if item.UnderZero == true {
			//&& prevItem.Diff > 0
			return false
		}

		return true
	}

	return false
}
