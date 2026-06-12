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
