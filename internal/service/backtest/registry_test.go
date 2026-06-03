package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

func TestLookupKnownAndUnknown(t *testing.T) {
	if _, ok := Lookup("NOPE"); ok {
		t.Fatal("expected unknown ticker to miss")
	}
	b, ok := Lookup("RUAL")
	if !ok {
		t.Fatal("expected RUAL binding")
	}
	if b.DefaultParams == nil || b.Build == nil || b.ParseParams == nil {
		t.Fatal("binding has nil funcs")
	}
}

func TestRUALBindingBuildsStrategy(t *testing.T) {
	b, _ := Lookup("RUAL")
	def := b.DefaultParams()
	s := b.Build(def)
	if s.Ticker() != "RUAL" {
		t.Fatalf("built strategy ticker = %q, want RUAL", s.Ticker())
	}
}

func TestRUALParseParamsOverridesDefaults(t *testing.T) {
	b, _ := Lookup("RUAL")
	raw := []byte(`{"EMAPeriod": 50}`)
	parsed, err := b.ParseParams(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := parsed.(rusal.Params)
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
