package specification

import (
	"tinvest/internal/model"
)

type Macd struct {
	SearchArea int
}

func (s *Macd) IsSatisfiedBy(itemTechAnalyse []*model.MacDItemTechAnalyse) bool {
	j := 0

	if len(itemTechAnalyse) <= s.SearchArea {
		return false
	}

	for i := len(itemTechAnalyse) - 1; j < s.SearchArea; i-- {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i-1]

		if prevItem.MacDLine.Units > 0 || prevItem.SignalLine.Units > 0 || prevItem.SignalLine.Nano > 0 || prevItem.MacDLine.Nano > 0 {
			j++

			continue
		}

		if item.MacDLine.Units > item.SignalLine.Units && i > 0 {
			if prevItem.MacDLine.Units < prevItem.SignalLine.Units {
				return true
			}
		}

		if item.MacDLine.Units == item.SignalLine.Units && i > 0 && item.MacDLine.Nano > item.SignalLine.Nano {
			if prevItem.MacDLine.Units == prevItem.SignalLine.Units && prevItem.MacDLine.Nano < prevItem.SignalLine.Nano {
				return true
			}

			if prevItem.MacDLine.Units < prevItem.SignalLine.Units {
				return true
			}
		}

		j++
	}

	return false
}
