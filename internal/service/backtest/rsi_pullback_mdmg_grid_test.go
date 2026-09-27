package backtest

import (
	"strings"
	"testing"
)

// mdmgGridFiles перечисляет сетки MDMG ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN, RTKM, SVCB, RAGR.
var mdmgGridFiles = []string{
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
	"cal_trend_spent.json",
	"cal_trend_day.json",
}

// mdmgRound2GridFiles перечисляет узкие сетки второго круга MDMG (Task 12, §5.14 спеки) поимённо, по
// той же причине, что и mdmgGridFiles.
var mdmgRound2GridFiles = []string{
	"cal2_trend_spent.json",
	"cal2_trend_day.json",
	"cal2_exit.json",
	"cal2_risk.json",
	"cal2_trail.json",
}

// mdmgAllGridFiles склеивает сетки обоих кругов: жёсткие инварианты §5.1 держатся на обоих.
func mdmgAllGridFiles() []string {
	out := make([]string, 0, len(mdmgGridFiles)+len(mdmgRound2GridFiles))
	out = append(out, mdmgGridFiles...)
	return append(out, mdmgRound2GridFiles...)
}

// mdmgCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast (trend_day), живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const mdmgCoreEMAFast = 10

// mdmgCoreEMASlow — дефолт ядра core.DefaultParams().EMASlow. Сетка, свипующая EMAFast без
// EMASlow (trend_spent), живёт на этом значении.
const mdmgCoreEMASlow = 100

// mdmgStopCeiling — потолок риск-гейта A (§5.4 спеки). Правила каталога на MDMG расходятся:
// выживаемость 30% даёт 1.0, частота срабатывания не реже 10% сделок — 0.8; берётся строжайшее.
const mdmgStopCeiling = 0.8

// TestMDMGGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestMDMGGridsStayWide(t *testing.T) {
	for _, file := range mdmgAllGridFiles() {
		grid := rsiPullbackTickerGrid(t, "mdmg", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("mdmg/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("mdmg/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["EMAFast"] {
			if v < 1 {
				t.Errorf("mdmg/%s: EMAFast=%v нарушает EMAFast >= 1", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("mdmg/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{mdmgCoreEMAFast}
		}
		slow := grid["EMASlow"]
		if len(slow) == 0 {
			slow = []float64{mdmgCoreEMASlow}
		}
		for _, f := range fast {
			for _, s := range slow {
				if f >= s {
					t.Errorf("mdmg/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
		// Стоп меряет только тема risk (§5.1 спеки): арбитры первого круга MDMG — связки тренда и
		// дневного гейта, а не связки со стопом.
		if !strings.HasPrefix(file, "cal2_") && file != "cal_risk.json" && len(grid["StopDailyATR"]) > 0 {
			t.Errorf("mdmg/%s: свипует StopDailyATR %v — стоп меряет только cal_risk.json", file, grid["StopDailyATR"])
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSIPeriod", []float64{2, 14}},
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMAFast", []float64{1}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{30, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.1, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0.05, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 2.5}},
		{"cal_volume.json", "VolMult", []float64{4.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.2, 1.5}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "mdmg", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("mdmg/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}

	// Потолок гейта A во втором круге не двигается (§5.14 спеки).
	for _, file := range mdmgRound2GridFiles {
		for _, v := range rsiPullbackTickerGrid(t, "mdmg", file)["StopDailyATR"] {
			if v > mdmgStopCeiling {
				t.Errorf("mdmg/%s: StopDailyATR=%v выше потолка гейта A %v", file, v, mdmgStopCeiling)
			}
		}
	}
}

// TestMDMGRound2FieldsHaveOneSource держит правило §5.14 спеки: во втором круге каждое поле
// свипует ровно одна тема, иначе две темы проголосуют за одно поле по-разному.
func TestMDMGRound2FieldsHaveOneSource(t *testing.T) {
	owner := map[string]string{}
	for _, file := range mdmgRound2GridFiles {
		for field, values := range rsiPullbackTickerGrid(t, "mdmg", file) {
			if len(values) < 2 {
				continue // зафиксированное поле не голосует
			}
			if prev, ok := owner[field]; ok {
				t.Errorf("mdmg: поле %s свипуют две темы второго круга: %s и %s", field, prev, file)
			}
			owner[field] = file
		}
	}
}
