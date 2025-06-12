package specification

import (
	"fmt"
	"tinvest/internal/model"
)

type EmaSpecification struct{}

func (s *EmaSpecification) IsSatisfiedBy(itemTechAnalyse []*model.EmaItemTechAnalyse) bool {
	for _, item := range itemTechAnalyse {
		fmt.Println(item)
	}

	return true
}
