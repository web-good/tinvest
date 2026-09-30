package backtest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// rsiPullbackZoneFieldsRequired — поля, без которых zone-вход молча выключен: ноль у любого из
// трёх числовых полей отключает его даже при включённом UseZoneEntry.
var rsiPullbackZoneFieldsRequired = []string{"ZoneRSIPeriod", "ZoneRSILower", "ZoneEMAPeriod"}

// rsiPullbackZoneFileName reports whether a file NAME claims to belong to the zone-only
// calibration procedure (docs/rsi_pullback/strategy.md §8.2).
func rsiPullbackZoneFileName(name string) bool {
	for _, prefix := range []string{"cal_zone_", "cal2_zone_", "plateau_zone_", "r2_point_zone", "r2_probe_zone"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// rsiPullbackZoneFileViolations сторожит ловушку нулевого якоря. У незарегистрированного тикера и
// у литерала, равного дефолтам ядра, UseZoneEntry=0 и zone-поля нулевые, поэтому сетка, забывшая
// UseZoneEntry, молча считает pullback, а забывшая одно zone-поле — молча даёт ноль сделок.
// Файл, хоть одна фаза которого задаёт UseZoneEntry, обязан в КАЖДОЙ фазе задать UseZoneEntry
// ровно 1 или 2 и все три zone-поля строго положительными; файл с zone-именем обязан задавать
// UseZoneEntry. Требование на фазу, а не на файл, — чтобы не зависеть от того, как калибратор
// переносит значения между фазами.
func rsiPullbackZoneFileViolations(name string, phases []Phase) []string {
	var armed bool
	for _, ph := range phases {
		if _, ok := ph.Grid["UseZoneEntry"]; ok {
			armed = true
			break
		}
	}
	if !armed {
		if rsiPullbackZoneFileName(name) {
			return []string{fmt.Sprintf("%s: имя zone-файла, но ни одна фаза не задаёт UseZoneEntry — прогон посчитает pullback", name)}
		}
		return nil
	}
	var out []string
	for _, ph := range phases {
		modes, ok := ph.Grid["UseZoneEntry"]
		if !ok || len(modes) == 0 {
			out = append(out, fmt.Sprintf("%s: фаза %q не задаёт UseZoneEntry", name, ph.Name))
		}
		for _, v := range modes {
			if v != core.ZoneEntryAlso && v != core.ZoneEntryOnly {
				out = append(out, fmt.Sprintf("%s: фаза %q: UseZoneEntry=%v, допустимы только 1 и 2", name, ph.Name, v))
			}
		}
		for _, field := range rsiPullbackZoneFieldsRequired {
			values, ok := ph.Grid[field]
			if !ok || len(values) == 0 {
				out = append(out, fmt.Sprintf("%s: фаза %q не задаёт %s — zone-вход молча выключен", name, ph.Name, field))
				continue
			}
			for _, v := range values {
				if v <= 0 {
					out = append(out, fmt.Sprintf("%s: фаза %q: %s=%v выключает zone-вход", name, ph.Name, field, v))
				}
			}
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
		name   string
		file   string
		phases []Phase
		bad    bool
	}{
		{"обычный pullback-файл", "cal_exit.json",
			[]Phase{{Name: "exit", Grid: Grid{"RSIUpper": {55, 70, 85}}}}, false},
		{"zone-only якорь", "cal_zone_exit.json",
			[]Phase{zoneArmedPhase("exit", Grid{"RSIUpper": {55, 70}})}, false},
		{"режим 1 тоже допустим", "cal_zone_entry.json",
			[]Phase{zoneArmedPhase("entry", Grid{"UseZoneEntry": {core.ZoneEntryAlso}})}, false},
		{"точка якоря", "plateau_zone_anchor.json",
			[]Phase{zoneArmedPhase("point", Grid{"StopDailyATR": {0.5}})}, false},
		{"забыт ZoneRSIPeriod", "cal_zone_risk.json",
			[]Phase{zoneArmedPhase("risk", nil, "ZoneRSIPeriod")}, true},
		{"забыт ZoneRSILower", "cal_zone_risk.json",
			[]Phase{zoneArmedPhase("risk", nil, "ZoneRSILower")}, true},
		{"забыт ZoneEMAPeriod", "cal_zone_risk.json",
			[]Phase{zoneArmedPhase("risk", nil, "ZoneEMAPeriod")}, true},
		{"нулевое zone-поле в оси", "cal_zone_trend.json",
			[]Phase{zoneArmedPhase("trend", Grid{"ZoneEMAPeriod": {0, 50, 200}})}, true},
		{"UseZoneEntry свипает выкл", "cal_zone_entry.json",
			[]Phase{zoneArmedPhase("entry", Grid{"UseZoneEntry": {0, 2}})}, true},
		{"UseZoneEntry дробный", "cal_zone_entry.json",
			[]Phase{zoneArmedPhase("entry", Grid{"UseZoneEntry": {2.5}})}, true},
		{"вторая фаза без якоря", "cal_zone_phased.json",
			[]Phase{zoneArmedPhase("entry", nil), {Name: "exit", Grid: Grid{"RSIUpper": {70}}}}, true},
		{"zone-имя без UseZoneEntry", "cal_zone_exit.json",
			[]Phase{{Name: "exit", Grid: Grid{"RSIUpper": {70}}}}, true},
		{"точка второго круга без UseZoneEntry", "r2_point_zone.json",
			[]Phase{{Name: "point", Grid: Grid{"RSIUpper": {70}}}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := rsiPullbackZoneFileViolations(c.file, c.phases)
			if c.bad && len(got) == 0 {
				t.Fatalf("%s: нарушение не найдено", c.file)
			}
			if !c.bad && len(got) != 0 {
				t.Fatalf("%s: ложное нарушение: %v", c.file, got)
			}
		})
	}
}

// TestRSIPullbackZoneFilesPinZone применяет проверку ко всем сеткам каталога. Пока zone-сеток нет,
// тест проходит на pullback-файлах — и этим же сторожит, что проверка их не задевает.
func TestRSIPullbackZoneFilesPinZone(t *testing.T) {
	for _, path := range rsiPullbackGridFiles(t) {
		name := filepath.Base(path)
		for _, v := range rsiPullbackZoneFileViolations(name, rsiPullbackPhases(t, path)) {
			t.Errorf("%s/%s", filepath.Base(filepath.Dir(path)), v)
		}
	}
}
