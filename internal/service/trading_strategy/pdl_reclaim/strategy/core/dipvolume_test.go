package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// tuesdayDip is pdlDay moved to Tuesday 2026-09-15, so the warm-up holds Monday's main session:
// the series under PDL is 09:45, 10:00 and the reclaim 10:15; 09:45 is outside the main session.
func tuesdayDip() []bar {
	return path("2026-09-15 09:30", 100, 99.5, 98.7, 98.6, 99.2)
}

// dipMD gives every bar volume 100 and the series bars of the main session (the last two) dip.
func dipMD(bars []bar, dip int64) strategy.MarketData {
	md := fixture(warm(bars))
	md.Volumes = make([]int64, len(md.Times))
	for j := range md.Volumes {
		md.Volumes[j] = 100
	}
	n := len(md.Volumes)
	md.Volumes[n-1], md.Volumes[n-2] = dip, dip
	return md
}

func dipP(mult float64) Params {
	p := DefaultParams()
	p.PDLDipVolMult = mult
	return p
}

func TestDipVolumeAboveNormal(t *testing.T) {
	sig := newS(dipP(1.5)).Decide(dipMD(tuesdayDip(), 151))
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "покупка сразу, объём серии 151 > 1.50 × обычный 100") {
		t.Fatalf("dip 151: Kind = %v, reason %q", sig.Kind, sig.EntryReason)
	}
	if sig := newS(dipP(1.5)).Decide(dipMD(tuesdayDip(), 150)); sig.Kind != model.SignalNone {
		t.Fatalf("dip 150: Kind = %v, want None", sig.Kind)
	}
}

// The 09:45 bar is outside the main session: a spike on it alone doesn't lift the series.
func TestDipVolumeSkipsOffSessionBars(t *testing.T) {
	md := dipMD(tuesdayDip(), 100)
	md.Volumes[len(md.Volumes)-3] = 10000
	if sig := newS(dipP(1.5)).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("spike at 09:45: Kind = %v, want None", sig.Kind)
	}
}

// A series wholly before 10:00 has no main-session bars: no entry.
func TestDipVolumeMorningSeriesFailsClosed(t *testing.T) {
	bars := path("2026-09-15 09:00", 100, 99.5, 98.7, 98.6, 99.2)
	if sig := newS(dipP(1.5)).Decide(dipMD(bars, 1000)); sig.Kind != model.SignalNone {
		t.Fatalf("morning series: Kind = %v, want None", sig.Kind)
	}
	if sig := newS(dipP(0)).Decide(dipMD(bars, 1000)); sig.Kind != model.SignalBuy {
		t.Fatalf("filter off: Kind = %v, want Buy", sig.Kind)
	}
}

func TestDipVolumeNeedsVolumes(t *testing.T) {
	if sig := newS(dipP(1.5)).Decide(fixture(warm(tuesdayDip()))); sig.Kind != model.SignalNone {
		t.Fatalf("no volumes: Kind = %v, want None", sig.Kind)
	}
}

// The filter warms up VolBars main-session bars even with the PDH volume filter off.
func TestLookbackCoversDipVolume(t *testing.T) {
	p := dipP(1.5)
	p.VolumeMult = 0
	if got, want := newS(p).Lookback(), newS(DefaultParams()).Lookback(); got != want {
		t.Fatalf("Lookback = %d, want %d", got, want)
	}
}

func TestExplainDipVolume(t *testing.T) {
	out := newS(dipP(1.5)).Explain(dipMD(tuesdayDip(), 151))
	for _, want := range []string{"✓ объём серии под PDL 151 > 1.50 × обычный 100", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	out = newS(dipP(1.5)).Explain(dipMD(tuesdayDip(), 150))
	for _, want := range []string{"✗ объём серии под PDL 150 > 1.50 × обычный 100", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
