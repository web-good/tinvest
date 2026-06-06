package core

import (
	"testing"

	"tinvest/internal/service/trading_strategy/levels"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// testParams returns sane params with every gate active so tests exercise the
// real decision logic rather than the zero-value short-circuits.
func testParams() Params {
	return Params{
		ProfileWindow:     120,
		ProfileBins:       40,
		HVNFactor:         1.5,
		ATRPeriod:         14,
		LevelTolATR:       0.5,
		RoomATR:           2.0,
		MaxExtensionATR:   1.0,
		SLMult:            1.0,
		TrailMult:         2.5,
		ChandelierWindow:  20,
		TrailArmATR:       1.0,
		BreakoutLookback:  10,
		TrendFilterPeriod: 50,
		TrendSlopeTol:     0.0,
		ConfirmCloseFrac:  0.5,
		MinATRFrac:        0.003,
		MinRR:             1.5,
	}
}

func newCore() *Strategy { return NewWithParams("TEST", testParams()) }

// bounceInput builds a flat, valid bounce setup: support at 100, resistance well
// above for room, price just above support, a bullish bar that touched support.
func bounceInput() decideInput {
	return decideInput{
		price: 101,
		atr:   1.0,
		levels: []levels.Level{
			{Price: 100, Strength: 0.4},
			{Price: 110, Strength: 0.4},
		},
		barHigh:       101.5,
		barLow:        99.8, // dipped to support (100) within LevelTolATR*ATR
		barClose:      101,  // closed in top half and above support
		recentHigh:    102,
		downtrend:     false,
		recentlyBelow: false,
		pos:           nil,
	}
}

func TestDecideBounceBuy(t *testing.T) {
	s := newCore()
	sig := s.decide(bounceInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got %v (reason %q)", sig.Kind, sig.Reason)
	}
	if sig.Reason != "BOUNCE" {
		t.Errorf("want reason BOUNCE, got %q", sig.Reason)
	}
	if sig.StopLoss != 100-1.0*1.0 {
		t.Errorf("stop = %v, want %v", sig.StopLoss, 99.0)
	}
	if sig.TakeProfit != 110 {
		t.Errorf("target = %v, want resistance 110", sig.TakeProfit)
	}
}

func TestDecideRetestBuy(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.recentlyBelow = true // broke above a former resistance, now retesting it
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got %v", sig.Kind)
	}
	if sig.Reason != "RETEST" {
		t.Errorf("want reason RETEST, got %q", sig.Reason)
	}
}

func TestDecideBlockedByDowntrend(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.downtrend = true
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("downtrend must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByNoRoom(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// Resistance only 1 ATR away, RoomATR requires 2.
	in.levels = []levels.Level{{Price: 100, Strength: 0.4}, {Price: 102, Strength: 0.4}}
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("insufficient room must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByNoResistance(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.levels = []levels.Level{{Price: 100, Strength: 0.4}} // no resistance above
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("missing resistance must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByOverExtension(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// price 102, support 100 -> dist 2 ATR > MaxExtensionATR (1). Keep room valid.
	in.price = 102
	in.barClose = 102
	in.barHigh = 102.5
	in.levels = []levels.Level{{Price: 100, Strength: 0.4}, {Price: 120, Strength: 0.4}}
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("over-extension must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByNonBullishClose(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// Bearish bar: close near the low -> (close-low)/(high-low) < ConfirmCloseFrac.
	in.barClose = 100.1
	in.barLow = 99.8
	in.barHigh = 101.5
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("non-bullish close must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByCloseBelowLevel(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// Bar closed below the support level even if shaped bullishly.
	in.barLow = 98.0
	in.barHigh = 100.0
	in.barClose = 99.9 // top of range but below support 100
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("close below level must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByNoTouch(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// Bar never reached the level: low stayed far above support+tol.
	in.barLow = 101.0
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("no touch must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByMinRR(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// Resistance close enough for room (2 ATR) but reward/risk under MinRR:
	// price 101, stop 99 (risk 2), target 103 (reward 2) -> RR 1 < 1.5.
	in.levels = []levels.Level{{Price: 100, Strength: 0.4}, {Price: 103, Strength: 0.4}}
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("sub-MinRR setup must block entry, got %v", sig.Kind)
	}
}

func TestDecideBlockedByAntiChurn(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.atr = 0.001 // atr < MinATRFrac*price (0.003*101)
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("sub-ATR setup must block entry, got %v", sig.Kind)
	}
}

func TestDecideFlatNoSetup(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.levels = nil
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("no levels must yield None, got %v", sig.Kind)
	}
}

func TestDecideHardStopExit(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.price = 98.9 // <= entry - SLMult*atr = 99
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want Sell/SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 99 {
		t.Errorf("stop = %v, want 99", sig.StopLoss)
	}
}

func TestDecideTrailExit(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110
	// armed: price >= entry + TrailArmATR*atr = 101. chandelier = 110 - 2.5 = 107.5.
	in.price = 107
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("want Sell/TRAIL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 107.5 {
		t.Errorf("trail stop = %v, want 107.5", sig.StopLoss)
	}
}

func TestDecideTrailNotArmed(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 100.5    // below entry + 1 ATR -> not armed, above hard SL -> hold
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("unarmed trail with price above hard SL must hold, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestDecideInProfitHold(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 108.5    // armed (>=101) but above chandelier -> hold
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("in-profit above chandelier must hold, got %v/%q", sig.Kind, sig.Reason)
	}
}

// --- Integration: Decide over synthetic candles whose volume profile yields a
// known support and resistance. ---

// buildSeries returns n flat-ish bars at base price with the given per-bar volume,
// used to seed the profile with a volume cluster at a price band.
func appendBars(highs, lows, closes *[]float64, vols *[]int64, n int, low, high, close float64, vol int64) {
	for i := 0; i < n; i++ {
		*highs = append(*highs, high)
		*lows = append(*lows, low)
		*closes = append(*closes, close)
		*vols = append(*vols, vol)
	}
}

func TestDecideIntegrationBounceBuy(t *testing.T) {
	p := testParams()
	p.TrendFilterPeriod = 0 // no HTF data in this synthetic series
	s := NewWithParams("TEST", p)

	var highs, lows, closes []float64
	var vols []int64

	// Heavy volume cluster around 100 (support) and around 110 (resistance),
	// thin elsewhere so HVN extraction isolates the two bands.
	appendBars(&highs, &lows, &closes, &vols, 30, 99.5, 100.5, 100, 1000)  // support cluster
	appendBars(&highs, &lows, &closes, &vols, 20, 104.0, 106.0, 105, 50)   // thin middle
	appendBars(&highs, &lows, &closes, &vols, 30, 109.5, 110.5, 110, 1000) // resistance cluster
	appendBars(&highs, &lows, &closes, &vols, 20, 104.0, 106.0, 105, 50)   // thin, drift back down

	// Current bar: dip to support and bounce, closing bullishly above 100.
	highs = append(highs, 101.5)
	lows = append(lows, 99.6)
	closes = append(closes, 101.0)
	vols = append(vols, 200)

	md := strategy.MarketData{
		Price:   101.0,
		Highs:   highs,
		Lows:    lows,
		Closes:  closes,
		Volumes: vols,
	}
	sig := s.Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("integration: want Buy, got %v/%q (stop %v target %v)",
			sig.Kind, sig.Reason, sig.StopLoss, sig.TakeProfit)
	}
	if sig.Ticker != "TEST" {
		t.Errorf("ticker not set: %q", sig.Ticker)
	}
	if sig.TakeProfit <= sig.Price {
		t.Errorf("target %v should be above price %v", sig.TakeProfit, sig.Price)
	}
}

func TestDecideIntegrationNoSetup(t *testing.T) {
	p := testParams()
	p.TrendFilterPeriod = 0
	s := NewWithParams("TEST", p)

	var highs, lows, closes []float64
	var vols []int64
	// Uniform volume, no distinct HVN clusters -> no actionable levels near price.
	for i := 0; i < 60; i++ {
		highs = append(highs, 100.5)
		lows = append(lows, 99.5)
		closes = append(closes, 100.0)
		vols = append(vols, 100)
	}
	md := strategy.MarketData{
		Price:   100.0,
		Highs:   highs,
		Lows:    lows,
		Closes:  closes,
		Volumes: vols,
	}
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("integration: flat uniform profile should yield None, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestRecentBounceOff(t *testing.T) {
	// level 100, tol 0.5 (absolute), confirmFrac 0.5.
	// Bars: [0] bounce off 100 (low 99.6, bullish close 101 above level),
	//       [1] no touch (low 101), [2] current bar (excluded from the scan).
	highs := []float64{101.5, 101.5, 101.5}
	lows := []float64{99.6, 101.0, 99.6}
	closes := []float64{101.0, 101.0, 101.0}

	tests := []struct {
		name     string
		lookback int
		want     bool
	}{
		{"sees the bounce two bars back", 5, true},
		{"lookback 1 only sees bar[1] (no touch)", 1, false},
		{"lookback 0 disabled", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recentBounceOff(highs, lows, closes, 100, 0.5, 0.5, tt.lookback)
			if got != tt.want {
				t.Fatalf("recentBounceOff = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecentBounceOffExcludesCurrentBar(t *testing.T) {
	// Only the current (last) bar is a bounce; it must NOT count.
	highs := []float64{100.5, 101.5}
	lows := []float64{101.0, 99.6} // bar[0] no touch, bar[1]=current bounces
	closes := []float64{100.2, 101.0}
	if recentBounceOff(highs, lows, closes, 100, 0.5, 0.5, 5) {
		t.Fatal("current bar must be excluded from the cooldown scan")
	}
}

func TestRecentBounceOffOtherLevel(t *testing.T) {
	// A bounce off 100 must not trigger cooldown for a candidate level 110.
	highs := []float64{101.5, 110.0}
	lows := []float64{99.6, 109.0}
	closes := []float64{101.0, 109.5}
	if recentBounceOff(highs, lows, closes, 110, 0.5, 0.5, 5) {
		t.Fatal("bounce off a different level must not count")
	}
}

// Interface satisfaction.
var _ strategy.Strategy = (*Strategy)(nil)
