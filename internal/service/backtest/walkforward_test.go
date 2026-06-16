package backtest

import (
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestWalkForwardFolds(t *testing.T) {
	tests := []struct {
		name                    string
		from, to                time.Time
		trainMonths, testMonths int
		wantFolds               int
		wantErr                 bool
	}{
		{
			name: "24m/12train/3test -> 4 folds",
			from: date(2024, time.January, 1), to: date(2026, time.January, 1),
			trainMonths: 12, testMonths: 3, wantFolds: 4,
		},
		{
			name: "9m/3train/3test -> 2 folds",
			from: date(2025, time.January, 1), to: date(2025, time.October, 1),
			trainMonths: 3, testMonths: 3, wantFolds: 2,
		},
		{
			name: "train+test exceed window -> error",
			from: date(2025, time.January, 1), to: date(2025, time.April, 1),
			trainMonths: 3, testMonths: 3, wantErr: true,
		},
		{
			name: "zero train -> error",
			from: date(2025, time.January, 1), to: date(2025, time.October, 1),
			trainMonths: 0, testMonths: 3, wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			folds, err := walkForwardFolds(tc.from, tc.to, tc.trainMonths, tc.testMonths)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %d folds", len(folds))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(folds) != tc.wantFolds {
				t.Fatalf("folds = %d, want %d", len(folds), tc.wantFolds)
			}
		})
	}
}

func TestWalkForwardFoldsBoundaries(t *testing.T) {
	folds, err := walkForwardFolds(date(2025, time.January, 1), date(2025, time.October, 1), 3, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fold 0: train Jan-Apr, test Apr-Jul. Fold 1: train Apr-Jul, test Jul-Oct.
	if !folds[0].trainFrom.Equal(date(2025, time.January, 1)) || !folds[0].testTo.Equal(date(2025, time.July, 1)) {
		t.Errorf("fold0 = %+v", folds[0])
	}
	if !folds[1].trainFrom.Equal(date(2025, time.April, 1)) || !folds[1].testTo.Equal(date(2025, time.October, 1)) {
		t.Errorf("fold1 = %+v", folds[1])
	}
}

func TestSliceByRange(t *testing.T) {
	mk := func(h int) backtest.Candle {
		return backtest.Candle{Time: date(2025, time.January, 1).Add(time.Duration(h) * time.Hour)}
	}
	candles := []backtest.Candle{mk(0), mk(1), mk(2), mk(3), mk(4)}
	got := sliceByRange(candles, date(2025, time.January, 1).Add(time.Hour), date(2025, time.January, 1).Add(3*time.Hour))
	// half-open [1h, 3h): expect bars at h=1 and h=2.
	if len(got) != 2 || !got[0].Time.Equal(mk(1).Time) || !got[1].Time.Equal(mk(2).Time) {
		t.Fatalf("sliceByRange = %+v", got)
	}
}

func TestTradesEnteredFrom(t *testing.T) {
	tr := func(h int) backtest.Trade {
		return backtest.Trade{EntryTime: date(2025, time.January, 1).Add(time.Duration(h) * time.Hour)}
	}
	trades := []backtest.Trade{tr(0), tr(1), tr(2), tr(3)}
	boundary := date(2025, time.January, 1).Add(2 * time.Hour)
	got := tradesEnteredFrom(trades, boundary)
	// Keep entries at/after boundary: h=2 and h=3.
	if len(got) != 2 || !got[0].EntryTime.Equal(tr(2).EntryTime) {
		t.Fatalf("tradesEnteredFrom = %+v", got)
	}
}

func TestSumPnL(t *testing.T) {
	trades := []backtest.Trade{{PnL: 100}, {PnL: -40}, {PnL: 10}}
	if got := sumPnL(trades); got != 70 {
		t.Fatalf("sumPnL = %v, want 70", got)
	}
}

func TestTradeReplayDrawdownPct(t *testing.T) {
	// Equity from 1000: +200 -> 1200 (peak), -360 -> 840. DD = (1200-840)/1200 = 0.30.
	trades := []backtest.Trade{{PnL: 200}, {PnL: -360}, {PnL: 60}}
	got := tradeReplayDrawdownPct(trades, 1000)
	if got < 0.2999 || got > 0.3001 {
		t.Fatalf("tradeReplayDrawdownPct = %v, want ~0.30", got)
	}
	if tradeReplayDrawdownPct(nil, 1000) != 0 {
		t.Fatalf("empty trades should give 0 drawdown")
	}
	if tradeReplayDrawdownPct(trades, 0) != 0 {
		t.Fatalf("zero cash should give 0 drawdown (guard)")
	}
}

func TestCompoundReturns(t *testing.T) {
	// (1+0.10)(1-0.05)(1+0.20) - 1 = 0.254.
	got := compoundReturns([]float64{0.10, -0.05, 0.20})
	if got < 0.2539 || got > 0.2541 {
		t.Fatalf("compoundReturns = %v, want ~0.254", got)
	}
	if compoundReturns(nil) != 0 {
		t.Fatalf("empty should give 0")
	}
}

func TestParamStability(t *testing.T) {
	folds := []WalkForwardFold{
		{WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "0.8"}}},
		{WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "1.2"}}},
		{Note: "0 OOS-сделок"}, // skipped fold: no WinnerRows, must be ignored
	}
	stable, varied := paramStability(folds)
	if stable["RSIPeriod"] != "6" {
		t.Errorf("RSIPeriod should be stable at 6, got %q", stable["RSIPeriod"])
	}
	if _, ok := stable["StopATRMult"]; ok {
		t.Errorf("StopATRMult should not be stable")
	}
	got := varied["StopATRMult"]
	if len(got) != 2 || got[0] != "0.8" || got[1] != "1.2" {
		t.Errorf("StopATRMult varied = %v, want [0.8 1.2]", got)
	}
}

// alternatingStrategy buys whenever flat and sells the next bar — it produces a trade
// roughly every two bars across the whole series, at deterministic entry times. Lookback
// 1 keeps warm-up trivial. It ignores params, so any swept grid value yields the same run.
type alternatingStrategy struct{}

func (alternatingStrategy) Ticker() string { return "TEST" }
func (alternatingStrategy) Lookback() int  { return 1 }
func (alternatingStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position == nil {
		return model.Signal{Kind: model.SignalBuy}
	}
	return model.Signal{Kind: model.SignalSell, Reason: "TP"}
}

type fakeParams struct{ Threshold int }

func fakeBinding() Binding {
	return Binding{
		DefaultParams: func() any { return fakeParams{Threshold: 1} },
		Build:         func(any) strategy.Strategy { return alternatingStrategy{} },
		ParseParams:   func([]byte) (any, error) { return fakeParams{}, nil },
	}
}

// genHourly builds 1h candles over [from, to) with a slight up-drift so some trades
// profit (keeps PooledMetrics non-degenerate).
func genHourly(from, to time.Time) []backtest.Candle {
	var out []backtest.Candle
	price := 100.0
	for ts, i := from, 0; ts.Before(to); ts, i = ts.Add(time.Hour), i+1 {
		if i%2 == 0 {
			price += 1
		} else {
			price -= 0.5
		}
		out = append(out, backtest.Candle{Time: ts, Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 1})
	}
	return out
}

func TestRunWalkForward(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.October, 1) // 9 months
	candles := genHourly(from, to)
	phases := []Phase{{Grid: Grid{"Threshold": {1, 2}}}}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}

	s, err := RunWalkForward(fakeBinding(), phases, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err != nil {
		t.Fatalf("RunWalkForward: %v", err)
	}
	if len(s.Folds) != 2 {
		t.Fatalf("folds = %d, want 2", len(s.Folds))
	}
	var oosSum int
	for _, f := range s.Folds {
		if f.Note != "" {
			t.Fatalf("fold %d unexpectedly skipped: %s", f.Index, f.Note)
		}
		if f.OOSTrades == 0 {
			t.Fatalf("fold %d has no OOS trades", f.Index)
		}
		if f.WinnerRows == nil {
			t.Fatalf("fold %d missing winner rows", f.Index)
		}
		oosSum += f.OOSTrades
	}
	if s.PooledOOS.TotalTrades != oosSum {
		t.Fatalf("pooled trades = %d, want sum of folds %d", s.PooledOOS.TotalTrades, oosSum)
	}
}

func TestRunWalkForwardNoFold(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.April, 1) // 3 months
	candles := genHourly(from, to)
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}
	_, err := RunWalkForward(fakeBinding(), []Phase{{Grid: Grid{"Threshold": {1}}}}, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err == nil {
		t.Fatal("want error when no fold fits")
	}
}
