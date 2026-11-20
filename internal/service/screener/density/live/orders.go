package live

import (
	"context"
	"errors"
	"fmt"
	"time"
	"tinvest/internal/service/screener/density"

	"github.com/google/uuid"
)

func (e *Engine) applyOrder(dst *Order, got Order) error {
	if got.InstrumentID != "" && got.InstrumentID != e.episode.Instrument.ID || got.Kind != "" && (dst.Kind == "buy") != (got.Kind == "buy") || dst.Key != "" && got.Key != "" && dst.Key != got.Key {
		return e.block("broker order instrument, side or request UUID mismatch")
	}
	if got.ID == "" || got.Lots != dst.Lots || got.Filled < dst.Filled || got.Filled < 0 || got.Filled > dst.Lots || dst.ID != "" && dst.ID != got.ID || dst.Terminal && !got.Terminal || got.Amount < 0 || got.Fees < 0 {
		return e.block("inconsistent broker order status")
	}
	previousFilled := dst.Filled
	dst.ID = got.ID
	dst.Filled = got.Filled
	dst.Amount = got.Amount
	dst.Fees = got.Fees
	dst.FeesKnown = got.FeesKnown
	dst.Terminal = got.Terminal
	dst.Status = got.Status
	if ep := e.episode; dst == &ep.Entry && dst.Filled > 0 {
		if (e.cfg.FixedLimitEntry || !ep.EntryUntil.IsZero()) && !ep.Stop.Sent && !ep.Take.Sent && dst.Amount > 0 {
			step := Price(ep.Instrument.PriceStep)
			entry := dst.Amount / ep.Instrument.Lot / dst.Filled
			stop := ep.StopPrice
			if e.cfg.FixedLimitEntry {
				stop = density.EntryStop(entry, step)
			}
			if take, ok := density.TakeProfit(entry, stop, step, e.cfg.RewardRisk); ok && (e.cfg.FixedLimitEntry || entry%step == 0) {
				ep.StopPrice, ep.TakePrice = stop, take
			}
		}
		e.feed.filled(ep, e.Now())
	}
	queueFillNotification(&e.state, e.episode, dst, previousFilled)
	return e.save("order_status:" + dst.Kind)
}
func (e *Engine) pollOrder(ctx context.Context, o *Order) error {
	id, request := o.ID, false
	if id == "" {
		id, request = o.Key, true
	}
	got, err := e.broker.Order(ctx, id, request)
	if err != nil {
		return fmt.Errorf("resolve %s order: %w", o.Kind, err)
	}
	return e.applyOrder(o, got)
}
func (e *Engine) cancelOrder(ctx context.Context, o *Order) error {
	if o.Terminal {
		return nil
	}
	o.CancelRequested = true
	if err := e.save("cancel_order_intent:" + o.Kind); err != nil {
		return err
	}
	id, request := o.ID, false
	if id == "" {
		id, request = o.Key, true
	}
	cancelErr := e.broker.CancelOrder(ctx, id, request)
	// A successful cancellation response is not a replacement for fill reconciliation.
	if err := e.pollOrder(ctx, o); err != nil {
		return errors.Join(cancelErr, err)
	}
	if o.Terminal {
		return nil
	}
	return cancelErr
}
func (e *Engine) placeStop(ctx context.Context, s *Stop, kind string, price int64) error {
	ep := e.episode
	s.Key = uuid.NewString()
	s.Kind = kind
	s.Sent = true
	if err := e.save("stop_intent:" + kind); err != nil {
		return err
	}
	id, err := e.broker.PostStop(ctx, StopRequest{Key: s.Key, InstrumentID: ep.Instrument.ID, Kind: kind, Lots: ep.Remaining(), Price: price})
	if err != nil {
		if errors.Is(err, ErrRejected) {
			s.Status = "rejected"
			ep.Reason = "protection_rejected"
			if saveErr := e.save("protection_rejected"); saveErr != nil {
				return saveErr
			}
			return nil
		}
		return e.block("unknown " + kind + " placement; retain UUID and reconcile manually: " + err.Error())
	}
	if id == "" {
		return e.block("empty broker stop ID; placement result unknown")
	}
	s.ID = id
	for _, other := range []*Stop{&ep.Stop, &ep.Take} {
		if other != s && other.ID == id {
			return e.block("broker returned the same ID for two stop orders")
		}
	}
	s.Status = "active"
	return e.save("stop_accepted:" + kind)
}

func (e *Engine) refreshStops(ctx context.Context) error {
	ep := e.episode
	needed := false
	for _, s := range []*Stop{&ep.Stop, &ep.Take} {
		if s.Sent && s.Status != "rejected" {
			if s.ID == "" {
				return e.block("unknown persisted stop placement; manual reconciliation required")
			}
			needed = true
		}
	}
	if !needed {
		return nil
	}
	list, err := e.broker.Stops(ctx, ep.Created.Add(-time.Minute))
	if err != nil {
		return err
	}
	for _, s := range []*Stop{&ep.Stop, &ep.Take} {
		if !s.Sent || s.Status == "rejected" {
			continue
		}
		var found *BrokerStop
		for j := range list {
			if list[j].ID == s.ID {
				if found != nil {
					return e.block("duplicate stop IDs in broker response")
				}
				found = &list[j]
			}
		}
		if found == nil {
			return fmt.Errorf("known %s stop absent from broker history; no competing sale allowed", s.Kind)
		}
		got := *found
		if got.InstrumentID != "" && got.InstrumentID != ep.Instrument.ID {
			return e.block("stop belongs to another instrument")
		}
		switch got.Status {
		case "active", "executed", "cancelled", "expired":
		default:
			return fmt.Errorf("unknown broker stop status")
		}
		if s.ChildID != "" && got.ChildID != "" && s.ChildID != got.ChildID {
			return e.block("broker changed child order ID")
		}
		s.Status = got.Status
		if got.ChildID != "" {
			for _, other := range []*Stop{&ep.Stop, &ep.Take} {
				if other != s && other.ChildID == got.ChildID {
					return e.block("two stops reference the same child sale")
				}
			}
			s.ChildID = got.ChildID
			if s.Child == nil {
				s.Child = &Order{ID: s.ChildID, Kind: s.Kind, Lots: ep.Entry.Filled, Sent: true}
			}
		}
		if (got.Status == "executed" || got.Activated) && s.ChildID == "" {
			return fmt.Errorf("activated %s stop has no child order ID yet; sale blocked", s.Kind)
		}
		if s.Child != nil {
			if err = e.pollOrder(ctx, s.Child); err != nil {
				return err
			}
		}
	}
	return e.save("stops_reconciled")
}

func (e *Engine) settleStops(ctx context.Context) error {
	ep := e.episode
	for _, s := range []*Stop{&ep.Take, &ep.Stop} {
		if !s.Sent || s.Status == "rejected" {
			continue
		}
		if s.ID == "" {
			return e.block("cannot cancel unknown stop placement")
		}
		if s.Status == "active" {
			s.CancelRequested = true
			if err := e.save("cancel_stop_intent:" + s.Kind); err != nil {
				return err
			}
			cancelErr := e.broker.CancelStop(ctx, s.ID)
			// Re-read both stops. Activation can race the cancellation of either one.
			if err := e.refreshStops(ctx); err != nil {
				return errors.Join(cancelErr, err)
			}
			if s.Status == "active" {
				return errors.Join(cancelErr, fmt.Errorf("stop cancellation not terminal"))
			}
		}
		if s.Status == "executed" && s.Child == nil {
			return fmt.Errorf("activated child not yet resolved")
		}
		if s.Child != nil && !s.Child.Terminal {
			if err := e.cancelOrder(ctx, s.Child); err != nil {
				return err
			}
			if !s.Child.Terminal {
				return fmt.Errorf("child sale still active")
			}
		}
	}
	return nil
}

func (e *Engine) exit(ctx context.Context, now time.Time) error {
	ep := e.episode
	if !ep.Entry.Terminal {
		if err := e.cancelOrder(ctx, &ep.Entry); err != nil {
			return err
		}
		if !ep.Entry.Terminal {
			return nil
		}
	}
	if ep.Remaining() > 0 {
		if err := e.broker.CheckAccount(ctx); err != nil {
			return err
		}
		if ep.Exit == nil {
			if err := e.checkExitInventory(ctx); err != nil {
				return err
			}
		}
	}
	if err := e.settleStops(ctx); err != nil {
		return err
	}
	if ep.Exit != nil {
		if !ep.Exit.Terminal {
			if err := e.pollOrder(ctx, ep.Exit); err != nil {
				return err
			}
		}
		if !ep.Exit.Terminal {
			return nil
		}
		if ep.Remaining() > 0 {
			return e.block("market exit terminated without closing inventory; manual action required")
		}
	}
	left := ep.Remaining()
	if left < 0 {
		return e.block("sales exceeded owned lots")
	}
	if left == 0 {
		return e.finish(ctx, now)
	}
	a, err := e.broker.Account(ctx)
	if err != nil {
		return err
	}
	if err = e.reconcileInventory(a, ep.Instrument, left); err != nil {
		// Another owner's execution can change the aggregate during cancellation.
		// Do not leave our confirmed remainder naked while its state catches up.
		if a.Holdings[ep.Instrument.ID].Units >= left*ep.Instrument.Lot && ep.Stop.Status == "cancelled" && (ep.Stop.Child == nil || ep.Stop.Child.Terminal && ep.Stop.Child.Filled == 0) {
			ep.Stop = Stop{}
			return errors.Join(err, e.placeStop(ctx, &ep.Stop, "stop", ep.StopPrice))
		}
		return err
	}
	ep.Exit = &Order{Key: uuid.NewString(), Kind: "exit", Lots: left, Sent: true}
	if err = e.save("market_exit_intent"); err != nil {
		return err
	}
	got, err := e.broker.PostOrder(ctx, OrderRequest{Key: ep.Exit.Key, InstrumentID: ep.Instrument.ID, Kind: "exit", Lots: left})
	if err != nil {
		if errors.Is(err, ErrRejected) {
			ep.Exit.Terminal = true
			ep.Exit.Status = "rejected"
			return e.block("market exit rejected; inventory remains open")
		}
		return fmt.Errorf("market exit result unknown; same UUID retained: %w", err)
	}
	if err = e.applyOrder(ep.Exit, got); err != nil {
		return err
	}
	if ep.Exit.Terminal && ep.Remaining() == 0 {
		return e.finish(ctx, now)
	}
	return nil
}
