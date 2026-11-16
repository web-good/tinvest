package core

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func TestTargetLevel(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		mult, entry, atr, want float64
	}{
		{"one and a half ATR", 1.5, 100, 10, 115},
		{"disabled", 0, 100, 10, 0},
		{"no ATR", 1.5, 100, 0, 0},
		{"no entry", 1.5, 0, 10, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.TPDailyATR = tc.mult
			if got := TargetLevel(p, tc.entry, tc.atr); got != tc.want {
				t.Fatalf("target = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEntryCarriesATRTargetDistance(t *testing.T) {
	p := DefaultParams()
	p.TPDailyATR = 1.5
	md := fixture(trendCloses(3), mondayNoon)
	sig := NewWithParams("TEST", p).Decide(md)
	if sig.Kind != model.SignalBuy || math.Abs(sig.TargetDistance-1.5*dailyWidth) > 1e-9 || math.Abs(sig.TakeProfit-sig.Price-sig.TargetDistance) > 1e-9 {
		t.Fatalf("entry target must follow the fill by 1.5 ATR: %+v", sig)
	}
}

func TestTakeProfitUsesWickAndKeepsStopPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		stopped      bool
	}{
		{"wick", "TP", false},
		{"both touched", "SL", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.TPDailyATR = 1.5
			md := fixture(recoveryCloses(4), mondayNoon)
			md.Position = openPosition(dailyWidth)
			md.Position.TakeProfit = TargetLevel(p, md.Position.PurchasePrice, md.Position.EntryATR)
			i := len(md.Closes) - 1
			md.Highs[i] = md.Position.TakeProfit + .01
			if md.Closes[i] >= md.Position.TakeProfit {
				t.Fatal("fixture close must stay below target")
			}
			if tc.stopped {
				md.Lows[i] = md.Position.StopLoss - .01
			}
			sig := NewWithParams("TEST", p).Decide(md)
			if sig.Kind != model.SignalSell || sig.Reason != tc.reason {
				t.Fatalf("got %+v, want SELL/%s", sig, tc.reason)
			}
			if tc.reason == "TP" && sig.TakeProfit != md.Position.TakeProfit {
				t.Fatal("exit lost frozen target")
			}
		})
	}
}
