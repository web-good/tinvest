package backtest

import "testing"

// svcbGridFiles перечисляет сетки SVCB ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN, RTKM.
var svcbGridFiles = []string{
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
	"cal_entry_stop.json",
	"cal_day_stop.json",
}

// svcbCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast, живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const svcbCoreEMAFast = 10

// svcbStopCeiling — потолок риск-гейта A (§5.4 спеки): оба правила каталога, выживаемость 30% и
// частота срабатывания не реже 10% сделок, дают на SVCB одно значение.
const svcbStopCeiling = 1.0

// TestSVCBGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-25-svcb-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestSVCBGridsStayWide(t *testing.T) {
	for _, file := range svcbGridFiles {
		grid := rsiPullbackTickerGrid(t, "svcb", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("svcb/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("svcb/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("svcb/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{svcbCoreEMAFast}
		}
		for _, f := range fast {
			for _, s := range grid["EMASlow"] {
				if f >= s {
					t.Errorf("svcb/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSIPeriod", []float64{2, 14}},
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{35, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.1, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.05, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 2.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "svcb", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("svcb/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}

	// Арбитры меряют связку «ось × стоп» только внутри гейта A (§5.4 спеки).
	for _, file := range []string{"cal_entry_stop.json", "cal_day_stop.json"} {
		for _, v := range rsiPullbackTickerGrid(t, "svcb", file)["StopDailyATR"] {
			if v > svcbStopCeiling {
				t.Errorf("svcb/%s: StopDailyATR=%v выше потолка гейта A %v", file, v, svcbStopCeiling)
			}
		}
	}
}
