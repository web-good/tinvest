package core

import (
	"strings"
	"testing"
)

func TestExplainBuy(t *testing.T) {
	out := NewWithParams("TEST", step, noTrend()).Explain(fixture(bars(fall, hammerAfterFall), at(wednesday), 40))
	for _, want := range []string{"✓ календарь", "✓ молот", "✓ RSI(5)", "✓ объём", "✓ стоп под молотом", "ИТОГ: ПОКУПКА"} {
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

func TestExplainBelowEMA(t *testing.T) {
	out := NewWithParams("TEST", step, DefaultParams()).Explain(fixture(trendBars(100), at(wednesday), 40))
	if !strings.Contains(out, "✗ тренд") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Errorf("want a failed trend gate:\n%s", out)
	}
}

func TestExplainExits(t *testing.T) {
	out := NewWithParams("TEST", step, DefaultParams()).Explain(held(fadedBounce(), 101, 50, 200))
	for _, want := range []string{"--- выходы ---", "цель снята", "✓ MACD(11,15,9)", "ИТОГ: ПРОДАЖА MACD"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in:\n%s", want, out)
		}
	}
}

func TestExplainConfirm(t *testing.T) {
	out := NewWithParams("TEST", step, confirmParams(3)).Explain(confirmed(nil, confirmBar))
	for _, want := range []string{"✓ подтверждает молот", "✓ молот", "✓ стоп под молотом", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in:\n%s", want, out)
		}
	}
	out = NewWithParams("TEST", step, confirmParams(3)).Explain(fixture(bars(fall, hammerAfterFall), at(wednesday), 40))
	if !strings.Contains(out, "подтверждённого молота нет") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Errorf("want no confirmed hammer:\n%s", out)
	}
}
