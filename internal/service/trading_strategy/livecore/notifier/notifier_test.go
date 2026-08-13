package notifier

import (
	"strings"
	"testing"
)

func TestEntry(t *testing.T) {
	msg := Entry("UGLD", 100.5, 10, 100, false)
	for _, want := range []string{"UGLD", "100.5", "🟢"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Entry msg %q missing %q", msg, want)
		}
	}
	if !strings.Contains(Entry("UGLD", 1, 1, 1, true), "БУМАЖНАЯ") {
		t.Fatal("paper-mode entry must be flagged")
	}
}

func TestExitAndSkipAndAlert(t *testing.T) {
	if !strings.Contains(Exit("NVTK", "OB", 200, 50, false), "NVTK") {
		t.Fatal("Exit must name the ticker")
	}
	if !strings.Contains(Skip("EUTR", "кэша не хватает"), "кэша не хватает") {
		t.Fatal("Skip must carry the reason")
	}
	if !strings.Contains(Alert("Reversion", "UGLD", "стейт потерян"), "⚠️") {
		t.Fatal("Alert must be visibly flagged")
	}
}

func TestAlertUsesStrategyLabel(t *testing.T) {
	got := Alert("RSI Pullback", "UGLD", "стоп не выставлен")
	if !strings.Contains(got, "RSI Pullback UGLD") {
		t.Fatalf("Alert = %q, want it to name the strategy and the ticker", got)
	}
	if strings.Contains(got, "Reversion") {
		t.Fatalf("Alert = %q, must not hardcode Reversion", got)
	}
}

func TestStopSet(t *testing.T) {
	msg := StopSet("UGLD", 107.5, "TRAIL", true)
	for _, want := range []string{"UGLD", "107.5", "TRAIL", "БУМАЖНАЯ"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("StopSet msg %q missing %q", msg, want)
		}
	}
	if strings.Contains(StopSet("UGLD", 107.5, "TRAIL", false), "БУМАЖНАЯ") {
		t.Fatal("non-paper StopSet must not be flagged as БУМАЖНАЯ")
	}
}

// Стартовое сообщение — единственный способ снаружи отличить поднятый раннер от
// невзлетевшего: событий у стратегии может не быть неделями, и молчание в теме
// одинаково выглядит в обоих случаях. Поэтому оно обязано нести и вселенную (по ней
// видно, что конфиг доехал), и режим (бумажный он или боевой).
func TestStartupNamesStrategyUniverseAndPaperMode(t *testing.T) {
	msg := Startup("RSI Pullback", []string{"UGLD", "GAZP"}, true)
	for _, want := range []string{"RSI Pullback", "UGLD", "GAZP", "бумажный"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Startup msg %q missing %q", msg, want)
		}
	}
}

// Боевой режим обязан отличаться от бумажного в тексте: перепутать их — значит не
// заметить, что раннер выставляет реальные ордера.
func TestStartupFlagsLiveTradingMode(t *testing.T) {
	msg := Startup("RSI Pullback", []string{"UGLD"}, false)
	if strings.Contains(msg, "бумажный") {
		t.Fatalf("Startup msg %q must not call live trading бумажный", msg)
	}
	if !strings.Contains(msg, "боевой") {
		t.Fatalf("Startup msg %q must name the боевой mode", msg)
	}
}

// Пустая вселенная — рабочее состояние конфига (RSI_PULLBACK_TICKERS не задан и
// дефолт затёрт пустой строкой), и оно обязано быть видно в сообщении: раннер,
// который поднялся без единого тикера, снаружи неотличим от работающего.
func TestStartupSpellsOutEmptyUniverse(t *testing.T) {
	msg := Startup("RSI Pullback", nil, true)
	if !strings.Contains(msg, "вселенная пуста") {
		t.Fatalf("Startup msg %q must spell out the empty universe", msg)
	}
}
