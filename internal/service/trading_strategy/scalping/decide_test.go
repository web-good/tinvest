package scalping

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func testSettings() model.Settings {
	return model.Settings{
		RsiReversalLevel:  35,
		AtrTakeProfitMult: 1.5,
		AtrStopLossMult:   1.0,
		MaxOpenPositions:  5,
	}
}

func TestDecide(t *testing.T) {
	s := testSettings()

	tests := []struct {
		name       string
		cand       Candidate
		openCount  int
		wantKind   model.SignalKind
		wantTP     float64
		wantSL     float64
		wantReason string
	}{
		{
			name: "buy on trend + rsi reversal",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: true, RSIPrev: 30, RSINow: 36,
			},
			openCount: 0,
			wantKind:  model.SignalBuy,
			wantTP:    103, // 100 + 1.5*2
			wantSL:    98,  // 100 - 1.0*2
		},
		{
			name: "no buy when below ema",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: false, RSIPrev: 30, RSINow: 36,
			},
			openCount: 0,
			wantKind:  model.SignalNone,
		},
		{
			name: "no buy when rsi did not cross upward",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: true, RSIPrev: 36, RSINow: 40,
			},
			openCount: 0,
			wantKind:  model.SignalNone,
		},
		{
			name: "no buy when position cap reached",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: true, RSIPrev: 30, RSINow: 36,
			},
			openCount: 5,
			wantKind:  model.SignalNone,
		},
		{
			name: "sell on take profit",
			cand: Candidate{
				Price: 104, ATR: 2, HasPosition: true, PurchasePrice: 100,
			},
			openCount:  1,
			wantKind:   model.SignalSell,
			wantTP:     103,
			wantSL:     98,
			wantReason: "TP",
		},
		{
			name: "sell on stop loss",
			cand: Candidate{
				Price: 97, ATR: 2, HasPosition: true, PurchasePrice: 100,
			},
			openCount:  1,
			wantKind:   model.SignalSell,
			wantTP:     103,
			wantSL:     98,
			wantReason: "SL",
		},
		{
			name: "hold position inside the band",
			cand: Candidate{
				Price: 101, ATR: 2, HasPosition: true, PurchasePrice: 100,
			},
			openCount: 1,
			wantKind:  model.SignalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.cand, s, tt.openCount)
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
