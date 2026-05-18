package golden_x

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestKForKind(t *testing.T) {
	tests := []struct {
		name string
		kind dto.StrategyKind
		want float64
	}{
		{"Gold (Dividend) -> 2.0", dto.StrategyKindDividend, 2.0},
		{"Growth -> 1.5", dto.StrategyKindGrowth, 1.5},
		{"unknown zero-value defaults to Gold's multiplier", dto.StrategyKind(0), 2.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := kForKind(tc.kind)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("kForKind(%v) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}

func TestStopFromATR(t *testing.T) {
	tests := []struct {
		name      string
		lastClose float64
		atr       float64
		k         float64
		wantPrice float64
		wantPct   float64
		wantEmpty bool
		tol       float64
	}{
		{
			name:      "Gold normal: 2410.5 minus 2.0*30",
			lastClose: 2410.5,
			atr:       30.0,
			k:         2.0,
			wantPrice: 2350.5,
			wantPct:   60.0 / 2410.5 * 100,
			tol:       1e-6,
		},
		{
			name:      "Growth normal: 100 minus 1.5*4",
			lastClose: 100.0,
			atr:       4.0,
			k:         1.5,
			wantPrice: 94.0,
			wantPct:   6.0,
			tol:       1e-9,
		},
		{
			name:      "atr zero -> empty Stop",
			lastClose: 100.0,
			atr:       0.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "atr negative -> empty Stop",
			lastClose: 100.0,
			atr:       -1.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "lastClose zero -> empty Stop",
			lastClose: 0.0,
			atr:       4.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "stop price would be <= 0 (k*atr exceeds close) -> empty Stop",
			lastClose: 5.0,
			atr:       4.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "stop price exactly zero -> empty Stop",
			lastClose: 8.0,
			atr:       4.0,
			k:         2.0,
			wantEmpty: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stopFromATR(tc.lastClose, tc.atr, tc.k)
			if tc.wantEmpty {
				if got != (dto.Stop{}) {
					t.Fatalf("expected empty Stop{}, got %+v", got)
				}
				return
			}
			if math.Abs(got.Price-tc.wantPrice) > tc.tol {
				t.Fatalf("Price = %v, want %v", got.Price, tc.wantPrice)
			}
			if math.Abs(got.DistancePct-tc.wantPct) > tc.tol {
				t.Fatalf("DistancePct = %v, want %v", got.DistancePct, tc.wantPct)
			}
		})
	}
}
