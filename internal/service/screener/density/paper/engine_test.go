package paper

import (
	"math"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

var start = time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)

func p(v int64) density.Price { return density.Price{Units: v} }
func book(at time.Time, qty int64) density.Book {
	return density.Book{InstrumentID: "x", Lot: 1, Time: at, Bids: []density.Level{{Price: p(101), Quantity: 10}, {Price: p(100), Quantity: qty}, {Price: p(99), Quantity: 100}}, Asks: []density.Level{{Price: p(103), Quantity: 100}}}
}
func engine(t *testing.T) *Engine {
	t.Helper()
	c := DefaultConfig()
	c.InitialCash = 10000
	c.PositionRubles = 1000
	c.Latency = time.Second
	c.Commission = .001
	c.Freshness = 10 * time.Second
	e, err := New(c, []density.Instrument{{ID: "x", Ticker: "X", Lot: 1, PriceStep: p(1)}, {ID: "y", Ticker: "Y", Lot: 1, PriceStep: p(1)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func place(t *testing.T, e *Engine) {
	t.Helper()
	e.OnBook(book(start, 100), start)
	e.Observe(density.Signal{InstrumentID: "x", Price: p(100), Quantity: 100}, start)
	if len(e.orders) != 1 {
		t.Fatal("entry not placed")
	}
}
func trade(e *Engine, at time.Time, price, lots int64) {
	e.OnTrade(density.Event{InstrumentID: "x", Price: p(price), Quantity: lots, Time: at}, at)
}
func TestEntryLatencyTouchAndPartialFill(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Millisecond*500), 100, 20)
	trade(e, start.Add(time.Second), 101, 20)
	if e.orders["x"].FilledLots != 0 {
		t.Fatal("latency or touch filled")
	}
	trade(e, start.Add(2*time.Second), 100, 2)
	o := e.orders["x"]
	if o.FilledLots != 2 || o.OpenLots != 2 || o.State != "open" {
		t.Fatalf("order %+v", o)
	}
	trade(e, start.Add(2100*time.Millisecond), 100, 20)
	if o.FilledLots != 2 {
		t.Fatal("partial entry remainder was not canceled")
	}
	if math.Abs(e.cash-(10000-202-0.202)) > 1e-8 {
		t.Fatal(e.cash)
	}
}

func TestSingleLotModeOnlyAlertsAfterExecution(t *testing.T) {
	e := engine(t)
	e.cfg.MaxEntryLots = 1
	var fills []Record
	e.record = func(r Record) error {
		if r.Kind == "entry_filled" {
			fills = append(fills, r)
		}
		return nil
	}
	place(t, e)
	trade(e, start.Add(500*time.Millisecond), 100, 100)
	trade(e, start.Add(time.Second), 101, 100)
	if len(fills) != 0 {
		t.Fatal("alert before execution")
	}
	trade(e, start.Add(2*time.Second), 100, 100)
	if len(fills) != 1 || fills[0].FillLots != 1 || e.orders["x"].FilledLots != 1 {
		t.Fatalf("fills %+v", fills)
	}
}

func TestSignalSimulationDoesNotRequireVirtualCapital(t *testing.T) {
	e := engine(t)
	e.cfg.IgnoreCapital = true
	e.cfg.MaxEntryLots = 1
	e.cash = 0
	place(t, e)
	trade(e, start.Add(2*time.Second), 100, 1)
	if e.orders["x"].FilledLots != 1 {
		t.Fatal("virtual capital suppressed a price alert")
	}
}
func TestDensityLossCancelsPendingAndExitsPosition(t *testing.T) {
	e := engine(t)
	place(t, e)
	e.OnBook(book(start.Add(time.Second), 49), start.Add(time.Second))
	if e.orders["x"].State != "canceled" {
		t.Fatal("pending retained")
	}
	e = engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	e.OnBook(book(start.Add(2*time.Second), 49), start.Add(2*time.Second))
	if e.orders["x"].State != "open" {
		t.Fatal("reduced density volume triggered exit")
	}
	b := book(start.Add(3*time.Second), 100)
	b.Bids = []density.Level{{Price: p(101), Quantity: 10}, {Price: p(99), Quantity: 100}}
	e.OnBook(b, b.Time)
	if e.orders["x"].State != "exiting" {
		t.Fatal("missing density-loss exit")
	}
	b.Time = start.Add(4 * time.Second)
	e.OnBook(b, b.Time)
	if e.orders["x"].State != "closed" {
		t.Fatal("exit not executed")
	}
	if math.Abs(e.realized-(-.404)) > 1e-8 {
		t.Fatalf("pnl %v", e.realized)
	}
}
func TestTakeProfitRequiresTradeThroughAndChargesBothFees(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	// Entry 101, stop 98, target 107.
	trade(e, start.Add(2*time.Second), 107, 20)
	if e.orders["x"].OpenLots != 2 {
		t.Fatal("take filled on touch")
	}
	trade(e, start.Add(3*time.Second), 108, 1)
	trade(e, start.Add(4*time.Second), 108, 1)
	if e.orders["x"].State != "closed" || math.Abs(e.realized-11.584) > 1e-8 {
		t.Fatalf("%+v pnl=%v", e.orders["x"], e.realized)
	}
}
func TestGapMarksUncertainDoesNotInventFill(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	e.Gap("stream_disconnected", start.Add(2*time.Second))
	s := e.Summary(start.Add(2 * time.Second))
	if s.DataComplete || s.OpenPositions != 1 || s.ClosedTrades != 0 || s.ValuationFresh {
		t.Fatalf("summary %+v", s)
	}
	trade(e, start.Add(3*time.Second), 108, 10)
	if e.orders["x"].OpenLots != 2 {
		t.Fatal("filled through missing book")
	}
	e.OnBook(book(start.Add(4*time.Second), 100), start.Add(4*time.Second))
	if e.orders["x"].State != "open" || e.orders["x"].OpenLots != 2 {
		t.Fatal("data recovery liquidated position")
	}
}

func TestDataProblemsKeepPositionOpenAndStopResumesAfterRecovery(t *testing.T) {
	for _, problem := range []string{"stale_book", "invalid_book", "off_tick_book", "stream_disconnected"} {
		t.Run(problem, func(t *testing.T) {
			e := engine(t)
			place(t, e)
			trade(e, start.Add(time.Second), 100, 2)
			at := start.Add(20 * time.Second)
			switch problem {
			case "stale_book":
				e.Tick(at)
			case "invalid_book":
				b := book(at, 100)
				b.Asks = nil
				e.OnBook(b, at)
			case "off_tick_book":
				b := book(at, 100)
				b.Asks[0].Price.Nano = 1
				e.OnBook(b, at)
			case "stream_disconnected":
				e.Gap(problem, at)
			}
			o := e.orders["x"]
			if o.State != "open" || o.OpenLots != 2 || o.ExitReason != "" {
				t.Fatalf("data problem requested exit: %+v", o)
			}
			e.OnBook(book(at.Add(time.Second), 100), at.Add(time.Second))
			if o.State != "open" || o.OpenLots != 2 {
				t.Fatalf("recovery liquidated position: %+v", o)
			}
			trade(e, at.Add(2*time.Second), 97, 1)
			if o.State != "exiting" || o.ExitReason != "stop" {
				t.Fatalf("stop did not resume: %+v", o)
			}
			e.OnBook(book(at.Add(3*time.Second), 100), at.Add(3*time.Second))
			if o.State != "closed" || o.ExitReason != "stop" {
				t.Fatalf("stop exit not executed: %+v", o)
			}
		})
	}
}
func TestCapitalReservationAndBadData(t *testing.T) {
	e := engine(t)
	e.cfg.InitialCash = 1000
	e.cash = 1000
	place(t, e)
	b := book(start, 100)
	b.InstrumentID = "y"
	e.OnBook(b, start)
	e.Observe(density.Signal{InstrumentID: "y", Price: p(100), Quantity: 100}, start)
	if o := e.orders["y"]; o != nil {
		t.Fatal("cash allocated twice")
	}
	b = book(start.Add(time.Second), 100)
	b.Asks[0].Price = p(100)
	e.OnBook(b, start.Add(time.Second))
	trade(e, start.Add(2*time.Second), 100, 10)
	if e.orders["x"].FilledLots != 0 {
		t.Fatal("invalid crossed book allowed execution")
	}
}
func TestStopSweepsVisibleDepthOnce(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 5)
	b := book(start.Add(2*time.Second), 100)
	b.Bids = []density.Level{{Price: p(97), Quantity: 1}, {Price: p(96), Quantity: 1}}
	e.OnBook(b, start.Add(2*time.Second))
	b.Time = start.Add(3 * time.Second)
	e.OnBook(b, b.Time)
	if e.orders["x"].OpenLots != 3 {
		t.Fatal("did not respect visible depth")
	}
	e.Tick(start.Add(3100 * time.Millisecond))
	e.Tick(start.Add(3200 * time.Millisecond))
	if e.orders["x"].OpenLots != 3 {
		t.Fatal("same liquidity reused")
	}
	b.Time = start.Add(4 * time.Second)
	e.OnBook(b, b.Time)
	if e.orders["x"].OpenLots != 1 {
		t.Fatal("residual not exited")
	}
}
func TestConfigAndJournalFailure(t *testing.T) {
	c := DefaultConfig()
	c.Commission = math.NaN()
	if _, err := New(c, nil, nil); err == nil {
		t.Fatal("NaN accepted")
	}
	e := engine(t)
	e.record = func(Record) error { return fmtError{} }
	place(t, e)
	if e.Err() == nil {
		t.Fatal("journal error ignored")
	}
}

type fmtError struct{}

func (fmtError) Error() string { return "disk full" }

func TestStaleGapIsCountedOnceAndValuationUnavailable(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	e.Tick(start.Add(20 * time.Second))
	first := e.Summary(start.Add(20 * time.Second))
	e.Tick(start.Add(21 * time.Second))
	next := e.Summary(start.Add(21 * time.Second))
	if next.DataGaps != first.DataGaps {
		t.Fatal("same outage repeatedly counted")
	}
	if next.Equity != nil || next.UnrealizedPnL != nil || next.NetPnL != nil {
		t.Fatal("invented valuation during gap")
	}
}
func TestShutdownPreservesUnresolvedInventoryAndCancelsPending(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	e.Stop(start.Add(2 * time.Second))
	s := e.Summary(start.Add(2 * time.Second))
	if s.OpenPositions != 1 || s.ClosedTrades != 0 || s.DataComplete || s.Equity != nil {
		t.Fatalf("%+v", s)
	}
	e = engine(t)
	place(t, e)
	e.Stop(start.Add(time.Second))
	s = e.Summary(start.Add(time.Second))
	if s.PendingOrders != 0 || s.CanceledEntries != 1 || !s.DataComplete {
		t.Fatal(s)
	}
}
func TestZeroLatencyStillNeedsNewSnapshotForExit(t *testing.T) {
	e := engine(t)
	e.cfg.Latency = 0
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	b := book(start.Add(2*time.Second), 1)
	e.OnBook(b, b.Time)
	if e.orders["x"].OpenLots != 2 {
		t.Fatal("exit filled with triggering snapshot")
	}
}

func TestExitStopsImmediatelyOnJournalFailure(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 5)
	b := book(start.Add(2*time.Second), 100)
	b.Bids = []density.Level{{Price: p(97), Quantity: 1}, {Price: p(96), Quantity: 1}, {Price: p(95), Quantity: 1}}
	e.OnBook(b, b.Time)
	e.record = func(r Record) error {
		if r.Kind == "exit_filled" {
			return fmtError{}
		}
		return nil
	}
	b.Time = start.Add(3 * time.Second)
	e.OnBook(b, b.Time)
	if e.Err() == nil || e.orders["x"].OpenLots != 4 {
		t.Fatalf("continued unjournaled execution: %+v", e.orders["x"])
	}
}
func TestPreGapSnapshotCannotExecuteTwice(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 5)
	b := book(start.Add(2*time.Second), 100)
	b.Bids = []density.Level{{Price: p(97), Quantity: 1}}
	e.OnBook(b, b.Time)
	b.Time = start.Add(3 * time.Second)
	e.OnBook(b, b.Time)
	e.Gap("disconnect", start.Add(3100*time.Millisecond))
	e.OnBook(b, start.Add(3200*time.Millisecond))
	if e.orders["x"].OpenLots != 4 {
		t.Fatal("replayed liquidity executed twice")
	}
	b.Time = start.Add(4 * time.Second)
	e.OnBook(b, b.Time)
	if e.orders["x"].OpenLots != 3 || e.orders["x"].ExitReason != "stop" {
		t.Fatal("data gap discarded previously requested stop exit")
	}
}
func TestTradeTriggersStopBeforeNextBook(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 2)
	trade(e, start.Add(2*time.Second), 97, 1)
	if e.orders["x"].State != "exiting" || e.orders["x"].ExitReason != "stop" {
		t.Fatal("stop ignored trade below threshold")
	}
}

func TestZeroMaxHoldKeepsPositionUntilStrategyExit(t *testing.T) {
	for _, source := range []string{"tick", "trade"} {
		t.Run(source, func(t *testing.T) {
			e := engine(t)
			e.cfg.MaxHold = 0
			if err := e.cfg.Validate(); err != nil {
				t.Fatal(err)
			}
			place(t, e)
			trade(e, start.Add(time.Second), 100, 2)
			at := start.Add(time.Hour)
			e.OnBook(book(at, 100), at)
			if source == "tick" {
				e.Tick(at)
			} else {
				trade(e, at, 102, 1)
			}
			o := e.orders["x"]
			if o.State != "open" || o.OpenLots != 2 {
				t.Fatalf("holding time closed position: %+v", o)
			}
			trade(e, at.Add(time.Second), 97, 1)
			if o.State != "exiting" || o.ExitReason != "stop" {
				t.Fatalf("stop no longer works: %+v", o)
			}
		})
	}
}
