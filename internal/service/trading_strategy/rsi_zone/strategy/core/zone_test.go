package core

import (
	"reflect"
	"testing"

	"tinvest/pkg/indicators"
)

func TestStochDSeriesAlignsToBars(t *testing.T) {
	n := 30
	highs, lows, closes := make([]float64, n), make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		c := 100 + float64(i%7) - float64(i%3)
		closes[i], highs[i], lows[i] = c, c+1, c-1
	}
	const k, d = 5, 3
	series, warm, ok := stochDSeries(highs, lows, closes, k, d)
	if !ok {
		t.Fatal("ok = false on a valid series")
	}
	if len(series) != n || warm != k+d-2 {
		t.Fatalf("len = %d, warm = %d; want %d, %d", len(series), warm, n, k+d-2)
	}
	_, ds := indicators.StochasticSeries(highs, lows, closes, k, d)
	if !reflect.DeepEqual(series[warm:], ds) {
		t.Fatalf("series[warm:] = %v, want the raw %%D %v", series[warm:], ds)
	}
	for b := 0; b < warm; b++ {
		if series[b] != 0 {
			t.Fatalf("warm-up bar %d = %v, want the unset 0", b, series[b])
		}
	}
	// d = 1 is raw %K: warm = k-1.
	if _, w, ok := stochDSeries(highs, lows, closes, k, 1); !ok || w != k-1 {
		t.Fatalf("d=1: warm = %d ok = %v, want %d true", w, ok, k-1)
	}
}

func TestStochDSeriesRefuses(t *testing.T) {
	h, l, c := []float64{2, 2, 2}, []float64{1, 1, 1}, []float64{1.5, 1.5, 1.5}
	for _, kd := range [][2]int{{0, 3}, {3, 0}, {-1, 3}, {3, -1}, {5, 1}} {
		if _, _, ok := stochDSeries(h, l, c, kd[0], kd[1]); ok {
			t.Errorf("k=%d d=%d: ok = true, want refusal", kd[0], kd[1])
		}
	}
}

// Review Focus 2: a collapsed high/low range makes StochasticSeries report %K = 0, which then
// reads as in-zone. Inherited from reversion; pinned so it never changes silently.
func TestStochDSeriesFlatRangeReadsAsZero(t *testing.T) {
	n := 10
	flat := make([]float64, n)
	for i := range flat {
		flat[i] = 100
	}
	series, warm, ok := stochDSeries(flat, flat, flat, 3, 1)
	if !ok {
		t.Fatal("ok = false")
	}
	if !(series[n-1] == 0 && seenInZone(oscillator{series, warm, 20}, n-1, 1) == n-1) {
		t.Fatalf("flat range: %%D = %v, in zone = %v; want 0 and in zone", series[n-1], seenInZone(oscillator{series, warm, 20}, n-1, 1))
	}
}

func TestZoneTrigger(t *testing.T) {
	// Series are hand-built; warm 0 means every index is a genuine reading. RSI band 25, Stoch 20.
	osc := func(s []float64, warm int, level float64) oscillator { return oscillator{s, warm, level} }
	cases := []struct {
		name      string
		rsi       oscillator
		stoch     oscillator
		i, window int
		wantOK    bool
		wantBy    string
		wantAt    int
	}{
		{"RSI cross, stoch in zone at window's oldest bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 10, 30, 30}, 0, 20), 5, 3, true, "RSI", 3},
		{"RSI cross, stoch in zone one bar before the window",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 10, 30, 30, 30}, 0, 20), 5, 3, false, "", 0},
		{"stoch cross, RSI in zone without its own cross",
			osc([]float64{50, 50, 50, 20, 18, 15}, 0, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 2, true, "Stoch", 5},
		{"both in zone, nobody crosses now",
			osc([]float64{50, 50, 50, 20, 18, 15}, 0, 25), osc([]float64{50, 50, 50, 15, 12, 10}, 0, 20), 5, 5, false, "", 0},
		{"window 1: both in zone on the cross bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 1, true, "RSI", 5},
		{"window 1: stoch in zone only on the previous bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 50, 10, 30}, 0, 20), 5, 1, false, "", 0},
		// Review Focus 4: one entry, reported as the RSI trigger.
		{"both cross on the same bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 3, true, "RSI", 5},
		{"stoch warm-up zero does not confirm",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{0, 0, 0, 0, 30, 30}, 4, 20), 5, 5, false, "", 0},
		{"genuine post-warm-up stoch zero confirms",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{0, 0, 0, 0, 0, 30}, 4, 20), 5, 5, true, "RSI", 4},
		{"RSI warm-up zero does not confirm a stoch cross",
			osc([]float64{0, 0, 0, 0, 30, 30}, 4, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 5, false, "", 0},
		// Review Focus 3: the window reaches before bar 0; only existing bars count, no panic.
		{"window clipped at series start",
			osc([]float64{30, 20}, 0, 25), osc([]float64{10, 30}, 0, 20), 1, 5, true, "RSI", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hit, ok := zoneTrigger(c.rsi, c.stoch, c.i, c.window)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v (hit %+v)", ok, c.wantOK, hit)
			}
			if ok && (hit.by != c.wantBy || hit.otherAt != c.wantAt) {
				t.Fatalf("hit = %+v, want by %q at %d", hit, c.wantBy, c.wantAt)
			}
		})
	}
}
