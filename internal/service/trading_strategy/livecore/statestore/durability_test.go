package statestore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveSyncFailureDoesNotPublishState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store := New(path)
	failure := errors.New("fsync failed")
	called := false
	store.syncFile = func(f *os.File) error {
		called = true
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Error("state published before file sync")
		}
		return failure
	}
	if err := store.Save(map[string]Entry{"T": {Ticker: "T", Quantity: 1}}); !errors.Is(err, failure) || !called {
		t.Fatalf("save ignored durability failure: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unsynced state became visible")
	}
}
