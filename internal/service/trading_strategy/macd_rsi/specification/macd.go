package specification

import (
	"tinvest/internal/model"
	"tinvest/internal/utils"
)

type Macd struct {
}

func (s *Macd) IsSatisfiedBy(itemTechAnalyse []*model.MacDItemTechAnalyse) bool {
	for i := len(itemTechAnalyse) - 1; i >= 1; i-- {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i-1]

		/*if item.SignalLine.Nano > 0 || item.MacDLine.Nano > 0 {
			continue
		}*/

		if item.MacDLine.Units > item.SignalLine.Units {
			if utils.CombinePrice(prevItem.MacDLine.Units, prevItem.MacDLine.Nano) < utils.CombinePrice(prevItem.SignalLine.Units, prevItem.SignalLine.Nano) {
				return true
			}
		}

		if item.MacDLine.Units == item.SignalLine.Units && item.MacDLine.Nano > item.SignalLine.Nano {
			if prevItem.MacDLine.Units == prevItem.SignalLine.Units && prevItem.MacDLine.Nano < prevItem.SignalLine.Nano {
				return true
			}

			if prevItem.MacDLine.Units < prevItem.SignalLine.Units {
				return true
			}
		}
	}

	return false
}
