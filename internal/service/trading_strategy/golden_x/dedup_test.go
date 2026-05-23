package golden_x

import "testing"

func TestAlertState_ShouldAlert(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	tests := []struct {
		name string
		snap alertSnapshot
		want bool
	}{
		{"первый Yellow — алерт", alertSnapshot{tier: tierYellow}, true},
		{"повторный Yellow — нет", alertSnapshot{tier: tierYellow}, false},
		{"переход Yellow→Green — алерт", alertSnapshot{tier: tierGreen}, true},
		{"повторный Green — нет", alertSnapshot{tier: tierGreen}, false},
		{"откат Green→None (RSI выше p15) — нет (молчим)", alertSnapshot{tier: tierNone}, false},
		{"снова Yellow после None — алерт", alertSnapshot{tier: tierYellow}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.ShouldAlert(id, tc.snap)
			if got != tc.want {
				t.Fatalf("ShouldAlert(%+v) = %v, want %v", tc.snap, got, tc.want)
			}
		})
	}
}

func TestAlertState_IndependentShares(t *testing.T) {
	s := newAlertState()
	if !s.ShouldAlert("a", alertSnapshot{tier: tierYellow}) {
		t.Fatal("first Yellow for a must alert")
	}
	if !s.ShouldAlert("b", alertSnapshot{tier: tierYellow}) {
		t.Fatal("first Yellow for b must alert")
	}
	if s.ShouldAlert("a", alertSnapshot{tier: tierYellow}) {
		t.Fatal("repeat Yellow for a must not alert")
	}
}

func TestAlertState_BuyToSellTransition(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	if !s.ShouldAlert(id, alertSnapshot{tier: tierYellow}) {
		t.Fatal("first Yellow must alert")
	}
	if s.ShouldAlert(id, alertSnapshot{tier: tierNone}) {
		t.Fatal("None must not alert (silent reset)")
	}
	if !s.ShouldAlert(id, alertSnapshot{tier: tierSellYellow}) {
		t.Fatal("SellYellow after reset must alert")
	}
	if s.ShouldAlert(id, alertSnapshot{tier: tierSellYellow}) {
		t.Fatal("repeat SellYellow must not alert")
	}
	if !s.ShouldAlert(id, alertSnapshot{tier: tierSellRed}) {
		t.Fatal("escalation SellYellow→SellRed must alert")
	}
	if s.ShouldAlert(id, alertSnapshot{tier: tierNone}) {
		t.Fatal("trailing None must not alert")
	}
}

func TestAlertState_TrendImprovement(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	if !s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: false}) {
		t.Fatal("first Yellow must alert")
	}
	if !s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: true}) {
		t.Fatal("trend improvement (false→true) at same tier must alert")
	}
	if s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: true}) {
		t.Fatal("repeat with same trend must not alert")
	}
}

func TestAlertState_DivergenceImprovement(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	if !s.ShouldAlert(id, alertSnapshot{tier: tierGreen}) {
		t.Fatal("first Green must alert")
	}
	if !s.ShouldAlert(id, alertSnapshot{tier: tierGreen, divergence: true}) {
		t.Fatal("divergence improvement at same tier must alert")
	}
	if s.ShouldAlert(id, alertSnapshot{tier: tierGreen, divergence: true}) {
		t.Fatal("repeat with same divergence must not alert")
	}
}

func TestAlertState_VolumeImprovement(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	if !s.ShouldAlert(id, alertSnapshot{tier: tierYellow}) {
		t.Fatal("first Yellow must alert")
	}
	if !s.ShouldAlert(id, alertSnapshot{tier: tierYellow, volumeOK: true}) {
		t.Fatal("volume improvement at same tier must alert")
	}
	if s.ShouldAlert(id, alertSnapshot{tier: tierYellow, volumeOK: true}) {
		t.Fatal("repeat with same volume must not alert")
	}
}

func TestAlertState_NoAlertOnRegression(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: true, divergence: true, volumeOK: true})
	if s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: false, divergence: false, volumeOK: false}) {
		t.Fatal("regression (true→false) must not alert")
	}
}

func TestAlertState_ReimprovementAfterRegression(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: true})
	s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: false})
	if !s.ShouldAlert(id, alertSnapshot{tier: tierYellow, trendOK: true}) {
		t.Fatal("re-improvement after regression must alert")
	}
}

func TestAlertState_NoEnrichmentAlertOnTierNone(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	if s.ShouldAlert(id, alertSnapshot{tier: tierNone, trendOK: true, divergence: true, volumeOK: true}) {
		t.Fatal("enrichments at tierNone must not alert")
	}
}
