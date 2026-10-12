// engine_test.go runs drop_bounce through the real backtest engine to pin what it relies on: daily
// volumes reaching the strategy, entry at the next bar's open, the daily-ATR stop filled at the gap
// open overnight, the RSI exit, and a signal dropped when the next slot is missing.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// toCandles converts bars to candles that all trade vol.
func toCandles(bars []bar, vol int64) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c, Volume: vol}
	}
	return out
}

// spikeBars is Lookback flat bars at 100 then a red bar 100 → 99.5 starting at t: the engine
// decides only from bar Lookback()−1 on, so the lead puts the signal past it.
func spikeBars(t string) []bar {
	n := NewWithParams("TEST", step15, DefaultParams()).Lookback()
	end := at(t)
	bars := path(end.Add(-time.Duration(n)*step15).Format("2006-01-02 15:04"), flatN(n, 100))
	return append(bars, bar{end, 100, 100, 99.5, 99.5})
}

// runEngine runs the default drop_bounce over bars where the bar at spike trades 0.5 of an average
// day (dailies of range fixtureATR and volume fixtureDayVol before the first bar).
func runEngine(bars []bar, spike int) bt.Result {
	candles := toCandles(bars, quietVol)
	candles[spike].Volume = fixtureDayVol / 2
	return bt.Run(NewWithParams("TEST", step15, DefaultParams()), candles,
		toCandles(dailies(bars[0].t, 20, fixtureATR), fixtureDayVol), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
}

func TestEngineOvernightGapStop(t *testing.T) {
	bars := spikeBars("2026-09-14 13:00")
	sig := len(bars) - 1
	f := 99.7 // the fill bar opens away from the signal close 99.5
	gap := f * 0.97
	bars = append(bars,
		bar{at("2026-09-14 13:15"), f, f, f, f},
		bar{at("2026-09-15 07:00"), gap, gap, gap * 0.999, gap},
	)
	res := runEngine(bars, sig)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-f) > eps || !tr.EntryTime.Equal(at("2026-09-14 13:15")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:15 (next bar's open)", tr.EntryPrice, tr.EntryTime, f)
	}
	if stop := f - fixtureATR; gap >= stop {
		t.Fatalf("fixture: gap open %v must be below the stop %v", gap, stop)
	}
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-gap) > 1e-6 || !tr.ExitTime.Equal(at("2026-09-15 07:00")) {
		t.Fatalf("exit %q @ %v %v, want SL at the gap open %v next morning", tr.Reason, tr.ExitPrice, tr.ExitTime, gap)
	}
}

func TestEngineRSIExit(t *testing.T) {
	bars := spikeBars("2026-09-14 13:00")
	sig := len(bars) - 1
	bars = append(bars, path("2026-09-14 13:15", steps(99.7, 15, 0.3))...)
	k := firstCross(bars, 6, 70, sig+1)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 after the fill")
	}
	res := runEngine(bars, sig)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	if tr := res.Trades[0]; tr.Reason != "RSI" || !tr.ExitTime.Equal(bars[k].t) {
		t.Fatalf("exit %q @ %v, want RSI at %v", tr.Reason, tr.ExitTime, bars[k].t)
	}
}

// Zero-volume dailies leave no average (they are discarded), so the engine-fed strategy never enters.
func TestEngineZeroDailyVolumeNoTrade(t *testing.T) {
	bars := append(spikeBars("2026-09-14 13:00"), bar{at("2026-09-14 13:15"), 99.7, 99.7, 99.7, 99.7})
	candles := toCandles(bars, fixtureDayVol) // every bar trades a whole average day — if one were known
	res := bt.Run(NewWithParams("TEST", step15, DefaultParams()), candles,
		toCandles(dailies(bars[0].t, 20, fixtureATR), 0), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 0 || res.BarsInMarket != 0 {
		t.Fatalf("trades = %d, barsInMarket = %d, want no position", len(res.Trades), res.BarsInMarket)
	}
}

// A signal on the 23:30 bar may fill only at 23:45; when that candle is missing the next candle
// is the next morning and the signal is dropped instead of carrying over.
func TestEngineDropsSignalWhenNextSlotMissing(t *testing.T) {
	bars := spikeBars("2026-09-14 23:30")
	sig := len(bars) - 1
	bars = append(bars,
		bar{at("2026-09-15 07:00"), 99.5, 99.5, 99.5, 99.5},
		bar{at("2026-09-15 07:15"), 99.5, 99.5, 99.5, 99.5},
	)
	res := runEngine(bars, sig)
	if len(res.Trades) != 0 || res.BarsInMarket != 0 {
		t.Fatalf("trades = %d, barsInMarket = %d, want no position: %+v", len(res.Trades), res.BarsInMarket, res.Trades)
	}
}
