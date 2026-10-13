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
	for _, want := range []string{"✓ окно", "✓ тренд", "✓ RSI(6)", "ветка ZONE", "✓ ход дня", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// The Friday cutoff shows up as the failed window gate and the verdict agrees with Decide.
func TestExplainFridayAfternoon(t *testing.T) {
	out := explainAt("2026-09-18 15:00", trendCloses(6))
	for _, want := range []string{"✗ окно", "пятница до 12:00 ✗", "ИТОГ: входа нет"} {
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

// A held position lists the exits; the Friday green-histogram cross ends in a FRI sale.
func TestExplainHeldFriday(t *testing.T) {
	bars := armedPathFriday()
	md := fixture(bars[:armedCross+1])
	md.Position = position(100, bars[95].t, fixtureATR)
	out := NewWithParams("TEST", stepH1, stopOnlyOff()).Explain(md)
	for _, want := range []string{"--- выходы ---", "пятница и RSI ≥ 70 (FRI) ✓", "ИТОГ: ПРОДАЖА FRI"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
