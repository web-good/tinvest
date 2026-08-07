package backtest

import (
	"reflect"
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

func TestVisibleDaily(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	day := func(y int, m time.Month, d int) Candle {
		return Candle{
			Time:  time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
			High:  float64(d) + 0.5,
			Low:   float64(d) - 0.5,
			Close: float64(d),
		}
	}
	daily := []Candle{day(2026, 1, 1), day(2026, 1, 2), day(2026, 1, 3)}

	t3 := time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)
	closes, highs, lows, times := visibleDaily(daily, t3, msk)
	if len(closes) != 2 || closes[0] != 1 || closes[1] != 2 {
		t.Fatalf("closes on Jan 3 = %v, want [1 2]", closes)
	}
	if len(highs) != 2 || len(lows) != 2 || len(times) != 2 {
		t.Fatalf("series not index-aligned: %d closes, %d highs, %d lows, %d times",
			len(closes), len(highs), len(lows), len(times))
	}
	if highs[1] != 2.5 || lows[1] != 1.5 {
		t.Fatalf("highs/lows on Jan 3 = %v/%v, want 2.5/1.5", highs[1], lows[1])
	}
	if !times[1].Equal(daily[1].Time) {
		t.Fatalf("times[1] = %v, want %v", times[1], daily[1].Time)
	}

	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	if closes, _, _, times := visibleDaily(daily, t1, msk); len(closes) != 0 || len(times) != 0 {
		t.Fatalf("visible on Jan 1 = %v / %v, want empty", closes, times)
	}

	t9 := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	if closes, _, _, times := visibleDaily(daily, t9, msk); len(closes) != 3 || len(times) != 3 {
		t.Fatalf("visible on Jan 9 = %d closes / %d times, want 3/3", len(closes), len(times))
	}
}

func TestRunPopulatesDailyTimes(t *testing.T) {
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC) // понедельник
	candles := make([]Candle, 0, 4)
	for i := 0; i < 4; i++ {
		candles = append(candles, Candle{
			Time: base.AddDate(0, 0, i), High: 101, Low: 99, Close: 100, Volume: 10,
		})
	}
	daily := []Candle{
		{Time: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), High: 102, Low: 98, Close: 100},
		{Time: time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC), High: 103, Low: 97, Close: 101},
	}
	var seen []time.Time
	var seenCloses []float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seen = md.DailyTimes
		seenCloses = md.DailyCloses
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, daily, nil, Config{InitialCash: 1000, Fraction: 1.0, Lot: 1})
	if len(seen) != len(seenCloses) {
		t.Fatalf("DailyTimes len %d != DailyCloses len %d", len(seen), len(seenCloses))
	}
	if len(seen) != 2 {
		t.Fatalf("DailyTimes on the last bar = %d, want 2", len(seen))
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

func TestEngineFillsATRSLAtStopLevel(t *testing.T) {
	// Вход по 100, затем sell ATRSL со StopLoss 95 на баре с open 97 и close 90:
	// исполнение должно быть по min(95, 97) = 95, а не по close 90.
	candles := flatCandles([]float64{10, 100, 90})
	candles[2].Open = 97
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		if md.Position != nil && md.Price == 90 {
			return model.Signal{Kind: model.SignalSell, Reason: "ATRSL", StopLoss: 95}
		}
		return model.Signal{}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 1000, Fraction: 1, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 95 {
		t.Fatalf("ATRSL exit price = %v, want 95 (stop level)", got)
	}
}

func TestEngineStopReasonWithZeroStopFallsBackToClose(t *testing.T) {
	// Вход по 100, затем sell ATRSL со StopLoss 0 (не задан) на баре с open 97 и
	// close 90: гвард должен оставить исполнение по close 90, а не min(0, 97) = 0.
	candles := flatCandles([]float64{10, 100, 90})
	candles[2].Open = 97
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		if md.Position != nil && md.Price == 90 {
			return model.Signal{Kind: model.SignalSell, Reason: "ATRSL", StopLoss: 0}
		}
		return model.Signal{}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 1000, Fraction: 1, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 90 {
		t.Fatalf("zero-stop ATRSL exit price = %v, want 90 (close fill, not min(0, open))", got)
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

// Точечная проверка отбора завершённых серий на руками посчитанных числах. Совпадение
// сборки с тем, что Run отдаёт в Decide, проверяет TestAssembleMarketDataMatchesWhatRunFeedsDecide.
func TestAssembleMarketDataSelectsCompletedSeries(t *testing.T) {
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC) // Monday 10:00
	window := []Candle{
		{Time: base, Open: 10, High: 11, Low: 9, Close: 10, Volume: 100},
		{Time: base.Add(time.Hour), Open: 10, High: 12, Low: 10, Close: 11, Volume: 200},
	}
	daily := []Candle{
		{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Close: 8, High: 9, Low: 7},   // completed
		{Time: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), Close: 10, High: 11, Low: 9}, // current day -> excluded
	}
	htf := []Candle{
		{Time: base.Add(-4 * time.Hour), Close: 9, High: 9.5, Low: 8.5}, // closed by base+1h
		{Time: base.Add(4 * time.Hour), Close: 12, High: 13, Low: 11},   // not closed
	}
	cur := window[len(window)-1].Time

	md := AssembleMarketData(window, daily, htf, cur)

	if md.Price != 11 {
		t.Fatalf("Price = %v, want 11", md.Price)
	}
	if len(md.Closes) != 2 || md.Closes[1] != 11 {
		t.Fatalf("Closes = %v, want last 11", md.Closes)
	}
	if len(md.DailyCloses) != 1 || md.DailyCloses[0] != 8 {
		t.Fatalf("DailyCloses = %v, want [8]", md.DailyCloses)
	}
	if len(md.HTFCloses) != 1 || md.HTFCloses[0] != 9 {
		t.Fatalf("HTFCloses = %v, want [9]", md.HTFCloses)
	}
}

// Живые раннеры собирают MarketData через AssembleMarketData + TodayExtent, а бэктест — в
// Run своим внутренним кодом. Расхождение между этими двумя сборками означает, что live
// торгует не по той стратегии, которую откалибровал бэктест, — и оно не видно ни одному
// тесту, который проверяет AssembleMarketData по руками посчитанным числам. Здесь снимок,
// который Run реально передал в Decide, сравнивается со снимком публичной сборки целиком
// (reflect.DeepEqual), поэтому новое поле MarketData автоматически попадает под проверку.
func TestAssembleMarketDataMatchesWhatRunFeedsDecide(t *testing.T) {
	// Три торговых дня по 8 часовых баров: смена MSK-суток внутри ленты (TodayHigh/Low),
	// закрытие 4H-баров внутри дня (HTF) и появление новых завершённых дневок (Daily).
	// Сессия начинается в 01:00 MSK, то есть первые два бара каждого дня приходятся ещё на
	// предыдущие сутки UTC, и на них же посажен размах-«шип»: календарь дня обязан считаться
	// по Москве, иначе шип выпадает из TodayHigh/TodayLow и снимки разойдутся.
	msk := time.FixedZone("MSK", 3*60*60)
	start := time.Date(2026, 1, 12, 1, 0, 0, 0, msk) // понедельник
	var candles []Candle
	price := 100.0
	for d := 0; d < 3; d++ {
		day := start.AddDate(0, 0, d)
		for h := 0; h < 8; h++ {
			price += float64((d*8+h)%5) - 2
			width := 1.5
			if h == 0 {
				width = 50
			}
			ts := day.Add(time.Duration(h) * time.Hour)
			candles = append(candles, Candle{Time: ts, Open: price, High: price + width, Low: price - width, Close: price, Volume: int64(100 + h)})
		}
	}
	var daily, htf []Candle
	for d := -2; d < 3; d++ {
		t0 := time.Date(2026, 1, 12+d, 0, 0, 0, 0, msk)
		daily = append(daily, Candle{Time: t0, Open: 90 + float64(d), High: 95 + float64(d), Low: 85 + float64(d), Close: 92 + float64(d)})
	}
	for i := 0; i < 20; i++ {
		t0 := start.Add(-8 * time.Hour).Add(time.Duration(i) * 4 * time.Hour)
		htf = append(htf, Candle{Time: t0, Open: 80 + float64(i), High: 85 + float64(i), Low: 75 + float64(i), Close: 82 + float64(i)})
	}

	const lookback = 4
	var seen []strategy.MarketData
	s := scriptedStrategy{lookback: lookback, decide: func(md strategy.MarketData) model.Signal {
		seen = append(seen, md)
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, daily, htf, Config{InitialCash: 100000, Fraction: 1, Lot: 1})

	if want := len(candles) - lookback + 1; len(seen) != want {
		t.Fatalf("Decide вызван %d раз, want %d", len(seen), want)
	}
	for k, got := range seen {
		i := k + lookback - 1
		want := AssembleMarketData(candles[i-lookback+1:i+1], daily, htf, candles[i].Time)
		want.TodayHigh, want.TodayLow = TodayExtent(candles, i)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("бар %d (%s): Run передал в Decide снимок, отличный от AssembleMarketData+TodayExtent\n got: %+v\nwant: %+v",
				i, candles[i].Time.Format(time.RFC3339), got, want)
		}
	}

	// Проверка не должна быть пустой: лента обязана двигать каждое из сравниваемых
	// семейств полей, иначе DeepEqual сверяет нули с нулями.
	first, last := seen[0], seen[len(seen)-1]
	switch {
	case len(last.DailyCloses) <= len(first.DailyCloses) || len(first.DailyCloses) == 0:
		t.Fatalf("лента не добавляет завершённых дневок: %d -> %d", len(first.DailyCloses), len(last.DailyCloses))
	case len(last.HTFCloses) <= len(first.HTFCloses) || len(first.HTFCloses) == 0:
		t.Fatalf("лента не закрывает 4H-баров: %d -> %d", len(first.HTFCloses), len(last.HTFCloses))
	case last.TodayHigh == first.TodayHigh:
		t.Fatalf("лента не меняет MSK-сутки: TodayHigh неизменен (%v)", last.TodayHigh)
	}
}

func TestConfigHTFSpanDefaultsTo4H(t *testing.T) {
	if got := (Config{}).htfSpan(); got != 4*time.Hour {
		t.Fatalf("zero HTFInterval must mean 4h, got %v", got)
	}
	if got := (Config{HTFInterval: time.Hour}).htfSpan(); got != time.Hour {
		t.Fatalf("explicit HTFInterval must win, got %v", got)
	}
}

func TestAssembleMarketDataWithHourlyHTF(t *testing.T) {
	base := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	window := []Candle{{Time: base.Add(2 * time.Hour), Open: 10, High: 11, Low: 9, Close: 10}}
	htf := []Candle{
		{Time: base, High: 2, Low: 1, Close: 1.5},
		{Time: base.Add(time.Hour), High: 3, Low: 2, Close: 2.5},
		{Time: base.Add(2 * time.Hour), High: 4, Low: 3, Close: 3.5},
	}
	cur := base.Add(2 * time.Hour) // третий часовой бар ещё формируется

	md := AssembleMarketDataWithHTFInterval(window, nil, htf, cur, time.Hour)
	if len(md.HTFCloses) != 2 {
		t.Fatalf("hourly HTF: got %d completed bars, want 2", len(md.HTFCloses))
	}
	if md.HTFCloses[len(md.HTFCloses)-1] != 2.5 {
		t.Fatalf("last completed hourly close = %v want 2.5", md.HTFCloses[len(md.HTFCloses)-1])
	}
	if len(md.HTFHighs) != 2 || len(md.HTFLows) != 2 {
		t.Fatalf("highs/lows must stay index-aligned with closes: %d/%d", len(md.HTFHighs), len(md.HTFLows))
	}

	// Прежняя 4-часовая семантика: ни один бар ещё не закрыт.
	if md4 := AssembleMarketData(window, nil, htf, cur); len(md4.HTFCloses) != 0 {
		t.Fatalf("4h default: got %d completed bars, want 0", len(md4.HTFCloses))
	}
}

// TestHTFCursorMatchesVisibleCompletedHTF pins htfCursor's incremental prefix to be
// exactly what visibleCompletedHTF computes by rescanning from scratch, for every cur
// in an increasing sequence — including the exact closing boundary (a bar whose
// close time equals cur must be visible) and the empty-prefix case (cur before the
// first bar closes).
func TestHTFCursorMatchesVisibleCompletedHTF(t *testing.T) {
	interval := time.Hour
	base := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	htf := []Candle{
		{Time: base, High: 2, Low: 1, Close: 1.5},
		{Time: base.Add(1 * time.Hour), High: 3, Low: 2, Close: 2.5},
		{Time: base.Add(2 * time.Hour), High: 4, Low: 3, Close: 3.5},
		{Time: base.Add(4 * time.Hour), High: 5, Low: 4, Close: 4.5}, // gap: no bar at +3h
	}

	// Increasing cur values, including: before any bar closes (empty prefix), the
	// exact boundary of each close time, mid-bar values, and past the last close.
	curs := []time.Time{
		base.Add(-time.Minute),              // empty prefix
		base,                                // before first bar closes (close at base+1h)
		base.Add(30 * time.Minute),          // still empty
		base.Add(1 * time.Hour),             // exact boundary: bar 0 closes now
		base.Add(90 * time.Minute),          // between boundaries
		base.Add(2 * time.Hour),             // exact boundary: bar 1 closes now
		base.Add(3 * time.Hour),             // exact boundary: bar 2 closes now
		base.Add(3*time.Hour + time.Minute), // still only 3 visible (gap)
		base.Add(5 * time.Hour),             // exact boundary: bar 3 closes now
		base.Add(24 * time.Hour),            // long past the end
	}

	cur := newHTFCursor(htf, interval)
	for _, at := range curs {
		gotC, gotH, gotL := cur.visible(at)
		wantC, wantH, wantL := visibleCompletedHTF(htf, at, interval)
		if !reflect.DeepEqual(gotC, wantC) {
			t.Fatalf("cur=%v closes = %v, want %v", at, gotC, wantC)
		}
		if !reflect.DeepEqual(gotH, wantH) {
			t.Fatalf("cur=%v highs = %v, want %v", at, gotH, wantH)
		}
		if !reflect.DeepEqual(gotL, wantL) {
			t.Fatalf("cur=%v lows = %v, want %v", at, gotL, wantL)
		}
	}
}

// TestHTFCursorNilForEmptyHTF pins the no-HTF short-circuit: a nil htf slice must
// yield a nil cursor whose visible() is a zero-cost no-op returning nil, nil, nil.
func TestHTFCursorNilForEmptyHTF(t *testing.T) {
	cur := newHTFCursor(nil, time.Hour)
	if cur != nil {
		t.Fatalf("newHTFCursor(nil, ...) = %v, want nil cursor", cur)
	}
	c, h, l := cur.visible(time.Now())
	if c != nil || h != nil || l != nil {
		t.Fatalf("nil cursor visible() = %v/%v/%v, want nil/nil/nil", c, h, l)
	}
}
