package backtest

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

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

// RunGrid runs the engine for every combination in the grid and returns the
// results ranked by metric (best first). periodDays feeds CAGR.
func RunGrid(b Binding, grid Grid, candles []backtest.Candle, dailyCandles []backtest.Candle,
	cfg backtest.Config, metric string, periodDays float64,
) ([]CalibResult, error) {
	if err := validateMetric(metric); err != nil {
		return nil, err
	}
	combos, err := expandGrid(b.DefaultParams(), grid)
	if err != nil {
		return nil, err
	}
	results := make([]CalibResult, 0, len(combos))
	for _, params := range combos {
		res := backtest.Run(b.Build(params), candles, dailyCandles, cfg)
		m := backtest.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
		results = append(results, CalibResult{Params: params, Metrics: m})
	}
	return rankResults(results, metric), nil
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
	default: // profit_factor
		return m.ProfitFactor
	}
}

// rankResults sorts best-first: ascending for max_drawdown, descending otherwise.
func rankResults(results []CalibResult, metric string) []CalibResult {
	sort.SliceStable(results, func(i, j int) bool {
		a, b := metricValue(results[i].Metrics, metric), metricValue(results[j].Metrics, metric)
		if metric == "max_drawdown" {
			return a < b
		}
		return a > b
	})
	return results
}

var supportedMetrics = map[string]struct{}{
	"profit_factor": {}, "net_pnl": {}, "win_rate": {}, "max_drawdown": {}, "expectancy": {},
}

func validateMetric(metric string) error {
	if _, ok := supportedMetrics[metric]; !ok {
		return fmt.Errorf("backtest: unknown metric %q (want profit_factor|net_pnl|win_rate|max_drawdown|expectancy)", metric)
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
		b.WriteString("\n## Лучшая комбинация — параметры\n\n| Параметр | Значение |\n|---|---|\n")
		for _, row := range ParamRows(results[0].Params) {
			fmt.Fprintf(&b, "| %s | %s |\n", row.Name, row.Value)
		}
	}
	return b.String()
}
