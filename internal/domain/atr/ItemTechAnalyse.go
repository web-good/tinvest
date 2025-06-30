package atr

import "fmt"

type ItemTechAnalyse struct {
	Passed int64
	Value  float64
}

func (item *ItemTechAnalyse) ToString() string {
	return fmt.Sprintf("ATR %.2f (пройдено за день %%%d)", item.Value, item.Passed)
}
