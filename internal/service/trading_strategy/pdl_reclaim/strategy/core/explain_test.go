package core

import (
	"strings"
	"testing"
)

func TestExplainReclaimEntry(t *testing.T) {
	out := newS(DefaultParams()).Explain(fixture(warm(upTo(reclaimAt))))
	for _, want := range []string{"✓ окно: сигнал 15m", "PDL 99.0000", "✓ возврат над PDL на этой свече",
		"✓ цель 100.0000", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// The bar after the reclaim is no trigger: the buy came on the reclaim bar.
func TestExplainAfterReclaim(t *testing.T) {
	out := newS(DefaultParams()).Explain(fixture(warm(upTo(reclaimAt + 1))))
	for _, want := range []string{"✗ возврат над PDL на этой свече", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainBreakoutVolume(t *testing.T) {
	out := newS(DefaultParams()).Explain(pdhMD(150))
	for _, want := range []string{"✓ пробой PDH на этой свече", "✗ объём пробоя 150 > 1.50 × среднее 20", "✗ пробоя PDH 1 св. назад", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainBreakoutHeld(t *testing.T) {
	out := newS(DefaultParams()).Explain(pdhMD(151, holdC))
	for _, want := range []string{"✓ пробой PDH на свече 10:00 держится над PDH 1 св. (мин. close 101.4000)", "✓ цель 102.0000", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainOutsideWindow(t *testing.T) {
	out := newS(DefaultParams()).Explain(fixture(path("2026-09-14 06:30", 100, 99.5, 99.4)))
	if !strings.Contains(out, "✗ окно") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestExplainHeldStop(t *testing.T) {
	md := fixture([]bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.4, 98.6}})
	pos := pos0()
	md.Position = &pos
	out := newS(DefaultParams()).Explain(md)
	for _, want := range []string{"--- выходы ---", "стоп", "ИТОГ: ПРОДАЖА SL"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
