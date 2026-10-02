package backtest

import (
	"testing"
	"time"
)

// TestBuildMarketDataCarriesOpens pins that the strategy snapshot exposes bar opens aligned with
// closes: gap strategies measure the day's first open against the previous close.
func TestBuildMarketDataCarriesOpens(t *testing.T) {
	w := []Candle{
		{Time: time.Unix(0, 0), Open: 1, High: 3, Low: 0.5, Close: 2},
		{Time: time.Unix(1800, 0), Open: 2.5, High: 4, Low: 2, Close: 3},
	}
	md := buildMarketData(w)
	if len(md.Opens) != len(md.Closes) || md.Opens[0] != 1 || md.Opens[1] != 2.5 {
		t.Fatalf("Opens = %v, want [1 2.5] aligned with Closes %v", md.Opens, md.Closes)
	}
}
