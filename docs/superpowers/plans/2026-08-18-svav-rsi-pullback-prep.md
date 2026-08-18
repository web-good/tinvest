# SVAV под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** довести SVAV (СОЛЛЕРС) до вердикта по стратегии `rsi_pullback`: каталог сеток с осями по
замерам инструмента, пакет параметров, девять тем rolling walk-forward, литерал и решение о боевой
вселенной.

**Architecture:** тикер-специфичный каталог `data/params/rsi_pullback/svav/` (девять однотемных
файлов + точка) + сторожевой тест осей в `internal/service/backtest` + пакет параметров
`strategy/svav` (сначала baseline-состояние, затем литерал со снимком) + записи в двух реестрах
(бэктест и живой раннер) + заведение в боевую вселенную + разделы документации. Прогоны идут по
ШТАТНОЙ схеме каталога `-months 36 -train-months 12 -test-months 6`: истории ровно 36.0 месяца.

**Tech Stack:** Go 1.25, `cmd/backtest` (grid-search + rolling walk-forward), `cmd/pullparity`,
`go test`, `./bin/golangci-lint`, `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-08-18-svav-rsi-pullback-prep-design.md`

## Global Constraints

- Схема прогонов — **`-months 36 -train-months 12 -test-months 6 -metric profit_factor`**,
  `-min-trades 20` (у темы `screen` — `-min-trades 1`). Это штатная схема каталога: история SVAV
  ровно 36.0 месяца, адаптировать её, как для IVAT и DOMRF, не нужно, и числа сопоставимы
  построчно с UGLD, FESH, WUSH, LENT, RENI, NVTK и LSNGP.
- **Запаса истории нет ни одного дня.** `-months 37` упрётся в начало кэша. Повторный `-refresh`
  во время калибровки НЕ запускать: он сдвинет окно вперёд, укоротит его слева и сделает часть
  прогонов несравнимой с остальными.
- Кэш освежён 2026-08-18: `SVAV_Minutes30.json` — 33 093 бара, 23 869 будних, окно
  2023-08-21 … 2026-08-18 (36.0 мес); `SVAV_Day1.json` — 1 128 дневных свечей (1 016 будних).
  Все замеры ниже сняты по нему.
- Планка, объявленная ДО прогонов и не пересматриваемая после: темы `entry` и `trend` **обе** дают
  pooled OOS PF ≥ 1.5 при ≥ 20 сделках; ведущая ось темы (`RSILower` для `entry`, `EMASlow` для
  `trend`) выбрана одинаково в ≥ 3 фолдах из 4. Вырожденный фолд (ни одной убыточной сделки) не
  засчитывается в пользу тикера. Условие «≥ 5 сделок в каждом фолде» из плана IVAT здесь НЕ
  применяется — оно было костылём под 25-месячную схему.
- Правило прода: литерал ставится и тикер заводится в `RSI_PULLBACK_TICKERS` двенадцатым даже при
  непройденной планке. **Стоп-условие:** pooled OOS PF < 1.0 либо < 20 сделок за 36 месяцев →
  остановиться, принести числа владельцу, задачи 11–14 не выполнять.
- Замеры инструмента, на которые ссылаются `_comment` сеток (все — будние бары):
  - кроссы RSI вниз на уровнях 10/15/20/25/30/35/40: RSI(4) 264/550/920/1383/1929/2448/2913;
    RSI(5) 139/325/624/960/1448/2022/2500; RSI(6) 81/206/442/738/1128/1695/2200;
    RSI(7) 43/146/298/573/913/1417/1953. Для справки RSI(3): 575/1026/1515/2068/2618/3106/3480 —
    в сетку не берётся (три бара это шум, правило каталога);
  - кроссы RSI вверх (полоса выхода), RSI(4): 55 — 2873, 60 — 2515, 65 — 2007, 70 — 1530,
    75 — 1114, 80 — 786; RSI(6): 2243 / 1778 / 1295 / 951 / 627 / 371;
  - доля баров с `EMAFast > EMASlow`: **42.4–43.6% на всех 35 парах** (5/50 — 43.4%, 10/100 —
    42.4%, 20/150 — 43.1%, 40/120 — 43.6%, 30/200 — 42.9%). Размах 1.2 п.п. — допуск от выбора
    пары практически не зависит, в отличие от IVAT (29.4–36.0%, монотонно);
  - дневной ATR(14): медиана **4.38%** цены, p10 2.73, p90 6.93, n=1002; круг издержек 0.1% =
    0.023 ATR, то есть на стопе 0.3 ATR комиссия съедает 7.6% риска;
  - выживаемость стопа (доля дней, чей размах достаёт уровня): 0.3 — 97.4%, 0.5 — 81.8%,
    0.7 — 60.2%, 0.8 — 51.8%, 1.0 — 37.0%, 1.25 — 21.8%, 1.3 — 19.7%, 1.5 — 12.7%;
  - день ко второму бару: медиана **0.31 ATR**; ветка «свежий день» ловит 6.1% баров при 0.2,
    12.9% при 0.3, 23.6% при 0.4, 36.0% при 0.5; ветка «день исчерпан» — 51.4% при 0.6, 35.2% при
    0.8, 29.3% при 0.9, 24.3% при 1.0, 13.4% при 1.25, 12.3% при 1.3, 7.7% при 1.5;
  - объёмный гейт (доля баров, проходящих порог) при базе 14 дней: 48.2% при 1.0, 41.9% при 1.2,
    35.4% при 1.5, **27.3% при 2.0**; база 10 дней — 50.9 / 45.0 / 37.8 / 28.9%; база 5 дней —
    57.5 / 51.2 / 43.5 / 34.4%; база 3 дня — 62.8 / 56.8 / 49.3 / 39.8%;
  - оборот: медиана 68 млн ₽/день, p10 18 млн, p90 351 млн (лот = 1);
  - режим: train −38.5%, holdout −33.2%, вся история −58.9%, пик-минимум −83.4%; из шести
    полугодий растущее одно (+0.8%);
  - контрольный baseline-прогон на 36 мес: **146 сделок, PF 1.368**, net +30 946.
- Отчёты прогонов пишутся в `./reports/SVAV_<тема>` (каталог `reports/` в `.gitignore`).
- Коммиты делать по завершении каждой задачи; сообщения на русском, в стиле существующей истории
  (`feat(rsi_pullback): ...`). Ветка — текущая `feat/svav-pullback-prep` (отведена от
  `feat/ivat-pullback-prep`, HEAD `3dd4e63`).

---

### Task 1: Каталог сеток SVAV со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/svav/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`, `cal_risk.json`, `cal_exit.json`,
  `cal_trail.json`
- Test: `internal/service/backtest/rsi_pullback_svav_grid_test.go`

**Interfaces:**
- Consumes: хелперы `rsiPullbackTickerGrid(t, ticker, file)` (`rsi_pullback_grid_test.go:40`),
  `sameSet(got, want...)` (`rsi_pullback_reni_grid_test.go:17`) и `containsValue(axis, want)`
  (`rsi_pullback_ivat_grid_test.go:88`). **`containsValue` уже объявлена в пакете — НЕ
  переобъявлять**, иначе пакет не соберётся.
- Produces: каталог `svav/`, на который ссылаются все прогоны Task 3–10, и функцию
  `svavGrid(t, file)`.

- [ ] **Step 1: Написать падающий тест осей**

Создать `internal/service/backtest/rsi_pullback_svav_grid_test.go`:

```go
package backtest

import "testing"

// svavGrid читает файл сеток SVAV через общий хелпер.
func svavGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "svav", file)
}

// TestSVAVGridsPinTheirMeasuredAxes сторожит оси каталога svav/. Каталог собран 2026-08-18
// копированием формы ivat/ с пересадкой каждой оси на замеры самого SVAV (33 093 30-минутных
// бара, 23 869 будних, 36.0 месяца с 2023-08-21). Три особенности инструмента, из-за которых
// чужие обоснования сюда не переносятся: истории ровно 36 месяцев, поэтому схема прогонов
// ШТАТНАЯ (-months 36 -train-months 12 -test-months 6) в отличие от ivat/; трендовый допуск
// не зависит от пары EMA (42.4-43.6% на всех 35 парах против 29.4-36.0% с монотонной
// зависимостью у IVAT); объёмный гейт мягкий — на каноническом верхнем крае VolMult=2.0 через
// него проходит ещё 27.3% баров, поэтому ось расширена до 2.5.
func TestSVAVGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := svavGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := svavGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: на SVAV он живой — RSI(4)@10 даёт 264 будних кросса за 36
	// месяцев, слабейший угол сетки RSI(7)@10 — 43, выше LSNGP (29) и RENI (23).
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на SVAV это живой угол (264 кросса RSI(4)@10)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3)@10 даёт 575 кроссов — вдвое больше
	// RSI(4), и это дыхание цены, а не откаты.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}

	trend := svavGrid(t, "cal_trend.json")
	// Ось тренда режется только вместе с этим тестом. На SVAV доля баров с EMAFast > EMASlow
	// укладывается в 42.4-43.6% на ВСЕХ 35 парах: выбор пары меняет не объём допуска, а то,
	// какие именно бары в него попадают. Значит, ни одна пара не мертва по выборке, и сужать
	// сетку не за что — а разница PF между парами читается как качество фильтра.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — медленный край оси обязан остаться, допуск у него тот же 42.5%%", trend["EMASlow"])
	}
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json: пара EMAFast=%v EMASlow=%v вырождена — быстрая обязана быть быстрее медленной", fast, slow)
			}
		}
	}

	risk := svavGrid(t, "cal_risk.json")
	// Строка 0.3 сохранена намеренно: при дневном ATR 4.38% круг издержек 0.1% съедает 7.6%
	// риска — вдвое ниже черты 17%, по которой её вырезали из domrf/.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на SVAV комиссия съедает там 7.6%% риска, строка живая", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: стоп обязан быть положительным", v)
		}
		// Верхний край 1.3: уровня 1.5 ATR достаёт лишь 12.7% дней, такой стоп перестаёт быть
		// защитой и становится способом вытеснить убыток в RSI-выход.
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: до него доходит меньше 13%% дней — это не стоп", v)
		}
	}

	day := svavGrid(t, "cal_day.json")
	// Ветка «свежий день» ловит 12.9% будних баров при пороге 0.3 и 36.0% при 0.5; ноль в оси —
	// это выключенная ветка, и она обязана остаться, потому что на всех прод-тикерах каталога
	// победил именно ноль.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — выключенная ветка обязана быть в сетке", day["FreshDayATR"])
	}

	volume := svavGrid(t, "cal_volume.json")
	// Ось расширена против образца ivat/ по замеру, а не по желанию перебрать больше: при базе
	// 14 дней канонический верхний край 2.0 пропускает ещё 27.3% баров (у IVAT там было 15.8%),
	// то есть отсекающая способность оси не исчерпана.
	if !containsValue(volume["VolMult"], 2.5) {
		t.Errorf("cal_volume.json: VolMult = %v, не содержит 2.5 — на SVAV порог 2.0 пропускает ещё 27.3%% баров, ось не исчерпана", volume["VolMult"])
	}

	exit := svavGrid(t, "cal_exit.json")
	// Полоса выхода рабочая на всём диапазоне: кроссов вверх RSI(4) от 2873 (55) до 786 (80).
	for _, v := range exit["RSIUpper"] {
		if v <= 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: выход обязан стоять выше средней линии", v)
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestSVAVGridsPinTheirMeasuredAxes -v`
Expected: FAIL — каталога `data/params/rsi_pullback/svav/` ещё нет, хелпер падает на чтении файла.

- [ ] **Step 3: Создать девять файлов сеток**

Каждый файл — объект с `_comment` и массивом `phases`. Сетки даны ниже дословно. `_comment`
пишется по образцу `ivat/` и обязан содержать четыре части: (1) что тема меряет и сколько в ней
прогонов; (2) замер из Global Constraints, из которого получена каждая ось этого файла, с прямым
указанием, почему край оси стоит там, где стоит; (3) команду запуска целиком (схема
`-months 36 -train-months 12 -test-months 6`); (4) пустое место под строку
«РЕЗУЛЬТАТ ПРОГОНА 2026-08-18: …», которую заполняют задачи 3–9.

**Жёсткое требование пакета, а не стиля:** `TestRSIPullbackCalFilesValid`
(`internal/service/backtest/rsi_pullback_grid_test.go:88`) падает, если `_comment` файла не
содержит его собственный путь вида `svav/cal_entry.json`. Полная команда запуска с
`data/params/rsi_pullback/svav/cal_entry.json` это условие выполняет; проверка ловит `_comment`,
скопированный у соседнего тикера без правки пути.

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

`cal_volume.json` — 20 прогонов (ось `VolMult` расширена до 2.5):
```json
{"phases": [{"name": "volume", "grid": {
  "UseVolume": [1],
  "VolMult": [1.0, 1.2, 1.5, 2.0, 2.5],
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

Run: `go test ./internal/service/backtest/ -run 'RSIPullback|SVAV' -v`
Expected: PASS, включая общий `TestRSIPullbackGridControlPoints` — он обходит каталог рекурсивно и
требует, чтобы файл, свипующий `StopDailyATR`, свипевал и цель шире самого широкого стопа
(`cal_risk.json` это выполняет: 2.5 > 1.3).

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/svav internal/service/backtest/rsi_pullback_svav_grid_test.go
git commit -m "feat(rsi_pullback): сетки SVAV с замеренными осями"
```

---

### Task 2: Пакет `strategy/svav` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/svav/svav.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/svav/svav_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт + запись в карту)
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (сторожевой тест baseline)

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `svav.Ticker` (константа `"SVAV"`) и `svav.DefaultParams() core.Params` — их используют
  Task 11 (литерал), Task 12 (реестр живого раннера) и Task 13 (вселенная).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/svav/svav_test.go`:

```go
package svav

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Пакет заведён 2026-08-18 ДО калибровки: он обязан отслеживать baseline, чтобы правка дефолтов
// доходила до тикера, а не расходилась с ним молча. Тест держит это состояние и подлежит замене
// снимком литерала ровно тогда, когда калибровка закончится (задача 11 плана).
func TestParamsTrackTheBaseline(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("SVAV ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSVAV(t *testing.T) {
	if Ticker != "SVAV" {
		t.Fatalf("Ticker = %q, want SVAV", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/svav/ -v`
Expected: FAIL — пакета нет, сборка не проходит.

- [ ] **Step 3: Создать пакет**

`internal/service/trading_strategy/rsi_pullback/strategy/svav/svav.go`:

```go
// Package svav supplies the ticker and rsi_pullback Params for SVAV (СОЛЛЕРС).
//
// СОСТОЯНИЕ: калибровка не проводилась. Пакет возвращает core.DefaultParams(), то есть
// отслеживает baseline: правка дефолтов ядра доходит до этого тикера. Ставить SVAV в боевую
// вселенную RSI_PULLBACK_TICKERS в таком состоянии нельзя — торговля пошла бы параметрами,
// которые на этом инструменте никогда не проверялись.
//
// Что известно об инструменте до прогонов (кэш 2026-08-18, 33 093 30-минутных бара, 23 869
// будних, 36.0 месяца с 2023-08-21):
//
//   - ИСТОРИИ РОВНО 36.0 МЕСЯЦА, поэтому штатный протокол §8 docs/rsi_pullback/strategy.md
//     (-months 36 -train-months 12 -test-months 6) исполним встык, и числа SVAV сопоставимы
//     построчно с остальным каталогом — в отличие от ivat (26 мес) и domrf (8.8 мес). Запаса
//     сверх 36 месяцев нет ни одного дня.
//   - ТРЕНДОВЫЙ ДОПУСК НЕ ЗАВИСИТ ОТ ПАРЫ: доля баров с EMAFast > EMASlow укладывается в
//     42.4-43.6% на всех 35 парах сетки, размах 1.2 процентного пункта. У IVAT тот же замер
//     даёт 29.4-36.0% с монотонной зависимостью от EMASlow, у LSNGP — 54-60%. Практическое
//     следствие: на SVAV выбор пары меняет не объём допуска, а то, какие именно бары в него
//     попадают, поэтому дефицит выборки из-за медленной пары здесь невозможен, а разница PF
//     между парами — это качество фильтра, а не размер выборки.
//   - ВОЛАТИЛЬНОСТЬ ВЫСОКАЯ: дневной ATR(14) медианой 4.38% цены (p10 2.73, p90 6.93) — второй
//     в каталоге после FESH (4.42). Круг издержек 0.1% оборота стоит 0.023 ATR, то есть на
//     стопе 0.3 ATR комиссия съедает 7.6% риска (черта, по которой строку 0.3 вырезали из
//     domrf/, — 17%). Стоп 0.5 ATR это 2.2% цены: при Fraction=1 одна сделка двигает счёт
//     заметно сильнее, чем на LSNGP с его ATR 2.64%.
//   - РЕЖИМ: падают ОБА окна протокола (train −38.5%, holdout −33.2%), вся история −58.9%,
//     пик-минимум −83.4%, и из шести полугодий растущее одно (+0.8%). Завысить лонговый
//     результат режимом здесь нечем, но и на растущем рынке конфигурация не проверена ни разу.
//   - ЛИКВИДНОСТЬ: оборот медианой 68 млн ₽/день при p10 = 18 млн — лучше IVAT (43), LENT (38)
//     и LSNGP (41): больше половины дней проходят гейт отбора скринера в 50 млн самостоятельно.
//   - Контрольный прогон baseline на 36 месяцах: 146 сделок, PF 1.368 — инструмент торгует,
//     дефицита сделок уровня «стратегия молчит» здесь нет.
//
// Сетки калибровки лежат в data/params/rsi_pullback/svav/, их оси прибиты
// internal/service/backtest/rsi_pullback_svav_grid_test.go.
package svav

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SVAV"

// DefaultParams returns the strategy baseline: SVAV is not calibrated yet.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт
`rsipullbacksvav "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svav"` и строку
карты рядом с остальными:

```go
	rsipullbacksvav.Ticker:  rsiPullbackBindingFor(rsipullbacksvav.Ticker, rsipullbacksvav.DefaultParams),
```

- [ ] **Step 5: Добавить сторожевой тест baseline в реестр бэктеста**

В `internal/service/backtest/rsi_pullback_registry_test.go`:

```go
// TestRSIPullbackSVAVTracksBaseline держит состояние «калибровка не проводилась»: пакет
// strategy/svav заведён 2026-08-18 под будущий литерал, и до конца калибровки обязан возвращать
// core.DefaultParams(). Тест заменяется снимком литерала в тот день, когда литерал появится, —
// ровно так это было с reni, fesh, wush, lent, lsngp, nvtk и ivat.
func TestRSIPullbackSVAVTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["SVAV"]
	if !ok {
		t.Fatal("SVAV отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("SVAV: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("SVAV отклонился от baseline до калибровки: %+v", p)
	}
	if got := b.Build(p).Ticker(); got != "SVAV" {
		t.Fatalf("Ticker() = %q, want SVAV", got)
	}
}
```

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'SVAV|RSIPullback' -v`
Expected: PASS. Тест
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go:TestBaselineTrackingTickersStayOutOfTheDefaultUniverse`
обязан остаться зелёным: SVAV пока не в реестре живого раннера и не в дефолтной вселенной.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/svav internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): пакет и реестр SVAV в состоянии «калибровка не проводилась»"
```

---

### Task 3: Тема `screen` — цена двух опциональных гейтов

**Files:**
- Modify: `data/params/rsi_pullback/svav/cal_screen.json` (дописать результат в `_comment`)

**Interfaces:**
- Consumes: каталог сеток из Task 1, пакет из Task 2.
- Produces: знание, сколько сделок стоит каждый гейт — им пользуются задачи 6 и 7 при разборе.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_screen.json -out ./reports/SVAV_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Прочитать отчёт**

Открыть свежий файл в `./reports/SVAV_screen/`. Выписать: pooled OOS PF и счёт сделок каждой из
четырёх комбинаций, выбор калибратора по фолдам, и во сколько сделок обходится каждый гейт.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать строку вида `РЕЗУЛЬТАТ ПРОГОНА 2026-08-18: pooled OOS PF <...> на <...> сделках, фолды
<...>; гейт дня стоит <...> сделок, объёмный — <...>.` Числа — фактические из отчёта, без
округления в свою пользу.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/svav/cal_screen.json
git commit -m "feat(rsi_pullback): SVAV, тема screen прогнана"
```

---

### Task 4: Тема `entry` — ключевая, полоса RSI целиком

**Files:**
- Modify: `data/params/rsi_pullback/svav/cal_entry.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: первое из двух чисел планки (pooled OOS PF темы `entry`, счёт сделок, устойчивость
  `RSILower` по фолдам) — Task 11 выносит по ним вердикт.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_entry.json -out ./reports/SVAV_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа планки**

Из отчёта: pooled OOS PF, счёт сделок пула, PF и счёт сделок каждого из четырёх фолдов, выбор
`RSIPeriod` / `RSILower` / `RSIUpper` по каждому фолду. Отдельно отметить вырожденные фолды (без
убыточных сделок) — планка их не засчитывает. Сравнить in-sample и OOS по фолдам: разрыв втрое и
больше означает переобучение темы, и это записывается явно (случай IVAT).

- [ ] **Step 3: Записать результат в `_comment` сетки**

Формат: `РЕЗУЛЬТАТ ПРОГОНА 2026-08-18: pooled OOS PF <...> на <...> сделках, фолды <...> — порог
1.5 <взят|не взят>. Ведущая ось RSILower выбрана <...> — устойчивость <N> из 4.`

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/svav/cal_entry.json
git commit -m "feat(rsi_pullback): SVAV, тема entry прогнана"
```

---

### Task 5: Тема `trend` — вторая ключевая

**Files:**
- Modify: `data/params/rsi_pullback/svav/cal_trend.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: второе число планки (pooled OOS PF темы `trend`, устойчивость `EMASlow`).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_trend.json -out ./reports/SVAV_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить счёт сделок против замера допуска**

Ключевая проверка именно этой темы на этом тикере: допуск фильтра одинаков (42.4–43.6%) на всех 35
парах, поэтому **счёт сделок фолда не должен заметно меняться от выбора `EMASlow`**. Выписать счёт
сделок по нескольким парам с разных краёв оси (5/50, 10/100, 40/200). Если счёт всё-таки заметно
разошёлся — причина в другом гейте (дневном или объёмном), и это надо записать числом: замер
допуска сам по себе тогда не объясняет тему.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Тот же формат, что в Task 4, но ведущая ось — `EMASlow`. Дополнительно записать счёт сделок по
парам из Step 2.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/svav/cal_trend.json
git commit -m "feat(rsi_pullback): SVAV, тема trend прогнана"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

**Files:**
- Modify: `data/params/rsi_pullback/svav/cal_day.json`, `cal_day_spent.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3 (цена гейта в сделках).
- Produces: значения `FreshDayATR` и `SpentDayATR` для литерала Task 10.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_day.json -out ./reports/SVAV_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_day_spent.json -out ./reports/SVAV_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить ветку «свежий день» со своим замером**

Ветка ловит 12.9% баров при пороге 0.3, 23.6% при 0.4 и 36.0% при 0.5 — на SVAV она щедрее, чем на
IVAT (7.5% при 0.3). Если калибратор выбирает ненулевой `FreshDayATR`, проверить, что прирост PF не
куплен обвалом качества: выписать pooled PF и счёт сделок обоих вариантов. На всех прод-тикерах
каталога победил ноль, и отклонение от этого должно опираться на число, а не на выбор калибратора.

- [ ] **Step 3: Записать результаты в оба `_comment`**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/svav/cal_day.json data/params/rsi_pullback/svav/cal_day_spent.json
git commit -m "feat(rsi_pullback): SVAV, темы дневного гейта прогнаны"
```

---

### Task 7: Тема `volume` — фон объёмов на расширенной оси

**Files:**
- Modify: `data/params/rsi_pullback/svav/cal_volume.json`

**Interfaces:**
- Consumes: каталог из Task 1, результат Task 3.
- Produces: решение о `UseVolume`, `VolMult`, `VolBaseDays` для литерала Task 10.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_volume.json -out ./reports/SVAV_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Отдельно разобрать расширенный край оси**

`VolMult = 2.5` — единственное значение каталога, которого нет в образце `ivat/`. Выписать pooled PF
и счёт сделок отдельно для 2.0 и 2.5: если 2.5 выигрывает, надо назвать, чем именно — качеством
отбора или сокращением выборки (при базе 14 дней порог 2.0 пропускает 27.3% баров, 2.5 — меньше).

- [ ] **Step 3: Проверить на вырождение фолда**

На GAZP и NVTK объёмный гейт покупал pooled PF вырожденным фолдом (17.146 на 19 сделках). Если
здесь повторится — гейт отвергается, и причина записывается числом, а не мнением.

- [ ] **Step 4: Записать результат в `_comment`**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/svav/cal_volume.json
git commit -m "feat(rsi_pullback): SVAV, тема volume прогнана"
```

---

### Task 8: Тема `risk` — стоп и цель

**Files:**
- Modify: `data/params/rsi_pullback/svav/cal_risk.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `StopDailyATR` и `TPDailyATR` для литерала Task 10.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_risk.json -out ./reports/SVAV_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить ось стопа на вытеснение убытков**

Капкан, разобранный на WUSH, LENT, LSNGP и IVAT: profit factor растёт монотонно с шириной стопа, а
доля выходов по стопу падает — это вытеснение убытка в RSI-выход, а не улучшение защиты. **Выписать
долю стоп-выходов для каждой из пяти точек оси, а не только PF.** Опорный замер выживаемости на
SVAV: уровня 0.3 ATR достаёт 97.4% дней, 0.5 — 81.8%, 0.7 — 60.2%, 1.0 — 37.0%, 1.3 — 19.7%.

- [ ] **Step 3: Записать результат в `_comment`**

Кроме pooled PF и фолдов записать таблицу «StopDailyATR → доля стоп-выходов → счёт сделок».

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/svav/cal_risk.json
git commit -m "feat(rsi_pullback): SVAV, тема risk прогнана"
```

---

### Task 9: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/svav/cal_exit.json`, `cal_trail.json`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` для литерала Task 10.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_exit.json -out ./reports/SVAV_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/cal_trail.json -out ./reports/SVAV_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить характер сделки при выбранной полосе**

На LSNGP `RSIUpper` 55 уронил медиану удержания до 4 баров — многодневная стратегия стала
внутридневной. Если тема выбирает низкую полосу, выписать медиану удержания и долю сделок длиннее
одного торгового дня: это плата, которую надо назвать явно.

- [ ] **Step 3: Учесть структурный перекос темы `trail`**

`-min-trades 20` структурно топит ветку `UseRSIExit=0`: без RSI-выхода удержание длиннее, сделок
меньше, открытая позиция блокирует входы. Если все строки с `UseRSIExit=0` ушли под порог, это
процедурная причина, а не вывод о трейле — записать это в `_comment` прямо, а не выдавать за
результат.

- [ ] **Step 4: Записать результаты в оба `_comment`**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/svav/cal_exit.json data/params/rsi_pullback/svav/cal_trail.json
git commit -m "feat(rsi_pullback): SVAV, темы выходов прогнаны"
```

---

### Task 10: Сборка литерала и точечный walk-forward принятой точки

**Files:**
- Create: `data/params/rsi_pullback/svav/plateau_point.json`

**Interfaces:**
- Consumes: результаты задач 3–9.
- Produces: конкретный набор из восемнадцати полей `core.Params` и его замеры — их прибивает
  Task 11.

- [ ] **Step 1: Собрать точку из лидербордов тем**

Взять по каждой теме её выбор. Где тема мерила ось поверх дефолтов, стоящих вне рабочей зоны
инструмента (случай NVTK, где дефолтный дневной гейт стоял там, где стратегии нет), — проверить ось
точечными прогонами `-params` и записать, что выбор расходится с темой и почему.

- [ ] **Step 2: Создать файл точки**

`plateau_point.json` — одна фаза `point`, каждое из восемнадцати полей задано массивом из одного
значения (формат `ivat/plateau_point.json`; массив из двух значений уронит
`TestRSIPullbackPlateauFilesArePoints`). `_comment` обязан содержать: замеры принятой точки,
явную оговорку «для фиксированной точки это НЕ out-of-sample», как собиралась каждая ось (выбор
темы или точечный прогон), и команду запуска.

Поля, которые обязаны быть в файле: `RSIPeriod`, `RSILower`, `RSIUpper`, `EMAFast`, `EMASlow`,
`DailyATRPeriod`, `UseDayATRGate`, `FreshDayATR`, `SpentDayATR`, `StopDailyATR`, `TPDailyATR`,
`UseVolume`, `VolBaseDays`, `VolLookbackBars`, `VolMult`, `UseRSIExit`, `UseTrail`, `TrailDailyATR`.

- [ ] **Step 3: Прогнать точку тем же walk-forward**

```bash
go run ./cmd/backtest -ticker SVAV -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svav/plateau_point.json -out ./reports/SVAV_point \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 4: Проверить стоп-условие плана**

Если pooled OOS PF < 1.0 или сделок меньше 20 — остановиться, вынести числа владельцу, задачи 11–14
не выполнять до его решения.

- [ ] **Step 5: Замерить плато соседями**

По каждой оси прогнать соседние значения точечно и выписать pooled PF и счёт сделок: плато шириной
в один шаг — это пик, а не полка, и в доке пакета это должно быть названо (случай UGLD, где
`RSILower` 20 роняет точку с 3.627 до 1.580). Отдельно проверить, не держится ли pooled PF одним
фолдом или одной неделей — на IVAT 85% результата сделала неделя июля 2026, и без этой проверки
число читается неверно.

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/svav/plateau_point.json
git commit -m "feat(rsi_pullback): SVAV, принятая точка и её замеры"
```

---

### Task 11: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/svav/svav.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/svav/svav_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (заменить baseline-тест снимком)

**Interfaces:**
- Consumes: набор полей из Task 10.
- Produces: `svav.DefaultParams()`, возвращающий литерал, — его читают Task 12 (реестр раннера) и
  Task 13 (вселенная).

- [ ] **Step 1: Заменить тест baseline снимком литерала**

В `svav_test.go` удалить `TestParamsTrackTheBaseline` и написать снимок по образцу `ivat_test.go`:
`TestCalibratedLiteralIsPinned` (все восемнадцать полей), `TestParamsDoNotTrackTheBaseline`,
`TestStopIsArmed`, `TestRSIExitIsArmed`, плюс тесты под фактически принятую конфигурацию —
`TestOnlySpentDayBranchIsArmed` / `TestVolumeGateStaysOff` / `TestTrailStaysOff` пишутся под то, что
получилось, а не копируются вслепую. Каждый тест несёт в комментарии замер, объясняющий, почему
поле именно такое.

Обязательные инварианты, которые снимок обязан сторожить независимо от результата калибровки:
`StopDailyATR > 0`, `UseRSIExit == 1` (живой раннер держит RSI-выход обязательным для всех тикеров
реестра), `RSIUpper > RSILower`, `TPDailyATR > 0`, и при `UseTrail == 0` — `TrailDailyATR == 0`.

- [ ] **Step 2: Запустить и убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/svav/ -v`
Expected: FAIL — `DefaultParams()` ещё возвращает baseline.

- [ ] **Step 3: Поставить литерал**

В `svav.go` заменить `return core.DefaultParams()` литералом из Task 10 и переписать доку пакета:
раздел «СОСТОЯНИЕ: калибровка не проводилась» → разбор калибровки (результат девяти тем, вердикт по
планке пункт за пунктом, разбор каждого поля литерала, граница приёма, замеры инструмента из
прежней редакции доки сохраняются).

- [ ] **Step 4: Заменить сторожевой тест в реестре бэктеста**

`TestRSIPullbackSVAVTracksBaseline` → `TestRSIPullbackSVAVIsRegisteredAndCalibrated` по образцу
теста IVAT (`rsi_pullback_registry_test.go:432`): проверяет наличие в карте, несовпадение с
baseline, равенство литералу пакета и `Ticker()`. Комментарий теста обязан назвать вердикт по
планке — взята она или нет, с числами обеих ключевых тем.

- [ ] **Step 5: Запустить тесты и линт**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ && ./bin/golangci-lint run ./internal/service/...`
Expected: PASS, 0 issues.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/svav internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): SVAV откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 12: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`

**Interfaces:**
- Consumes: `svav.Ticker`, `svav.DefaultParams()` из Task 11.
- Produces: `ParamsFor("SVAV")` и `StrategyFor("SVAV")`, без которых раннер тикер не увидит.

- [ ] **Step 1: Добавить импорт и запись в карту**

```go
	svav.Ticker:  svav.DefaultParams(),
```

- [ ] **Step 2: Дописать комментарий карты**

Абзац про SVAV: штатная схема прогонов (в отличие от IVAT и DOMRF — числа сопоставимы с каталогом),
вердикт по планке с числами обеих ключевых тем, режим без единого растущего полугодия кроме одного
на 0.8%, дневной ATR 4.38% как второй в каталоге, ликвидность 68 млн ₽/день медианой.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
Expected: PASS — включая `TestRegisteredTickersKeepTheRSIExitArmed`, который обходит всю карту, и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse`.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live/registry.go
git commit -m "feat(rsi_pullback): SVAV в реестре живого раннера"
```

---

### Task 13: Заведение в боевую вселенную

**Files:**
- Modify: `internal/config/rsi_pullback.go` (дефолт `Tickers` + комментарий)
- Modify: `internal/config/rsi_pullback_test.go:54` (ожидание дефолта)
- Modify: `env/prod.env:20`, `env/prod.env.example:30`, `env/local.env.example:22-27`
- Modify: `docs/rsi_pullback/live.md` (таблица §8, раздел про реестр, §9 порядок выката)

**Interfaces:**
- Consumes: литерал из Task 11, запись реестра из Task 12.
- Produces: боевую вселенную из двенадцати тикеров.

- [ ] **Step 1: Проверить стоп-условие**

Если Task 10 остановился на стоп-условии — эта задача не выполняется. Иначе продолжать.

- [ ] **Step 2: Обновить тест дефолта**

```go
	want := []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT", "SVAV"}
```

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -run TestNewRSIPullbackConfig_Defaults -v`
Expected: FAIL — дефолт ещё из одиннадцати тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

`Tickers: []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT", "SVAV"}`
и та же строка в `RSI_PULLBACK_TICKERS` трёх env-файлов. В комментарий функции дописать абзац про
SVAV с типом его риска (высокая волатильность 4.38% ATR, режим без растущих окон, штатная схема
прогонов).

- [ ] **Step 5: Обновить live.md**

Таблица §8 (дефолт переменной), раздел про реестр («знает двенадцать пакетов»), §9 пункт 1 —
добавить SVAV в список сверки `pullparity`.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS — `TestEveryDefaultTickerIsRegistered` и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` читают вселенную из конфига и покроют новый
состав автоматически.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести SVAV в боевую вселенную"
```

---

### Task 14: Документация — разбор калибровки и принятый риск

**Files:**
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1 строка каталога + раздел с разбором прогонов)
- Modify: `docs/rsi_pullback/live.md` (§10, риск 15 — следующий за риском 14 про IVAT)

**Interfaces:**
- Consumes: числа задач 3–10, решение задачи 13.
- Produces: справочник, по которому тикер сопровождают в живой торговле.

- [ ] **Step 1: Дописать §8.0.1 `strategy.md`**

В ячейку «откалиброван» добавить `svav` с датой, схемой прогонов (штатная — назвать это явно,
потому что предыдущие два тикера каталога шли по адаптированным), вердиктом по планке, замерами
принятой точки и ссылкой на риск 15 в `live.md`.

- [ ] **Step 2: Дописать раздел с разбором прогонов**

По образцу разделов GAZP, NVTK и UGLD: рамки данных, режим, вердикт по планке пункт за пунктом,
разбор каждого поля литерала, граница приёма («для фиксированной точки это НЕ out-of-sample»).
Отдельным абзацем — что дала плоская трендовая ось: тема `trend` на этом тикере меряет качество
фильтра, а не размер выборки, и это первый такой случай в каталоге.

- [ ] **Step 3: Дописать риск 15 в `live.md` §10**

Замеры, практические следствия для наблюдения (распределение выходов, медиана удержания, просадка),
три ограничения — высокая волатильность (стоп 0.5 ATR = 2.2% цены при `Fraction=1`), режим без
растущих окон, p10 оборота 18 млн ₽/день, — и условия пересмотра.

- [ ] **Step 4: Проверить, что доки не противоречат числам**

Run: `grep -rn "SVAV" docs/rsi_pullback/*.md internal/service/trading_strategy/rsi_pullback/strategy/svav/*.go internal/config/rsi_pullback.go`
Сверить каждое число с отчётами прогонов в `./reports/SVAV_*`.

- [ ] **Step 5: Коммит**

```bash
git add docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки SVAV и принятый риск"
```

---

### Task 15: Финальная проверка

**Files:** нет изменений, только проверки.

- [ ] **Step 1: Полный гейт качества**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + проверка дрейфа моков — зелёные.

- [ ] **Step 2: Сверка живой сборки с бэктестом**

```bash
go run ./cmd/pullparity -tickers SVAV -months 24
```
Expected: ноль расхождений. **24 месяца, а не 36:** живой сборщик тянет дневные свечи окном
`dailyFetchDays = 730`, и на большем горизонте появляются ожидаемые расхождения длины `Daily*`
рядов (`maxDailyHorizonMonths`, выяснено на IVAT). Расхождение на 24 месяцах означает, что живой
раннер и бэктест считают сигнал по-разному, и заведение в прод откатывается до выяснения.

- [ ] **Step 3: Сообщить владельцу результат**

Короткий отчёт: вердикт по планке пункт за пунктом, замеры принятой точки, что заведено в прод,
какие риски записаны, что осталось (первые живые сделки, условия пересмотра).

---

## Self-review

**Покрытие спеки.** Рамки данных → Global Constraints; штатный протокол → Global Constraints и
каждая команда прогона; режим → доки пакета (Task 2, Task 11) и риск 15 (Task 14); плоская
трендовая ось → сторожевой тест Task 1, Step 2 задачи 5, доки Task 14; оси девяти сеток → Task 1;
расширение `VolMult` до 2.5 → Task 1 (сетка + сторожевой тест) и Task 7 Step 2; планка → Global
Constraints, вердикт выносится в Task 11 и Task 14; правило прода и стоп-условие → Task 10 Step 4 и
Task 13 Step 1; артефакты 1-6 спеки → задачи 1, 2, 11, 12, 13, 14; порядок работы спеки → порядок
задач; риск «нет запаса истории» → Global Constraints (запрет на `-refresh`).

**Плейсхолдеры.** В задачах 3–10 значения результатов прогонов не выдуманы: там, где число станет
известно только после запуска, стоит явное указание выписать его из отчёта — это данные, которых до
прогона не существует, а не плейсхолдер плана. Код сторожевого теста осей, теста baseline, доки
пакета и все девять сеток даны целиком. Снимок литерала (Task 11) задан списком обязательных
инвариантов, потому что его значения — результат Task 10.

**Согласованность типов.** `svav.Ticker` (строка `"SVAV"`) и `svav.DefaultParams() core.Params`
объявлены в Task 2 и используются под теми же именами в задачах 11, 12, 13. Хелпер `svavGrid`
объявлен в Task 1; `containsValue`, `sameSet` и `rsiPullbackTickerGrid` берутся из существующих
файлов пакета и НЕ переобъявляются (это отдельно оговорено в Task 1 — иначе пакет не соберётся).
Имена тестов, заменяемых на следующем шаге (`TestParamsTrackTheBaseline` → снимок,
`TestRSIPullbackSVAVTracksBaseline` → `TestRSIPullbackSVAVIsRegisteredAndCalibrated`), названы в
обеих задачах одинаково.
