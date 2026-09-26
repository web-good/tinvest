package backtest

import (
	"strings"
	"testing"
)

// ragrGridFiles перечисляет сетки RAGR ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN, RTKM, SVCB.
var ragrGridFiles = []string{
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
	"cal_exit_target.json",
	"cal_trend_day.json",
}

// ragrRound2GridFiles перечисляет узкие сетки второго круга RAGR поимённо (Task 12, §5.14 спеки).
var ragrRound2GridFiles = []string{
	"cal2_exit_target.json",
	"cal2_day.json",
	"cal2_vol.json",
	"cal2_trail.json",
}

// ragrAllGridFiles склеивает сетки обоих кругов: жёсткие инварианты §5.1 держатся на обоих.
func ragrAllGridFiles() []string {
	out := make([]string, 0, len(ragrGridFiles)+len(ragrRound2GridFiles))
	out = append(out, ragrGridFiles...)
	return append(out, ragrRound2GridFiles...)
}

// ragrCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast (trend_day), живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const ragrCoreEMAFast = 10

// ragrStopCeiling — потолок риск-гейта A (§5.4 спеки). Правила каталога на RAGR расходятся:
// выживаемость 30% даёт 1.0, частота срабатывания не реже 10% сделок — 0.8; берётся строжайшее.
const ragrStopCeiling = 0.8

// TestRAGRGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-26-ragr-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestRAGRGridsStayWide(t *testing.T) {
	for _, file := range ragrAllGridFiles() {
		grid := rsiPullbackTickerGrid(t, "ragr", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("ragr/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("ragr/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("ragr/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{ragrCoreEMAFast}
		}
		for _, f := range fast {
			for _, s := range grid["EMASlow"] {
				if f >= s {
					t.Errorf("ragr/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
		// Стоп меряет только тема risk (§5.1 спеки): арбитры первого круга RAGR — связки выхода и
		// тренда, а не связки со стопом.
		if !strings.HasPrefix(file, "cal2_") && file != "cal_risk.json" && len(grid["StopDailyATR"]) > 0 {
			t.Errorf("ragr/%s: свипует StopDailyATR %v — стоп меряет только cal_risk.json", file, grid["StopDailyATR"])
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSIPeriod", []float64{2, 14}},
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMAFast", []float64{2}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{30, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.1, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0.05, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 2.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.2, 1.5}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "ragr", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("ragr/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}
	// Потолок гейта A во втором круге не двигается (§5.14 спеки).
	for _, file := range ragrRound2GridFiles {
		for _, v := range rsiPullbackTickerGrid(t, "ragr", file)["StopDailyATR"] {
			if v > ragrStopCeiling {
				t.Errorf("ragr/%s: StopDailyATR=%v выше потолка гейта A %v", file, v, ragrStopCeiling)
			}
		}
	}
}

// TestRAGRRound2FieldsHaveOneSource держит правило §5.14 спеки: во втором круге каждое поле
// свипует ровно одна тема, иначе две темы проголосуют за одно поле по-разному.
func TestRAGRRound2FieldsHaveOneSource(t *testing.T) {
	owner := map[string]string{}
	for _, file := range ragrRound2GridFiles {
		for field, values := range rsiPullbackTickerGrid(t, "ragr", file) {
			if len(values) < 2 {
				continue // зафиксированное поле не голосует
			}
			if prev, ok := owner[field]; ok {
				t.Errorf("ragr: поле %s свипуют две темы второго круга: %s и %s", field, prev, file)
			}
			owner[field] = file
		}
	}
}
