package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func ramp(from, step float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = from + step*float64(i)
	}
	return out
}

func decideWithH1(p Params, h1 []float64) model.Signal {
	md := fixture(warm(upTo(reclaimAt)))
	md.HTFCloses = h1
	return newS(p).Decide(md)
}

func TestH1TrendFilter(t *testing.T) {
	p := DefaultParams()
	p.H1TrendEMA = 3
	sig := decideWithH1(p, ramp(100, 1, 12))
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, ", close часа 111.0000 > EMA(3) часа 110.0000") {
		t.Fatalf("rising hours: Kind = %v, reason %q, want Buy with the hourly reason", sig.Kind, sig.EntryReason)
	}
	if sig := decideWithH1(p, ramp(100, -1, 12)); sig.Kind != model.SignalNone {
		t.Fatalf("falling hours: Kind = %v, want None", sig.Kind)
	}
	if sig := decideWithH1(p, []float64{100, 100, 100, 100}); sig.Kind != model.SignalNone {
		t.Fatalf("close at the EMA: Kind = %v, want None", sig.Kind)
	}
	if sig := decideWithH1(p, nil); sig.Kind != model.SignalNone {
		t.Fatalf("no hourly bars: Kind = %v, want None", sig.Kind)
	}
	if sig := decideWithH1(p, []float64{100, 101}); sig.Kind != model.SignalNone {
		t.Fatalf("fewer hours than the period: Kind = %v, want None", sig.Kind)
	}
	p.H1TrendEMA = 0
	if sig := decideWithH1(p, ramp(100, -1, 12)); sig.Kind != model.SignalBuy || strings.Contains(sig.EntryReason, "часа") {
		t.Fatalf("filter off: Kind = %v, reason %q, want Buy without the hourly reason", sig.Kind, sig.EntryReason)
	}
}

func TestExplainH1Trend(t *testing.T) {
	p := DefaultParams()
	p.H1TrendEMA = 3
	md := fixture(warm(upTo(reclaimAt)))
	md.HTFCloses = ramp(100, -1, 12)
	out := newS(p).Explain(md)
	for _, want := range []string{"✗ тренд часа: close часа 89.0000 < EMA(3) часа 90.0000", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
