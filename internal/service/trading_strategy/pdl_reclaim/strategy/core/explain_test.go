package core

import (
	"strings"
	"testing"
)

func TestExplainEntryA(t *testing.T) {
	out := newS(DefaultParams()).Explain(fixture(warm([]bar{pre, barA})))
	for _, want := range []string{"✓ окно", "PDL 99.0000", "✓ A", "✓ RSI(6) сигнальной свечи", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainOutsideWindow(t *testing.T) {
	b := barA
	b.t = "2026-09-14 06:55"
	q := pre
	q.t = "2026-09-14 06:50"
	out := newS(DefaultParams()).Explain(fixture([]bar{q, b}))
	if !strings.Contains(out, "✗ окно") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestExplainHeldStop(t *testing.T) {
	md := fixture([]bar{fillBar, {"2026-09-14 10:10", 99.4, 99.5, 98.4, 98.6}})
	pos := pos0()
	md.Position = &pos
	out := newS(quiet()).Explain(md)
	for _, want := range []string{"--- выходы ---", "стоп", "ИТОГ: ПРОДАЖА SL"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
