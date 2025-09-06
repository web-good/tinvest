package specification

import (
	"tinvest/internal/domain"
	"tinvest/internal/domain/atr"
	"tinvest/internal/utils"
	"tinvest/pkg/client/grpc/model"
)

type ProfitEqualsAtr struct {
	ATRProfit float64
}

func (p *ProfitEqualsAtr) IsSatisfiedBy(atr atr.ItemTechAnalyse, position model.Position, rsi *domain.RSIItemTechAnalyse) bool {
	price := utils.CombinePrice(position.Price.Units, position.Price.Nano)
	profitPrice := utils.CombinePrice(position.PurchasePrice.Units, position.PurchasePrice.Nano) + (atr.Value * p.ATRProfit)

	if price >= profitPrice && rsi.SignalLine.Units < 70 {
		return true
	}

	return false
}
