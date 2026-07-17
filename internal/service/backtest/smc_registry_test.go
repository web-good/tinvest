package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/smc/strategy/core"
	smcsber "tinvest/internal/service/trading_strategy/smc/strategy/sber"
)

func TestSMCLookupOrGeneric(t *testing.T) {
	b := SMCLookupOrGeneric(smcsber.Ticker)
	if got := b.Build(b.DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q, want SBER", got)
	}
	if b.DefaultParams().(core.Params) != smcsber.DefaultParams() {
		t.Fatal("registered ticker must use its package defaults")
	}
	// Незнакомый тикер получает generic-байндинг, привязанный к нему.
	g := SMCLookupOrGeneric("XXXX")
	if got := g.Build(g.DefaultParams()).Ticker(); got != "XXXX" {
		t.Fatalf("generic Ticker = %q, want XXXX", got)
	}
	// Частичный JSON перекрывает только свои поля.
	p, err := b.ParseParams([]byte(`{"SwingK": 5}`))
	if err != nil {
		t.Fatal(err)
	}
	want := smcsber.DefaultParams()
	want.SwingK = 5
	if p.(core.Params) != want {
		t.Fatalf("ParseParams = %+v, want %+v", p, want)
	}
}
