package simulation

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"tinvest/internal/service/screener/density/paper"
)

func fillRecord() paper.Record {
	return paper.Record{At: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC), Kind: "entry_filled", FillLots: 1, FillRubles: 250, EntryRubles: 250, StopRubles: 249, TakeRubles: 252, Order: &paper.Order{Ticker: "SBER", InstrumentID: "x", Lot: 10, FilledLots: 1, OpenLots: 1}}
}

func TestOutboxOnlyQueuesFillsAndSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	q, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	r := fillRecord()
	r.Kind = "entry_placed"
	if err = q.Record(r); err != nil {
		t.Fatal(err)
	}
	if len(q.pending) != 0 {
		t.Fatal("placement generated execution alert")
	}
	r.Kind = "entry_filled"
	if err = q.Record(r); err != nil {
		t.Fatal(err)
	}
	q, err = OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.pending) != 1 || !strings.Contains(q.pending[0].Text, "СИМУЛЯЦИЯ") || !strings.Contains(q.pending[0].Text, "Купили бы") || !strings.Contains(q.pending[0].Text, "250.0000") {
		t.Fatalf("pending %+v", q.pending)
	}
}

func TestOutboxRetriesWithoutBlockingProducer(t *testing.T) {
	q, err := OpenOutbox(filepath.Join(t.TempDir(), "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	delivered := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		q.Run(ctx, func(context.Context, string) error {
			if calls.Add(1) == 1 {
				return errors.New("offline")
			}
			delivered <- struct{}{}
			return nil
		}, nil)
	}()
	defer func() { cancel(); <-done }()
	if err = q.Record(fillRecord()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-delivered:
	case <-time.After(3 * time.Second):
		t.Fatal("fill was not retried")
	}
}

func TestFormatExitShowsPriceWithoutPositionSize(t *testing.T) {
	r := fillRecord()
	r.Kind = "exit_filled"
	r.FillRubles = 252
	r.Detail = "take_profit"
	r.Order.NetPnL = 17.49
	text := FormatFill(r)
	if !strings.Contains(text, "Продали бы по 252.0000") || !strings.Contains(text, "take_profit") || strings.Contains(text, "Лоты") || strings.Contains(text, "комисси") {
		t.Fatal(text)
	}
}

func TestFillOutboxSurvivesSessionJournalFailure(t *testing.T) {
	root := t.TempDir()
	journal, err := paper.NewJournal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = journal.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "outbox.json")
	q, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = recordEvent(journal, q, fillRecord()); err == nil {
		t.Fatal("journal failure ignored")
	}
	q, err = OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.pending) != 1 || q.pending[0].Fill == nil || q.pending[0].Fill.Kind != "entry_filled" {
		t.Fatal("fill lost before journal commit")
	}
}

func TestSlowDeliveryDoesNotBlockAnotherExecution(t *testing.T) {
	q, err := OpenOutbox(filepath.Join(t.TempDir(), "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		q.Run(ctx, func(ctx context.Context, _ string) error { close(started); <-ctx.Done(); return ctx.Err() }, nil)
	}()
	defer func() { cancel(); <-done }()
	if err = q.Record(fillRecord()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("delivery did not start")
	}
	recorded := make(chan error, 1)
	go func() { r := fillRecord(); r.Kind = "exit_filled"; r.FillRubles = 252; recorded <- q.Record(r) }()
	select {
	case err := <-recorded:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Telegram blocked next execution")
	}
}
