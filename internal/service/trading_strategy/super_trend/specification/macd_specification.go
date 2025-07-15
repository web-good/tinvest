package specification

import (
	"tinvest/internal/model"
)

type MacDSpecification struct {
	differenceValue float64
}

func (s *MacDSpecification) IsSatisfiedBy(itemTechAnalyse []*model.MacDItemTechAnalyse) bool {
	iterLen := 3
	j := 0

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]

		if item.MacDLine.Units > 0 || item.SignalLine.Units > 0 || item.SignalLine.Nano > 0 || item.MacDLine.Nano > 0 {
			j++

			continue
		}

		if item.MacDLine.Units > item.SignalLine.Units && i > 0 {
			prevItem := itemTechAnalyse[i-1]

			if prevItem.MacDLine.Units < prevItem.SignalLine.Units {
				return true
			}
		}

		if item.MacDLine.Units == item.SignalLine.Units && i > 0 && item.MacDLine.Nano > item.SignalLine.Nano {
			prevItem := itemTechAnalyse[i-1]

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
