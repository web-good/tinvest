package live

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

type booksOnlySource struct {
	events     chan []density.Event
	tradeCalls atomic.Int32
}

func (s *booksOnlySource) OpenStream(ctx context.Context, _ map[string]int64) (density.Stream, error) {
	return &booksOnlyStream{channelStream: channelStream{ctx: ctx, events: s.events}, source: s}, nil
}

type booksOnlyStream struct {
	channelStream
	source *booksOnlySource
}

func (s *booksOnlyStream) SubscribeTrades([]string, bool) error {
	s.source.tradeCalls.Add(1)
	return errors.New("unexpected trade subscription")
}

func TestBookOnlyConnectionGatesAdditionalTickersWithoutTradeSubscriptions(t *testing.T) {
	cfg := density.DefaultConfig()
	cfg.Persistence = 0
	cfg.MinNeighbors = 1
	cfg.MaxNeighbors = 1
	cfg.MinRubles = 1
	cfg.MinMultiple = 2
	f := NewFeed(DefaultConfig())
	source := &booksOnlySource{events: make(chan []density.Event, 4)}
	var allowed atomic.Bool
	checked := make(chan struct{}, 2)
	r := Runner{Source: source, BookOnly: true, Detector: cfg, Feed: f, Universe: []density.Instrument{{ID: "x", Ticker: "EXTRA", Lot: 1}}, AllowEntry: func(density.Instrument, density.Signal) bool {
		allow := allowed.Load()
		select {
		case checked <- struct{}{}:
		default:
		}
		return allow
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- r.connection(ctx) }()
	book := func() density.Event {
		return density.Event{Kind: density.EventBook, InstrumentID: "x", Book: density.Book{InstrumentID: "x", Lot: 1, Time: time.Now(), Bids: []density.Level{{Price: density.Price{Units: 100}, Quantity: 1000}, {Price: density.Price{Units: 99}, Quantity: 1}}, Asks: []density.Level{{Price: density.Price{Units: 101}, Quantity: 10}}}}
	}
	source.events <- []density.Event{{Kind: density.EventBookAck, InstrumentID: "x", Success: true}, {Kind: density.EventInfoAck, InstrumentID: "x", Success: true}, {Kind: density.EventTradingStatus, InstrumentID: "x", TradingOpen: true}, book()}
	select {
	case <-checked:
	case err := <-done:
		t.Fatalf("connection stopped: %v", err)
	case <-time.After(time.Second):
		t.Fatal("book not processed")
	}
	if len(f.Candidates(time.Now())) != 0 {
		t.Fatal("unannounced ticker admitted")
	}
	allowed.Store(true)
	source.events <- []density.Event{book()}
	deadline := time.After(time.Second)
	for len(f.Candidates(time.Now())) == 0 {
		select {
		case err := <-done:
			t.Fatalf("connection stopped: %v", err)
		case <-deadline:
			t.Fatal("announced ticker did not become eligible without trade ack")
		case <-time.After(time.Millisecond):
		}
	}
	allowed.Store(false)
	source.events <- []density.Event{book()}
	cancel()
	<-done
	if source.tradeCalls.Load() != 0 {
		t.Fatal("trade subscription used for extended universe")
	}
}
