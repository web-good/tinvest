package app

import (
	"context"
	"tinvest/pkg/collection"
	"tinvest/pkg/logger"
)

type Collection struct {
	GoldInstruments *collection.InstrumentCollection
	GrowthShare     *collection.InstrumentCollection
}

func (a *App) initCollection(ctx context.Context) error {
	logger.InfoContext(ctx, "Start init list")
	a.collection = &Collection{}
	a.collection.GoldInstruments = initCompanyListForGoldenStrategy()
	a.collection.GrowthShare = initGrowthShare()
	logger.InfoContext(ctx, "End init list")

	return nil
}

func initCompanyListForGoldenStrategy() *collection.InstrumentCollection {
	companyCollection := collection.NewCollection()
	companyCollection.
		Add(
			collection.Instrument{
				ID:        "962e2a95-02a9-4171-abd7-aa198dbe643a",
				RSILength: 8,
				Name:      "Газпром",
			}).
		Add(
			collection.Instrument{
				ID:        "a797f14a-8513-4b84-b15e-a3b98dc4cc00",
				RSILength: 10,
				Name:      "Сургутнефтегаз - прив",
			}).
		Add(
			collection.Instrument{
				ID:        "efdb54d3-2f92-44da-b7a3-8849e96039f6",
				RSILength: 9,
				Name:      "Татнефть - прив",
			}).
		Add(
			collection.Instrument{
				ID:        "fd417230-19cf-4e7b-9623-f7c9ca18ec6b",
				RSILength: 9,
				Name:      "Роснефть",
			}).
		Add(
			collection.Instrument{
				ID:        "02cfdf61-6298-4c0f-a9ca-9cabc82afaf3",
				RSILength: 9,
				Name:      "Лукойл",
			}).
		Add(
			collection.Instrument{
				ID:        "c190ff1f-1447-4227-b543-316332699ca5",
				RSILength: 8,
				Name:      "Сбер Банк - прив",
			}).
		Add(
			collection.Instrument{
				ID:        "fa6aae10-b8d5-48c8-bbfd-d320d925d096",
				RSILength: 11,
				Name:      "Северсталь",
			}).
		Add(
			collection.Instrument{
				ID:        "161eb0d0-aaac-4451-b374-f5d0eeb1b508",
				RSILength: 8,
				Name:      "НЛМК",
			}).
		Add(
			collection.Instrument{
				ID:        "7132b1c9-ee26-4464-b5b5-1046264b61d9",
				RSILength: 9,
				Name:      "ММК",
			}).
		Add(
			collection.Instrument{
				ID:        "9978b56f-782a-4a80-a4b1-a48cbecfd194",
				RSILength: 7,
				Name:      "ФосАгро",
			}).
		Add(
			collection.Instrument{
				ID:        "653d47e9-dbd4-407a-a1c3-47f897df4694",
				RSILength: 9,
				Name:      "Транс нефть",
			}).
		Add(
			collection.Instrument{
				ID:        "1e19953d-01c6-4ecd-a5f4-53ae3ed44029",
				RSILength: 8,
				Name:      "Банк Санкт-Петербург",
			})

	return companyCollection
}

func initGrowthShare() *collection.InstrumentCollection {
	companyCollection := collection.NewCollection()
	companyCollection.
		Add(
			collection.Instrument{
				ID:        "7de75794-a27f-4d81-a39b-492345813822",
				RSILength: 7,
				Name:      "Яндекс",
			}).
		Add(
			collection.Instrument{
				ID:        "10620843-28ce-44e8-80c2-f26ceb1bd3e1",
				RSILength: 7,
				Name:      "Полюс",
			}).
		Add(
			collection.Instrument{
				ID:        "87db07bc-0e02-4e29-90bb-05e8ef791d7b",
				RSILength: 8,
				Name:      "Т-Технологии",
			}).
		Add(
			collection.Instrument{
				ID:        "0da66728-6c30-44c4-9264-df8fac2467ee",
				RSILength: 9,
				Name:      "НОВАТЭК",
			})

	return companyCollection
}
