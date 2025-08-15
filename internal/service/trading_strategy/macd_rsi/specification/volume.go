package specification

import "tinvest/internal/model"

type Volume struct{}

func (s *Volume) IsSatisfiedBy(itemTechAnalyse *model.CandleItemTechAnalyse, volume float64) bool {
	if float64(itemTechAnalyse.Volume) > volume {
		return true
	}

	return false
}
