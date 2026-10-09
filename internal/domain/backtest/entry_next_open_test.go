package backtest

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// openCandles builds bars at 30m steps with explicit opens and closes; H/L span both.
func openCandles(opens, closes []float64) []Candle {
	out := make([]Candle, len(closes))
	base := time.Date(2026, 1, 5, 7, 0, 0, 0, time.UTC)
	for i := range closes {
		out[i] = Candle{
			Time: base.Add(time.Duration(i) * 30 * time.Minute),
			Open: opens[i], High: max(opens[i], closes[i]), Low: min(opens[i], closes[i]), Close: closes[i], Volume: 1,
		}
	}
	return out
}

var nextOpenCfg = Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1, EntryAtNextOpen: true}

// TestEntryAtNextOpenFillsAtFollowingBarOpen pins the realistic fill: a Buy decided on bar i's
// close is bought at bar i+1's open, timestamped bar i+1, with the signal's frozen levels.
func TestEntryAtNextOpenFillsAtFollowingBarOpen(t *testing.T) {
	candles := openCandles([]float64{10, 100, 105, 120}, []float64{10, 100, 110, 120})
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, StopLoss: 90, TakeProfit: 130}
		}
		if md.Position != nil && md.Price == 120 {
			return model.Signal{Kind: model.SignalSell, Reason: "EOD"}
		}
		return model.Signal{}
	}}
	res := Run(s, candles, nil, nil, nextOpenCfg)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.EntryPrice != 105 || !tr.EntryTime.Equal(candles[2].Time) {
		t.Fatalf("entry = %.2f at %v, want 105 at %v", tr.EntryPrice, tr.EntryTime, candles[2].Time)
	}
	if tr.ExitPrice != 120 {
		t.Fatalf("exit = %.2f, want 120", tr.ExitPrice)
	}
}

// TestEntryAtNextOpenManagesTheFillBar pins that the fill bar is managed: its whole range comes
// after the open fill, so a stop touched on that bar exits there, with the frozen stop level.
func TestEntryAtNextOpenManagesTheFillBar(t *testing.T) {
	candles := openCandles([]float64{10, 100, 99, 95}, []float64{10, 100, 85, 95})
	var seen *strategy.Position
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, StopLoss: 90, TakeProfit: 130}
		}
		if md.Position != nil && md.Lows[len(md.Lows)-1] <= md.Position.StopLoss {
			seen = md.Position
			return model.Signal{Kind: model.SignalSell, Reason: "SL", StopLoss: md.Position.StopLoss}
		}
		return model.Signal{}
	}}
	res := Run(s, candles, nil, nil, nextOpenCfg)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.EntryPrice != 99 || tr.ExitPrice != 90 || !tr.ExitTime.Equal(candles[2].Time) {
		t.Fatalf("trade = %+v, want entry 99, exit 90 on the fill bar", tr)
	}
	if seen == nil || seen.StopLoss != 90 || seen.TakeProfit != 130 {
		t.Fatalf("position on fill bar = %+v, want frozen stop 90 target 130", seen)
	}
}

// TestEntryAtNextOpenDropsBuyOnLastBar pins that a Buy on the final bar never fills: there is no
// next open to buy at.
func TestEntryAtNextOpenDropsBuyOnLastBar(t *testing.T) {
	candles := openCandles([]float64{10, 100}, []float64{10, 100})
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		return model.Signal{}
	}}
	res := Run(s, candles, nil, nil, nextOpenCfg)
	if len(res.Trades) != 0 || res.BarsInMarket != 0 || res.FinalEquity != 100000 {
		t.Fatalf("got trades=%d barsInMarket=%d equity=%.2f, want no position", len(res.Trades), res.BarsInMarket, res.FinalEquity)
	}
}
