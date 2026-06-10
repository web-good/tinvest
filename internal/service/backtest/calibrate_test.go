package backtest

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

func tinyCandles(n int) []backtest.Candle {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]backtest.Candle, n)
	for i := 0; i < n; i++ {
		price := 100.0 + float64(i%7)
		out[i] = backtest.Candle{
			Time: base.Add(time.Duration(i) * time.Hour),
			Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 100,
		}
	}
	return out
}

func TestApplyFieldIntAndFloat(t *testing.T) {
	p := rusal.DefaultParams()
	updated, err := applyField(p, "EMAPeriod", 50) // int field
	if err != nil {
		t.Fatal(err)
	}
	if updated.(adaptive.Params).EMAPeriod != 50 {
		t.Fatalf("EMAPeriod = %d, want 50", updated.(adaptive.Params).EMAPeriod)
	}
	updated2, err := applyField(p, "SLMult", 2.5) // float64 field
	if err != nil {
		t.Fatal(err)
	}
	if updated2.(adaptive.Params).SLMult != 2.5 {
		t.Fatalf("SLMult = %f, want 2.5", updated2.(adaptive.Params).SLMult)
	}
}

func TestApplyFieldUnknownErrors(t *testing.T) {
	if _, err := applyField(rusal.DefaultParams(), "Nonexistent", 1); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestRunGridCartesianProduct(t *testing.T) {
	b, _ := Lookup("RUAL")
	grid := Grid{
		"EMAPeriod": {12, 21},
		"SLMult":    {1.0, 1.5, 2.0},
	}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	results, err := RunGrid(b, grid, tinyCandles(400), nil, cfg, "profit_factor", 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 6 { // 2 * 3
		t.Fatalf("combos = %d, want 6", len(results))
	}
}

func TestRunGridRanksByMetric(t *testing.T) {
	in := []CalibResult{
		{Metrics: backtest.Metrics{ProfitFactor: 1.2, MaxDrawdown: 500, TotalTrades: 30}},
		{Metrics: backtest.Metrics{ProfitFactor: 2.5, MaxDrawdown: 900, TotalTrades: 30}},
		{Metrics: backtest.Metrics{ProfitFactor: 0.8, MaxDrawdown: 100, TotalTrades: 30}},
	}
	byPF := rankResults(append([]CalibResult(nil), in...), "profit_factor", 0)
	if byPF[0].Metrics.ProfitFactor != 2.5 {
		t.Fatalf("top PF = %f, want 2.5", byPF[0].Metrics.ProfitFactor)
	}
	byDD := rankResults(append([]CalibResult(nil), in...), "max_drawdown", 0)
	if byDD[0].Metrics.MaxDrawdown != 100 {
		t.Fatalf("top DD = %f, want 100", byDD[0].Metrics.MaxDrawdown)
	}
}

func TestRankResultsMinTradesFloor(t *testing.T) {
	in := []CalibResult{
		{Metrics: backtest.Metrics{ProfitFactor: 3.0, TotalTrades: 2}},  // high PF, too few trades
		{Metrics: backtest.Metrics{ProfitFactor: 1.2, TotalTrades: 25}}, // qualified
	}
	got := rankResults(append([]CalibResult(nil), in...), "profit_factor", 10)
	if got[0].Metrics.TotalTrades != 25 {
		t.Fatalf("top trades = %d, want 25 (qualified combo ranks ahead of a 2-trade fluke)", got[0].Metrics.TotalTrades)
	}
}

func TestRankResultsSortino(t *testing.T) {
	in := []CalibResult{
		{Metrics: backtest.Metrics{Sortino: 0.5, TotalTrades: 30}},
		{Metrics: backtest.Metrics{Sortino: 1.5, TotalTrades: 30}},
	}
	got := rankResults(append([]CalibResult(nil), in...), "sortino", 0)
	if got[0].Metrics.Sortino != 1.5 {
		t.Fatalf("top sortino = %f, want 1.5", got[0].Metrics.Sortino)
	}
}

func TestRunGridUnknownMetricErrors(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Lot: 1}
	if _, err := RunGrid(b, Grid{}, tinyCandles(400), nil, cfg, "sharpe", 0, 16); err == nil {
		t.Fatal("expected error for unknown metric")
	}
}

func TestRunGridUnknownFieldErrors(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Lot: 1}
	if _, err := RunGrid(b, Grid{"Bogus": {1, 2}}, tinyCandles(400), nil, cfg, "profit_factor", 0, 16); err == nil {
		t.Fatal("expected error for unknown grid field")
	}
}

func TestRenderCalibrationMarkdown(t *testing.T) {
	results := []CalibResult{
		{Params: rusal.DefaultParams(), Metrics: backtest.Metrics{ProfitFactor: 2.0, NetPnL: 1000, TotalTrades: 5}},
	}
	out := RenderCalibrationMarkdown("profit_factor", results, 10)
	if !strings.Contains(out, "profit_factor") || !strings.Contains(out, "Калибровка") {
		t.Fatalf("calibration markdown missing headers:\n%s", out)
	}
}

func TestRenderCalibrationMarkdownTopParams(t *testing.T) {
	first := rusal.DefaultParams()
	second, err := applyField(first, "EMAPeriod", 99)
	if err != nil {
		t.Fatal(err)
	}
	results := []CalibResult{
		{Params: first, Metrics: backtest.Metrics{ProfitFactor: 2.0, TotalTrades: 30}},
		{Params: second, Metrics: backtest.Metrics{ProfitFactor: 1.5, TotalTrades: 25}},
	}
	out := RenderCalibrationMarkdown("profit_factor", results, 10)
	// Both the best and the runner-up combos must have their own params block.
	if strings.Count(out, "| Параметр | Значение |") != 2 {
		t.Fatalf("want a params table per top combo (2), got:\n%s", out)
	}
	if !strings.Contains(out, "#2") {
		t.Fatalf("runner-up combo not rendered:\n%s", out)
	}
	if !strings.Contains(out, "99") {
		t.Fatalf("runner-up params (EMAPeriod=99) not rendered:\n%s", out)
	}
}
