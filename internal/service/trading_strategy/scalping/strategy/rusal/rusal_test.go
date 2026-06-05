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
	if p.TrendFilterPeriod != 100 {
		t.Errorf("TrendFilterPeriod = %d, want 100 (RUAL calibrated)", p.TrendFilterPeriod)
	}
}

// TestDecide_FlatUptrendIsNone: a monotonic uptrend keeps RSI high (no upward cross)
// and price runs above the EMA, so no entry fires.
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
	md := strategy.MarketData{Price: closes[199], Highs: highs, Lows: lows, Closes: closes}
	got := s.Decide(md)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", got.Kind)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
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
