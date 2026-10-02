package statestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileStore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reversion_acc.json")
	s := New(path)

	in := map[string]Entry{
		"UGLD": {Ticker: "UGLD", EntryTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
			EntryPrice: 100.5, EntryATR: 2.3, MaxFav: 105.0, Quantity: 10},
	}
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := out["UGLD"]
	if got.EntryPrice != 100.5 || got.EntryATR != 2.3 || got.MaxFav != 105.0 || got.Quantity != 10 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestFileStore_LoadMissingFileIsEmpty(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "does_not_exist.json"))
	out, err := s.Load()
	if err != nil {
		t.Fatalf("Load missing file should not error, got %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("Load missing file = %v, want empty", out)
	}
}

func TestEntryRoundTripsTakeProfit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := New(path)
	want := map[string]Entry{"UGLD": {Ticker: "UGLD", EntryPrice: 0.6, TakeProfit: 0.72}}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["UGLD"].TakeProfit != 0.72 {
		t.Fatalf("TakeProfit = %v, want 0.72", got["UGLD"].TakeProfit)
	}
}

// Стейт, записанный reversion (без поля takeProfit), обязан читаться как TakeProfit=0,
// а не ломать разбор файла: формат общий для обеих стратегий.
func TestEntryWithoutTakeProfitLoadsAsZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"UGLD":{"ticker":"UGLD","entryPrice":0.6}}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := New(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["UGLD"].TakeProfit != 0 {
		t.Fatalf("TakeProfit = %v, want 0", got["UGLD"].TakeProfit)
	}
}

// reversion никогда не пишет TakeProfit — omitempty обязан держать ключ "takeProfit" вне
// файла стейта при нулевом значении, иначе формат живого файла реальной стратегии
// начнёт обрастать лишним ключом на каждом Save. Проверяем сырые байты файла, а не
// разобранную структуру: это единственный способ поймать снятие omitempty.
func TestEntrySaveOmitsZeroTakeProfitFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := New(path)
	in := map[string]Entry{"UGLD": {Ticker: "UGLD", EntryPrice: 0.6}} // TakeProfit не задан — 0
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(b), "takeProfit") {
		t.Fatalf("state file must omit zero-value takeProfit, got: %s", b)
	}
}

// Прод-файл rsi_pullback записан до появления поля strategy. Он обязан читаться без
// ошибок, а поле — оставаться пустым: пустое значение и есть «позиция rsi_pullback».
func TestEntryWithoutStrategyFieldLoadsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	raw := `{"GAZP":{"ticker":"GAZP","entryPrice":100,"entryATR":10,"quantity":10}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := New(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := st["GAZP"].Strategy; got != "" {
		t.Fatalf("Strategy = %q, want пусто", got)
	}
}

// omitempty: reversion поле не пишет, и формат его файла меняться не должен.
func TestEmptyStrategyIsNotWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := New(path).Save(map[string]Entry{"UGLD": {Ticker: "UGLD"}}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "strategy") {
		t.Fatalf("пустое поле strategy попало в файл: %s", b)
	}
}

// StopLoss — замороженный уровень стопа gap_fade — переживает перезапуск.
func TestStopLossRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := New(path)
	if err := s.Save(map[string]Entry{"GAZP": {Ticker: "GAZP", StopLoss: 78.5}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["GAZP"].StopLoss != 78.5 {
		t.Fatalf("StopLoss = %v, want 78.5", got["GAZP"].StopLoss)
	}
}

// Стейт, записанный до появления поля, читается как прежде: StopLoss = 0.
func TestLegacyStateWithoutStopLossLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	legacy := `{"GAZP":{"ticker":"GAZP","entryTime":"2026-03-10T07:01:00+03:00","entryPrice":100,"entryATR":10,"maxFav":100,"quantity":10,"takeProfit":120}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := New(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	e := got["GAZP"]
	if e.StopLoss != 0 || e.TakeProfit != 120 || e.EntryPrice != 100 {
		t.Fatalf("legacy entry = %+v", e)
	}
}
