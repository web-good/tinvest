package specification

import (
	"tinvest/internal/domain"
)

type MacdCrossRed struct{}

func (s *MacdCrossRed) IsSatisfiedBy(itemTechAnalyse *domain.MACDItemTechAnalyse) bool {
	if itemTechAnalyse.IsCross == true {
		return true
	}

	return false
}
