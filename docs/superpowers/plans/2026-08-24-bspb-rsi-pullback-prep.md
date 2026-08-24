# BSPB под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести BSPB (ПАО «Банк „Санкт-Петербург“») до вердикта по стратегии `rsi_pullback`:
каталог сеток с осями по замерам инструмента, тематический walk-forward, принятая точка, литерал в
пакете и заведение в боевую вселенную шестнадцатым тикером.

**Architecture:** Процедура каталога, но с одним отступлением, объявленным в спеке: темы делятся на
ранние (`screen`, `entry`, `trend` — поверх дефолтов ядра, на них стоит планка) и поздние (семь тем
поверх якоря, выбранного ранними). Поэтому каталог сеток создаётся в два приёма: три файла в Task 1
и семь в Task 6, после того как якорь появится.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-08-24-bspb-rsi-pullback-prep-design.md`

## Global Constraints

- **Схема прогонов — штатная:** `-months 36 -train-months 12 -test-months 6 -min-trades 20 -metric
  profit_factor`, четыре фолда встык (12 + 4×6 = 36). У темы `screen` — `-min-trades 1`.
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш перезалит 2026-08-24 до первого замера:
  `BSPB_Minutes30.json` — 35 261 бар (2023-08-24 … 2026-08-24), `BSPB_Day1.json` — 1130 свечей
  (2022-08-24 … 2026-08-23). Запас истории — **ноль дней**: получасовая серия начинается ровно за
  36 месяцев до правой границы. Любой refresh сдвинет ОБЕ границы и сделает часть прогонов
  несравнимой.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` обе дают pooled
  OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для `entry`, `EMASlow` для `trend`)
  выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд (ни одной убыточной сделки) в пользу
  тикера не засчитывается; счёт по дефолтной комиссии (круг 0.1%).
- **Правило прода:** литерал ставится и BSPB заводится в `RSI_PULLBACK_TICKERS` шестнадцатым
  **независимо от того, взята планка или нет**.
- **Стоп-условие** (канон плюс третий пункт): работа останавливается, если принятая точка даёт
  pooled OOS PF < 1.0, **либо** меньше 20 сделок за расчётное окно, **либо** PF < 1.0 под
  удвоенными издержками (`-commission 0.001`). При срабатывании — числа владельцу, задачи 13–16 не
  выполняются.
- **Якорь поздних тем** собирается по правилу спеки: вход из победителя pooled OOS темы `entry`,
  тренд из победителя `trend`, все остальные поля — дефолты ядра; при выборке победителя `entry`
  меньше 20 сделок на полной истории берётся второе место лидерборда, и оба кандидата пишутся в
  `_comment`.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер,
  из которого получена каждая ось, с обоснованием края; полную команду запуска с путём
  `data/params/rsi_pullback/bspb/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); для
  поздних тем — якорь и оговорку об условности; место под строку `РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: …`.
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Дефолты ядра** (`core.DefaultParams()`), от которых считают ранние темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`,
  `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`,
  `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`,
  `UseTrail 0`, `TrailDailyATR 0`.

---

### Task 1: Каталог ранних сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/bspb/cal_screen.json`
- Create: `data/params/rsi_pullback/bspb/cal_entry.json`
- Create: `data/params/rsi_pullback/bspb/cal_trend.json`
- Create: `internal/service/backtest/rsi_pullback_bspb_grid_test.go`

**Interfaces:**
- Consumes: хелперы `rsiPullbackTickerGrid(t, ticker, file)`, `sameSet`, `containsValue` из
  `internal/service/backtest/rsi_pullback_grid_test.go`.
- Produces: каталог `data/params/rsi_pullback/bspb/` и функцию `bspbGrid(t, file)`, которой
  пользуется Task 6.

- [ ] **Step 1: Написать падающий тест осей**

Создать `internal/service/backtest/rsi_pullback_bspb_grid_test.go`:

```go
package backtest

import "testing"

// bspbGrid читает файл сеток BSPB через общий хелпер.
func bspbGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "bspb", file)
}

// TestBSPBEarlyGridsPinTheirMeasuredAxes сторожит оси трёх РАННИХ тем каталога bspb/ — тех, что
// идут поверх дефолтов ядра. Каталог собран 2026-08-24 по замерам самого BSPB: 35 261
// получасовой бар в кэше (2023-08-24 … 2026-08-24), из них 24 953 будних; дневная серия — 1130
// свечей с 2022-08-24. Запас истории НУЛЕВОЙ: 30m у поставщика отдаёт ровно три года.
//
// Три решения отличают этот каталог от образца dias/, и каждое опирается на замер сделками, а не
// на счёт кроссов:
//
//   - RSIPeriod РАСШИРЕН ВНИЗ ДО 3 — отмена правила каталога для этой бумаги, решение владельца
//     от 2026-08-24. Правило гласило: тройка не берётся, потому что кроссов у неё вдвое больше,
//     чем у четвёрки (на BSPB 660 против 311 на уровне 10), и это дыхание цены. Проверка
//     сделками правило опровергает: RSI(3)@15 даёт 94 сделки при PF 1.327, RSI(4)@15 — 56 при
//     1.068. Тройка и чаще торгует, и точнее.
//   - RSIPeriod ОБОРВАН НА 6, хотя счёт кроссов позволял бы 7 и 8: RSI(7)@10 даёт 54 будних
//     кросса — выше планки живого угла каталога (29). Обрывает ось замер сделками: RSI(7)@10 —
//     2 сделки, RSI(8)@10 — НОЛЬ, RSI(7)@15 — 10, RSI(8)@15 — 7. Дневной гейт и тренд-фильтр
//     вырезают всё, что оставляет медленный RSI.
//   - Окно оси EMA СДВИНУТО ВНИЗ целиком. Максимум замера стоял на нижнем крае каталожной оси
//     (10/30 → PF 2.055, 5/50 → 1.950), а весь её верх мёртв (10/150 → 1.020, 20/150 → 1.135,
//     40/200 → 1.058) при ровном допуске 48.2–50.4% на всех парах. Побочное следствие: пары с
//     EMAFast >= EMASlow либо вырождены (30/30 и 40/40 дают 0.0% допуска), либо инвертированы
//     (40/30 — «медленная над быстрой», другой фильтр), поэтому EMAFast обрезан до 20.
func TestBSPBEarlyGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := bspbGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := bspbGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: RSI(4)@10 даёт 311 будних кроссов и 32 сделки при PF 1.727 —
	// лучший PF всей оси входа на дефолтных прочих полях.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на BSPB это самый прибыльный угол оси (32 сделки, PF 1.727)", entry["RSILower"])
	}
	// Тройка — сознательное отступление от правила каталога, и держится она замером сделками.
	if !containsValue(entry["RSIPeriod"], 3) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 3 — на BSPB тройка бьёт четвёрку и по числу сделок (94 против 56), и по PF (1.327 против 1.068); решение владельца 2026-08-24", entry["RSIPeriod"])
	}
	// Ось обрывается на 6: медленнее её выборка мертва (RSI(7)@10 — 2 сделки, RSI(8)@10 — ноль).
	for _, v := range entry["RSIPeriod"] {
		if v > 6 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: на BSPB медленный RSI не оставляет сделок — RSI(7)@10 даёт 2, RSI(8)@10 ноль", v)
		}
		if v < 3 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче тройки — уже не откат, а тик", v)
		}
	}
	// Полоса выхода расширена до 85, чтобы разворот (75 → 1.806, 80 → 1.636, 85 → 1.324) попал
	// ВНУТРЬ сетки, а не на её край.
	if !containsValue(entry["RSIUpper"], 85) {
		t.Errorf("cal_entry.json: RSIUpper = %v, не содержит 85 — без него максимум полосы (75) стоит у края оси", entry["RSIUpper"])
	}

	trend := bspbGrid(t, "cal_trend.json")
	// Ось расширена вниз: максимум оси — пара 10/30 (PF 2.055), то есть ЗА нижним краем
	// каталожной оси EMASlow [50…200].
	if !containsValue(trend["EMASlow"], 30) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 30 — максимум замера (пара 10/30, PF 2.055) стоял бы вне сетки", trend["EMASlow"])
	}
	// И обрезана сверху: весь верх мёртв при ровном допуске, значит обрезка не отнимает выборку.
	for _, v := range trend["EMASlow"] {
		if v > 150 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: верх оси на BSPB мёртв — 10/150 даёт 1.020, 40/200 даёт 1.058", v)
		}
	}
	// Главное следствие расширения вниз: ни одной пары EMAFast >= EMASlow. Такая пара либо
	// вырождена (30/30 → 0.0% допуска), либо инвертирована (40/30 → «медленная над быстрой»).
	maxFast := 0.0
	for _, v := range trend["EMAFast"] {
		if v > maxFast {
			maxFast = v
		}
	}
	minSlow := 1e9
	for _, v := range trend["EMASlow"] {
		if v < minSlow {
			minSlow = v
		}
	}
	if maxFast >= minSlow {
		t.Errorf("cal_trend.json: max(EMAFast) = %v >= min(EMASlow) = %v — сетка порождает пары, где фильтр вырожден (EMAFast == EMASlow даёт 0.0%% допуска) или инвертирован", maxFast, minSlow)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestBSPBEarlyGridsPinTheirMeasuredAxes -v`
Expected: FAIL — каталога `data/params/rsi_pullback/bspb/` ещё нет, хелпер падает на чтении файла.

- [ ] **Step 3: Создать три файла ранних сеток**

`data/params/rsi_pullback/bspb/cal_screen.json`:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_screen.json — тема screen для BSPB (ПАО «Банк „Санкт-Петербург“»), шестнадцатый тикер каталога rsi_pullback. Тема меряет ЦЕНУ ДВУХ ОПЦИОНАЛЬНЫХ ГЕЙТОВ в сделках: 4 прогона (UseDayATRGate x UseVolume). РАННЯЯ ТЕМА — идёт поверх дефолтов ядра, как у всего каталога. Точечные замеры до прогона (36 месяцев, дефолты ядра): оба гейта в дефолтном положении (день включён, объём выключен) — 142 сделки, PF 1.114, net +6774; день ВЫКЛЮЧЕН — 495 сделок, PF 0.771; объём включён при VolMult 1.2 — 109 сделок, PF 1.040. Дневной гейт стоит дорого и работает; объёмный неразличим с выключенным (вся его ось лежит в полосе 1.04–1.13 вокруг baseline). ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_screen.json -out ./reports/BSPB_screen -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor. У ЭТОЙ темы -min-trades 1, а не 20: выключенный дневной гейт даёт втрое больше сделок, а включённые оба — втрое меньше, и порог 20 отсёк бы половину смысла темы. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "screen", "grid": {"UseDayATRGate": [0, 1], "UseVolume": [0, 1]}}]
}
```

`data/params/rsi_pullback/bspb/cal_entry.json`:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_entry.json — КЛЮЧЕВАЯ тема entry для BSPB, 196 прогонов (RSIUpper x RSIPeriod x RSILower = 7 x 4 x 7). РАННЯЯ ТЕМА — поверх дефолтов ядра: на ней стоит планка, и только прогон от дефолтов делает вердикт BSPB сравнимым с пятнадцатью предшественниками. ОСЬ ПЕРИОДА РАСШИРЕНА ВНИЗ ДО 3 — отмена правила каталога для этой бумаги, решение владельца 2026-08-24. Правило гласило: тройка не берётся, кроссов у неё вдвое больше, чем у четвёрки. На BSPB отношение ровно такое (660 против 311 на уровне 10), но проверка СДЕЛКАМИ его опровергает: RSI(3)@15 — 94 сделки при PF 1.327, RSI(4)@15 — 56 при 1.068. Кроссы вниз на будних барах расчётного окна: RSI(3) 660/1092/1603/2047/2505/2892/3198, RSI(4) 311/622/1008/1463/1885/2272/2642, RSI(5) 164/373/687/1074/1503/1887/2285, RSI(6) 104/259/468/803/1188/1637/2026 (уровни 10/15/20/25/30/35/40). ОСЬ ОБОРВАНА НА 6, хотя кроссов хватало бы и на 7 (RSI(7)@10 — 54, выше планки живого угла каталога 29): обрывает её замер сделками — RSI(7)@10 даёт 2 сделки, RSI(8)@10 НОЛЬ, RSI(7)@15 — 10, RSI(8)@15 — 7. На BSPB ось периода меряется сделками, а не кроссами, и это отличие от DIAS, ELFV и SIBN. RSILower не расширяется выше 40: каталог дважды получил отрицательный результат от такой растяжки (WUSH 2.000 -> 1.674 при растяжке до 50); выше 50 отката нет по определению. RSIUpper расширен до 85, потому что разворот полосы выхода замерен между 75 (PF 1.806) и 80 (1.636), то есть на верхнем крае каталожной оси. ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_entry.json -out ./reports/BSPB_entry -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "entry", "grid": {"RSIUpper": [55, 60, 65, 70, 75, 80, 85], "RSIPeriod": [3, 4, 5, 6], "RSILower": [10, 15, 20, 25, 30, 35, 40]}}]
}
```

`data/params/rsi_pullback/bspb/cal_trend.json`:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_trend.json — ВТОРАЯ КЛЮЧЕВАЯ тема trend для BSPB, 21 прогон (EMAFast x EMASlow = 3 x 7). РАННЯЯ ТЕМА — поверх дефолтов ядра. ОКНО ОСИ СДВИНУТО ВНИЗ ЦЕЛИКОМ, и оба края двигает один замер (точечные прогоны поверх RSI(3)@15, SpentDayATR 0.9, стоп 0.7, цель 0.6): 10/30 -> PF 2.055 на 55 сделках, 5/50 -> 1.950/52, 10/40 -> 1.672/59, 5/40 -> 1.673/47, 10/100 -> 1.551/66, 10/150 -> 1.020/69, 20/100 -> 1.076/76, 20/150 -> 1.135/74, 30/150 -> 1.065/75, 40/200 -> 1.058/78. ВНИЗ добавлены 30 и 40: максимум оси стоял на её нижнем крае, а ось, чей максимум на границе, каталог читать не умеет (ошибка, разобранная на WUSH). СВЕРХУ убраны 170 и 200: весь верх мёртв, а допуск ровный — доля будних баров с EMAFast > EMASlow лежит в полосе 48.2–50.4% на ВСЕХ парах (самая ровная ось каталога, размах два процентных пункта), значит обрезка не отнимает выборку, а отнимает заведомо мёртвые комбинации. EMAFast ОБРЕЗАН ДО 20 — вынужденное следствие расширения вниз: при EMAFast >= EMASlow пара либо вырождена (30/30 и 40/40 дают 0.0% допуска), либо инвертирована (40/30 -> 51.2%, но это уже «медленная над быстрой», другой фильтр). Цена обрезки нулевая: EMAFast 30 и 40 замерены мёртвыми на любой медленной (1.05–1.14). ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_trend.json -out ./reports/BSPB_trend -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "trend", "grid": {"EMAFast": [5, 10, 20], "EMASlow": [30, 40, 50, 70, 100, 120, 150]}}]
}
```

- [ ] **Step 4: Запустить тесты пакета**

Run: `go test ./internal/service/backtest/ -run 'TestBSPB|TestRSIPullback' -v`
Expected: PASS, включая общие `TestRSIPullbackCalFilesValid` (проверяет, что `_comment` называет
собственный путь файла) и `TestRSIPullbackGridControlPoints`.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/bspb internal/service/backtest/rsi_pullback_bspb_grid_test.go
git commit -m "feat(rsi_pullback): ранние сетки BSPB с замеренными осями"
```

---

### Task 2: Пакет `strategy/bspb` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/bspb/bspb.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/bspb/bspb_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `bspb.Ticker` (строка `"BSPB"`) и `bspb.DefaultParams() core.Params` — их используют
  Task 13 (литерал) и Task 14 (реестр живого раннера).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/bspb/bspb_test.go`:

```go
package bspb

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка BSPB ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 13, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("BSPB ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsBSPB(t *testing.T) {
	if Ticker != "BSPB" {
		t.Fatalf("Ticker = %q, want BSPB", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/bspb/ -v`
Expected: FAIL — пакета `bspb` не существует.

- [ ] **Step 3: Создать пакет**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/bspb/bspb.go`:

```go
// Package bspb supplies the ticker and rsi_pullback Params for BSPB (ПАО «Банк „Санкт-Петербург“»,
// обыкновенные акции, лот 10).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. Пакет возвращает core.DefaultParams() — baseline ядра, не
// подобранный под этот инструмент. Так и должно быть до конца калибровки: пакет заведён заранее,
// чтобы прогоны шли через тот же реестр, что и у остальных пятнадцати тикеров, а не через
// generic-ветку. Состояние держит bspb_test.go.
//
// СХЕМА ПРОГОНОВ ШТАТНАЯ — впервые за пять тикеров. История ровно 36 месяцев (получасовая серия
// 2023-08-24 … 2026-08-24, 35 261 бар), поэтому протокол §8 docs/rsi_pullback/strategy.md
// (-months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor) исполняется
// без адаптации: четыре фолда train 12 + OOS 6 встык. IVAT шёл по 25/9/4, DIAS по 30/10/5 — оба
// из-за короткой истории; числа BSPB сопоставимы построчно со всем остальным каталогом.
//
// ЗАПАС ИСТОРИИ НУЛЕВОЙ. Поставщик отдаёт 30m ровно за три года, и расчётное окно совпадает с
// физическим. -refresh во время калибровки НЕ запускать: он сдвинет обе границы, а не только
// правую, как это было на DIAS.
//
// АПРИОР, записанный ДО прогонов. Скринер (pullback_screen_Minutes30_20260804_232456.md, строка 13
// из 99 прошедших вселенную): оборот 363 млн ₽, дневной ATR 3.09%, TradesMed 51, PFmed 1.56,
// Capped 2/24, SilentCfg 0/24, плато 50%, PFmed HO 0.26 на 3 сделках. Контрольный прогон дефолтов
// ядра на расчётных 36 месяцах: 142 сделки, PF 1.114, net +6774 ₽ — САМЫЙ СЛАБЫЙ baseline
// каталога. Веса априору не придаётся: вопрос «предсказывают ли колонки скринера исход протокола»
// каталог закрыл трижды подряд (SIBN, ELFV, DIAS) — не предсказывают ни снизу, ни сверху.
//
// ЧТО ЗНАЕМ ОБ ИНСТРУМЕНТЕ ДО ПРОГОНОВ (замеры на расчётном окне):
//
//   - Режим СБАЛАНСИРОВАННЫЙ, зеркало DIAS: два растущих полугодия (+28.5% и +8.1%) против
//     четырёх падающих, итог окна −15.9%, максимальная просадка −49.0% (пик 419.97 2025-08-09).
//     Оба режима внутри окна есть — в отличие от DIAS, где падали все шесть.
//   - Шаг цены 0.01 ₽ при медианной цене 339.00 = 0.0029% — САМЫЙ ДЕШЁВЫЙ в каталоге (у DIAS
//     0.0165%, у ELFV 0.039%). Реальный круг издержек ≈ 0.106% против моделируемых 0.1%.
//   - Ликвидность ЛУЧШАЯ из всех кандидатов: медиана оборота 333 млн ₽/день, p10 127 млн. Гейт
//     скринера в 50 млн проходится и по медиане, и по p10.
//   - ДИВИДЕНДНЫЕ ОТСЕЧКИ — худшее место бумаги: семь гэпов хуже −3% за 36 месяцев (у SIBN пять,
//     у DIAS два), худший −8.44% (2024-05-06), цикл регулярный — начало мая и начало октября.
//     Стратегия держит позицию через ночь; стоп 0.7 ATR это 1.77% цены, такой гэп пробивает его
//     почти впятеро. Отличие от всего каталога: даты известны заранее.
//   - Дневной ATR(14) 2.53% медианой — НИЖЕ всего каталога.
//   - Дефолты ядра стоят почти в мёртвой зоне (PF 1.114), а edge живёт в другом углу: быстрый RSI
//     на глубоком уровне, исчерпанный день, короткий тренд-фильтр, короткая цель. Отсюда
//     гибридная процедура тем, описанная в спеке.
package bspb

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "BSPB"

// DefaultParams returns the baseline until calibration replaces it with a literal.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт рядом с остальными
(алфавитный порядок группы `rsipullback*`):

```go
	rsipullbackbspb "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"
```

и строку в карту `rsiPullbackRegistry` (рядом с `rsipullbackdias`):

```go
	rsipullbackbspb.Ticker:  rsiPullbackBindingFor(rsipullbackbspb.Ticker, rsipullbackbspb.DefaultParams),
```

- [ ] **Step 5: Добавить сторожевой тест baseline в реестр бэктеста**

В `internal/service/backtest/rsi_pullback_registry_test.go` добавить импорт:

```go
	rsipullbackbspb "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"
```

и тест:

```go
// TestRSIPullbackBSPBTracksBaseline сторожит ЧЕСТНОЕ состояние BSPB до конца калибровки: тикер
// зарегистрирован (иначе прогоны пошли бы в generic-ветку и молча считали бы не тот инструмент),
// но параметры равны baseline ядра. Тест заменяется снимком литерала в Task 13; пока он стоит,
// «почти откалиброванные» значения не могут просочиться в прод незамеченными.
func TestRSIPullbackBSPBTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["BSPB"]
	if !ok {
		t.Fatal("BSPB отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("BSPB: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if want := rsipullbackbspb.DefaultParams(); got != want {
		t.Fatalf("BSPB: реестр отдаёт не то, что пакет:\n got: %+v\nwant: %+v", got, want)
	}
	if got != core.DefaultParams() {
		t.Fatal("BSPB больше не отслеживает baseline — если калибровка закончена, замените этот тест снимком литерала")
	}
}
```

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'BSPB|RSIPullback' -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/bspb internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): пакет и реестр BSPB в состоянии «калибровка не проводилась»"
```

---

### Task 3: Тема `screen` — цена двух опциональных гейтов

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_screen.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1, реестр из Task 2.
- Produces: числа цены гейтов, на которые ссылаются `_comment` поздних тем и дока пакета.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_screen.json -out ./reports/BSPB_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Прочитать отчёт**

Открыть свежий файл в `reports/BSPB_screen/`. Выписать для каждой из четырёх комбинаций: pooled OOS
PF, число сделок в пуле, выбор по фолдам. Ожидание из точечных замеров (полная история, без
walk-forward): день включён + объём выключен — 142 сделки/1.114; день выключен — 495/0.771; объём
включён при 1.2 — 109/1.040. Прогон темы считает то же по фолдам, и числа будут другими — сравнивать
надо порядок, а не значения.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать после `РЕЗУЛЬТАТ ПРОГОНА 2026-08-24:` числа всех четырёх комбинаций и вывод: какой гейт
сколько стоит в сделках и в PF. Обязательно отметить, подтвердил ли прогон точечный замер, что
объёмный гейт неразличим с выключенным.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_screen.json
git commit -m "feat(rsi_pullback): BSPB, тема screen прогнана"
```

---

### Task 4: Тема `entry` — ключевая, тройка против четвёрки

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_entry.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: победитель pooled OOS — вход якоря для Task 6 (`RSIPeriod`, `RSILower`, `RSIUpper`).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_entry.json -out ./reports/BSPB_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа планки**

Из отчёта выписать: pooled OOS PF, число сделок в пуле, разбивку по четырём фолдам (PF и сделки
каждого), выбор ведущей оси `RSILower` в каждом фолде, выбор `RSIPeriod` в каждом фолде.

Критерий 1 планки: pooled OOS PF ≥ 1.5 при ≥ 20 сделках.
Критерий 2 планки: `RSILower` выбран одинаково в ≥ 3 фолдах из 4.
Критерий 3: фолд без единой убыточной сделки в пользу тикера не засчитывается — если такой есть,
отметить это отдельной строкой.

- [ ] **Step 3: Проверить, что решение по тройке подтвердилось**

Отдельно выписать, какой `RSIPeriod` выбрал каждый фолд. Это прямая проверка отступления от правила
каталога (риск 6 спеки): если тройку не выбрал ни один фолд, записать это честно — правило каталога
устояло, а точечный замер оказался свойством полной истории, а не бумаги.

- [ ] **Step 4: Проверить край оси периода**

Если победил `RSIPeriod = 6` (верхний край оси), проверить точечным прогоном, не растёт ли результат
дальше:

```bash
cat > /tmp/bspb_p7.json <<'EOF'
{"RSIPeriod":7,"RSILower":15,"RSIUpper":70,"EMAFast":10,"EMASlow":100,"DailyATRPeriod":14,"UseDayATRGate":1,"FreshDayATR":0,"SpentDayATR":0.8,"StopDailyATR":0.5,"TPDailyATR":0.6,"UseVolume":0,"VolBaseDays":14,"VolLookbackBars":3,"VolMult":1.2,"UseRSIExit":1,"UseTrail":0,"TrailDailyATR":0}
EOF
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/bspb_p7.json -out ./reports/BSPB_entry_edge -months 36
```

Ожидание из замеров спеки: 10 сделок — выборка мертва, край оси стоит правильно. Записать факт в
`_comment`.

- [ ] **Step 5: Записать результат в `_comment` сетки**

Дописать после `РЕЗУЛЬТАТ ПРОГОНА 2026-08-24:` все числа Steps 2–4 и **вердикт по обоим критериям
планки отдельными фразами** («критерий PF взят/не взят: …», «критерий устойчивости взят/не взят:
…»).

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_entry.json
git commit -m "feat(rsi_pullback): BSPB, тема entry прогнана"
```

---

### Task 5: Тема `trend` — вторая ключевая

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_trend.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: победитель pooled OOS — тренд якоря для Task 6 (`EMAFast`, `EMASlow`).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_trend.json -out ./reports/BSPB_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа планки и прочитать их правильно**

Выписать pooled OOS PF, сделки пула, разбивку по фолдам, выбор ведущей оси `EMASlow` в каждом фолде.

Ведущая ось темы `trend` — **`EMASlow`**, не `EMAFast`: так считает планка у всего каталога.

Отдельно отметить, попал ли выбор на **30 или 40** — расширенную часть оси. Если да, расширение
оправдалось прямо; если победил 50 и выше, записать, что расширение цены не имело, но и вреда не
принесло (пары валидны, допуск ровный).

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать числа и вердикт по обоим критериям планки отдельными фразами.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_trend.json
git commit -m "feat(rsi_pullback): BSPB, тема trend прогнана"
```

---

### Task 6: Фиксация якоря и семь поздних сеток

**Files:**
- Create: `data/params/rsi_pullback/bspb/cal_day.json`
- Create: `data/params/rsi_pullback/bspb/cal_day_spent.json`
- Create: `data/params/rsi_pullback/bspb/cal_volume.json`
- Create: `data/params/rsi_pullback/bspb/cal_vol_window.json`
- Create: `data/params/rsi_pullback/bspb/cal_risk.json`
- Create: `data/params/rsi_pullback/bspb/cal_exit.json`
- Create: `data/params/rsi_pullback/bspb/cal_trail.json`
- Modify: `internal/service/backtest/rsi_pullback_bspb_grid_test.go`

**Interfaces:**
- Consumes: победители тем `entry` (Task 4) и `trend` (Task 5); хелпер `bspbGrid` из Task 1.
- Produces: семь файлов поздних тем с вписанным якорем — их прогоняют Tasks 7–11.

- [ ] **Step 1: Собрать якорь по правилу спеки**

Правило (спека, раздел «Процедура тем», пункты 1–4):

1. `RSIPeriod`, `RSILower`, `RSIUpper` — из победителя **pooled OOS** темы `entry`; `EMAFast`,
   `EMASlow` — из победителя pooled OOS темы `trend`. Не из лучшего фолда, не из точечных замеров
   спеки.
2. Остальные девять полей — дефолты ядра: `DailyATRPeriod 14`, `UseDayATRGate 1`, `FreshDayATR 0`,
   `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`, `UseVolume 0`, `VolBaseDays 14`,
   `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`, `UseTrail 0`, `TrailDailyATR 0`.
3. Если победитель `entry` на полной истории даёт меньше 20 сделок — берётся второе место
   лидерборда, и оба кандидата с их числами пишутся в `_comment` всех семи файлов.
4. Якорь фиксируется один раз и между поздними темами не меняется.

Проверить выборку победителя точечным прогоном:

```bash
cat > /tmp/bspb_anchor.json <<'EOF'
{"RSIPeriod":<A>,"RSILower":<B>,"RSIUpper":<C>,"EMAFast":<D>,"EMASlow":<E>,"DailyATRPeriod":14,"UseDayATRGate":1,"FreshDayATR":0,"SpentDayATR":0.8,"StopDailyATR":0.5,"TPDailyATR":0.6,"UseVolume":0,"VolBaseDays":14,"VolLookbackBars":3,"VolMult":1.2,"UseRSIExit":1,"UseTrail":0,"TrailDailyATR":0}
EOF
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/bspb_anchor.json -out ./reports/BSPB_anchor -months 36
```

Здесь `<A>…<E>` — значения из Steps 2 задач 4 и 5. Записать сделки и PF якоря: они пойдут в
`_comment` всех семи файлов как точка отсчёта, относительно которой читается каждая поздняя тема.

- [ ] **Step 2: Создать семь файлов**

Во всех семи файлах якорные поля вписываются в `grid` **однозначными списками** (список из одного
значения не размножает комбинации), а `_comment` начинается с блока
`ЯКОРЬ: RSIPeriod <A>, RSILower <B>, RSIUpper <C>, EMAFast <D>, EMASlow <E>, прочее — дефолты ядра;
на полной истории якорь даёт <N> сделок при PF <P>. ТЕМА УСЛОВНА от результата тем entry и trend:
её числа сравнимы внутри BSPB, но НЕ построчно с каталогом, где все десять тем шли от дефолтов.`

`cal_day.json` — 24 комбинации:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_day.json — ЯКОРЬ: <блок якоря>. Тема меряет ДВУСТОРОННИЙ дневной гейт, 24 прогона (FreshDayATR x SpentDayATR = 4 x 6). Оси из замеров BSPB: ветка «свежий день» отбирает 8.8% будних баров при 0.3, 16.6% при 0.4, 25.9% при 0.5; ветка «день исчерпан» — 64.2% при 0.6, 44.0% при 0.8, 34.7% при 0.9, 28.1% при 1.0, 16.3% при 1.25, 9.7% при 1.5 (n = 24 108 будних баров с известным ATR). Верхний край 1.5 СОХРАНЁН: он отбирает 9.7% баров — больше, чем на SVAV (7.7%), где край держался, и вдвое больше, чем на SIBN (5.3%), где его сдвигали на 1.3. Ноль в оси FreshDayATR обязателен: на всех прод-тикерах каталога победил именно он. Сам гейт стоит дорого и работает: без него 36 месяцев дают 495 сделок при PF 0.771 против 142 при 1.114. ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_day.json -out ./reports/BSPB_day -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "day", "grid": {"RSIPeriod": [<A>], "RSILower": [<B>], "RSIUpper": [<C>], "EMAFast": [<D>], "EMASlow": [<E>], "UseDayATRGate": [1], "FreshDayATR": [0, 0.3, 0.4, 0.5], "SpentDayATR": [0.6, 0.8, 0.9, 1.0, 1.25, 1.5]}}]
}
```

`cal_day_spent.json` — 8 комбинаций:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_day_spent.json — ЯКОРЬ: <блок якоря>. Тема меряет ТОЛЬКО ветку «день исчерпан» на уплотнённой оси, 8 прогонов. Уровни 0.7 (53.2% баров) и 1.1 (22.2%) добавлены к канону — приём, сработавший на SIBN, ELFV и DIAS; здесь он подкреплён прямым замером поверх RSI(3)@15: между 0.7 и 1.1 PF растёт с 1.273 до 1.598, а выборка падает со 132 сделок до 37. Прямое сравнение с cal_day.json показывает, сколько стоит открытая ветка свежего дня. ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_day_spent.json -out ./reports/BSPB_day_spent -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "day_spent", "grid": {"RSIPeriod": [<A>], "RSILower": [<B>], "RSIUpper": [<C>], "EMAFast": [<D>], "EMASlow": [<E>], "UseDayATRGate": [1], "FreshDayATR": [0], "SpentDayATR": [0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5]}}]
}
```

`cal_volume.json` — 20 комбинаций:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_volume.json — ЯКОРЬ: <блок якоря>. Тема меряет объёмный гейт, 20 прогонов (VolMult x VolBaseDays = 5 x 4). ОСЬ СУЖЕНА ДО 2.5. Точечные прогоны на дефолтах (36 месяцев, база 14, окно 3): выключен — 142 сделки/1.114, 1.0 — 120/1.116, 1.2 — 109/1.040, 1.5 — 92/1.133, 2.0 — 68/1.059, 2.5 — 56/0.839, 3.0 — 46/0.840, 4.0 — 31/0.527. Весь верх оси лежит в полосе 1.04–1.13 вокруг baseline: гейт неразличим с выключенным. Значение 2.5 оставляет 56 сделок = 19 на двенадцатимесячное обучающее окно, ниже -min-trades 20, и PF там уже ниже baseline — такая строка может победить только процедурным артефактом; 3.0 и выше тем более. То же решение, что на DIAS; обратное ELFV, где все 24 комбинации решётки били baseline. База при множителе 1.5: 3 дня — 95 сделок/0.990, 5 — 95/1.100, 10 — 90/0.970, 14 — 92/1.133. ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_volume.json -out ./reports/BSPB_volume -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "volume", "grid": {"RSIPeriod": [<A>], "RSILower": [<B>], "RSIUpper": [<C>], "EMAFast": [<D>], "EMASlow": [<E>], "UseVolume": [1], "VolMult": [1.0, 1.2, 1.5, 2.0, 2.5], "VolBaseDays": [3, 5, 10, 14]}}]
}
```

`cal_vol_window.json` — 18 комбинаций:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_vol_window.json — ЯКОРЬ: <блок якоря>. Тема меряет ОКНО объёмного гейта, 18 прогонов (VolLookbackBars x VolMult = 6 x 3). Третья в каталоге тема этой оси (после ELFV и DIAS) и ПЕРВАЯ на ликвидной бумаге — она разрешает спор двух предыдущих. Точечные прогоны (VolMult 2.0, база 14): окно 1 — 54 сделки/0.832, 2 — 60/0.911, 3 (дефолт ядра) — 68/1.059, 5 — 76/1.170, 8 — 93/1.021, 12 — 100/1.023, 16 — 105/1.026. Максимум на 5, дефолт хуже его на 0.111 PF, дальше плато. Край 12 берётся, чтобы плато попало ВНУТРЬ сетки; край 16 не нужен — на DIAS он ловил разворот, здесь разворот стоит на 5 и уже внутри. ELFV (18.9% дней короче двадцати баров) получил максимум ровно на дефолте и записал «тему не заводить»; DIAS (11.9%) получил максимум на 12 и это опроверг. BSPB ликвиден — 7.7% коротких дней, медиана 34 бара в дне — и максимум сместился на два бара. ГИПОТЕЗА, которую тема проверяет: чем ликвиднее бумага, тем ближе оптимум окна к дефолту ядра. ЧИТАТЬ ОСТОРОЖНО: ось ОСЛАБЛЯЕТ гейт, и при большом окне выбор надо проверить на эквивалентность UseVolume = 0, предпочтя выключенный гейт при неразличимых числах. ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_vol_window.json -out ./reports/BSPB_vol_window -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "vol_window", "grid": {"RSIPeriod": [<A>], "RSILower": [<B>], "RSIUpper": [<C>], "EMAFast": [<D>], "EMASlow": [<E>], "UseVolume": [1], "VolLookbackBars": [1, 2, 3, 5, 8, 12], "VolMult": [1.2, 1.5, 2.0]}}]
}
```

`cal_risk.json` — 30 комбинаций:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_risk.json — ЯКОРЬ: <блок якоря>. Тема меряет стоп и цель, 30 прогонов (StopDailyATR x TPDailyATR = 5 x 6). СТРОКА 0.3 ВОЗВРАЩЕНА (у domrf/ и elfv/ она вырезана): стоп 0.3 ATR это 0.76% цены при медианном дневном ATR 2.53%, реальный круг издержек на BSPB ≈ 0.106% (шаг цены 0.01 ₽ = 0.0029% медианной цены 339.00 — самый дешёвый тик каталога), значит круг съедает 14.0% риска — под чертой 17%, по которой строку резали. Выживаемость стопов (доля будних дней, чей размах достаёт уровня, n = 747): 0.3 — 99.6%, 0.5 — 92.4%, 0.7 — 74.2%, 1.0 — 45.6%, 1.3 — 24.8%, 1.5 — 17.1%. Верхний край 1.3 сохранён: при 1.5 стоп достаёт лишь 17.1% дней и перестаёт быть защитой, становясь способом вытеснить убыток в RSI-выход (капкан, разобранный на WUSH, LENT, LSNGP, IVAT, SVAV и SIBN). ОСЬ ЦЕЛИ МЕНЯЕТСЯ В ОБЕ СТОРОНЫ: добавлена 0.4, потому что весь edge живёт в коротких целях (0.5–0.6 бьют 1.0 на 0.1–0.2 PF при любом стопе) и нижний край каталожной оси оказался краем максимума; убраны 2.0 и 2.5, потому что колонки 1.0 и 1.5 совпадают ПОБАЙТОВО — цель шире дневного ATR на BSPB недостижима, RSI-выход или стоп срабатывают раньше. Строка 1.5 остаётся, чтобы это было видно в отчёте и чтобы выполнялась контрольная точка пакета TestRSIPullbackGridControlPoints (цель 1.5 шире самого широкого стопа 1.3). Замер стоп x цель поверх RSI(3)@15 и SpentDayATR 0.8: стоп 0.3 — 1.264/1.293/1.180/1.180, стоп 0.5 — 1.305/1.327/1.126/1.126, стоп 0.7 — 1.291/1.320/1.094/1.094, стоп 1.0 — 1.347/1.362/1.273/1.273 (цели 0.5/0.6/1.0/1.5). Ось стопа инертна: 1.26–1.36 на всём диапазоне, число сделок меняется с 96 до 93. ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_risk.json -out ./reports/BSPB_risk -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "risk", "grid": {"RSIPeriod": [<A>], "RSILower": [<B>], "RSIUpper": [<C>], "EMAFast": [<D>], "EMASlow": [<E>], "StopDailyATR": [0.3, 0.5, 0.7, 1.0, 1.3], "TPDailyATR": [0.4, 0.5, 0.6, 0.8, 1.0, 1.5]}}]
}
```

`cal_exit.json` — 7 комбинаций:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_exit.json — ЯКОРЬ: <блок якоря>, кроме RSIUpper, который эта тема и свипует. Тема меряет полосу выхода, 7 прогонов. ОСЬ РАСШИРЕНА ДО 85: замер поверх RSI(3)@15, EMA 10/100 и SpentDayATR 0.9 даёт 55 — 1.248, 60 — 1.320, 65 — 1.333, 70 — 1.551, 75 — 1.806, 80 — 1.636, 85 — 1.324, то есть максимум на 75 с разворотом между 75 и 80 — на верхнем крае каталожной оси. Уровень 85 живой на всей оси периода: кроссов вверх 623 у RSI(4), 379 у RSI(5), 252 у RSI(6). ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_exit.json -out ./reports/BSPB_exit -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "exit", "grid": {"RSIPeriod": [<A>], "RSILower": [<B>], "EMAFast": [<D>], "EMASlow": [<E>], "RSIUpper": [55, 60, 65, 70, 75, 80, 85]}}]
}
```

`cal_trail.json` — 12 комбинаций:

```json
{
  "_comment": "data/params/rsi_pullback/bspb/cal_trail.json — ЯКОРЬ: <блок якоря>. Тема меряет трейл против RSI-выхода, 12 прогонов (UseRSIExit x TrailDailyATR = 2 x 6). Ось каноническая. Точечный замер поверх RSI(3)@15, EMA 10/100, SpentDayATR 0.9: трейл 0.3 — PF 0.913, 0.5 — 1.401, 0.7 — 1.521, 1.0 — 1.551, что РАВНО результату без трейла (при 1.0 трейл не срабатывает ни разу). Ожидание: трейл на BSPB только портит, и тема должна это подтвердить или опровергнуть на фолдах. ЗАПУСК: go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/bspb/cal_trail.json -out ./reports/BSPB_trail -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-24: ",
  "phases": [{"name": "trail", "grid": {"RSIPeriod": [<A>], "RSILower": [<B>], "RSIUpper": [<C>], "EMAFast": [<D>], "EMASlow": [<E>], "UseRSIExit": [0, 1], "UseTrail": [1], "TrailDailyATR": [0, 0.3, 0.5, 0.7, 1.0, 1.3]}}]
}
```

- [ ] **Step 3: Расширить сторожевой тест осей**

Дописать в `internal/service/backtest/rsi_pullback_bspb_grid_test.go`:

```go
// TestBSPBLateGridsPinTheirAxesAndAnchor сторожит семь ПОЗДНИХ тем — тех, что идут поверх якоря,
// выбранного темами entry и trend. Гибридная процедура объявлена в спеке 2026-08-24 и вызвана
// тем, что дефолты ядра на BSPB стоят почти в мёртвой зоне (baseline PF 1.114, слабейший в
// каталоге): тема свипует только свои оси, а остальные поля берёт из дефолтов, поэтому восемь тем
// из десяти мерили бы шум. Ось объёмного гейта это показала прямо — вся она лежит в полосе
// 1.04–1.13 вокруг baseline.
//
// Тест прибивает два свойства. Первое: каждая поздняя тема ДЕЙСТВИТЕЛЬНО несёт якорь — все пять
// якорных полей присутствуют и однозначны (список из одного значения не свипует, а фиксирует).
// Второе: якорь ОДИН И ТОТ ЖЕ во всех семи файлах — иначе темы меряют свои оси из разных точек и
// их лидерборды несравнимы между собой.
func TestBSPBLateGridsPinTheirAxesAndAnchor(t *testing.T) {
	late := []string{
		"cal_day.json", "cal_day_spent.json", "cal_volume.json",
		"cal_vol_window.json", "cal_risk.json", "cal_exit.json", "cal_trail.json",
	}
	// cal_exit свипует RSIUpper — это его тема, поэтому якорным полем он его не считает.
	anchorFields := map[string][]string{
		"cal_exit.json": {"RSIPeriod", "RSILower", "EMAFast", "EMASlow"},
	}
	defaultFields := []string{"RSIPeriod", "RSILower", "RSIUpper", "EMAFast", "EMASlow"}

	anchor := map[string]float64{}
	for _, file := range late {
		g := bspbGrid(t, file)
		fields := defaultFields
		if custom, ok := anchorFields[file]; ok {
			fields = custom
		}
		for _, f := range fields {
			values := g[f]
			if len(values) != 1 {
				t.Errorf("%s: %s = %v, want ровно одно значение — якорь фиксируется, а не свипуется", file, f, values)
				continue
			}
			if seen, ok := anchor[f]; ok && seen != values[0] {
				t.Errorf("%s: %s = %v, а другая поздняя тема несёт %v — якорь обязан быть один и тот же во всех семи файлах", file, f, values[0], seen)
				continue
			}
			anchor[f] = values[0]
		}
	}

	// Ось объёмного гейта сужена до 2.5: при 2.5 остаётся 56 сделок = 19 на двенадцатимесячное
	// обучающее окно при -min-trades 20, и PF там 0.839 — ниже baseline 1.114.
	volume := bspbGrid(t, "cal_volume.json")
	for _, v := range volume["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: на BSPB это меньше 56 сделок за 36 месяцев (19 на обучающее окно) при PF ниже baseline", v)
		}
	}

	// Окно объёмного гейта: максимум замера на 5, край 12 нужен, чтобы плато было видно внутри.
	window := bspbGrid(t, "cal_vol_window.json")
	if !containsValue(window["VolLookbackBars"], 5) {
		t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит 5 — это замеренный максимум оси (PF 1.170 против 1.059 у дефолта ядра)", window["VolLookbackBars"])
	}

	// Стоп: строка 0.3 возвращена, верхний край 1.3 сохранён.
	risk := bspbGrid(t, "cal_risk.json")
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на BSPB круг издержек съедает там 14.0%% риска, под чертой 17%%", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: такой стоп достаёт меньше 17%% дней и вытесняет убыток в RSI-выход", v)
		}
	}
	// Цель: 0.4 добавлена, всё выше 1.5 убрано как недостижимое.
	if !containsValue(risk["TPDailyATR"], 0.4) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит 0.4 — весь edge BSPB живёт в коротких целях", risk["TPDailyATR"])
	}
	for _, v := range risk["TPDailyATR"] {
		if v > 1.5 {
			t.Errorf("cal_risk.json свипует TPDailyATR=%v: цель шире дневного ATR на BSPB недостижима — колонки 1.0 и 1.5 совпадают побайтово", v)
		}
	}

	// Полоса выхода расширена до 85, чтобы разворот (максимум на 75) стоял внутри оси.
	exit := bspbGrid(t, "cal_exit.json")
	if !containsValue(exit["RSIUpper"], 85) {
		t.Errorf("cal_exit.json: RSIUpper = %v, не содержит 85 — без него максимум полосы (75) стоит у края", exit["RSIUpper"])
	}
}
```

- [ ] **Step 4: Запустить тесты**

Run: `go test ./internal/service/backtest/ -run 'TestBSPB|TestRSIPullback' -v`
Expected: PASS — все десять файлов валидны, якорь единый, оси прибиты.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/bspb internal/service/backtest/rsi_pullback_bspb_grid_test.go
git commit -m "feat(rsi_pullback): BSPB, якорь зафиксирован и поздние сетки созданы"
```

---

### Task 7: Темы `day` и `day_spent` — дневной гейт поверх якоря

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_day.json`, `cal_day_spent.json` (только `_comment`)

**Interfaces:**
- Consumes: якорь из Task 6.
- Produces: выбор `FreshDayATR` и `SpentDayATR` для сборки точки (Task 12).

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_day.json -out ./reports/BSPB_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_day_spent.json -out ./reports/BSPB_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сравнить темы прямо**

Выписать pooled OOS PF и сделки обеих тем. Разница между ними — **цена открытой ветки «свежего
дня»**: `cal_day` её свипует, `cal_day_spent` держит на нуле. Записать эту цену числом.

- [ ] **Step 3: Проверить верхний край 1.5**

Если победил `SpentDayATR = 1.5` (верхний край), проверить точечным прогоном, сколько сделок он
оставляет на полной истории: замер спеки поверх RSI(3)@15 даёт 14 сделок за 36 месяцев — это меньше
пяти на обучающее окно, и такой выбор в точку брать нельзя даже при высоком PF. Записать вывод.

- [ ] **Step 4: Записать результаты в `_comment` обеих сеток**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_day.json data/params/rsi_pullback/bspb/cal_day_spent.json
git commit -m "feat(rsi_pullback): BSPB, темы дневного гейта прогнаны"
```

---

### Task 8: Тема `volume` — объёмный гейт поверх якоря

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_volume.json` (только `_comment`)

**Interfaces:**
- Consumes: якорь из Task 6.
- Produces: решение о `UseVolume` для точки (Task 12).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_volume.json -out ./reports/BSPB_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить лидерборд с выключенным гейтом**

Тема свипует только включённый гейт (`UseVolume [1]`), поэтому сравнить её победителя надо с
якорем, у которого гейт выключен (число из Task 6 Step 1). Если победитель темы не бьёт якорь —
записать это прямо: гейт не окупается, и в точку он не идёт.

Ожидание из точечных замеров: вся ось лежит в полосе 1.04–1.13 вокруг baseline 1.114, то есть гейт
неразличим с выключенным.

- [ ] **Step 3: Записать результат в `_comment` сетки**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_volume.json
git commit -m "feat(rsi_pullback): BSPB, тема volume прогнана"
```

---

### Task 9: Тема `vol_window` — окно объёмного гейта, третий замер оси в каталоге

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_vol_window.json` (только `_comment`)

**Interfaces:**
- Consumes: якорь из Task 6.
- Produces: ответ на вопрос спеки — «чем ликвиднее бумага, тем ближе оптимум окна к дефолту ядра».

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_vol_window.json -out ./reports/BSPB_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Ответить на вопрос каталога**

Выписать победившее окно и сравнить с двумя предшественниками: ELFV (18.9% дней короче двадцати
баров) дал максимум на дефолте 3, DIAS (11.9%) — на 12. BSPB несёт 7.7% таких дней. Записать,
подтвердилась ли формулировка спеки: чем ликвиднее бумага, тем ближе оптимум к дефолту ядра.

- [ ] **Step 3: Проверить, не эквивалентен ли выбор выключенному гейту**

Если тема выбрала большое окно (8 или 12), прогнать точечно ту же конфигурацию с `UseVolume = 0` и
сравнить сделки и PF:

```bash
cat > /tmp/bspb_novol.json <<'EOF'
{"RSIPeriod":<A>,"RSILower":<B>,"RSIUpper":<C>,"EMAFast":<D>,"EMASlow":<E>,"DailyATRPeriod":14,"UseDayATRGate":1,"FreshDayATR":0,"SpentDayATR":0.8,"StopDailyATR":0.5,"TPDailyATR":0.6,"UseVolume":0,"VolBaseDays":14,"VolLookbackBars":3,"VolMult":1.2,"UseRSIExit":1,"UseTrail":0,"TrailDailyATR":0}
EOF
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/bspb_novol.json -out ./reports/BSPB_vol_window -months 36
```

При неразличимых числах в точку идёт **выключенный гейт** как более простая конфигурация.

- [ ] **Step 4: Записать результат в `_comment` сетки**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_vol_window.json
git commit -m "feat(rsi_pullback): BSPB, тема окна объёмного гейта прогнана"
```

---

### Task 10: Тема `risk` — стоп и цель поверх якоря

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_risk.json` (только `_comment`)

**Interfaces:**
- Consumes: якорь из Task 6.
- Produces: выбор `StopDailyATR` и `TPDailyATR` для точки (Task 12).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_risk.json -out ./reports/BSPB_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить капкан широкого стопа**

Если победил `StopDailyATR` 1.0 или 1.3, посмотреть в отчёте распределение выходов: если доля
выходов по стопу мала, а по RSI велика, значит стоп не защищает, а вытесняет убыток в RSI-выход —
капкан, разобранный на WUSH, LENT, LSNGP, IVAT, SVAV и SIBN. Записать вывод явно.

- [ ] **Step 3: Отдельно проверить возвращённую строку 0.3**

Выписать результат всех шести целей при `StopDailyATR = 0.3` и сравнить с результатом при 0.5.
Строка возвращена в сетку по замеру издержек (14.0% риска при реальном круге 0.106%); проверка
должна показать, окупилась ли она.

- [ ] **Step 4: Проверить недостижимость широкой цели**

Сравнить колонки `TPDailyATR` 1.0 и 1.5 в лидерборде. Точечный замер спеки показал, что они
совпадают побайтово. Если тема это подтверждает, записать: цель шире дневного ATR на BSPB —
неработающая ручка, а не выбор.

- [ ] **Step 5: Записать результат в `_comment` сетки**

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_risk.json
git commit -m "feat(rsi_pullback): BSPB, тема risk прогнана"
```

---

### Task 11: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/bspb/cal_exit.json`, `cal_trail.json` (только `_comment`)

**Interfaces:**
- Consumes: якорь из Task 6.
- Produces: выбор `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` для точки (Task 12).

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_exit.json -out ./reports/BSPB_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/cal_trail.json -out ./reports/BSPB_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Разобрать выходы с поправкой на удержание**

Выписать победителей обеих тем. Для `exit` отдельно отметить, попал ли выбор на расширенную часть
оси (85) — и если да, проверить, не вырождена ли эта ветка: уровень 85 при быстром RSI достигается
редко, и выход может фактически не срабатывать, превращая конфигурацию в «держать до стопа или
цели».

Для `trail`: точечный замер спеки показал, что трейл только портит (0.3 → 0.913 против 1.551 без
трейла), а при 1.0 не срабатывает вовсе. Записать, подтвердили ли фолды это.

- [ ] **Step 3: Записать результаты в `_comment` обеих сеток**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/bspb/cal_exit.json data/params/rsi_pullback/bspb/cal_trail.json
git commit -m "feat(rsi_pullback): BSPB, темы выходов прогнаны"
```

---

### Task 12: Сборка точки, её walk-forward и проверка стоп-условия

**Files:**
- Create: `data/params/rsi_pullback/bspb/plateau_point.json`

**Interfaces:**
- Consumes: лидерборды всех десяти тем (Tasks 3–11).
- Produces: принятую точку — 18 полей `core.Params`, которые Task 13 переносит в литерал.

- [ ] **Step 1: Собрать точку из лидербордов тем**

Правило сборки, одинаковое для всего каталога: каждое поле берётся из темы, которая его меряет;
там, где выбор темы под остальными осями точки не выживает, поле проверяется точечным прогоном и
решение записывается в `_comment`.

Источники: `RSIPeriod`, `RSILower` — `cal_entry`; `RSIUpper` — `cal_exit` (тема entry свипует его
тоже, но exit меряет его поверх якоря); `EMAFast`, `EMASlow` — `cal_trend`; `FreshDayATR`,
`SpentDayATR` — `cal_day`/`cal_day_spent`; `UseVolume`, `VolMult`, `VolBaseDays`,
`VolLookbackBars` — `cal_volume`/`cal_vol_window`/`cal_screen`; `StopDailyATR`, `TPDailyATR` —
`cal_risk`; `UseRSIExit`, `UseTrail`, `TrailDailyATR` — `cal_trail`; `DailyATRPeriod` — 14, ось не
свипуется нигде.

- [ ] **Step 2: Создать файл точки**

`data/params/rsi_pullback/bspb/plateau_point.json` — все 18 полей списками из ОДНОГО значения
(этого требует `TestRSIPullbackPlateauFilesArePoints`). В `_comment` обязательно: откуда взято
каждое поле; оговорка, что точка собрана человеком, видевшим всю историю, и потому её числа нельзя
сравнивать с pooled OOS тем; вердикт по планке пункт за пунктом; замеры Steps 3–7.

- [ ] **Step 3: Прогнать точку тем же walk-forward**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/plateau_point.json -out ./reports/BSPB_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Выписать pooled OOS PF, сделки пула, PF и сделки каждого из четырёх фолдов.

- [ ] **Step 4: Проверить стоп-условие плана — пункты 1 и 2**

- pooled OOS PF < 1.0 → **СТОП**;
- меньше 20 сделок за расчётное окно → **СТОП**.

При срабатывании: числа приносятся владельцу, Tasks 13–16 НЕ выполняются.

- [ ] **Step 5: Проверить стоп-условие плана — пункт 3 (удвоенные издержки)**

```bash
go run ./cmd/backtest -ticker BSPB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/bspb/plateau_point.json -out ./reports/BSPB_point_comm \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.001
```

PF < 1.0 → **СТОП**. Это пункт, которого в каталоге не было; он добавлен потому, что baseline
дефолтов уходит в убыток от одного удвоения (1.114 → 0.881). Записать процент потери PF и сравнить
с каталогом: у DIAS точка теряла 19.7%, у ELFV 18.1%.

- [ ] **Step 6: Замерить плато соседями**

Прогнать точечно 4–6 конфигураций-соседей точки (по одному шагу в каждую сторону вдоль ведущих
осей: `RSILower` ±1 шаг сетки, `SpentDayATR` ±1 шаг, `TPDailyATR` ±1 шаг). Записать в `_comment`:
стоит ли точка на плато или на пике. Точка на пике — риск, и он записывается, а не скрывается.

- [ ] **Step 7: Проверить, что результат не держится одной неделей и одним режимом**

Разбить сделки точки по полугодиям расчётного окна и записать вклад каждого. Обязательно из-за
риска 9 спеки: два полугодия из шести росли, одно на +28.5%, и часть результата может принадлежать
росту 2024 года, а не стратегии. Если больше половины net-результата даёт одно полугодие — записать
это прямо (прецедент IVAT, где 85% результата делала одна неделя июля).

- [ ] **Step 8: Коммит**

```bash
git add data/params/rsi_pullback/bspb/plateau_point.json
git commit -m "feat(rsi_pullback): BSPB, принятая точка и её замеры"
```

---

### Task 13: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/bspb/bspb.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/bspb/bspb_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: точку из Task 12.
- Produces: `bspb.DefaultParams()`, возвращающий литерал, — его читают Tasks 14 и 15.

- [ ] **Step 1: Заменить тест baseline снимком литерала**

В `bspb_test.go` удалить `TestParamsTrackTheBaselineUntilCalibrated` и написать вместо него набор
тестов по образцу `dias_test.go`, где каждое утверждение несёт число из прогонов:

```go
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{ /* 18 полей принятой точки */ }
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал BSPB изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("BSPB вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

func TestStopStaysAboveTheCostFloor(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR < 0.3 {
		t.Fatalf("StopDailyATR = %v: на стопе 0.3 ATR издержки съедают 14.0%% риска, уже — ещё больше", p.StopDailyATR)
	}
}

func TestTargetIsArmed(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR <= 0 {
		t.Fatalf("TPDailyATR = %v, want > 0", p.TPDailyATR)
	}
}

func TestTargetStaysWithinReach(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR > 1.5 {
		t.Fatalf("TPDailyATR = %v: цель шире дневного ATR на BSPB недостижима — колонки 1.0 и 1.5 в теме risk совпадают побайтово", p.TPDailyATR)
	}
}

func TestRSIBandIsNotDegenerate(t *testing.T) {
	p := DefaultParams()
	if p.RSIUpper <= p.RSILower {
		t.Fatalf("полоса RSI вырождена: RSIUpper = %v, RSILower = %v", p.RSIUpper, p.RSILower)
	}
}

func TestTrendPairIsValid(t *testing.T) {
	p := DefaultParams()
	if p.EMAFast >= p.EMASlow {
		t.Fatalf("EMAFast = %d >= EMASlow = %d: фильтр вырожден (равные периоды дают 0%% допуска) или инвертирован", p.EMAFast, p.EMASlow)
	}
}

func TestTickerIsBSPB(t *testing.T) {
	if Ticker != "BSPB" {
		t.Fatalf("Ticker = %q, want BSPB", Ticker)
	}
}
```

К этому добавить тесты положения гейтов (`TestOnlySpentDayBranchIsArmed`, `TestVolumeGateStaysOff`
или `...StaysOn`, `TestTrailStaysOff` или `...StaysOn`) — по фактическому составу точки, и в каждом
сообщении об ошибке указать число из прогона, которое обосновывает выбор.

- [ ] **Step 2: Запустить и убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/bspb/ -v`
Expected: FAIL — пакет ещё возвращает baseline.

- [ ] **Step 3: Поставить литерал**

В `bspb.go` заменить тело `DefaultParams()` литералом точки и переписать доку пакета: состояние
(«откалиброван 2026-08-24»), вердикт по планке пункт за пунктом, замеры точки (pooled OOS,
пофолдовая разбивка, удвоенные издержки, плато, разбивка по полугодиям), гибридная процедура тем и
почему она понадобилась, отступление по `RSIPeriod = 3` и чем оно кончилось, риски инструмента
(дивидендные гэпы прежде всего).

- [ ] **Step 4: Заменить сторожевой тест в реестре бэктеста**

В `rsi_pullback_registry_test.go` заменить `TestRSIPullbackBSPBTracksBaseline` на
`TestRSIPullbackBSPBIsRegisteredAndCalibrated` по образцу DIAS: тикер в реестре, реестр отдаёт то
же, что пакет, и параметры **не равны** baseline.

- [ ] **Step 5: Запустить тесты и линт**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -v`
Expected: PASS.
Run: `./bin/golangci-lint run ./internal/...`
Expected: 0 issues.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/bspb internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): BSPB откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 14: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `bspb.Ticker`, `bspb.DefaultParams()` из Task 13.
- Produces: запись в `paramsByTicker` — её читает `ParamsFor` живого раннера и `cmd/pullparity`.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"` и строка в
`paramsByTicker` рядом с `dias.Ticker`:

```go
	bspb.Ticker:  bspb.DefaultParams(),
```

- [ ] **Step 2: Дописать комментарий карты**

Перед картой в `registry.go` уже стоят абзацы про каждый заведённый тикер. Дописать абзац про BSPB
на английском, в стиле соседних: штатная схема 36/12/6 (впервые за пять тикеров), вердикт по
планке, гибридная процедура тем и её цена, дивидендные гэпы как главный эксплуатационный риск,
нулевой запас истории.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -v`
Expected: PASS. В пакете есть два теста, которые прямо сторожат эту пару задач:
`TestEveryDefaultTickerIsRegistered` (каждый тикер боевой вселенной обязан быть в карте) и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (тикер, всё ещё отслеживающий baseline, в
боевую вселенную попасть не может). Второй — причина, по которой Task 13 идёт раньше Task 15:
поставь BSPB во вселенную до литерала, и он упадёт.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live
git commit -m "feat(rsi_pullback): BSPB в реестре живого раннера"
```

---

### Task 15: Боевая вселенная

**Files:**
- Modify: `internal/config/rsi_pullback.go`
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Modify: `internal/config/rsi_pullback_test.go:54` (список `want` в тесте дефолта вселенной)

**Interfaces:**
- Consumes: литерал из Task 13, реестр из Task 14.
- Produces: `RSI_PULLBACK_TICKERS` из шестнадцати тикеров.

- [ ] **Step 1: Проверить стоп-условие ещё раз**

Прежде чем заводить тикер в прод, перечитать числа Task 12 Steps 4–5. Все три пункта стоп-условия
не сработали — иначе эта задача не выполняется вовсе.

- [ ] **Step 2: Обновить тест дефолта**

В тесте пакета `internal/config`, который сторожит состав `Tickers`, добавить `"BSPB"` в ожидаемый
список шестнадцатым.

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -v`
Expected: FAIL — дефолт ещё из пятнадцати тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

В `internal/config/rsi_pullback.go` дописать `"BSPB"` в конец списка `Tickers` и добавить перед ним
комментарий-абзац в стиле соседних (DIAS, ELFV): когда заведён, вердикт по планке, числа точки,
**тип риска** (у BSPB он эксплуатационно-дивидендный, а не режимный, как у DIAS, и не
исполнительный, как у ELFV), гибридная процедура и её цена.

Во всех трёх env-файлах дописать `,BSPB` в конец `RSI_PULLBACK_TICKERS`.

- [ ] **Step 5: Обновить `live.md` §8**

Дописать BSPB в таблицу боевой вселенной: тикер, дата заведения, взята ли планка, pooled OOS PF
точки и число сделок, ключевой риск.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести BSPB в боевую вселенную"
```

---

### Task 16: Документация — разбор калибровки и принятый риск

**Files:**
- Modify: `docs/rsi_pullback/strategy.md`
- Modify: `docs/rsi_pullback/live.md`

**Interfaces:**
- Consumes: все числа Tasks 3–12.
- Produces: раздел BSPB в справочнике стратегии и риск в `live.md` §10.

- [ ] **Step 1: Дописать §8.0.1 `strategy.md`**

В таблицу откалиброванных тикеров добавить строку BSPB: схема прогонов (36/12/6 — штатная), вердикт
по планке, pooled OOS PF точки и сделки, дата.

- [ ] **Step 2: Дописать раздел с разбором прогонов**

Новый раздел про BSPB по образцу разделов DIAS и ELFV. Обязательно:

1. **Гибридная процедура тем** — зачем понадобилась (дефолты в мёртвой зоне, baseline 1.114),
   как устроена (ранние поверх дефолтов, поздние поверх якоря), чего стоила (поздние темы условны).
   Это первый такой случай в каталоге, и следующий тикер со слабым baseline должен его найти.
2. **Чем кончилось отступление по `RSIPeriod = 3`** — выбрали ли фолды тройку, и если да, устояла
   ли она в точке. Правило каталога либо получает исключение с обоснованием, либо восстанавливается.
3. **Третий замер оси `vol_window`** — подтвердилась ли формулировка «чем ликвиднее бумага, тем
   ближе оптимум окна к дефолту ядра». Это закрывает спор ELFV и DIAS.
4. **Режим** — два растущих полугодия из шести против нуля у DIAS, и что это значит для доверия к
   числам: часть результата принадлежит росту 2024 года.

- [ ] **Step 3: Дописать риск в `live.md` §10**

Новый риск номер 19 (после риска 18 про DIAS) — **дивидендные отсечки BSPB**:

- семь гэпов хуже −3% за 36 месяцев, худший −8.44% (2024-05-06), цикл регулярный: начало мая и
  начало октября;
- стратегия держит позицию через ночь, стоп 0.7 ATR = 1.77% цены такой гэп пробивает почти
  впятеро;
- отличие от всего каталога: **даты известны заранее** по календарю дивидендов, и это делает риск
  управляемым вручную — в отличие от режимного риска DIAS, который нечем предвидеть;
- условие пересмотра: первая живая сделка, открытая за день до отсечки.

Отдельным пунктом дописать, что ликвидность BSPB снимает исполнительный риск, записанный у ELFV,
DIAS и LENT: медиана оборота 333 млн ₽/день, p10 127 млн — гейт скринера в 50 млн проходится и по
медиане, и по p10.

- [ ] **Step 4: Проверить, что доки не противоречат числам**

Перечитать написанное рядом с `_comment` сеток и докой пакета: одни и те же числа обязаны совпадать
во всех трёх местах. Расхождение — ошибка, а не стилистика (на SVAV и DIAS такие расхождения
находились и исправлялись отдельными коммитами).

- [ ] **Step 5: Коммит**

```bash
git add docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки BSPB и принятый риск"
```

---

### Task 17: Финальная проверка

**Files:** нет изменений, только проверки (кроме Step 3).

- [ ] **Step 1: Полный гейт качества**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + проверка дрейфа моков — зелёные.

- [ ] **Step 2: Сверка живой сборки с бэктестом**

```bash
go run ./cmd/pullparity -tickers BSPB -months 24
```

Expected: ноль расхождений. **24 месяца, а не 36:** живой сборщик тянет дневные свечи окном
`dailyFetchDays = 730` (`live/marketdata/marketdata.go:47`), и на большем горизонте появляются
ожидаемые расхождения длины `Daily*` рядов (`maxDailyHorizonMonths`, выяснено на IVAT). Расхождение
на 24 месяцах означает, что живой раннер и бэктест считают сигнал по-разному, и заведение в прод
откатывается до выяснения.

- [ ] **Step 3: Записать результат сверки в `live.md` §9**

Строка вида «BSPB заведён 2026-08-24 и сверяется за 24 месяца (`go run ./cmd/pullparity -tickers
BSPB -months 24` — <N> баров, **ноль расхождений**)» рядом с такими же строками про IVAT, SVAV,
SIBN, ELFV и DIAS. Коммит: `docs(rsi_pullback): сверка BSPB — 24 месяца, ноль расхождений`.

- [ ] **Step 4: Сообщить владельцу результат**

Короткий отчёт: вердикт по планке пункт за пунктом; замеры принятой точки (pooled OOS, пофолдовая
разбивка, удвоенные издержки, плато, разбивка по полугодиям); что заведено в прод; какие риски
записаны; что осталось (первые живые сделки, условия пересмотра). Тремя отдельными строками:

1. **чем кончилось отступление по `RSIPeriod = 3`** — выбрали ли фолды тройку, устояла ли она в
   точке, и получает ли правило каталога исключение или восстанавливается;
2. **что дала гибридная процедура тем** — стоило ли делить темы на ранние и поздние, и по какому
   признаку опознавать следующий тикер, которому это понадобится;
3. **чем кончился третий замер оси `vol_window`** — подтвердилась ли связь ликвидности бумаги и
   оптимального окна, то есть закрыт ли спор ELFV и DIAS.

---

## Self-review

**Покрытие спеки.** Кэш и нулевой запас истории → Global Constraints, дока пакета (Task 2 Step 3),
Task 17; штатная схема 36/12/6 → Global Constraints и каждая команда прогона, дока пакета; априор →
дока пакета (Task 2 Step 3); Свойство 1 (режим) → дока пакета, Task 12 Step 7, Task 16 Step 2
пункт 4; Свойство 2 (шаг цены и издержки) → `_comment` `cal_risk` (Task 6), Task 12 Step 5, Task 13
Step 1 (`TestStopStaysAboveTheCostFloor`); Свойство 3 (ликвидность) → дока пакета, Task 16 Step 3;
Свойство 4 (дивгэпы) → дока пакета, Task 16 Step 3 (риск 19); Свойство 5 (трендовый допуск и
вырожденные пары) → сторож Task 1 Step 1, `_comment` `cal_trend`, Task 13 Step 1
(`TestTrendPairIsValid`); Свойство 6 (ATR и выживаемость стопов) → `_comment` `cal_risk`, Task 10
Step 2; Свойство 7 (объёмный гейт) → `cal_volume` и Task 8; Свойство 8 (окно гейта) →
`cal_vol_window` и Task 9; Свойство 9 (дневной гейт) → `cal_day`/`cal_day_spent` и Task 7;
Свойство 10 (где живёт edge) → оси всех сеток, гибридная процедура, Task 6; процедура тем →
Task 1 (три файла), Task 6 (якорь и семь файлов), сторож `TestBSPBLateGridsPinTheirAxesAndAnchor`;
правило сборки якоря → Task 6 Step 1; планка → Global Constraints, вердикт выносится в Tasks 4, 5,
13, 16; правило прода и стоп-условие → Global Constraints, Task 12 Steps 4–5, Task 15 Step 1;
артефакты спеки → задачи 1, 2, 6, 12, 13, 14, 15, 16; порядок работы спеки → порядок задач; риск 3
спеки (малочисленная зона edge) → Task 6 Step 1 пункт 3, Task 7 Step 3, Task 12 Step 6; риск 5
(вырожденные пары EMA) → сторож Task 1 и тест Task 13; риск 6 (отступление по тройке) → Task 4
Step 3, Task 16 Step 2 пункт 2, Task 17 Step 4 пункт 1.

**Плейсхолдеры.** В задачах 3–12 значения результатов прогонов не выдуманы: там, где число станет
известно только после запуска, стоит явное указание выписать его из отчёта — это данные, которых до
прогона не существует, а не плейсхолдер плана. Маркеры `<A>…<E>` в Task 6 и `<блок якоря>` —
единственные подстановки, и они определены правилом сборки якоря в том же шаге (Task 6 Step 1);
раньше Task 6 их значений не существует физически. Код сторожевых тестов, теста baseline, доки
пакета и трёх ранних сеток дан целиком; семь поздних сеток даны целиком с точностью до якоря.
Снимок литерала (Task 13) задан списком обязательных инвариантов плюс конкретными тестами, потому
что остальные его значения — результат Task 12.

**Согласованность имён.** `bspbGrid` определён в Task 1 и используется в Task 6;
`TestRSIPullbackBSPBTracksBaseline` (Task 2) заменяется на
`TestRSIPullbackBSPBIsRegisteredAndCalibrated` (Task 13); `TestParamsTrackTheBaselineUntilCalibrated`
(Task 2) заменяется на `TestCalibratedLiteralIsPinned` (Task 13); `bspb.Ticker` и
`bspb.DefaultParams()` объявлены в Task 2 и потребляются в Tasks 13–15. Импорт в реестре бэктеста
назван `rsipullbackbspb` — как соседние `rsipullbackdias`; в живом реестре — `bspb`, как соседний
`dias`.
