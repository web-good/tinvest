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
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
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
	res := Run(buys, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
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
	res2 := Run(sells, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res2.Trades) != 0 || res2.BarsInMarket != 0 {
		t.Fatalf("sell-when-flat changed state: trades=%d inMarket=%d", len(res2.Trades), res2.BarsInMarket)
	}
}

func TestEngineEmptyWhenNotEnoughHistory(t *testing.T) {
	candles := flatCandles([]float64{10, 20})
	s := scriptedStrategy{lookback: 5, decide: func(md strategy.MarketData) model.Signal {
		return model.Signal{Kind: model.SignalBuy}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
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
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
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
	Run(s, candles, nil, nil, Config{InitialCash: 1000, Fraction: 1.0, Lot: 1})
	if seen != 3 {
		t.Fatalf("window size = %d, want 3", seen)
	}
}

func TestVisibleDailyCloses(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	day := func(y int, m time.Month, d int) Candle {
		return Candle{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC), Close: float64(d)}
	}
	daily := []Candle{day(2026, 1, 1), day(2026, 1, 2), day(2026, 1, 3)}

	t3 := time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)
	got := visibleDailyCloses(daily, t3, msk)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("visible on Jan 3 = %v, want [1 2]", got)
	}

	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	if got := visibleDailyCloses(daily, t1, msk); len(got) != 0 {
		t.Fatalf("visible on Jan 1 = %v, want []", got)
	}

	t9 := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	if got := visibleDailyCloses(daily, t9, msk); len(got) != 3 {
		t.Fatalf("visible on Jan 9 = %v, want 3 closes", got)
	}
}

func TestVisibleCompletedHTF(t *testing.T) {
	// Четыре 4H-бара, открытия в UTC: 00:00, 04:00, 08:00, 12:00.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	htf := []Candle{
		{Time: base, High: 11, Low: 9, Close: 10},
		{Time: base.Add(4 * time.Hour), High: 21, Low: 19, Close: 20},
		{Time: base.Add(8 * time.Hour), High: 31, Low: 29, Close: 30},
		{Time: base.Add(12 * time.Hour), High: 41, Low: 39, Close: 40},
	}

	// На 09:00 закрылись бары 00:00 (→04:00) и 04:00 (→08:00); бар 08:00
	// закроется только в 12:00, текущий формирующийся невидим.
	cur := base.Add(9 * time.Hour)
	closes, highs, lows := visibleCompletedHTF(htf, cur, 4*time.Hour)
	if len(closes) != 2 || closes[0] != 10 || closes[1] != 20 {
		t.Fatalf("closes на 09:00 = %v, want [10 20]", closes)
	}
	if len(highs) != 2 || len(lows) != 2 || highs[1] != 21 || lows[1] != 19 {
		t.Fatalf("highs/lows на 09:00 = %v/%v, want выровнены с closes", highs, lows)
	}

	// Ровно на границе закрытия (08:00) видим оба бара: бар 00:00 закрылся в 04:00,
	// бар 04:00 закрывается ровно в 08:00 включительно (правило c.Time.Add(interval) <= cur).
	if c, _, _ := visibleCompletedHTF(htf, base.Add(8*time.Hour), 4*time.Hour); len(c) != 2 {
		t.Fatalf("на 08:00 видимы %v, want 2 (бары 00:00 и 04:00 закрыты)", c)
	}

	// До первого закрытия — пусто.
	if c, _, _ := visibleCompletedHTF(htf, base.Add(time.Hour), 4*time.Hour); len(c) != 0 {
		t.Fatalf("на 01:00 видимы %v, want пусто", c)
	}
}

func TestEngineSuppliesDailyCloses(t *testing.T) {
	candles := []Candle{
		{Time: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
	}
	daily := []Candle{
		{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Close: 1},
		{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Close: 2},
	}

	var seen [][]float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seen = append(seen, append([]float64(nil), md.DailyCloses...))
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, daily, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})

	if len(seen) != 3 {
		t.Fatalf("decided %d bars, want 3", len(seen))
	}
	if len(seen[0]) != 0 {
		t.Errorf("bar on Jan 1 daily = %v, want empty", seen[0])
	}
	if len(seen[1]) != 1 || seen[1][0] != 1 {
		t.Errorf("bar on Jan 2 daily = %v, want [1]", seen[1])
	}
	if len(seen[2]) != 2 || seen[2][1] != 2 {
		t.Errorf("bar on Jan 3 daily = %v, want [1 2]", seen[2])
	}
}

func TestEngineSuppliesHTF(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base.Add(5 * time.Hour), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: base.Add(9 * time.Hour), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
	}
	htf := []Candle{
		{Time: base, High: 11, Low: 9, Close: 10},                     // closes at 04:00
		{Time: base.Add(4 * time.Hour), High: 21, Low: 19, Close: 20}, // closes at 08:00
		{Time: base.Add(8 * time.Hour), High: 31, Low: 29, Close: 30}, // closes at 12:00 (unseen)
	}

	var seen [][]float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seen = append(seen, append([]float64(nil), md.HTFCloses...))
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, nil, htf, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})

	if len(seen) != 2 {
		t.Fatalf("decided %d bars, want 2", len(seen))
	}
	if len(seen[0]) != 1 || seen[0][0] != 10 {
		t.Errorf("bar 05:00 HTF = %v, want [10]", seen[0])
	}
	if len(seen[1]) != 2 || seen[1][1] != 20 {
		t.Errorf("bar 09:00 HTF = %v, want [10 20]", seen[1])
	}
}

// stopExitStrategy buys on the first bar (when flat), then on every subsequent
// bar returns Sell with the configured reason and StopLoss.
type stopExitStrategy struct {
	reason   string
	stopLoss float64
}

func (s stopExitStrategy) Ticker() string { return "TEST" }
func (s stopExitStrategy) Lookback() int  { return 1 }
func (s stopExitStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position == nil {
		return model.Signal{Kind: model.SignalBuy}
	}
	return model.Signal{Kind: model.SignalSell, Reason: s.reason, StopLoss: s.stopLoss}
}

func TestEngineStopFillsAtLevelOnIntrabarPierce(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},                  // buy here
		{Time: base.Add(time.Hour), Open: 100, High: 100.5, Low: 98, Close: 98.5, Volume: 1}, // SL pierced intrabar
	}
	s := stopExitStrategy{reason: "SL", stopLoss: 99}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 99 {
		t.Fatalf("exit = %v, want 99 (filled at stop level, not close 98.5)", got)
	}
}

func TestEngineStopFillsAtOpenOnGapDown(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},              // buy here
		{Time: base.Add(time.Hour), Open: 97, High: 97, Low: 96, Close: 96.5, Volume: 1}, // gap below stop
	}
	s := stopExitStrategy{reason: "SL", stopLoss: 99}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 97 {
		t.Fatalf("exit = %v, want 97 (filled at gap open, worse than stop 99)", got)
	}
}

func TestEngineNonStopSellFillsAtClose(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},                // buy here
		{Time: base.Add(time.Hour), Open: 100, High: 101, Low: 98, Close: 98.5, Volume: 1}, // TP-style exit
	}
	// StopLoss is set but must be ignored for a non-stop reason.
	s := stopExitStrategy{reason: "TP", stopLoss: 99}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 98.5 {
		t.Fatalf("exit = %v, want 98.5 (non-stop sell fills at close)", got)
	}
}

func TestEngineStampsEntryContextOnTrade(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 110})
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, Level: 98, TakeProfit: 108, ATR: 1.5}
		}
		if md.Position != nil && md.Price == 110 {
			return model.Signal{Kind: model.SignalSell, Reason: "TP"}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.SupportLevel != 98 || tr.ResistanceLevel != 108 || tr.ATR != 1.5 {
		t.Fatalf("entry context = {%v %v %v}, want {98 108 1.5}", tr.SupportLevel, tr.ResistanceLevel, tr.ATR)
	}
}

func TestEngineFillsTPAtTarget(t *testing.T) {
	// Bar 1 buys at 100; bar 2 has a high of 130 reaching the TP=120, closing at 110.
	// The TP exit must fill at the target (120), not the close (110).
	candles := []Candle{
		{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},
		{Time: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC), Open: 105, High: 130, Low: 104, Close: 110, Volume: 1},
	}
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, TakeProfit: 120, StopLoss: 90}
		}
		if md.Position != nil {
			return model.Signal{Kind: model.SignalSell, Reason: "TP", TakeProfit: 120}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades=%d want 1", len(res.Trades))
	}
	if res.Trades[0].ExitPrice != 120 {
		t.Fatalf("TP exit price=%f want 120 (filled at target)", res.Trades[0].ExitPrice)
	}
}

func TestEngineSuppliesDailyHighsLowsAndTodayExtent(t *testing.T) {
	msk := mskLoc
	mk := func(y, mo, d, h int, o, hi, lo, c float64) Candle {
		return Candle{Time: time.Date(y, time.Month(mo), d, h, 0, 0, 0, msk), Open: o, High: hi, Low: lo, Close: c, Volume: 1}
	}
	candles := []Candle{
		mk(2026, 1, 2, 10, 10, 12, 9, 11),
		mk(2026, 1, 2, 11, 11, 15, 8, 14),  // day1 running extent now H=15 L=8
		mk(2026, 1, 3, 10, 14, 16, 13, 15), // new day: extent resets to this bar
	}
	daily := []Candle{
		mk(2026, 1, 1, 0, 100, 110, 90, 105),
		mk(2026, 1, 2, 0, 105, 120, 100, 118),
	}
	var seenHi, seenLo []float64
	var seenDailyHighs [][]float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seenHi = append(seenHi, md.TodayHigh)
		seenLo = append(seenLo, md.TodayLow)
		seenDailyHighs = append(seenDailyHighs, append([]float64(nil), md.DailyHighs...))
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, daily, nil, Config{InitialCash: 1000, Fraction: 1, Commission: 0, Lot: 1})

	if seenHi[0] != 12 || seenLo[0] != 9 {
		t.Fatalf("bar0 extent H=%v L=%v want 12/9", seenHi[0], seenLo[0])
	}
	if seenHi[1] != 15 || seenLo[1] != 8 {
		t.Fatalf("bar1 extent H=%v L=%v want 15/8", seenHi[1], seenLo[1])
	}
	if seenHi[2] != 16 || seenLo[2] != 13 {
		t.Fatalf("bar2 extent H=%v L=%v want 16/13", seenHi[2], seenLo[2])
	}
	if len(seenDailyHighs[0]) != 1 || seenDailyHighs[0][0] != 110 {
		t.Fatalf("bar0 daily highs=%v want [110]", seenDailyHighs[0])
	}
	if len(seenDailyHighs[2]) != 2 {
		t.Fatalf("bar2 daily highs len=%d want 2", len(seenDailyHighs[2]))
	}
}

func TestBuildMarketDataCopiesTimes(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	window := []Candle{
		{Time: base, Close: 10, High: 10, Low: 10, Volume: 1},
		{Time: base.Add(time.Hour), Close: 11, High: 11, Low: 11, Volume: 2},
		{Time: base.Add(2 * time.Hour), Close: 12, High: 12, Low: 12, Volume: 3},
	}
	md := buildMarketData(window)
	if len(md.Times) != len(window) {
		t.Fatalf("Times length: want %d, got %d", len(window), len(md.Times))
	}
	for i := range window {
		if !md.Times[i].Equal(window[i].Time) {
			t.Fatalf("Times[%d]: want %v, got %v", i, window[i].Time, md.Times[i])
		}
	}
}

func TestEngineFreezesEntryStopAndTracksFavorable(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 105, 98})
	// Buy at price 100 (bar 1) with a frozen stop of 95. On bar 2 (price 105) the
	// favourable max should rise to 105; on bar 3 (price 98) the position must
	// still carry the frozen stop 95 and the latched max 105, then sell SL.
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, StopLoss: 95}
		}
		if md.Position != nil && md.Price == 98 {
			if md.Position.StopLoss != 95 {
				t.Errorf("frozen StopLoss = %v, want 95", md.Position.StopLoss)
			}
			if md.Position.MaxFavorablePrice != 105 {
				t.Errorf("MaxFavorablePrice = %v, want 105 (latched high)", md.Position.MaxFavorablePrice)
			}
			return model.Signal{Kind: model.SignalSell, Reason: "SL", StopLoss: 95}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if res.Trades[0].Reason != "SL" {
		t.Fatalf("exit reason = %q, want SL", res.Trades[0].Reason)
	}
}
