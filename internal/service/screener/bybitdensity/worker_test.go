package bybitdensity

import (
	"context"
	"flag"
	"io"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/feed"
	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/bybitdensity/strategy"
)

func TestDestinations(t *testing.T) {
	group := int64(-1001)
	screener := telegramDestination{chatID: group, topicID: 5809}
	simulation := telegramDestination{chatID: group, topicID: 3}
	var chats []telegramDestination
	send := func(_ context.Context, destination telegramDestination, _ string) error {
		chats = append(chats, destination)
		return nil
	}
	screens, fills := destinations(send, screener, simulation)
	for _, publish := range []func(context.Context, string) error{screens, fills} {
		if err := publish(context.Background(), "test"); err != nil {
			t.Fatal(err)
		}
	}
	if len(chats) != 2 || chats[0] != screener || chats[1] != simulation {
		t.Fatalf("wrong Telegram destinations: %v", chats)
	}
}

func TestOptions(t *testing.T) {
	t.Setenv("BYBIT_DENSITY_TELEGRAM_TOKEN", "test")
	t.Setenv("BYBIT_DENSITY_SCREENER_CHAT_ID", "-1001")
	t.Setenv("BYBIT_DENSITY_SIMULATION_CHAT_ID", "-1001")
	t.Setenv("BYBIT_DENSITY_SCREENER_TOPIC_ID", "0")
	t.Setenv("BYBIT_DENSITY_SIMULATION_TOPIC_ID", "0")
	t.Setenv("BYBIT_DENSITY_MIN_NOTIONAL", "150000")
	_, err := readOptions(nil, io.Discard, false)
	if err == nil || !strings.Contains(err.Error(), "different") {
		t.Fatalf("same destination accepted: %v", err)
	}
	t.Setenv("BYBIT_DENSITY_SIMULATION_CHAT_ID", "-1002")
	op, err := readOptions([]string{"-min-notional", "110000", "-spot-symbols", "btcusdt, ETHUSDT"}, io.Discard, false)
	if err != nil {
		t.Fatal(err)
	}
	if op.strategy.MinNotional != 110_000 || len(op.spotSymbols) != 2 || op.spotSymbols[0] != "BTCUSDT" {
		t.Fatal("invalid parsed options")
	}
	t.Setenv("BYBIT_DENSITY_SIMULATION_CHAT_ID", "-1001")
	t.Setenv("BYBIT_DENSITY_SCREENER_TOPIC_ID", "5809")
	t.Setenv("BYBIT_DENSITY_SIMULATION_TOPIC_ID", "3")
	t.Setenv("BYBIT_DENSITY_TELEGRAM_TOKEN", "")
	t.Setenv("TELEGRAM", "existing-bot")
	op, err = readOptions(nil, io.Discard, false)
	if err != nil || op.telegramToken != "existing-bot" ||
		op.screener != (telegramDestination{chatID: -1001, topicID: 5809}) ||
		op.simulation != (telegramDestination{chatID: -1001, topicID: 3}) {
		t.Fatalf("shared forum and existing bot setup failed: %v", err)
	}
	t.Setenv("BYBIT_DENSITY_SCREENER_TOPIC_ID", "3")
	if _, err := readOptions(nil, io.Discard, false); err == nil {
		t.Fatal("identical chat topics accepted")
	}
	if _, err := readOptions([]string{"-help"}, io.Discard, false); err != flag.ErrHelp {
		t.Fatalf("expected help: %v", err)
	}
}

func TestOptionsReadBybitEntryAndRiskFromEnv(t *testing.T) {
	t.Setenv("BYBIT_DENSITY_ENTRY_TICKS", "3")
	t.Setenv("BYBIT_DENSITY_STOP_TICKS", "4")
	t.Setenv("BYBIT_DENSITY_REWARD_RISK", "5")
	t.Setenv("DENSITY_LIVE_STOP_TICKS", "8")
	t.Setenv("DENSITY_LIVE_RR", "9")
	t.Setenv("DENSITY_LIVE_ENTRY_ZONE_TICKS", "10")
	op, err := readOptions(nil, io.Discard, true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg := op.strategy; cfg.EntryTicks != 3 || cfg.StopTicks != 4 || cfg.RewardRisk != 5 {
		t.Fatalf("incorrect entry and risk parameters: %+v", cfg)
	}
}

type replayMarket struct {
	advance <-chan struct{}
}

func (r replayMarket) Instruments(_ context.Context, market model.Market, _ []string, _ int) ([]model.Instrument, error) {
	if market != model.Spot {
		return nil, feed.ErrInvalidSymbols
	}
	return []model.Instrument{{Key: model.Key{Market: market, Symbol: "BTCUSDT"}, TickSize: .1, QtyStep: .1}}, nil
}

func (r replayMarket) AverageVolume(context.Context, model.Key, time.Time, time.Time) (float64, error) {
	return 9.25, nil
}

func (r replayMarket) Stream(ctx context.Context, instruments []model.Instrument, _ int, emit func(model.Event) error) error {
	key := instruments[0].Key
	end := time.Now().Truncate(time.Minute)
	makeBook := func(at time.Time, snapshot bool) *model.Book {
		book := &model.Book{At: at, Received: at, Snapshot: snapshot}
		for n := int64(0); n < 8; n++ {
			book.Bids = append(book.Bids, model.Level{Ticks: 1000 - n, Qty: 1})
			book.Asks = append(book.Asks, model.Level{Ticks: 1001 + n, Qty: 1})
		}
		book.Bids[0].Qty, book.Asks[0].Qty = 10, 1.5
		if at.Before(end) {
			for i := range book.Asks {
				book.Asks[i].Ticks++
			}
		}
		return book
	}
	for _, event := range []model.Event{
		{Book: makeBook(end.Add(-2*time.Minute-time.Second), true), TradeReady: true},
		{Book: makeBook(end.Add(-2*time.Minute), false)},
	} {
		event.Key = key
		if err := emit(event); err != nil {
			return err
		}
	}
	if key.Market == model.Spot {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.advance:
		}
		for _, event := range []model.Event{
			{Key: key, Candle: &model.Candle{Start: end.Add(-2 * time.Minute), End: end.Add(-time.Minute), Low: 995, Close: 1000}},
			{Key: key, Book: makeBook(end.Add(-time.Minute), false)},
			{Key: key, Book: makeBook(end.Add(-time.Second), false)},
			{Key: key, Candle: &model.Candle{Start: end.Add(-time.Minute), End: end, Low: 1001, Close: 1002}},
			{Key: key, Book: makeBook(end, false)},
		} {
			if err := emit(event); err != nil {
				return err
			}
		}
		take := makeBook(end.Add(time.Millisecond), false)
		take.Bids = append([]model.Level{{Ticks: 1007, Qty: 1.5}}, take.Bids...)
		for n := range take.Asks {
			take.Asks[n].Ticks += 9
		}
		if err := emit(model.Event{Key: key, Book: take}); err != nil {
			return err
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestReplayRoutesConfirmedDensityAndVirtualFills(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op := options{stateDir: t.TempDir(), depth: 200, symbolLimit: 1, strategy: strategy.DefaultConfig()}
	op.strategy.MinNotional = 500
	op.strategy.Persistence = time.Second
	op.strategy.Freshness = 3 * time.Minute
	advance := make(chan struct{})
	densities := make(chan string, 4)
	fills := make(chan string, 4)
	capture := func(dest chan<- string) sendMessage {
		return func(_ context.Context, message string) error {
			dest <- message
			return nil
		}
	}
	done := make(chan error, 1)
	go func() { done <- serve(ctx, replayMarket{advance}, op, capture(densities), capture(fills)) }()
	wait := func(source <-chan string) string {
		t.Helper()
		select {
		case message := <-source:
			return message
		case <-ctx.Done():
			t.Fatal("missing alert")
		case err := <-done:
			t.Fatalf("process stopped during replay: %v", err)
		}
		return ""
	}
	first := wait(densities)
	if !strings.Contains(first, "Спот · BTCUSDT") || !strings.Contains(first, "Средний объём M5: 9.25 монет") {
		t.Fatalf("spot market alert missing in %s", first)
	}
	close(advance)
	report := wait(fills)
	for _, want := range []string{
		"СИМУЛЯЦИЯ · Спот", "Сделка закрыта", "Купили по 100.1 USDT", "Продали по 100.7 USDT",
		"Плотность при входе — 100.0 USDT", "Средний объём M5 при входе: 9.25 монет", "Стоп-лосс — 99.8 USDT", "Тейк-профит — 100.7 USDT",
		"Причина — тейк", "Результат после комиссий +0.40% (+0.5988 USDT)",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("missing %q in %q", want, report)
		}
	}
	if strings.Count(report, "МСК") != 2 {
		t.Fatalf("expected entry and exit timestamps in %q", report)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestNewWorkerFromEnv(t *testing.T) {
	t.Setenv("BYBIT_DENSITY_ENABLED", "")
	t.Setenv("BYBIT_DENSITY_SCREENER_CHAT_ID", "not-a-chat")
	if worker, err := NewWorkerFromEnv(); worker != nil || err != nil {
		t.Fatal("disabled Bybit worker loaded its settings")
	}
	t.Setenv("BYBIT_DENSITY_ENABLED", "true")
	if worker, err := NewWorkerFromEnv(); worker != nil || err == nil {
		t.Fatal("enabled Bybit worker accepted an invalid destination")
	}
	t.Setenv("BYBIT_DENSITY_SCREENER_CHAT_ID", "-1001")
	t.Setenv("BYBIT_DENSITY_SCREENER_TOPIC_ID", "9686")
	t.Setenv("BYBIT_DENSITY_SIMULATION_CHAT_ID", "-1001")
	t.Setenv("BYBIT_DENSITY_SIMULATION_TOPIC_ID", "9688")
	t.Setenv("BYBIT_DENSITY_STATE_DIR", t.TempDir())
	t.Setenv("TELEGRAM", "test")
	if worker, err := NewWorkerFromEnv(); worker == nil || err != nil {
		t.Fatalf("enabled Bybit worker failed to load: %v", err)
	}
}

type solSelectionMarket struct{ replayMarket }

func (solSelectionMarket) Instruments(_ context.Context, market model.Market, symbols []string, _ int) ([]model.Instrument, error) {
	if len(symbols) == 0 {
		symbols = []string{"BTCUSDT"}
	}
	var instruments []model.Instrument
	for _, symbol := range symbols {
		instruments = append(instruments, model.Instrument{Key: model.Key{Market: market, Symbol: symbol}})
	}
	return instruments, nil
}

func TestSOLBreakoutIncludesSpotSOL(t *testing.T) {
	t.Setenv("BYBIT_DENSITY_SOL_BREAKOUT", "true")
	op, err := readOptions(nil, io.Discard, true)
	if err != nil {
		t.Fatal(err)
	}
	instruments, err := spotInstruments(t.Context(), solSelectionMarket{}, op)
	if err != nil || len(instruments) != 2 || instruments[0].Key.Symbol != "BTCUSDT" ||
		instruments[1].Key != (model.Key{Market: model.Spot, Symbol: "SOLUSDT"}) {
		t.Fatalf("SOLUSDT should be added to the normal selection: %+v, %v", instruments, err)
	}
}
