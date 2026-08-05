package candles

import (
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	imodel "tinvest/internal/model"
)

// Quotation в CandleItemTechAnalyse — значение, не указатель (см. reversion/live/service_test.go).
func q(units int64, nano int32) imodel.Quotation { return imodel.Quotation{Units: units, Nano: nano} }

func TestToCandlesDropsIncompleteWhenAsked(t *testing.T) {
	in := []*imodel.CandleItemTechAnalyse{
		{Time: time.Unix(0, 0), Open: q(1, 0), High: q(2, 0), Low: q(1, 0), Close: q(2, 0), Volume: 10, IsComplete: true},
		{Time: time.Unix(1800, 0), Open: q(2, 0), High: q(3, 0), Low: q(2, 0), Close: q(3, 0), Volume: 20, IsComplete: false},
	}
	if got := ToCandles(in, true); len(got) != 1 || got[0].Volume != 10 {
		t.Fatalf("ToCandles(completedOnly) = %+v, want only the completed bar", got)
	}
	if got := ToCandles(in, false); len(got) != 2 {
		t.Fatalf("ToCandles(all) length = %d, want 2", len(got))
	}
}

// TestToCandlesCombinesUnitsAndNano pins the price conversion itself: units+nano must
// combine into the fractional float64 price, not just pass Units through. Nano=500000000
// (0.5) exercises the fractional part that a units-only conversion would silently drop —
// a real risk once rsi_pullback reuses this package for instruments like UGLD priced in
// tenths of a ruble.
func TestToCandlesCombinesUnitsAndNano(t *testing.T) {
	ts := time.Unix(0, 0)
	in := []*imodel.CandleItemTechAnalyse{
		{
			Time:       ts,
			Open:       q(100, 500000000), // 100.5
			High:       q(101, 250000000), // 101.25
			Low:        q(99, 750000000),  // 99.75
			Close:      q(100, 0),         // 100.0
			Volume:     42,
			IsComplete: true,
		},
	}
	want := []backtest.Candle{
		{Time: ts, Open: 100.5, High: 101.25, Low: 99.75, Close: 100.0, Volume: 42},
	}

	got := ToCandles(in, true)
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("ToCandles(units+nano) = %+v, want %+v", got, want[0])
	}
}
