package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/paper"
)

func fillRecord() paper.Record {
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	return paper.Record{At: at, Kind: "entry_filled", FillLots: 1, FillRubles: 250, EntryRubles: 250, StopRubles: 249, TakeRubles: 252, Order: &paper.Order{Ticker: "SBER", InstrumentID: "x", Lot: 10, FilledLots: 1, OpenLots: 1, State: "open", Entry: 250_000_000_000, Density: 249_500_000_000, DensityLots: 15000, Opened: at}}
}

func closedRecord() paper.Record {
	r := fillRecord()
	r.Kind, r.Detail, r.FillRubles = "exit_filled", "take_profit", 252
	r.At = r.At.Add(5 * time.Minute)
	r.Order.State, r.Order.OpenLots = "closed", 0
	r.Order.Closed, r.Order.ExitReason, r.Order.GrossPnL = r.At, r.Detail, 20
	return r
}

func TestOutboxOnlyQueuesCompletedTradesAndSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	q, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	r := fillRecord()
	for _, kind := range []string{"entry_placed", "entry_filled", "entry_canceled", "unresolved", "exit_requested", "exit_filled"} {
		r.Kind = kind
		if err = q.Record(r); err != nil {
			t.Fatal(err)
		}
		if len(q.pending) != 0 {
			t.Fatalf("%s generated an alert before full close", kind)
		}
	}
	r = closedRecord()
	if err = q.Record(r); err != nil {
		t.Fatal(err)
	}
	if err = q.Record(r); err != nil {
		t.Fatal(err)
	}
	q, err = OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.pending) != 1 {
		t.Fatalf("pending %+v", q.pending)
	}
	for _, want := range []string{"СИМУЛЯЦИЯ", "Сделка завершена", "Купили бы по 250.0000 ₽", "Продали бы по 252.0000 ₽", "Стоп-лосс: 249.0000 ₽", "Тейк-профит: 252.0000 ₽", "Плотность при входе: 249.5000 ₽", "Размер плотности: 15000 лотов", "2026-11-16 15:00:00 МСК", "2026-11-16 15:05:00 МСК"} {
		if !strings.Contains(q.pending[0].Text, want) {
			t.Fatalf("missing %q: %s", want, q.pending[0].Text)
		}
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
	if err = q.Record(closedRecord()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-delivered:
	case <-time.After(3 * time.Second):
		t.Fatal("fill was not retried")
	}
}

func TestFormatExitShowsPriceWithoutPositionSize(t *testing.T) {
	r := closedRecord()
	r.Order.TrackedDensity = 251_000_000_000
	r.Kind = "exit_filled"
	r.FillRubles = 252
	r.Detail = "take_profit"
	r.Order.NetPnL = 17.49
	text := FormatFill(r)
	if !strings.Contains(text, "Продали бы по 252.0000") || !strings.Contains(text, "Тейк-профит") || strings.Contains(text, "Лоты") || strings.Contains(text, "комисси") {
		t.Fatal(text)
	}
}

func TestFormatExitExplainsReason(t *testing.T) {
	for reason, want := range map[string]string{"density_lost": "Плотность исчезла", "stop": "Стоп-лосс", "take_profit": "Тейк-профит", "position_timeout": "Истекло время удержания", "future_reason": "future_reason"} {
		r := closedRecord()
		r.Kind, r.Detail = "exit_filled", reason
		if text := FormatFill(r); !strings.Contains(text, "Причина: "+want) {
			t.Fatalf("%s: %s", reason, text)
		}
	}
}

func TestFormatExitShowsReturnFromEntry(t *testing.T) {
	for _, tc := range []struct {
		name string
		exit float64
		want string
	}{
		{name: "profit", exit: 252, want: "Результат: +0.80%"},
		{name: "loss", exit: 247, want: "Результат: -1.20%"},
		{name: "unchanged", exit: 250, want: "Результат: +0.00%"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := closedRecord()
			r.Kind = "exit_filled"
			r.FillRubles = tc.exit
			r.Order.GrossPnL = (tc.exit - r.EntryRubles) * float64(r.Order.FilledLots*r.Order.Lot)
			if text := FormatFill(r); !strings.Contains(text, tc.want) {
				t.Fatal(text)
			}
		})
	}
}

func TestFormatFillOmitsReturnWithoutExitOrEntryPrice(t *testing.T) {
	r := closedRecord()
	r.Kind = "entry_filled"
	if text := FormatFill(r); text != "" {
		t.Fatal(text)
	}
	r.Kind = "exit_filled"
	r.EntryRubles = 0
	if text := FormatFill(r); strings.Contains(text, "Результат:") {
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
	if err = recordEvent(journal, q, closedRecord()); err == nil {
		t.Fatal("journal failure ignored")
	}
	q, err = OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.pending) != 1 || q.pending[0].Fill == nil || q.pending[0].Fill.Kind != "exit_filled" {
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
	if err = q.Record(closedRecord()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("delivery did not start")
	}
	recorded := make(chan error, 1)
	go func() {
		r := closedRecord()
		r.At = r.At.Add(time.Hour)
		r.Order.Opened = r.Order.Opened.Add(time.Hour)
		r.Order.Closed = r.At
		recorded <- q.Record(r)
	}()
	select {
	case err := <-recorded:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Telegram blocked next execution")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) != 2 {
		t.Fatalf("second completed trade was not queued: %+v", q.pending)
	}
}

func TestFormatFillUsesAllExitExecutions(t *testing.T) {
	r := closedRecord()
	r.Order.FilledLots = 3
	r.FillLots, r.FillRubles = 1, 247
	// Two lots sold at 252 and the final lot at 247: average 250.3333.
	r.Order.GrossPnL = 10
	text := FormatFill(r)
	for _, want := range []string{"Продали бы по 250.3333 ₽", "Результат: +0.13%"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
}

func TestOutboxRestoresOnlyCompletedTradeReports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	entry := fillRecord()
	partial := closedRecord()
	partial.Order.State, partial.Order.OpenLots = "open", 1
	closed := closedRecord()
	data, err := json.Marshal([]message{{ID: "entry", Text: "old entry", Fill: &entry}, {ID: "partial", Text: "old partial exit", Fill: &partial}, {ID: "closed", Text: "old final exit", Fill: &closed}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	q, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.pending) != 1 || q.pending[0].ID != "closed" || q.pending[0].Text != FormatFill(closed) {
		t.Fatalf("legacy alerts were not converted: %+v", q.pending)
	}
}

func TestPartialTakeThenStopQueuesOneCompleteTrade(t *testing.T) {
	q, err := OpenOutbox(filepath.Join(t.TempDir(), "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := paper.DefaultConfig()
	cfg.Latency, cfg.Commission = 0, 0
	cfg.IgnoreCapital, cfg.MaxEntryLots = true, 3
	cfg.Freshness = 10 * time.Second
	e, err := paper.New(cfg, []density.Instrument{{ID: "x", Ticker: "X", Lot: 1, PriceStep: density.Price{Units: 1}}}, q.Record)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	b := density.Book{InstrumentID: "x", Lot: 1, Time: at,
		Bids: []density.Level{{Price: density.Price{Units: 101}, Quantity: 10}, {Price: density.Price{Units: 100}, Quantity: 100}},
		Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 100}}}
	e.OnBook(b, at)
	e.Observe(density.Signal{InstrumentID: "x", Price: density.Price{Units: 100}, Quantity: 100}, at)
	for n, trade := range []struct{ price, lots int64 }{{100, 3}, {108, 2}, {97, 1}} {
		now := at.Add(time.Duration(n+1) * time.Second)
		e.OnTrade(density.Event{InstrumentID: "x", Time: now, Price: density.Price{Units: trade.price}, Quantity: trade.lots}, now)
		if len(q.pending) != 0 {
			t.Fatal("alert before the remaining lot was sold")
		}
	}
	b.Time = at.Add(4 * time.Second)
	e.OnBook(b, b.Time)
	if e.Err() != nil {
		t.Fatal(e.Err())
	}
	if len(q.pending) != 1 {
		t.Fatalf("expected one completed trade: %+v", q.pending)
	}
	for _, want := range []string{"Купили бы по 101.0000 ₽", "Продали бы по 105.0000 ₽", "Плотность при входе: 100.0000 ₽", "Размер плотности: 100 лотов", "Стоп-лосс: 98.0000 ₽", "Тейк-профит: 107.0000 ₽", "Причина: Стоп-лосс", "2026-11-17 13:00:01 МСК", "2026-11-17 13:00:04 МСК"} {
		if !strings.Contains(q.pending[0].Text, want) {
			t.Fatalf("missing %q: %s", want, q.pending[0].Text)
		}
	}
}
