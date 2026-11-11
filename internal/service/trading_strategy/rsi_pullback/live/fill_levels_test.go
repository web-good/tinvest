package live

import (
	"testing"
	"time"
	"tinvest/internal/service/trading_strategy/scalping/model"
)

func TestFillLevels(t *testing.T) {
	sig := model.Signal{StopLoss: 90, TakeProfit: 120, StopDistance: 5, TargetDistance: 10}
	stop, target := fillLevels(sig, 103)
	if stop != 98 || target != 113 {
		t.Fatalf("got %v %v", stop, target)
	}
	stop, target = fillLevels(model.Signal{StopLoss: 90, TakeProfit: 120}, 103)
	if stop != 90 || target != 120 {
		t.Fatalf("existing signal levels changed: %v %v", stop, target)
	}
}

func TestEntrySignalExecutable(t *testing.T) {
	now := time.Date(2026, 11, 11, 10, 1, 0, 0, time.UTC)
	if entrySignalExecutable(model.Signal{FillBy: now.Add(-31 * time.Minute)}, now) {
		t.Fatal("stale signal accepted")
	}
	if !entrySignalExecutable(model.Signal{MinEntryPrice: 100}, now) {
		t.Fatal("price floor must not block a market order")
	}
	if !entrySignalExecutable(model.Signal{FillBy: now.Add(-time.Minute)}, now) {
		t.Fatal("fresh unrestricted signal rejected")
	}
}
