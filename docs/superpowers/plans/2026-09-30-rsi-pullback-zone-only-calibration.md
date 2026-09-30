# Процедура калибровки zone-only тикера — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Поставить повторяемую процедуру калибровки тикера rsi_pullback в режиме `UseZoneEntry=2`: сторожевой тест zone-сеток, описание процедуры в `strategy.md` и команду `/pullback-zone-calibrate`.

**Architecture:** Чистая функция-проверка `rsiPullbackZoneFileViolations(name, phases)` в новом тестовом файле пакета `backtest` + тест, прогоняющий её по всем файлам `data/params/rsi_pullback/**`. Процедура — текстовая: раздел §8.2 в `docs/rsi_pullback/strategy.md` (механика, без чисел тикеров) и команда `.claude/commands/pullback-zone-calibrate.md` (пошаговый протокол со СТОП-точками). Калибровка тикеров в объём не входит.

**Tech Stack:** Go 1.25, `testing`; Markdown.

**Spec:** `docs/superpowers/specs/2026-09-30-rsi-pullback-zone-only-calibration-design.md`

## Global Constraints

- Ветка — `feat/pullback-zone-entry` (НЕ мержить, НЕ пушить).
- Якорь: `UseZoneEntry = 2`, `ZoneRSIPeriod = 4`, `ZoneRSILower = 25`, `ZoneEMAPeriod = 200`.
- Имена zone-файлов: `cal_zone_*`, `cal2_zone_*`, `plateau_zone_*`, `r2_point_zone*`, `r2_probe_zone*`.
- `docs/rsi_pullback/*.md` — только механика: без чисел тикеров, без результатов прогонов (правило CLAUDE.md).
- Коммиты на русском, в конце строка `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Проверка: `go test ./internal/service/backtest/ -run <Имя> -count=1`; финал — `./bin/mage ci` зелёный. `go build ./...` не использовать (падает на `magefiles`).

## Review Focus

1. Файл с `UseZoneEntry: [1]` (режим «pullback + zone») — должен проходить ту же проверку, что и `[2]`; тест-случай в Task 1.
2. Многофазный файл, где zone-поля заданы только в первой фазе, — нарушение по второй фазе; тест-случай в Task 1.
3. Дробное значение `UseZoneEntry: [2.5]` — `applyField` молча усечёт его до 2; проверка обязана его отвергнуть; тест-случай в Task 1.
4. Обычные pullback-файлы (без `UseZoneEntry`, имя не zone) — ноль нарушений, иначе упадёт весь каталог; тест-случай в Task 1.
5. Файл с zone-именем, но без `UseZoneEntry` (включая `r2_point_zone*`) — нарушение; тест-случай в Task 1.

---

### Task 1: Сторож zone-сеток

**Files:**
- Create: `internal/service/backtest/rsi_pullback_zone_grid_test.go`

**Interfaces:**
- Consumes: `Phase{Name string; KeepTop int; Grid Grid}`, `type Grid map[string][]float64` (`internal/service/backtest/calibrate.go`); `rsiPullbackGridFiles(t) []string`, `rsiPullbackPhases(t, path) []Phase` (`rsi_pullback_grid_test.go`); `core.ZoneEntryAlso = 1`, `core.ZoneEntryOnly = 2`.
- Produces: `rsiPullbackZoneFileName(name string) bool`, `rsiPullbackZoneFileViolations(name string, phases []Phase) []string`, тесты `TestRSIPullbackZoneFileViolations`, `TestRSIPullbackZoneFilesPinZone`.

- [ ] **Step 1: Написать падающий тест на чистую проверку**

Создать `internal/service/backtest/rsi_pullback_zone_grid_test.go`:

```go
package backtest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

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
```

- [ ] **Step 2: Убедиться, что тест не компилируется**

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackZoneFileViolations -count=1`
Expected: FAIL — `undefined: rsiPullbackZoneFileViolations`

- [ ] **Step 3: Реализовать проверку и тест по реальным файлам**

Дописать в тот же файл (после импортов, перед `zoneArmedPhase`):

```go
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
```

- [ ] **Step 4: Прогнать оба теста**

Run: `go test ./internal/service/backtest/ -run 'TestRSIPullbackZoneFileViolations|TestRSIPullbackZoneFilesPinZone' -count=1 -v 2>&1 | tail -25`
Expected: PASS, все 13 подслучаев и тест по каталогу.

- [ ] **Step 5: Мутационная проверка**

Временно заменить в `rsiPullbackZoneFieldsRequired` список на `{"ZoneRSIPeriod", "ZoneRSILower"}` — прогнать Step 4: подслучай «забыт ZoneEMAPeriod» обязан упасть. Вернуть. Затем временно заменить условие `v != core.ZoneEntryAlso && v != core.ZoneEntryOnly` на `v == 0` — подслучай «UseZoneEntry дробный» обязан упасть. Вернуть. Проверить `git diff` — только новый файл.

- [ ] **Step 6: Линт пакета**

Run: `./bin/golangci-lint run ./internal/service/backtest/...`
Expected: `0 issues.`

- [ ] **Step 7: Commit**

```bash
git add internal/service/backtest/rsi_pullback_zone_grid_test.go
git commit -m "test(rsi_pullback): сторож zone-сеток — каждая фаза задаёт UseZoneEntry 1/2 и три zone-поля

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Процедура в strategy.md и CLAUDE.md

**Files:**
- Modify: `docs/rsi_pullback/strategy.md` — новый подраздел `### 8.2.` сразу перед строкой `## 9. На что смотреть в отчёте калибровки`; строки таблицы §2 для четырёх zone-полей.
- Modify: `CLAUDE.md` — строка про zone-вход в разделе Layout.

**Interfaces:**
- Consumes: имена из Task 1 (`TestRSIPullbackZoneFilesPinZone`, префиксы файлов).
- Produces: ссылку «§8.2», на которую опирается команда Task 3.

- [ ] **Step 1: Обновить таблицу §2**

В `docs/rsi_pullback/strategy.md` в четырёх строках таблицы параметров (`UseZoneEntry`, `ZoneRSIPeriod`, `ZoneRSILower`, `ZoneEMAPeriod`) заменить текст последней колонки `нет (тема zone — со спекой калибровки)` на `только в zone-only процедуре (§8.2)`.

- [ ] **Step 2: Вставить §8.2 перед `## 9.`**

```markdown
### 8.2. Калибровка тикера в режиме «только zone» (`UseZoneEntry = 2`)

Режим 2 заводят на тикере, где pullback-вход не торгуется. Каноническая процедура §8–§8.1 мерит
поля pullback-входа, которые в этом режиме решений не дают, поэтому у zone-only тикера своя
процедура; пошаговый протокол — команда `/pullback-zone-calibrate <TICKER>`
(`.claude/commands/pullback-zone-calibrate.md`).

**Якорь.** Каждая сетка во всех полях, которые она не свипует, фиксирует `UseZoneEntry = 2`,
`ZoneRSIPeriod = 4`, `ZoneRSILower = 25`, `ZoneEMAPeriod = 200`. Незарегистрированный тикер и
литерал, равный дефолтам ядра, несут `UseZoneEntry = 0` и нулевые zone-поля: сетка, забывшая
`UseZoneEntry`, молча посчитает pullback, забывшая одно zone-поле — молча даст ноль сделок.
Сторож — `TestRSIPullbackZoneFilesPinZone`: файл, задающий `UseZoneEntry`, обязан в каждой фазе
задать его значением 1 или 2 и все три zone-поля строго положительными; файл с zone-именем
(`cal_zone_*`, `cal2_zone_*`, `plateau_zone_*`, `r2_point_zone*`, `r2_probe_zone*`) обязан задавать
`UseZoneEntry`. Baseline процедуры — сам якорь (`plateau_zone_anchor.json`), а не дефолты ядра:
дефолты ядра в режиме 0 — это pullback.

**Темы** (`cal_zone_<тема>.json`, отдельно от pullback-сеток тикера), две волны:

| Волна | Тема | Оси |
|---|---|---|
| вход | `zone_entry` | `ZoneRSIPeriod` × `ZoneRSILower` |
| вход | `zone_trend` | `ZoneEMAPeriod` |
| выход | `exit` | `RSIPeriod` × `RSIUpper` |
| выход | `risk` | `StopDailyATR` × `TPDailyATR` |
| выход | `trail` | `UseRSIExit` × `UseTrail` × `TrailDailyATR` |

Сначала вход над якорем; принятые правилом большинства значения входа вписываются в якорь, и
выходы калибруются поверх зафиксированного входа. `exit` свипает `RSIPeriod` связкой с порогом: в
режиме 2 это длина RSI только выхода. Темы pullback-входа (entry, trend, trend_low, day, volume,
vol_window, screen) и арбитры не гоняются.

**Сборка и проверка** — как в каноне (большинство 3/4 невырожденных фолдов, соседи плато, гейты
A–D), с двумя отличиями: все сравнения идут против якоря, а к семи пунктам стоп-условия добавлен
восьмой — **ноль SL-выходов у точки на полном окне означает отказ**: стоп, который ни разу не
сработал (трейл его перекрыл), не защищает. Ось `FreshDayATR` в режиме 2 не действует, поэтому
провал гейта входов в ночные часы не лечится уводом поля. Кандидаты в прод — точка и якорь; при
первом включении режима на тикере обязательна сверка `cmd/pullparity` (окно `Lookback` режима 2
короче обычного, §7).
```

- [ ] **Step 3: Проверить, что в §8.2 нет чисел тикеров**

Run: `awk '/^### 8.2/,/^## 9/' docs/rsi_pullback/strategy.md | grep -n -E 'AFKS|SBER|DOMRF|PF [0-9]|[0-9]\.[0-9]{3}'`
Expected: пустой вывод.

- [ ] **Step 4: Обновить CLAUDE.md**

В `CLAUDE.md` в строке Layout заменить фрагмент
`— docs/rsi_pullback/strategy.md §3.1; отдельная стратегия rsi_zone удалена 2026-09-30.`
на
`— docs/rsi_pullback/strategy.md §3.1; режим \`UseZoneEntry=2\` — только zone-вход, калибровка такого тикера — \`/pullback-zone-calibrate <TICKER>\` (§8.2); отдельная стратегия rsi_zone удалена 2026-09-30.`

- [ ] **Step 5: Commit**

```bash
git add docs/rsi_pullback/strategy.md CLAUDE.md
git commit -m "docs(rsi_pullback): §8.2 — процедура калибровки тикера в режиме «только zone»

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Команда `/pullback-zone-calibrate` и финальная проверка

**Files:**
- Create: `.claude/commands/pullback-zone-calibrate.md`

**Interfaces:**
- Consumes: §8.2 (Task 2), `TestRSIPullbackZoneFilesPinZone` (Task 1), спека.
- Produces: команду для владельца.

- [ ] **Step 1: Создать файл команды**

Создать `.claude/commands/pullback-zone-calibrate.md` с содержимым ровно как ниже:

````markdown
---
description: Калибровка тикера rsi_pullback в режиме «только zone» (UseZoneEntry=2) — якорь, темы входа и выходов, walk-forward, вердикт, пакет тикера
argument-hint: <TICKER> [заметки владельца]
---

# Калибровка zone-only тикера rsi_pullback: $ARGUMENTS

Ты калибруешь rsi_pullback в режиме `UseZoneEntry=2` (только zone-вход, выходы общие) под тикер из
аргумента: сам строишь сетки, сам гоняешь бэктесты, сам выбираешь точку и доводишь тикер до
готового пакета. Владелец хочет вердикт и готовый код, а не меню: выбирай сам и обосновывай
числами. Останавливайся и спрашивай только в точках **СТОП**.

Прочитай до начала:

- `docs/superpowers/specs/2026-09-30-rsi-pullback-zone-only-calibration-design.md` — спека процедуры;
- `docs/rsi_pullback/strategy.md` — §3.1 (zone-вход и режимы), §4 (выходы), §7 (`Lookback`), §8.2;
- `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` — `Params`, `ZoneEntryOnly`, `Lookback`;
- `docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md` — образец канона (гейты A–D, стоп-условие, второй круг, правило прода);
- пакет тикера `internal/service/trading_strategy/rsi_pullback/strategy/<t>/` и `data/params/rsi_pullback/<t>/`, если они есть, и память проекта про этот тикер.

## 0. Жёсткие правила каждого прогона

- **`-strategy rsi_pullback -interval Minutes30` в каждой команде `cmd/backtest`.** Дефолт CLI —
  `Hour1`: ошибки не будет, будут чужие числа.
- **`-months`, `-train-months`, `-test-months` в каждой команде walk-forward.** Дефолт `-months` — 12.
- **`-refresh` — только в самом первом прогоне.** Дальше хвост кэша дотягивается сам.
- **Прогоны строго последовательно**: параллельные запуски ломают файл кэша.
- **Окно считается от момента запуска** — сравниваемые прогоны делай в один день; годы и хвосты
  режь фиксированными датами; дату прогона пиши в `_comment` и в отчёт.
- Прогон, упавший на ошибке API (`rpc error: code = Internal ...`), перезапусти, его числа не пиши.
- Отчёты — в `./reports/<TICKER>_zone/...` (вне git).
- **Каждая фаза каждой сетки задаёт `UseZoneEntry` и три zone-поля** (сторож
  `TestRSIPullbackZoneFilesPinZone`). У незарегистрированного тикера и у литерала-дефолтов
  `UseZoneEntry=0`: забытое поле — это pullback или ноль сделок без ошибки.
- **Если литерал тикера уводит поля выхода от дефолтов ядра** (`RSIPeriod`, `RSIUpper`,
  `UseRSIExit`, `StopDailyATR`, `TPDailyATR`, `UseTrail`, `TrailDailyATR`, `DailyATRPeriod`), каждая
  сетка перечисляет их явно значениями дефолтов ядра: выходы zone-only калибруются с нуля.
- Поля pullback-входа (`RSILower`, `EMAFast`, `EMASlow`, `UseDayATRGate`, `FreshDayATR`,
  `SpentDayATR`, `UseVolume`, `Vol*`) в zone-сетках не появляются: в режиме 2 они решений не дают.
- Документация: `docs/rsi_pullback/*.md` — только механика. Результаты — в doc-comment пакета
  тикера и в `_comment` сеток.
- Ветка `feat/<ticker>-zone-only` от текущей ветки с режимом `ZoneEntryOnly` (если он ещё не в
  `main`) или от `main`. Коммиты на русском, в конце строка `Co-Authored-By` из системной подсказки.

## 1. Разведка и якорь

1. Ветка `feat/<ticker>-zone-only`.
2. Длина истории — первый бар `data/candles/<T>_Minutes30.json` (после прогрева кэша):
   - ≥ 36 мес — основная схема **36/12/6** (4 фолда), контрольная **36/18/6** (3 фолда);
   - 24–36 мес — **24/12/3** (4 фолда), контрольная **24/15/3** (3 фолда);
   - меньше 24 мес — **СТОП**, доложи.
3. Файл якоря `data/params/rsi_pullback/<t>/plateau_zone_anchor.json`: одна фаза `point`, по
   одному значению `UseZoneEntry 2`, `ZoneRSIPeriod 4`, `ZoneRSILower 25`, `ZoneEMAPeriod 200` и
   все восемь полей выхода по значениям `core.DefaultParams()`. `_comment` — что это и команды.
4. Baseline якоря на полном окне (первый прогон — с `-refresh`):
   ```
   go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 \
     -calibrate data/params/rsi_pullback/<t>/plateau_zone_anchor.json -out ./reports/<T>_zone/anchor \
     -months <M> -min-trades 1 -refresh
   ```
   Сверь в журнале, что у всех сделок `EntryReason` начинается с `zone:`. Иначе — **СТОП**.
5. Якорь меньше 20 сделок на полном окне — **СТОП**: доложи, предложи отказ.
6. Walk-forward якоря: основная схема, контрольная, основная с `-commission 0.001`
   (`-min-trades 1`). Сверь «Фолдов: N» в шапке с ожидаемым; расхождение — **СТОП**.
7. Профиль якоря: сделки, PF, net, max DD (₽ и %), win rate, expectancy, доли выходов
   SL/TRAIL/TP/RSI, удержание (медиана / p90 / максимум), ночёвки, входы и выходы в выходные и в
   часы 02–06, календарные годы и полугодия, хвосты 6 и 12 месяцев.
8. **Реальный круг издержек:** 2 × `MinPriceIncrement` / цена. Выше 0.2% — порог пункта 4 (§6)
   ужесточается до реального круга, запиши.
9. **Дивидендные отсечки** окна — из T-Invest `GetDividends` (образец `reports/_analysis/divprobe/`).
   Объяви до прогонов, не меняй.
10. **Скрипт журнала** `reports/_analysis/zone_journal.py` (есть от rsi_zone): проверь, что он
    понимает выходы TP и TRAIL, иначе дополни по образцу `reports/_analysis/mdmg_journal.py`.
    Даты отсечек и хвостов — аргументами. Сверь его вывод с шапкой отчёта якоря.

## 2. Сетки первого круга

Файлы `data/params/rsi_pullback/<t>/cal_zone_<тема>.json`. У каждого `_comment`: что меряет тема,
полная команда запуска с путём `<t>/cal_zone_<тема>.json` (этого требует
`TestRSIPullbackCalFilesValid`), после прогона — строка `РЕЗУЛЬТАТ ПРОГОНА <дата>: …` (pooled OOS,
пул, пофолдовые PF, голоса фолдов).

Минимальные оси (шире можно, уже нельзя):

| Волна | Тема | Оси |
|---|---|---|
| 1 | `zone_entry` | `ZoneRSIPeriod` 2,3,4,5,6,8,10,14 × `ZoneRSILower` 5,10,15,20,25,30,35,40,45 |
| 1 | `zone_trend` | `ZoneEMAPeriod` 20,30,50,75,100,150,200,250,300,400 |
| 2 | `exit` | `RSIPeriod` 2,3,4,5,6,8,10,14 × `RSIUpper` 50,55,60,65,70,75,80,85,90,95 |
| 2 | `risk` | `StopDailyATR` 0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.2,1.5,2.0 × `TPDailyATR` 0.1,0.15,0.2,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.5 |
| 2 | `trail` | `UseRSIExit` 0,1 × `UseTrail` 1 × `TrailDailyATR` 0.2,0.3,0.4,0.5,0.7,1.0,1.5 |

Инварианты:

- `StopDailyATR` нигде не 0; ни одна тема, кроме `risk`, не свипает стоп;
- `ZoneRSIPeriod ≥ 2`, `RSIPeriod ≥ 2`;
- при истории < 36 мес `ZoneEMAPeriod`, чьё окно `2·N+20` съедает больше ~10% окна, выбрасывается с
  записью в `_comment`;
- упор в край — повод для зонда за краем (§3), а не для обрезки оси.

Сторожевой тест осей: в `internal/service/backtest/rsi_pullback_<t>_grid_test.go` (завести по
образцу `rsi_pullback_afks_grid_test.go`, если его нет) — отдельная функция
`Test<T>ZoneGridsStayWide` с краевыми узлами из таблицы через `rsiPullbackTickerGrid`. Сетки волны 2
пишутся после сборки входа (в них фиксируются принятые zone-значения). Тест + сетки волны 1 —
первый коммит; прогони `go test ./internal/service/backtest/ -run 'Zone|<T>' -count=1`.

## 3. Walk-forward тем и сборка

Каждая тема:
```
go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/<t>/cal_zone_<тема>.json -out ./reports/<T>_zone/<тема> \
  -months <M> -train-months <TR> -test-months <TE> -min-trades 20 -metric profit_factor
```

**Вырожденный фолд** — меньше пяти сделок OOS или ни одной убыточной: не голосует, как PF не
засчитывается.

**Правило большинства:** поле уходит от якоря, только если за значение ≥ 3 из 4 невырожденных
фолдов основной схемы (на 3 фолдах — единогласно); ничья 2/2 — не большинство. Источники:
`ZoneRSIPeriod`, `ZoneRSILower` — `zone_entry`; `ZoneEMAPeriod` — `zone_trend`; `RSIPeriod`,
`RSIUpper` — `exit`; `StopDailyATR`, `TPDailyATR` — `risk`; `UseRSIExit`, `UseTrail`,
`TrailDailyATR` — `trail`. Таблица: поле → тема → голоса по фолдам → вырожденные → принято → якорь.

**Порядок:** волна 1 → сборка входа → `plateau_zone_entry.json` (якорь с принятыми zone-значениями)
→ волна 2 над ним → сборка выходов → `plateau_zone_point.json`.

**Соседи плато:** для каждого поля, ушедшего от якоря, — значение и два соседа по оси
(`plateau_zone_<поле>_<v>.json`), у каждого годы и хвосты; край — зонд за краем; плато уже 0.05 PF —
«сигнала нет», прямым текстом.

**Гейт A (потолок стопа):** выживаемость уровня (доля будних дней, чей размах достаёт уровня в
дневных ATR(14) предыдущего дня) и таблица срабатываний стопа **поверх зафиксированного входа**.
Потолок — строжайшее из «выживаемость ≥ 30%» и «самый широкий стоп, срабатывающий не реже чем в 10%
сделок». Применяется к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`.

**Гейт B:** max DD точки на полном окне не выше якоря ни в ₽, ни в %. Ничьи < 0.05 PF — в пользу
меньшей просадки, затем меньшего числа убыточных полугодий.

**Третий контур и гейт C:** профиль точки против якоря. Падение доли SL вместе с ростом удержания и
ночёвок — подпись капкана: бери более узкий стоп в пределах плато. Входов в выходные — ноль.

**Гейт D:** сделки через объявленные отсечки (`вход < отсечка ≤ выход`) — ноль.

## 4. Проверка точки — восемь пунктов стоп-условия

`plateau_zone_point.json` (`-min-trades 1`), в один день: основная схема, контрольная, основная с
`-commission 0.001` (или реальным кругом), полное окно.

1. pooled OOS PF < 1.0 на основной схеме;
2. меньше 20 сделок в пуле OOS основной схемы;
3. pooled OOS PF < 1.0 на контрольной;
4. pooled OOS PF < 1.0 при удвоенных (или реальных) издержках;
5. убыточен один из двух последних полных календарных лет или текущий год (огрызок в начале окна —
   не пункт, но пишется);
6. хвост 6 или хвост 12 (по дате входа) с PF < 1.0, либо меньше пяти сделок в хвосте 6; знак хвоста
   6 сверь с объединённым OOS PF двух последних фолдов (net = `OOS NetPnL%` × 1000,
   `GL = net/(PF−1)`, `GP = PF·GL`; формулу сначала проверь на якоре) — знаки разошлись: **СТОП**;
7. провал гейта C или D;
8. **ноль SL-выходов у точки на полном окне.**

Гейты A и B — ограничения сборки; точку, которую внутри них не собрать, считай сработавшей по
пункту 1. По каждому пункту — число и вердикт.

## 5. Второй круг — только при провале первого

Файлы `cal2_zone_<тема>.json`, точка — `r2_point_zone.json`. Зоны — по фактическим голосам фолдов
первого круга, шаг — половина шага первого круга, не больше четырёх тем, каждое поле свипает ровно
одна тема. Большинство — в зоне ±1 шаг (голоса соседних узлов идут центральному, если он сам набрал
голос). `-min-trades 1`. Пункт 2 снят (пул меньше десяти — непредставителен, закрывает работу);
пункты 1, 3–8 в силе. Гейты A, B и окно — прежние.

## 6. Правило прода

Кандидаты — точка (первого или второго круга) и якорь; допускается кандидат, не сработавший ни по
одному из восьми пунктов. Выбор: больший pooled OOS основной схемы → при разнице < 0.05 меньшая max
DD в % → меньше убыточных полугодий → якорь.

Не допущен никто — **отказ**: литерал пакета не меняется, zone-сетки остаются с `_comment` как
протокол, в памяти — причина. Сводка владельцу (§8), без мержа.

## 7. Финал при допуске

1. Литерал пакета `internal/service/trading_strategy/rsi_pullback/strategy/<t>/<t>.go`:
   `UseZoneEntry: core.ZoneEntryOnly`, три zone-поля и поля выхода победителя; doc-comment — полный
   разбор (дата, протокол, темы одной строкой, таблица голосов, плато, гейты A–D, восемь пунктов с
   числами, точка против якоря, риски). Тест-снимок литерала обновить. Если пакета нет — завести
   его по образцу соседей и алиас в `internal/service/backtest/rsi_pullback_registry.go` и в
   `internal/service/trading_strategy/rsi_pullback/live/registry.go`.
2. `go run ./cmd/pullparity` по тикеру — ноль расхождений.
3. `go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 -months <M>` без
   `-calibrate` — ровно числа точки на полном окне (доказывает, что реестр взял литерал).
4. `./bin/mage ci` зелёный.
5. Коммиты по смыслу (якорь+волна 1+тест; волна 2; точка+вердикт; пакет+реестр).
6. **СТОП:** сводка владельцу и вопрос про мерж и заведение в `RSI_PULLBACK_TICKERS`
   (`env/prod.env`, `.example`-файлы, сторож env) — это отдельное решение владельца.
7. Память: проектная запись тикера и строка в `MEMORY.md`.

## 8. Сводка владельцу

Коротко, по-русски: вердикт (точка / якорь / отказ); литерал и отличие от якоря; таблица тем
(pooled/пул/голоса); точка против якоря на всех схемах; восемь пунктов с числами; главные риски
строкой; что осталось владельцу.
````

- [ ] **Step 2: Сверить команду со спекой**

Проверить построчно по спеке: якорь (§4 спеки), схемы (§5), пять тем и оси (§6), источники полей и гейты (§7), восемь пунктов (§8), правило прода и финал (§9). Любое расхождение — исправить команду, не спеку.

Run: `grep -c "СТОП" .claude/commands/pullback-zone-calibrate.md`
Expected: число ≥ 6.

- [ ] **Step 3: Полная проверка**

Run: `./bin/mage ci 2>&1 | grep -E "FAIL|issues|^exit"; echo "exit=${PIPESTATUS[0]}"`
Expected: `exit=0`, `0 issues.`, без `FAIL`.

- [ ] **Step 4: Commit**

```bash
git add .claude/commands/pullback-zone-calibrate.md
git commit -m "chore(claude): команда /pullback-zone-calibrate — калибровка тикера в режиме «только zone»

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
