package core

import (
	"strings"
	"testing"
)

func TestExplainConfirmedEntry(t *testing.T) {
	bars, g := greenAfterSignal()
	out := NewWithParams("TEST", stepBar, entryParams()).Explain(fixture(bars[:g+1]))
	for _, want := range []string{"✓ будни", "✓ взведено: сигнал час", "(ZONE), MACD 1h зелёная", "(через 1 ч), бар 1 из 6", "✓ MACD 15m", "✓ дневной ATR(14)", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// The signal hour with a red hourly MACD waits, and the verdict agrees with Decide.
func TestExplainWaitsOnRedHour(t *testing.T) {
	bars, a, _ := signalThenGreenHour()
	out := NewWithParams("TEST", stepBar, entryParams()).Explain(fixture(bars[:a+1]))
	for _, want := range []string{"сигнал ZONE", "(красная)", "✗ взвода нет", "ИТОГ: входа нет"} {
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
	out := NewWithParams("TEST", stepBar, stopOnlyOff()).Explain(md)
	for _, want := range []string{"--- выходы ---", "закрыт на этом баре ✓", "крест 70 вверх ✓", "ИТОГ: ПРОДАЖА RSI"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
