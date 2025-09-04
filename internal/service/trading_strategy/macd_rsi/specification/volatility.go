package specification

import (
	"tinvest/internal/domain"
	"tinvest/internal/utils"
)

type Volatility struct {
}

func (s *Volatility) IsSatisfiedBy(itemTechAnalyse *domain.VolatilityItemTechAnalyse) bool {
	if utils.CombinePrice(itemTechAnalyse.Value.Units, itemTechAnalyse.Value.Nano) < 0.3 {
		return false
	}

	return true
}
