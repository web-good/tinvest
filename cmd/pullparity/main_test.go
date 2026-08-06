package main

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestDiffMarketDataFindsNothingWhenIdentical(t *testing.T) {
	md := strategy.MarketData{
		Price: 1, Closes: []float64{1, 2}, Highs: []float64{1, 2}, Lows: []float64{1, 2},
		Volumes: []int64{5, 6}, Times: []time.Time{time.Unix(0, 0), time.Unix(1800, 0)},
		DailyCloses: []float64{9}, DailyHighs: []float64{9}, DailyLows: []float64{8},
		DailyTimes: []time.Time{time.Unix(0, 0)}, TodayHigh: 2, TodayLow: 1,
	}
	if d := diffMarketData(md, md); len(d) != 0 {
		t.Fatalf("diffMarketData on identical snapshots = %v, want none", d)
	}
}

// Именно эти поля рвутся при неверной сборке: смещение окна, невидимая дневная свеча,
// посчитанный по-своему диапазон дня. Каждое обязано быть названо в отчёте.
func TestDiffMarketDataNamesTheDivergingField(t *testing.T) {
	base := strategy.MarketData{
		Closes: []float64{1, 2}, DailyCloses: []float64{9}, TodayHigh: 2, TodayLow: 1,
	}
	cases := map[string]func(m *strategy.MarketData){
		"Closes":      func(m *strategy.MarketData) { m.Closes = []float64{1, 3} },
		"DailyCloses": func(m *strategy.MarketData) { m.DailyCloses = []float64{9, 10} },
		"TodayHigh":   func(m *strategy.MarketData) { m.TodayHigh = 5 },
		"TodayLow":    func(m *strategy.MarketData) { m.TodayLow = 0.5 },
	}
	for field, mutate := range cases {
		got := base
		mutate(&got)
		d := diffMarketData(base, got)
		if len(d) == 0 {
			t.Fatalf("%s: diffMarketData found no divergence", field)
		}
		found := false
		for _, line := range d {
			if contains(line, field) {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: diff = %v, want the field named", field, d)
		}
	}
}

func TestDiffSignalComparesTradeRelevantFields(t *testing.T) {
	a := model.Signal{Kind: model.SignalBuy, Reason: "", StopLoss: 1, TakeProfit: 2, ATR: 0.5}
	if d := diffSignal(a, a); len(d) != 0 {
		t.Fatalf("diffSignal on identical signals = %v, want none", d)
	}
	b := a
	b.StopLoss = 1.1
	if d := diffSignal(a, b); len(d) == 0 {
		t.Fatal("diffSignal missed a StopLoss divergence")
	}
	c := a
	c.Kind = model.SignalSell
	if d := diffSignal(a, c); len(d) == 0 {
		t.Fatal("diffSignal missed a Kind divergence")
	}
}
