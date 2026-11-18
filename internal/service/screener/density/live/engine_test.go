package live

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"

	"github.com/google/uuid"
)

type fakeBroker struct {
	orders       map[string]Order
	stops        map[string]BrokerStop
	holdings     map[string]Holding
	posts        []OrderRequest
	protections  []StopRequest
	canceled     []string
	onCancel     func(string)
	onCancelStop func(string)
	onPost       func()
	onCanBuy     func()
	stopErr      error
	unknownChild bool
	marginErr    error
}

func orderInstrument(id string) string {
	if id == "" {
		return "uid"
	}
	return id
}

func (b *fakeBroker) CheckAccount(context.Context) error { return b.marginErr }
func (b *fakeBroker) Account(context.Context) (Account, error) {
	a := Account{Holdings: b.holdings, Stops: map[string]string{}}
	for _, r := range b.orders {
		if !r.Terminal {
			a.Orders = append(a.Orders, ActiveOrder{ID: r.ID, InstrumentID: orderInstrument(r.InstrumentID)})
		}
	}
	for _, s := range b.stops {
		if s.Status == "active" {
			a.Stops[s.ID] = orderInstrument(s.InstrumentID)
		}
	}
	return a, nil
}
func (b *fakeBroker) CanBuy(context.Context, string, int64, int64) (bool, error) {
	if b.onCanBuy != nil {
		b.onCanBuy()
	}
	return true, nil
}
func (b *fakeBroker) PostOrder(_ context.Context, r OrderRequest) (Order, error) {
	b.posts = append(b.posts, r)
	o := Order{ID: "order-" + r.Key, Key: r.Key, Kind: r.Kind, InstrumentID: r.InstrumentID, Lots: r.Lots, Status: "new", Sent: true}
	if r.Kind == "exit" {
		o.Filled = r.Lots
		o.Terminal = true
		o.Status = "filled"
		o.Amount = 990 * Nano
		o.Fees = Nano
		h := b.holdings[r.InstrumentID]
		h.Units -= r.Lots * 10
		b.holdings[r.InstrumentID] = h
	}
	b.orders[o.ID] = o
	if b.onPost != nil {
		b.onPost()
	}
	return o, nil
}
func (b *fakeBroker) Order(_ context.Context, id string, request bool) (Order, error) {
	for _, o := range b.orders {
		if o.ID == id || request && o.Key == id {
			return o, nil
		}
	}
	return Order{}, fmt.Errorf("order not yet visible")
}
func (b *fakeBroker) CancelOrder(_ context.Context, id string, _ bool) error {
	b.canceled = append(b.canceled, id)
	if b.onCancel != nil {
		b.onCancel(id)
	} else {
		o := b.orders[id]
		o.Terminal = true
		o.Status = "cancelled"
		b.orders[id] = o
	}
	return nil
}
func (b *fakeBroker) PostStop(_ context.Context, r StopRequest) (string, error) {
	b.protections = append(b.protections, r)
	if b.stopErr != nil {
		return "", b.stopErr
	}
	id := "stop-" + r.Key
	b.stops[id] = BrokerStop{ID: id, Status: "active", InstrumentID: r.InstrumentID}
	return id, nil
}
func (b *fakeBroker) Stops(context.Context, time.Time) ([]BrokerStop, error) {
	var out []BrokerStop
	for _, s := range b.stops {
		out = append(out, s)
	}
	return out, nil
}
func (b *fakeBroker) CancelStop(_ context.Context, id string) error {
	s := b.stops[id]
	if b.unknownChild {
		s.Status = "executed"
		s.Activated = true
		s.ChildID = ""
	} else if s.Status == "active" {
		s.Status = "cancelled"
	}
	b.stops[id] = s
	if b.onCancelStop != nil {
		b.onCancelStop(id)
	}
	return nil
}

type fixture struct {
	account string
	e       *Engine
	b       *fakeBroker
	f       *Feed
	s       *Store
	i       density.Instrument
	now     time.Time
}

func setup(t *testing.T) *fixture {
	t.Helper()
	now := time.Now()
	cfg := DefaultConfig()
	account := uuid.NewString()
	s, err := OpenStore(t.TempDir(), account)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	b := &fakeBroker{orders: map[string]Order{}, stops: map[string]BrokerStop{}, holdings: map[string]Holding{}}
	f := NewFeed(cfg)
	i := density.Instrument{ID: "uid", Ticker: "T", Lot: 10, PriceStep: density.Price{Units: 1}}
	e, err := NewEngine(cfg, b, s, f, []density.Instrument{i})
	if err != nil {
		t.Fatal(err)
	}
	x := &fixture{e: e, b: b, f: f, s: s, i: i, now: now, account: account}
	e.Now = func() time.Time { return x.now }
	x.book(1000, true)
	return x
}
func (x *fixture) book(qty int64, signal bool) {
	x.now = x.now.Add(time.Millisecond)
	b := density.Book{InstrumentID: "uid", Lot: 10, Time: x.now, Bids: []density.Level{{Price: density.Price{Units: 100}, Quantity: qty}}, Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}}}
	var ss []density.Signal
	if signal {
		ss = []density.Signal{{InstrumentID: "uid", Price: density.Price{Units: 100}, Quantity: qty}}
	}
	x.f.SetReady("uid", true)
	x.f.Book(b, ss, x.now)
}
func (x *fixture) step(t *testing.T) {
	t.Helper()
	if err := x.e.Step(context.Background(), x.now); err != nil {
		t.Fatal(err)
	}
	x.now = x.now.Add(2 * time.Second)
	x.book(1000, false)
}
func (x *fixture) fill() {
	o := x.b.posts[0]
	r := x.b.orders["order-"+o.Key]
	r.Status = "filled"
	r.Filled = 1
	r.Terminal = true
	r.Amount = 1010 * Nano
	r.Fees = Nano
	x.b.orders[r.ID] = r
	x.b.holdings["uid"] = Holding{Units: 10}
}

func TestPendingBuyCanceledOnDensityLoss(t *testing.T) {
	x := setup(t)
	x.step(t)
	if len(x.b.posts) != 1 || x.b.posts[0].Lots != 1 {
		t.Fatal("not one-lot entry")
	}
	x.book(499, false)
	x.step(t)
	if len(x.b.canceled) != 1 || x.e.State().Episodes["uid"] != nil || x.e.State().Canceled != 1 {
		t.Fatalf("pending not canceled: %+v", x.e.State())
	}
}
func TestFillWhileCancelingTriggersSaleNotLostInventory(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.b.onCancel = func(string) { x.fill() }
	x.book(499, false)
	for j := 0; j < 4 && x.e.State().Episodes["uid"] != nil; j++ {
		x.step(t)
	}
	if len(x.b.posts) != 2 || x.b.posts[1].Kind != "exit" || x.b.posts[1].Lots != 1 || x.e.State().Closed != 1 {
		t.Fatalf("fill/cancel race: posts=%+v state=%+v", x.b.posts, x.e.State())
	}
}
func TestConfirmedFillCreatesRealStopAndTakeThenClosesOnLoss(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	if len(x.b.protections) != 2 || x.b.protections[0].Kind != "stop" || x.b.protections[1].Kind != "take" {
		t.Fatalf("protection missing: %+v", x.b.protections)
	}
	for _, p := range x.b.protections {
		if p.Lots != 1 {
			t.Fatal("wrong protection quantity")
		}
	}
	x.book(499, false)
	for j := 0; j < 4 && x.e.State().Episodes["uid"] != nil; j++ {
		x.step(t)
	}
	if x.e.State().Closed != 1 || x.e.State().Realized != -22*Nano || len(x.b.posts) != 2 {
		t.Fatalf("wrong exit/result: %+v", x.e.State())
	}
}
func TestStopActivationWithoutChildIDBlocksCompetingSale(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	x.b.unknownChild = true
	x.book(499, false)
	for j := 0; j < 3; j++ {
		_ = x.e.Step(context.Background(), x.now)
		x.now = x.now.Add(2 * time.Second)
	}
	if len(x.b.posts) != 1 || x.e.State().Episodes["uid"] == nil {
		t.Fatal("sold despite unknown activated child")
	}
}
func TestUnknownStopPlacementNeverRepeatsWithNewUUID(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.b.stopErr = errors.New("response lost after acceptance")
	if err := x.e.Step(context.Background(), x.now); err == nil {
		t.Fatal("unknown protection ignored")
	}
	ep := x.e.State().Episodes["uid"]
	if ep == nil || ep.Stop.Key == "" {
		t.Fatal("intent not preserved")
	}
	key := ep.Stop.Key
	for j := 0; j < 3; j++ {
		_ = x.e.Step(context.Background(), x.now.Add(time.Minute))
	}
	if len(x.b.protections) != 1 || len(x.b.posts) != 1 || x.e.State().Episodes["uid"].Stop.Key != key {
		t.Fatal("unknown order duplicated")
	}
}
func TestSlowBuyRPCDoesNotEraseDensityLoss(t *testing.T) {
	x := setup(t)
	x.b.onPost = func() { x.book(499, false); x.book(1000, false) }
	x.step(t)
	x.b.onPost = nil
	for j := 0; j < 3 && x.e.State().Episodes["uid"] != nil; j++ {
		x.step(t)
	}
	if len(x.b.canceled) != 1 || len(x.b.posts) != 1 {
		t.Fatal("missed loss during post RPC")
	}
}
func TestRestartDoesNotRepostUncertainBuy(t *testing.T) {
	x := setup(t)
	st := x.s.State()
	ep, err := Plan(DefaultConfig(), x.i, density.Signal{InstrumentID: "uid", Price: density.Price{Units: 100}}, x.f.Candidates(x.now)[0].Book, x.now)
	if err != nil {
		t.Fatal(err)
	}
	ep.Entry.Key = uuid.NewString()
	ep.Entry.Sent = true
	st.Episodes[ep.Instrument.ID] = ep
	x.s.Save(st, "entry_intent")
	e, err := NewEngine(DefaultConfig(), x.b, x.s, x.f, []density.Instrument{x.i})
	if err != nil {
		t.Fatal(err)
	}
	e.Now = func() time.Time { return x.now }
	for j := 0; j < 3; j++ {
		_ = e.Step(context.Background(), x.now.Add(time.Duration(j)*time.Minute))
	}
	if len(x.b.posts) != 0 || e.State().Episodes["uid"] == nil {
		t.Fatal("replayed uncertain buy")
	}
}

func TestRSIProfitExitReconcilesProtectionBeforeSelling(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	x.e.rsiExit = func(id string, entry, bid int64, _ time.Time) bool {
		return id == x.i.ID && entry == 101*Nano && bid == 102*Nano
	}
	x.now = x.now.Add(time.Millisecond)
	x.f.Book(density.Book{
		InstrumentID: x.i.ID, Lot: x.i.Lot, Time: x.now,
		Bids: []density.Level{
			{Price: density.Price{Units: 102}, Quantity: 10},
			{Price: density.Price{Units: 100}, Quantity: 1000},
		},
		Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}},
	}, nil, x.now)
	x.b.onPost = func() {
		for _, s := range x.b.stops {
			if s.Status == "active" {
				t.Error("exit posted before protection was canceled")
			}
		}
	}
	x.step(t)
	if len(x.b.posts) != 2 || x.b.posts[1].Kind != "exit" {
		t.Fatalf("expected market exit, got %+v", x.b.posts)
	}
	notes := x.e.PendingNotifications()
	if len(notes) != 2 || notes[1].Reason != "rsi_profit" {
		t.Fatalf("expected confirmed RSI sale notification, got %+v", notes)
	}
}
