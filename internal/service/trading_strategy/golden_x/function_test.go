package golden_x

import (
	"math"
	"testing"

	"tinvest/internal/domain/ema"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
)

func TestComputeEMA(t *testing.T) {
	// Period=3 EMA over the sequence [1,2,3,4,5,6]:
	// seed (pos 2) = (1+2+3)/3 = 2
	// multiplier  = 2/(3+1) = 0.5
	// pos 3: (4-2)*0.5 + 2 = 3
	// pos 4: (5-3)*0.5 + 3 = 4
	// pos 5: (6-4)*0.5 + 4 = 5
	got := ema.Compute([]float64{1, 2, 3, 4, 5, 6}, 3)
	want := []float64{0, 0, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("pos %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComputeEMA_InsufficientReturnsZeroes(t *testing.T) {
	got := ema.Compute([]float64{1, 2}, 3)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for i, v := range got {
		if v != 0 {
			t.Errorf("pos %d: got %v, want 0", i, v)
		}
	}
}

func TestTrendStatus_Mark(t *testing.T) {
	tests := []struct {
		status gxmodel.TrendStatus
		want   string
	}{
		{gxmodel.TrendWith, "✅"},
		{gxmodel.TrendAgainst, "🚫"},
		{gxmodel.TrendUnknown, ""},
	}
	for _, tc := range tests {
		if got := tc.status.Mark(); got != tc.want {
			t.Errorf("status %v: got %q, want %q", tc.status, got, tc.want)
		}
	}
}
