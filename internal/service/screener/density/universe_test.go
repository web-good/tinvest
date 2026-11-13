package density

import (
	"fmt"
	"testing"
	"time"
)

func TestSelectUniverse(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]DailyCandle, 10)
	for i := range candles {
		candles[i] = DailyCandle{Time: now.Add(-time.Duration(i+1) * 24 * time.Hour), Close: Price{Units: 100}, Volume: 100_000, Complete: true}
	}
	input := UniverseInput{Instrument: Instrument{ID: "b", Ticker: "B", Lot: 10}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true, Candles: candles}
	other := input
	other.Instrument.ID = "a"
	closed := input
	closed.Instrument.ID = "closed"
	closed.TradingOpen = false
	invalid := input
	invalid.Instrument.ID = "foreign"
	invalid.Currency = "usd"
	short := input
	short.Instrument.ID = "short"
	short.Candles = candles[:9]
	got := SelectUniverse([]UniverseInput{input, other, closed, invalid, short}, DefaultConfig())
	if len(got) != 3 || got[0].ID != "a" || got[1].ID != "b" || got[2].ID != "closed" {
		t.Fatalf("unexpected universe: %+v", got)
	}
}

func TestSelectUniverseFiltersAndLastTwenty(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]DailyCandle, 21)
	for i := range candles {
		candles[i] = DailyCandle{Time: now.Add(-time.Duration(i+1) * 24 * time.Hour), Close: Price{Units: 100}, Volume: 100_000, Complete: true}
	}
	candles[20].Volume = 1
	base := UniverseInput{Instrument: Instrument{ID: "x", Ticker: "X", Lot: 10}, Currency: "RUB", Exchange: "MOEX", BuyAvailable: true, Candles: candles}
	for _, tc := range []struct {
		name   string
		change func(*UniverseInput)
		want   int
	}{
		{"last twenty", func(*UniverseInput) {}, 1},
		{"closed session", func(in *UniverseInput) { in.TradingOpen = false }, 1},
		{"not buyable", func(in *UniverseInput) { in.BuyAvailable = false }, 0},
		{"wrong exchange", func(in *UniverseInput) { in.Exchange = "other" }, 0},
		{"expensive lot", func(in *UniverseInput) { in.Instrument.Lot = 301 }, 0},
		{"incomplete", func(in *UniverseInput) {
			in.Candles = append([]DailyCandle(nil), in.Candles[:10]...)
			in.Candles[0].Complete = false
		}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.change(&in)
			if got := SelectUniverse([]UniverseInput{in}, DefaultConfig()); len(got) != tc.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}

func TestSelectUniverseLimitsAndOrdersByTurnover(t *testing.T) {
	candles := make([]DailyCandle, 10)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range candles {
		candles[i] = DailyCandle{Time: now.Add(-time.Duration(i+1) * 24 * time.Hour), Close: Price{Units: 100}, Volume: 100000, Complete: true}
	}
	inputs := make([]UniverseInput, 81)
	for i := range inputs {
		inputs[i] = UniverseInput{Instrument: Instrument{ID: fmt.Sprintf("%03d", 80-i), Ticker: "T", Lot: 10}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true, Candles: candles}
	}
	got := SelectUniverse(inputs, DefaultConfig())
	if len(got) != 80 || got[0].ID != "000" || got[79].ID != "079" {
		t.Fatalf("top universe: %+v", got)
	}
}
