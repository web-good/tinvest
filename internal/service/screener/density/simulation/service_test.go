package simulation

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/paper"
)

type marketFake struct{}

func (marketFake) Instruments(context.Context) ([]density.UniverseInput, error) {
	return []density.UniverseInput{{Instrument: density.Instrument{ID: "x", Ticker: "X", Lot: 1, PriceStep: density.Price{Units: 1}}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true}}, nil
}
func (m marketFake) InstrumentsByTickers(ctx context.Context, tickers []string) ([]density.UniverseInput, error) {
	if len(tickers) != 1 || tickers[0] != "X" {
		return nil, errors.New("unexpected tickers")
	}
	return m.Instruments(ctx)
}
func (marketFake) Candles(context.Context, string) ([]density.DailyCandle, error) {
	var c []density.DailyCandle
	for n := 1; n <= 20; n++ {
		c = append(c, density.DailyCandle{Time: time.Now().Add(-time.Duration(n) * 24 * time.Hour), Close: density.Price{Units: 100}, Volume: 2000000, Complete: true})
	}
	return c, nil
}
func (marketFake) OpenStream(ctx context.Context, _ map[string]int64) (density.Stream, error) {
	return &streamFake{ctx: ctx}, nil
}

type streamFake struct {
	ctx  context.Context
	step int
}

func (*streamFake) SubscribeBooks([]string, int32, bool) error { return nil }
func (*streamFake) SubscribeTrades([]string, bool) error       { return nil }
func (*streamFake) SubscribeInfo([]string, bool) error         { return nil }
func (*streamFake) Close() error                               { return nil }
func (s *streamFake) Receive() ([]density.Event, error) {
	s.step++
	if s.step == 1 {
		var events []density.Event
		for _, k := range []density.EventKind{density.EventBookAck, density.EventTradeAck, density.EventInfoAck} {
			events = append(events, density.Event{Kind: k, InstrumentID: "x", Success: true})
		}
		events = append(events, density.Event{Kind: density.EventTradingStatus, InstrumentID: "x", TradingOpen: true})
		b := density.Book{InstrumentID: "x", Lot: 1, Time: time.Now(), Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}}}
		for n := int64(0); n < 6; n++ {
			qty := int64(1)
			if n == 0 {
				qty = 100000
			}
			b.Bids = append(b.Bids, density.Level{Price: density.Price{Units: 100 - n}, Quantity: qty})
		}
		return append(events, density.Event{Kind: density.EventBook, InstrumentID: "x", Book: b}), nil
	}
	if s.step == 2 || s.step == 3 {
		timer := time.NewTimer(30 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-s.ctx.Done():
			return nil, s.ctx.Err()
		}
		price := int64(100)
		if s.step == 3 {
			price = 108
		}
		return []density.Event{{Kind: density.EventTrade, InstrumentID: "x", Time: time.Now(), Price: density.Price{Units: price}, Quantity: 10}}, nil
	}
	<-s.ctx.Done()
	return nil, s.ctx.Err()
}

func TestServiceAlertsOnlyAfterVirtualTakeClosesTrade(t *testing.T) {
	cfg := paper.DefaultConfig()
	cfg.MaxEntryLots = 1
	cfg.Latency = 10 * time.Millisecond
	detector := density.DefaultConfig()
	detector.Persistence = 0
	detector.TradeReadiness = 0
	sent := make(chan string, 1)
	service := Service{Market: marketFake{}, Config: cfg, Detector: detector, Tickers: []string{"X"}, Root: t.TempDir(), HistoryInterval: time.Millisecond, Notify: func(_ context.Context, msg string) error { sent <- msg; return nil }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	select {
	case msg := <-sent:
		if !strings.Contains(msg, "Купили бы") || !strings.Contains(msg, "Продали бы") || !strings.Contains(msg, "Причина: Тейк-профит") {
			cancel()
			<-done
			t.Fatalf("incomplete trade alert: %s", msg)
		}
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatalf("no execution alert, service error: %v", <-done)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
