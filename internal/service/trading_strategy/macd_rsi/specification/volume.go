package specification

import (
	"tinvest/internal/model"
)

type Volume struct{}

func (s *Volume) IsSatisfiedBy(itemTechAnalyse *model.CandleItemTechAnalyse, volume float64) bool {
	return float64(itemTechAnalyse.Volume) > volume
}
