package density

import (
	"fmt"
	"testing"
	"time"
)

func TestCandidateSelectionAndGeneration(t *testing.T) {
	c := NewCandidates(2, 15*time.Second)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	signals := []Signal{{InstrumentID: "b", Multiple: 5, Rubles: 10}, {InstrumentID: "a", Multiple: 7, Rubles: 10}, {InstrumentID: "a", Multiple: 6, Rubles: 20}, {InstrumentID: "c", Multiple: 5, Rubles: 20}}
	ids := c.Select(signals, now)
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "c" {
		t.Fatalf("selection %v", ids)
	}
	c.Acknowledge("a", 1, now)
	if c.Ready("a", now.Add(14*time.Second)) || !c.Ready("a", now.Add(15*time.Second)) {
		t.Fatal("readiness delay")
	}
	c.Reconnect(2)
	c.Acknowledge("a", 1, now)
	if c.Ready("a", now.Add(time.Minute)) {
		t.Fatal("old acknowledgement accepted")
	}
}

func TestCandidateTradeCountsAndReselection(t *testing.T) {
	c := NewCandidates(1, 15*time.Second)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := Signal{InstrumentID: "a", Price: Price{Units: 100}, Multiple: 5}
	b := Signal{InstrumentID: "b", Multiple: 6}
	c.Select([]Signal{a}, now)
	c.Acknowledge("a", 1, now)
	c.RecordTrade("a", a.Price, 2, 1)
	c.RecordTrade("a", a.Price, 2, 1)
	if n, ok := c.Trades("a", a.Price, now.Add(20*time.Second)); !ok || n != 4 {
		t.Fatalf("trades: %d %v", n, ok)
	}
	if got := c.Select([]Signal{b}, now.Add(time.Second)); len(got) != 1 || got[0] != "a" {
		t.Fatalf("reselected too early: %v", got)
	}
	c.Reconnect(2)
	if _, ok := c.Trades("a", a.Price, now.Add(time.Minute)); ok {
		t.Fatal("reconnect retained trades")
	}
}

func TestCandidatesKeepExistingOnEqualRanking(t *testing.T) {
	c := NewCandidates(1, 15*time.Second)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.Select([]Signal{{InstrumentID: "z", Multiple: 5, Rubles: 10}}, now)
	got := c.Select([]Signal{{InstrumentID: "a", Multiple: 5, Rubles: 10}, {InstrumentID: "z", Multiple: 5, Rubles: 10}}, now.Add(15*time.Second))
	if len(got) != 1 || got[0] != "z" {
		t.Fatalf("selection churned: %v", got)
	}
}

func TestCandidatesLimitTenDistinctInstruments(t *testing.T) {
	c := NewCandidates(10, 15*time.Second)
	var signals []Signal
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("%02d", i)
		signals = append(signals, Signal{InstrumentID: id, Multiple: float64(20 - i), Rubles: 5_000_000}, Signal{InstrumentID: id, Multiple: 1, Rubles: 1})
	}
	got := c.Select(signals, time.Now())
	if len(got) != 10 || got[0] != "00" || got[9] != "09" {
		t.Fatalf("candidate list: %v", got)
	}
}
