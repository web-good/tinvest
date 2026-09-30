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
//
// Table covers all 12 fields diffMarketData compares, not just the 4 the original brief
// spelled out (Closes/DailyCloses/TodayHigh/TodayLow): a field diffMarketData never
// touches would otherwise report a silent "0 расхождений" on real data — exactly the
// failure mode this tool exists to catch (2026-08-06 review, fix round 1).
//
// Same-type, same-length "twin" fields that a copy-paste could plausibly swap
// (Closes/Highs/Lows; DailyCloses/DailyHighs/DailyLows; Price/TodayHigh/TodayLow) start
// EQUAL in base — not just distinguishable from their own mutated value. If a comparison
// line ever reads the wrong source (e.g. diffFloats("DailyHighs", want.DailyHighs,
// got.DailyLows), the reviewer's own example), mutating only the intended field leaves
// the swapped-in source untouched and therefore still equal to its base twin: the buggy
// line reports NO divergence and the test fails, exactly as it must. With base values
// merely distinct (the pre-fix-round-1 shape), such a swap can coincidentally still show
// *some* diff under the right hardcoded label and slip through — which is what made the
// original 4-field table insufficient in the first place.
func TestDiffMarketDataNamesTheDivergingField(t *testing.T) {
	base := strategy.MarketData{
		Price:       2,
		Closes:      []float64{1, 2},
		Highs:       []float64{1, 2},
		Lows:        []float64{1, 2},
		Volumes:     []int64{5, 6},
		Times:       []time.Time{time.Unix(0, 0), time.Unix(1800, 0)},
		DailyCloses: []float64{9},
		DailyHighs:  []float64{9},
		DailyLows:   []float64{9},
		DailyTimes:  []time.Time{time.Unix(0, 0)},
		TodayHigh:   2,
		TodayLow:    2,
	}
	cases := map[string]func(m *strategy.MarketData){
		"Price":       func(m *strategy.MarketData) { m.Price = 99 },
		"Closes":      func(m *strategy.MarketData) { m.Closes = []float64{1, 3} },
		"Highs":       func(m *strategy.MarketData) { m.Highs = []float64{1, 9} },
		"Lows":        func(m *strategy.MarketData) { m.Lows = []float64{1, 8} },
		"Volumes":     func(m *strategy.MarketData) { m.Volumes = []int64{5, 99} },
		"Times":       func(m *strategy.MarketData) { m.Times = []time.Time{time.Unix(0, 0), time.Unix(3600, 0)} },
		"DailyCloses": func(m *strategy.MarketData) { m.DailyCloses = []float64{11} },
		"DailyHighs":  func(m *strategy.MarketData) { m.DailyHighs = []float64{12} },
		"DailyLows":   func(m *strategy.MarketData) { m.DailyLows = []float64{13} },
		"DailyTimes":  func(m *strategy.MarketData) { m.DailyTimes = []time.Time{time.Unix(3600, 0)} },
		"TodayHigh":   func(m *strategy.MarketData) { m.TodayHigh = 50 },
		"TodayLow":    func(m *strategy.MarketData) { m.TodayLow = 0.1 },
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

// TestDiffSignalComparesTradeRelevantFields covers all 5 fields diffSignal compares
// (Kind/Reason/StopLoss/TakeProfit/ATR), not just the 2 the original brief spelled out —
// same rationale as TestDiffMarketDataNamesTheDivergingField above (2026-08-06 review,
// fix round 1): an uncovered comparison line can drift or get its argument swapped
// without any test noticing. StopLoss/TakeProfit/ATR are all float64 "twins" a copy-paste
// could confuse, so — same defense as above — they start EQUAL in base: a swapped source
// (e.g. comparing want.StopLoss against got.ATR) would then see its untouched operand
// still matching, report no divergence, and correctly fail the test.
func TestDiffSignalComparesTradeRelevantFields(t *testing.T) {
	base := model.Signal{Kind: model.SignalBuy, Reason: "RSI50", StopLoss: 1, TakeProfit: 1, ATR: 1}
	if d := diffSignal(base, base); len(d) != 0 {
		t.Fatalf("diffSignal on identical signals = %v, want none", d)
	}

	cases := map[string]func(s *model.Signal){
		"Kind":       func(s *model.Signal) { s.Kind = model.SignalSell },
		"Reason":     func(s *model.Signal) { s.Reason = "SL" },
		"StopLoss":   func(s *model.Signal) { s.StopLoss = 1.1 },
		"TakeProfit": func(s *model.Signal) { s.TakeProfit = 2.2 },
		"ATR":        func(s *model.Signal) { s.ATR = 3.3 },
	}
	for field, mutate := range cases {
		got := base
		mutate(&got)
		d := diffSignal(base, got)
		if len(d) == 0 {
			t.Fatalf("%s: diffSignal found no divergence", field)
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
