package simulation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/paper"
)

type expandedMarket struct {
	marketFake
	next       chan struct{}
	tradeCalls atomic.Int32
}

func (m *expandedMarket) Instruments(context.Context) ([]density.UniverseInput, error) {
	var inputs []density.UniverseInput
	for n := 0; n < 11; n++ {
		inputs = append(inputs, density.UniverseInput{Instrument: density.Instrument{ID: fmt.Sprint(n), Ticker: fmt.Sprintf("EXTRA%d", n), Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true})
	}
	return inputs, nil
}
func (m *expandedMarket) InstrumentsByTickers(context.Context, []string) ([]density.UniverseInput, error) {
	return nil, errors.New("expanded mode requested fixed tickers")
}
func (m *expandedMarket) OpenStream(ctx context.Context, lots map[string]int64) (density.Stream, error) {
	if len(lots) != 11 {
		return nil, fmt.Errorf("got %d subscriptions, want 11", len(lots))
	}
	return &expandedStream{ctx: ctx, market: m}, nil
}

type expandedStream struct {
	ctx    context.Context
	market *expandedMarket
	step   int
}

func (*expandedStream) SubscribeBooks([]string, int32, bool) error { return nil }
func (*expandedStream) SubscribeInfo([]string, bool) error         { return nil }
func (s *expandedStream) SubscribeTrades([]string, bool) error {
	s.market.tradeCalls.Add(1)
	return errors.New("unexpected tape subscription")
}
func (*expandedStream) Close() error { return nil }
func (s *expandedStream) Receive() ([]density.Event, error) {
	s.step++
	if s.step > 1 {
		select {
		case <-s.market.next:
		case <-s.ctx.Done():
			return nil, s.ctx.Err()
		}
	}
	b := density.Book{InstrumentID: "10", Lot: 1, Time: time.Now(), Asks: []density.Level{{Price: density.Price{Units: 100, Nano: 200_000_000}, Quantity: 10}}}
	for n := int64(0); n < 6; n++ {
		qty := int64(1)
		if n == 0 {
			qty = 100000
		}
		b.Bids = append(b.Bids, density.Level{Price: density.Price{Units: 100 - n}, Quantity: qty})
	}
	events := []density.Event{{Kind: density.EventBook, InstrumentID: "10", Book: b}}
	if s.step == 1 {
		events = append([]density.Event{{Kind: density.EventBookAck, InstrumentID: "10", Success: true}, {Kind: density.EventInfoAck, InstrumentID: "10", Success: true}, {Kind: density.EventTradingStatus, InstrumentID: "10", TradingOpen: true}}, events...)
	}
	return events, nil
}

func TestSimulationFollowsAdditionalTickerInUniverseLargerThanTen(t *testing.T) {
	market := &expandedMarket{next: make(chan struct{}, 1)}
	var admitted atomic.Bool
	checked := make(chan struct{}, 2)
	cfg := paper.DefaultConfig()
	cfg.EntryPct = .3
	cfg.Latency = 0
	cfg.MaxEntryLots = 1
	detector := density.DefaultConfig()
	detector.Persistence = 0
	sent := make(chan string, 1)
	service := Service{Market: market, FollowScreener: true, AllowEntry: func(i density.Instrument, _ density.Signal) bool {
		allow := admitted.Load() && i.ID == "10"
		select {
		case checked <- struct{}{}:
		default:
		}
		return allow
	}, Config: cfg, Detector: detector, Tickers: []string{"BASE"}, Root: t.TempDir(), HistoryInterval: time.Nanosecond, Notify: func(_ context.Context, msg string) error { sent <- msg; return nil }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-checked:
	case err := <-done:
		done <- err
		t.Fatalf("service stopped: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("additional ticker not observed")
	}
	select {
	case msg := <-sent:
		t.Fatalf("entry before alert admission: %s", msg)
	default:
	}
	admitted.Store(true)
	market.next <- struct{}{}
	select {
	case msg := <-sent:
		if !strings.Contains(msg, "EXTRA10") || !strings.Contains(msg, "100.20") {
			t.Fatal(msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no entry for admitted additional ticker")
	}
	if market.tradeCalls.Load() != 0 {
		t.Fatal("expanded mode subscribed to tape")
	}
}
