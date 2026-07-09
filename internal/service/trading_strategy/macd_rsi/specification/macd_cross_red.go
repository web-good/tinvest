package specification

import (
	"tinvest/internal/domain"
)

type MacdCrossRed struct{}

func (s *MacdCrossRed) IsSatisfiedBy(itemTechAnalyse *domain.MACDItemTechAnalyse) bool {
	return itemTechAnalyse.IsCross
}
