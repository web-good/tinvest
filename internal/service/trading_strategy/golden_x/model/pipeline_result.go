package model

import (
	"tinvest/internal/model"
	"tinvest/pkg/collection"
)

type FetchResult struct {
	Share   collection.Instrument
	Candles []*model.CandleItemTechAnalyse
	Err     error
}

type DetectResult struct {
	Share            collection.Instrument
	Signal           Signal
	FundamentalBonus int
}
