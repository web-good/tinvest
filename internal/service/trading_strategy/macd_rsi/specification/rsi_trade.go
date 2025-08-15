package specification

import (
	"tinvest/internal/model"
)

type RsiTrade struct {
}

func (s *RsiTrade) IsSatisfiedBy(itemTechAnalyse *model.RsiItemTechAnalyse) bool {
	if itemTechAnalyse.SignalLine.Units > 50 && itemTechAnalyse.SignalLine.Units < 70 {
		return true
	}

	return false
}
