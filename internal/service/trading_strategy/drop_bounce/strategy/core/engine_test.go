// engine_test.go runs drop_bounce through the real backtest engine on 30m bars to pin what it
// relies on: dailies reaching the strategy, entry at the next bar's open after the 30m MACD
// confirms the hourly signal, the hourly RSI exit on the bar that completes the hour, and a signal
// dropped when the next slot is missing.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// toCandles converts bars to candles.
func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c, Volume: 10}
	}
	return out
}

// engineBars is Lookback flat bars at 100 (whole hours) followed by the greenAfterSignal fixture
// scaled from 100; it returns the bars and the index of the confirming (green) bar.
func engineBars() ([]bar, int) {
	lead := NewWithParams("TEST", stepM30, entryParams()).Lookback()
	lead += lead % 2
	sig, a := greenAfterSignal()
	closes := flatN(lead, 100)
	for _, b := range sig {
		closes = append(closes, b.c)
	}
	// Keep the fixture's clock: the lead runs before its first bar.
	bars := path(sig[0].t.Add(-time.Duration(lead)*stepM30).Format("2006-01-02 15:04"), closes)
	return bars, lead + a + 1
}

func runEngine(bars []bar) bt.Result {
	return bt.Run(NewWithParams("TEST", stepM30, entryParams()), toCandles(bars),
		toCandles(dailies(bars[0].t, 20, fixtureATR)), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
}

func TestEngineEntryOnConfirmation(t *testing.T) {
	bars, g := engineBars()
	if h := histOf(bars); isRed(h[g]) || !isRed(h[g-1]) {
		t.Fatalf("fixture: histogram must flip green at %d", g)
	}
	f := bars[g].c
	// The fill bar completes the hour of the +15% jump: the hourly RSI(10) crosses 70 on it, so the
	// position sells at that bar's close.
	bars = append(bars, bar{bars[g].t.Add(stepM30), f, f, f, f})
	res := runEngine(bars)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-f) > eps || !tr.EntryTime.Equal(bars[g+1].t) {
		t.Fatalf("entry %v @ %v, want %v @ %v (next bar's open)", tr.EntryPrice, tr.EntryTime, f, bars[g+1].t)
	}
	if tr.Reason != "RSI" || !tr.ExitTime.Equal(bars[g+1].t) {
		t.Fatalf("exit %q @ %v, want RSI at the fill bar %v", tr.Reason, tr.ExitTime, bars[g+1].t)
	}
}

func TestEngineDropsSignalWhenNextSlotMissing(t *testing.T) {
	bars, g := engineBars()
	f := bars[g].c
	bars = append(bars, bar{bars[g].t.Add(2 * stepM30), f, f, f, f})
	if res := runEngine(bars); len(res.Trades) != 0 {
		t.Fatalf("trades = %d, want 0 (fill slot missing): %+v", len(res.Trades), res.Trades)
	}
}
