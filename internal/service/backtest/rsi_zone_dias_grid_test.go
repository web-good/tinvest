package backtest

import (
	"os"
	"path/filepath"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// rsiZoneDIASGridFiles перечисляет тематические сетки DIAS поимённо, а не обходом каталога: в
// том же каталоге лежат файлы-точки (baseline_point.json, point.json, plateau_*.json), которые
// законно фиксируют ось одним значением и упали бы на проверке краёв осей. Сетки второго круга
// (cal2_*) узкие по построению: для них проверяются инварианты, но не минимальные края осей.
var rsiZoneDIASGridFiles = []string{
	"cal_entry.json",
	"cal_exit.json",
	"cal_entry_exit.json",
	"cal_trend.json",
	"cal_risk.json",
	"cal_trend_risk.json",
	"cal_phased.json",
	"cal2_entry.json",
	"cal2_exit.json",
	"cal2_trend.json",
	"cal2_risk.json",
}

// rsiZoneDIASMaxEMA — потолок EMAPeriod на 24-месячном окне (история ~31.5 мес): Lookback 2·N+20 не должен
// съедать больше ~10% окна (~22 400 баров), 2·400+20 = 820 баров ≈ 3.7%.
const rsiZoneDIASMaxEMA = 400

func rsiZoneDIASPhases(t *testing.T, file string) []Phase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("../../../data/params/rsi_zone/dias", file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return phases
}

// TestRSIZoneDIASGridsStayWide держит инварианты процедуры /rsi-zone-calibrate: каждое значение
// применимо к core.Params, стоп никогда не выключен, RSIPeriod >= 2, EMA не длиннее истории,
// каждая сетка перечисляет все шесть полей (после регистрации DIAS иначе оси мерялись бы поверх
// его литерала, и повторный прогон сетки дал бы другие числа) и оси не урезаны ниже минимальных
// краёв.
func TestRSIZoneDIASGridsStayWide(t *testing.T) {
	fields := []string{"RSIPeriod", "RSILower", "RSIUpper", "EMAPeriod", "DailyATRPeriod", "StopDailyATR"}
	axes := map[string]map[string][]float64{}
	for _, file := range rsiZoneDIASGridFiles {
		axes[file] = map[string][]float64{}
		for _, ph := range rsiZoneDIASPhases(t, file) {
			for name, values := range ph.Grid {
				for _, v := range values {
					if _, err := applyField(core.DefaultParams(), name, v); err != nil {
						t.Fatalf("dias/%s phase %q: apply %s=%v: %v", file, ph.Name, name, v, err)
					}
				}
				axes[file][name] = append(axes[file][name], values...)
			}
		}
		grid := axes[file]
		for _, f := range fields {
			if len(grid[f]) == 0 {
				t.Errorf("dias/%s: поле %s не перечислено — ось мерялась бы поверх литерала DIAS", file, f)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v <= 0 {
				t.Errorf("dias/%s: StopDailyATR=%v — многодневный лонг без стопа не может быть точкой", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("dias/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["EMAPeriod"] {
			if v > rsiZoneDIASMaxEMA {
				t.Errorf("dias/%s: EMAPeriod=%v длиннее, чем позволяет история (потолок %d)", file, v, rsiZoneDIASMaxEMA)
			}
		}
	}

	entry := []float64{2, 3, 4, 5, 6, 8, 10, 14}
	lower := []float64{5, 10, 15, 20, 25, 30, 35, 40, 45}
	upper := []float64{50, 55, 60, 65, 70, 75, 80, 85, 90, 95}
	trend := []float64{10, 20, 30, 50, 75, 100, 150, 200, 300, 400}
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
	}
	for _, m := range mustHave {
		for _, w := range m.want {
			if !containsFloat(axes[m.file][m.axis], w) {
				t.Errorf("dias/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, axes[m.file][m.axis])
			}
		}
	}

	// Арбитры держат порядок фаз процедуры.
	for file, want := range map[string][]string{
		"cal_trend_risk.json": {"trend", "risk"},
		"cal_phased.json":     {"entry", "trend", "exit", "risk"},
	} {
		phases := rsiZoneDIASPhases(t, file)
		if len(phases) != len(want) {
			t.Fatalf("dias/%s: фаз %d, want %d", file, len(phases), len(want))
		}
		for i, name := range want {
			if phases[i].Name != name {
				t.Errorf("dias/%s: фаза %d = %q, want %q", file, i, phases[i].Name, name)
			}
		}
	}
}
