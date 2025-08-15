package specification

import (
	"tinvest/internal/model"
	"tinvest/internal/utils"
)

type MacdCrossRed struct{}

func (s *MacdCrossRed) IsSatisfiedBy(itemTechAnalyse *model.MacDItemTechAnalyse) bool {
	signalLinePrice := utils.CombinePrice(itemTechAnalyse.SignalLine.Units, itemTechAnalyse.SignalLine.Nano)
	macdLinePrice := utils.CombinePrice(itemTechAnalyse.MacDLine.Units, itemTechAnalyse.MacDLine.Nano)

	if signalLinePrice-macdLinePrice > 0 {
		return true
	}

	return false
}
