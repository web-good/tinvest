package specification

import (
	"tinvest/internal/domain"
)

type RsiTrade struct {
}

func (s *RsiTrade) IsSatisfiedBy(itemTechAnalyse *domain.RSIItemTechAnalyse) bool {
	if itemTechAnalyse.SignalLine.Units < 70 {
		return true
	}

	return false
}
