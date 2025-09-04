package specification

import (
	"tinvest/internal/domain"
	"tinvest/internal/utils"
)

type RsiTend struct {
}

func (s *RsiTend) IsSatisfiedBy(itemTechAnalyse []*domain.RSIItemTechAnalyse) bool {
	if utils.CombinePrice(itemTechAnalyse[0].SignalLine.Units, itemTechAnalyse[0].SignalLine.Nano) > 50 &&
		utils.CombinePrice(itemTechAnalyse[0].SignalLine.Units, itemTechAnalyse[0].SignalLine.Nano) < 70 &&
		utils.CombinePrice(itemTechAnalyse[1].SignalLine.Units, itemTechAnalyse[1].SignalLine.Nano) <= 50 {
		return true
	}

	return false
}
