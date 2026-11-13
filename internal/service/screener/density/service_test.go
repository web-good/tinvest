package density

import (
	"context"
	"errors"
	"testing"
	"time"
)

type emptySource struct{}

func (emptySource) Instruments(context.Context) ([]UniverseInput, error)   { return nil, nil }
func (emptySource) Candles(context.Context, string) ([]DailyCandle, error) { return nil, nil }

type noStream struct{}

func (noStream) OpenStream(context.Context, map[string]int64) (Stream, error) {
	return nil, errors.New("must not connect to empty universe")
}

type testSender struct{}

func (testSender) Send(context.Context, Signal) error { return nil }
func TestServiceCancellationWithEmptyUniverse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	svc := NewService(DefaultConfig(), emptySource{}, noStream{}, testSender{}, time.Millisecond)
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked")
	}
}

type blockingStreamSource struct{ opened chan struct{} }
type blockingStream struct{ ctx context.Context }

func (f *blockingStreamSource) OpenStream(ctx context.Context, _ map[string]int64) (Stream, error) {
	close(f.opened)
	return &blockingStream{ctx}, nil
}
func (*blockingStream) SubscribeBooks([]string, int32, bool) error { return nil }
func (*blockingStream) SubscribeTrades([]string, bool) error       { return nil }
func (*blockingStream) SubscribeInfo([]string, bool) error         { return nil }
func (f *blockingStream) Receive() ([]Event, error)                { <-f.ctx.Done(); return nil, f.ctx.Err() }
func (*blockingStream) Close() error                               { return nil }
func TestConnectionStopsOnCancel(t *testing.T) {
	source := &blockingStreamSource{opened: make(chan struct{})}
	cfg := DefaultConfig()
	svc := NewService(cfg, emptySource{}, source, testSender{}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Lot: 10}}) }()
	<-source.opened
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("connection did not stop")
	}
}

type feedSource struct{ feed *feedStream }

func (f feedSource) OpenStream(ctx context.Context, _ map[string]int64) (Stream, error) {
	f.feed.ctx = ctx
	return f.feed, nil
}

type feedStream struct {
	ctx             context.Context
	events          chan []Event
	tradeSubscribed chan struct{}
}

func (f *feedStream) SubscribeBooks(ids []string, _ int32, _ bool) error {
	f.events <- []Event{{Kind: EventBookAck, InstrumentID: ids[0], Success: true}}
	return nil
}
func (f *feedStream) SubscribeInfo(ids []string, _ bool) error {
	f.events <- []Event{{Kind: EventInfoAck, InstrumentID: ids[0], Success: true}, {Kind: EventTradingStatus, InstrumentID: ids[0], TradingOpen: true}}
	return nil
}
func (f *feedStream) SubscribeTrades(ids []string, subscribe bool) error {
	if subscribe {
		f.events <- []Event{{Kind: EventTradeAck, InstrumentID: ids[0], Success: true}}
		select {
		case f.tradeSubscribed <- struct{}{}:
		default:
		}
	}
	return nil
}
func (f *feedStream) Receive() ([]Event, error) {
	select {
	case e := <-f.events:
		return e, nil
	case <-f.ctx.Done():
		return nil, f.ctx.Err()
	}
}
func (*feedStream) Close() error { return nil }

type captureSender struct{ sent chan Signal }

func (c captureSender) Send(ctx context.Context, s Signal) error {
	select {
	case c.sent <- s:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestConnectionSendsOnceAfterReadiness(t *testing.T) {
	feed := &feedStream{events: make(chan []Event, 20), tradeSubscribed: make(chan struct{}, 1)}
	sent := make(chan Signal, 10)
	cfg := DefaultConfig()
	cfg.Persistence = time.Millisecond
	cfg.TradeReadiness = time.Millisecond
	svc := NewService(cfg, emptySource{}, feedSource{feed}, captureSender{sent}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Ticker: "X", Lot: 100}}) }()
	book := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	for i := 0; i < 20; i++ {
		book.Time = time.Now()
		feed.events <- []Event{{Kind: EventBook, InstrumentID: "x", Book: book}}
		time.Sleep(100 * time.Millisecond)
	}
	select {
	case sig := <-sent:
		if sig.Ticker != "X" {
			t.Fatal(sig)
		}
	case <-time.After(time.Second):
		t.Fatal("signal missing")
	}
	select {
	case <-sent:
		t.Fatal("duplicate signal")
	default:
	}
	cancel()
	<-done
}

type slowSender struct {
	started chan struct{}
	release chan struct{}
	calls   chan Signal
}

func (s slowSender) Send(ctx context.Context, sig Signal) error {
	s.calls <- sig
	select {
	case s.started <- struct{}{}:
	default:
	}
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestQueuedSignalDroppedAfterBookInvalidated(t *testing.T) {
	feed := &feedStream{events: make(chan []Event, 50), tradeSubscribed: make(chan struct{}, 1)}
	sender := slowSender{started: make(chan struct{}, 1), release: make(chan struct{}), calls: make(chan Signal, 10)}
	cfg := DefaultConfig()
	cfg.Persistence = time.Millisecond
	cfg.TradeReadiness = time.Millisecond
	svc := NewService(cfg, emptySource{}, feedSource{feed}, sender, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Ticker: "X", Lot: 100}}) }()
	book := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 250}, {Price: Price{Units: 249}, Quantity: 250}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}, {Price: Price{Units: 244}, Quantity: 10}}}
	started := false
	for i := 0; i < 40 && !started; i++ {
		book.Time = time.Now()
		feed.events <- []Event{{Kind: EventBook, InstrumentID: "x", Book: book}}
		select {
		case <-sender.started:
			started = true
		case <-time.After(100 * time.Millisecond):
		}
	}
	if !started {
		t.Fatal("first send not started")
	}
	feed.events <- []Event{{Kind: EventInvalidBook, InstrumentID: "x"}}
	time.Sleep(100 * time.Millisecond)
	close(sender.release)
	time.Sleep(100 * time.Millisecond)
	if len(sender.calls) != 1 {
		t.Fatalf("stale queued signal sent: %d", len(sender.calls))
	}
	cancel()
	<-done
}

type cancellationSuccessSender struct{ started chan struct{} }

func (s cancellationSuccessSender) Send(ctx context.Context, _ Signal) error {
	close(s.started)
	<-ctx.Done()
	return nil
}
func TestConnectionKeepsSuccessDuringShutdown(t *testing.T) {
	feed := &feedStream{events: make(chan []Event, 40), tradeSubscribed: make(chan struct{}, 1)}
	sender := cancellationSuccessSender{started: make(chan struct{})}
	cfg := DefaultConfig()
	cfg.Persistence = time.Millisecond
	cfg.TradeReadiness = time.Millisecond
	svc := NewService(cfg, emptySource{}, feedSource{feed}, sender, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Ticker: "X", Lot: 100}}) }()
	book := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	started := false
	for i := 0; i < 40 && !started; i++ {
		book.Time = time.Now()
		feed.events <- []Event{{Kind: EventBook, InstrumentID: "x", Book: book}}
		select {
		case <-sender.started:
			started = true
		case <-time.After(100 * time.Millisecond):
		}
	}
	if !started {
		cancel()
		t.Fatal("send not started")
	}
	cancel()
	<-done
	if _, ok := svc.sent[key{"x", Price{Units: 250}}]; !ok {
		t.Fatal("successful send lost during cancellation")
	}
}

func TestSubscriptionFailureDoesNotDisconnectHealthyInstruments(t *testing.T) {
	feed := &feedStream{events: make(chan []Event, 20), tradeSubscribed: make(chan struct{}, 1)}
	svc := NewService(DefaultConfig(), emptySource{}, feedSource{feed}, testSender{}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Lot: 10}, {ID: "y", Lot: 10}}) }()
	feed.events <- []Event{{Kind: EventBookAck, InstrumentID: "y", Success: false}}
	select {
	case err := <-done:
		t.Fatalf("one failed instrument disconnected stream: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	cancel()
	<-done
}

type retrySender struct {
	calls chan int
	n     int
}

func (s *retrySender) Send(_ context.Context, _ Signal) error {
	s.n++
	s.calls <- s.n
	if s.n == 1 {
		return errors.New("temporary")
	}
	return nil
}
func TestSendFailureRetriesWhileEpisodeRemainsCurrent(t *testing.T) {
	feed := &feedStream{events: make(chan []Event, 50), tradeSubscribed: make(chan struct{}, 1)}
	sender := &retrySender{calls: make(chan int, 4)}
	cfg := DefaultConfig()
	cfg.Persistence = time.Millisecond
	cfg.TradeReadiness = time.Millisecond
	svc := NewService(cfg, emptySource{}, feedSource{feed}, sender, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Ticker: "X", Lot: 100}}) }()
	book := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(9 * time.Second)
	defer timeout.Stop()
	count := 0
	for count < 2 {
		select {
		case <-ticker.C:
			book.Time = time.Now()
			feed.events <- []Event{{Kind: EventBook, InstrumentID: "x", Book: book}}
		case <-sender.calls:
			count++
		case <-timeout.C:
			t.Fatalf("only %d send attempts", count)
		}
	}
	cancel()
	<-done
}

func TestNewEpisodeCooldown(t *testing.T) {
	for _, tc := range []struct {
		name    string
		elapsed time.Duration
		want    bool
	}{{"within cooldown", 29 * time.Minute, false}, {"after cooldown", 31 * time.Minute, true}} {
		t.Run(tc.name, func(t *testing.T) {
			feed := &feedStream{events: make(chan []Event, 50), tradeSubscribed: make(chan struct{}, 1)}
			sent := make(chan Signal, 10)
			cfg := DefaultConfig()
			cfg.Persistence = time.Millisecond
			cfg.TradeReadiness = time.Millisecond
			svc := NewService(cfg, emptySource{}, feedSource{feed}, captureSender{sent}, time.Millisecond)
			svc.sent[key{"x", Price{Units: 250}}] = time.Now().Add(-tc.elapsed)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Ticker: "X", Lot: 100}}) }()
			defer func() { cancel(); <-done }()
			b := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
			tick := time.NewTicker(50 * time.Millisecond)
			defer tick.Stop()
			timeout := time.NewTimer(2 * time.Second)
			defer timeout.Stop()
			for {
				select {
				case <-tick.C:
					b.Time = time.Now()
					feed.events <- []Event{{Kind: EventBook, InstrumentID: "x", Book: b}}
				case <-sent:
					if !tc.want {
						t.Fatal("sent during cooldown")
					}
					return
				case <-timeout.C:
					if tc.want {
						t.Fatal("signal missing after cooldown")
					}
					return
				}
			}
		})
	}
}

func TestQueueFullAllowsLaterRetry(t *testing.T) {
	jobs := make(chan sendJob, 1)
	pending := make(map[key]time.Time)
	now := time.Now()
	first := Signal{InstrumentID: "first", Price: Price{Units: 1}, Since: now}
	second := Signal{InstrumentID: "second", Price: Price{Units: 1}, Since: now}
	if !enqueueSignal(jobs, pending, first) {
		t.Fatal("first signal not queued")
	}
	if enqueueSignal(jobs, pending, second) {
		t.Fatal("full queue accepted signal")
	}
	if _, ok := pending[key{"second", second.Price}]; ok {
		t.Fatal("full queue marked signal pending")
	}
	<-jobs
	if !enqueueSignal(jobs, pending, second) {
		t.Fatal("signal could not be retried")
	}
}

func TestTradingHaltResetsObservation(t *testing.T) {
	feed := &feedStream{events: make(chan []Event, 100), tradeSubscribed: make(chan struct{}, 1)}
	sent := make(chan Signal, 10)
	cfg := DefaultConfig()
	cfg.Persistence = 2 * time.Second
	cfg.TradeReadiness = time.Millisecond
	svc := NewService(cfg, emptySource{}, feedSource{feed}, captureSender{sent}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Ticker: "X", Lot: 100}}) }()
	defer func() { cancel(); <-done }()
	b := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	pump := func(d time.Duration) {
		t.Helper()
		end := time.Now().Add(d)
		for time.Now().Before(end) {
			b.Time = time.Now()
			feed.events <- []Event{{Kind: EventBook, InstrumentID: "x", Book: b}}
			time.Sleep(100 * time.Millisecond)
		}
	}
	pump(1200 * time.Millisecond)
	feed.events <- []Event{{Kind: EventTradingStatus, InstrumentID: "x", TradingOpen: false}, {Kind: EventTradingStatus, InstrumentID: "x", TradingOpen: true}}
	pump(1100 * time.Millisecond)
	select {
	case <-sent:
		t.Fatal("halt did not reset persistence")
	default:
	}
	pump(1200 * time.Millisecond)
	select {
	case <-sent:
	case <-time.After(time.Second):
		t.Fatal("new episode did not mature")
	}
}

func TestPingDoesNotKeepObservationFresh(t *testing.T) {
	feed := &feedStream{events: make(chan []Event, 100), tradeSubscribed: make(chan struct{}, 1)}
	sent := make(chan Signal, 10)
	cfg := DefaultConfig()
	cfg.Persistence = 2 * time.Second
	cfg.Freshness = 300 * time.Millisecond
	cfg.TradeReadiness = time.Millisecond
	svc := NewService(cfg, emptySource{}, feedSource{feed}, captureSender{sent}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runConnection(ctx, svc, []Instrument{{ID: "x", Ticker: "X", Lot: 100}}) }()
	defer func() { cancel(); <-done }()
	b := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	pump := func(d time.Duration, books bool) {
		end := time.Now().Add(d)
		for time.Now().Before(end) {
			if books {
				b.Time = time.Now()
				feed.events <- []Event{{Kind: EventBook, InstrumentID: "x", Book: b}}
			} else {
				feed.events <- nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	pump(1200*time.Millisecond, true)
	pump(1200*time.Millisecond, false)
	pump(500*time.Millisecond, true)
	select {
	case <-sent:
		t.Fatal("ping kept old observation alive")
	default:
	}
}

func TestLateResultDoesNotMarkNewEpisode(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	old := Signal{InstrumentID: "x", Price: Price{Units: 250}, Since: now}
	newSince := now.Add(time.Minute)
	k := key{old.InstrumentID, old.Price}
	pending := map[key]time.Time{k: newSince}
	delivered := make(map[key]time.Time)
	sent := make(map[key]time.Time)
	retry := make(map[key]time.Time)
	applySendResult(sendResult{signal: old}, now.Add(2*time.Minute), pending, delivered, sent, retry)
	if !pending[k].Equal(newSince) || len(delivered) != 0 || len(sent) != 0 {
		t.Fatal("late response modified new episode")
	}
}

type refreshFailureSource struct{ calls int }

func (s *refreshFailureSource) Instruments(context.Context) ([]UniverseInput, error) {
	s.calls++
	if s.calls > 1 {
		return nil, errors.New("temporary refresh failure")
	}
	return []UniverseInput{{Instrument: Instrument{ID: "x", Ticker: "X", Lot: 10}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true}}, nil
}
func (*refreshFailureSource) Candles(context.Context, string) ([]DailyCandle, error) {
	out := make([]DailyCandle, 10)
	for i := range out {
		out[i] = DailyCandle{Time: time.Now().Add(-time.Duration(i+1) * 24 * time.Hour), Close: Price{Units: 100}, Volume: 100000, Complete: true}
	}
	return out, nil
}

type countStreamSource struct{ opened chan string }

func (s countStreamSource) OpenStream(ctx context.Context, lots map[string]int64) (Stream, error) {
	if lots["x"] > 0 {
		s.opened <- "x"
	}
	return &blockingStream{ctx}, nil
}
func TestRefreshFailureRetainsPreviousUniverse(t *testing.T) {
	source := &refreshFailureSource{}
	streams := countStreamSource{opened: make(chan string, 4)}
	svc := NewService(DefaultConfig(), source, streams, testSender{}, time.Millisecond)
	svc.refreshInterval = 100 * time.Millisecond
	svc.refreshRetry = 100 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()
	defer func() { cancel(); <-done }()
	for i := 0; i < 2; i++ {
		select {
		case <-streams.opened:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d stream starts", i)
		}
	}
}
