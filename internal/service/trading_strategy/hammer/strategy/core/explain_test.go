package core

import (
	"strings"
	"testing"
)

func TestExplainBuy(t *testing.T) {
	out := NewWithParams("TEST", step, noTrend()).Explain(fixture(bars(fall, hammerAfterFall), at(wednesday), 40))
	for _, want := range []string{"✓ календарь", "✓ молот", "✓ RSI(5)", "✓ объём", "✓ дневные уровни", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in:\n%s", want, out)
		}
	}
}

func TestExplainNoHammer(t *testing.T) {
	out := NewWithParams("TEST", step, noTrend()).Explain(fixture(bars(fall, bar{98.4, 100, 97, 98.6}), at(wednesday), 40))
	if !strings.Contains(out, "✗ молот") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Errorf("want a failed hammer and no entry:\n%s", out)
	}
}

func TestExplainUnderEMA(t *testing.T) {
	md := fixture(trendBars(100), at(wednesday), 40)
	out := NewWithParams("TEST", step, DefaultParams()).Explain(md)
	if !strings.Contains(out, "✗ тренд") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Errorf("BelowEMA off: want a failed trend gate:\n%s", out)
	}
	out = NewWithParams("TEST", step, below()).Explain(md)
	if !strings.Contains(out, "под EMA(200)") || !strings.Contains(out, "✓ RSI(6)") || !strings.Contains(out, "ИТОГ: ПОКУПКА") {
		t.Errorf("BelowEMA on: want the under-EMA mode on RSI(6) and a buy:\n%s", out)
	}
}

func TestExplainExits(t *testing.T) {
	out := NewWithParams("TEST", step, below()).Explain(heldBelow(bars(fall, bar{98.2, 101.5, 98.2, 101.3})))
	for _, want := range []string{"--- выходы ---", "режим: под EMA", "✓ RSI(6)", "ИТОГ: ПРОДАЖА RSI50"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in:\n%s", want, out)
		}
	}
}
