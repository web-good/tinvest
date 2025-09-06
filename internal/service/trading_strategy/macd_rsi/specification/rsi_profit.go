package specification

import (
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/utils"
	"tinvest/pkg/client/grpc/model"
)

type RsiProfit struct{}

func (p *RsiProfit) IsSatisfiedBy(itemTechAnalyse []*domain.RSIItemTechAnalyse, purchaseTime time.Time, nowTime time.Time, timeFrame enum.Interval) bool {
	iterLen := 1

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	j := 0

	for i := 0; i < len(itemTechAnalyse)-2; i++ {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i+1]

		if utils.CombinePrice(prevItem.SignalLine.Units, prevItem.SignalLine.Nano) > float64(70) && utils.CombinePrice(item.SignalLine.Units, item.SignalLine.Nano) <= float64(70) {
			candelesCount := nowTime.Sub(purchaseTime)

			if timeFrame == enum.Hour1 && candelesCount.Hours() > 3 {
				return true
			}

			if timeFrame == enum.Day1 && candelesCount.Hours()/24 > 3 {
				return true
			}

			if timeFrame == enum.Minutes15 && candelesCount.Minutes()/15 > 3 {
				return true
			}

			if timeFrame == enum.Hour4 && candelesCount.Hours()/4 > 3 {
				return true
			}

			return false
		}

		if prevItem.SignalLine.Units > 70 && item.SignalLine.Units < 70 {
			return true
		}

		j++
	}

	return false
}

func (p *RsiProfit) IsSatisfiedByProfit(itemTechAnalyse []*domain.RSIItemTechAnalyse, position model.Position) bool {
	price := utils.CombinePrice(position.Price.Units, position.Price.Nano)
	purchasePrice := utils.CombinePrice(position.PurchasePrice.Units, position.PurchasePrice.Nano)

	if price < purchasePrice {
		return false
	}

	if itemTechAnalyse[0].SignalLine.Units >= 70 && itemTechAnalyse[1].SignalLine.Units < 70 ||
		itemTechAnalyse[0].SignalLine.Units >= 75 && itemTechAnalyse[1].SignalLine.Units < 75 ||
		itemTechAnalyse[0].SignalLine.Units >= 80 && itemTechAnalyse[1].SignalLine.Units < 80 ||
		itemTechAnalyse[0].SignalLine.Units >= 85 && itemTechAnalyse[1].SignalLine.Units < 85 {
		return true
	}

	return false
}
