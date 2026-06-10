package backtest

import (
	"reflect"
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

func TestRunPhasesNarrowsAndCarriesSeeds(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	phases := []Phase{
		{Name: "core", KeepTop: 2, Grid: Grid{"EMAPeriod": {44, 55}}},
		{Name: "gates", Grid: Grid{"SLMult": {1.0, 1.5, 2.0}}},
	}
	var prog []PhaseProgress
	results, err := RunPhases(b, phases, tinyCandles(400), nil, cfg, "profit_factor", 0, 16,
		func(p PhaseProgress) { prog = append(prog, p) })
	if err != nil {
		t.Fatal(err)
	}
	if len(prog) != 2 {
		t.Fatalf("want 2 phase callbacks, got %d", len(prog))
	}
	if prog[0].Combos != 2 {
		t.Fatalf("phase 1 combos = %d, want 2", prog[0].Combos)
	}
	if prog[0].Kept != 2 {
		t.Fatalf("phase 1 kept = %d, want 2", prog[0].Kept)
	}
	if prog[1].Combos != 6 {
		t.Fatalf("phase 2 combos = %d, want 6", prog[1].Combos)
	}
	if len(results) != 6 {
		t.Fatalf("final results = %d, want 6", len(results))
	}
	for i, r := range results {
		ema := r.Params.(adaptive.Params).EMAPeriod
		if ema != 44 && ema != 55 {
			t.Fatalf("result %d EMAPeriod = %d, want carried 44 or 55", i, ema)
		}
	}
}

func TestRunPhasesKeepTopDefaultsAndClamps(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	phases := []Phase{
		{Grid: Grid{"EMAPeriod": {44, 55}}},
		{Grid: Grid{"SLMult": {1.0, 1.5}}},
	}
	var prog []PhaseProgress
	results, err := RunPhases(b, phases, tinyCandles(400), nil, cfg, "profit_factor", 0, 16,
		func(p PhaseProgress) { prog = append(prog, p) })
	if err != nil {
		t.Fatal(err)
	}
	if prog[0].Kept != 2 {
		t.Fatalf("phase 1 kept = %d, want 2 (default 5 clamped to 2 results)", prog[0].Kept)
	}
	if prog[1].Combos != 4 {
		t.Fatalf("phase 2 combos = %d, want 4", prog[1].Combos)
	}
	if len(results) != 4 {
		t.Fatalf("final results = %d, want 4", len(results))
	}
	if prog[0].Name != "phase-1" {
		t.Fatalf("phase 1 default name = %q, want phase-1", prog[0].Name)
	}
}

func TestRunPhasesSinglePhaseMatchesRunGrid(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	grid := Grid{"EMAPeriod": {44, 55}, "SLMult": {1.0, 1.5}}
	want, err := RunGrid(b, grid, tinyCandles(400), nil, cfg, "profit_factor", 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RunPhases(b, []Phase{{Grid: grid}}, tinyCandles(400), nil, cfg, "profit_factor", 0, 16, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	if !reflect.DeepEqual(got[0].Params, want[0].Params) {
		t.Fatalf("top params differ:\ngot  %+v\nwant %+v", got[0].Params, want[0].Params)
	}
}

func TestRunPhasesEmptyErrors(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Lot: 1}
	if _, err := RunPhases(b, nil, tinyCandles(400), nil, cfg, "profit_factor", 0, 16, nil); err == nil {
		t.Fatal("expected error for empty phases")
	}
}

func TestParsePhasesPhasedFormat(t *testing.T) {
	raw := []byte(`{"phases":[{"name":"core","keepTop":3,"grid":{"EMAPeriod":[44,55]}},{"grid":{"SLMult":[1.0,1.5]}}]}`)
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) != 2 {
		t.Fatalf("phases = %d, want 2", len(phases))
	}
	if phases[0].Name != "core" || phases[0].KeepTop != 3 {
		t.Fatalf("phase 0 = %+v, want name=core keepTop=3", phases[0])
	}
	if got := phases[0].Grid["EMAPeriod"]; len(got) != 2 || got[0] != 44 {
		t.Fatalf("phase 0 grid EMAPeriod = %v, want [44 55]", got)
	}
}

func TestParsePhasesLegacyFlatFormat(t *testing.T) {
	raw := []byte(`{"EMAPeriod":[44,55],"SLMult":[1.0,1.5]}`)
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) != 1 {
		t.Fatalf("phases = %d, want 1 (flat wrapped as single phase)", len(phases))
	}
	if got := phases[0].Grid["EMAPeriod"]; len(got) != 2 {
		t.Fatalf("flat grid EMAPeriod = %v, want 2 values", got)
	}
}

func TestParsePhasesMalformedErrors(t *testing.T) {
	if _, err := ParsePhases([]byte(`{not json`)); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestRunGridDeterministicOrder(t *testing.T) {
	b, _ := Lookup("RUAL")
	grid := Grid{
		"EMAPeriod": {12, 21, 30},
		"SLMult":    {1.0, 1.5, 2.0},
	}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	candles := tinyCandles(600)

	// Run twice; the ranked output must be byte-stable across runs.
	first, err := RunGrid(b, grid, candles, nil, cfg, "profit_factor", 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunGrid(b, grid, candles, nil, cfg, "profit_factor", 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 9 || len(second) != 9 { // 3 * 3
		t.Fatalf("combos = %d/%d, want 9", len(first), len(second))
	}
	for i := range first {
		pf1, pf2 := first[i].Metrics.ProfitFactor, second[i].Metrics.ProfitFactor
		if pf1 != pf2 {
			t.Fatalf("run mismatch at rank %d: PF %v != %v", i, pf1, pf2)
		}
		tr1, tr2 := first[i].Metrics.TotalTrades, second[i].Metrics.TotalTrades
		if tr1 != tr2 {
			t.Fatalf("run mismatch at rank %d: trades %d != %d", i, tr1, tr2)
		}
		if !reflect.DeepEqual(first[i].Params, second[i].Params) {
			t.Fatalf("run mismatch at rank %d: params %v != %v", i, first[i].Params, second[i].Params)
		}
	}
}
