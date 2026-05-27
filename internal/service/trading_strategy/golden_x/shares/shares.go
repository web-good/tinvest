// Package shares is the source-of-truth share list for the Golden X
// strategy. Importable from prod (cmd) without dragging in package app.
package shares

import "tinvest/pkg/collection"

// Dividend returns the curated Gold (long-hold dividend) share list.
// Per-share RSILength tracks each ticker's empirically chosen smoothing.
func Dividend() *collection.InstrumentCollection {
	c := collection.NewCollection()
	c.
		Add(collection.Instrument{ID: "a797f14a-8513-4b84-b15e-a3b98dc4cc00", RSILength: 10, Name: "Сургутнефтегаз - прив", Sector: "Нефть"}).
		Add(collection.Instrument{ID: "efdb54d3-2f92-44da-b7a3-8849e96039f6", RSILength: 9, Name: "Татнефть - прив", Sector: "Нефть"}).
		Add(collection.Instrument{ID: "fd417230-19cf-4e7b-9623-f7c9ca18ec6b", RSILength: 9, Name: "Роснефть", Sector: "Нефть"}).
		Add(collection.Instrument{ID: "02cfdf61-6298-4c0f-a9ca-9cabc82afaf3", RSILength: 9, Name: "Лукойл", Sector: "Нефть"}).
		Add(collection.Instrument{ID: "c190ff1f-1447-4227-b543-316332699ca5", RSILength: 8, Name: "Сбер Банк - прив", Sector: "Банки"}).
		Add(collection.Instrument{ID: "fa6aae10-b8d5-48c8-bbfd-d320d925d096", RSILength: 11, Name: "Северсталь", Sector: "Металлы"}).
		Add(collection.Instrument{ID: "161eb0d0-aaac-4451-b374-f5d0eeb1b508", RSILength: 8, Name: "НЛМК", Sector: "Металлы"}).
		Add(collection.Instrument{ID: "7132b1c9-ee26-4464-b5b5-1046264b61d9", RSILength: 9, Name: "ММК", Sector: "Металлы"}).
		Add(collection.Instrument{ID: "9978b56f-782a-4a80-a4b1-a48cbecfd194", RSILength: 7, Name: "ФосАгро", Sector: "Химия"}).
		Add(collection.Instrument{ID: "653d47e9-dbd4-407a-a1c3-47f897df4694", RSILength: 9, Name: "Транс нефть", Sector: "Нефть"}).
		Add(collection.Instrument{ID: "1e19953d-01c6-4ecd-a5f4-53ae3ed44029", RSILength: 8, Name: "Банк Санкт-Петербург", Sector: "Банки"})
	return c
}

// Growth returns the curated growth share list (single-sell-tier strategy).
func Growth() *collection.InstrumentCollection {
	c := collection.NewCollection()
	c.
		Add(collection.Instrument{ID: "0d53d29a-3794-41c6-ba72-556d46bacb46", RSILength: 7, Name: "Мать и дитя", Sector: "Здоровье"}).
		Add(collection.Instrument{ID: "962e2a95-02a9-4171-abd7-aa198dbe643a", RSILength: 8, Name: "Газпром", Sector: "Энергетика"}).
		Add(collection.Instrument{ID: "7de75794-a27f-4d81-a39b-492345813822", RSILength: 7, Name: "Яндекс", Sector: "IT"}).
		Add(collection.Instrument{ID: "10620843-28ce-44e8-80c2-f26ceb1bd3e1", RSILength: 7, Name: "Полюс", Sector: "Золото"}).
		Add(collection.Instrument{ID: "87db07bc-0e02-4e29-90bb-05e8ef791d7b", RSILength: 8, Name: "Т-Технологии", Sector: "IT"}).
		Add(collection.Instrument{ID: "0da66728-6c30-44c4-9264-df8fac2467ee", RSILength: 9, Name: "НОВАТЭК", Sector: "Энергетика"})
	return c
}
