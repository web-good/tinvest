package live

import "testing"

func TestStrategyFor(t *testing.T) {
	st, ok := StrategyFor("UGLD")
	if !ok {
		t.Fatal("UGLD should be registered")
	}
	if st.Ticker() != "UGLD" {
		t.Fatalf("Ticker() = %q, want UGLD", st.Ticker())
	}
	if _, ok := StrategyFor("NOSUCH"); ok {
		t.Fatal("unknown ticker must return ok=false")
	}
}

func TestMaxHTFTrendEMA(t *testing.T) {
	// NVTK has HTFTrendEMA=150; UGLD/EUTR have 0.
	if got := MaxHTFTrendEMA([]string{"UGLD", "EUTR"}); got != 0 {
		t.Fatalf("MaxHTFTrendEMA(UGLD,EUTR) = %d, want 0", got)
	}
	if got := MaxHTFTrendEMA([]string{"UGLD", "NVTK"}); got != 150 {
		t.Fatalf("MaxHTFTrendEMA(UGLD,NVTK) = %d, want 150", got)
	}
}
