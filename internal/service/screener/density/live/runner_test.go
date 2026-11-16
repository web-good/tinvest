package live

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

type sourceFake struct {
	events   []density.Event
	received chan struct{}
}

func TestSlowTelegramDoesNotDelayDensityExit(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	x.e.Now = time.Now
	sending, exited := make(chan struct{}), make(chan struct{})
	var once sync.Once
	x.b.onPost = func() { once.Do(func() { close(exited) }) }
	r := Runner{Source: channelSource{events: make(chan []density.Event)}, Universe: []density.Instrument{x.i}, Detector: density.DefaultConfig(), Feed: x.f, Engine: x.e, Notify: func(ctx context.Context, _ Notification) error {
		close(sending)
		<-ctx.Done()
		return ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	select {
	case <-sending:
	case <-time.After(time.Second):
		cancel()
		<-done
		t.Fatal("notification worker did not start")
	}
	x.f.Gap(time.Now())
	select {
	case <-exited:
	case <-time.After(time.Second):
		cancel()
		<-done
		t.Fatal("blocked Telegram send delayed trading exit")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(x.b.posts) != 2 || x.e.State().Closed != 1 {
		t.Fatal("density trade not closed")
	}
}

type channelSource struct{ events chan []density.Event }

func (s channelSource) OpenStream(ctx context.Context, _ map[string]int64) (density.Stream, error) {
	return &channelStream{ctx: ctx, events: s.events}, nil
}

type channelStream struct {
	ctx    context.Context
	events chan []density.Event
}

func (s *channelStream) SubscribeBooks([]string, int32, bool) error { return nil }
func (s *channelStream) SubscribeInfo([]string, bool) error         { return nil }
func (s *channelStream) SubscribeTrades([]string, bool) error       { return nil }
func (s *channelStream) Close() error                               { return nil }
func (s *channelStream) Receive() ([]density.Event, error) {
	select {
	case ev := <-s.events:
		return ev, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func TestRunnerProcessesDensityLossWhileBrokerRPCBlocked(t *testing.T) {
	x := setup(t)
	x.e.Now = time.Now
	cfg := density.DefaultConfig()
	cfg.MinNeighbors = 1
	cfg.MaxNeighbors = 1
	cfg.MinRubles = 1
	cfg.MinMultiple = 2
	cfg.Persistence = time.Nanosecond
	cfg.TradeReadiness = 0
	x.f.Gap(time.Now())
	source := channelSource{events: make(chan []density.Event, 4)}
	posting, release, canceled := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	x.b.onPost = func() { close(posting); <-release }
	x.b.onCancel = func(id string) {
		o := x.b.orders[id]
		o.Terminal = true
		o.Status = "cancelled"
		x.b.orders[id] = o
		once.Do(func() { close(canceled) })
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := Runner{Source: source, Universe: []density.Instrument{x.i}, Detector: cfg, Feed: x.f, Engine: x.e}
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	var acks []density.Event
	for _, kind := range []density.EventKind{density.EventBookAck, density.EventInfoAck, density.EventTradeAck} {
		acks = append(acks, density.Event{Kind: kind, InstrumentID: "uid", Success: true})
	}
	acks = append(acks, density.Event{Kind: density.EventTradingStatus, InstrumentID: "uid", TradingOpen: true})
	book := func(qty int64) []density.Event {
		return []density.Event{{Kind: density.EventBook, InstrumentID: "uid", Book: density.Book{InstrumentID: "uid", Lot: 10, Time: time.Now(), Bids: []density.Level{{Price: density.Price{Units: 100}, Quantity: qty}, {Price: density.Price{Units: 99}, Quantity: 1}}, Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}}}}}
	}
	source.events <- acks
	source.events <- book(1000)
	source.events <- book(1000)
	select {
	case <-posting:
	case <-time.After(2 * time.Second):
		t.Fatal("buy not submitted")
	}
	source.events <- book(499)
	source.events <- book(1000)
	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for x.f.Reason(time.Now()) != "density_lost" {
		select {
		case <-tick.C:
		case <-deadline:
			close(release)
			t.Fatal("market updates blocked by order RPC")
		}
	}
	close(release)
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("density loss did not cancel buy")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(x.b.posts) != 1 {
		t.Fatal("extra entry submitted")
	}
}

func (s sourceFake) OpenStream(ctx context.Context, _ map[string]int64) (density.Stream, error) {
	return &streamFake{ctx: ctx, events: s.events, received: s.received}, nil
}

type streamFake struct {
	ctx      context.Context
	events   []density.Event
	received chan struct{}
	sent     bool
}

func (s *streamFake) SubscribeBooks([]string, int32, bool) error { return nil }
func (s *streamFake) SubscribeInfo([]string, bool) error         { return nil }
func (s *streamFake) SubscribeTrades([]string, bool) error       { return nil }
func (s *streamFake) Close() error                               { return nil }
func (s *streamFake) Receive() ([]density.Event, error) {
	if !s.sent {
		s.sent = true
		return s.events, nil
	}
	if s.received != nil {
		close(s.received)
		s.received = nil
	}
	<-s.ctx.Done()
	return nil, s.ctx.Err()
}

func TestMarketFeedRequiresAcknowledgements(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing_acks", true: "ready"}[ready], func(t *testing.T) {
			cfg := DefaultConfig()
			f := NewFeed(cfg)
			now := time.Now().Add(-time.Millisecond)
			book := density.Book{InstrumentID: "uid", Lot: 10, Time: now, Bids: []density.Level{{Price: density.Price{Units: 100}, Quantity: 10000}, {Price: density.Price{Units: 99}, Quantity: 1}}, Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}}}
			var events []density.Event
			if ready {
				for _, kind := range []density.EventKind{density.EventBookAck, density.EventInfoAck, density.EventTradeAck} {
					events = append(events, density.Event{InstrumentID: "uid", Kind: kind, Success: true})
				}
				events = append(events, density.Event{Kind: density.EventTradingStatus, InstrumentID: "uid", TradingOpen: true})
			}
			events = append(events, density.Event{Kind: density.EventBook, InstrumentID: "uid", Book: book})
			book.Time = now.Add(time.Microsecond)
			events = append(events, density.Event{Kind: density.EventBook, InstrumentID: "uid", Book: book})
			dc := density.DefaultConfig()
			dc.MinNeighbors = 1
			dc.MaxNeighbors = 1
			dc.MinRubles = 1
			dc.MinMultiple = 2
			dc.Persistence = time.Nanosecond
			dc.TradeReadiness = 0
			ctx, cancel := context.WithCancel(context.Background())
			received := make(chan struct{})
			r := Runner{Source: sourceFake{events: events, received: received}, Feed: f, Universe: []density.Instrument{{ID: "uid", Lot: 10}}, Detector: dc}
			done := make(chan error, 1)
			go func() { done <- r.connection(ctx) }()
			select {
			case <-received:
			case <-time.After(time.Second):
				t.Fatal("feed stalled")
			}
			deadline := time.After(time.Second)
			wait := time.NewTicker(time.Millisecond)
			defer wait.Stop()
			for {
				f.mu.Lock()
				processed := f.watermarks["uid"].Equal(book.Time)
				f.mu.Unlock()
				if processed {
					break
				}
				select {
				case <-wait.C:
				case <-deadline:
					t.Fatal("last book not processed")
				}
			}
			got := len(f.Candidates(time.Now()))
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if (got > 0) != ready {
				t.Fatalf("ready=%v candidates=%d", ready, got)
			}
		})
	}
}

func TestTradeStopUsesFreshExchangeTimeWithoutInventingFill(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	x.f.Trade(density.Event{InstrumentID: "uid", Price: density.Price{Units: 98}, Quantity: 1, Time: x.now.Add(-time.Minute)}, x.now)
	if x.f.Reason(x.now) != "" {
		t.Fatal("stale trade triggered stop")
	}
	x.f.Trade(density.Event{InstrumentID: "uid", Price: density.Price{Units: 98}, Quantity: 1, Time: x.now}, x.now)
	if x.f.Reason(x.now) != "stop_trade" || len(x.b.posts) != 1 {
		t.Fatal("trade stop was ignored or invented a fill")
	}
}
