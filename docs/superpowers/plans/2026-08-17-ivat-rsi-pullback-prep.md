# IVAT под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** довести IVAT до вердикта по стратегии `rsi_pullback`: каталог сеток с осями по замерам
инструмента, пакет параметров, девять тем rolling walk-forward, литерал и решение о боевой
вселенной.

**Architecture:** тикер-специфичный каталог `data/params/rsi_pullback/ivat/` (девять однотемных
файлов) + сторожевой тест осей в `internal/service/backtest` + пакет параметров
`strategy/ivat` (сначала baseline-состояние, затем литерал со снимком) + записи в двух реестрах
(бэктест и живой раннер) + разделы документации. Прогоны идут по адаптированной схеме
`-months 25 -train-months 9 -test-months 4`, потому что истории у инструмента 26 месяцев.

**Tech Stack:** Go 1.25, `cmd/backtest` (grid-search + rolling walk-forward), `go test`,
`./bin/golangci-lint`, `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-08-17-ivat-rsi-pullback-prep-design.md`

## Global Constraints

- Схема прогонов для IVAT — **`-months 25 -train-months 9 -test-months 4 -metric profit_factor`**,
  `-min-trades 20` (у темы `screen` — `-min-trades 1`). Штатная схема каталога
  (`-months 36 -train-months 12 -test-months 6`) к этому тикеру НЕ применяется: истории 26.0 мес.
- Планка, объявленная ДО прогонов и не пересматриваемая после: темы `entry` и `trend` **обе**
  дают pooled OOS PF ≥ 1.5 при ≥ 20 сделках; ведущая ось темы (`RSILower` для `entry`, `EMASlow`
  для `trend`) выбрана одинаково в ≥ 3 фолдах из 4; в каждом фолде OOS ≥ 5 сделок. Вырожденный
  фолд (ни одной убыточной сделки) не засчитывается в пользу тикера.
- Правило прода: литерал ставится и тикер заводится в `RSI_PULLBACK_TICKERS` даже при непройденной
  планке. **Стоп-условие:** pooled OOS PF < 1.0 либо < 20 сделок за 25 месяцев → остановиться,
  принести числа владельцу, в прод не заводить.
- Кэш освежён 2026-08-17: `IVAT_Minutes30.json` — 26 979 баров, 18 772 будних, окно
  2024-06-18 … 2026-08-17 (26.0 мес); `IVAT_Day1.json` — 651 дневная свеча. Все замеры ниже — по
  нему; повторный `-refresh` во время калибровки НЕ запускать (сдвинет окно и рассинхронизирует
  числа с документацией).
- Замеры инструмента, на которые ссылаются `_comment` сеток (все — будние бары):
  - кроссы RSI вниз: RSI(4) 178/406/688/1103/1540/2030/2404 на уровнях 10/15/20/25/30/35/40;
    RSI(5) 96/222/443/751/1151/1678/2085; RSI(6) 50/138/302/541/895/1365/1822;
    RSI(7) 31/92/192/411/718/1122/1630;
  - кроссы RSI вверх (полоса выхода), RSI(4): 55 — 2299, 60 — 1919, 65 — 1579, 70 — 1210,
    75 — 893, 80 — 614;
  - доля баров с `EMAFast > EMASlow`: 29.4–36.0% на всех двадцати парах (10/50 — 36.0%,
    20/100 — 33.5%, 10/150 — 32.2%, 30/200 — 29.5%, 40/200 — 29.4%);
  - дневной ATR(14): медиана 3.59% цены, p10 2.21, p90 6.98, n=651; круг издержек 0.1% = 0.028 ATR;
  - выживаемость стопа (доля дней, чей размах достаёт уровня): 0.6 — 65.7%, 0.8 — 49.4%,
    0.9 — 44.2%, 1.0 — 37.4%, 1.25 — 24.8%, 1.5 — 16.5%;
  - день ко второму бару: медиана 0.39 ATR; ветка «свежий день» ловит 7.5% баров при 0.3, 15.3%
    при 0.4, 23.5% при 0.5; ветка «день исчерпан» — 67.6% при 0.6, 50.4% при 0.8, 43.9% при 0.9,
    37.9% при 1.0, 26.1% при 1.25, 18.7% при 1.5;
  - объёмный гейт (доля баров, проходящих порог): база 14 дней — 29.2% при 1.0, 25.0% при 1.2,
    20.5% при 1.5, 15.8% при 2.0; база 3 дня — 38.2 / 33.4 / 27.9 / 22.0%;
  - оборот: медиана 43 млн ₽/день, p10 7 млн, p90 360 млн;
  - режим: train −45.6%, holdout −58.8%, вся история −77.5%, пик-минимум −87.6%, все пять
    полугодий отрицательные;
  - контрольный baseline-прогон на 26 мес: 143 сделки, PF 1.432.
- Отчёты прогонов пишутся в `./reports/IVAT_<тема>` (каталог `reports/` в `.gitignore`).
- Коммиты делать по завершении каждой задачи; сообщения на русском, в стиле существующей истории
  (`feat(rsi_pullback): ...`). Ветка — текущая `feat/wush-pullback-prep`.

---

### Task 1: Каталог сеток IVAT со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/ivat/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`, `cal_risk.json`, `cal_exit.json`,
  `cal_trail.json`
- Test: `internal/service/backtest/rsi_pullback_ivat_grid_test.go`

**Interfaces:**
- Consumes: хелперы `rsiPullbackTickerGrid(t, ticker, file)` и `sameSet(got, want...)` из
  `internal/service/backtest/rsi_pullback_grid_test.go` и `rsi_pullback_reni_grid_test.go`.
- Produces: каталог `ivat/`, на который ссылаются все прогоны Task 3–11, и функция
  `ivatGrid(t, file)` для последующих проверок.

- [ ] **Step 1: Написать падающий тест осей**

Создать `internal/service/backtest/rsi_pullback_ivat_grid_test.go`:

```go
package backtest

import "testing"

// ivatGrid читает файл сеток IVAT через общий хелпер.
func ivatGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "ivat", file)
}

// TestIVATGridsPinTheirMeasuredAxes сторожит оси каталога ivat/. Каталог собран 2026-08-17
// копированием формы nvtk/ с пересадкой каждой оси на замеры самого IVAT (26 979 30-минутных
// баров, 18 772 будних, 26.0 месяца с 2024-06-18). Две особенности инструмента, из-за которых
// чужие обоснования сюда не переносятся: истории всего 26 месяцев (схема прогонов адаптирована
// до train 9 / OOS 4) и трендовый фильтр открыт 29.4-36.0% времени — самый закрытый в каталоге.
func TestIVATGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := ivatGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := ivatGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: на IVAT он живой — RSI(4)@10 даёт 178 будних кроссов за 26
	// месяцев, слабейший угол RSI(7)@10 — 31, на уровне LSNGP (29) и RENI (23).
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на IVAT это живой угол (178 кроссов RSI(4)@10)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}

	trend := ivatGrid(t, "cal_trend.json")
	// Ось тренда режется только вместе с этим тестом: на IVAT доля баров с EMAFast > EMASlow
	// укладывается в 29.4-36.0% на всех двадцати замеренных парах, и зависимость монотонна —
	// чем медленнее EMASlow, тем уже допуск. Мёртвых пар в сетке нет, сужать её не за что.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — медленный край оси меряет, во что обходится самый узкий допуск (29.4%%)", trend["EMASlow"])
	}
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json: пара EMAFast=%v EMASlow=%v вырождена — быстрая обязана быть быстрее медленной", fast, slow)
			}
		}
	}

	risk := ivatGrid(t, "cal_risk.json")
	// Строка 0.3 сохранена намеренно: при дневном ATR 3.59% круг издержек 0.1% съедает 9.4%
	// риска — заметно ниже черты 17%, по которой её вырезали из domrf/.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на IVAT комиссия съедает там 9.4%% риска, строка живая", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: стоп обязан быть положительным", v)
		}
	}

	day := ivatGrid(t, "cal_day.json")
	// Ветка «свежий день» ловит 7.5% будних баров при пороге 0.3 и 23.5% при 0.5; ноль в оси —
	// это выключенная ветка, и она обязана остаться, потому что на всех прод-тикерах каталога
	// победил именно ноль.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — выключенная ветка обязана быть в сетке", day["FreshDayATR"])
	}

	exit := ivatGrid(t, "cal_exit.json")
	// Полоса выхода рабочая на всём диапазоне: кроссов вверх RSI(4) от 2299 (55) до 614 (80).
	for _, v := range exit["RSIUpper"] {
		if v <= 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: выход обязан стоять выше средней линии", v)
		}
	}
}

// containsValue сообщает, есть ли значение в оси.
func containsValue(axis []float64, want float64) bool {
	for _, v := range axis {
		if v == want {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestIVATGridsPinTheirMeasuredAxes -v`
Expected: FAIL — каталога `data/params/rsi_pullback/ivat/` ещё нет, хелпер падает на чтении файла.

- [ ] **Step 3: Создать девять файлов сеток**

Каждый файл — объект с `_comment` и массивом `phases`. Ниже даны сами сетки; `_comment` пишется
по образцу `nvtk/`: что тема меряет, из какого замера получена каждая ось, команда запуска (со
схемой `-months 25 -train-months 9 -test-months 4`), и оставленное место под строку
«РЕЗУЛЬТАТ ПРОГОНА 2026-08-17», которую заполняют задачи 3–11.

`cal_screen.json` — 4 прогона:
```json
{"phases": [{"name": "screen", "grid": {"UseDayATRGate": [0, 1], "UseVolume": [0, 1]}}]}
```

`cal_entry.json` — 168 прогонов:
```json
{"phases": [{"name": "entry", "grid": {
  "RSIUpper": [55, 60, 65, 70, 75, 80],
  "RSIPeriod": [4, 5, 6, 7],
  "RSILower": [10, 15, 20, 25, 30, 35, 40]
}}]}
```

`cal_trend.json` — 35 прогонов:
```json
{"phases": [{"name": "trend", "grid": {
  "EMAFast": [5, 10, 20, 30, 40],
  "EMASlow": [50, 70, 100, 120, 150, 170, 200]
}}]}
```

`cal_day.json` — 24 прогона:
```json
{"phases": [{"name": "day", "grid": {
  "UseDayATRGate": [1],
  "FreshDayATR": [0, 0.3, 0.4, 0.5],
  "SpentDayATR": [0.6, 0.8, 0.9, 1.0, 1.25, 1.5]
}}]}
```

`cal_day_spent.json` — 7 прогонов:
```json
{"phases": [{"name": "day_spent", "grid": {
  "UseDayATRGate": [1],
  "FreshDayATR": [0],
  "SpentDayATR": [0.6, 0.8, 0.9, 1.0, 1.25, 1.3, 1.5]
}}]}
```

`cal_volume.json` — 16 прогонов:
```json
{"phases": [{"name": "volume", "grid": {
  "UseVolume": [1],
  "VolMult": [1.0, 1.2, 1.5, 2.0],
  "VolBaseDays": [3, 5, 10, 14]
}}]}
```

`cal_risk.json` — 35 прогонов:
```json
{"phases": [{"name": "risk", "grid": {
  "StopDailyATR": [0.3, 0.5, 0.7, 1.0, 1.3],
  "TPDailyATR": [0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5]
}}]}
```

`cal_exit.json` — 6 прогонов:
```json
{"phases": [{"name": "exit", "grid": {"RSIUpper": [55, 60, 65, 70, 75, 80]}}]}
```

`cal_trail.json` — 12 прогонов:
```json
{"phases": [{"name": "trail", "grid": {
  "UseRSIExit": [0, 1],
  "UseTrail": [1],
  "TrailDailyATR": [0, 0.3, 0.5, 0.7, 1.0, 1.3]
}}]}
```

- [ ] **Step 4: Запустить тесты пакета целиком**

Run: `go test ./internal/service/backtest/ -run 'RSIPullback|IVAT' -v`
Expected: PASS, включая общий `TestRSIPullbackGridFiles*` (он обходит каталог рекурсивно и
проверит новые файлы наравне со старыми).

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/ivat internal/service/backtest/rsi_pullback_ivat_grid_test.go
git commit -m "feat(rsi_pullback): сетки IVAT с замеренными осями"
```

---

### Task 2: Пакет `strategy/ivat` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/ivat/ivat.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/ivat/ivat_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт + запись в карту)
- Test: `internal/service/backtest/rsi_pullback_registry_test.go` (сторожевой тест baseline)

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `ivat.Ticker` (константа `"IVAT"`) и `ivat.DefaultParams() core.Params` — их
  используют Task 12 (литерал) и Task 13 (реестр живого раннера).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/ivat/ivat_test.go`:

```go
package ivat

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Пакет заведён 2026-08-17 ДО калибровки: он обязан отслеживать baseline, чтобы правка дефолтов
// доходила до тикера, а не расходилась с ним молча. Тест держит это состояние и подлежит замене
// снимком литерала ровно тогда, когда калибровка закончится (задача 12 плана).
func TestParamsTrackTheBaseline(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("IVAT ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsIVAT(t *testing.T) {
	if Ticker != "IVAT" {
		t.Fatalf("Ticker = %q, want IVAT", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/ivat/ -v`
Expected: FAIL — пакета нет, сборка не проходит.

- [ ] **Step 3: Создать пакет**

`internal/service/trading_strategy/rsi_pullback/strategy/ivat/ivat.go`:

```go
// Package ivat supplies the ticker and rsi_pullback Params for IVAT (ИВА Технологии).
//
// СОСТОЯНИЕ: калибровка не проводилась. Пакет возвращает core.DefaultParams(), то есть
// отслеживает baseline: правка дефолтов ядра доходит до этого тикера. Ставить IVAT в боевую
// вселенную RSI_PULLBACK_TICKERS в таком состоянии нельзя — торговля пошла бы параметрами,
// которые на этом инструменте никогда не проверялись.
//
// Что известно об инструменте до прогонов (кэш 2026-08-17, 26 979 30-минутных баров, 18 772
// будних, 26.0 месяца с 2024-06-18):
//
//   - ИСТОРИИ 26 МЕСЯЦЕВ, и это главное ограничение. Штатный протокол §8
//     docs/rsi_pullback/strategy.md (-months 36 -train-months 12 -test-months 6) неисполним:
//     четырёх фолдов на такой истории нет. Решением владельца схема адаптирована до
//     -months 25 -train-months 9 -test-months 4 — четыре фолда встык при обучающем окне на
//     четверть короче штатного. Числа IVAT из-за этого не сопоставимы построчно с остальным
//     каталогом.
//   - РЕЖИМ: падают ОБА окна протокола (train −45.6%, holdout −58.8%), вся история −77.5%,
//     пик-минимум −87.6%, и ни одного растущего полугодия за всю историю. Это зеркало LSNGP,
//     у которого нет ни одного падающего окна: завысить лонговый результат режимом здесь
//     нечем, но и на растущем рынке конфигурация не проверена ни разу.
//   - ТРЕНДОВЫЙ ФИЛЬТР — самый закрытый в каталоге: доля баров с EMAFast > EMASlow укладывается
//     в 29.4-36.0% на всех двадцати замеренных парах (у WUSH 41-43%, у UGLD 45-47%, у LSNGP
//     54-60%), и зависимость монотонна — медленные пары режут выборку сильнее.
//   - Дневной ATR(14) идёт медианой 3.59% цены (p10 2.21, p90 6.98) — середина каталога.
//     Круг издержек 0.1% оборота стоит 0.028 ATR, то есть на стопе 0.3 ATR комиссия съедает
//     9.4% риска (черта, по которой строку 0.3 вырезали из domrf/, — 17%).
//   - ЛИКВИДНОСТЬ: оборот медианой 43 млн ₽/день при p10 = 7 млн — уровень LENT (38) и LSNGP
//     (41). Половина дней тоньше гейта отбора скринера в 50 млн: это ограничение исполнения,
//     а не статистики.
//   - Контрольный прогон baseline на 26 месяцах: 143 сделки, PF 1.432 — инструмент торгует,
//     дефицита сделок уровня «стратегия молчит» здесь нет.
//
// Сетки калибровки лежат в data/params/rsi_pullback/ivat/, их оси прибиты
// internal/service/backtest/rsi_pullback_ivat_grid_test.go.
package ivat

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "IVAT"

// DefaultParams returns the strategy baseline: IVAT is not calibrated yet.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт
`rsipullbackivat "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ivat"` и строку
карты:

```go
	rsipullbackivat.Ticker:  rsiPullbackBindingFor(rsipullbackivat.Ticker, rsipullbackivat.DefaultParams),
```

- [ ] **Step 5: Добавить сторожевой тест baseline в реестр бэктеста**

В `internal/service/backtest/rsi_pullback_registry_test.go`:

```go
// TestRSIPullbackIVATTracksBaseline держит состояние «калибровка не проводилась»: пакет
// strategy/ivat заведён 2026-08-17 под будущий литерал, и до конца калибровки обязан возвращать
// core.DefaultParams(). Тест заменяется снимком литерала в тот день, когда литерал появится, —
// ровно так это было с reni, fesh, wush, lent, lsngp и nvtk.
func TestRSIPullbackIVATTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["IVAT"]
	if !ok {
		t.Fatal("IVAT отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("IVAT: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("IVAT отклонился от baseline до калибровки: %+v", p)
	}
	if got := b.Build(p).Ticker(); got != "IVAT" {
		t.Fatalf("Ticker() = %q, want IVAT", got)
	}
}
```

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'IVAT|RSIPullback' -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/ivat internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): пакет и реестр IVAT в состоянии «калибровка не проводилась»"
```

---

### Task 3: Тема `screen` — цена двух опциональных гейтов

**Files:**
- Modify: `data/params/rsi_pullback/ivat/cal_screen.json` (дописать результат в `_comment`)

**Interfaces:**
- Consumes: каталог сеток из Task 1, пакет из Task 2.
- Produces: знание, сколько сделок стоит каждый гейт — им пользуются задачи 6 и 7 при разборе.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_screen.json -out ./reports/IVAT_screen \
  -months 25 -train-months 9 -test-months 4 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Прочитать отчёт**

Открыть свежий файл в `./reports/IVAT_screen/`. Выписать: pooled OOS PF и счёт сделок каждой из
четырёх комбинаций, выбор калибратора по фолдам, и во сколько сделок обходится каждый гейт.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать в `_comment` файла `cal_screen.json` строку вида
`РЕЗУЛЬТАТ ПРОГОНА 2026-08-17: pooled OOS PF <...> на <...> сделках, фолды <...>; гейт дня стоит
<...> сделок, объёмный — <...>.` Числа — фактические из отчёта, без округления в свою пользу.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ivat/cal_screen.json
git commit -m "feat(rsi_pullback): IVAT, тема screen прогнана"
```

---

### Task 4: Тема `entry` — ключевая, полоса RSI целиком

**Files:**
- Modify: `data/params/rsi_pullback/ivat/cal_entry.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: первое из двух чисел планки (pooled OOS PF темы `entry`, счёт сделок, устойчивость
  `RSILower` по фолдам) — Task 12 выносит по ним вердикт.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_entry.json -out ./reports/IVAT_entry \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать три числа планки**

Из отчёта: pooled OOS PF, счёт сделок пула, PF и счёт сделок каждого из четырёх фолдов, выбор
`RSIPeriod` / `RSILower` / `RSIUpper` по каждому фолду. Отдельно отметить фолды с числом сделок
меньше 5 и вырожденные (без убыточных сделок) — планка их не засчитывает.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Формат тот же, что у `nvtk/cal_entry.json`: `РЕЗУЛЬТАТ ПРОГОНА 2026-08-17: pooled OOS PF <...> на
<...> сделках, фолды <...> — порог 1.5 <взят|не взят>. Ведущая ось RSILower выбрана <...> —
устойчивость <N> из 4.`

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ivat/cal_entry.json
git commit -m "feat(rsi_pullback): IVAT, тема entry прогнана"
```

---

### Task 5: Тема `trend` — вторая ключевая

**Files:**
- Modify: `data/params/rsi_pullback/ivat/cal_trend.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: второе число планки (pooled OOS PF темы `trend`, устойчивость `EMASlow`).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_trend.json -out ./reports/IVAT_trend \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа и проверить дефицит выборки**

Кроме pooled PF, фолдов и выбора `EMAFast`/`EMASlow`, отдельно выписать счёт сделок каждого
фолда: при допуске фильтра 29.4–36.0% времени именно здесь выборка разваливается первой. Фолд
меньше 5 сделок — провал третьего условия планки.

- [ ] **Step 3: Записать результат в `_comment` сетки**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ivat/cal_trend.json
git commit -m "feat(rsi_pullback): IVAT, тема trend прогнана"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

**Files:**
- Modify: `data/params/rsi_pullback/ivat/cal_day.json`, `cal_day_spent.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3 (цена гейта в сделках).
- Produces: значения `FreshDayATR` и `SpentDayATR` для литерала Task 12.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_day.json -out ./reports/IVAT_day \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_day_spent.json -out ./reports/IVAT_day_spent \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить ветку «свежий день» со своим замером**

Ветка ловит 7.5% баров при пороге 0.3 и 23.5% при 0.5. Если калибратор выбирает ненулевой
`FreshDayATR`, проверить, что прирост PF не куплен обвалом счёта сделок; на всех прод-тикерах
каталога победил ноль.

- [ ] **Step 3: Записать результаты в оба `_comment`**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ivat/cal_day.json data/params/rsi_pullback/ivat/cal_day_spent.json
git commit -m "feat(rsi_pullback): IVAT, темы дневного гейта прогнаны"
```

---

### Task 7: Тема `volume` — фон объёмов

**Files:**
- Modify: `data/params/rsi_pullback/ivat/cal_volume.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3.
- Produces: решение о `UseVolume` для литерала Task 12.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_volume.json -out ./reports/IVAT_volume \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить на вырождение фолда**

На GAZP и NVTK объёмный гейт покупал pooled PF вырожденным фолдом (17.146 на 19 сделках). Если
здесь повторится — гейт отвергается, и причина записывается числом, а не мнением.

- [ ] **Step 3: Записать результат в `_comment`**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ivat/cal_volume.json
git commit -m "feat(rsi_pullback): IVAT, тема volume прогнана"
```

---

### Task 8: Тема `risk` — стоп и цель

**Files:**
- Modify: `data/params/rsi_pullback/ivat/cal_risk.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `StopDailyATR` и `TPDailyATR` для литерала Task 12.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_risk.json -out ./reports/IVAT_risk \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить ось стопа на монотонность**

Капкан, разобранный на WUSH, LENT и LSNGP: profit factor растёт монотонно с шириной стопа, а доля
выходов по стопу падает — это вытеснение убытков в RSI-выход, а не улучшение. Выписать долю
стоп-выходов для каждой точки оси. Выживаемость стопа на IVAT: 0.6 ATR переживается 34.3% дней,
1.0 — 62.6%, 1.3 — примерно 77%.

- [ ] **Step 3: Записать результат в `_comment`**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ivat/cal_risk.json
git commit -m "feat(rsi_pullback): IVAT, тема risk прогнана"
```

---

### Task 9: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/ivat/cal_exit.json`, `cal_trail.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` для литерала Task 12.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_exit.json -out ./reports/IVAT_exit \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/cal_trail.json -out ./reports/IVAT_trail \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить характер сделки при выбранной полосе**

На LSNGP `RSIUpper` 55 уронил медиану удержания до 4 баров — многодневная стратегия стала
внутридневной. Если тема выбирает низкую полосу, выписать медиану удержания и долю сделок длиннее
одного торгового дня: это плата, которую надо назвать явно.

- [ ] **Step 3: Записать результаты в оба `_comment`**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ivat/cal_exit.json data/params/rsi_pullback/ivat/cal_trail.json
git commit -m "feat(rsi_pullback): IVAT, темы выходов прогнаны"
```

---

### Task 10: Сборка литерала и точечный walk-forward принятой точки

**Files:**
- Create: `data/params/rsi_pullback/ivat/plateau_point.json`
- Create (по необходимости): `data/params/rsi_pullback/ivat/plateau_<сосед>.json` для соседей,
  которые стоит держать файлами

**Interfaces:**
- Consumes: результаты задач 3–9.
- Produces: конкретный набор из восемнадцати полей `core.Params` и его замеры — их прибивает
  Task 11.

- [ ] **Step 1: Собрать точку из лидербордов тем**

Взять по каждой теме её выбор. Где тема мерила ось поверх дефолтов, стоящих вне рабочей зоны
инструмента (случай NVTK, где дефолтный дневной гейт стоял там, где стратегии нет), — проверить
ось точечными прогонами и записать, что выбор расходится с темой и почему.

- [ ] **Step 2: Создать файл точки**

`plateau_point.json` — сетка из одного значения по каждому полю, формат тот же, что у
`nvtk/plateau_point.json`: одна фаза, каждый ключ — массив из одного элемента.

- [ ] **Step 3: Прогнать точку тем же walk-forward**

```bash
go run ./cmd/backtest -ticker IVAT -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ivat/plateau_point.json -out ./reports/IVAT_point \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 4: Проверить стоп-условие плана**

Если pooled OOS PF < 1.0 или сделок меньше 20 — остановиться, вынести числа владельцу, задачи
11–14 не выполнять до его решения.

- [ ] **Step 5: Замерить плато соседями**

По каждой оси прогнать соседние значения точечно (`-params`) и выписать pooled PF: плато шириной
в один шаг — это пик, а не полка, и в доке пакета это должно быть названо (случай UGLD, где
`RSILower` 20 роняет точку с 3.627 до 1.580).

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/ivat/plateau_point.json
git commit -m "feat(rsi_pullback): IVAT, принятая точка и её замеры"
```

---

### Task 11: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/ivat/ivat.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/ivat/ivat_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (заменить baseline-тест
  снимком)

**Interfaces:**
- Consumes: набор полей из Task 10.
- Produces: `ivat.DefaultParams()`, возвращающий литерал, — его читают Task 13 (реестр раннера)
  и Task 14 (вселенная).

- [ ] **Step 1: Заменить тест baseline снимком литерала**

В `ivat_test.go` удалить `TestParamsTrackTheBaseline` и написать снимок по образцу `nvtk_test.go`:
`TestCalibratedLiteralIsPinned` со всеми восемнадцатью полями, `TestParamsDoNotTrackTheBaseline`,
`TestStopIsArmed`, `TestRSIExitIsArmed`, `TestTargetClearsTheStop`. Каждый тест несёт в комментарии
замер, объясняющий, почему поле именно такое.

- [ ] **Step 2: Запустить и убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/ivat/ -v`
Expected: FAIL — `DefaultParams()` ещё возвращает baseline.

- [ ] **Step 3: Поставить литерал**

В `ivat.go` заменить `return core.DefaultParams()` литералом из Task 10 и переписать доку пакета:
раздел «СОСТОЯНИЕ: калибровка не проводилась» → разбор калибровки (результат девяти тем, вердикт
по планке пункт за пунктом, разбор каждого поля литерала, граница приёма, замеры инструмента).

- [ ] **Step 4: Заменить сторожевой тест в реестре бэктеста**

`TestRSIPullbackIVATTracksBaseline` → `TestRSIPullbackIVATIsRegisteredAndCalibrated` по образцу
теста LSNGP: проверяет наличие в карте, несовпадение с baseline, равенство литералу пакета и
`Ticker()`.

- [ ] **Step 5: Запустить тесты и линт**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ && ./bin/golangci-lint run ./internal/service/...`
Expected: PASS, 0 issues.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/ivat internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): IVAT откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 12: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`

**Interfaces:**
- Consumes: `ivat.Ticker`, `ivat.DefaultParams()` из Task 11.
- Produces: `ParamsFor("IVAT")` и `StrategyFor("IVAT")`, без которых раннер тикер не увидит.

- [ ] **Step 1: Добавить импорт и запись в карту**

```go
	ivat.Ticker:  ivat.DefaultParams(),
```

- [ ] **Step 2: Дописать комментарий карты**

Абзац про IVAT: адаптированная схема прогонов и почему (26 месяцев истории), вердикт по планке,
режим без единого растущего полугодия, ликвидность 43 млн ₽/день медианой.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
Expected: PASS — включая `TestRegisteredTickersKeepTheRSIExitArmed`, который обходит всю карту.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live/registry.go
git commit -m "feat(rsi_pullback): IVAT в реестре живого раннера"
```

---

### Task 13: Заведение в боевую вселенную

**Files:**
- Modify: `internal/config/rsi_pullback.go` (дефолт `Tickers` + комментарий)
- Modify: `internal/config/rsi_pullback_test.go` (ожидание дефолта)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Modify: `docs/rsi_pullback/live.md` (таблица §8, раздел про реестр, §9 порядок выката)

**Interfaces:**
- Consumes: литерал из Task 11, запись реестра из Task 12.
- Produces: боевую вселенную из одиннадцати тикеров.

- [ ] **Step 1: Проверить стоп-условие**

Если Task 10 остановился на стоп-условии — эта задача не выполняется. Иначе продолжать.

- [ ] **Step 2: Обновить тест дефолта**

```go
	want := []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT"}
```

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -run TestNewRSIPullbackConfig_Defaults -v`
Expected: FAIL — дефолт ещё из десяти тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

`Tickers: []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT"}`
и та же строка в `RSI_PULLBACK_TICKERS` трёх env-файлов. В комментарий функции дописать абзац про
IVAT с типом его риска.

- [ ] **Step 5: Обновить live.md**

Таблица §8 (дефолт переменной), раздел про реестр («знает одиннадцать пакетов»), §9 пункт 1
(добавить IVAT в список сверки `pullparity`).

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS — `TestEveryDefaultTickerIsRegistered` и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` читают вселенную из конфига и покроют
новый состав автоматически.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести IVAT в боевую вселенную"
```

---

### Task 14: Документация — разбор калибровки и принятый риск

**Files:**
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1 строка каталога + раздел с разбором прогонов)
- Modify: `docs/rsi_pullback/live.md` (§10, риск 14)

**Interfaces:**
- Consumes: числа задач 3–10, решение задачи 13.
- Produces: справочник, по которому тикер сопровождают в живой торговле.

- [ ] **Step 1: Дописать §8.0.1 `strategy.md`**

В ячейку «откалиброван» добавить `ivat` с датой, схемой прогонов (и оговоркой, что она
адаптирована), вердиктом по планке, замерами принятой точки и ссылкой на риск в `live.md`.

- [ ] **Step 2: Дописать раздел с разбором прогонов**

По образцу разделов GAZP, NVTK и UGLD: рамки данных, режим, вердикт по планке пункт за пунктом,
разбор каждого поля литерала, граница приёма («для фиксированной точки это НЕ out-of-sample»).

- [ ] **Step 3: Дописать риск 14 в `live.md` §10**

Замеры, практические следствия для наблюдения (распределение выходов, медиана удержания,
просадка), два ограничения — короткая история и ликвидность 43 млн ₽/день, условия пересмотра.

- [ ] **Step 4: Проверить, что доки не противоречат числам**

Run: `grep -n "IVAT" docs/rsi_pullback/*.go docs/rsi_pullback/*.md internal/service/trading_strategy/rsi_pullback/strategy/ivat/*.go`
Сверить каждое число с отчётами прогонов.

- [ ] **Step 5: Коммит**

```bash
git add docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки IVAT и принятый риск"
```

---

### Task 15: Финальная проверка

**Files:** нет изменений, только проверки.

- [ ] **Step 1: Полный гейт качества**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + проверка дрейфа моков — зелёные.

- [ ] **Step 2: Сверка живой сборки с бэктестом**

```bash
go run ./cmd/pullparity -tickers IVAT -months 25
```
Expected: ноль расхождений. Расхождение означает, что живой раннер и бэктест считают сигнал
по-разному, и заведение в прод откатывается до выяснения.

- [ ] **Step 3: Сообщить владельцу результат**

Короткий отчёт: вердикт по планке, замеры принятой точки, что заведено в прод, какие риски
записаны, что осталось (первые живые сделки, условия пересмотра).

---

## Self-review

**Покрытие спеки.** Рамки данных → Global Constraints; адаптированная схема → Global Constraints и
каждая команда прогона; режим и трендовая ось → доки пакета (Task 2, Task 11) и риск 14 (Task 14);
оси девяти сеток → Task 1; планка → Global Constraints, вердикт выносится в Task 11 и Task 14;
правило прода и стоп-условие → Task 10 Step 4 и Task 13 Step 1; артефакты 1-6 спеки → задачи 1, 2,
11, 12, 13, 14; порядок работы спеки → порядок задач.

**Плейсхолдеры.** В задачах 3–10 значения результатов прогонов не выдуманы: там, где число станет
известно только после запуска, стоит явное указание выписать его из отчёта — это не плейсхолдер
плана, а данные, которых до прогона не существует. Код всех тестов и структуры всех сеток даны
целиком.

**Согласованность типов.** `ivat.Ticker` (строка `"IVAT"`) и `ivat.DefaultParams() core.Params`
объявлены в Task 2 и используются под теми же именами в задачах 11, 12, 13. Хелпер `ivatGrid` и
`containsValue` объявлены в Task 1 и там же используются. Имена тестов, заменяемых на следующем
шаге (`TestParamsTrackTheBaseline` → снимок, `TestRSIPullbackIVATTracksBaseline` →
`TestRSIPullbackIVATIsRegisteredAndCalibrated`), названы в обеих задачах одинаково.
