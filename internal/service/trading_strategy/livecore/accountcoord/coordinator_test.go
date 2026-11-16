package accountcoord

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestAccountLeaseSerializesParticipantsAndHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	a := New(root, "account", nil)
	b := New(root, "account", nil)
	lease, err := a.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err = b.TryLock(); !errors.Is(err, ErrBusy) {
		t.Fatalf("parallel entry admitted: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err = b.Lock(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}
func TestOwnershipSurvivesRestartAndCannotOverwriteAnActiveClaim(t *testing.T) {
	root := t.TempDir()
	c := New(root, "account", nil)
	l, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	claim := Claim{Ticker: "T", InstrumentID: "uid", Owner: "density", Key: "first"}
	if err = c.Reserve(l, claim); err != nil {
		t.Fatal(err)
	}
	l.Close()
	again := New(root, "account", nil)
	claims, err := again.Claims(context.Background())
	if err != nil || claims["T"].Key != "first" {
		t.Fatalf("%v %v", claims, err)
	}
	l, err = again.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	claim.Key = "second"
	if err = again.Reserve(l, claim); !errors.Is(err, ErrBusy) {
		t.Fatalf("overwrote unresolved owner: %v", err)
	}
}
func TestClosingClaimDoesNotNeedEntryLeaseAndCannotEraseNewOwner(t *testing.T) {
	c := New(t.TempDir(), "account", nil)
	l, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	old := Claim{Ticker: "T", InstrumentID: "uid", Owner: "density", Key: "old"}
	if err = c.Reserve(l, old); err != nil {
		t.Fatal(err)
	}
	if err = c.MarkClosed(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	snapshot, err := c.Claims(context.Background())
	if err != nil || snapshot["T"].Status != "closed" {
		t.Fatal(snapshot, err)
	}
	if err = c.Remove(l, snapshot["T"]); err != nil {
		t.Fatal(err)
	}
	fresh := Claim{Ticker: "T", InstrumentID: "uid", Owner: "rsi_pullback", Key: "fresh"}
	if err = c.Reserve(l, fresh); err != nil {
		t.Fatal(err)
	}
	if err = c.MarkClosed(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	now, _ := c.Claims(context.Background())
	if now["T"].Key != "fresh" || now["T"].Status != "active" {
		t.Fatal("late old release changed new owner")
	}
	if snapshot["T"].Key != "old" {
		t.Fatal("ownership snapshot mutated")
	}
}
func TestLegacyStateBlocksOnlyThatTicker(t *testing.T) {
	store := statestore.New(filepath.Join(t.TempDir(), "state.json"))
	if err := store.Save(map[string]statestore.Entry{"T": {Ticker: "T", Strategy: "rsi_pullback", Quantity: 1}}); err != nil {
		t.Fatal(err)
	}
	c := New(t.TempDir(), "account", store)
	l, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if ok, err := c.CanEnter(l, "T", "density"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := c.CanEnter(l, "SBER", "density"); err != nil || !ok {
		t.Fatal(ok, err)
	}
}
func TestMissingOrCorruptRegistryNeverLooksLikeEmptyOwnership(t *testing.T) {
	c := New(t.TempDir(), "account", nil)
	l, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
	if _, err = c.Claims(context.Background()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(c.Root(), "owners.json")
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Claims(context.Background()); err == nil {
		t.Fatal("lost registry treated as empty")
	}
	if err = os.WriteFile(path, []byte("{bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Claims(context.Background()); err == nil {
		t.Fatal("corrupt ownership discarded")
	}
}

func TestDensityReservationsAreIndependent(t *testing.T) {
	c := New(t.TempDir(), "account", nil)
	lease, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err = c.Reserve(lease, Claim{Ticker: "T", InstrumentID: "t", Owner: "density", Key: "first"}); err != nil {
		t.Fatal(err)
	}
	if err = c.Reserve(lease, Claim{Ticker: "SBER", InstrumentID: "sber", Owner: "density", Key: "second"}); err != nil {
		t.Fatalf("second density position rejected: %v", err)
	}
}
func TestDensityLifetimeLeaseIsSharedAcrossInstancesAndBindsJournal(t *testing.T) {
	root := t.TempDir()
	a, b := New(root, "account", nil), New(root, "account", nil)
	journal := t.TempDir()
	lease, err := a.LockDensity(context.Background(), journal)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err = b.LockDensity(context.Background(), journal); !errors.Is(err, ErrBusy) {
		t.Fatalf("second journal writer admitted: %v", err)
	}
	lease.Close()
	if other, err := b.LockDensity(context.Background(), t.TempDir()); err == nil {
		other.Close()
		t.Fatal("different journal bypassed shared binding")
	}
}

func TestDensityAndCandleShareTickerWithoutSharingClaim(t *testing.T) {
	c := New(t.TempDir(), "account", nil)
	l, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	d := Claim{Ticker: "T", InstrumentID: "uid", Owner: "density", Key: "d"}
	a := Claim{Ticker: "T", InstrumentID: "uid", Owner: "rsi_zone", Key: "a"}
	if err := c.Reserve(l, d); err != nil {
		t.Fatal(err)
	}
	if err := c.Reserve(l, a); err != nil {
		t.Fatal(err)
	}
	if err := c.Verify(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	a.Owner = "rsi_pullback"
	a.Key = "b"
	if err := c.Reserve(l, a); !errors.Is(err, ErrBusy) {
		t.Fatalf("second candle admitted: %v", err)
	}
}

func TestVersionOneRegistryMigratesAtomicallyAndKeepsIdentity(t *testing.T) {
	c := New(t.TempDir(), "account", nil)
	if err := os.MkdirAll(c.Root(), 0700); err != nil {
		t.Fatal(err)
	}
	old := []byte(`{"Version":1,"AccountHash":"` + c.AccountHash() + `","Claims":{"T":{"Ticker":"T","InstrumentID":"uid","Owner":"density","Key":"old","Status":"active"}}}`)
	path := filepath.Join(c.Root(), "owners.json")
	if err := os.WriteFile(path, old, 0600); err != nil {
		t.Fatal(err)
	}
	claims, err := c.DensityClaims(context.Background())
	if err != nil || claims["T"].Key != "old" {
		t.Fatal(claims, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var migrated registry
	if err := json.Unmarshal(data, &migrated); err != nil {
		t.Fatal(err)
	}
	if migrated.Version != 2 || migrated.Claims["density:T"].Key != "old" {
		t.Fatalf("migration failed: %s", data)
	}
}
