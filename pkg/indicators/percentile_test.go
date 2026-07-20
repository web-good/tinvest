package indicators

import (
	"math"
	"testing"
)

func TestPercentile_R7(t *testing.T) {
	s := []float64{1, 2, 3, 4}
	if got := Percentile(s, 50); math.Abs(got-2.5) > 1e-9 {
		t.Fatalf("p50 = %v, want 2.5", got)
	}
	if got := Percentile(s, 0); got != 1 {
		t.Fatalf("p0 = %v, want 1", got)
	}
	if got := Percentile(s, 100); got != 4 {
		t.Fatalf("p100 = %v, want 4", got)
	}
	if got := Percentile(nil, 50); got != 0 {
		t.Fatalf("empty = %v, want 0", got)
	}
}

func TestPercentileRank(t *testing.T) {
	vals := []float64{10, 20, 30, 40}
	if got := PercentileRank(vals, 25); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("rank(25) = %v, want 0.5", got)
	}
	if got := PercentileRank(vals, 5); got != 0 {
		t.Fatalf("rank(5) = %v, want 0", got)
	}
	if got := PercentileRank(vals, 100); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("rank(100) = %v, want 1", got)
	}
}
