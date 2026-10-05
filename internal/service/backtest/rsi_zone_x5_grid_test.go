package backtest

import (
	"os"
	"path/filepath"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// rsiZoneX5GridFiles перечисляет тематические сетки X5 поимённо, а не обходом каталога: в
// том же каталоге лежат файлы-точки (baseline_point.json, point.json, plateau_*.json), которые
// законно фиксируют ось одним значением и упали бы на проверке краёв осей. Сетки второго круга
// (cal2_*) узкие по построению: для них проверяются инварианты, но не
// минимальные края осей.
var rsiZoneX5GridFiles = []string{
	"cal_entry.json",
	"cal_exit.json",
	"cal_entry_exit.json",
	"cal_trend.json",
	"cal_risk.json",
	"cal_trend_risk.json",
	"cal_phased.json",
	"cal_stoch.json",
	"cal_stuck.json",
	"cal_profit.json",
}

// rsiZoneX5MaxEMA — потолок EMAPeriod на 20-месячном окне (20/8/3, история X5 однородна только
// после редомициляции 2025-01-09): Lookback 2·N+20 не должен съедать больше ~10% окна (~22 850
// баров), 2·600+20 = 1220 баров ≈ 5.3%.
const rsiZoneX5MaxEMA = 600

func rsiZoneX5Phases(t *testing.T, file string) []Phase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("../../../data/params/rsi_zone/x5", file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return phases
}

// TestRSIZoneX5GridsStayWide держит инварианты процедуры /rsi-zone-calibrate: каждое значение
// применимо к core.Params, стоп никогда не выключен, RSIPeriod >= 2, EMA не длиннее истории,
// каждая сетка перечисляет все шесть полей (после регистрации X5 иначе оси мерялись бы поверх
// его литерала, и повторный прогон сетки дал бы другие числа) и оси не урезаны ниже минимальных
// краёв.
func TestRSIZoneX5GridsStayWide(t *testing.T) {
	fields := []string{"RSIPeriod", "RSILower", "RSIUpper", "EMAPeriod", "DailyATRPeriod", "StopDailyATR"}
	axes := map[string]map[string][]float64{}
	for _, file := range rsiZoneX5GridFiles {
		axes[file] = map[string][]float64{}
		for _, ph := range rsiZoneX5Phases(t, file) {
			for name, values := range ph.Grid {
				for _, v := range values {
					if _, err := applyField(core.DefaultParams(), name, v); err != nil {
						t.Fatalf("x5/%s phase %q: apply %s=%v: %v", file, ph.Name, name, v, err)
					}
				}
				axes[file][name] = append(axes[file][name], values...)
			}
		}
		grid := axes[file]
		for _, f := range fields {
			if len(grid[f]) == 0 {
				t.Errorf("x5/%s: поле %s не перечислено — ось мерялась бы поверх литерала X5", file, f)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v <= 0 {
				t.Errorf("x5/%s: StopDailyATR=%v — многодневный лонг без стопа не может быть точкой", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("x5/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["EMAPeriod"] {
			if v > rsiZoneX5MaxEMA {
				t.Errorf("x5/%s: EMAPeriod=%v длиннее, чем позволяет история (потолок %d)", file, v, rsiZoneX5MaxEMA)
			}
		}
	}

	entry := []float64{2, 3, 4, 5, 6, 8, 10, 14}
	lower := []float64{5, 10, 15, 20, 25, 30, 35, 40, 45}
	upper := []float64{50, 55, 60, 65, 70, 75, 80, 85, 90, 95}
	trend := []float64{10, 20, 30, 50, 75, 100, 150, 200, 300, 400, 600}
	stop := []float64{0.3, 0.5, 0.7, 1.0, 1.25, 1.5, 2.0, 2.5, 3.0}
	atr := []float64{5, 7, 10, 14, 21, 30}
	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSIPeriod", entry},
		{"cal_entry.json", "RSILower", lower},
		{"cal_exit.json", "RSIUpper", upper},
		{"cal_entry_exit.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8}},
		{"cal_entry_exit.json", "RSILower", []float64{10, 15, 20, 25, 30, 35}},
		{"cal_entry_exit.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85, 90}},
		{"cal_trend.json", "EMAPeriod", trend},
		{"cal_risk.json", "StopDailyATR", stop},
		{"cal_risk.json", "DailyATRPeriod", atr},
		{"cal_trend_risk.json", "EMAPeriod", trend},
		{"cal_trend_risk.json", "StopDailyATR", stop},
		{"cal_trend_risk.json", "DailyATRPeriod", atr},
		{"cal_phased.json", "RSIPeriod", entry},
		{"cal_phased.json", "RSILower", lower},
		{"cal_phased.json", "EMAPeriod", trend},
		{"cal_phased.json", "RSIUpper", upper},
		{"cal_phased.json", "StopDailyATR", stop},
		{"cal_phased.json", "DailyATRPeriod", atr},
		{"cal_stoch.json", "UseStoch", []float64{1}},
		{"cal_stoch.json", "StochKPeriod", []float64{5, 9, 14}},
		{"cal_stoch.json", "StochDSmooth", []float64{1, 3}},
		{"cal_stoch.json", "StochLower", []float64{10, 15, 20, 25, 30}},
		{"cal_stoch.json", "ZoneWindowBars", []float64{1, 2, 3, 5, 8}},
		{"cal_stuck.json", "StuckExitBars", []float64{0, 1, 2, 3, 4, 5, 6, 8, 10}},
		{"cal_profit.json", "ProfitExitBars", []float64{0, 1, 2, 3, 4, 6, 8, 12}},
		{"cal_profit.json", "ProfitExitPct", []float64{0, 0.3, 0.5, 0.75, 1.0, 1.5, 2.0}},
	}
	for _, m := range mustHave {
		for _, w := range m.want {
			if !containsFloat(axes[m.file][m.axis], w) {
				t.Errorf("x5/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, axes[m.file][m.axis])
			}
		}
	}

	// Арбитры держат порядок фаз процедуры.
	for file, want := range map[string][]string{
		"cal_trend_risk.json": {"trend", "risk"},
		"cal_phased.json":     {"entry", "trend", "exit", "risk"},
	} {
		phases := rsiZoneX5Phases(t, file)
		if len(phases) != len(want) {
			t.Fatalf("x5/%s: фаз %d, want %d", file, len(phases), len(want))
		}
		for i, name := range want {
			if phases[i].Name != name {
				t.Errorf("x5/%s: фаза %d = %q, want %q", file, i, phases[i].Name, name)
			}
		}
	}
}
