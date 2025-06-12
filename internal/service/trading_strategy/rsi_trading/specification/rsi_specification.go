package specification

import (
	"fmt"
	"tinvest/internal/model"
)

type RsiSpecification struct {
	//purchaseValue int8
}

func (s *RsiSpecification) IsSatisfiedBy(itemTechAnalyse []*model.RsiItemTechAnalyse) bool {
	iterLen := 3

	if len(itemTechAnalyse) <= iterLen {
		return false
	}

	j := 0

	for i := len(itemTechAnalyse) - 1; j < iterLen; i-- {
		item := itemTechAnalyse[i]
		prevItem := itemTechAnalyse[i-1]

		if prevItem.SignalLine.Units < int64(30) && item.SignalLine.Units >= int64(30) {
			fmt.Println(item)
			return true
		}

		j++
	}

	return false
}
