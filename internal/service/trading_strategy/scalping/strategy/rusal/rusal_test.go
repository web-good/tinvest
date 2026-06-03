package rusal

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestTickerAndLookback(t *testing.T) {
	s := New()
	if s.Ticker() != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", s.Ticker())
	}
	// 6*14 + 20 + 50 = 154
	if got := s.Lookback(); got != 154 {
		t.Errorf("Lookback = %d, want 154", got)
	}
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	if p.ADXTrendLevel <= p.ADXRangeLevel {
		t.Errorf("ADXTrendLevel (%v) must exceed ADXRangeLevel (%v)", p.ADXTrendLevel, p.ADXRangeLevel)
	}
	if p.EMAPeriod <= 0 || p.ADXPeriod <= 0 || p.RSIPeriod <= 0 || p.DonchianPeriod <= 0 || p.ATRPeriod <= 0 {
		t.Errorf("all periods must be positive: %+v", p)
	}
}

func TestEMATouched(t *testing.T) {
	ema := []float64{10, 10, 10, 10, 10}
	// A low at index 3 dips to the EMA within tolerance.
	lows := []float64{12, 12, 12, 10.01, 12}
	if !emaTouched(lows, ema, 3, 0.002) { // window covers indices 2,3,4
		t.Error("expected touch within last 3 bars")
	}
	if emaTouched(lows, ema, 1, 0.002) { // window covers only index 4 (low 12, no touch)
		t.Error("did not expect touch within last 1 bar")
	}
	if emaTouched(nil, nil, 3, 0.002) {
		t.Error("empty input must not touch")
	}
}

func TestRecentHigh(t *testing.T) {
	highs := []float64{5, 9, 3, 7, 4}
	if got := recentHigh(highs, 3); got != 7 { // last 3 -> {3,7,4} -> 7
		t.Errorf("recentHigh = %v, want 7", got)
	}
	if got := recentHigh(highs, 10); got != 9 { // window > len -> all -> 9
		t.Errorf("recentHigh = %v, want 9", got)
	}
	if got := recentHigh(nil, 3); got != 0 {
		t.Errorf("recentHigh(nil) = %v, want 0", got)
	}
	if got := recentHigh([]float64{5, 9, 3, 7, 4}, 0); got != 4 { // window<=0 clamps to last bar, no panic
		t.Errorf("recentHigh(window=0) = %v, want 4", got)
	}
}

func TestRegimeOf(t *testing.T) {
	s := New() // ADXTrendLevel 25, ADXRangeLevel 20
	cases := []struct {
		adx  float64
		want regime
	}{
		{0, regimeDead},   // sentinel / insufficient history
		{-5, regimeDead},  // defensive
		{30, regimeTrend}, // >= 25
		{25, regimeTrend}, // boundary
		{15, regimeRange}, // <= 20
		{20, regimeRange}, // boundary
		{22, regimeDead},  // dead zone between 20 and 25
	}
	for _, c := range cases {
		if got := s.regimeOf(c.adx); got != c.want {
			t.Errorf("regimeOf(%v) = %v, want %v", c.adx, got, c.want)
		}
	}
}

// TestDecide_FlatUptrendIsNone: a monotonic uptrend keeps RSI high (no upward cross)
// and price runs above the EMA, so no entry fires. Holds for the stub and the real core.
func TestDecide_FlatUptrendIsNone(t *testing.T) {
	s := New()
	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i)
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}
	md := strategy.MarketData{
		Price:  closes[199],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
	}
	got := s.Decide(md)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", got.Kind)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}

// testParams uses clean levels/multipliers so expected stops/targets are exact.
// Indicator periods are irrelevant here: decide() consumes pre-computed scalars.
func testParams() Params {
	return Params{
		EMAPeriod: 3, ADXPeriod: 2, ADXTrendLevel: 25, ADXRangeLevel: 20,
		RSIPeriod: 2, RSITrendLevel: 45, RSIRangeLevel: 35,
		PullbackWindow: 5, DonchianPeriod: 3, ATRPeriod: 2,
		SLMult: 1.0, TrailMult: 2.0, ChandelierWindow: 3,
		EMATouchTol: 0.002, BandTol: 0.003,
	}
}

func TestDecideCore(t *testing.T) {
	s := NewWithParams(testParams())

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
			wantKind: model.SignalBuy, wantTP: 104, wantSL: 98, // 100+2*2 ; 100-1*2
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
			wantKind: model.SignalNone, // price 100 <= emaNow 101 -> trend not intact -> no entry
		},
		{
			name: "range entry: at lower band + rsi cross",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8, // 100 <= 99.8*1.003 = 100.0994
			},
			wantKind: model.SignalBuy, wantTP: 104.9, wantSL: 98, // mid=(110+99.8)/2 ; 100-2
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
				price: 106, atr: 2, adx: 15, donUpper: 110, donLower: 100, // mid=105
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "TP", wantTP: 105, wantSL: 98,
		},
		{
			name: "range exit: stop loss",
			in: decideInput{
				price: 97, atr: 2, adx: 15, donUpper: 110, donLower: 100, // mid=105
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "SL", wantTP: 105, wantSL: 98,
		},
		{
			name: "range exit: degenerate donchian does not fire spurious TP",
			in: decideInput{
				price: 99, atr: 2, adx: 15, donUpper: 0, donLower: 0, // mid=0
				pos: &strategy.Position{PurchasePrice: 100}, // hardSL=98, price 99 > 98 (no SL)
			},
			wantKind: model.SignalNone, // mid=0 guard prevents a bogus take-profit
		},
		{
			name: "trend exit: chandelier trail",
			in: decideInput{
				price: 105, atr: 2, adx: 30, chandelierHigh: 110, // chandelier=110-2*2=106
				pos: &strategy.Position{PurchasePrice: 100}, // hardSL=98, price 105 > 98
			},
			wantKind: model.SignalSell, wantReason: "TRAIL", wantSL: 106,
		},
		{
			name: "trend exit: initial stop wins over trail",
			in: decideInput{
				price: 97.5, atr: 2, adx: 30, chandelierHigh: 101, // chandelier=97, hardSL=98
				pos: &strategy.Position{PurchasePrice: 100}, // 97.5 <= 98 -> SL first
			},
			wantKind: model.SignalSell, wantReason: "SL", wantSL: 98,
		},
		{
			name: "trend hold while rising -> none",
			in: decideInput{
				price: 118, atr: 2, adx: 30, chandelierHigh: 120, // chandelier=116, hardSL=98
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

// TestDecide_CrushedPriceIsSL: with an open position and a collapsed price, the hard
// ATR stop fires regardless of regime — exercises the full Decide wiring end-to-end.
func TestDecide_CrushedPriceIsSL(t *testing.T) {
	s := New()
	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i%5) // choppy, bounded
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}
	md := strategy.MarketData{
		Price:    1, // crushed far below any stop
		Highs:    highs,
		Lows:     lows,
		Closes:   closes,
		Position: &strategy.Position{PurchasePrice: 100, Quantity: 1},
	}
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "SL" {
		t.Fatalf("got Kind=%v Reason=%q, want Sell/SL", got.Kind, got.Reason)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}
