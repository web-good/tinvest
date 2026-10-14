package core

import (
	"strings"
	"testing"
)

func TestExplainConfirmedEntry(t *testing.T) {
	bars, a := greenAfterSignal()
	out := NewWithParams("TEST", stepM30, entryParams()).Explain(fixture(bars[:a+2]))
	for _, want := range []string{"✓ будни", "✓ взведено часом", "(ZONE), бар 2 из 4", "✓ MACD 30m", "✓ дневной ATR(14)", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// The arming bar with a red histogram waits, and the verdict agrees with Decide.
func TestExplainWaitsOnRed(t *testing.T) {
	bars, a := greenAfterSignal()
	out := NewWithParams("TEST", stepM30, entryParams()).Explain(fixture(bars[:a+1]))
	for _, want := range []string{"сигнал ZONE", "✓ взведено часом", "✗ MACD 30m", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// A held position lists the exits; the hourly RSI(10) cross ends in an RSI sale.
func TestExplainHeldRSIExit(t *testing.T) {
	bars := slideAndRise()
	_, d := exitHour(t)
	md := fixture(bars[:d+1])
	md.Position = position(200, bars[60].t, fixtureATR)
	out := NewWithParams("TEST", stepM30, stopOnlyOff()).Explain(md)
	for _, want := range []string{"--- выходы ---", "закрыт на этом баре ✓", "крест 70 вверх ✓", "ИТОГ: ПРОДАЖА RSI"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
