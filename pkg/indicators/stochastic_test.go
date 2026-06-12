package indicators

import (
	"math"
	"testing"
)

func TestStochastic_RampFixture(t *testing.T) {
	// Windows of 3 over this series:
	//   [10,12,11] -> hi12 lo10 close11 -> %K = 100*(11-10)/2 = 50
	//   [12,11,13] -> hi13 lo11 close13 -> %K = 100
	//   [11,13,14] -> hi14 lo11 close14 -> %K = 100
	// %D (smooth 3) = (50+100+100)/3 = 83.33
	highs := []float64{10, 12, 11, 13, 14}
	lows := []float64{10, 12, 11, 13, 14}
	closes := []float64{10, 12, 11, 13, 14}

	k, d := Stochastic(highs, lows, closes, 3, 3)
	if math.Abs(k-100) > 0.01 {
		t.Errorf("%%K = %v, want 100", k)
	}
	if math.Abs(d-83.33) > 0.01 {
		t.Errorf("%%D = %v, want 83.33", d)
	}
}

func TestStochastic_TooFewBarsReturnsZero(t *testing.T) {
	k, d := Stochastic([]float64{1, 2}, []float64{1, 2}, []float64{1, 2}, 3, 3)
	if k != 0 || d != 0 {
		t.Errorf("got k=%v d=%v, want 0,0", k, d)
	}
}

func TestStochastic_FlatRangeYieldsZeroK(t *testing.T) {
	// A collapsed high/low range must not divide by zero.
	k, _ := Stochastic([]float64{5, 5, 5}, []float64{5, 5, 5}, []float64{5, 5, 5}, 3, 1)
	if k != 0 {
		t.Errorf("%%K = %v, want 0 on zero range", k)
	}
}

func TestStochastic_FewerThanSmoothReturnsZeroD(t *testing.T) {
	// Exactly one %K value but dSmooth=3 -> %D not computable yet.
	_, d := Stochastic([]float64{10, 12, 11}, []float64{10, 12, 11}, []float64{10, 12, 11}, 3, 3)
	if d != 0 {
		t.Errorf("%%D = %v, want 0 (insufficient %%K history)", d)
	}
}

func TestStochasticSeries_RampFixture(t *testing.T) {
	// Same ramp as TestStochastic_RampFixture. Windows of 3:
	//   %K = [50, 100, 100]; %D(smooth 3) only one full value: (50+100+100)/3 = 83.33
	highs := []float64{10, 12, 11, 13, 14}
	lows := []float64{10, 12, 11, 13, 14}
	closes := []float64{10, 12, 11, 13, 14}

	ks, ds := StochasticSeries(highs, lows, closes, 3, 3)
	if len(ks) != 3 {
		t.Fatalf("len(ks)=%d want 3", len(ks))
	}
	wantKs := []float64{50, 100, 100}
	for i, w := range wantKs {
		if math.Abs(ks[i]-w) > 1e-9 {
			t.Fatalf("ks[%d]=%v want %v", i, ks[i], w)
		}
	}
	if len(ds) != 1 {
		t.Fatalf("len(ds)=%d want 1", len(ds))
	}
	if math.Abs(ds[0]-83.333333) > 1e-4 {
		t.Fatalf("ds[0]=%v want ~83.33", ds[0])
	}
}

func TestStochasticSeries_DSmoothTwoYieldsPrev(t *testing.T) {
	// dSmooth=2 over the ramp gives ds of length 2 so a cross has prev+now.
	highs := []float64{10, 12, 11, 13, 14}
	lows := []float64{10, 12, 11, 13, 14}
	closes := []float64{10, 12, 11, 13, 14}
	_, ds := StochasticSeries(highs, lows, closes, 3, 2)
	// ks=[50,100,100]; ds=[(50+100)/2=75, (100+100)/2=100]
	if len(ds) != 2 || math.Abs(ds[0]-75) > 1e-9 || math.Abs(ds[1]-100) > 1e-9 {
		t.Fatalf("ds=%v want [75 100]", ds)
	}
}

func TestStochasticSeries_InsufficientHistory(t *testing.T) {
	ks, ds := StochasticSeries([]float64{1, 2}, []float64{1, 2}, []float64{1, 2}, 5, 3)
	if ks != nil || ds != nil {
		t.Fatalf("want nil,nil for short history; got ks=%v ds=%v", ks, ds)
	}
}
