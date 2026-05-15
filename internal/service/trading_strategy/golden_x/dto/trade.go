package dto

import (
	"tinvest/internal/enum"
	"tinvest/pkg/collection"
)

type Trade struct {
	Kind      StrategyKind
	Interval  enum.Interval
	ShareList collection.InstrumentCollection
	Scheduler string
}
