package core

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// indexDays returns weekday index dailies at 10:00 MSK ending the weekday before Monday 2026-09-14
// (plus Saturday and Sunday sessions that must be skipped), closing at closes[i] oldest-first.
func indexDays(closes ...float64) ([]time.Time, []float64) {
	var times []time.Time
	var out []float64
	d := at("2026-09-11 10:00")
	for i := len(closes) - 1; i >= 0; {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			times = append([]time.Time{d}, times...)
			out = append([]float64{1e6}, out...) // a weekend print must not move the EMA
		} else {
			times = append([]time.Time{d}, times...)
			out = append([]float64{closes[i]}, out...)
			i--
		}
		d = d.AddDate(0, 0, -1)
	}
	return times, out
}

func ramp(from, step float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = from + step*float64(i)
	}
	return out
}

func decideWithIndex(p Params, times []time.Time, closes []float64) model.Signal {
	return newS(p).WithIndex(times, closes).Decide(fixture(warm(upTo(reclaimAt))))
}

func TestIndexTrendFilter(t *testing.T) {
	p := DefaultParams()
	p.IndexTrendEMA = 3
	falling, rising := ramp(3000, -10, 12), ramp(3000, 10, 12)

	ft, fc := indexDays(falling...)
	sig := decideWithIndex(p, ft, fc)
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "EMA(3) индекса") {
		t.Fatalf("falling index: Kind = %v, reason %q, want Buy with the index reason", sig.Kind, sig.EntryReason)
	}
	rt, rc := indexDays(rising...)
	if sig := decideWithIndex(p, rt, rc); sig.Kind != model.SignalNone {
		t.Fatalf("rising index: Kind = %v, want None", sig.Kind)
	}
	if sig := decideWithIndex(p, nil, nil); sig.Kind != model.SignalNone {
		t.Fatalf("no index: Kind = %v, want None", sig.Kind)
	}
	short, sc := indexDays(falling[:3+indexSlopeDays-1]...)
	if sig := decideWithIndex(p, short, sc); sig.Kind != model.SignalNone {
		t.Fatalf("too few index dailies: Kind = %v, want None", sig.Kind)
	}
	p.IndexTrendEMA = 0
	if sig := decideWithIndex(p, rt, rc); sig.Kind != model.SignalBuy || strings.Contains(sig.EntryReason, "индекса") {
		t.Fatalf("filter off: Kind = %v, reason %q, want Buy without the index", sig.Kind, sig.EntryReason)
	}
}

// The index daily of the trigger's own date is not complete yet: a crash printed on Monday must
// not turn a rising index into a falling one.
func TestIndexTrendIgnoresTodaysDaily(t *testing.T) {
	p := DefaultParams()
	p.IndexTrendEMA = 3
	times, closes := indexDays(ramp(3000, 10, 12)...)
	times, closes = append(times, at("2026-09-14 07:00")), append(closes, 1)
	if sig := decideWithIndex(p, times, closes); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None: Monday's own index daily leaked in", sig.Kind)
	}
}
