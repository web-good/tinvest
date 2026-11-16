package live

import (
	"context"
	"fmt"
	"time"
)

// Notification is a durable, immutable snapshot of a confirmed trade event.
// Delivered is retained for compatibility with earlier journal snapshots.
type Notification struct {
	ID, Side, TradeID, Ticker, Reason   string
	Lots, Units, Amount, Fees, Realized int64
	Complete, Delivered, PriceKnown     bool
}

func queueFillNotification(state *State, ep *Episode, order *Order, previousFilled int64) {
	if order.Filled <= previousFilled || order.Filled <= 0 || order.Key != ep.Entry.Key {
		return
	}
	queueNotification(state, Notification{
		ID: fmt.Sprintf("%s:buy:%d", ep.Entry.Key, order.Filled), Side: "BUY", TradeID: ep.Entry.Key, Ticker: ep.Instrument.Ticker,
		Lots: order.Filled, Units: order.Filled * ep.Instrument.Lot, Amount: order.Amount, Complete: order.Amount > 0, PriceKnown: order.Amount > 0,
	})
}

func queueClosedNotification(state *State, ep *Episode, realized, fees int64, complete bool) {
	if ep.Entry.Filled <= 0 || ep.Remaining() != 0 {
		return
	}
	sales := int64(0)
	priceKnown := true
	for _, stop := range []*Stop{&ep.Stop, &ep.Take} {
		if stop.Child != nil {
			sales += stop.Child.Amount
			if stop.Child.Filled > 0 && stop.Child.Amount <= 0 {
				priceKnown = false
			}
		}
	}
	if ep.Exit != nil {
		sales += ep.Exit.Amount
		if ep.Exit.Filled > 0 && ep.Exit.Amount <= 0 {
			priceKnown = false
		}
	}
	queueNotification(state, Notification{ID: ep.Entry.Key + ":sell", Side: "SELL", TradeID: ep.Entry.Key, Ticker: ep.Instrument.Ticker, Reason: ep.Reason,
		Lots: ep.Entry.Filled, Units: ep.Entry.Filled * ep.Instrument.Lot, Amount: sales, Fees: fees, Realized: realized, Complete: complete, PriceKnown: priceKnown})
}

func queueNotification(state *State, event Notification) {
	for _, existing := range state.Notifications {
		if existing.ID == event.ID {
			return
		}
	}
	state.Notifications = append(state.Notifications, event)
}

// FormatNotification uses plain text so broker and ticker strings cannot inject markup.
func FormatNotification(n Notification) string {
	price := "неизвестна"
	if n.Amount > 0 && n.Units > 0 && (n.PriceKnown || n.Complete) {
		price = fmt.Sprintf("%.4f RUB", float64(n.Amount)/float64(n.Units)/1e9)
	}
	text := fmt.Sprintf("Density %s %s\nСредняя цена: %s\nЛоты: %d; акции: %d\nСделка: %s", n.Side, n.Ticker, price, n.Lots, n.Units, n.TradeID)
	if n.Side == "SELL" {
		text += fmt.Sprintf("\nПричина: %s\nРезультат с комиссиями: %+.2f RUB\nКомиссии: %.2f RUB", n.Reason, float64(n.Realized)/1e9, float64(n.Fees)/1e9)
	}
	if !n.Complete {
		text += "\nДанные неполные: цена, результат или комиссии требуют сверки"
	}
	return text
}

// PendingNotifications returns an independent snapshot for the delivery worker.
// It must be called by the same goroutine that owns the engine.
func (e *Engine) PendingNotifications() []Notification {
	var pending []Notification
	for _, n := range e.state.Notifications {
		if !n.Delivered {
			pending = append(pending, n)
		}
	}
	return pending
}

// AcknowledgeNotification durably removes a successful send on the engine owner.
func (e *Engine) AcknowledgeNotification(id string) error {
	for i := range e.state.Notifications {
		if e.state.Notifications[i].ID == id {
			remaining := make([]Notification, 0, len(e.state.Notifications)-1)
			remaining = append(remaining, e.state.Notifications[:i]...)
			remaining = append(remaining, e.state.Notifications[i+1:]...)
			e.state.Notifications = remaining
			return e.save("notification_delivered")
		}
	}
	return nil
}

// RunNotificationDelivery sends snapshots without accessing mutable engine state.
// Failed sends are retried when the owner supplies its next pending batch.
func RunNotificationDelivery(ctx context.Context, batches <-chan []Notification, acknowledgements chan<- string, notify func(context.Context, Notification) error, warn func(error)) {
	delivered := make(map[string]bool)
	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-batches:
			if !ok {
				return
			}
			current := make(map[string]bool, len(batch))
			for _, n := range batch {
				current[n.ID] = true
			}
			for id := range delivered {
				if !current[id] {
					delete(delivered, id)
				}
			}
			for _, n := range batch {
				if n.Delivered || delivered[n.ID] || notify == nil {
					continue
				}
				if ctx.Err() != nil {
					return
				}
				sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := notify(sendCtx, n)
				cancel()
				if err != nil {
					if warn != nil {
						warn(fmt.Errorf("density notification %s: %w", n.ID, err))
					}
					continue
				}
				delivered[n.ID] = true
				select {
				case acknowledgements <- n.ID:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// FlushNotifications attempts one batch after trading has stopped. Only the
// caller mutates the engine; Telegram failures leave the event for a later run.
// The caller supplies the bounded shutdown or recovery context.
func (e *Engine) FlushNotifications(ctx context.Context, notify func(context.Context, Notification) error, warn func(error)) error {
	if notify == nil {
		return nil
	}
	pending := e.PendingNotifications()
	if len(pending) == 0 {
		return nil
	}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	batches := make(chan []Notification, 1)
	batches <- pending
	close(batches)
	acknowledgements := make(chan string)
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunNotificationDelivery(workerCtx, batches, acknowledgements, notify, warn)
	}()
	for {
		select {
		case id := <-acknowledgements:
			if err := e.AcknowledgeNotification(id); err != nil {
				return err
			}
		case <-done:
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}
