package journal

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestOpenLocksDirectory(t *testing.T) {
	dir := t.TempDir()
	j := openJournal(t, dir)
	if other, err := Open(dir); err == nil {
		_ = other.Close()
		t.Fatal("second open succeeded while the state directory was locked")
	}
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	openJournal(t, dir)
}

func TestOpenRecoversInterruptedSessionAndPendingFills(t *testing.T) {
	dir := t.TempDir()
	j := openJournal(t, dir)
	entry := model.Fill{
		Key: model.Key{Market: model.Spot, Symbol: "BTCUSDT"}, Side: model.Bid,
		Entry: true, Price: 100, Qty: 5, At: time.Now().UTC(),
	}
	exit := model.Fill{
		Key: entry.Key, Side: model.Bid, Partial: true, Price: 104, Qty: 2,
		Reason: "take", Fee: 0.208, NetPnL: 7.592, At: entry.At.Add(time.Second),
	}
	if err := j.Record(entry, "entry"); err != nil {
		t.Fatal(err)
	}
	if err := j.Record(exit, "partial exit"); err != nil {
		t.Fatal(err)
	}
	linear := entry
	linear.Key.Market = model.Linear
	linear.Side = model.Ask
	if err := j.Record(linear, "linear entry"); err != nil {
		t.Fatal(err)
	}
	linear.Entry = false
	linear.Reason = "stop"
	if err := j.Record(linear, "linear exit"); err != nil {
		t.Fatal(err)
	}

	// Emulate process termination after a previous successful commit and a torn write.
	if _, err := j.file.WriteString(`{"unfinished":`); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(j.file.Close(), j.lock.Close()); err != nil {
		t.Fatal(err)
	}
	j.file = nil

	reopened := openJournal(t, dir)
	if err := reopened.Record(entry, "new session entry"); err != nil {
		t.Fatal(err)
	}
	assertDelivery(t, reopened, []string{"entry", "partial exit", "linear entry", "linear exit", "new session entry"})
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}

	type eventRow struct {
		Kind    string
		Session uint64
		Reason  string
		Open    []struct {
			Entry     model.Fill
			Remaining float64
		}
		Fill    model.Fill
		Message string
	}
	var ends []eventRow
	f, err := os.Open(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var persistedExit model.Fill
	lines := bufio.NewScanner(f)
	for lines.Scan() {
		var row eventRow
		if err := json.Unmarshal(lines.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		if row.Kind == "session_ended" {
			ends = append(ends, row)
		}
		if row.Message == "partial exit" {
			persistedExit = row.Fill
		}
	}
	if err := lines.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ends) != 2 || ends[0].Reason != "interrupted" || ends[1].Reason != "shutdown" || ends[0].Session == ends[1].Session {
		t.Fatalf("expected distinct interrupted and stopped sessions, got %+v", ends)
	}
	for i, remaining := range []float64{3, 5} {
		if positions := ends[i].Open; len(positions) != 1 || positions[0].Entry.Key != entry.Key || positions[0].Remaining != remaining {
			t.Fatalf("session %d positions = %+v, want %s with %v remaining", i, positions, entry.Key.Symbol, remaining)
		}
	}
	if persistedExit.Fee != exit.Fee || persistedExit.NetPnL != exit.NetPnL {
		t.Fatalf("persisted exit = %+v, want recorded fees and net PnL %+v", persistedExit, exit)
	}

	third := openJournal(t, dir)
	if err := third.Record(entry, "third session entry"); err != nil {
		t.Fatal(err)
	}
	assertDelivery(t, third, []string{"third session entry"})
}

func TestDeliverRetriesWithoutHoldingUpRecord(t *testing.T) {
	j := openJournal(t, t.TempDir())
	entry := model.Fill{Key: model.Key{Market: model.Linear, Symbol: "BTCUSDT"}, Side: model.Ask, Entry: true, Price: 100, Qty: 1}
	if err := j.Record(entry, "entry"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	var sent []string
	go func() {
		finished <- j.Deliver(ctx, func(ctx context.Context, message string) error {
			sent = append(sent, message)
			if len(sent) == 1 {
				close(started)
				select {
				case <-release:
				case <-ctx.Done():
					return ctx.Err()
				}
				return errors.New("temporary transport failure")
			}
			if len(sent) == 3 {
				cancel()
			}
			return nil
		})
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("delivery did not start")
	}
	entry.Key.Market = model.Spot
	entry.Side = model.Bid
	recorded := make(chan error, 1)
	go func() { recorded <- j.Record(entry, "second entry") }()
	select {
	case err := <-recorded:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("recording waited for Telegram delivery")
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sent, []string{"entry", "entry", "second entry"}) {
		t.Fatalf("sent = %q, want retry before next entry", sent)
	}
}

func openJournal(t *testing.T, dir string) *Journal {
	t.Helper()
	j, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := j.Close(); err != nil {
			t.Error(err)
		}
	})
	return j
}

func assertDelivery(t *testing.T, j *Journal, expected []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sent []string
	if err := j.Deliver(ctx, func(_ context.Context, message string) error {
		sent = append(sent, message)
		if len(sent) == len(expected) {
			cancel()
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sent, expected) {
		t.Fatalf("sent = %q, want %q", sent, expected)
	}
}
