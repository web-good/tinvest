// engine_test.go runs drop_bounce through the real backtest engine on 15m bars to pin what it
// relies on: dailies reaching the strategy, entry at the next bar's open after the hourly and the
// 15m MACD confirm the hourly signal, the stop measured from the fill, and a signal dropped when the
// next slot is missing.
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

// engineBars is Lookback flat bars at 100 (whole hours) followed by the signalThenGreenHour
// fixture; it returns the bars and the index of the buying bar.
func engineBars() ([]bar, int) {
	lead := NewWithParams("TEST", stepBar, entryParams()).Lookback()
	lead += (perHour - lead%perHour) % perHour
	sig, g := greenAfterSignal()
	closes := flatN(lead, 100)
	for _, b := range sig {
		closes = append(closes, b.c)
	}
	// Keep the fixture's clock: the lead runs before its first bar.
	bars := path(sig[0].t.Add(-time.Duration(lead)*stepBar).Format("2006-01-02 15:04"), closes)
	return bars, lead + g
}

func runEngine(bars []bar) bt.Result {
	return bt.Run(NewWithParams("TEST", stepBar, entryParams()), toCandles(bars),
		toCandles(dailies(bars[0].t, 20, fixtureATR)), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
}

func TestEngineEntryOnConfirmation(t *testing.T) {
	bars, g := engineBars()
	f := bars[g].c
	// The fill bar opens at f; the next bar dips below the stop f − 1 ATR (fixtureATR 2).
	bars = append(bars, bar{bars[g].t.Add(stepBar), f, f, f, f}, bar{bars[g].t.Add(2 * stepBar), f, f, f - 3, f - 3})
	res := runEngine(bars)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-f) > eps || !tr.EntryTime.Equal(bars[g+1].t) {
		t.Fatalf("entry %v @ %v, want %v @ %v (next bar's open)", tr.EntryPrice, tr.EntryTime, f, bars[g+1].t)
	}
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-(f-fixtureATR)) > eps || !tr.ExitTime.Equal(bars[g+2].t) {
		t.Fatalf("exit %q %v @ %v, want SL at %v on %v", tr.Reason, tr.ExitPrice, tr.ExitTime, f-fixtureATR, bars[g+2].t)
	}
}

func TestEngineDropsSignalWhenNextSlotMissing(t *testing.T) {
	bars, g := engineBars()
	f := bars[g].c
	bars = append(bars, bar{bars[g].t.Add(2 * stepBar), f, f, f, f})
	if res := runEngine(bars); len(res.Trades) != 0 {
		t.Fatalf("trades = %d, want 0 (fill slot missing): %+v", len(res.Trades), res.Trades)
	}
}
