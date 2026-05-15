package golden_x

import "testing"

func TestAlertState_ShouldAlert(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	tests := []struct {
		name string
		tier alertTier
		want bool
	}{
		{"первый Brown — алерт", tierBrown, true},
		{"повторный Brown — нет", tierBrown, false},
		{"переход Brown→Yellow — алерт", tierYellow, true},
		{"повторный Yellow — нет", tierYellow, false},
		{"переход Yellow→Green — алерт", tierGreen, true},
		{"повторный Green — нет", tierGreen, false},
		{"откат Green→None (RSI > 40) — нет (молчим)", tierNone, false},
		{"снова Brown после None — алерт", tierBrown, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.ShouldAlert(id, tc.tier)
			if got != tc.want {
				t.Fatalf("ShouldAlert(%v) = %v, want %v", tc.tier, got, tc.want)
			}
		})
	}
}

func TestAlertState_IndependentShares(t *testing.T) {
	s := newAlertState()
	if !s.ShouldAlert("a", tierBrown) {
		t.Fatal("first Brown for a must alert")
	}
	if !s.ShouldAlert("b", tierBrown) {
		t.Fatal("first Brown for b must alert")
	}
	if s.ShouldAlert("a", tierBrown) {
		t.Fatal("repeat Brown for a must not alert")
	}
}

func TestTierFromRSI(t *testing.T) {
	tests := []struct {
		rsi  float64
		want alertTier
	}{
		{20, tierGreen},
		{30.99, tierGreen},
		{31, tierYellow},
		{34.99, tierYellow},
		{35, tierBrown},
		{40, tierBrown},
		{40.01, tierNone},
		{50, tierNone},
	}
	for _, tc := range tests {
		if got := tierFromRSI(tc.rsi); got != tc.want {
			t.Errorf("tierFromRSI(%v) = %v, want %v", tc.rsi, got, tc.want)
		}
	}
}
