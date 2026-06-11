package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// defaultParams returns a valid entry-capable Params for tests.
// RSIPeriod=14, RSICrossLevel=50 — on the fixture the RSI does NOT fire at 50
// (rsiPrev≈70.99 is already above 50), so callers that need confluence must use 75.
func defaultParams() Params {
	return Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 0,
		VolLookback: 20, VolMultiplier: 1.2,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		RSIPeriod: 14, RSICrossLevel: 50, RSIOverbought: 70,
		SignalValidBars: 0,
	}
}

// buildEntryMD constructs a snapshot engineered to trigger a bullish MACD cross on the last bar.
// 260 rising-then-dip-then-pop closes keep close > EMA200; last-bar volume is well above average.
// Empirically: rsiPrev≈70.99, rsiNow≈88.07, macdNow≈3.54 (crossUp=true).
// With RSICrossLevel=75 both MACD and RSI fire (confluence).
// With RSICrossLevel=50 RSI does NOT fire (already above 50 on the prev bar).
func buildEntryMD() strategy.MarketData {
	n := 260
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		base := 100.0 + float64(i)*0.5
		closes[i] = base
		highs[i] = base + 0.3
		lows[i] = base - 0.3
		vols[i] = 1000
	}
	// Engineer a fresh MACD bullish cross at the last bar: dip for a few bars then pop.
	closes[n-4], closes[n-3], closes[n-2] = closes[n-5]-1, closes[n-5]-2, closes[n-5]-2.5
	highs[n-4], highs[n-3], highs[n-2] = closes[n-4]+0.3, closes[n-3]+0.3, closes[n-2]+0.3
	lows[n-4], lows[n-3], lows[n-2] = closes[n-4]-0.3, closes[n-3]-0.3, closes[n-2]-0.3
	closes[n-1] = closes[n-5] + 8 // strong pop -> MACD crosses up, RSI spikes to ~88
	highs[n-1] = closes[n-1] + 0.5
	lows[n-1] = closes[n-1] - 0.5
	vols[n-1] = 5000 // above 1.2x average
	return strategy.MarketData{
		Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: vols,
	}
}

// passingInput returns a decideInput that passes all gates (trend, volume, RR) with
// both signal counters saturated and no fresh events. The pure decide() core should
// return no signal unless we override macdFired/rsiFired and the counters.
func passingInput() decideInput {
	return decideInput{
		price:              100,
		emaTrend:           90,
		atr:                1,
		adx:                30,
		volumeOK:           true,
		recentLow:          98,
		barsSinceMACDCross: signalSaturate,
		barsSinceRSICross:  signalSaturate,
	}
}

// --- a) Confluence pairing via decide() ---

func TestConfluencePairing(t *testing.T) {
	cases := []struct {
		name               string
		macdFired          bool
		rsiFired           bool
		barsSinceMACDCross int
		barsSinceRSICross  int
		window             int
		wantBuy            bool
	}{
		// same bar, window 0 -> Buy
		{"same_bar_w0", true, true, signalSaturate, signalSaturate, 0, true},
		// same bar, window 4 -> Buy
		{"same_bar_w4", true, true, signalSaturate, signalSaturate, 4, true},
		// RSI earlier (age 2), MACD fires now, window 2 -> Buy
		{"rsi_earlier_macd_now_w2", true, false, signalSaturate, 2, 2, true},
		// RSI earlier (age 2), MACD fires now, window 1 -> no (gap 2 > window 1)
		{"rsi_earlier_macd_now_w1", true, false, signalSaturate, 2, 1, false},
		// MACD earlier (age 2), RSI fires now, window 2 -> Buy
		{"macd_earlier_rsi_now_w2", false, true, 2, signalSaturate, 2, true},
		// MACD earlier (age 2), RSI fires now, window 1 -> no
		{"macd_earlier_rsi_now_w1", false, true, 2, signalSaturate, 1, false},
		// gap exceeds window: rsiFired now, macd age 3, window 2 -> no
		{"gap_exceeds_window", false, true, 3, signalSaturate, 2, false},
		// window 0, gap 1: macdFired now, rsiAge 1 -> no
		{"w0_gap1_no", true, false, signalSaturate, 1, 0, false},
		// no fresh event (both fired=false, counters 0,0) -> no (edge-trigger)
		{"no_fresh_event_edge_trigger", false, false, 0, 0, 4, false},
	}

	p := defaultParams()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p.SignalValidBars = tc.window
			s := NewWithParams("TEST", p)
			in := passingInput()
			in.macdFired = tc.macdFired
			in.rsiFired = tc.rsiFired
			in.barsSinceMACDCross = tc.barsSinceMACDCross
			in.barsSinceRSICross = tc.barsSinceRSICross
			sig := s.decide(in)
			if tc.wantBuy && sig.Kind != model.SignalBuy {
				t.Fatalf("want Buy, got kind=%v", sig.Kind)
			}
			if !tc.wantBuy && sig.Kind == model.SignalBuy {
				t.Fatalf("want no Buy, got Buy")
			}
		})
	}
}

// --- b) advanceSignals mechanics ---

func TestAdvanceSignalsMechanics(t *testing.T) {
	p := defaultParams()
	s := NewWithParams("TEST", p)

	// Fresh strategy: both counters start at signalSaturate.
	if s.barsSinceMACDCross != signalSaturate {
		t.Fatalf("initial barsSinceMACDCross=%d want %d", s.barsSinceMACDCross, signalSaturate)
	}
	if s.barsSinceRSICross != signalSaturate {
		t.Fatalf("initial barsSinceRSICross=%d want %d", s.barsSinceRSICross, signalSaturate)
	}

	// A bar with macdFired: barsSinceMACDCross -> 0, barsSinceRSICross stays saturated.
	in1 := decideInput{macdFired: true}
	s.advanceSignals(in1)
	if s.barsSinceMACDCross != 0 {
		t.Fatalf("after macdFired: barsSinceMACDCross=%d want 0", s.barsSinceMACDCross)
	}
	if s.barsSinceRSICross != signalSaturate {
		t.Fatalf("after macdFired: barsSinceRSICross=%d want %d (saturated, no overflow)", s.barsSinceRSICross, signalSaturate)
	}

	// Next bar with no events: barsSinceMACDCross increments to 1.
	in2 := decideInput{}
	s.advanceSignals(in2)
	if s.barsSinceMACDCross != 1 {
		t.Fatalf("after no-event bar: barsSinceMACDCross=%d want 1", s.barsSinceMACDCross)
	}
}

// --- c) Edge-once integration ---

func TestEdgeOnceIntegration(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 0
	s := NewWithParams("TEST", p)

	// Bar A: both signals fire on the same bar -> Buy.
	a := passingInput()
	a.macdFired = true
	a.rsiFired = true
	s.advanceSignals(a)
	a.barsSinceMACDCross = s.barsSinceMACDCross
	a.barsSinceRSICross = s.barsSinceRSICross
	sigA := s.decide(a)
	if sigA.Kind != model.SignalBuy {
		t.Fatalf("bar A: want Buy, got kind=%v", sigA.Kind)
	}

	// Bar B: no fresh events, both counters advanced to age 1 -> no Buy (edge-trigger).
	s.advanceSignals(decideInput{}) // advance: macd age 1, rsi age 1
	b := passingInput()
	b.macdFired = false
	b.rsiFired = false
	b.barsSinceMACDCross = s.barsSinceMACDCross
	b.barsSinceRSICross = s.barsSinceRSICross
	sigB := s.decide(b)
	if sigB.Kind == model.SignalBuy {
		t.Fatal("bar B: no fresh event, should NOT Buy (edge-trigger)")
	}
}

// --- d) Gate blocks via decide() synthetic ---

func TestGateBlocks(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 0

	base := passingInput()
	base.macdFired = true
	base.rsiFired = true
	base.barsSinceMACDCross = 0
	base.barsSinceRSICross = 0

	t.Run("trend_block", func(t *testing.T) {
		s := NewWithParams("TEST", p)
		in := base
		in.emaTrend = 110 // price 100 < ema 110
		if sig := s.decide(in); sig.Kind == model.SignalBuy {
			t.Fatal("want no Buy when price < EMA")
		}
	})

	t.Run("volume_block", func(t *testing.T) {
		s := NewWithParams("TEST", p)
		in := base
		in.volumeOK = false
		if sig := s.decide(in); sig.Kind == model.SignalBuy {
			t.Fatal("want no Buy on weak volume")
		}
	})

	t.Run("risk_zero_block", func(t *testing.T) {
		s := NewWithParams("TEST", p)
		in := base
		in.recentLow = 200 // stop = 200 - 0.5*1 = 199.5 > price 100 -> risk <= 0
		if sig := s.decide(in); sig.Kind == model.SignalBuy {
			t.Fatal("want no Buy when risk <= 0")
		}
	})

	t.Run("minrr_block", func(t *testing.T) {
		pp := p
		pp.MinRR = 10 // TakeProfitRR=2 gives 2R < 10 -> reject
		s := NewWithParams("TEST", pp)
		if sig := s.decide(base); sig.Kind == model.SignalBuy {
			t.Fatal("want no Buy when RR < MinRR")
		}
	})
}

// --- ADX regime gate ---

func TestADXRegimeGate(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 0

	// A fully entry-qualifying input (confluence + trend + volume + RR all pass).
	base := passingInput()
	base.macdFired = true
	base.rsiFired = true
	base.barsSinceMACDCross = 0
	base.barsSinceRSICross = 0

	t.Run("blocks_below_threshold", func(t *testing.T) {
		s := NewWithParams("TEST", p)
		in := base
		in.adx = 20 // < 25 -> chop -> no entry despite full confluence
		if sig := s.decide(in); sig.Kind == model.SignalBuy {
			t.Fatal("want no Buy when ADX below threshold (chop)")
		}
	})

	t.Run("allows_at_threshold", func(t *testing.T) {
		s := NewWithParams("TEST", p)
		in := base
		in.adx = adxThreshold // exactly 25 -> allowed (gate is >=)
		if sig := s.decide(in); sig.Kind != model.SignalBuy {
			t.Fatalf("want Buy when ADX == threshold, got kind=%v", sig.Kind)
		}
	})
}

// --- e) buildInput wiring ---

func TestBuildInputWiring(t *testing.T) {
	md := buildEntryMD()

	t.Run("macdFired_true_belowZeroOff", func(t *testing.T) {
		p := defaultParams() // MACDBelowZeroOnly=0
		s := NewWithParams("TEST", p)
		in := s.buildInput(md)
		if !in.macdFired {
			t.Fatalf("want macdFired=true with MACDBelowZeroOnly=0 and crossUp")
		}
	})

	t.Run("macdFired_false_belowZeroOn_macdPositive", func(t *testing.T) {
		p := defaultParams()
		p.MACDBelowZeroOnly = 1 // macdNow ≈ 3.54 > 0 -> cross doesn't count
		s := NewWithParams("TEST", p)
		in := s.buildInput(md)
		if in.macdFired {
			t.Fatalf("want macdFired=false with MACDBelowZeroOnly=1 and macdNow>0 (got %.4f)", in.macdNow)
		}
	})

	t.Run("rsiFired_true_crossLevel75", func(t *testing.T) {
		p := defaultParams()
		p.RSICrossLevel = 75 // rsiPrev≈70.99 <= 75, rsiNow≈88.07 > 75 -> fires
		s := NewWithParams("TEST", p)
		in := s.buildInput(md)
		if !in.rsiFired {
			t.Fatalf("want rsiFired=true with RSICrossLevel=75 (rsiPrev=%.2f rsiNow=%.2f)", in.rsiPrev, in.rsiNow)
		}
	})

	t.Run("rsiFired_false_crossLevel50", func(t *testing.T) {
		p := defaultParams()
		p.RSICrossLevel = 50 // rsiPrev≈70.99 > 50 -> no cross
		s := NewWithParams("TEST", p)
		in := s.buildInput(md)
		if in.rsiFired {
			t.Fatalf("want rsiFired=false with RSICrossLevel=50 (rsiPrev=%.2f already above 50)", in.rsiPrev)
		}
	})

	t.Run("rsiFired_false_rsiPeriodZero", func(t *testing.T) {
		p := defaultParams()
		p.RSIPeriod = 0
		s := NewWithParams("TEST", p)
		in := s.buildInput(md)
		if in.rsiFired {
			t.Fatal("want rsiFired=false when RSIPeriod=0")
		}
	})

	t.Run("adx_above_threshold_on_trending_fixture", func(t *testing.T) {
		p := defaultParams()
		s := NewWithParams("TEST", p)
		in := s.buildInput(buildEntryMD())
		if in.adx < adxThreshold {
			t.Fatalf("buildEntryMD ADX=%.2f want >= %.0f (sustained uptrend should be strongly trending)", in.adx, adxThreshold)
		}
	})
}

// --- f) End-to-end entry through Decide ---

func TestEndToEndEntryDecide(t *testing.T) {
	t.Run("fires_with_rsiCrossLevel75", func(t *testing.T) {
		p := defaultParams()
		p.RSICrossLevel = 75
		s := NewWithParams("TEST", p)
		sig := s.Decide(buildEntryMD())
		if sig.Kind != model.SignalBuy {
			t.Fatalf("want Buy, got kind=%v", sig.Kind)
		}
		if sig.StopLoss <= 0 || sig.TakeProfit <= sig.Price {
			t.Fatalf("SL=%f TP=%f price=%f want SL>0 and TP>price", sig.StopLoss, sig.TakeProfit, sig.Price)
		}
		if sig.ATR <= 0 {
			t.Fatalf("ATR=%f want >0", sig.ATR)
		}
		if sig.Ticker != "TEST" {
			t.Fatalf("ticker=%q want TEST", sig.Ticker)
		}
		if !strings.Contains(sig.EntryReason, "MACD") || !strings.Contains(sig.EntryReason, "RSI") {
			t.Fatalf("EntryReason missing MACD or RSI: %q", sig.EntryReason)
		}
		if !strings.Contains(sig.EntryReason, "ADX") {
			t.Fatalf("EntryReason missing ADX prefix: %q", sig.EntryReason)
		}
	})

	t.Run("blocked_flat_closes_price_below_ema", func(t *testing.T) {
		md := buildEntryMD()
		for i := range md.Closes {
			md.Closes[i] = 50
			md.Highs[i] = 50.3
			md.Lows[i] = 49.7
		}
		md.Price = 50
		p := defaultParams()
		p.RSICrossLevel = 75
		s := NewWithParams("TEST", p)
		if sig := s.Decide(md); sig.Kind == model.SignalBuy {
			t.Fatal("entry should be blocked when all closes flat at 50 (price below EMA)")
		}
	})

	t.Run("blocked_weak_volume", func(t *testing.T) {
		md := buildEntryMD()
		md.Volumes[len(md.Volumes)-1] = 100 // below average -> volume gate fails
		p := defaultParams()
		p.RSICrossLevel = 75
		s := NewWithParams("TEST", p)
		if sig := s.Decide(md); sig.Kind == model.SignalBuy {
			t.Fatal("entry should be blocked on weak last-bar volume")
		}
	})
}

// --- g) Explain ---

func TestExplain(t *testing.T) {
	t.Run("no_rsi_fire_at_level50_reports_blocked", func(t *testing.T) {
		// RSICrossLevel=50 -> RSI does not fire (already above 50 on prev bar) -> confluence blocked
		p := defaultParams()
		p.RSICrossLevel = 50
		s := NewWithParams("TEST", p)
		out := s.Explain(buildEntryMD())
		if !strings.Contains(out, "Confluence") {
			t.Fatalf("Explain should mention Confluence, got: %q", out)
		}
		if !strings.Contains(out, "ВХОДА НЕТ") {
			t.Fatalf("Explain should report ВХОДА НЕТ when RSI doesn't fire at 50, got: %q", out)
		}
	})

	t.Run("rsi_fires_at_level75_all_pass", func(t *testing.T) {
		// RSICrossLevel=75 -> both MACD and RSI fire -> all gates pass -> ВХОД
		p := defaultParams()
		p.RSICrossLevel = 75
		s := NewWithParams("TEST", p)
		out := s.Explain(buildEntryMD())
		if !strings.Contains(out, "Confluence") {
			t.Fatalf("Explain should mention Confluence, got: %q", out)
		}
		if !strings.Contains(out, "ВХОД") {
			t.Fatalf("Explain should report ВХОД when all gates pass, got: %q", out)
		}
		if !strings.Contains(out, "ADX") {
			t.Fatalf("Explain should report the ADX gate, got: %q", out)
		}
	})

	t.Run("low_adx_reports_blocked", func(t *testing.T) {
		// Flatten all bars -> real ADX = 0 < threshold -> regime gate blocks first.
		md := buildEntryMD()
		for i := range md.Closes {
			md.Closes[i] = 50
			md.Highs[i] = 50.3
			md.Lows[i] = 49.7
		}
		md.Price = 50
		p := defaultParams()
		p.RSICrossLevel = 75
		s := NewWithParams("TEST", p)
		out := s.Explain(md)
		if !strings.Contains(out, "ADX") {
			t.Fatalf("Explain should mention the ADX gate, got: %q", out)
		}
		if !strings.Contains(out, "ВХОДА НЕТ") {
			t.Fatalf("Explain should report ВХОДА НЕТ when ADX below threshold, got: %q", out)
		}
	})
}

// ============================================================
// EXIT tests — carried verbatim from core_test.go.bak
// ============================================================

func inPositionMD(barLow, barHigh, recentHigh float64, pos *strategy.Position) strategy.MarketData {
	// 30 tight flat bars around the recentHigh level so ATR is small and
	// well-defined (~1); override the last bar's high/low. Price is unused by the
	// manage branch (it keys off Position + bar high/low), so it stays 100.
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		closes[i], highs[i], lows[i], vols[i] = recentHigh, recentHigh, recentHigh-1, 1000
	}
	highs[n-1], lows[n-1] = barHigh, barLow
	return strategy.MarketData{
		Price: 100, Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Position: pos,
	}
}

func TestExitStopLoss(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 101, 100, pos)) // barLow 94 <= SL 95
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("kind=%v reason=%q want Sell/SL", sig.Kind, sig.Reason)
	}
}

func TestExitTakeProfit(t *testing.T) {
	// TP = entry + RR*(entry-stop) = 100 + 2*5 = 110. barHigh 111 >= 110.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 111, 100, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind=%v reason=%q want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != 110 {
		t.Fatalf("TP=%f want 110", sig.TakeProfit)
	}
}

func TestExitStopLossWinsOverTP(t *testing.T) {
	// Both SL (95) and TP (110) touched in the same bar: SL has priority.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 111, 100, pos))
	if sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL (priority over TP)", sig.Reason)
	}
}

func TestExitHoldsWhenNeitherHit(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 109, 100, pos)) // below TP 110, above SL 95
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind=%v want None (hold)", sig.Kind)
	}
}

func TestExitTrailWhenEnabled(t *testing.T) {
	p := defaultParams()
	p.UseTrail = 1
	p.TrailArmATR = 0  // arm immediately
	p.TakeProfitRR = 0 // isolate trail: no fixed TP so only TRAIL can fire
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 120}
	s := NewWithParams("TEST", p)
	// recentHigh 120, ATR≈1, TrailMult 2.5 -> chandelier≈117.5; barLow 117 <= chandelier.
	sig := s.Decide(inPositionMD(117, 121, 120, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("kind=%v reason=%q want Sell/TRAIL", sig.Kind, sig.Reason)
	}
}

func TestNoTakeProfitExitWhenDisabled(t *testing.T) {
	p := defaultParams()
	p.TakeProfitRR = 0 // fixed TP disabled
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", p)
	// barHigh 999 would smash any fixed TP; with TakeProfitRR=0 there must be no TP exit.
	sig := s.Decide(inPositionMD(100, 999, 100, pos))
	if sig.Kind == model.SignalSell && sig.Reason == "TP" {
		t.Fatal("no TP exit should fire when TakeProfitRR=0")
	}
}

func TestFixedTPFiresEvenWithTrailEnabled(t *testing.T) {
	// New behavior: TP and trail are independent. Trail on, but TP target hit first.
	p := defaultParams()
	p.UseTrail = 1
	p.TrailArmATR = 0  // armed immediately
	p.TakeProfitRR = 2 // TP = 100 + 2*5 = 110
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", p)
	// recentHigh 100 -> chandelier ~97.5; barLow 111 stays above it (no trail);
	// barHigh 111 >= TP 110 -> fixed TP must still fire even though UseTrail=1.
	sig := s.Decide(inPositionMD(111, 111, 100, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind=%v reason=%q want Sell/TP (TP independent of trail)", sig.Kind, sig.Reason)
	}
}

// inPositionMDWithCloses builds an in-position snapshot from an explicit close
// series so indicator-driven exits (MACD/RSI) can be engineered. Highs/lows
// straddle each close by 0.3; the last bar's high/low are taken from the series
// too. Position keys the manage branch.
func inPositionMDWithCloses(closes []float64, pos *strategy.Position) strategy.MarketData {
	n := len(closes)
	highs := make([]float64, n)
	lows := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		highs[i], lows[i], vols[i] = closes[i]+0.3, closes[i]-0.3, 1000
	}
	return strategy.MarketData{
		Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Position: pos,
	}
}

// risingThenDropCloses builds an uptrend of `up` bars (+step each), with a
// 3-bar dip / 3-bar recovery embedded before the last bar so that MACD is above
// its signal on the penultimate bar, followed by a single sharp drop of `drop`
// that forces a bearish MACD cross on the last bar.
func risingThenDropCloses(up int, start, step, drop float64) []float64 {
	closes := make([]float64, up+1)
	pureRise := up - 6
	if pureRise < 1 {
		pureRise = 1
	}
	for i := 0; i < pureRise; i++ {
		closes[i] = start + float64(i)*step
	}
	peak := closes[pureRise-1]
	for i := 0; i < 3; i++ {
		closes[pureRise+i] = peak - float64(i+1)*1.0
	}
	dipLow := closes[pureRise+2]
	for i := 0; i < 3; i++ {
		closes[pureRise+3+i] = dipLow + float64(i+1)*3.0
	}
	closes[up] = closes[up-1] - drop
	return closes
}

func TestExitMACDBearishCross(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6) // long rise, then -6 last bar
	p := defaultParams()
	p.UseMACDExit = 1
	p.TakeProfitRR = 0 // isolate: no TP
	p.UseTrail = 0
	// SL far below the last bar low so only the MACD exit can fire.
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "MACD" {
		t.Fatalf("kind=%v reason=%q want Sell/MACD", sig.Kind, sig.Reason)
	}
}

func TestNoMACDExitWhenDisabled(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6)
	p := defaultParams()
	p.UseMACDExit = 0 // disabled
	p.TakeProfitRR = 0
	p.UseTrail = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Kind == model.SignalSell && sig.Reason == "MACD" {
		t.Fatal("no MACD exit should fire when UseMACDExit=0")
	}
}

func TestExitStopLossWinsOverMACD(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6)
	p := defaultParams()
	p.UseMACDExit = 1
	p.TakeProfitRR = 0
	p.UseTrail = 0
	// StopLoss above the last bar low (lastClose-0.3) so SL triggers on the same bar.
	last := closes[len(closes)-1]
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: last + 0.1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL (priority over MACD)", sig.Reason)
	}
}

func TestExitRSIOverboughtCrossDown(t *testing.T) {
	const period = 14
	closes := risingThenDropCloses(40, 100, 1.0, 4) // rise then a drop -> RSI falls last bar
	r := indicators.RSISeries(closes, period)
	prev, now := r[len(r)-2], r[len(r)-1]
	if !(prev > now) {
		t.Fatalf("test setup: need prev RSI > now RSI, got prev=%v now=%v", prev, now)
	}
	ob := (prev + now) / 2 // overbought line strictly between -> a top-down cross

	p := defaultParams()
	p.RSIPeriod = period
	p.RSIOverbought = ob
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("kind=%v reason=%q want Sell/RSI (prev=%.2f now=%.2f ob=%.2f)", sig.Kind, sig.Reason, prev, now, ob)
	}
}

func TestNoRSIExitWhenDisabled(t *testing.T) {
	closes := risingThenDropCloses(40, 100, 1.0, 4)
	p := defaultParams()
	p.RSIPeriod = 0 // RSI exit disabled
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Kind == model.SignalSell && sig.Reason == "RSI" {
		t.Fatal("no RSI exit should fire when RSIPeriod=0")
	}
}

func TestNoRSIExitWhenLineNotCrossed(t *testing.T) {
	const period = 14
	closes := risingThenDropCloses(40, 100, 1.0, 4)
	r := indicators.RSISeries(closes, period)
	prev, now := r[len(r)-2], r[len(r)-1]
	p := defaultParams()
	p.RSIPeriod = period
	p.RSIOverbought = now - 5 // line below both values -> no top-down cross
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Kind == model.SignalSell && sig.Reason == "RSI" {
		t.Fatalf("no RSI exit when line %.2f is below both prev=%.2f now=%.2f", now-5, prev, now)
	}
}

func TestExitReasonSL(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 101, 100, pos)) // barLow 94 <= SL 95
	if sig.Reason != "SL" || !strings.Contains(sig.ExitReason, "стоп") {
		t.Fatalf("reason=%q exitReason=%q want SL with 'стоп'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonTP(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 111, 100, pos)) // barHigh 111 >= TP 110
	if sig.Reason != "TP" || !strings.Contains(sig.ExitReason, "цель") {
		t.Fatalf("reason=%q exitReason=%q want TP with 'цель'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonTrail(t *testing.T) {
	p := defaultParams()
	p.UseTrail = 1
	p.TrailArmATR = 0
	p.TakeProfitRR = 0 // isolate trail
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 120}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMD(117, 121, 120, pos)) // barLow 117 <= chandelier ~117.5
	if sig.Reason != "TRAIL" || !strings.Contains(sig.ExitReason, "шанделье") {
		t.Fatalf("reason=%q exitReason=%q want TRAIL with 'шанделье'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonMACD(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6)
	p := defaultParams()
	p.UseMACDExit = 1
	p.TakeProfitRR = 0
	p.UseTrail = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Reason != "MACD" || !strings.Contains(sig.ExitReason, "кросс") {
		t.Fatalf("reason=%q exitReason=%q want MACD with 'кросс'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonRSI(t *testing.T) {
	const period = 14
	closes := risingThenDropCloses(40, 100, 1.0, 4)
	r := indicators.RSISeries(closes, period)
	prev, now := r[len(r)-2], r[len(r)-1]
	if !(prev > now) {
		t.Fatalf("test setup: need prev RSI > now RSI, got prev=%v now=%v", prev, now)
	}
	p := defaultParams()
	p.RSIPeriod = period
	p.RSIOverbought = (prev + now) / 2
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Reason != "RSI" || !strings.Contains(sig.ExitReason, "пересёк границу") {
		t.Fatalf("reason=%q exitReason=%q want RSI with 'пересёк границу'", sig.Reason, sig.ExitReason)
	}
}

// TestEntryFiresWithFixedTPDisabled checks that TakeProfitRR=0 does not block entry.
// Needs RSICrossLevel=75 for confluence to fire.
func TestEntryFiresWithFixedTPDisabled(t *testing.T) {
	p := defaultParams()
	p.TakeProfitRR = 0 // no fixed TP -> MinRR filter must not block the entry
	p.RSICrossLevel = 75
	s := NewWithParams("TEST", p)
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire when TakeProfitRR=0 (MinRR check skipped)")
	}
}
