package core

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// closeStopParams arms the close-based stop with a catastrophe stop at twice the stop distance.
func closeStopParams() Params {
	p := DefaultParams()
	p.StopOnClose = 1
	p.CatStopMult = 2
	return p
}

// closeStopFixture is a position held through a bar that does NOT cross RSI back up, so only
// the stops can fire; it returns the protective level and the catastrophe level as well.
func closeStopFixture() (md strategy.MarketData, level, cat float64) {
	md = fixture(recoveryCloses(4), mondayNoon)
	md.Position = openPosition(dailyWidth)
	p := closeStopParams()
	level = md.Position.PurchasePrice - p.StopDailyATR*dailyWidth
	cat = md.Position.PurchasePrice - p.CatStopMult*p.StopDailyATR*dailyWidth
	return md, level, cat
}

func TestStopOnCloseIgnoresAWickThroughTheLevel(t *testing.T) {
	md, level, cat := closeStopFixture()
	i := len(md.Closes) - 1
	md.Lows[i] = (level + cat) / 2 // тень между рабочим и аварийным уровнем
	md.Closes[i] = level + 0.01
	if sig := NewWithParams("TEST", closeStopParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want no exit: only the close may trigger the stop", sig.Kind, sig.Reason)
	}
}

func TestStopOnCloseExitsAtTheClose(t *testing.T) {
	for _, tc := range []struct {
		name  string
		shift float64
	}{{"close below the level", -0.01}, {"close exactly on the level", 0}} {
		t.Run(tc.name, func(t *testing.T) {
			md, level, cat := closeStopFixture()
			i := len(md.Closes) - 1
			md.Lows[i] = (level + cat) / 2
			md.Closes[i] = level + tc.shift
			sig := NewWithParams("TEST", closeStopParams()).Decide(md)
			if sig.Kind != model.SignalSell || sig.Reason != "SLC" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/SLC", sig.Kind, sig.Reason)
			}
			if model.IsStopReason(sig.Reason) {
				t.Fatal("SLC must fill at the bar close, not at the level")
			}
		})
	}
}

func TestStopOnCloseCatastropheStopFiresIntrabar(t *testing.T) {
	md, level, cat := closeStopFixture()
	i := len(md.Closes) - 1
	md.Lows[i] = cat // касание считается
	md.Closes[i] = level + 0.5
	sig := NewWithParams("TEST", closeStopParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "CATSL" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/CATSL", sig.Kind, sig.Reason)
	}
	if math.Abs(sig.StopLoss-cat) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v", sig.StopLoss, cat)
	}
}

func TestStopOnCloseWithoutCatastropheStopHoldsThroughAnyWick(t *testing.T) {
	md, level, _ := closeStopFixture()
	i := len(md.Closes) - 1
	md.Lows[i] = level / 2
	md.Closes[i] = level + 0.01
	p := closeStopParams()
	p.CatStopMult = 0
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want no exit with CatStopMult=0", sig.Kind, sig.Reason)
	}
}

func TestCatastropheStopLevel(t *testing.T) {
	off := closeStopParams()
	off.StopOnClose = 0
	noMult := closeStopParams()
	noMult.CatStopMult = 0
	tests := []struct {
		name string
		p    Params
		atr  float64
		want float64
	}{
		{"armed", closeStopParams(), 10, 100 - 2*closeStopParams().StopDailyATR*10},
		{"intrabar stop mode has no catastrophe stop", off, 10, 0},
		{"zero multiplier disables it", noMult, 10, 0},
		{"no ATR", closeStopParams(), 0, 0},
		{"level at or below zero is no stop", closeStopParams(), 1000, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CatastropheStop(tc.p, 100, tc.atr); math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("CatastropheStop = %v, want %v", got, tc.want)
			}
		})
	}
}
