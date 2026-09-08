package backtest

import "testing"

// TestMVIDGridsStayWide держит НИЖНЮЮ границу ширины сеток MVID: для каждой перечисленной пары
// (файл, поле) проверяет, что обязательные узлы из §5.1 спеки
// (docs/superpowers/specs/2026-09-07-mvid-rsi-pullback-prep-design.md) присутствуют в сетке —
// узел RSILower=5 в cal_entry.json, узел RSIUpper=20 на оси exit_low, узел VolBaseDays=30 в
// cal_volume.json, узлы горба 25/30/35/40/45 на оси EMASlow в cal_trend_hump.json, узел
// EMASlow=250 в cal_trend.json и т.д. Владелец потребовал максимально широкие оси, поэтому тест
// запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие обязательных значений и не
// запрещает лишних.
//
// Остальные инварианты каталога из §5.1 держат соседние тесты этого файла и файла
// rsi_pullback_grid_test.go: RSILower <= 50 — TestMVIDRSILowerStaysBelow50, RSIPeriod >= 2 —
// TestMVIDRSIPeriodStaysAtLeastTwo, RSIUpper > RSILower на entry_deep/exit_low —
// TestMVIDEntryGridsKeepRSIUpperAboveRSILower, EMAFast < EMASlow на обеих трендовых темах —
// TestMVIDTrendGridsKeepFastBelowSlow, StopDailyATR нигде не равен нулю —
// rsi_pullback_grid_test.go:106 (общий тест каталога, а не тест, специфичный для MVID).
func TestMVIDGridsStayWide(t *testing.T) {
	cases := []struct {
		file   string
		field  string
		values []float64
	}{
		{"cal_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8, 10, 12}},
		{"cal_entry.json", "RSILower", []float64{5, 10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_entry_deep.json", "RSIPeriod", []float64{2, 3, 4, 5, 6}},
		{"cal_entry_deep.json", "RSILower", []float64{8, 10, 12, 15, 18, 20}},
		{"cal_entry_deep.json", "RSIUpper", []float64{40, 55, 65, 70}},
		{"cal_trend.json", "EMAFast", []float64{3, 5, 8, 10, 15, 20, 30, 40}},
		{"cal_trend.json", "EMASlow", []float64{50, 75, 100, 150, 200, 250}},
		{"cal_trend_hump.json", "EMAFast", []float64{3, 5, 8, 10}},
		{"cal_trend_hump.json", "EMASlow", []float64{15, 20, 25, 30, 35, 40, 45}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5, 2.0}},
		{"cal_day_spent.json", "FreshDayATR", []float64{0}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5, 2.0}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20, 30}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24, 32}},
		{"cal_vol_window.json", "VolMult", []float64{1.2, 2.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.2, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}},
		{"cal_exit_low.json", "RSILower", []float64{15}},
		{"cal_exit_low.json", "RSIUpper", []float64{20, 25, 30, 35, 40, 45}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
		{"cal_trail.json", "UseTrail", []float64{1}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "mvid", c.file)
		got := grid[c.field]
		for _, v := range c.values {
			var found bool
			for _, g := range got {
				if g == v {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("mvid/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestMVIDRSILowerStaysBelow50 сторожит явный инвариант §5.1: RSILower <= 50 везде, где эта ось
// встречается (cal_entry.json, cal_entry_deep.json).
func TestMVIDRSILowerStaysBelow50(t *testing.T) {
	for _, file := range []string{"cal_entry.json", "cal_entry_deep.json"} {
		grid := rsiPullbackTickerGrid(t, "mvid", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("mvid/%s: RSILower=%v нарушает инвариант RSILower <= 50", file, v)
			}
		}
	}
}

// TestMVIDRSIPeriodStaysAtLeastTwo сторожит инвариант §5.1: RSIPeriod >= 2 везде, где эта ось
// встречается (cal_entry.json, cal_entry_deep.json).
func TestMVIDRSIPeriodStaysAtLeastTwo(t *testing.T) {
	for _, file := range []string{"cal_entry.json", "cal_entry_deep.json"} {
		grid := rsiPullbackTickerGrid(t, "mvid", file)
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("mvid/%s: RSIPeriod=%v нарушает инвариант RSIPeriod >= 2", file, v)
			}
		}
	}
}

// TestMVIDEntryGridsKeepRSIUpperAboveRSILower проверяет оба файла, где RSIUpper и RSILower
// меряются в одной сетке (cal_entry_deep.json и cal_exit_low.json — в последнем RSILower пинован
// на 15, ведущий кандидат оси входа, §4.1 спеки): одна лишняя точка в любой из осей может дать
// пару, где выход стоит НИЖЕ входа. Такая пара не ошибка запуска — ядро её честно посчитает и
// вернёт мусорную конфигурацию, которая войдёт в ранжирование темы.
func TestMVIDEntryGridsKeepRSIUpperAboveRSILower(t *testing.T) {
	for _, file := range []string{"cal_entry_deep.json", "cal_exit_low.json"} {
		grid := rsiPullbackTickerGrid(t, "mvid", file)
		for _, lower := range grid["RSILower"] {
			for _, upper := range grid["RSIUpper"] {
				if upper <= lower {
					t.Errorf("mvid/%s: пара RSILower=%v, RSIUpper=%v нарушает RSIUpper > RSILower", file, lower, upper)
				}
			}
		}
	}
}

// TestMVIDTrendGridsKeepFastBelowSlow проверяет ОБА файла тренда (cal_trend.json,
// cal_trend_hump.json). Разложение темы на канонический диапазон и горб существует ровно потому,
// что инвариант EMAFast < EMASlow не позволяет мерить горб 15-45 в одной сетке с EMAFast до 40;
// если кто-нибудь сольёт файлы обратно, тест это поймает.
func TestMVIDTrendGridsKeepFastBelowSlow(t *testing.T) {
	for _, file := range []string{"cal_trend.json", "cal_trend_hump.json"} {
		grid := rsiPullbackTickerGrid(t, "mvid", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("mvid/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}
}
