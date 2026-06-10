package backtest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"tinvest/internal/domain/backtest"
)

// Grid maps a Params field name to the values to sweep over it. Fields not
// listed keep their default.
type Grid map[string][]float64

// CalibResult pairs a parameter combination with its run metrics.
type CalibResult struct {
	Params  any
	Metrics backtest.Metrics
}

// defaultKeepTop is how many top-ranked parameter sets carry into the next phase
// when a phase does not set KeepTop.
const defaultKeepTop = 5

// calibWorkers bounds the goroutine pool that evaluates grid combinations in
// runCombos. Kept conservative so a calibration run does not peg the whole machine.
const calibWorkers = 4

// Phase is one stage of a staged calibration: a sub-grid plus how many top-ranked
// parameter sets survive into the next phase.
type Phase struct {
	Name    string `json:"name"`
	KeepTop int    `json:"keepTop"`
	Grid    Grid   `json:"grid"`
}

// PhasedGrid is the ordered list of calibration phases.
type PhasedGrid struct {
	Phases []Phase `json:"phases"`
}

// PhaseProgress reports one phase's outcome to a RunPhases caller (e.g. for stdout).
type PhaseProgress struct {
	Index      int     // 1-based phase index
	Name       string  // phase name (defaults to phase-<index>)
	Combos     int     // combinations run in this phase
	Kept       int     // survivors carried forward (clamped to results; full ranking on the last phase)
	BestMetric float64 // best metric value after this phase
}

// runCombos runs the engine once per parameter combination and pairs each with its
// metrics. periodDays feeds CAGR. The result is unranked but its order matches combos
// exactly (results[i] is combos[i]), so ranking downstream is deterministic.
//
// Combinations are independent — each gets a fresh strategy via b.Build, its own
// portfolio inside backtest.Run, and only reads the shared candle slices — so they run
// on a bounded pool of calibWorkers goroutines. Each result slot has a single writer
// (its own index), so no mutex is needed.
func runCombos(b Binding, combos []any, candles, dailyCandles []backtest.Candle,
	cfg backtest.Config, periodDays float64,
) []CalibResult {
	results := make([]CalibResult, len(combos))

	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := calibWorkers
	if workers > len(combos) {
		workers = len(combos)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				res := backtest.Run(b.Build(combos[i]), candles, dailyCandles, cfg)
				m := backtest.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
				results[i] = CalibResult{Params: combos[i], Metrics: m}
			}
		}()
	}
	for i := range combos {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return results
}

// RunGrid runs the engine for every combination in the grid and returns the
// results ranked by metric (best first). minTrades is the floor: combos with
// fewer trades sink below all qualified combos. periodDays feeds CAGR.
func RunGrid(b Binding, grid Grid, candles []backtest.Candle, dailyCandles []backtest.Candle,
	cfg backtest.Config, metric string, minTrades int, periodDays float64,
) ([]CalibResult, error) {
	if err := validateMetric(metric); err != nil {
		return nil, err
	}
	combos, err := expandGrid(b.DefaultParams(), grid)
	if err != nil {
		return nil, err
	}
	results := runCombos(b, combos, candles, dailyCandles, cfg, periodDays)
	return rankResults(results, metric, minTrades), nil
}

// ParsePhases decodes a calibration grid file into ordered phases. It accepts the
// phased format ({"phases":[{grid, keepTop, name}, ...]}) and the legacy flat format
// ({"Field":[...], ...}), wrapping the latter as a single phase.
func ParsePhases(raw []byte) ([]Phase, error) {
	var pg PhasedGrid
	if err := json.Unmarshal(raw, &pg); err == nil && len(pg.Phases) > 0 {
		return pg.Phases, nil
	}
	var flat Grid
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, fmt.Errorf("backtest: parse grid: %w", err)
	}
	return []Phase{{Grid: flat}}, nil
}

// RunPhases runs a staged calibration: phase k+1 sweeps its grid over the top-KeepTop
// survivors of phase k. It returns the final phase's full ranking (best first).
// onProgress, when non-nil, is called once per phase. metric and minTrades are global
// across phases; minTrades floors every phase's ranking so a low-trade fluke cannot
// survive forward.
func RunPhases(b Binding, phases []Phase, candles, dailyCandles []backtest.Candle,
	cfg backtest.Config, metric string, minTrades int, periodDays float64,
	onProgress func(PhaseProgress),
) ([]CalibResult, error) {
	if err := validateMetric(metric); err != nil {
		return nil, err
	}
	if len(phases) == 0 {
		return nil, fmt.Errorf("backtest: phased grid has no phases")
	}
	seeds := []any{b.DefaultParams()}
	var results []CalibResult
	for i, ph := range phases {
		combos := make([]any, 0, len(seeds))
		for _, seed := range seeds {
			expanded, err := expandGrid(seed, ph.Grid)
			if err != nil {
				return nil, err
			}
			combos = append(combos, expanded...)
		}
		results = rankResults(runCombos(b, combos, candles, dailyCandles, cfg, periodDays), metric, minTrades)

		keep := ph.KeepTop
		if keep <= 0 {
			keep = defaultKeepTop
		}
		if keep > len(results) {
			keep = len(results)
		}
		if onProgress != nil {
			name := ph.Name
			if name == "" {
				name = fmt.Sprintf("phase-%d", i+1)
			}
			var best float64
			if len(results) > 0 {
				best = metricValue(results[0].Metrics, metric)
			}
			onProgress(PhaseProgress{Index: i + 1, Name: name, Combos: len(combos), Kept: keep, BestMetric: best})
		}
		if i < len(phases)-1 {
			seeds = make([]any, 0, keep)
			for _, r := range results[:keep] {
				seeds = append(seeds, r.Params)
			}
		}
	}
	return results, nil
}

// expandGrid builds the cartesian product of the grid, applying each field over
// a copy of the default params.
func expandGrid(defaults any, grid Grid) ([]any, error) {
	combos := []any{defaults}
	// Stable field order for deterministic output.
	names := make([]string, 0, len(grid))
	for name := range grid {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		values := grid[name]
		var next []any
		for _, base := range combos {
			for _, v := range values {
				updated, err := applyField(base, name, v)
				if err != nil {
					return nil, err
				}
				next = append(next, updated)
			}
		}
		combos = next
	}
	return combos, nil
}

// applyField returns a copy of params with field `name` set to `value`,
// converting to the field's int or float kind. Unknown/unsettable fields error.
func applyField(params any, name string, value float64) (any, error) {
	v := reflect.ValueOf(params)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("backtest: params is not a struct")
	}
	out := reflect.New(v.Type()).Elem()
	out.Set(v)
	f := out.FieldByName(name)
	if !f.IsValid() {
		return nil, fmt.Errorf("backtest: unknown grid field %q", name)
	}
	if !f.CanSet() {
		return nil, fmt.Errorf("backtest: field %q is not settable", name)
	}
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.SetInt(int64(value))
	case reflect.Float32, reflect.Float64:
		f.SetFloat(value)
	default:
		return nil, fmt.Errorf("backtest: field %q has unsupported kind %s", name, f.Kind())
	}
	return out.Interface(), nil
}

// metricValue extracts the ranking key for a metric (already validated).
func metricValue(m backtest.Metrics, metric string) float64 {
	switch metric {
	case "net_pnl":
		return m.NetPnL
	case "win_rate":
		return m.WinRate
	case "max_drawdown":
		return m.MaxDrawdown
	case "expectancy":
		return m.Expectancy
	case "sortino":
		return m.Sortino
	default: // profit_factor
		return m.ProfitFactor
	}
}

// rankResults sorts best-first. Combos with fewer than minTrades trades are treated
// as statistically unreliable and sink below all qualified combos, regardless of
// their metric. Within each group: ascending for max_drawdown, descending otherwise.
func rankResults(results []CalibResult, metric string, minTrades int) []CalibResult {
	qualifies := func(m backtest.Metrics) bool { return m.TotalTrades >= minTrades }
	sort.SliceStable(results, func(i, j int) bool {
		qi, qj := qualifies(results[i].Metrics), qualifies(results[j].Metrics)
		if qi != qj {
			return qi // qualified ranks ahead of unqualified
		}
		a, b := metricValue(results[i].Metrics, metric), metricValue(results[j].Metrics, metric)
		if metric == "max_drawdown" {
			return a < b
		}
		return a > b
	})
	return results
}

var supportedMetrics = map[string]struct{}{
	"profit_factor": {}, "net_pnl": {}, "win_rate": {}, "max_drawdown": {}, "expectancy": {}, "sortino": {},
}

func validateMetric(metric string) error {
	if _, ok := supportedMetrics[metric]; !ok {
		return fmt.Errorf("backtest: unknown metric %q (want profit_factor|net_pnl|win_rate|max_drawdown|expectancy|sortino)", metric)
	}
	return nil
}

// RenderCalibrationMarkdown renders the top-N combinations as a Markdown table.
func RenderCalibrationMarkdown(metric string, results []CalibResult, topN int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Калибровка (ранжирование по %s)\n\n", metric)
	fmt.Fprintf(&b, "Всего комбинаций: %d. Топ-%d:\n\n", len(results), topN)
	b.WriteString("| # | Метрика | Profit factor | Net PnL | Win rate | Max DD | Сделок |\n|---|---|---|---|---|---|---|\n")
	for i, r := range results {
		if i >= topN {
			break
		}
		m := r.Metrics
		fmt.Fprintf(&b, "| %d | %.4g | %.3f | %.2f | %.2f%% | %.2f | %d |\n",
			i+1, metricValue(m, metric), m.ProfitFactor, m.NetPnL, m.WinRate*100, m.MaxDrawdown, m.TotalTrades)
	}
	if len(results) > 0 {
		b.WriteString("\n## Параметры топ-комбинаций\n")
		for i, r := range results {
			if i >= topN {
				break
			}
			m := r.Metrics
			fmt.Fprintf(&b, "\n### #%d — %s %.4g, сделок %d\n\n| Параметр | Значение |\n|---|---|\n",
				i+1, metric, metricValue(m, metric), m.TotalTrades)
			for _, row := range ParamRows(r.Params) {
				fmt.Fprintf(&b, "| %s | %s |\n", row.Name, row.Value)
			}
		}
	}
	return b.String()
}
