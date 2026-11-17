package live

import (
	"context"
	"errors"
	"time"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"

	"github.com/google/uuid"
)

// replaceEntry is reached only after the old BUY is terminal with zero fills.
// Its cancellation runs outside the account lease; the new allocation does not.
func (e *Engine) replaceEntry(ctx context.Context) (retErr error) {
	old := e.episode
	var lease *accountcoord.Lease
	if e.coordinator != nil {
		var err error
		lease, err = e.coordinator.TryLock()
		if errors.Is(err, accountcoord.ErrBusy) {
			return nil
		}
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, lease.Close()) }()
		if err = e.coordinator.Verify(ctx, episodeClaim(old)); err != nil {
			return err
		}
		pending, err := e.pendingCandleEntry(ctx, old.Instrument.Ticker)
		if err != nil || pending {
			return err
		}
	}
	plan, selected, err := e.feed.replacement(old, e.Now())
	if err != nil {
		return nil
	}
	if err = e.broker.CheckAccount(ctx); err != nil {
		return err
	}
	a, err := e.broker.Account(ctx)
	if err != nil {
		return err
	}
	if err = e.reconcileInventory(a, old.Instrument, 0); err != nil {
		return err
	}
	allowed, err := e.broker.CanBuy(ctx, old.Instrument.ID, plan.EntryPrice, 1)
	if err != nil || !allowed {
		return err
	}
	// Money RPCs can take longer than a book update, profile expiry or entry TTL.
	if err = e.reason(e.Now()); err != nil {
		return err
	}
	if old.Reason != "" {
		return nil
	}
	current, signal, err := e.feed.replacement(old, e.Now())
	if err != nil || signal.Price != selected.Price || !signal.Since.Equal(selected.Since) || current.EntryPrice != plan.EntryPrice {
		return nil
	}
	plan = current
	plan.Created = old.Created
	plan.ClaimKey = episodeClaim(old).Key
	plan.RetiredEntries = append(append([]Order(nil), old.RetiredEntries...), old.Entry)
	plan.Entry.Key, plan.Entry.Sent = uuid.NewString(), true
	if !e.feed.rebase(plan, signal, e.Now()) {
		return nil
	}
	e.episode = plan
	e.state.Episodes[plan.Instrument.ID] = plan
	if err = e.save("entry_replacement_intent"); err != nil {
		return err
	}
	if err = e.reason(e.Now()); err != nil {
		return err
	}
	if plan.Reason != "" || e.feed.moved(plan, e.Now()) {
		plan.Entry.Terminal, plan.Entry.Status = true, "not_submitted"
		plan.Repricing = plan.Reason == ""
		return e.save("replacement_not_submitted")
	}
	e.next = time.Time{}
	return e.submitEntry(ctx)
}
