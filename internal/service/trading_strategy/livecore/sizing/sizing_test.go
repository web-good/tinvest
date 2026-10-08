package sizing

import "testing"

func TestLots(t *testing.T) {
	tests := []struct {
		name                              string
		buyPct, accountValue, cash, price float64
		lot                               int32
		wantLots                          int64
		wantOK                            bool
	}{
		// 10% of 100000 = 10000 budget; price 100, lot 1 -> 100 shares = 100 lots; cash ample.
		{"basic", 10, 100000, 100000, 100, 1, 100, true},
		// lot size 10: 10000 budget / (100*10=1000) = 10 lots.
		{"lot10", 10, 100000, 100000, 100, 10, 10, true},
		// budget buys < 1 lot -> skip.
		{"sub_lot_budget", 1, 1000, 1000, 100, 10, 0, false},
		// budget allows 10 lots but cash only covers 3.
		{"cash_capped", 10, 100000, 3500, 100, 10, 3, true},
		// cash cannot cover even one lot -> skip.
		{"insufficient_cash", 10, 100000, 500, 100, 10, 0, false},
		// zero/garbage price -> skip, no panic.
		{"zero_price", 10, 100000, 100000, 0, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lots, ok, reason := Lots(tt.buyPct, tt.accountValue, tt.cash, tt.price, tt.lot)
			if lots != tt.wantLots || ok != tt.wantOK {
				t.Fatalf("Lots(%+v) = (%d, %v, %q), want (%d, %v)", tt, lots, ok, reason, tt.wantLots, tt.wantOK)
			}
			if !ok && reason == "" {
				t.Fatalf("skip case must carry a reason")
			}
		})
	}
}

func TestBudgetLots(t *testing.T) {
	cases := []struct {
		name              string
		pct, total, price float64
		lot               int32
		want              int64
	}{
		{"обычный", 5, 1_000_000, 100, 1, 500},
		{"лот 10", 5, 1_000_000, 100, 10, 50},
		{"меньше лота", 0.001, 1_000_000, 100, 1, 0},
		{"нулевая цена", 5, 1_000_000, 0, 1, 0},
		{"нулевой лот", 5, 1_000_000, 100, 0, 0},
	}
	for _, c := range cases {
		if got := BudgetLots(c.pct, c.total, c.price, c.lot); got != c.want {
			t.Errorf("%s: BudgetLots = %d, want %d", c.name, got, c.want)
		}
	}
}
