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
		{"первый Yellow — алерт", tierYellow, true},
		{"повторный Yellow — нет", tierYellow, false},
		{"переход Yellow→Green — алерт", tierGreen, true},
		{"повторный Green — нет", tierGreen, false},
		{"откат Green→None (RSI выше p15) — нет (молчим)", tierNone, false},
		{"снова Yellow после None — алерт", tierYellow, true},
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
	if !s.ShouldAlert("a", tierYellow) {
		t.Fatal("first Yellow for a must alert")
	}
	if !s.ShouldAlert("b", tierYellow) {
		t.Fatal("first Yellow for b must alert")
	}
	if s.ShouldAlert("a", tierYellow) {
		t.Fatal("repeat Yellow for a must not alert")
	}
}

func TestAlertState_BuyToSellTransition(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	// Yellow buy fires → moves through None (RSI in mid-zone) →
	// SellYellow fires → SellRed fires → silent reset.
	if !s.ShouldAlert(id, tierYellow) {
		t.Fatal("first Yellow must alert")
	}
	if s.ShouldAlert(id, tierNone) {
		t.Fatal("None must not alert (silent reset)")
	}
	if !s.ShouldAlert(id, tierSellYellow) {
		t.Fatal("SellYellow after reset must alert")
	}
	if s.ShouldAlert(id, tierSellYellow) {
		t.Fatal("repeat SellYellow must not alert")
	}
	if !s.ShouldAlert(id, tierSellRed) {
		t.Fatal("escalation SellYellow→SellRed must alert")
	}
	if s.ShouldAlert(id, tierNone) {
		t.Fatal("trailing None must not alert")
	}
}
