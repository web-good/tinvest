package candles

import (
	"testing"
	"time"

	imodel "tinvest/internal/model"
)

// Quotation в CandleItemTechAnalyse — значение, не указатель (см. reversion/live/service_test.go).
func q(v int64) imodel.Quotation { return imodel.Quotation{Units: v} }

func TestToCandlesDropsIncompleteWhenAsked(t *testing.T) {
	in := []*imodel.CandleItemTechAnalyse{
		{Time: time.Unix(0, 0), Open: q(1), High: q(2), Low: q(1), Close: q(2), Volume: 10, IsComplete: true},
		{Time: time.Unix(1800, 0), Open: q(2), High: q(3), Low: q(2), Close: q(3), Volume: 20, IsComplete: false},
	}
	if got := ToCandles(in, true); len(got) != 1 || got[0].Volume != 10 {
		t.Fatalf("ToCandles(completedOnly) = %+v, want only the completed bar", got)
	}
	if got := ToCandles(in, false); len(got) != 2 {
		t.Fatalf("ToCandles(all) length = %d, want 2", len(got))
	}
}
