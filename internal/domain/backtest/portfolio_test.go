package backtest

import (
	"math"
	"testing"
	"time"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestPortfolioOpenSizesByFractionAndLots(t *testing.T) {
	// cash 100000, fraction 0.5 -> budget 50000; price 100, lot 10,
	// commission 0 -> lotCost 1000; lots = floor(50) = 50; qty = 500.
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 0.5, Commission: 0, Lot: 10})
	p.open(100, time.Unix(0, 0), 0, 0, 0, 0, "")
	if p.qty != 500 {
		t.Fatalf("qty = %d, want 500", p.qty)
	}
	// cash spent = 500*100 = 50000.
	if !approx(p.cash, 50000) {
		t.Fatalf("cash = %f, want 50000", p.cash)
	}
}

func TestPortfolioOpenChargesCommission(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.001, Lot: 1})
	p.open(100, time.Unix(0, 0), 0, 0, 0, 0, "")
	// budget 100000; lotCost = 100*1*1.001 = 100.1; lots = floor(999.0) = 999.
	if p.qty != 999 {
		t.Fatalf("qty = %d, want 999", p.qty)
	}
	// cost 99900; commission 99.9; cash = 100000 - 99999.9 = 0.1.
	if !approx(p.cash, 0.1) {
		t.Fatalf("cash = %f, want 0.1", p.cash)
	}
}

func TestPortfolioRefusesEntryWhenCashTooSmallForALot(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 50, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0), 0, 0, 0, 0, "") // one share costs 100 > 50
	if p.qty != 0 {
		t.Fatalf("qty = %d, want 0 (no entry)", p.qty)
	}
	if !approx(p.cash, 50) {
		t.Fatalf("cash = %f, want 50 (untouched)", p.cash)
	}
}

func TestPortfolioCloseComputesPnLAndBarsHeld(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.bar = 3
	p.open(100, time.Unix(3, 0), 0, 0, 0, 0, "") // qty = 1000, cash = 0
	p.bar = 8
	tr := p.close(110, time.Unix(8, 0), "TP")
	// revenue 110000; entryCost 100000; PnL 10000; PnLPct 0.1.
	if !approx(tr.PnL, 10000) || !approx(tr.PnLPct, 0.1) {
		t.Fatalf("PnL=%f PnLPct=%f, want 10000 / 0.1", tr.PnL, tr.PnLPct)
	}
	if tr.BarsHeld != 5 {
		t.Fatalf("BarsHeld = %d, want 5", tr.BarsHeld)
	}
	if tr.Reason != "TP" || tr.Quantity != 1000 {
		t.Fatalf("Reason=%q Quantity=%d, want TP / 1000", tr.Reason, tr.Quantity)
	}
	if p.qty != 0 {
		t.Fatalf("qty = %d after close, want 0", p.qty)
	}
}

func TestPortfolioEquityMarksToMarket(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0), 0, 0, 0, 0, "") // qty 1000, cash 0
	if !approx(p.equity(120), 120000) {
		t.Fatalf("equity = %f, want 120000", p.equity(120))
	}
}

func TestStrategyPositionNilWhenFlat(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if p.strategyPosition() != nil {
		t.Fatal("expected nil position when flat")
	}
	p.open(100, time.Unix(0, 0), 0, 0, 0, 0, "")
	pos := p.strategyPosition()
	if pos == nil || !approx(pos.PurchasePrice, 100) || pos.Quantity != 1000 {
		t.Fatalf("position = %+v, want {100, 1000}", pos)
	}
}

func TestPortfolioOpenFreezesStopAndSeedsFavorable(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0), 0, 0, 2, 95, "") // atr 2, stop 95
	pos := p.strategyPosition()
	if pos == nil {
		t.Fatal("expected a position after open")
	}
	if !approx(pos.StopLoss, 95) {
		t.Errorf("StopLoss = %v, want 95 (frozen at entry)", pos.StopLoss)
	}
	if !approx(pos.EntryATR, 2) {
		t.Errorf("EntryATR = %v, want 2", pos.EntryATR)
	}
	if !approx(pos.MaxFavorablePrice, 100) {
		t.Errorf("MaxFavorablePrice = %v, want 100 (seeded to entry price)", pos.MaxFavorablePrice)
	}
}

func TestPortfolioMarkRaisesFavorableMonotonically(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0), 0, 0, 1, 95, "") // maxFavorable seeded to 100
	p.mark(105)
	if !approx(p.maxFavorable, 105) {
		t.Fatalf("maxFavorable = %v, want 105 after mark(105)", p.maxFavorable)
	}
	p.mark(103) // lower -> no change
	if !approx(p.maxFavorable, 105) {
		t.Fatalf("maxFavorable = %v, want 105 (monotonic, mark(103) is a no-op)", p.maxFavorable)
	}
	p.mark(105) // equal: still not a new high -> no change
	if !approx(p.maxFavorable, 105) {
		t.Fatalf("maxFavorable = %v, want 105 (equal price is not a new high)", p.maxFavorable)
	}
}

func TestPortfolioMarkIsNoOpWhenFlat(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.mark(200) // flat: must not record anything
	if !approx(p.maxFavorable, 0) {
		t.Fatalf("maxFavorable = %v, want 0 (mark while flat is a no-op)", p.maxFavorable)
	}
}

func TestPortfolioCloseResetsEntryState(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.bar = 1
	p.open(100, time.Unix(0, 0), 0, 0, 1, 95, "")
	p.mark(110)
	p.bar = 2
	p.close(108, time.Unix(1, 0), "TRAIL")
	if !approx(p.entryStop, 0) || !approx(p.maxFavorable, 0) {
		t.Fatalf("entryStop=%v maxFavorable=%v, want 0/0 after close", p.entryStop, p.maxFavorable)
	}
}
