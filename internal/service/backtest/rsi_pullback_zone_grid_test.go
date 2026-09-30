package backtest

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// rsiPullbackZoneFieldsRequired — поля, без которых zone-вход молча выключен: ноль у любого из
// трёх числовых полей отключает его даже при включённом UseZoneEntry.
var rsiPullbackZoneFieldsRequired = []string{"ZoneRSIPeriod", "ZoneRSILower", "ZoneEMAPeriod"}

// rsiPullbackZoneFileName reports whether a file NAME claims to belong to the zone calibration
// (docs/rsi_pullback/strategy.md §8.2): такой файл обязан задавать режим 1 или 2.
func rsiPullbackZoneFileName(name string) bool {
	for _, prefix := range []string{"cal_zone_", "cal2_zone_", "plateau_zone_", "plateau_entry_zone", "r2_point_zone", "r2_probe_zone"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// rsiPullbackModeBySuffix — режим, который обязаны задавать plateau_{base,mode,final}_<m>.
var rsiPullbackModeBySuffix = map[string]float64{"pullback": 0, "zone": core.ZoneEntryOnly, "both": core.ZoneEntryAlso}

// rsiPullbackModeFromName возвращает режим, жёстко заданный именем plateau_{base,mode,final}_<m>.
// known=false — имя не из этого семейства; bad — суффикс вне pullback/zone/both.
// suffix — часть имени после префикса без «.json».
func rsiPullbackModeFromName(name string) (mode float64, suffix string, known bool, bad bool) {
	for _, prefix := range []string{"plateau_base_", "plateau_mode_", "plateau_final_"} {
		if strings.HasPrefix(name, prefix) {
			suffix = strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".json")
			m, ok := rsiPullbackModeBySuffix[suffix]
			return m, suffix, true, !ok
		}
	}
	return 0, "", false, false
}

// rsiPullbackZoneFileViolations — сторож режима файла для обеих процедур калибровки (zone-only и
// единой, где режим выбирается по файлу). У незарегистрированного тикера и у литерала, равного
// дефолтам ядра, UseZoneEntry=0 и zone-поля нулевые, поэтому сетка, забывшая UseZoneEntry, молча
// считает pullback, а забывшая zone-поле при режиме 1/2 — молча даёт ноль сделок. Правила:
//  1. если хоть одна фаза задаёт UseZoneEntry, его задаёт каждая фаза ровно одним значением,
//     одинаковым во всех фазах файла;
//  2. значение из {0, 1, 2}; при 1 и 2 три zone-поля заданы и строго > 0, при 0 не требуются;
//  3. zone-имена (cal_zone_, cal2_zone_, plateau_zone_, plateau_entry_zone, r2_point_zone,
//     r2_probe_zone) обязаны задавать режим 1 или 2;
//  4. plateau_base_/mode_/final_<m> обязаны задавать режим 0/2/1 для <m> pullback/zone/both,
//     иной суффикс — нарушение;
//  5. plateau_entry_pullback — режим не задан или 0;
//  6. cal_out_*, cal2_out_* обязаны задавать режим (и вне каталога единой процедуры);
//  7. при unified (в каталоге лежит plateau_base_pullback.json) каждая фаза каждого файла
//     задаёт UseZoneEntry явно (0 допустим).
//
// Требование на фазу, а не на файл, — чтобы не зависеть от того, как калибратор переносит
// значения между фазами.
func rsiPullbackZoneFileViolations(name string, unified bool, phases []Phase) []string {
	var out []string
	armed := false
	for _, ph := range phases {
		if _, ok := ph.Grid["UseZoneEntry"]; ok {
			armed = true
			break
		}
	}
	var mode float64
	modeSet := false
	if armed {
		for _, ph := range phases {
			modes := ph.Grid["UseZoneEntry"]
			if len(modes) != 1 {
				out = append(out, fmt.Sprintf("%s: фаза %q: UseZoneEntry должен быть ровно одним значением, задано %v", name, ph.Name, modes))
				continue
			}
			v := modes[0]
			if v != 0 && v != core.ZoneEntryAlso && v != core.ZoneEntryOnly {
				out = append(out, fmt.Sprintf("%s: фаза %q: UseZoneEntry=%v, допустимы только 0, 1 и 2", name, ph.Name, v))
				continue
			}
			if !modeSet {
				mode, modeSet = v, true
			} else if v != mode {
				out = append(out, fmt.Sprintf("%s: фаза %q: UseZoneEntry=%v отличается от %v в других фазах", name, ph.Name, v, mode))
			}
			if v == 0 {
				continue
			}
			for _, field := range rsiPullbackZoneFieldsRequired {
				values, ok := ph.Grid[field]
				if !ok || len(values) == 0 {
					out = append(out, fmt.Sprintf("%s: фаза %q не задаёт %s — zone-вход молча выключен", name, ph.Name, field))
					continue
				}
				for _, fv := range values {
					if fv <= 0 {
						out = append(out, fmt.Sprintf("%s: фаза %q: %s=%v выключает zone-вход", name, ph.Name, field, fv))
					}
				}
			}
		}
	}
	required := false
	if rsiPullbackZoneFileName(name) {
		required = true
		if !armed {
			out = append(out, fmt.Sprintf("%s: имя zone-файла, но ни одна фаза не задаёт UseZoneEntry — прогон посчитает pullback", name))
		} else if modeSet && mode == 0 {
			out = append(out, fmt.Sprintf("%s: имя zone-файла, но UseZoneEntry=0", name))
		}
	}
	if want, suffix, known, bad := rsiPullbackModeFromName(name); known {
		required = true
		switch {
		case bad:
			out = append(out, fmt.Sprintf("%s: суффикс режима %q не из pullback/zone/both", name, suffix))
		case !armed:
			out = append(out, fmt.Sprintf("%s: режим обязателен (ожидается UseZoneEntry=%v)", name, want))
		case modeSet && mode != want:
			out = append(out, fmt.Sprintf("%s: UseZoneEntry=%v, имя требует %v", name, mode, want))
		}
	}
	if strings.HasPrefix(name, "plateau_entry_pullback") && armed && modeSet && mode != 0 {
		out = append(out, fmt.Sprintf("%s: pullback-вход, а UseZoneEntry=%v", name, mode))
	}
	if !required && !armed {
		switch {
		case strings.HasPrefix(name, "cal_out_") || strings.HasPrefix(name, "cal2_out_"):
			out = append(out, fmt.Sprintf("%s: режим обязателен, но ни одна фаза не задаёт UseZoneEntry", name))
		case unified:
			out = append(out, fmt.Sprintf("%s: файл в каталоге единой процедуры, а ни одна фаза не задаёт UseZoneEntry явно (0 допустим)", name))
		}
	}
	return out
}

// zoneArmedPhase — фаза, задающая якорь zone-only процедуры целиком; extra дописывает или
// перекрывает оси, drop убирает ключи.
func zoneArmedPhase(name string, extra Grid, drop ...string) Phase {
	g := Grid{
		"UseZoneEntry":  {core.ZoneEntryOnly},
		"ZoneRSIPeriod": {4},
		"ZoneRSILower":  {25},
		"ZoneEMAPeriod": {200},
	}
	for k, v := range extra {
		g[k] = v
	}
	for _, k := range drop {
		delete(g, k)
	}
	return Phase{Name: name, Grid: g}
}

func TestRSIPullbackZoneFileViolations(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		unified bool
		phases  []Phase
		bad     bool
	}{
		{"обычный pullback-файл", "cal_exit.json", false,
			[]Phase{{Name: "exit", Grid: Grid{"RSIUpper": {55, 70, 85}}}}, false},
		{"zone-only якорь", "cal_zone_exit.json", false,
			[]Phase{zoneArmedPhase("exit", Grid{"RSIUpper": {55, 70}})}, false},
		{"режим 1 тоже допустим", "cal_zone_entry.json", false,
			[]Phase{zoneArmedPhase("entry", Grid{"UseZoneEntry": {core.ZoneEntryAlso}})}, false},
		{"точка якоря", "plateau_zone_anchor.json", false,
			[]Phase{zoneArmedPhase("point", Grid{"StopDailyATR": {0.5}})}, false},
		{"забыт ZoneRSIPeriod", "cal_zone_risk.json", false,
			[]Phase{zoneArmedPhase("risk", nil, "ZoneRSIPeriod")}, true},
		{"забыт ZoneRSILower", "cal_zone_risk.json", false,
			[]Phase{zoneArmedPhase("risk", nil, "ZoneRSILower")}, true},
		{"забыт ZoneEMAPeriod", "cal_zone_risk.json", false,
			[]Phase{zoneArmedPhase("risk", nil, "ZoneEMAPeriod")}, true},
		{"нулевое zone-поле в оси", "cal_zone_trend.json", false,
			[]Phase{zoneArmedPhase("trend", Grid{"ZoneEMAPeriod": {0, 50, 200}})}, true},
		{"UseZoneEntry свипает выкл", "cal_zone_entry.json", false,
			[]Phase{zoneArmedPhase("entry", Grid{"UseZoneEntry": {0, 2}})}, true},
		{"UseZoneEntry дробный", "cal_zone_entry.json", false,
			[]Phase{zoneArmedPhase("entry", Grid{"UseZoneEntry": {2.5}})}, true},
		{"вторая фаза без якоря", "cal_zone_phased.json", false,
			[]Phase{zoneArmedPhase("entry", nil), {Name: "exit", Grid: Grid{"RSIUpper": {70}}}}, true},
		{"zone-имя без UseZoneEntry", "cal_zone_exit.json", false,
			[]Phase{{Name: "exit", Grid: Grid{"RSIUpper": {70}}}}, true},
		{"точка второго круга без UseZoneEntry", "r2_point_zone.json", false,
			[]Phase{{Name: "point", Grid: Grid{"RSIUpper": {70}}}}, true},
		{"явный режим 0 допустим", "cal_out_risk.json", false,
			[]Phase{{Name: "risk", Grid: Grid{"UseZoneEntry": {0}, "StopDailyATR": {0.5, 0.7}}}}, false},
		{"cal_out без режима", "cal_out_risk.json", false,
			[]Phase{{Name: "risk", Grid: Grid{"StopDailyATR": {0.5}}}}, true},
		{"разные режимы по фазам", "cal_out_exit.json", false,
			[]Phase{zoneArmedPhase("a", nil), zoneArmedPhase("b", Grid{"UseZoneEntry": {core.ZoneEntryAlso}})}, true},
		{"режим 3", "cal_out_exit.json", false,
			[]Phase{zoneArmedPhase("exit", Grid{"UseZoneEntry": {3}})}, true},
		{"mode_both = 1", "plateau_mode_both.json", false,
			[]Phase{zoneArmedPhase("point", Grid{"UseZoneEntry": {core.ZoneEntryAlso}})}, false},
		{"mode_both с режимом 2", "plateau_mode_both.json", false,
			[]Phase{zoneArmedPhase("point", nil)}, true},
		{"final_pullback = 0", "plateau_final_pullback.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, false},
		{"base_zone с режимом 0", "plateau_base_zone.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, true},
		{"неизвестный суффикс режима", "plateau_mode_mixed.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, true},
		{"entry_pullback с режимом 2", "plateau_entry_pullback.json", false,
			[]Phase{zoneArmedPhase("point", nil)}, true},
		{"entry_zone с режимом 0", "plateau_entry_zone.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, true},
		{"точка старой калибровки без режима", "plateau_point.json", false,
			[]Phase{{Name: "point", Grid: Grid{"RSIUpper": {70}}}}, false},
		{"точка единой процедуры без режима", "plateau_point.json", true,
			[]Phase{{Name: "point", Grid: Grid{"RSIUpper": {70}}}}, true},
		{"единая: cal2_entry без режима", "cal2_entry.json", true,
			[]Phase{{Name: "entry", Grid: Grid{"RSILower": {20, 25}}}}, true},
		{"единая: plateau_RSIUpper_70 без режима", "plateau_RSIUpper_70.json", true,
			[]Phase{{Name: "p", Grid: Grid{"RSIUpper": {70}}}}, true},
		{"единая: cal_entry с режимом 0", "cal_entry.json", true,
			[]Phase{{Name: "entry", Grid: Grid{"UseZoneEntry": {0}, "RSILower": {20, 25}}}}, false},
		{"не единая: cal_entry без режима", "cal_entry.json", false,
			[]Phase{{Name: "entry", Grid: Grid{"RSILower": {20, 25}}}}, false},
		{"plateau_entry_pullback без режима", "plateau_entry_pullback.json", false,
			[]Phase{{Name: "point", Grid: Grid{"RSILower": {25}}}}, false},
		{"plateau_entry_pullback с режимом 0", "plateau_entry_pullback.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := rsiPullbackZoneFileViolations(c.file, c.unified, c.phases)
			if c.bad && len(got) == 0 {
				t.Fatalf("%s: нарушение не найдено", c.file)
			}
			if !c.bad && len(got) != 0 {
				t.Fatalf("%s: ложное нарушение: %v", c.file, got)
			}
		})
	}
}

func TestRSIPullbackZoneFileViolationsUnifiedMessage(t *testing.T) {
	got := rsiPullbackZoneFileViolations("cal2_entry.json", true, []Phase{{Name: "entry", Grid: Grid{"RSILower": {25}}}})
	if len(got) != 1 || !strings.Contains(got[0], "единой процедуры") {
		t.Fatalf("сообщение должно называть каталог единой процедуры: %v", got)
	}
}

func TestRSIPullbackZoneFileViolationsUnknownSuffixMessage(t *testing.T) {
	got := rsiPullbackZoneFileViolations("plateau_mode_mixed.json", false, []Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}})
	if len(got) != 1 || !strings.Contains(got[0], `"mixed"`) || strings.Contains(got[0], "plateau_mode_mixed\"") {
		t.Fatalf("сообщение должно печатать один суффикс: %v", got)
	}
}

// TestRSIPullbackZoneFilesPinZone применяет проверку ко всем сеткам каталога. Пока zone-сеток нет,
// тест проходит на pullback-файлах — и этим же сторожит, что проверка их не задевает.
func TestRSIPullbackZoneFilesPinZone(t *testing.T) {
	for _, path := range rsiPullbackGridFiles(t) {
		name := filepath.Base(path)
		_, err := os.Stat(filepath.Join(filepath.Dir(path), "plateau_base_pullback.json"))
		for _, v := range rsiPullbackZoneFileViolations(name, err == nil, rsiPullbackPhases(t, path)) {
			t.Errorf("%s/%s", filepath.Base(filepath.Dir(path)), v)
		}
	}
}

// rsiPullbackUnifiedMinAxes — минимальные оси тем единой процедуры (спека 2026-09-30, §4.6):
// шире можно, уже нельзя. Ключ — имя файла темы.
var rsiPullbackUnifiedMinAxes = map[string]map[string][]float64{
	"cal_screen.json":          {"UseDayATRGate": {0, 1}, "UseVolume": {0, 1}},
	"cal_entry.json":           {"RSIPeriod": {2, 3, 4, 5, 6, 7, 8, 10, 12, 14}, "RSILower": {5, 10, 15, 20, 25, 30, 35, 40, 45, 50}},
	"cal_trend.json":           {"EMAFast": {3, 5, 8, 10, 15, 20, 30, 40}, "EMASlow": {50, 75, 100, 150, 200, 250}},
	"cal_trend_low.json":       {"EMAFast": {1, 2, 3, 5, 8, 10}, "EMASlow": {12, 15, 20, 25, 30, 35, 40, 45}},
	"cal_day.json":             {"FreshDayATR": {0, 0.05, 0.1, 0.15, 0.2, 0.3, 0.4, 0.5}, "SpentDayATR": {0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5, 2.0, 2.5}},
	"cal_volume.json":          {"UseVolume": {1}, "VolMult": {1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0}, "VolBaseDays": {3, 5, 10, 14, 20, 30}},
	"cal_vol_window.json":      {"UseVolume": {1}, "VolLookbackBars": {1, 2, 3, 5, 8, 12, 16, 24, 32}, "VolMult": {1.0, 1.2, 2.0, 3.0}},
	"cal_trend_spent.json":     {"EMAFast": {1, 2, 3, 5, 10}, "SpentDayATR": {0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5}},
	"cal_trend_day.json":       {"EMASlow": {15, 20, 25, 30, 40, 50, 75, 100}, "FreshDayATR": {0, 0.05, 0.1, 0.15, 0.2}},
	"cal_zone_zone_entry.json": {"ZoneRSIPeriod": {2, 3, 4, 5, 6, 8, 10, 14}, "ZoneRSILower": {5, 10, 15, 20, 25, 30, 35, 40, 45}},
	"cal_zone_zone_trend.json": {"ZoneEMAPeriod": {20, 30, 50, 75, 100, 150, 200, 250, 300, 400}},
	"cal_out_exit.json":        {"RSIUpper": {30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}},
	"cal_out_risk.json":        {"StopDailyATR": {0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.2, 1.5, 2.0}, "TPDailyATR": {0.1, 0.15, 0.2, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.5}},
	"cal_out_trail.json":       {"UseTrail": {0, 1}, "TrailDailyATR": {0.2, 0.3, 0.4, 0.5, 0.7, 1.0, 1.5}},
}

// rsiPullbackExitZoneOnlyRSIPeriod — в режиме 2 RSIPeriod только выходной и свипается темой exit.
var rsiPullbackExitZoneOnlyRSIPeriod = []float64{2, 3, 4, 5, 6, 8, 10, 14}

// rsiPullbackUnifiedAxisViolations проверяет, что оси темы единой процедуры не уже минимальных.
// Файл вне таблицы не проверяется.
func rsiPullbackUnifiedAxisViolations(name string, phases []Phase) []string {
	axes, ok := rsiPullbackUnifiedMinAxes[name]
	if !ok {
		return nil
	}
	union := map[string][]float64{}
	var mode float64
	for _, ph := range phases {
		for field, values := range ph.Grid {
			union[field] = append(union[field], values...)
		}
		if m := ph.Grid["UseZoneEntry"]; len(m) == 1 {
			mode = m[0]
		}
	}
	required := map[string][]float64{}
	for field, nodes := range axes {
		required[field] = nodes
	}
	if name == "cal_out_exit.json" && mode == core.ZoneEntryOnly {
		required["RSIPeriod"] = rsiPullbackExitZoneOnlyRSIPeriod
	}
	fields := make([]string, 0, len(required))
	for field := range required {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	var out []string
	for _, field := range fields {
		for _, node := range required[field] {
			found := false
			for _, v := range union[field] {
				if math.Abs(v-node) < 1e-9 {
					found = true
					break
				}
			}
			if !found {
				out = append(out, fmt.Sprintf("%s: ось %s без узла %v — сетка уже минимальной (спека §4.6)", name, field, node))
			}
		}
	}
	return out
}

func TestRSIPullbackUnifiedAxisViolations(t *testing.T) {
	wide := Grid{"StopDailyATR": {0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.2, 1.5, 2.0},
		"TPDailyATR": {0.1, 0.15, 0.2, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.5}, "UseZoneEntry": {0}}
	if got := rsiPullbackUnifiedAxisViolations("cal_out_risk.json", []Phase{{Name: "risk", Grid: wide}}); len(got) != 0 {
		t.Fatalf("широкая сетка: ложное нарушение %v", got)
	}
	narrow := Grid{"StopDailyATR": {0.5, 0.7}, "TPDailyATR": {0.6}, "UseZoneEntry": {0}}
	if got := rsiPullbackUnifiedAxisViolations("cal_out_risk.json", []Phase{{Name: "risk", Grid: narrow}}); len(got) == 0 {
		t.Fatal("узкая сетка не поймана")
	}
	exitMode0 := Grid{"RSIUpper": {30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}, "UseZoneEntry": {0}}
	if got := rsiPullbackUnifiedAxisViolations("cal_out_exit.json", []Phase{{Name: "exit", Grid: exitMode0}}); len(got) != 0 {
		t.Fatalf("exit режима 0 не обязан свипать RSIPeriod: %v", got)
	}
	exitMode2 := zoneArmedPhase("exit", Grid{"RSIUpper": exitMode0["RSIUpper"]})
	if got := rsiPullbackUnifiedAxisViolations("cal_out_exit.json", []Phase{exitMode2}); len(got) == 0 {
		t.Fatal("exit режима 2 без оси RSIPeriod не пойман")
	}
	exit2Full := zoneArmedPhase("exit", Grid{"RSIUpper": exitMode0["RSIUpper"], "RSIPeriod": rsiPullbackExitZoneOnlyRSIPeriod})
	if got := rsiPullbackUnifiedAxisViolations("cal_out_exit.json", []Phase{exit2Full}); len(got) != 0 {
		t.Fatalf("exit режима 2 с полными осями: ложное нарушение %v", got)
	}
	exitMode1 := zoneArmedPhase("exit", Grid{"RSIUpper": exitMode0["RSIUpper"], "UseZoneEntry": {core.ZoneEntryAlso}})
	if got := rsiPullbackUnifiedAxisViolations("cal_out_exit.json", []Phase{exitMode1}); len(got) != 0 {
		t.Fatalf("exit режима 1 не обязан свипать RSIPeriod: %v", got)
	}
	if got := rsiPullbackUnifiedAxisViolations("plateau_point.json", []Phase{{Name: "p", Grid: Grid{"RSIUpper": {70}}}}); len(got) != 0 {
		t.Fatalf("файл вне таблицы не проверяется: %v", got)
	}
}

// TestRSIPullbackUnifiedGridsStayWide проверяет ширину осей в каталогах единой процедуры —
// тех, где лежит plateau_base_pullback.json (создаётся в первый день калибровки). Каталоги старых калибровок не проверяются.
func TestRSIPullbackUnifiedGridsStayWide(t *testing.T) {
	for _, path := range rsiPullbackGridFiles(t) {
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), "plateau_base_pullback.json")); err != nil {
			continue
		}
		for _, v := range rsiPullbackUnifiedAxisViolations(filepath.Base(path), rsiPullbackPhases(t, path)) {
			t.Errorf("%s/%s", filepath.Base(filepath.Dir(path)), v)
		}
	}
}
