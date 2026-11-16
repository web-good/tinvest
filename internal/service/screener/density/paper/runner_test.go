package paper

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestJournalPersistsRecordsAndSummaryAndReportsFailures(t *testing.T) {
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = j.Record(Record{Kind: "test", At: start}); err != nil {
		t.Fatal(err)
	}
	if err = j.WriteSummary(engine(t).Summary(start)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(j.Dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s Summary
	if err = json.Unmarshal(data, &s); err != nil || s.Cash != 10000 {
		t.Fatalf("%s %v", data, err)
	}
	if err = j.Close(); err != nil {
		t.Fatal(err)
	}
	if err = j.Record(Record{}); err == nil {
		t.Fatal("closed journal write accepted")
	}
}

type fakeSource struct{ events []density.Event }

func (f fakeSource) OpenStream(ctx context.Context, _ map[string]int64) (density.Stream, error) {
	return &fakeStream{ctx: ctx, events: f.events}, nil
}

type fakeStream struct {
	ctx    context.Context
	events []density.Event
	sent   bool
}

func (f *fakeStream) SubscribeBooks([]string, int32, bool) error { return nil }
func (f *fakeStream) SubscribeTrades([]string, bool) error       { return nil }
func (f *fakeStream) SubscribeInfo([]string, bool) error         { return nil }
func (f *fakeStream) Close() error                               { return nil }
func (f *fakeStream) Receive() ([]density.Event, error) {
	if !f.sent {
		f.sent = true
		return f.events, nil
	}
	<-f.ctx.Done()
	return nil, f.ctx.Err()
}
func TestRunnerRequiresAcksAndOpenStatus(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(map[bool]string{false: "not_ready", true: "ready"}[ready], func(t *testing.T) {
			e := engine(t)
			e.cfg.Freshness = time.Minute
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			now := time.Now().Add(-time.Millisecond)
			b := book(now, 100000)
			b.Bids = append(b.Bids, density.Level{Price: p(98), Quantity: 1}, density.Level{Price: p(97), Quantity: 1}, density.Level{Price: p(96), Quantity: 1})
			events := []density.Event{}
			if ready {
				for _, k := range []density.EventKind{density.EventBookAck, density.EventTradeAck, density.EventInfoAck} {
					events = append(events, density.Event{Kind: k, InstrumentID: "x", Success: true})
				}
				events = append(events, density.Event{Kind: density.EventTradingStatus, InstrumentID: "x", TradingOpen: true})
			}
			events = append(events, density.Event{Kind: density.EventBook, InstrumentID: "x", Book: b})
			next := b
			next.Time = now.Add(time.Microsecond)
			events = append(events, density.Event{Kind: density.EventBook, InstrumentID: "x", Book: next})
			dc := density.DefaultConfig()
			dc.Persistence = time.Nanosecond
			dc.TradeReadiness = 0
			r := Runner{Source: fakeSource{events}, Universe: []density.Instrument{{ID: "x", Ticker: "X", Lot: 1, PriceStep: p(1)}}, Engine: e, Detector: dc, ReportInterval: time.Millisecond, Report: func(Summary) error { cancel(); return nil }}
			err := r.Run(ctx)
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if ready && len(e.orders) == 0 {
				t.Fatal("no confirmed density entry")
			}
			if !ready && len(e.orders) > 0 {
				t.Fatal("entry without acknowledgments")
			}
		})
	}
}

func TestForwardTestWritesClosedTradeAndNetProfit(t *testing.T) {
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	e := engine(t)
	e.record = j.Record
	if err = j.Session(e.cfg, density.DefaultConfig(), nil); err != nil {
		t.Fatal(err)
	}
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	trade(e, start.Add(3*time.Second), 108, 2)
	e.Stop(start.Add(4 * time.Second))
	s := e.Summary(start.Add(4 * time.Second))
	if s.ClosedTrades != 1 || s.NetPnL == nil || math.Abs(*s.NetPnL-11.584) > 1e-8 || math.Abs(*s.NetPnL-s.RealizedPnL) > 1e-8 {
		t.Fatalf("accounting %+v", s)
	}
	if err = j.WriteSummary(s); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(j.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	fills := 0
	for {
		var r Record
		err := dec.Decode(&r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if r.Kind == "entry_filled" || r.Kind == "exit_filled" {
			fills++
		}
	}
	if fills != 2 {
		t.Fatalf("missing persisted fills: %d", fills)
	}
}
