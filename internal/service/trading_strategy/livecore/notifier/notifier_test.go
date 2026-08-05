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
