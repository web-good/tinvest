package journal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestResetClearsStateAndStartsNewSession(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "bybit-density")
	j, err := open(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := j.Close(); err != nil {
			t.Error(err)
		}
	})

	entry := model.Fill{Key: model.Key{Market: model.Spot, Symbol: "BTCUSDT"}, Side: model.Bid, Entry: true, Qty: 1}
	if err := j.Record(entry, "old pending notification"); err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(parent, "other-strategy")
	if err := os.WriteFile(sibling, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sibling, filepath.Join(dir, "nested", "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "runner.lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := j.Reset(); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name() != "events.jsonl" {
		t.Fatalf("unexpected files after reset: %v", files)
	}
	if data, err := os.ReadFile(sibling); err != nil || string(data) != "keep" {
		t.Fatalf("sibling = %q, err = %v", data, err)
	}
	if other, err := open(dir, true); err == nil {
		_ = other.Close()
		t.Fatal("directory lock was lost during reset")
	}
	if len(j.pending) != 0 {
		t.Fatal("old notifications remain queued")
	}
	if err := j.Record(entry, "new entry"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("old pending notification")) || !bytes.Contains(data, []byte("new entry")) {
		t.Fatalf("unexpected journal after reset: %s", data)
	}
}

func TestNextReset(t *testing.T) {
	now := time.Date(2026, time.November, 19, 2, 59, 0, 0, time.FixedZone("MSK", 3*60*60))
	j := openJournal(t, t.TempDir())
	if got := j.NextReset(now); !got.IsZero() {
		t.Fatalf("custom state directory is scheduled for deletion at %v", got)
	}
	dir := t.TempDir()
	daily, err := open(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := daily.Close(); err != nil {
			t.Error(err)
		}
	}()
	want := now.Add(time.Minute)
	if got := daily.NextReset(now); !got.Equal(want) {
		t.Fatalf("next reset = %v, want %v", got, want)
	}
	if got := daily.NextReset(want); !got.Equal(want.Add(24 * time.Hour)) {
		t.Fatalf("next reset after boundary = %v", got)
	}
}
