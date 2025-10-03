package specification

import (
	"tinvest/internal/domain"
	"tinvest/internal/utils"
)

type Rsi struct {
	Value int16
}

func (r *Rsi) IsSatisfiedBy(rsi domain.RSIItemTechAnalyse) bool {
	if utils.CombinePrice(rsi.SignalLine.Units, rsi.SignalLine.Nano) < float64(r.Value) {
		return true
	}

	return false
}
