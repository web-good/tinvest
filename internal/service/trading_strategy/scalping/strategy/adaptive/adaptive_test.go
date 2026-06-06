package adaptive

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// testParams uses clean levels/multipliers so expected stops/targets are exact.
// Indicator periods are irrelevant where decide() consumes pre-computed scalars.
func testParams() Params {
	return Params{
		EMAPeriod: 3, ADXPeriod: 2, ADXTrendLevel: 25, ADXRangeLevel: 20,
		RSIPeriod: 2, RSITrendLevel: 45, RSIRangeLevel: 35,
		PullbackWindow: 5, DonchianPeriod: 3, ATRPeriod: 2,
		SLMult: 1.0, TrailMult: 2.0, ChandelierWindow: 3,
		EMATouchTol: 0.002, BandTol: 0.003,
	}
}

func TestNewWithParamsTicker(t *testing.T) {
	if got := NewWithParams("ZZZ", testParams()).Ticker(); got != "ZZZ" {
		t.Errorf("Ticker = %q, want ZZZ", got)
	}
}

func TestEMATouched(t *testing.T) {
	ema := []float64{10, 10, 10, 10, 10}
	lows := []float64{12, 12, 12, 10.01, 12}
	if !emaTouched(lows, ema, 3, 0.002) {
		t.Error("expected touch within last 3 bars")
	}
	if emaTouched(lows, ema, 1, 0.002) {
		t.Error("did not expect touch within last 1 bar")
	}
	if emaTouched(nil, nil, 3, 0.002) {
		t.Error("empty input must not touch")
	}
}

func TestRecentHigh(t *testing.T) {
	highs := []float64{5, 9, 3, 7, 4}
	if got := recentHigh(highs, 3); got != 7 {
		t.Errorf("recentHigh = %v, want 7", got)
	}
	if got := recentHigh(highs, 10); got != 9 {
		t.Errorf("recentHigh = %v, want 9", got)
	}
	if got := recentHigh(nil, 3); got != 0 {
		t.Errorf("recentHigh(nil) = %v, want 0", got)
	}
	if got := recentHigh([]float64{5, 9, 3, 7, 4}, 0); got != 4 {
		t.Errorf("recentHigh(window=0) = %v, want 4", got)
	}
}

func TestRegimeOf(t *testing.T) {
	s := NewWithParams("TST", testParams()) // ADXTrendLevel 25, ADXRangeLevel 20
	cases := []struct {
		adx  float64
		want regime
	}{
		{0, regimeDead},
		{-5, regimeDead},
		{30, regimeTrend},
		{25, regimeTrend},
		{15, regimeRange},
		{20, regimeRange},
		{22, regimeDead},
	}
	for _, c := range cases {
		if got := s.regimeOf(c.adx); got != c.want {
			t.Errorf("regimeOf(%v) = %v, want %v", c.adx, got, c.want)
		}
	}
}

func TestDecideCore(t *testing.T) {
	s := NewWithParams("TST", testParams())

	tests := []struct {
		name       string
		in         decideInput
		wantKind   model.SignalKind
		wantReason string
		wantTP     float64
		wantSL     float64
	}{
		{
			name: "trend entry: pullback + rsi cross + di+",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
			},
			wantKind: model.SignalBuy, wantTP: 104, wantSL: 98,
		},
		{
			name: "trend no pullback -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend di+ < di- -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 10, diMinus: 25, emaTouched: true,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend rsi did not cross -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 46, rsiNow: 50,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend entry blocked when price not above ema",
			in: decideInput{
				price: 100, atr: 2, emaNow: 101, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "range entry: at lower band + rsi cross",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
			},
			wantKind: model.SignalBuy, wantTP: 104.9, wantSL: 98,
		},
		{
			name: "trend entry blocked by HTF filter (filter on, trend down)",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
				trendFilterOn: true, trendUp: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend entry allowed when HTF filter passes",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
				trendFilterOn: true, trendUp: true,
			},
			wantKind: model.SignalBuy, wantTP: 104, wantSL: 98,
		},
		{
			name: "range entry blocked by HTF filter (filter on, trend down)",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
				trendFilterOn: true, trendUp: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "range entry allowed when HTF filter passes",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
				trendFilterOn: true, trendUp: true,
			},
			wantKind: model.SignalBuy, wantTP: 104.9, wantSL: 98,
		},
		{
			name: "range mid-channel (not near lower) -> none",
			in: decideInput{
				price: 105, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "dead zone flat -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 30, rsiNow: 46,
				adx: 22, diPlus: 25, diMinus: 10, emaTouched: true,
				donUpper: 110, donLower: 99.8,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "range exit: take profit at mid",
			in: decideInput{
				price: 106, atr: 2, adx: 15, donUpper: 110, donLower: 100,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "TP", wantTP: 105, wantSL: 98,
		},
		{
			name: "range exit: stop loss",
			in: decideInput{
				price: 97, atr: 2, adx: 15, donUpper: 110, donLower: 100,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "SL", wantTP: 105, wantSL: 98,
		},
		{
			name: "range exit: degenerate donchian does not fire spurious TP",
			in: decideInput{
				price: 99, atr: 2, adx: 15, donUpper: 0, donLower: 0,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend exit: chandelier trail",
			in: decideInput{
				price: 105, atr: 2, adx: 30, chandelierHigh: 110,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "TRAIL", wantSL: 106,
		},
		{
			name: "trend exit: initial stop wins over trail",
			in: decideInput{
				price: 97.5, atr: 2, adx: 30, chandelierHigh: 101,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "SL", wantSL: 98,
		},
		{
			name: "trend hold while rising -> none",
			in: decideInput{
				price: 118, atr: 2, adx: 30, chandelierHigh: 120,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.decide(tt.in)
			if got.Kind != tt.wantKind {
				t.Fatalf("Kind = %v, want %v", got.Kind, tt.wantKind)
			}
			if tt.wantKind == model.SignalNone {
				return
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if tt.wantTP != 0 && got.TakeProfit != tt.wantTP {
				t.Errorf("TakeProfit = %v, want %v", got.TakeProfit, tt.wantTP)
			}
			if tt.wantSL != 0 && got.StopLoss != tt.wantSL {
				t.Errorf("StopLoss = %v, want %v", got.StopLoss, tt.wantSL)
			}
		})
	}
}

func TestDecide_RangeMidBelowEntryNoPhantomTP(t *testing.T) {
	s := NewWithParams("TST", testParams()) // SLMult 1, ATR via input
	// Open long at 100; channel has slid so mid = (102+96)/2 = 99 < entry.
	in := decideInput{
		price: 99, atr: 2, adx: 15, donUpper: 102, donLower: 96,
		pos: &strategy.Position{PurchasePrice: 100},
	}
	got := s.decide(in)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None (mid below entry must not TP)", got.Kind)
	}
	if got.TakeProfit != 0 {
		t.Errorf("TakeProfit = %v, want 0 (no phantom target below entry)", got.TakeProfit)
	}
	// Hard stop must still be reported (entry - SLMult*ATR = 100 - 1*2 = 98).
	if got.StopLoss != 98 {
		t.Errorf("StopLoss = %v, want 98", got.StopLoss)
	}
}

func TestDecide_TrailArmsOnlyInProfit(t *testing.T) {
	p := testParams()
	p.TrailArmATR = 1.0 // arm only after +1 ATR of profit
	s := NewWithParams("TST", p)

	// Unarmed: profit 0.5 < TrailArmATR(1)*ATR(2) = 2 — trail must NOT fire.
	// chandelier = chandelierHigh(105) - TrailMult(2)*ATR(2) = 101;
	// price 100.5 <= 101 would fire IF armed, so the sub-threshold profit is the only guard.
	notArmed := decideInput{
		price: 100.5, atr: 2, adx: 30, chandelierHigh: 105,
		pos: &strategy.Position{PurchasePrice: 100},
	}
	if got := s.decide(notArmed); got.Kind != model.SignalNone {
		t.Fatalf("unarmed trail fired: Kind = %v, want None", got.Kind)
	}

	// Armed: profit 3 >= TrailArmATR(1)*ATR(2) = 2.
	// chandelier = chandelierHigh(110) - TrailMult(2)*ATR(2) = 106; price 103 <= 106, so TRAIL fires.
	armed := decideInput{
		price: 103, atr: 2, adx: 30, chandelierHigh: 110,
		pos: &strategy.Position{PurchasePrice: 100},
	}
	got := s.decide(armed)
	if got.Kind != model.SignalSell || got.Reason != "TRAIL" {
		t.Fatalf("armed trail did not fire: Kind=%v Reason=%q, want Sell/TRAIL", got.Kind, got.Reason)
	}
	if got.StopLoss != 106 {
		t.Errorf("StopLoss = %v, want 106 (chandelier)", got.StopLoss)
	}

	// Exact boundary: profit == TrailArmATR(1)*ATR(2) = 2; predicate is >=, so it must arm.
	// chandelier = chandelierHigh(110) - TrailMult(2)*ATR(2) = 106; price 102 <= 106, so TRAIL fires.
	boundary := decideInput{
		price: 102, atr: 2, adx: 30, chandelierHigh: 110,
		pos: &strategy.Position{PurchasePrice: 100},
	}
	gotB := s.decide(boundary)
	if gotB.Kind != model.SignalSell || gotB.Reason != "TRAIL" {
		t.Fatalf("boundary trail did not fire: Kind=%v Reason=%q, want Sell/TRAIL", gotB.Kind, gotB.Reason)
	}
	if gotB.StopLoss != 106 {
		t.Errorf("boundary StopLoss = %v, want 106 (chandelier)", gotB.StopLoss)
	}
}

func TestDecide_DailyFilterGate(t *testing.T) {
	p := testParams()
	p.TrendFilterPeriod = 3 // tiny period so a short daily series is enough

	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i)
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}

	upDaily := []float64{10, 20, 30, 40}
	downDaily := []float64{40, 30, 20, 10}

	mk := func(daily []float64) strategy.MarketData {
		return strategy.MarketData{
			Price: closes[199], Highs: highs, Lows: lows, Closes: closes,
			DailyCloses: daily,
		}
	}

	s := NewWithParams("TST", p)
	if got := s.Decide(mk(upDaily)); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on rising-only market (up daily)")
	}
	if got := s.Decide(mk(downDaily)); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on rising-only market (down daily)")
	}
	if got := s.Decide(mk([]float64{1, 2})); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on cold-start daily series")
	}

	p0 := testParams()
	p0.TrendFilterPeriod = 0
	if got := NewWithParams("TST", p0).Decide(mk(nil)); got.Kind != model.SignalNone {
		t.Fatalf("filter off changed behavior: got %v, want None", got.Kind)
	}
}
