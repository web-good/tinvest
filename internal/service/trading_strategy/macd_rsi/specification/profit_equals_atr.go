package specification

import (
	"tinvest/internal/domain/atr"
	"tinvest/internal/utils"
	"tinvest/pkg/client/grpc/model"
)

type ProfitEqualsAtr struct{}

func (p *ProfitEqualsAtr) IsSatisfiedBy(atr atr.ItemTechAnalyse, position model.Position) bool {
	price := utils.CombinePrice(position.Price.Units, position.Price.Nano)
	profitPrice := utils.CombinePrice(position.PurchasePrice.Units, position.PurchasePrice.Nano) + (atr.Value * 1.5)

	if price >= profitPrice {
		return true
	}

	return false
}
