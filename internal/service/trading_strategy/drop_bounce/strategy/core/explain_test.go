package core

import (
	"strings"
	"testing"
)

// explainAt runs Explain over closes ending at end with the entry fixture's params.
func explainAt(end string, closes []float64) string {
	return NewWithParams("TEST", stepH1, entryParams()).Explain(fixture(endingAt(end, closes)))
}

func TestExplainEntryBar(t *testing.T) {
	out := explainAt("2026-09-16 07:00", trendCloses(6))
	for _, want := range []string{"✓ окно", "✓ тренд", "✓ RSI(6)", "ветка ZONE", "✓ дневной ATR(14)", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// A midday signal fails the buy-hours gate and the verdict agrees with Decide.
func TestExplainMidday(t *testing.T) {
	out := explainAt("2026-09-16 12:00", trendCloses(6))
	for _, want := range []string{"✗ окно", "покупка в 13:00", "[07:00, 11:00) или [17:00, 24:00) ✗", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainTrendFails(t *testing.T) {
	out := explainAt("2026-09-16 07:00", trendCloses(12))
	if !strings.Contains(out, "✗ тренд") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Fatalf("want failed trend gate and no entry:\n%s", out)
	}
}

// A held position lists the exits; the RSI(10) cross ends in an RSI sale.
func TestExplainHeldRSIExit(t *testing.T) {
	bars := slideAndRise()
	k := exitCross(t, bars)
	md := fixture(bars[:k+1])
	md.Position = position(200, bars[30].t, fixtureATR)
	out := NewWithParams("TEST", stepH1, stopOnlyOff()).Explain(md)
	for _, want := range []string{"--- выходы ---", "крест 70 вверх ✓", "ИТОГ: ПРОДАЖА RSI"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
