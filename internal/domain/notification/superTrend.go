package notification

import (
	"tinvest/internal/domain/atr"
	"tinvest/internal/domain/share"
	"tinvest/internal/model"
)

type Color int

const (
	Green Color = iota
	Yellow
)

type SuperTrend struct {
	Share     model.Share
	Atr       atr.ItemTechAnalyse
	Indicator Color
}

type TakeProfit struct {
	Share share.Share
}
