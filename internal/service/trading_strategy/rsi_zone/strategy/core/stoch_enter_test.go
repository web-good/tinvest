package core

import (
	"math"
	"strings"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// stochOf computes the %D series the strategy will see for md with the given knobs.
func stochOf(t *testing.T, md strategy.MarketData, k, d int) []float64 {
	t.Helper()
	s, _, ok := stochDSeries(md.Highs, md.Lows, md.Closes, k, d)
	if !ok {
		t.Fatal("stochDSeries refused the fixture")
	}
	return s
}

func stochParams(lower float64, window int) Params {
	p := DefaultParams()
	p.UseStoch, p.StochKPeriod, p.StochDSmooth, p.StochLower, p.ZoneWindowBars = 1, 14, 3, lower, window
	return p
}

// trendCloses(3): RSI(4) crosses 25 on the last bar. The stochastic decides.
func TestStochGateOnRSITrigger(t *testing.T) {
	md := fixture(trendCloses(3), mondayNoon)
	st := stochOf(t, md, 14, 3)
	i := len(st) - 1

	t.Run("stoch in zone on the cross bar confirms", func(t *testing.T) {
		sig := NewWithParams("T", stochParams(st[i]+1, 1)).Decide(md)
		if sig.Kind != model.SignalBuy {
			t.Fatalf("Kind = %v, want Buy (stoch %.2f under band %.2f)", sig.Kind, st[i], st[i]+1)
		}
		if !strings.Contains(sig.EntryReason, "; подтверждение: крест дал RSI") {
			t.Fatalf("EntryReason = %q, want the RSI-trigger tail", sig.EntryReason)
		}
	})
	t.Run("stoch never in zone in the window blocks", func(t *testing.T) {
		lowest := math.Min(st[i], math.Min(st[i-1], st[i-2]))
		if sig := NewWithParams("T", stochParams(lowest, 3)).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatalf("Buy with band %.2f at or under every stoch reading of the window", lowest)
		}
	})
}

// trendCloses(4): RSI(4) is already under 25 on the last two bars, so there is no RSI cross.
// Only a stoch cross on the last bar can fire.
func TestStochGateOnStochTrigger(t *testing.T) {
	md := fixture(trendCloses(4), mondayNoon)
	st := stochOf(t, md, 14, 3)
	i := len(st) - 1
	if !(st[i] < st[i-1]) {
		t.Fatalf("fixture drift: stoch %.2f -> %.2f is not falling on the last bar", st[i-1], st[i])
	}
	mid := (st[i-1] + st[i]) / 2 // the band the stoch crosses exactly on bar i

	t.Run("stoch cross with RSI in zone enters", func(t *testing.T) {
		sig := NewWithParams("T", stochParams(mid, 1)).Decide(md)
		if sig.Kind != model.SignalBuy {
			t.Fatalf("Kind = %v, want Buy on the stoch cross", sig.Kind)
		}
		if !strings.Contains(sig.EntryReason, "; подтверждение: крест дал Stoch") {
			t.Fatalf("EntryReason = %q, want the Stoch-trigger tail", sig.EntryReason)
		}
	})
	t.Run("RSI out of its zone blocks the stoch cross", func(t *testing.T) {
		p := stochParams(mid, 1)
		p.RSILower = 10 // RSI(4) is 15.6 on the last bar: no longer in the zone
		if sig := NewWithParams("T", p).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatal("Buy although RSI never read under its band in the window")
		}
	})
	t.Run("gate off keeps the old behaviour: no RSI cross, no entry", func(t *testing.T) {
		if sig := NewWithParams("T", DefaultParams()).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatal("Buy with UseStoch=0 on a bar without an RSI cross")
		}
	})
	t.Run("weekend blocks the stoch trigger", func(t *testing.T) {
		mdW := fixture(trendCloses(4), saturdayNoon)
		if sig := NewWithParams("T", stochParams(mid, 1)).Decide(mdW); sig.Kind == model.SignalBuy {
			t.Fatal("Buy on a Saturday")
		}
	})
	t.Run("unknown daily ATR blocks the stoch trigger", func(t *testing.T) {
		mdA := fixture(trendCloses(4), mondayNoon)
		mdA.DailyHighs, mdA.DailyLows, mdA.DailyCloses, mdA.DailyTimes = nil, nil, nil, nil
		if sig := NewWithParams("T", stochParams(mid, 1)).Decide(mdA); sig.Kind == model.SignalBuy {
			t.Fatal("Buy without a daily ATR")
		}
	})
}

// downtrendCloses: RSI crosses on the last bar but close < EMA(200). A band of 101 puts every
// warmed stoch reading in the zone, so only the trend gate can reject.
func TestStochGateKeepsTheTrendGate(t *testing.T) {
	md := fixture(downtrendCloses(), mondayNoon)
	if sig := NewWithParams("T", stochParams(101, 1)).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("Buy below the trend EMA")
	}
}

// Review Focus 1: UseStoch alone on a literal without stochastic knobs behaves like explicit
// 14/3/20/1 — never like "no entries at all".
func TestUseStochAloneResolvesZeroFields(t *testing.T) {
	for _, closes := range [][]float64{trendCloses(3), trendCloses(4)} {
		md := fixture(closes, mondayNoon)
		bare := DefaultParams()
		bare.UseStoch = 1
		explicit := stochParams(20, 1)
		a := NewWithParams("T", bare).Decide(md)
		b := NewWithParams("T", explicit).Decide(md)
		if a.Kind != b.Kind || a.EntryReason != b.EntryReason {
			t.Fatalf("UseStoch alone = (%v, %q), explicit 14/3/20/1 = (%v, %q)", a.Kind, a.EntryReason, b.Kind, b.EntryReason)
		}
	}
}

func TestNegativeStochPeriodRefusesEntry(t *testing.T) {
	md := fixture(trendCloses(3), mondayNoon)
	for _, p := range []Params{
		func() Params { p := stochParams(101, 1); p.StochKPeriod = -1; return p }(),
		func() Params { p := stochParams(101, 1); p.StochDSmooth = -1; return p }(),
	} {
		if sig := NewWithParams("T", p).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatalf("Buy with a negative stochastic period: %+v", p)
		}
	}
}

// The whole engine path: with the gate on, a stoch cross (no RSI cross) opens the trade.
func TestEngineEntersOnStochTrigger(t *testing.T) {
	probe := NewWithParams("TEST", stochParams(20, 1))
	lookback := probe.Lookback()
	closes := engineUptrendCloses(lookback, 4)
	entryBarTime := mondayNoon.Add(-30 * time.Minute) // a Monday; the exit bar lands on mondayNoon
	candles := engineCandles(closes, entryBarTime)

	highs, lows := make([]float64, len(candles)), make([]float64, len(candles))
	for i, c := range candles {
		highs[i], lows[i] = c.High, c.Low
	}
	st, _, ok := stochDSeries(highs, lows, closes, 14, 3)
	if !ok {
		t.Fatal("stochDSeries refused the engine fixture")
	}
	i := len(st) - 1
	if !(st[i] < st[i-1]) {
		t.Fatalf("fixture drift: stoch %.2f -> %.2f is not falling on the last bar", st[i-1], st[i])
	}
	s := NewWithParams("TEST", stochParams((st[i-1]+st[i])/2, 1))

	gapOpen := closes[len(closes)-1] - 3*dailyWidth
	candles = append(candles, bt.Candle{Time: mondayNoon, Open: gapOpen, High: gapOpen, Low: gapOpen - 1, Close: gapOpen - 0.5})
	daily := engineDailyCandles(mondayNoon, 40, dailyWidth, dailyWidth/10)
	res := bt.Run(s, candles, daily, nil, bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Lot: 1})

	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-closes[len(closes)-1]) > 1e-6 {
		t.Fatalf("EntryPrice = %v, want the last down bar's close %v", tr.EntryPrice, closes[len(closes)-1])
	}
	if !strings.Contains(tr.EntryReason, "крест дал Stoch") {
		t.Fatalf("EntryReason = %q, want the Stoch trigger", tr.EntryReason)
	}
}
