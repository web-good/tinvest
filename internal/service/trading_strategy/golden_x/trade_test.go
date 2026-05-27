package golden_x

import (
	"testing"

	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
)

func TestCapSectors(t *testing.T) {
	tests := []struct {
		name            string
		buyShares       map[string]gxmodel.ShareResult
		sellShares      map[string]gxmodel.ShareResult
		wantBuy         map[string]bool // IDs expected in BuyShares
		wantCapped      map[string]bool // IDs expected in CappedBuyShares
		wantSellCount   int
	}{
		{
			name: "three shares same sector — top 2 kept, bottom 1 capped",
			buyShares: map[string]gxmodel.ShareResult{
				"oil-1": {InstrumentName: "Oil A", Score: 7, Sector: "oil"},
				"oil-2": {InstrumentName: "Oil B", Score: 5, Sector: "oil"},
				"oil-3": {InstrumentName: "Oil C", Score: 3, Sector: "oil"},
			},
			wantBuy:    map[string]bool{"oil-1": true, "oil-2": true},
			wantCapped: map[string]bool{"oil-3": true},
		},
		{
			name: "two shares same sector — exactly at cap, no capping",
			buyShares: map[string]gxmodel.ShareResult{
				"oil-1": {InstrumentName: "Oil A", Score: 7, Sector: "oil"},
				"oil-2": {InstrumentName: "Oil B", Score: 5, Sector: "oil"},
			},
			wantBuy:    map[string]bool{"oil-1": true, "oil-2": true},
			wantCapped: map[string]bool{},
		},
		{
			name: "different sectors — no capping",
			buyShares: map[string]gxmodel.ShareResult{
				"oil-1":  {InstrumentName: "Oil A", Score: 7, Sector: "oil"},
				"bank-1": {InstrumentName: "Bank A", Score: 5, Sector: "banks"},
				"it-1":   {InstrumentName: "IT A", Score: 3, Sector: "it"},
				"ret-1":  {InstrumentName: "Retail A", Score: 4, Sector: "retail"},
			},
			wantBuy:    map[string]bool{"oil-1": true, "bank-1": true, "it-1": true, "ret-1": true},
			wantCapped: map[string]bool{},
		},
		{
			name: "empty sector not capped",
			buyShares: map[string]gxmodel.ShareResult{
				"x-1": {InstrumentName: "X1", Score: 7, Sector: ""},
				"x-2": {InstrumentName: "X2", Score: 5, Sector: ""},
				"x-3": {InstrumentName: "X3", Score: 3, Sector: ""},
			},
			wantBuy:    map[string]bool{"x-1": true, "x-2": true, "x-3": true},
			wantCapped: map[string]bool{},
		},
		{
			name: "mixed sectors — oil capped to 2, banks and IT untouched",
			buyShares: map[string]gxmodel.ShareResult{
				"oil-1":  {InstrumentName: "Oil A", Score: 7, Sector: "oil"},
				"oil-2":  {InstrumentName: "Oil B", Score: 5, Sector: "oil"},
				"oil-3":  {InstrumentName: "Oil C", Score: 3, Sector: "oil"},
				"bank-1": {InstrumentName: "Bank A", Score: 6, Sector: "banks"},
				"bank-2": {InstrumentName: "Bank B", Score: 4, Sector: "banks"},
				"it-1":   {InstrumentName: "IT A", Score: 2, Sector: "it"},
			},
			wantBuy:    map[string]bool{"oil-1": true, "oil-2": true, "bank-1": true, "bank-2": true, "it-1": true},
			wantCapped: map[string]bool{"oil-3": true},
		},
		{
			name: "sell shares untouched",
			buyShares: map[string]gxmodel.ShareResult{
				"oil-1": {InstrumentName: "Oil A", Score: 7, Sector: "oil"},
			},
			sellShares: map[string]gxmodel.ShareResult{
				"sell-1": {InstrumentName: "Sell A", Score: 0, Sector: "oil"},
				"sell-2": {InstrumentName: "Sell B", Score: 0, Sector: "oil"},
				"sell-3": {InstrumentName: "Sell C", Score: 0, Sector: "oil"},
			},
			wantBuy:       map[string]bool{"oil-1": true},
			wantCapped:    map[string]bool{},
			wantSellCount: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := gxmodel.TradeResult{
				BuyShares:       tc.buyShares,
				SellShares:      tc.sellShares,
				CappedBuyShares: make(map[string]gxmodel.ShareResult),
			}
			if input.SellShares == nil {
				input.SellShares = make(map[string]gxmodel.ShareResult)
			}

			got := capSectors(input)

			// Check BuyShares
			if len(got.BuyShares) != len(tc.wantBuy) {
				t.Errorf("BuyShares count = %d, want %d", len(got.BuyShares), len(tc.wantBuy))
			}
			for id := range tc.wantBuy {
				if _, ok := got.BuyShares[id]; !ok {
					t.Errorf("BuyShares missing %q", id)
				}
			}

			// Check CappedBuyShares
			if len(got.CappedBuyShares) != len(tc.wantCapped) {
				t.Errorf("CappedBuyShares count = %d, want %d", len(got.CappedBuyShares), len(tc.wantCapped))
			}
			for id := range tc.wantCapped {
				if _, ok := got.CappedBuyShares[id]; !ok {
					t.Errorf("CappedBuyShares missing %q", id)
				}
			}

			// Check SellShares unchanged
			if tc.wantSellCount > 0 && len(got.SellShares) != tc.wantSellCount {
				t.Errorf("SellShares count = %d, want %d", len(got.SellShares), tc.wantSellCount)
			}
		})
	}
}

func TestCapSectors_TiebreakerByID(t *testing.T) {
	// When scores are equal, the share with the lexicographically smaller ID wins.
	input := gxmodel.TradeResult{
		BuyShares: map[string]gxmodel.ShareResult{
			"c": {InstrumentName: "C", Score: 5, Sector: "oil"},
			"a": {InstrumentName: "A", Score: 5, Sector: "oil"},
			"b": {InstrumentName: "B", Score: 5, Sector: "oil"},
		},
		SellShares:      make(map[string]gxmodel.ShareResult),
		CappedBuyShares: make(map[string]gxmodel.ShareResult),
	}

	got := capSectors(input)

	// "a" and "b" should be kept (lexicographically first), "c" capped.
	if _, ok := got.BuyShares["a"]; !ok {
		t.Error("expected 'a' in BuyShares")
	}
	if _, ok := got.BuyShares["b"]; !ok {
		t.Error("expected 'b' in BuyShares")
	}
	if _, ok := got.CappedBuyShares["c"]; !ok {
		t.Error("expected 'c' in CappedBuyShares")
	}
}
