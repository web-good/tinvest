package specification

import (
	"tinvest/internal/domain/atr"
	"tinvest/internal/utils"
	"tinvest/pkg/client/grpc/model"
)

type ProfitEqualsAtr struct{}

func (p *ProfitEqualsAtr) IsSatisfiedBy(atr atr.ItemTechAnalyse, position model.Position) bool {
	price := utils.CombinePrice(position.Price.Units, position.Price.Nano)
	proc80 := (atr.Value * 80) / 100
	purchasePrice := utils.CombinePrice(position.PurchasePrice.Units, position.PurchasePrice.Nano)

	if price >= purchasePrice+proc80 {
		return true
	}

	return false
}
