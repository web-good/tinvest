package specification

import (
	"tinvest/internal/domain"
)

type RsiTrade struct {
}

func (s *RsiTrade) IsSatisfiedBy(itemTechAnalyse *domain.RSIItemTechAnalyse) bool {
	return itemTechAnalyse.SignalLine.Units < 70
}
