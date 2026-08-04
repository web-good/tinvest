# DOMRF под `rsi_pullback` — план подготовки к калибровке

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Завести DOMRF как отслеживаемый тикер `rsi_pullback` — пакет параметров, каталог из девяти сеток калибровки с осями по замерам инструмента — и прогнать две первые темы как разведку.

**Architecture:** Три слоя, каждый со своим сторожем. Реестр (`rsi_pullback_registry.go`) связывает тикер с пакетом параметров; пакет `domrf` остаётся baseline-tracking, то есть возвращает `core.DefaultParams()` без копирования полей; каталог `data/params/rsi_pullback/domrf/` несёт по одному JSON-файлу на тему калибровки, и оси каждого обоснованы замером в его `_comment`.

**Tech Stack:** Go 1.25, `go test -race`, `./bin/mage ci`, `go run ./cmd/backtest`.

## Global Constraints

- Спека: `docs/superpowers/specs/2026-08-04-domrf-rsi-pullback-prep-design.md`. Все числовые оси взяты оттуда — не пересчитывать и не «улучшать» их по ходу.
- `StopDailyATR = 0` не должно появиться ни в одном файле сеток: стратегия держит позицию через ночи и выходные.
- `_comment` каждого `cal_*.json` обязан содержать подстроку `domrf/<имя этого файла>` — иначе падает `TestRSIPullbackCalFilesValid`, который так ловит команду, скопированную у соседнего файла.
- `DefaultParams()` пакета `domrf` возвращает `core.DefaultParams()` целиком. Явный литерал по `docs/rsi_pullback/strategy.md` §8.0.1 означает «тикер откалиброван» — этого статуса у DOMRF нет и по итогам этого плана не появится.
- Язык `_comment` в новых файлах — русский: это рабочая документация владельца, читаемая перед прогоном. Doc-комментарии Go — английские, как в соседних пакетах `nvtk`/`ugld`.
- Планка репо (pooled OOS PF ≥ 1.5 по walk-forward) к результатам этого плана **не применяется и не имитируется**: истории 8.4 месяца, walk-forward неисполним. Всё, что получено, маркируется как разведка.
- Ветка: `feat/rsi-pullback`. Коммиты — после каждой задачи.

---

### Task 1: Починить сборку — дочистить реестр `rsi_pullback`

Ветка сейчас не собирается: в индексе staged-удаление десяти пакетов тикеров, а реестр продолжает их импортировать. Все десять возвращали `core.DefaultParams()` без изменений, ровно то же самое даёт `RSIPullbackLookupOrGeneric` незарегистрированному тикеру — удаление не теряет ни одного значения параметра.

**Files:**
- Modify: `internal/service/backtest/rsi_pullback_registry.go:7-21` (импорты), `:47-62` (записи карты)
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1, строка таблицы состояний про `sber`)
- Test: `internal/service/backtest/rsi_pullback_registry_test.go` (существующий, менять не нужно)

**Interfaces:**
- Consumes: ничего
- Produces: собирающееся дерево; `rsiPullbackRegistry` с ключами `GAZP`, `NVTK`, `T`, `UGLD`

- [ ] **Step 1: Убедиться, что сборка сейчас падает (red)**

Run: `go build ./internal/... ./pkg/... ./cmd/...`
Expected: FAIL, десять ошибок вида `package tinvest/internal/service/trading_strategy/rsi_pullback/strategy/afks is not in std`

- [ ] **Step 2: Убрать десять импортов**

В `internal/service/backtest/rsi_pullback_registry.go` удалить строки импортов пакетов `afks`, `astr`, `eutr`, `mdmg`, `pikk`, `plzl`, `rusal`, `sber`, `sfin`, `ydex`. Блок импортов после правки:

```go
import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	rsipullbackgazp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	rsipullbacknvtk "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	rsipullbacktbank "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	rsipullbackugld "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)
```

- [ ] **Step 3: Убрать десять записей карты**

Тело `rsiPullbackRegistry` после правки:

```go
var rsiPullbackRegistry = map[string]Binding{
	rsipullbackgazp.Ticker:  rsiPullbackBindingFor(rsipullbackgazp.Ticker, rsipullbackgazp.DefaultParams),
	rsipullbacknvtk.Ticker:  rsiPullbackBindingFor(rsipullbacknvtk.Ticker, rsipullbacknvtk.DefaultParams),
	rsipullbacktbank.Ticker: rsiPullbackBindingFor(rsipullbacktbank.Ticker, rsipullbacktbank.DefaultParams),
	rsipullbackugld.Ticker:  rsiPullbackBindingFor(rsipullbackugld.Ticker, rsipullbackugld.DefaultParams),
}
```

- [ ] **Step 4: Проверить сборку и тесты (green)**

Run: `go build ./internal/... ./pkg/... ./cmd/...`
Expected: без вывода (успех)

Run: `go test ./internal/service/backtest/ -run TestRSIPullback -v`
Expected: PASS для всех. Обратить внимание на `TestRSIPullbackBindingBuildsForTicker`: он дёргает `SBER` через `RSIPullbackLookupOrGeneric` и после удаления пакета проверяет ровно то же самое — fallback на baseline.

- [ ] **Step 5: Поправить §8.0.1 документации**

В `docs/rsi_pullback/strategy.md`, таблица состояний в §8.0.1, строка «калибровка не проводилась». Заменить ячейку с примерами `` `sber`, ещё 11 тикеров `` на:

```
| калибровка не проводилась | `return core.DefaultParams()` — тикер отслеживает baseline, правка дефолтов доходит до него | `nvtk` |
```

Ниже таблицы добавить абзац:

```markdown
Реестр намеренно короткий: тикер, у которого нет собственного литерала, не нуждается в
пакете — `RSIPullbackLookupOrGeneric` прогоняет незарегистрированное имя на
`core.DefaultParams()`, ровно на том же, что вернул бы пустой пакет. Десять таких пакетов
(`afks`, `astr`, `eutr`, `mdmg`, `pikk`, `plzl`, `rusal`, `sber`, `sfin`, `ydex`) удалены
2026-08-04: они не несли ни одного значения параметра, но создавали впечатление, что тикер
отслеживается стратегией.
```

- [ ] **Step 6: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_registry.go docs/rsi_pullback/strategy.md \
  internal/service/trading_strategy/rsi_pullback/strategy/
git commit -m "$(cat <<'EOF'
fix(rsi_pullback): дочищаем реестр под удалённые baseline-пакеты

Десять пакетов тикеров были удалены из рабочего дерева, но реестр продолжал их
импортировать — сборка падала десятью ошибками. Все они возвращали
core.DefaultParams() без изменений, что RSIPullbackLookupOrGeneric даёт и
незарегистрированному тикеру: ни одного значения параметра не потеряно.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Пакет `domrf` и его регистрация

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/domrf/domrf.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт + запись карты)
- Test: `internal/service/backtest/rsi_pullback_registry_test.go` (дописать один тест в конец файла)

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из Task 1 (не менялись)
- Produces: `domrf.Ticker` (константа `"DOMRF"`), `domrf.DefaultParams() core.Params`; ключ `"DOMRF"` в `rsiPullbackRegistry`

- [ ] **Step 1: Написать падающий тест**

Дописать в конец `internal/service/backtest/rsi_pullback_registry_test.go`:

```go
// TestRSIPullbackDOMRFIsRegisteredAndTracksBaseline пинует ДВА разных факта, которые
// RSIPullbackLookupOrGeneric снаружи неотличимы: DOMRF присутствует в карте (а не проваливается
// в generic-ветку) И при этом возвращает baseline, а не литерал. Второе — это статус «калибровка
// не проводилась» из docs/rsi_pullback/strategy.md §8.0.1: истории у DOMRF 8.4 месяца с IPO
// 2025-11-20, и любой литерал здесь означал бы, что тикер откалиброван, чего нет.
func TestRSIPullbackDOMRFIsRegisteredAndTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["DOMRF"]
	if !ok {
		t.Fatal("DOMRF отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("DOMRF params = %+v, want baseline %+v — литерал означал бы «откалиброван»", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "DOMRF" {
		t.Fatalf("Ticker() = %q, want DOMRF", got)
	}
}
```

- [ ] **Step 2: Запустить тест, убедиться, что падает**

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackDOMRFIsRegisteredAndTracksBaseline -v`
Expected: FAIL с `DOMRF отсутствует в rsiPullbackRegistry`

- [ ] **Step 3: Создать пакет**

`internal/service/trading_strategy/rsi_pullback/strategy/domrf/domrf.go`:

```go
// Package domrf supplies the ticker and starting rsi_pullback Params for DOMRF (ДОМ.РФ).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches every
// uncalibrated ticker instead of silently drifting away from it.
//
// Read this before calibrating DOMRF. The instrument IPO'd on 2025-11-20, so its entire history
// is 8.4 months (158 weekday dailies) — the walk-forward protocol of docs/rsi_pullback/strategy.md
// §8 needs a 12-month train window and simply does not fit. Worse, that whole history is a single
// post-IPO uptrend: 1749.8 -> 2273.2 RUB, +29.9%, with no sustained downward regime. A long-only
// pullback strategy shows an inflated profit factor in that regime whatever its parameters, so a
// good number here is evidence of the regime, not of an edge. The daily ATR(14) runs a median
// 1.94% of price — half of UGLD's 4.28% and in the same class as GAZP and T — which is why the
// grids in data/params/rsi_pullback/domrf/ are rescaled rather than copied from a sibling.
//
// Once a walk-forward on 18-24 months of history picks a winning combination, replace the body
// with an explicit literal — from that point the ticker must stop tracking the baseline.
package domrf

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "DOMRF"

// DefaultParams returns DOMRF's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать в реестре**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт (алфавитный порядок — перед `gazp`):

```go
	rsipullbackdomrf "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
```

и запись в карту первой строкой:

```go
	rsipullbackdomrf.Ticker: rsiPullbackBindingFor(rsipullbackdomrf.Ticker, rsipullbackdomrf.DefaultParams),
```

- [ ] **Step 5: Запустить тесты (green)**

Run: `go test ./internal/service/backtest/ -run TestRSIPullback -v`
Expected: PASS, включая новый тест и `TestRSIPullbackRegistryEntriesMatchTheirTicker/DOMRF`, `TestRSIPullbackTickersKeepTheRSIExitArmed`

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/domrf/ \
  internal/service/backtest/rsi_pullback_registry.go \
  internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "$(cat <<'EOF'
feat(rsi_pullback): пакет параметров DOMRF (baseline-tracking)

Тикер отслеживает baseline, а не литерал: истории 8.4 месяца с IPO 2025-11-20,
walk-forward по §8 неисполним, и статус «откалиброван» тикеру не принадлежит.
Doc-комментарий называет обе рамки, которые калибровщик должен знать заранее:
+29.9% за всю историю без нисходящего режима и дневной ATR 1.94% — вдвое уже UGLD.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Сетки тем сигнала — `screen`, `entry`, `trend`, `exit`

Четыре файла, 37 прогонов. Оси взяты из раздела «Замеры, задающие оси сеток» спеки.

**Files:**
- Create: `data/params/rsi_pullback/domrf/cal_screen.json`, `cal_entry.json`, `cal_trend.json`, `cal_exit.json`
- Create: `internal/service/backtest/rsi_pullback_domrf_grid_test.go`

**Interfaces:**
- Consumes: `rsiPullbackPhases(t, path)` и `rsiPullbackParamsDir` из `internal/service/backtest/rsi_pullback_grid_test.go` (тот же пакет, вызывать напрямую)
- Produces: тест-хелпер `domrfGrid(t, file)` — читает файл каталога `domrf/` и возвращает `map[string][]float64` объединённых осей всех его фаз; используется в Task 4

- [ ] **Step 1: Написать падающий тест**

Создать `internal/service/backtest/rsi_pullback_domrf_grid_test.go`:

```go
package backtest

import (
	"path/filepath"
	"testing"
)

// domrfGrid читает файл сеток DOMRF и сливает оси всех его фаз в одну карту. Файлы каталога
// однотемные, поэтому слияние не теряет информации, а тестам не приходится знать имя фазы.
func domrfGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	path := filepath.Join(rsiPullbackParamsDir, "domrf", file)
	out := make(map[string][]float64)
	for _, ph := range rsiPullbackPhases(t, path) {
		for field, values := range ph.Grid {
			out[field] = append(out[field], values...)
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s: сетка пуста", file)
	}
	return out
}

// TestDOMRFSignalGridsPinTheirMeasuredAxes сторожит решения, которые обоснованы замерами
// инструмента, а не вкусом. Каталог domrf/ заведён копированием структуры ugld/, и типовая
// ошибка такой копии — притащить вместе с ней чужие оси: на UGLD дневной ATR 4.28%, на DOMRF
// 1.94%, и половина осей UGLD здесь либо мертва, либо неисполнима.
func TestDOMRFSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	entry := domrfGrid(t, "cal_entry.json")

	// RSI(6) пересекает 10 вниз 18 раз за ВСЮ историю инструмента: после шести гейтов входа
	// и многодневного удержания на этой точке не остаётся выборки вовсе.
	for _, v := range entry["RSILower"] {
		if v < 15 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 15 сигналов не остаётся (RSI(6)@10 = 18 кроссов за 8.4 мес)", v)
		}
	}
	// RSI(3) на инструменте с ATR 1.94% покупает шум, а не откат.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум при ATR 1.94%%", v)
		}
	}
	// Полоса выхода меряется отдельным cal_exit.json. В фазе entry она стоила бы 12x5
	// степеней свободы на выборке в 20-30 сделок — переобучение по построению.
	if _, ok := entry["RSIUpper"]; ok {
		t.Error("cal_entry.json свипует RSIUpper: полоса выхода принадлежит cal_exit.json")
	}

	if got := len(domrfGrid(t, "cal_exit.json")["RSIUpper"]); got != 5 {
		t.Errorf("cal_exit.json: RSIUpper имеет %d значений, want 5", got)
	}

	gates := domrfGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if len(gates[field]) != 2 {
			t.Errorf("cal_screen.json: %s должен свипуть обе точки [0,1], got %v", field, gates[field])
		}
	}

	trend := domrfGrid(t, "cal_trend.json")
	if len(trend["EMAFast"]) != 3 || len(trend["EMASlow"]) != 4 {
		t.Errorf("cal_trend.json: сетка %vx%v, want 3x4", len(trend["EMAFast"]), len(trend["EMASlow"]))
	}
}
```

- [ ] **Step 2: Запустить тест, убедиться, что падает**

Run: `go test ./internal/service/backtest/ -run TestDOMRFSignalGridsPinTheirMeasuredAxes -v`
Expected: FAIL — файлов каталога `domrf/` ещё нет, `rsiPullbackPhases` не сможет прочитать `cal_entry.json`

- [ ] **Step 3: Создать `cal_screen.json`**

```json
{
  "_comment": "SCREEN: цена двух опциональных гейтов в сделках. 4 прогона. Запускать ПЕРВЫМ и ради одного числа — сколько сделок остаётся при каждой комбинации, потому что на DOMRF именно счёт сделок является связывающим ограничением. Инструмент вышел на биржу 2025-11-20, вся история — 8152 30-минутных бара (5942 будних) и 158 будних дневок, то есть 8.4 месяца. Отчёт скринера pullback_screen_Minutes30_20260804_232456.md поставил DOMRF на первое место из 99 прошедших тикеров с PFmed 3.78, но там же стоит TradesMed 4 и Capped 8/24: PF по четырём сделкам не измеряет ничего. Сырых сигналов до гейтов немного: кроссов RSI(4) вниз через 20 — 229 будних за всю историю, через 15 — 146, RSI(6) через 15 — 58. -min-trades 1 обязателен, иначе измеряемые строки отфильтруются раньше, чем будут прочитаны. Если при UseDayATRGate=1 остаётся 3-5 сделок, дальнейшие темы измеряют отдельные сделки, а не конфигурации, и калибровку следует остановить здесь. DOMRF отслеживает baseline: rsi_pullback/strategy/domrf возвращает core.DefaultParams() (RSI 4, 30/70, EMA 10/100, SpentDayATR 0.8, стоп 0.5, TP 0.6), поэтому каждый файл каталога свипует поверх общих дефолтов, а не поверх тикерного литерала. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_screen.json -out ./reports/DOMRF_screen -months 12 -min-trades 1 -metric profit_factor.",
  "phases": [
    {
      "name": "gates",
      "grid": {
        "UseDayATRGate": [0, 1],
        "UseVolume": [0, 1]
      }
    }
  ]
}
```

- [ ] **Step 4: Создать `cal_entry.json`**

```json
{
  "_comment": "ENTRY, форма отката: RSIPeriod x RSILower, 12 прогонов. Ось УЖЕ, чем у ugld/cal_entry.json (80 прогонов), по двум независимым причинам. Первая — дефицит сигналов. Кроссы RSI вниз через уровень, будние бары за всю историю (8.4 мес): RSI(4) 69 через 10, 146 через 15, 229 через 20, 326 через 25, 423 через 30; RSI(5) 37/91/155/234/343; RSI(6) 18/58/109/174/270; RSI(7) 8/34/82/134/218. Уровень 10 из сетки убран: RSI(6)@10 даёт 18 кроссов ЗА ВСЮ ИСТОРИЮ, и после шести гейтов входа плюс многодневного удержания там не останется выборки. Скринер выбрал лучшей конфигурацией именно RSI 4/10 и получил TradesMed 4 — это подтверждает диагноз, а не опровергает его. RSIPeriod 3 не берём по обратной причине: на инструменте с дневным ATR 1.94% такая длина покупает шум, а не откат. Уровень 30 оставлен несмотря на мелкость отката — сигналов дефицит. Вторая причина — RSIUpper НЕ свипуется здесь, в отличие от ugld/. Там полоса выхода едет вместе с глубиной входа, и это оправдано изобилием сигналов; здесь 12x5 степеней свободы на выборке в 20-30 сделок — переобучение по построению. Полоса выхода меряется отдельно, файлом cal_exit.json. -min-trades 10, а не 20-30 как на UGLD: на 8.4 месяцах порог 20 отфильтрует всю сетку и лидерборд окажется пустым. Порог занижен осознанно, и именно поэтому результат остаётся разведочным — 10 сделок не подтверждают edge. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_entry.json -out ./reports/DOMRF_entry -months 12 -min-trades 10 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIPeriod": [4, 5, 6],
        "RSILower": [15, 20, 25, 30]
      }
    }
  ]
}
```

- [ ] **Step 5: Создать `cal_trend.json`**

```json
{
  "_comment": "TREND: EMAFast x EMASlow, 16 прогонов. Ось совпадает с ugld/cal_trend.json, и это единственная тема, где копия оправдана: периоды EMA задаются в барах, а не в единицах цены, поэтому вдвое более узкий ATR DOMRF на них не влияет. Проверено, что EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 5942 будних в кэше, то есть окно прогрева занимает 7% истории. Скринер на своей фиксированной сетке выбрал для DOMRF EMA 20/100 в 24 конфигурациях из 24 — эта пара и есть основная гипотеза темы, остальные строки её проверяют. Отдельно стоит смотреть на 20/200: вся история инструмента — один аптренд +29.9%, и медленная пара в таком режиме почти не выключает вход, то есть перестаёт быть фильтром. Если 20/200 и 20/50 дают близкий profit factor, тренд-гейт на этих данных не работает вовсе, и это надо знать до того, как настраивать риск. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_trend.json -out ./reports/DOMRF_trend -months 12 -min-trades 10 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "trend",
      "grid": {
        "EMAFast": [5, 10, 20],
        "EMASlow": [50, 100, 150, 200]
      }
    }
  ]
}
```

- [ ] **Step 6: Создать `cal_exit.json`**

```json
{
  "_comment": "EXIT: уровень RSIUpper, крест ВВЕРХ через который закрывает сделку. 5 прогонов. Запускать уже с зафиксированной парой стоп/цель из cal_risk.json, а не поверх дефолтов: выход по RSI — третий по приоритету после SL и TP, и при цели 0.6 ATR из baseline он почти не успевает сработать, так что свип поверх дефолтов измерил бы не полосу выхода, а то, насколько рано её опережает тейк. Ось та же, что у ugld/cal_exit.json: уровень RSI не зависит от волатильности инструмента. Смысл краёв: 60 отдаёт сделку при первом же отскоке и превращает стратегию в скальп с многодневным удержанием только на убыточных сделках — опасная асимметрия, её видно по средней длительности победителей против проигравших; 80 требует полноценного возврата в перекупленность и на инструменте с ATR 1.94% достигается редко. Смотреть долю выходов RSI против SL и TP, а не только profit factor. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_exit.json -out ./reports/DOMRF_exit -months 12 -min-trades 10 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "exit",
      "grid": {
        "RSIUpper": [60, 65, 70, 75, 80]
      }
    }
  ]
}
```

- [ ] **Step 7: Запустить тесты (green)**

Run: `go test ./internal/service/backtest/ -run 'TestDOMRFSignalGrids|TestRSIPullbackCalFilesValid' -v`
Expected: PASS. `TestRSIPullbackCalFilesValid` обходит каталог рекурсивно и должен показать подтесты `domrf/cal_screen.json`, `domrf/cal_entry.json`, `domrf/cal_trend.json`, `domrf/cal_exit.json` — каждый проверяет, что имена полей резолвятся через `applyField` и что `_comment` называет собственный путь файла.

- [ ] **Step 8: Коммит**

```bash
git add data/params/rsi_pullback/domrf/ internal/service/backtest/rsi_pullback_domrf_grid_test.go
git commit -m "$(cat <<'EOF'
feat(rsi_pullback): сетки тем сигнала для DOMRF

Четыре файла, 37 прогонов. Оси пересажены на замеры инструмента, а не
скопированы с ugld: RSILower не опускается ниже 15 (RSI(6)@10 даёт 18 кроссов
за всю историю), RSIUpper вынесен из фазы entry в свой cal_exit.json, чтобы не
тратить 12x5 степеней свободы на выборке в 20-30 сделок.

Тест пинует именно эти решения: копия каталога с чужими осями его роняет.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Сетки гейтов и риска — `day`, `day_spent`, `volume`, `risk`, `trail`

Пять файлов, 52 прогона. Здесь оси расходятся с `ugld/` сильнее всего: они выражены в долях дневного ATR, а он у DOMRF вдвое уже.

**Files:**
- Create: `data/params/rsi_pullback/domrf/cal_day.json`, `cal_day_spent.json`, `cal_volume.json`, `cal_risk.json`, `cal_trail.json`
- Modify: `internal/service/backtest/rsi_pullback_domrf_grid_test.go` (дописать один тест)

**Interfaces:**
- Consumes: `domrfGrid(t, file)` из Task 3
- Produces: ничего для последующих задач

- [ ] **Step 1: Написать падающий тест**

Дописать в конец `internal/service/backtest/rsi_pullback_domrf_grid_test.go`:

```go
// TestDOMRFRiskGridsPinTheirMeasuredAxes сторожит оси, выраженные в долях дневного ATR. Именно
// здесь копия ugld/ опаснее всего: там ATR 4.28% от цены, здесь 1.94%, поэтому одна и та же
// цифра означает вдвое меньшее движение и вдвое большую долю издержек.
func TestDOMRFRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	risk := domrfGrid(t, "cal_risk.json")

	// Круг издержек (0.05% за сторону) стоит 0.052 дневного ATR. Стоп 0.3 ATR = 0.58% цены,
	// из которых 17% съедает комиссия, а медианный день покрывает 0.99 ATR — такой стоп сидит
	// внутри обычного внутридневного шума и будет снят сносом, а не провалом сетапа.
	for _, v := range risk["StopDailyATR"] {
		if v < 0.5 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: ниже 0.5 стоп внутри дневного шума (медианный день 0.99 ATR)", v)
		}
		if v == 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=0: многодневное удержание без стопа недопустимо")
		}
	}
	if len(risk["TPDailyATR"]) != 4 {
		t.Errorf("cal_risk.json: TPDailyATR имеет %d значений, want 4", len(risk["TPDailyATR"]))
	}

	day := domrfGrid(t, "cal_day.json")
	// К 07:00 MSK медианный день уже прошёл 0.31 ATR, к 10:00 — 0.55. Порог 0.2 из ugld/
	// отсекает медианный день на первом же баре: ветка «день только начался» становится мёртвой.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: к 07:00 медианный день прошёл 0.31 ATR, порог мёртв", v)
		}
	}
	// Медианный день DOMRF покрывает 0.99 ATR против 0.67 у UGLD, и порога 0.6 достигают
	// 88.2% дней — на этом инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 88%% дней, это не гейт", v)
		}
	}
	// Фаза day всегда идёт со включённым гейтом: цена его отключения меряется cal_screen.json.
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1]", got)
	}

	spent := domrfGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}

	vol := domrfGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}

	trail := domrfGrid(t, "cal_trail.json")
	if len(trail["UseRSIExit"]) != 2 {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want обе точки [0,1]", trail["UseRSIExit"])
	}
}
```

- [ ] **Step 2: Запустить тест, убедиться, что падает**

Run: `go test ./internal/service/backtest/ -run TestDOMRFRiskGridsPinTheirMeasuredAxes -v`
Expected: FAIL — `cal_risk.json` и остальные четыре файла ещё не созданы

- [ ] **Step 3: Создать `cal_day.json`**

```json
{
  "_comment": "DAY: пороги двустороннего гейта состояния дня при UseDayATRGate=1, 12 прогонов. Обе оси сдвинуты относительно ugld/cal_day.json, каждая по своему замеру. FreshDayATR [0, 0.25, 0.35] вместо [0.3, 0.2, 0]: медиана доли дневного ATR, пройденной к слоту (будние бары, n=144 дня), составляет 0.31 к 07:00, 0.45 к 09:00, 0.55 к 10:00, 0.75 к 13:00, 0.98 к 19:00 — то есть порог 0.2 из UGLD отсекает медианный день уже на первом баре и делает ветку «день только начался» мёртвой, а 0.25-0.35 оставляет ей утреннее окно. Это тот же эффект «гейт есть фильтр времени суток», что описан в спеке 2026-07-28, но сдвинутый ранним раскрытием диапазона DOMRF. Ноль в оси — не пропуск, а осознанное выключение ветки целиком: dayStateOK защищает её условием fresh > 0, поэтому ровно ноль убирает её, а 0.01 — нет. SpentDayATR [0.8, 1.0, 1.25, 1.5] вместо [0.6 ... 1.3]: полный дневной размах в долях ATR даёт p10 0.58, p25 0.74, медиану 0.99, p75 1.25, p90 1.72 (у UGLD медиана 0.67), а порога достигают 88.2% дней при 0.6, 69.4% при 0.8, 47.2% при 1.0, 22.9% при 1.3 и 14.6% при 1.5 — порог 0.6 на этом инструменте пропускает почти всё и гейтом не является. UseDayATRGate зафиксирован на 1: цена полного отключения гейта меряется отдельно, файлом cal_screen.json, и в ранжирование фазы не подмешивается. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_day.json -out ./reports/DOMRF_day -months 12 -min-trades 10 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "day",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0, 0.25, 0.35],
        "SpentDayATR": [0.8, 1.0, 1.25, 1.5]
      }
    }
  ]
}
```

- [ ] **Step 4: Создать `cal_day_spent.json`**

```json
{
  "_comment": "DAY-SPENT: только ветка «день исчерпан», 6 прогонов. FreshDayATR=0 выключает ветку «день только начался» целиком (dayStateOK защищает её условием fresh > 0, поэтому ровно ноль убирает её), и остаётся чистый свип нижней границы позднего входа. Разделять ветки нужно потому, что это две разные стратегии на одном коде — ранний вход по тренду против возврата после распродажи, — и общий profit factor их усредняет: прибыльная ветка оплачивает убыточную, а лидерборд cal_day.json этого не показывает. Ось шире, чем в cal_day.json, и уходит до 1.75, потому что здесь порог не конкурирует за прогоны с утренней веткой. Доля будних дней, достигающих порога (n=144): 0.6 — 88.2%, 0.8 — 69.4%, 1.0 — 47.2%, 1.25 — 24.3%, 1.5 — 14.6%, 1.75 — 9.0%. Строка 0.6 оставлена именно как контроль «гейт почти выключен»: если она выигрывает, ветка «день исчерпан» на DOMRF не несёт информации. Правый край 1.75 отбирает девять процентов дней — при 8.4 месяцах истории это единицы сделок, и его profit factor читать нельзя, только счёт сделок. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_day_spent.json -out ./reports/DOMRF_day_spent -months 12 -min-trades 5 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "day_spent",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0],
        "SpentDayATR": [0.6, 0.8, 1.0, 1.25, 1.5, 1.75]
      }
    }
  ]
}
```

- [ ] **Step 5: Создать `cal_volume.json`**

```json
{
  "_comment": "VOLUME: фон объёмов при UseVolume=1, 6 прогонов. Замер по кэшу — отношение объёма бара к его слотовой базе за 5 будних дней (n=5801 будних баров): медиана 0.74, p75 1.52, p90 2.85. Доля баров, проходящих гейт: 39.2% при 1.0, 32.3% при 1.2, 25.4% при 1.5, 17.5% при 2.0. Ось [1.2, 1.5, 2.0] та же, что у ugld/, но читать её здесь надо иначе: у UGLD объёмный гейт прореживает избыток сигналов, у DOMRF он режет и без того дефицитную выборку. Кроссов RSI(4) вниз через 15 всего 146 будних за 8.4 месяца — гейт с VolMult=2.0 оставит от них порядка четверти, и это ДО остальных пяти гейтов входа. Поэтому основной результат этого файла — счёт сделок, а profit factor вторичен: конфигурация с лучшим PF и тремя сделками проигрывает конфигурации с PF похуже и пятнадцатью. VolBaseDays [5, 10]: короткая база быстрее реагирует на изменение активности, длинная устойчивее к одиночному всплеску; на инструменте с восьмимесячной историей и растущим оборотом (медиана 422 млн руб./день, p10 203, p90 955) короткая база систематически завышает отношение, потому что база отстаёт от тренда объёмов. UseVolume зафиксирован на 1: точка «гейт выключен» принадлежит cal_screen.json и в ранжирование фазы не подмешивается. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_volume.json -out ./reports/DOMRF_volume -months 12 -min-trades 5 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "volume",
      "grid": {
        "UseVolume": [1],
        "VolMult": [1.2, 1.5, 2.0],
        "VolBaseDays": [5, 10]
      }
    }
  ]
}
```

- [ ] **Step 6: Создать `cal_risk.json`**

```json
{
  "_comment": "RISK: стоп и цель, обе в единицах дневного ATR, обе замораживаются на входе. 16 прогонов. Вся ось перемасштабирована относительно ugld/cal_risk.json, потому что дневной ATR здесь означает другое: по кэшу дневок ATR(14) DOMRF даёт медиану 1.94% от цены (p10 1.44, p25 1.79, p75 2.26, p90 2.55) против 4.28% у UGLD — инструмент вдвое уже и стоит в одном классе с GAZP и T. Строки 0.3 и 0.4, которые ugld/ предлагает, сюда НЕ переносятся, и это не осторожность, а арифметика: 0.3 ATR = 0.58% цены при круге издержек 0.1% (0.05% за сторону), то есть комиссия съедает 17% стопа ещё до первого движения; в единицах риска круг стоит 0.052 ATR против 0.023 на UGLD. Вдобавок медианный будний день DOMRF покрывает 0.99 дневного ATR (p25 0.74, p75 1.25) — стоп в 0.3-0.4 ATR сидит внутри обычного внутридневного шума и снимается сносом, а не провалом сетапа. Верх оси стопа поднят до 1.3, а цели до 2.0 по той же причине узости: чтобы получить те же 2-3% движения, узкому инструменту нужно больше ATR. Цель свипуется и НИЖЕ стопа (0.7 против 1.0 и 1.3): такая асимметрия требует доли выигрышных сделок выше 50% просто чтобы выйти в ноль, и этот файл её либо подтверждает, либо убивает. StopDailyATR=0 намеренно отсутствует и не должен появиться: позиция держится через ночи и выходные, конфигурация без стопа калибровке не предлагается. Смотреть среднее время удержания: стоп 1.3 с целью 2.0 превращает это в позиционную стратегию на недели, а на 8.4 месяцах истории такая пара даст единицы сделок. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_risk.json -out ./reports/DOMRF_risk -months 12 -min-trades 10 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "risk",
      "grid": {
        "StopDailyATR": [0.5, 0.7, 1.0, 1.3],
        "TPDailyATR": [0.7, 1.0, 1.5, 2.0]
      }
    }
  ]
}
```

- [ ] **Step 7: Создать `cal_trail.json`**

```json
{
  "_comment": "TRAIL: трейл-стоп в дневных ATR плюс отключение RSI-выхода, 12 прогонов. Файл честен только ПОСЛЕ cal_risk.json, с зафиксированной парой стоп/цель: трейл становится связывающим стопом лишь когда maxFav - Trail*ATR поднимается выше entry - Stop*ATR, то есть когда цена прошла (Trail - Stop) дневных ATR вверх, и свип поверх baseline с TPDailyATR=0.6 измерял бы трейл, зажатый в окно 0.1 ATR. Ось доходит до 0.8, в отличие от потолка 0.6 в ugld/: там его задавала цель 0.6 ATR, выше которой трейл не успевал взвестись, здесь ось целей поднята до 2.0 и трейл получает пространство. Половины оси читаются по-разному. 0.3-0.4 — это не трейл, а ужатие начального стопа: maxFav стартует с цены входа, поэтому уровень стоит на entry - Trail*ATR с первого же бара, ближе фиксированного стопа 0.5, и выигрыш там является перенастройкой стопа в одежде трейла — его место в cal_risk.json. 0.5 — нейтральная точка, совпадающая с фиксированным стопом на входе. 0.6-0.8 — единственный по-настоящему поздний трейл. TrailDailyATR=0 при UseTrail=1 — это контрольная строка, а не ошибка: desiredStop защищает трейл условием TrailDailyATR > 0, поэтому строка воспроизводит конфигурацию с фиксированным стопом и даёт фазе собственную базу без дублей, которые породила бы ось UseTrail [0,1]. UseRSIExit свипуется рядом, потому что именно RSI-выход обычно не даёт трейлу шанса сработать. Смотреть долю выходов TRAIL, а не только profit factor: набор, где трейл ни разу не сработал, — это старая конфигурация с фиксированным стопом под новым именем. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/domrf/cal_trail.json -out ./reports/DOMRF_trail -months 12 -min-trades 10 -test-months 2 -metric profit_factor.",
  "phases": [
    {
      "name": "trail",
      "grid": {
        "UseRSIExit": [0, 1],
        "UseTrail": [1],
        "TrailDailyATR": [0, 0.3, 0.4, 0.5, 0.6, 0.8]
      }
    }
  ]
}
```

- [ ] **Step 8: Запустить тесты (green)**

Run: `go test ./internal/service/backtest/ -run 'TestDOMRF|TestRSIPullback' -v`
Expected: PASS. В выводе `TestRSIPullbackCalFilesValid` должны появиться все девять подтестов `domrf/cal_*.json`.

- [ ] **Step 9: Полный прогон CI**

Run: `./bin/mage ci`
Expected: зелёный — линт, `go test -race ./...`, mock-drift. Если `./bin/mage` отсутствует, сначала `go run ./magefiles tools` (детали — `docs/tooling/mage.md`).

- [ ] **Step 10: Коммит**

```bash
git add data/params/rsi_pullback/domrf/ internal/service/backtest/rsi_pullback_domrf_grid_test.go
git commit -m "$(cat <<'EOF'
feat(rsi_pullback): сетки гейтов и риска для DOMRF

Пять файлов, 52 прогона. Оси в долях дневного ATR перемасштабированы: у DOMRF
ATR(14) даёт медиану 1.94% против 4.28% у UGLD, поэтому строки стопа 0.3-0.4 не
переносятся (0.3 ATR = 0.58% при круге издержек 0.1%, внутри дневного шума), а
пороги SpentDayATR сдвинуты вверх — медианный день покрывает 0.99 ATR против 0.67.

FreshDayATR 0.2 заменён на 0.25-0.35: к 07:00 MSK медианный день уже прошёл 0.31 ATR.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Разведочные прогоны и маркировка результата

**Files:**
- Create: `reports/DOMRF_screen/*`, `reports/DOMRF_entry/*` (генерируются прогоном)
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1 — абзац про DOMRF)

**Interfaces:**
- Consumes: `data/params/rsi_pullback/domrf/cal_screen.json` и `cal_entry.json` из Task 3, пакет `domrf` из Task 2
- Produces: числа для доклада владельцу

- [ ] **Step 1: Проверить, что токен на месте**

Run: `grep -c T_BANK env/token.env`
Expected: `1`. Прогон требует gRPC-клиента для `resolveShare` даже при свежем кэше свечей. Если токена нет, попросить владельца добавить его в `env/token.env` — сам прогон без этого не стартует.

- [ ] **Step 2: Прогнать `cal_screen.json`**

```bash
go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/domrf/cal_screen.json -out ./reports/DOMRF_screen \
  -months 12 -min-trades 1 -metric profit_factor
```

Ожидается вывод `calibration: ... (combos=4, test_months=0)` и markdown-отчёт в `reports/DOMRF_screen/`. `-refresh` не нужен: кэш `data/candles/DOMRF_*` обновлён 2026-08-04 и покрывает всю историю инструмента с 2025-11-20.

- [ ] **Step 3: Прочитать счёт сделок и принять решение о продолжении**

Из отчёта выписать число сделок для каждой из четырёх комбинаций `UseDayATRGate` × `UseVolume`. Это число, ради которого запускался файл.

Развилка, которую нельзя пройти автоматически: **если при `UseDayATRGate=1` остаётся меньше 6 сделок, дальнейшие темы измеряют отдельные сделки, а не конфигурации.** В этом случае не запускать `cal_entry.json`, а доложить владельцу счёт сделок и остановиться — каталог сеток остаётся заготовкой на будущее, когда у инструмента накопится история.

- [ ] **Step 4: Прогнать `cal_entry.json` (если Step 3 разрешил)**

```bash
go run ./cmd/backtest -ticker DOMRF -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/domrf/cal_entry.json -out ./reports/DOMRF_entry \
  -months 12 -min-trades 10 -test-months 2 -metric profit_factor
```

Ожидается `combos=12`, holdout последних двух месяцев, лидерборд по in-sample и строка результата на holdout.

- [ ] **Step 5: Дописать абзац про DOMRF в §8.0.1**

В `docs/rsi_pullback/strategy.md`, после таблицы состояний §8.0.1, добавить (числа `<...>` заменить фактическими из отчётов Step 2 и Step 4):

```markdown
**DOMRF — разведка, не калибровка.** ДОМ.РФ вышел на биржу 2025-11-20, и вся его история на
2026-08-04 — 8.4 месяца (8152 30-минутных бара, 158 будних дневок). Протокол walk-forward
выше неисполним физически: он требует окна train в 12 месяцев. Вместо него взят holdout
последних 2 месяцев — одна точка OOS. Вдобавок весь доступный период является пост-IPO
аптрендом (+29.9% от 1749.8 до 2273.2 ₽) без нисходящего режима, а long-only покупка отката
в таком режиме показывает завышенный profit factor при любых параметрах. Поэтому результаты
прогонов `domrf/cal_screen.json` (<N> сделок при включённых гейтах) и `domrf/cal_entry.json`
(<PF> in-sample, <PF_HO> на holdout) — разведочные: они выбирают, куда смотреть дальше, и не
являются доказательством edge. Тикер остаётся baseline-tracking до накопления 18–24 месяцев
истории, то есть примерно до середины 2027 года.
```

- [ ] **Step 6: Коммит**

```bash
git add reports/DOMRF_screen reports/DOMRF_entry docs/rsi_pullback/strategy.md
git commit -m "$(cat <<'EOF'
chore(rsi_pullback): разведочные прогоны DOMRF (screen + entry)

Не калибровка: истории 8.4 месяца с IPO 2025-11-20, walk-forward по §8
неисполним, взят holdout 2 месяца — одна точка OOS. Весь период является
пост-IPO аптрендом +29.9%, в котором long-only покупка отката показывает
завышенный PF при любых параметрах.

§8.0.1 документации называет эти рамки явно, чтобы числа из отчётов не были
позже прочитаны как подтверждение edge.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7: Доложить владельцу**

Короткий отчёт: счёт сделок по четырём комбинациям гейтов, лидерборд `cal_entry` (in-sample PF и число сделок), результат на holdout. Явно сказать, подтверждает ли счёт сделок продолжение калибровки по остальным семи темам или каталог остаётся заготовкой.

---

## Self-Review

**Spec coverage.** Раздел «Блокер: сборка сломана» → Task 1. Раздел 1 «Реестр» → Task 1 (шаги 2–3, 5). Раздел 2 «Пакет `domrf`» → Task 2. Раздел 3 «Каталог сеток», таблица из девяти файлов → Task 3 (screen, entry, trend, exit) и Task 4 (day, day_spent, volume, risk, trail); все четыре обоснования отклонений от `ugld/` попали в `_comment` соответствующих файлов и в пин-тесты. Раздел 4 «Разведочный прогон» → Task 5 (шаги 2–4). Раздел 5 «Маркировка результата» → маркировка в `_comment` (Task 3–4), в §8.0.1 (Task 5 шаг 5), в сообщениях коммитов (Task 3–5). Раздел «Что НЕ делается» → отражён в Global Constraints (запрет литерала) и в развилке Task 5 шаг 3 (остановка вместо полной калибровки). Раздел «Приёмка» → Task 4 шаг 9 (`mage ci`), Task 1 шаг 4 и Task 3 шаг 7 (тесты). Раздел «Известное расхождение» — намеренно вне плана, это follow-up.

**Сумма прогонов.** Task 3: 4 + 12 + 16 + 5 = 37. Task 4: 12 + 6 + 6 + 16 + 12 = 52. Итого 89 — сходится со спекой.

**Type consistency.** `domrfGrid(t, file)` определён в Task 3 шаг 1 и вызывается в Task 4 шаг 1 с той же сигнатурой. `rsiPullbackPhases` и `rsiPullbackParamsDir` — существующие идентификаторы из `rsi_pullback_grid_test.go`, тот же пакет `backtest`. `domrf.Ticker` и `domrf.DefaultParams` из Task 2 используются в реестре там же. Имена полей сеток (`RSIPeriod`, `RSILower`, `RSIUpper`, `EMAFast`, `EMASlow`, `UseDayATRGate`, `FreshDayATR`, `SpentDayATR`, `UseVolume`, `VolMult`, `VolBaseDays`, `StopDailyATR`, `TPDailyATR`, `UseRSIExit`, `UseTrail`, `TrailDailyATR`) сверены с `core.Params`.
