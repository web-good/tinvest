package rusal

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestDecideCore(t *testing.T) {
	s := New() // rsiReversalLevel 35, tpMult 1.5, slMult 1.0

	tests := []struct {
		name       string
		price, atr float64
		aboveEMA   bool
		rsiPrev    float64
		rsiNow     float64
		pos        *strategy.Position
		wantKind   model.SignalKind
		wantTP     float64
		wantSL     float64
		wantReason string
	}{
		{
			name: "buy on trend + rsi reversal",
			price: 100, atr: 2, aboveEMA: true, rsiPrev: 30, rsiNow: 36,
			wantKind: model.SignalBuy, wantTP: 103, wantSL: 98,
		},
		{
			name: "no buy when below ema",
			price: 100, atr: 2, aboveEMA: false, rsiPrev: 30, rsiNow: 36,
			wantKind: model.SignalNone,
		},
		{
			name: "no buy when rsi did not cross upward",
			price: 100, atr: 2, aboveEMA: true, rsiPrev: 36, rsiNow: 40,
			wantKind: model.SignalNone,
		},
		{
			name: "sell on take profit",
			price: 104, atr: 2, pos: &strategy.Position{PurchasePrice: 100},
			wantKind: model.SignalSell, wantTP: 103, wantSL: 98, wantReason: "TP",
		},
		{
			name: "sell on stop loss",
			price: 97, atr: 2, pos: &strategy.Position{PurchasePrice: 100},
			wantKind: model.SignalSell, wantTP: 103, wantSL: 98, wantReason: "SL",
		},
		{
			name: "hold position inside the band",
			price: 101, atr: 2, pos: &strategy.Position{PurchasePrice: 100},
			wantKind: model.SignalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.decide(tt.price, tt.atr, tt.aboveEMA, tt.rsiPrev, tt.rsiNow, tt.pos)
			if got.Kind != tt.wantKind {
				t.Fatalf("Kind = %v, want %v", got.Kind, tt.wantKind)
			}
			if tt.wantKind == model.SignalNone {
				return
			}
			if got.TakeProfit != tt.wantTP {
				t.Errorf("TakeProfit = %v, want %v", got.TakeProfit, tt.wantTP)
			}
			if got.StopLoss != tt.wantSL {
				t.Errorf("StopLoss = %v, want %v", got.StopLoss, tt.wantSL)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
		})
	}
}

// smallStrategy uses tiny indicator periods so synthetic candle series stay short.
func smallStrategy() *Strategy {
	return &Strategy{
		emaPeriod: 3, rsiPeriod: 2, atrPeriod: 2,
		rsiReversalLevel: 35, tpMult: 1.5, slMult: 1.0,
	}
}

func TestDecide_SellOnTakeProfit(t *testing.T) {
	s := smallStrategy()
	closes := []float64{100, 101, 102, 103, 104, 105}
	md := strategy.MarketData{
		Price:    200, // far above any TP -> deterministic sell
		Highs:    []float64{100, 101, 102, 103, 104, 105},
		Lows:     []float64{99, 100, 101, 102, 103, 104},
		Closes:   closes,
		Position: &strategy.Position{PurchasePrice: 100, Quantity: 1},
	}
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "TP" {
		t.Fatalf("got Kind=%v Reason=%q, want Sell/TP", got.Kind, got.Reason)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}

func TestDecide_FlatNoCrossIsNone(t *testing.T) {
	s := smallStrategy()
	// Monotonic uptrend keeps RSI high -> no upward cross through 35 -> no buy.
	closes := []float64{10, 11, 12, 13, 14, 15}
	md := strategy.MarketData{
		Price:  15,
		Highs:  []float64{10, 11, 12, 13, 14, 15},
		Lows:   []float64{9, 10, 11, 12, 13, 14},
		Closes: closes,
	}
	got := s.Decide(md)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", got.Kind)
	}
}
