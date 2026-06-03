package backtest

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// scriptedStrategy decides from MarketData via an injected function, so tests
// can drive Buy/Sell at specific bars by inspecting md.Price / md.Position.
type scriptedStrategy struct {
	lookback int
	decide   func(md strategy.MarketData) model.Signal
}

func (s scriptedStrategy) Ticker() string                             { return "TEST" }
func (s scriptedStrategy) Lookback() int                              { return s.lookback }
func (s scriptedStrategy) Decide(md strategy.MarketData) model.Signal { return s.decide(md) }

// flatCandles builds n bars at 1h steps; close[i] = closes[i], H/L = close, vol 1.
func flatCandles(closes []float64) []Candle {
	out := make([]Candle, len(closes))
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, c := range closes {
		out[i] = Candle{
			Time: base.Add(time.Duration(i) * time.Hour),
			Open: c, High: c, Low: c, Close: c, Volume: 1,
		}
	}
	return out
}

func TestEngineBuysFlatSellsInPosition(t *testing.T) {
	candles := flatCandles([]float64{10, 10, 100, 10, 110})
	// Lookback 1; buy when price==100 & flat, sell when price==110 & in position.
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		if md.Position != nil && md.Price == 110 {
			return model.Signal{Kind: model.SignalSell, Reason: "TP"}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.EntryPrice != 100 || tr.ExitPrice != 110 || tr.Reason != "TP" {
		t.Fatalf("trade = %+v, want entry 100 exit 110 TP", tr)
	}
	if len(res.Equity) != 5 {
		t.Fatalf("equity points = %d, want 5", len(res.Equity))
	}
}

func TestEngineIgnoresBuyInPositionAndSellWhenFlat(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 100, 100})
	// Always tries to buy; sell never triggers -> exactly one entry, no exit.
	buys := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		return model.Signal{Kind: model.SignalBuy}
	}}
	res := Run(buys, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 0 {
		t.Fatalf("trades = %d, want 0 (never sold)", len(res.Trades))
	}
	if res.BarsInMarket == 0 {
		t.Fatal("expected some bars in market")
	}

	// Sell-when-flat must be ignored entirely.
	sells := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		return model.Signal{Kind: model.SignalSell, Reason: "SL"}
	}}
	res2 := Run(sells, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res2.Trades) != 0 || res2.BarsInMarket != 0 {
		t.Fatalf("sell-when-flat changed state: trades=%d inMarket=%d", len(res2.Trades), res2.BarsInMarket)
	}
}

func TestEngineEmptyWhenNotEnoughHistory(t *testing.T) {
	candles := flatCandles([]float64{10, 20})
	s := scriptedStrategy{lookback: 5, decide: func(md strategy.MarketData) model.Signal {
		return model.Signal{Kind: model.SignalBuy}
	}}
	res := Run(s, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 0 || len(res.Equity) != 0 {
		t.Fatalf("expected empty result, got %d trades %d equity", len(res.Trades), len(res.Equity))
	}
	if res.FinalEquity != 100000 {
		t.Fatalf("FinalEquity = %f, want 100000", res.FinalEquity)
	}
}

func TestEngineMarksOpenPositionToMarketAtEnd(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 100, 200})
	// Buy at the first price==100 bar, never sell.
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	// qty = floor(100000/100) = 1000; cash 0; final close 200 -> equity 200000.
	if res.FinalEquity != 200000 {
		t.Fatalf("FinalEquity = %f, want 200000 (mark-to-market open position)", res.FinalEquity)
	}
}

func TestEngineWindowIsLookbackSized(t *testing.T) {
	candles := flatCandles([]float64{1, 2, 3, 4, 5})
	var seen int
	s := scriptedStrategy{lookback: 3, decide: func(md strategy.MarketData) model.Signal {
		seen = len(md.Closes) // must always equal lookback
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, Config{InitialCash: 1000, Fraction: 1.0, Lot: 1})
	if seen != 3 {
		t.Fatalf("window size = %d, want 3", seen)
	}
}
