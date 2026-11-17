package live

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestNotificationsConfirmedFillsAndClose(t *testing.T) {
	state := State{}
	ep := &Episode{Instrument: density.Instrument{Ticker: "SBER", Lot: 10}, Entry: Order{Key: "trade", Kind: "buy"}, Reason: "take"}
	queueFillNotification(&state, ep, &ep.Entry, 0)
	if len(state.Notifications) != 0 {
		t.Fatal("accepted unfilled order generated BUY")
	}
	ep.Entry.Filled = 1
	ep.Entry.Amount = 2500 * 1e9
	queueFillNotification(&state, ep, &ep.Entry, 0)
	queueFillNotification(&state, ep, &ep.Entry, 0)
	queueFillNotification(&state, ep, &ep.Entry, 1)
	if len(state.Notifications) != 1 {
		t.Fatalf("duplicate fill events: %+v", state.Notifications)
	}
	buy := state.Notifications[0]
	if buy.ID != "trade:buy:1" || buy.Units != 10 || buy.Lots != 1 || buy.Amount != 2500*1e9 {
		t.Fatalf("wrong BUY: %+v", buy)
	}
	ep.Take.Child = &Order{Filled: 1, Amount: 2600 * 1e9}
	queueClosedNotification(&state, ep, 90*1e9, 10*1e9, false)
	queueClosedNotification(&state, ep, 90*1e9, 10*1e9, false)
	if len(state.Notifications) != 2 {
		t.Fatalf("duplicate close events: %+v", state.Notifications)
	}
	sell := state.Notifications[1]
	if sell.Amount != 2600*1e9 || sell.Reason != "take_profit" || sell.Realized != 90*1e9 || sell.Fees != 10*1e9 || sell.Complete {
		t.Fatalf("wrong SELL: %+v", sell)
	}
	text := FormatNotification(sell)
	for _, s := range []string{"Density SELL", "SBER", "260.0000", "Тейк-профит", "неполные", "trade"} {
		if !strings.Contains(text, s) {
			t.Errorf("missing %q in %s", s, text)
		}
	}
}

func TestFormatNotificationExplainsExitReason(t *testing.T) {
	for reason, want := range map[string]string{"density_lost": "Плотность исчезла", "stop_price": "Стоп-лосс", "take_profit": "Тейк-профит", "max_hold": "Истекло время удержания"} {
		text := FormatNotification(Notification{Side: "SELL", Reason: reason})
		if !strings.Contains(text, "Причина: "+want) {
			t.Fatalf("%s: %s", reason, text)
		}
	}
}

func TestConfirmedBrokerExitOverridesRequestedReason(t *testing.T) {
	for _, requested := range []string{"density_lost", "restart_reconcile"} {
		for _, kind := range []string{"stop", "take_profit"} {
			ep := &Episode{Instrument: density.Instrument{Ticker: "T", Lot: 10}, Entry: Order{Key: "trade", Filled: 1}, Reason: requested}
			child := &Order{Filled: 1, Amount: 1000 * Nano}
			if kind == "stop" {
				ep.Stop.Child = child
			} else {
				ep.Take.Child = child
			}
			state := State{}
			queueClosedNotification(&state, ep, 0, 0, false)
			if len(state.Notifications) != 1 || state.Notifications[0].Reason != kind {
				t.Fatalf("%s %s: %+v", requested, kind, state.Notifications)
			}
		}
	}
}

func TestNotificationDeliveryRetriesAndAcknowledgesWithoutDuplicates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	batches := make(chan []Notification, 3)
	acks := make(chan string, 3)
	warnings := make(chan error, 1)
	var attempts atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunNotificationDelivery(ctx, batches, acks, func(sendCtx context.Context, n Notification) error {
			if _, ok := sendCtx.Deadline(); !ok {
				t.Error("send missing deadline")
			}
			if attempts.Add(1) == 1 {
				return errors.New("telegram unavailable")
			}
			return nil
		}, func(err error) { warnings <- err })
	}()
	batch := []Notification{{ID: "trade:buy:1"}}
	batches <- batch
	select {
	case <-warnings:
	case <-time.After(time.Second):
		t.Fatal("failed send was not reported")
	}
	batches <- batch
	select {
	case id := <-acks:
		if id != batch[0].ID {
			t.Fatal(id)
		}
	case <-time.After(time.Second):
		t.Fatal("retry not acknowledged")
	}
	batches <- append(batch, Notification{ID: "other", Delivered: true})
	close(batches)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	if attempts.Load() != 2 {
		t.Fatalf("duplicate send after successful ack: %d", attempts.Load())
	}
}

func TestNotificationAcknowledgementSurvivesRestart(t *testing.T) {
	root, account := t.TempDir(), t.Name()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	store, err := OpenStore(root, account)
	if err != nil {
		t.Fatal(err)
	}
	state := store.State()
	state.Notifications = []Notification{{ID: "trade:buy:1", Side: "BUY", TradeID: "trade"}}
	if err = store.Save(state, "confirmed_fill"); err != nil {
		t.Fatal(err)
	}
	engine := &Engine{state: state, store: store}
	pending := engine.PendingNotifications()
	pending[0].Ticker = "mutated snapshot"
	if engine.PendingNotifications()[0].Ticker != "" {
		t.Fatal("snapshot aliases mutable state")
	}
	if err = engine.AcknowledgeNotification("trade:buy:1"); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(root, account)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	engine = &Engine{state: store.State(), store: store}
	if len(engine.PendingNotifications()) != 0 {
		t.Fatal("delivered event pending after restart")
	}
	if len(engine.state.Notifications) != 0 {
		t.Fatal("delivered record retained after restart")
	}
}

func TestNotificationDoesNotShowPartialSalesAsAveragePrice(t *testing.T) {
	state := State{}
	ep := &Episode{Instrument: density.Instrument{Ticker: "T", Lot: 10}, Entry: Order{Key: "trade", Filled: 2}, Exit: &Order{Filled: 1, Amount: 100 * 1e9}, Stop: Stop{Child: &Order{Filled: 1}}}
	queueClosedNotification(&state, ep, 0, 0, false)
	text := FormatNotification(state.Notifications[0])
	if !strings.Contains(text, "Средняя цена: неизвестна") {
		t.Fatalf("partial sale amount shown as average: %s", text)
	}
}

func TestNotificationAcknowledgementDoesNotRecreateLifecycleEvents(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	events := x.e.PendingNotifications()
	if len(events) != 1 || events[0].Side != "BUY" {
		t.Fatalf("missing confirmed BUY: %+v", events)
	}
	if err := x.e.AcknowledgeNotification(events[0].ID); err != nil {
		t.Fatal(err)
	}
	x.step(t)
	if len(x.e.state.Notifications) != 0 {
		t.Fatal("poll recreated acknowledged BUY")
	}
	x.book(499, false)
	for j := 0; j < 4 && len(x.e.state.Episodes) > 0; j++ {
		x.step(t)
	}
	events = x.e.PendingNotifications()
	if len(events) != 1 || events[0].Side != "SELL" {
		t.Fatalf("missing confirmed SELL: %+v", events)
	}
	if err := x.e.AcknowledgeNotification(events[0].ID); err != nil {
		t.Fatal(err)
	}
	x.step(t)
	if len(x.e.state.Notifications) != 0 {
		t.Fatal("closed episode recreated acknowledged SELL")
	}
}

func TestNotificationFlushAfterShutdownWithoutMarket(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	if err := x.e.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	var sent []Notification
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := x.e.FlushNotifications(ctx, func(_ context.Context, n Notification) error { sent = append(sent, n); return nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sent) != 2 || sent[0].Side != "BUY" || sent[1].Side != "SELL" {
		t.Fatalf("shutdown events not flushed: %+v", sent)
	}
	if len(x.e.state.Notifications) != 0 {
		t.Fatal("flush retained delivered events")
	}
}

func TestNotificationFlushKeepsFailedEventsDurable(t *testing.T) {
	x := setup(t)
	x.e.state.Notifications = []Notification{{ID: "failed"}}
	if err := x.e.save("confirmed_fill"); err != nil {
		t.Fatal(err)
	}
	var warnings atomic.Int32
	err := x.e.FlushNotifications(context.Background(), func(context.Context, Notification) error { return errors.New("telegram offline") }, func(error) { warnings.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	if len(x.e.state.Notifications) != 1 || warnings.Load() != 1 {
		t.Fatal("Telegram failure lost event or was not warned")
	}
}

func TestNotificationFlushReturnsJournalFailure(t *testing.T) {
	x := setup(t)
	x.e.state.Notifications = []Notification{{ID: "sent"}}
	if err := x.s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := x.e.FlushNotifications(context.Background(), func(context.Context, Notification) error { return nil }, nil); err == nil {
		t.Fatal("lost journal acknowledgement error")
	}
}

func TestNotificationFlushHonorsCallerDeadline(t *testing.T) {
	x := setup(t)
	x.e.state.Notifications = []Notification{{ID: "slow"}}
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- x.e.FlushNotifications(ctx, func(sendCtx context.Context, _ Notification) error {
			close(started)
			<-sendCtx.Done()
			return sendCtx.Err()
		}, nil)
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("flush ignored cancellation")
	}
	if len(x.e.state.Notifications) != 1 {
		t.Fatal("cancelled delivery lost durable event")
	}
}
