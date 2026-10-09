package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	dropbouncegazp "tinvest/internal/service/trading_strategy/drop_bounce/strategy/gazp"
	dropbouncetbank "tinvest/internal/service/trading_strategy/drop_bounce/strategy/tbank"
)

func dropBounceBinding(t *testing.T, ticker string, interval enum.Interval) Binding {
	t.Helper()
	b, err := DropBounceLookupOrGeneric(ticker, interval)
	if err != nil {
		t.Fatalf("lookup %s %s: %v", ticker, interval, err)
	}
	return b
}

func TestDropBounceGenericTickerUsesCoreDefaults(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Hour1} {
		b := dropBounceBinding(t, "ZZZZ", iv)
		got, ok := b.DefaultParams().(core.Params)
		if !ok || got != core.DefaultParams() {
			t.Fatalf("%s: defaults = %#v, want core.DefaultParams()", iv, b.DefaultParams())
		}
		if s := b.Build(got); s.Ticker() != "ZZZZ" {
			t.Fatalf("%s: ticker = %q, want ZZZZ", iv, s.Ticker())
		}
	}
}

func TestDropBounceGAZPUsesLiteralPerTimeframe(t *testing.T) {
	cases := map[enum.Interval]core.Params{
		enum.Minutes5:  dropbouncegazp.DefaultParams5m(),
		enum.Minutes15: dropbouncegazp.DefaultParams15m(),
		enum.Hour1:     dropbouncegazp.DefaultParams1h(),
	}
	for iv, want := range cases {
		b := dropBounceBinding(t, "GAZP", iv)
		if got := b.DefaultParams().(core.Params); got != want {
			t.Fatalf("%s: defaults = %+v, want %+v", iv, got, want)
		}
		if s := b.Build(b.DefaultParams()); s.Ticker() != "GAZP" {
			t.Fatalf("%s: ticker = %q, want GAZP", iv, s.Ticker())
		}
	}
}

func TestDropBounceTUsesLiteralPerTimeframe(t *testing.T) {
	cases := map[enum.Interval]core.Params{
		enum.Minutes5:  dropbouncetbank.DefaultParams5m(),
		enum.Minutes15: dropbouncetbank.DefaultParams15m(),
		enum.Hour1:     dropbouncetbank.DefaultParams1h(),
	}
	for iv, want := range cases {
		b := dropBounceBinding(t, "T", iv)
		if got := b.DefaultParams().(core.Params); got != want {
			t.Fatalf("%s: defaults = %+v, want %+v", iv, got, want)
		}
		if s := b.Build(b.DefaultParams()); s.Ticker() != "T" {
			t.Fatalf("%s: ticker = %q, want T", iv, s.Ticker())
		}
	}
}

func TestDropBounceRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes30, enum.Hour4, enum.Day1} {
		if _, err := DropBounceLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestDropBounceParseParamsLayersOverTickerLiteral(t *testing.T) {
	parsed, err := dropBounceBinding(t, "GAZP", enum.Minutes15).ParseParams([]byte(`{"MinDropPct": 2.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.MinDropPct != 2.5 || got.StopBody != dropbouncegazp.DefaultParams15m().StopBody {
		t.Fatalf("got %+v", got)
	}
}
