package core

import (
	"testing"
	"tinvest/internal/service/trading_strategy/scalping/model"
)

func TestRequireEntryAboveFVGLower(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled int
		close   float64
		buy     bool
	}{
		{"disabled below", 0, 99, true}, {"enabled below", 1, 99, false},
		{"at boundary", 1, 100, true}, {"above boundary", 1, 101, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.StochPeriod = 2
			p.RequireEntryAboveFVGLower = tc.enabled
			bars := append(formation(), bar{103, 120, 98, tc.close})
			sig := NewWithParams("TEST", step, p).Decide(snapshot(bars...))
			if (sig.Kind == model.SignalBuy) != tc.buy {
				t.Fatalf("unexpected signal: %+v", sig)
			}
			if tc.buy {
				want := float64(0)
				if tc.enabled == 1 {
					want = 100
				}
				if sig.MinEntryPrice != want {
					t.Fatalf("floor=%v, want %v", sig.MinEntryPrice, want)
				}
			}
		})
	}
}

func TestRequireEntryAboveFVGLowerValidation(t *testing.T) {
	for _, value := range []int{-1, 2} {
		p := testParams()
		p.RequireEntryAboveFVGLower = value
		if ValidateParams(p) == nil {
			t.Fatalf("accepted %d", value)
		}
	}
}
