package indicators

import (
	"errors"
	"math"
	"testing"
	"time"
)

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func TestXIRR(t *testing.T) {
	tests := []struct {
		name     string
		flows    []CashFlow
		want     float64 // ignored when wantErr != nil or checkNPV is true
		tol      float64 // ignored when wantErr != nil or checkNPV is true
		wantErr  error
		checkNPV bool // when true, verify NPV(result) ≈ 0 instead of asserting a specific rate
	}{
		{
			// One-year investment: invest 1000, get back 1100 exactly one year later.
			// XIRR = (1100/1000)^(1/1) - 1 = 0.10 exactly.
			name: "simple one-year gain: expect ~10%",
			flows: []CashFlow{
				{Date: date(2025, 1, 1), Amount: -1000},
				{Date: date(2026, 1, 1), Amount: 1100},
			},
			want: 0.10,
			tol:  1e-4,
		},
		{
			// One-year investment with a loss: invest 1000, get back 900.
			// XIRR = (900/1000)^(1/1) - 1 = -0.10 exactly.
			name: "simple one-year loss: expect ~-10%",
			flows: []CashFlow{
				{Date: date(2025, 1, 1), Amount: -1000},
				{Date: date(2026, 1, 1), Amount: 900},
			},
			want: -0.10,
			tol:  1e-4,
		},
		{
			// Multi-flow case:
			//   t=0:  invest 1000  (2025-01-01)
			//   t=6m: invest 500   (2025-07-01, ≈0.4932 years from base)
			//   t=1y: receive 1700 (2026-01-01, ≈1.0 years from base)
			//
			// Reference XIRR computed by Newton's method analytically:
			//   NPV(r) = -1000 + -500/(1+r)^0.4932 + 1700/(1+r)^1.0 = 0
			//
			// Manual bracketing: try r=0.10:
			//   NPV = -1000 - 500/1.04835 - 500/1.10 - ... let's compute precisely:
			//   years_1 = 0 (base)
			//   years_2 = (2025-07-01 - 2025-01-01).Days / 365 = 181/365 ≈ 0.49589
			//   years_3 = (2026-01-01 - 2025-01-01).Days / 365 = 365/365 = 1.0
			//
			// At r = 0.1366 (approx):
			//   -1000 / (1.1366)^0 = -1000
			//   -500 / (1.1366)^0.49589 ≈ -500 / 1.0658 ≈ -469.2
			//   +1700 / (1.1366)^1.0 ≈ +1700 / 1.1366 ≈ +1495.7
			//   NPV ≈ -1000 - 469.2 + 1495.7 ≈ 26.5 (too high, rate must be higher)
			//
			// At r = 0.20:
			//   -500 / (1.20)^0.49589 ≈ -500 / 1.0976 ≈ -455.5
			//   +1700 / 1.20 ≈ 1416.7
			//   NPV ≈ -1000 - 455.5 + 1416.7 ≈ -38.8 (negative)
			//
			// Root is between 0.1366 and 0.20. By bisection midpoint ~0.168.
			// At r=0.168:
			//   -500 / (1.168)^0.49589 ≈ -500/1.0804 ≈ -462.8
			//   +1700 / 1.168 ≈ 1455.5
			//   NPV ≈ -1000 - 462.8 + 1455.5 ≈ -7.3
			//
			// At r=0.155:
			//   (1.155)^0.49589 ≈ exp(0.49589*ln(1.155)) = exp(0.49589*0.14418) ≈ exp(0.07149) ≈ 1.0741
			//   -500/1.0741 ≈ -465.5
			//   1700/1.155 ≈ 1471.9
			//   NPV ≈ -1000 - 465.5 + 1471.9 ≈ 6.4
			//
			// Root ≈ 0.161. We accept tolerance 1e-4 so solver will find it precisely.
			// The reference value for the assertion is set to 0.161 with tolerance 5e-3
			// to allow for the exact day-count difference.
			name: "multi-flow with intermediate deposit",
			flows: []CashFlow{
				{Date: date(2025, 1, 1), Amount: -1000},
				{Date: date(2025, 7, 1), Amount: -500},
				{Date: date(2026, 1, 1), Amount: 1700},
			},
			want: 0.161,
			tol:  5e-3,
		},
		{
			// All negative flows — no root exists.
			name:    "all-negative flows: ErrXIRRSameSign",
			flows:   []CashFlow{{date(2025, 1, 1), -100}, {date(2026, 1, 1), -200}},
			wantErr: ErrXIRRSameSign,
		},
		{
			// All positive flows — no root exists.
			name:    "all-positive flows: ErrXIRRSameSign",
			flows:   []CashFlow{{date(2025, 1, 1), 100}, {date(2026, 1, 1), 200}},
			wantErr: ErrXIRRSameSign,
		},
		{
			name:    "empty slice: ErrXIRRNoFlows",
			flows:   []CashFlow{},
			wantErr: ErrXIRRNoFlows,
		},
		{
			name:    "single flow: ErrXIRRNoFlows",
			flows:   []CashFlow{{date(2025, 1, 1), -1000}},
			wantErr: ErrXIRRNoFlows,
		},
		{
			// Large irregular swings that can cause Newton to overshoot badly
			// (denominator blows up). The bisection fallback should stabilise it.
			// Invest 100k on day 0, get 1 back after 6 months, then 200k after 2 years.
			// The huge initial loss followed by a large win creates a volatile NPV curve.
			// We don't assert a specific rate — just that the returned rate gives NPV ≈ 0.
			name: "large-swing flows: bisection fallback, NPV near zero",
			flows: []CashFlow{
				{Date: date(2025, 1, 1), Amount: -100000},
				{Date: date(2025, 7, 1), Amount: 1},
				{Date: date(2027, 1, 1), Amount: 200000},
			},
			checkNPV: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := XIRR(tc.flows)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("XIRR() error = %v, want %v", err, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("XIRR() unexpected error: %v", err)
			}

			// For flows with a volatile NPV curve, verify the returned rate gives NPV ≈ 0.
			if tc.checkNPV {
				base := tc.flows[0].Date
				n := npv(tc.flows, base, got)
				if math.Abs(n) > 1e-3 {
					t.Fatalf("XIRR(): NPV at returned rate %v = %v, want ~0", got, n)
				}
				return
			}

			if math.Abs(got-tc.want) > tc.tol {
				t.Fatalf("XIRR() = %v, want %v ±%v", got, tc.want, tc.tol)
			}
		})
	}
}
