package dto

import (
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/pkg/collection"
)

type Trade struct {
	Kind           model.StrategyKind
	Interval       enum.Interval
	ShareList      collection.InstrumentCollection
	Scheduler      string
	UseTrendFilter bool
}
