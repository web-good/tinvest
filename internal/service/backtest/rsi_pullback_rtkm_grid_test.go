package backtest

import "testing"

// rtkmGridFiles перечисляет сетки RTKM ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN.
var rtkmGridFiles = []string{
	"cal_screen.json",
	"cal_entry.json",
	"cal_trend.json",
	"cal_trend_low.json",
	"cal_day.json",
	"cal_volume.json",
	"cal_vol_window.json",
	"cal_risk.json",
	"cal_exit.json",
	"cal_trail.json",
	"cal_trend_stop.json",
}

// rtkmCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast, живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const rtkmCoreEMAFast = 10

// TestRTKMGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-25-rtkm-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestRTKMGridsStayWide(t *testing.T) {
	for _, file := range rtkmGridFiles {
		grid := rsiPullbackTickerGrid(t, "rtkm", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("rtkm/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("rtkm/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("rtkm/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{rtkmCoreEMAFast}
		}
		for _, f := range fast {
			for _, s := range grid["EMASlow"] {
				if f >= s {
					t.Errorf("rtkm/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{35, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.15, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.05, 0.15, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{2.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "rtkm", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("rtkm/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}

	// Арбитр trend_stop меряет связку только внутри гейта A (потолок 0.8, §5.4 спеки).
	for _, v := range rsiPullbackTickerGrid(t, "rtkm", "cal_trend_stop.json")["StopDailyATR"] {
		if v > 0.8 {
			t.Errorf("rtkm/cal_trend_stop.json: StopDailyATR=%v выше потолка гейта A 0.8", v)
		}
	}
}
