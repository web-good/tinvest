package specification

import "tinvest/internal/model"

type RsiProfit struct{}

func (p *RsiProfit) IsSatisfiedBy(itemTechAnalyse []*model.RsiItemTechAnalyse) bool {

	return false
}
