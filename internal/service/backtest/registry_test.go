package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

func TestLookupKnownAndUnknown(t *testing.T) {
	if _, ok := Lookup("NOPE"); ok {
		t.Fatal("expected unknown ticker to miss")
	}
	for _, tk := range []string{"RUAL", "AFKS"} {
		b, ok := Lookup(tk)
		if !ok {
			t.Fatalf("expected %s binding", tk)
		}
		if b.DefaultParams == nil || b.Build == nil || b.ParseParams == nil {
			t.Fatalf("%s binding has nil funcs", tk)
		}
	}
}

func TestBindingsBuildStrategy(t *testing.T) {
	for _, tk := range []string{"RUAL", "AFKS"} {
		b, _ := Lookup(tk)
		s := b.Build(b.DefaultParams())
		if s.Ticker() != tk {
			t.Fatalf("built strategy ticker = %q, want %s", s.Ticker(), tk)
		}
	}
}

func TestParseParamsOverridesDefaults(t *testing.T) {
	b, _ := Lookup("RUAL")
	raw := []byte(`{"EMAPeriod": 50}`)
	parsed, err := b.ParseParams(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := parsed.(adaptive.Params)
	if p.EMAPeriod != 50 {
		t.Fatalf("EMAPeriod = %d, want 50 (override)", p.EMAPeriod)
	}
	if p.ADXPeriod != rusal.DefaultParams().ADXPeriod {
		t.Fatal("non-overridden field should keep its default")
	}
}

func TestParamRows(t *testing.T) {
	rows := ParamRows(rusal.DefaultParams())
	if len(rows) == 0 {
		t.Fatal("expected param rows")
	}
	var found bool
	for _, r := range rows {
		if r.Name == "EMAPeriod" && r.Value == "21" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected EMAPeriod=21 row")
	}
}

func TestLookupOrGenericFallback(t *testing.T) {
	// A registered ticker keeps its own binding.
	b := LookupOrGeneric("RUAL")
	if s := b.Build(b.DefaultParams()); s.Ticker() != "RUAL" {
		t.Fatalf("RUAL built ticker = %q, want RUAL", s.Ticker())
	}
	// An unregistered ticker gets a generic binding bound to that ticker.
	g := LookupOrGeneric("SBER")
	if g.DefaultParams == nil || g.Build == nil || g.ParseParams == nil {
		t.Fatal("generic binding has nil funcs")
	}
	s := g.Build(g.DefaultParams())
	if s.Ticker() != "SBER" {
		t.Fatalf("generic built ticker = %q, want SBER", s.Ticker())
	}
	// Generic defaults must keep the HTF filter on and the quality knobs opted in.
	p := g.DefaultParams().(adaptive.Params)
	if p.TrendFilterPeriod <= 0 || p.MinRR <= 0 || p.TrailArmATR <= 0 {
		t.Fatalf("generic defaults must opt into quality knobs: %+v", p)
	}
}
