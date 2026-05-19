package app

import (
	"context"

	"tinvest/internal/service/trading_strategy/golden_x/shares"
	"tinvest/pkg/collection"
	"tinvest/pkg/logger"
)

type Collection struct {
	GoldInstruments *collection.InstrumentCollection
	GrowthShare     *collection.InstrumentCollection
}

func (a *App) initCollection(ctx context.Context) error {
	logger.InfoContext(ctx, "Start init list")
	a.collection = &Collection{
		GoldInstruments: shares.Dividend(),
		GrowthShare:     shares.Growth(),
	}
	logger.InfoContext(ctx, "End init list")
	return nil
}
