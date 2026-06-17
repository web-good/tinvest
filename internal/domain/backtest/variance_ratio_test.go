package backtest

import (
	"math"
	"testing"
)

func approxVR(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSimpleReturns(t *testing.T) {
	r := SimpleReturns([]float64{100, 110, 99})
	if len(r) != 2 || !approxVR(r[0], 0.1) || !approxVR(r[1], -0.1) {
		t.Fatalf("returns = %v, want [0.1 -0.1]", r)
	}
	if len(SimpleReturns([]float64{100})) != 0 {
		t.Fatalf("single close must yield no returns")
	}
}

func TestVarianceRatioMeanReverting(t *testing.T) {
	// Alternating returns: 2-period overlapping sums are ~0 -> VR << 1.
	r := []float64{0.02, -0.02, 0.02, -0.02, 0.02, -0.02, 0.02, -0.02}
	vr := VarianceRatio(r, 2)
	if vr >= 1.0 {
		t.Fatalf("alternating series VR(2) = %.4f, want < 1", vr)
	}
}

func TestVarianceRatioZeroVarGuard(t *testing.T) {
	// Constant returns -> Var_1 == 0 -> guard returns 0, not NaN/Inf.
	r := []float64{0.01, 0.01, 0.01, 0.01, 0.01}
	if vr := VarianceRatio(r, 2); vr != 0 {
		t.Fatalf("zero-variance VR = %.4f, want 0", vr)
	}
}

func TestAutocorr1Alternating(t *testing.T) {
	r := []float64{1, -1, 1, -1, 1, -1}
	if ac := Autocorr1(r); ac >= 0 {
		t.Fatalf("alternating autocorr = %.4f, want negative", ac)
	}
}

func TestMeanReversionVerdict(t *testing.T) {
	if MeanReversionVerdict(0.8) != "mean-reverting" {
		t.Fatalf("0.8 should be mean-reverting")
	}
	if MeanReversionVerdict(1.2) != "trending" {
		t.Fatalf("1.2 should be trending")
	}
	if MeanReversionVerdict(1.0) != "neutral" {
		t.Fatalf("1.0 should be neutral")
	}
}
