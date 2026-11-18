package live

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"

	"github.com/google/uuid"
)

// ErrRejected means that the broker definitively rejected an unaccepted request.
// Transport errors must never be classified as a rejection.
var ErrRejected = errors.New("broker rejected request")

type Engine struct {
	coordinator *accountcoord.Coordinator
	cfg         Config
	broker      Broker
	store       *Store
	feed        *Feed
	instruments map[string]density.Instrument
	state       State
	episode     *Episode
	polls       map[string]time.Time
	entryNext   time.Time
	next        time.Time
	stopping    bool
	fatal       error
	Now         func() time.Time
}

func NewEngine(c Config, b Broker, s *Store, f *Feed, instruments []density.Instrument) (*Engine, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if b == nil || s == nil || f == nil || len(instruments) == 0 {
		return nil, fmt.Errorf("live engine requires broker, journal, feed and instruments")
	}
	e := &Engine{cfg: c, broker: b, store: s, feed: f, state: s.State(), instruments: map[string]density.Instrument{}, polls: map[string]time.Time{}, Now: time.Now}
	for _, i := range instruments {
		if i.ID == "" || i.Ticker == "" || i.Lot <= 0 || Price(i.PriceStep) <= 0 {
			return nil, fmt.Errorf("invalid instrument metadata")
		}
		if _, ok := e.instruments[i.ID]; ok {
			return nil, fmt.Errorf("duplicate instrument")
		}
		e.instruments[i.ID] = i
	}
	for id, ep := range e.state.Episodes {
		if ep == nil || id != ep.Instrument.ID {
			return nil, fmt.Errorf("invalid persisted episode index")
		}
		i, ok := e.instruments[ep.Instrument.ID]
		if !ok || i.Lot != ep.Instrument.Lot || i.PriceStep != ep.Instrument.PriceStep {
			return nil, fmt.Errorf("unresolved episode requires original instrument with matching lot and price step")
		}
		if ep.Entry.Key == "" || ep.Entry.Lots != 1 || ep.Remaining() < 0 {
			return nil, fmt.Errorf("invalid persisted episode; manual reconciliation required")
		}
		ep.Reason = "restart_reconcile"
		f.Watch(ep, e.Now())
	}
	return e, nil
}
func (e *Engine) State() State { return e.state }
func (e *Engine) Err() error   { return e.fatal }
func (e *Engine) save(event string) error {
	if e.fatal != nil {
		return e.fatal
	}
	rememberOwnership(&e.state)
	if err := e.store.Save(e.state, event); err != nil {
		e.fatal = err
		return err
	}
	e.state.LastEvent = event
	e.state.Updated = e.store.State().Updated
	return nil
}
func (e *Engine) block(reason string) error {
	e.episode.Blocked = reason
	return errors.Join(fmt.Errorf("live episode unresolved: %s", reason), e.save("blocked"))
}

func (e *Engine) reason(now time.Time) error {
	ep := e.episode
	if ep == nil || ep.Reason != "" {
		return nil
	}
	reason := e.feed.Reason(now)
	if reason == "" && ep.Entry.Filled == 0 && (now.Sub(ep.Created) >= e.cfg.EntryTTL || !ep.EntryUntil.IsZero() && !now.Before(ep.EntryUntil)) {
		reason = "entry_expired"
	}
	if reason == "" && !e.cfg.PinbarEntries && !ep.Opened.IsZero() && now.Sub(ep.Opened) >= e.cfg.MaxHold {
		reason = "max_hold"
	}
	if e.stopping {
		reason = "shutdown"
	}
	if reason != "" {
		ep.Reason = reason
		e.next = time.Time{}
		return e.save("exit_requested:" + reason)
	}
	return nil
}

// Step is called by one worker only. Each episode keeps its own polling clock.
func (e *Engine) Step(ctx context.Context, now time.Time) error {
	if e.fatal != nil {
		return e.fatal
	}
	if e.state.SharedRoot != "" && e.coordinator == nil {
		return fmt.Errorf("shared-account snapshot requires application coordinator")
	}
	var failed []error
	if err := e.releaseOwnership(ctx); err != nil {
		failed = append(failed, err)
	}
	ids := make([]string, 0, len(e.state.Episodes))
	for id := range e.state.Episodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if e.fatal != nil {
			return errors.Join(append(failed, e.fatal)...)
		}
		e.episode = e.state.Episodes[id]
		e.next = e.polls[id]
		if err := e.stepEpisode(ctx, now); err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", id, err))
		}
		if e.state.Episodes[id] != nil {
			e.polls[id] = e.next
		} else {
			delete(e.polls, id)
		}
	}
	e.episode = nil
	if e.fatal != nil {
		return errors.Join(append(failed, e.fatal)...)
	}
	if !e.stopping && !now.Before(e.entryNext) {
		e.entryNext = now.Add(e.cfg.PollInterval)
		if err := e.begin(ctx, now); err != nil {
			failed = append(failed, err)
		}
	}
	return errors.Join(failed...)
}

func (e *Engine) stepEpisode(ctx context.Context, now time.Time) error {
	ep := e.episode
	e.feed.Select(ep.Instrument.ID)
	if ep.EntryUntil.IsZero() && ep.Entry.Filled == 0 && (ep.Repricing || e.feed.moved(ep, now)) {
		e.next = time.Time{}
	}
	if err := e.reason(now); err != nil {
		return err
	}
	if now.Before(e.next) {
		return nil
	}
	e.next = now.Add(e.cfg.PollInterval)
	if e.coordinator != nil {
		if err := e.coordinator.Verify(ctx, episodeClaim(ep)); err != nil {
			return err
		}
	}
	if ep.Blocked != "" {
		return fmt.Errorf("live episode unresolved: %s", ep.Blocked)
	}
	if !ep.Entry.Terminal {
		if ep.Reason != "" {
			if err := e.cancelOrder(ctx, &ep.Entry); err != nil {
				return err
			}
		} else if err := e.pollOrder(ctx, &ep.Entry); err != nil {
			return err
		}
	}
	if ep.EntryUntil.IsZero() && ep.Entry.Filled == 0 && ep.Reason == "" && e.feed.moved(ep, e.Now()) {
		if !ep.Repricing {
			ep.Repricing = true
			if err := e.save("entry_reprice_requested"); err != nil {
				return err
			}
		}
		if !ep.Entry.Terminal {
			if err := e.cancelOrder(ctx, &ep.Entry); err != nil {
				return err
			}
		}
	}
	if ep.Entry.Filled > 0 && ep.Opened.IsZero() {
		ep.Repricing = false
		ep.Opened = e.Now()
		if err := e.save("entry_filled"); err != nil {
			return err
		}
	}
	if !ep.Entry.Terminal {
		if ep.Entry.Filled > 0 {
			ep.Reason = "partial_entry"
			return e.cancelOrder(ctx, &ep.Entry)
		}
		return nil
	}
	if ep.Entry.Filled == 0 {
		if ep.Repricing && ep.Reason == "" {
			return e.replaceEntry(ctx)
		}
		return e.finish(ctx, now)
	}
	if err := e.reason(e.Now()); err != nil {
		return err
	}
	if err := e.refreshStops(ctx); err != nil {
		return err
	}
	if ep.Remaining() < 0 {
		return e.block("confirmed sales exceed owned lots")
	}
	if ep.Remaining() == 0 && ep.Reason == "" {
		ep.Reason = "broker_exit"
		if ep.Stop.Child != nil && ep.Stop.Child.Filled > 0 {
			ep.Reason = "stop"
		} else if ep.Take.Child != nil && ep.Take.Child.Filled > 0 {
			ep.Reason = "take_profit"
		}
		if err := e.save("broker_exit"); err != nil {
			return err
		}
	}
	if ep.Stop.Status == "executed" && ep.Reason == "" {
		ep.Reason = "broker_stop"
		if err := e.save("broker_stop"); err != nil {
			return err
		}
	}
	for _, s := range []*Stop{&ep.Stop, &ep.Take} {
		if s.Sent && (s.Status == "cancelled" || s.Status == "expired" || s.Status == "rejected" || s.Child != nil && s.Child.Terminal && s.Child.Filled == 0) && ep.Reason == "" {
			ep.Reason = "protection_missing"
			if err := e.save("protection_missing"); err != nil {
				return err
			}
		}
	}
	if ep.Reason != "" {
		return e.exit(ctx, now)
	}
	if !ep.Stop.Sent {
		if err := e.placeStop(ctx, &ep.Stop, "stop", ep.StopPrice); err != nil {
			return err
		}
	}
	if err := e.reason(e.Now()); err != nil {
		return err
	}
	if ep.Reason != "" {
		return e.exit(ctx, now)
	}
	if !ep.Take.Sent {
		if err := e.placeStop(ctx, &ep.Take, "take", ep.TakePrice); err != nil {
			return err
		}
	}
	if err := e.reason(e.Now()); err != nil {
		return err
	}
	if ep.Reason != "" {
		return e.exit(ctx, now)
	}
	return nil
}

func (e *Engine) begin(ctx context.Context, now time.Time) (retErr error) {
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
	}
	candidates := e.feed.Candidates(now)
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i].Signal, candidates[j].Signal
		if a.InstrumentID != b.InstrumentID {
			return a.InstrumentID < b.InstrumentID
		}
		return Price(a.Price) > Price(b.Price)
	})
	for _, c := range candidates {
		i, ok := e.instruments[c.Signal.InstrumentID]
		if !ok || e.state.Episodes[i.ID] != nil || e.state.Releases[i.ID].Key != "" || now.Before(e.state.Cooldown[i.ID]) {
			continue
		}
		ep, err := plan(e.cfg, i, c.Signal, c.Book, now, c.EntryUntil)
		if err != nil {
			continue
		}
		if e.coordinator != nil {
			available, err := e.coordinator.CanEnter(lease, i.Ticker, "density")
			if err != nil {
				return err
			}
			if !available {
				continue
			}
			pending, err := e.pendingCandleEntry(ctx, i.Ticker)
			if err != nil {
				return err
			}
			if pending {
				continue
			}
		}
		if err = e.broker.CheckAccount(ctx); err != nil {
			return err
		}
		account, err := e.broker.Account(ctx)
		if err != nil {
			return err
		}
		if err = e.reconcileInventory(account, i, 0); err != nil {
			if errors.Is(err, errInventoryBusy) {
				continue
			}
			return err
		}
		allowed, err := e.broker.CanBuy(ctx, i.ID, ep.EntryPrice, 1)
		if err != nil {
			return err
		}
		if !allowed {
			continue
		}
		checkedPrice := ep.EntryPrice
		ep, err = e.feed.prepare(i, c, e.Now())
		if err != nil {
			return nil
		} // confirmation can expire during read-only broker checks
		if ep.EntryPrice > checkedPrice {
			e.feed.Unwatch(i.ID)
			return nil
		}
		if reason := e.feed.Reason(e.Now()); reason != "" {
			e.feed.Watch(nil, e.Now())
			return nil
		}
		ep.Entry.Key = uuid.NewString()
		ep.Entry.Sent = true
		e.episode = ep
		e.state.Episodes[i.ID] = ep
		if err = e.save("entry_intent"); err != nil {
			return err
		}
		if e.coordinator != nil {
			if err = e.coordinator.Reserve(lease, episodeClaim(ep)); err != nil {
				return e.block("ownership reservation failed before BUY: " + err.Error())
			}
		}
		// Check again after the fsync, immediately before submission.
		if err = e.reason(e.Now()); err != nil {
			return err
		}
		if ep.Reason != "" || !ep.EntryUntil.IsZero() && !e.feed.claimPinbar(i.ID, ep.EntryUntil, e.Now()) {
			ep.Entry.Terminal = true
			ep.Entry.Status = "not_submitted"
			return e.finish(ctx, now)
		}
		return e.submitEntry(ctx)
	}
	return nil
}

func (e *Engine) finish(ctx context.Context, now time.Time) error {
	ep := e.episode
	// Final accounting comes from order history, not an earlier placement response.
	orders := []*Order{&ep.Entry, ep.Exit, ep.Stop.Child, ep.Take.Child}
	for _, o := range orders {
		if o != nil && o.ID != "" {
			if err := e.pollOrder(ctx, o); err != nil {
				return err
			}
		}
	}
	a, err := e.broker.Account(ctx)
	if err != nil {
		return err
	}
	if err = e.reconcileInventory(a, ep.Instrument, 0); err != nil {
		return fmt.Errorf("closing reconciliation: %w", err)
	}
	fees, sales := ep.Entry.Fees, int64(0)
	known := func(o *Order) bool { return o.Filled == 0 || o.Amount > 0 && (o.FeesKnown || o.Fees > 0) }
	complete := known(&ep.Entry)
	for _, s := range []*Stop{&ep.Stop, &ep.Take} {
		if s.Child != nil {
			fees += s.Child.Fees
			sales += s.Child.Amount
			if !known(s.Child) {
				complete = false
			}
		}
	}
	if ep.Exit != nil {
		fees += ep.Exit.Fees
		sales += ep.Exit.Amount
		if !known(ep.Exit) {
			complete = false
		}
	}
	if ep.Entry.Filled > 0 {
		if ep.Remaining() != 0 {
			return e.block("cannot close episode with remaining inventory")
		}
		e.state.Closed++
		if complete {
			e.state.Realized += sales - ep.Entry.Amount - fees
		} else {
			e.state.PnLComplete = false
		}
	} else {
		e.state.Canceled++
	}
	e.state.Fees += fees
	e.state.Cooldown[ep.Instrument.ID] = now.Add(e.cfg.Cooldown)
	if e.coordinator != nil {
		claim := episodeClaim(ep)
		e.state.Releases[ep.Instrument.ID] = claim
	}
	queueClosedNotification(&e.state, ep, sales-ep.Entry.Amount-fees, fees, complete)
	delete(e.state.Episodes, ep.Instrument.ID)
	e.episode = nil
	e.feed.Unwatch(ep.Instrument.ID)
	if err = e.save("episode_closed"); err != nil {
		return err
	}
	return e.releaseOwnership(ctx)
}

func (e *Engine) Shutdown(ctx context.Context) error {
	e.stopping = true
	for id, ep := range e.state.Episodes {
		if ep.Reason == "" {
			ep.Reason = "shutdown"
		}
		e.polls[id] = time.Time{}
	}
	if err := e.save("shutdown"); err != nil {
		return err
	}
	timer := time.NewTicker(200 * time.Millisecond)
	defer timer.Stop()
	var last error
	for len(e.state.Episodes) > 0 || len(e.state.Releases) > 0 {
		if e.fatal != nil {
			return e.fatal
		}
		if err := e.Step(ctx, e.Now()); err != nil {
			last = err
		}
		if len(e.state.Episodes) == 0 && len(e.state.Releases) == 0 {
			return nil
		}
		allBlocked := len(e.state.Episodes) > 0 && len(e.state.Releases) == 0
		for _, ep := range e.state.Episodes {
			allBlocked = allBlocked && ep.Blocked != ""
		}
		if allBlocked {
			return errors.Join(fmt.Errorf("shutdown leaves unresolved broker state"), last)
		}
		select {
		case <-ctx.Done():
			return errors.Join(fmt.Errorf("shutdown incomplete; broker state retained"), ctx.Err(), last)
		case <-timer.C:
		}
	}
	return nil
}

func (e *Engine) submitEntry(ctx context.Context) error {
	ep := e.episode
	order, err := e.broker.PostOrder(ctx, OrderRequest{Key: ep.Entry.Key, InstrumentID: ep.Instrument.ID, Kind: "buy", Lots: 1, Price: ep.EntryPrice, Immediate: !ep.EntryUntil.IsZero()})
	if err != nil {
		if errors.Is(err, ErrRejected) {
			ep.Entry.Terminal = true
			ep.Entry.Status = "rejected"
			if saveErr := e.save("entry_rejected"); saveErr != nil {
				return saveErr
			}
			return e.finish(ctx, e.Now())
		}
		return fmt.Errorf("entry result unknown; request UUID retained: %w", err)
	}
	if err = e.applyOrder(&ep.Entry, order); err != nil {
		return err
	}
	if err = e.reason(e.Now()); err != nil {
		return err
	}
	// Immediately protect an entry that was filled in the placement response.
	if ep.Entry.Filled > 0 {
		e.next = time.Time{}
		return e.stepEpisode(ctx, e.Now())
	}
	// Loss during PostOrder is acted on before another polling interval.
	if ep.Reason != "" {
		e.next = time.Time{}
		return e.stepEpisode(ctx, e.Now())
	}
	return nil
}
