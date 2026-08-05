package statestore

import (
	"os"
	"path/filepath"
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
