# VSMO под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести VSMO (ПАО «Корпорация ВСМПО-АВИСМА») до вердикта по стратегии `rsi_pullback`:
каталог максимально широких сеток, канонический тематический walk-forward, принятая точка,
прошедшая два риск-гейта и четыре пункта стоп-условия, литерал в пакете и заведение в боевую
вселенную двадцать третьим тикером.

**Architecture:** Процедура **каноническая**: все одиннадцать тем идут поверх дефолтов ядра — на
VSMO дефолт входа `RSILower` 30 стоит в локальном максимуме неглубокой части оси (25 → 1.492,
30 → 1.658, 35 → 1.400), а глобальный максимум 10 вырожден по объёму (21 сделка за три года),
поэтому якорь (BSPB, SOFL) не нужен и числа тем сравнимы с каталогом построчно. Схема прогонов
**36/12/6** (четыре фолда, проверено контрольным прогоном до написания спеки), плюс **обязательный
контрольный прогон принятой точки по схеме 24/12/3** — внутри окна сессия расширилась почти вдвое
(19 → 34 получасовых бара в буднем дне). Два риск-гейта: гейт выживаемости применяется к
эффективной защите `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, гейт просадки — max DD точки
≤ 1.3 × max DD baseline. **Особенность VSMO:** шаг цены 20 ₽ при цене 22–24 тыс ₽ делает модель
издержек оптимистичной, поэтому стоп-условие получает четвёртый пункт по издержкам.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-04-vsmo-rsi-pullback-prep-design.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех без исключения прогонах. Флаг `-interval Minutes30` обязателен в
  каждой команде: дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Схема прогонов каноническая:** `-months 36 -train-months 12 -test-months 6 -min-trades 20
  -metric profit_factor`, четыре фолда встык. У темы `screen` — `-min-trades 1`.
- **Расчётное окно 2023-09-04 … 2026-09-04, 36.0 месяца.** Обрезки по началу истории нет.
- **Число фолдов сверяется на первой же теме.** Контрольный прогон 2026-09-04 уже дал «Фолдов: 4»
  (pooled OOS PF 1.752 на 36/12/6 и 2.681 на 24/12/3), но сверка повторяется: если в шапке отчёта
  темы `screen` окажется «Фолдов: 3» — **остановиться и доложить владельцу**, остальные темы не
  запускать.
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш дотянут штатным top-up 2026-09-04:
  `VSMO_Minutes30.json` — 27 684 бара (2023-08-04 … 2026-09-04); `VSMO_Day1.json` — 1 133 свечи
  (2022-08-04 … 2026-09-03).
- **Дыр в серии нет.** Разрыв длиннее четырёх дней ровно один — новогодние каникулы
  2023-12-29 → 2024-01-03 (4.6 дня), календарный.
- **ПРОЦЕДУРА КАНОНИЧЕСКАЯ:** все темы поверх дефолтов ядра, якоря нет.
- **Сетки держатся максимально широкими.** Обрезок осей не делается; замеры, которые в узком
  каталоге были бы основанием вырезать край, идут в `_comment` как **предупреждения**. Жёсткие
  инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower` на всех парах, ось тренда не
  порождает пар `EMAFast ≥ EMASlow`, `StopDailyATR` нигде не равен нулю.
- **Правило сборки точки:** поле берётся из темы, которая его меряет, и принимается только если за
  значение высказались **не менее трёх фолдов из четырёх**; иначе поле остаётся на **дефолте ядра**.
  Ничья 2/2 большинством не считается.
- **РИСК-ГЕЙТ A — выживаемость эффективной защиты.** Принимается только защита, достижимая **не
  менее чем в 30% будних дней окна**, и гейт применяется к `min(StopDailyATR, TrailDailyATR при
  UseTrail=1)`. Таблица выживаемости VSMO (n = 764): 0.3 — 97.9%, 0.4 — 92.8%, 0.5 — 85.1%,
  0.6 — 75.9%, 0.7 — 64.7%, 0.8 — 54.5%, **1.0 — 38.6%**, 1.3 — 21.2%, 1.5 — 13.0%, 2.0 — 6.3%.
  **Потолок — 1.0.** Если тема голосует выше, в точку идёт ближайшее значение оси внутри гейта, а
  цена решения в PF пишется прямым текстом. Порог 30% после начала прогонов не двигается.
- **РИСК-ГЕЙТ B — просадка точки.** Max DD принятой точки на полном окне (одиночный прогон
  `-params`) **не превышает 1.3 × max DD baseline**: 5.49% × 1.3 = **7.14%**.
- **Третий контур — риск-профиль точки против baseline.** Опорные числа baseline: доля SL-выходов
  **22.2%**, удержание медиана **8** / p90 **21** / максимум **47** бара, ночёвок **46.5%**, выходов
  в выходную сессию **6.2%**, max DD **5.49%**. Точка меряется по всем пяти. Выходная сессия —
  отдельный риск: её медианный оборот 1.97 млн ₽, проскальзывание там не моделируется.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` (каноническая, не
  `trend_low`) обе дают pooled OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для
  `entry`, `EMASlow` для `trend`) выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд в пользу
  тикера не засчитывается.
- **Стоп-условие из ЧЕТЫРЁХ пунктов.** Работа останавливается, если принятая точка даёт: (1) pooled
  OOS PF < 1.0 на 36/12/6, **либо** (2) меньше 20 сделок в пуле OOS, **либо** (3) pooled OOS PF < 1.0
  на контрольном прогоне 24/12/3, **либо** (4) pooled OOS PF < 1.0 на 36/12/6 при `-commission
  0.001`. При срабатывании — числа владельцу, **задачи 12–16 не выполняются**, маршрут выбирает
  владелец.
- **Издержки — ПУНКТ СТОП-УСЛОВИЯ, а не предупреждение** (отличие VSMO от всего каталога, решение
  владельца 2026-09-04). Настоящий круг VSMO **0.132–0.182%** (шаг 20 ₽ при медианной цене окна
  30 200 ₽ и последней 24 380 ₽) против моделируемых 0.1%: модель **оптимистична**, а не
  консервативна, как была у TGKA. Чувствительность baseline: 0.1% → 1.658, 0.2% → 1.379,
  0.3% → 1.148, 0.4% → 0.946.
- **Правило прода:** литерал ставится и VSMO заводится в `RSI_PULLBACK_TICKERS` двадцать третьим,
  **если не сработал ни один пункт стоп-условия** — независимо от того, взята планка или нет.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают все темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`, `UseDayATRGate 1`,
  `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`, `UseVolume 0`,
  `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`, `UseTrail 0`,
  `TrailDailyATR 0`. Контрольный прогон дефолтов: **144 сделки, PF 1.658, net +41 276.63 ₽**, win
  rate 71.53%, max DD 5.49%, выходы RSI 96 / SL 32 (22.2%) / TP 16. На 24 месяцах — 111 сделок,
  PF 1.829. Walk-forward дефолтов: 36/12/6 → 1.752/113 (фолды 1.290/40, 1.681/27, 3.958/23,
  1.985/23 — все прибыльны); 24/12/3 → 2.681/46 (фолд 3 вырожден: 5107.696 на 7 сделках).
- **Полугодия baseline** (нужны в Task 11): 2023-09..2024-03 — 13 сделок, PF 1.337, +2 921;
  2024-03..2024-09 — 18, 1.051, +368; 2024-09..2025-03 — 40, 1.290, +6 136;
  2025-03..2025-09 — 27, 1.731, +6 510; 2025-09..2026-03 — 23, 3.701, +11 136;
  2026-03..2026-09 — 23, 2.125, +14 206. **Все шесть прибыльны — такого baseline в каталоге не
  было.**
- **Ликвидность.** Медиана оборота будних дней (лот 1): 32.4 млн ₽ (36 мес), 29.5 (24 мес),
  **27.2 (12 мес)**; по годам 50.9 → 27.8 → 48.1 → **22.7 (2026)**. Гейт вселенной скринера 50 млн
  пройден **средним** (57 млн в отчёте `pullback_screen` 2026-08-04, место 24, PFmed 1.39,
  holdout PFmed 3.09/5, Plateau 71%, зажатых 0/24, молчащих 0/24). Прецеденты: NKHP (14 млн),
  TGKA (22 млн), LENT (38 млн). Выходная сессия: 252 дня, медиана 1.97 млн ₽. Дневной ATR(14)
  медиана 2.93%.
- **Режим:** buy&hold за окно **−53.0%**, по полугодиям −27.2%, −34.9%, +52.8%, −8.6%, −13.3%,
  −18.2% — одно растущее полугодие из шести.
- **Сессия расширилась внутри окна**: медиана баров в буднем дне 19 → 28 → 29 → 34 → 34 → 34 → 34.
  Отсюда контрольный прогон 24/12/3 и пункт 3 стоп-условия.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; полную команду запуска с путём
  `data/params/rsi_pullback/vsmo/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); место под
  строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md). Пер-тикерный
  разбор живёт в доке пакета `strategy/vsmo` и в `_comment` файлов сеток.
- **Каталог `reports/` — в `.gitignore`.**
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/vsmo-pullback-prep` от `feat/tgka-pullback-prep` (`2d349fa`), спека уже
  закоммичена (`df28abb`).

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/vsmo/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_vsmo_grid_test.go`

**Interfaces:**
- Consumes: helper `rsiPullbackTickerGrid(t, "vsmo", "<file>.json")` из
  `internal/service/backtest/rsi_pullback_grid_test.go` (возвращает `map[string][]float64` для
  первой фазы файла); общий валидатор `TestRSIPullbackCalFilesValid` требует непустой `_comment` с
  путём файла внутри.
- Produces: одиннадцать файлов сеток, на которые ссылаются Tasks 3–10.

- [ ] **Step 1: Написать падающий сторожевой тест осей** — файл
  `internal/service/backtest/rsi_pullback_vsmo_grid_test.go`, три теста по образцу
  `rsi_pullback_tgka_grid_test.go`:
  - `TestVSMOGridsStayWide` — проверяет НАЛИЧИЕ обязательных значений (лишние не запрещает), таблица
    `cases` ровно по таблице §4.2 спеки: `cal_screen.json` `UseDayATRGate` [0,1] и `UseVolume` [0,1];
    `cal_entry.json` `RSILower` [10,15,20,25,30,35,40,45,50], `RSIPeriod` [2,3,4,5,6,7,8,10],
    `RSIUpper` [55,60,65,70,75,80,85]; `cal_trend.json` `EMAFast` [3,5,8,10,15,20,30,40],
    `EMASlow` [50,75,100,150,200,250]; `cal_trend_low.json` `EMAFast` [3,5,8,10],
    `EMASlow` [20,30,40]; `cal_day.json` `FreshDayATR` [0,0.1,0.2,0.3,0.4,0.5,0.6],
    `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5]; `cal_day_spent.json`
    `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0]; `cal_volume.json`
    `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0,4.0], `VolBaseDays` [3,5,10,14,20]; `cal_vol_window.json`
    `VolLookbackBars` [1,2,3,5,8,12,16,24,32], `VolMult` [1.2,2.5,4.0]; `cal_risk.json`
    `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.3,1.5,2.0],
    `TPDailyATR` [0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.0,2.5]; `cal_exit.json`
    `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90]; `cal_trail.json`
    `TrailDailyATR` [0.3,0.5,0.7,1.0,1.5], `UseRSIExit` [0,1].
  - `TestVSMOEntryGridKeepsRSIUpperAboveRSILower` — по `cal_entry.json`: для каждой пары
    (`RSILower`, `RSIUpper`) требует `RSIUpper > RSILower`.
  - `TestVSMOTrendGridsKeepFastBelowSlow` — по обоим файлам тренда: ни одна пара
    (`EMAFast`, `EMASlow`) не должна давать `EMAFast >= EMASlow`.

  В доке теста записать четыре отступления от канона §8.1 с их замерами: `RSIPeriod` вниз до 2
  (1.534/360) и вверх до 10 (1.072/30) при исключённых 12 (0.745/17) и 14 (0.626/11);
  `EMAFast` 8 добавлен в обе сетки тренда (8 → 1.841/141 — максимум оси, канон его не содержит);
  `cal_trend_low.json` заведён отдельным файлом, потому что нижний угол (`EMASlow` 20 → 1.699,
  30 → 1.666, 40) канонической сетке при `EMAFast` до 40 недоступен; `StopDailyATR` получает узлы
  0.4, 0.6 и 0.8 сверх канона (0.6 → 1.675, 0.8 → 2.034 — первая точка капкана, где число сделок
  перестаёт меняться).

- [ ] **Step 2: Убедиться, что тест падает.**

Run: `go test ./internal/service/backtest/ -run TestVSMO`
Expected: FAIL — файлов сеток ещё нет.

- [ ] **Step 3: Создать одиннадцать файлов сеток** по таблице §4.2 спеки. Формат каждого файла:

```json
{
  "_comment": "data/params/rsi_pullback/vsmo/cal_<тема>.json — <что тема мерит>, N прогонов, поверх ДЕФОЛТОВ ЯДРА. ЗАМЕР 2026-09-04 (36 мес, in-sample): <ось: значение -> PF/сделок ...>. ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: <какой край чем опасен>. ЗАПУСК: go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/vsmo/cal_<тема>.json -out ./reports/VSMO_<тема> -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: <заполняется после прогона>",
  "phases": [
    { "name": "<тема>", "grid": { "<Поле>": [ ... ] }, "keepTop": 5 }
  ]
}
```

  Замеры для `_comment` берутся из §3 спеки целиком, по теме:
  - `screen`: `UseDayATRGate` 0 → 1.320/387, 1 → 1.658/144; `UseVolume` 0 → 1.658/144,
    1 → 1.704/110 (при дефолтной форме гейта). `keepTop` 4, запуск с `-min-trades 1`.
  - `entry`: полный ряд `RSILower` и `RSIPeriod` из §3.1 с пометкой, что максимум оси входа (10)
    вырожден по объёму, а 12 и 14 периода исключены как убыточные.
  - `trend` и `trend_low`: ряды `EMASlow` и `EMAFast` из §3.2 с пометкой, что обе оси пологие и
    тренд на VSMO фильтрует слабо.
  - `day` и `day_spent`: ряды `FreshDayATR` и `SpentDayATR` из §3.3 с пометкой, что `SpentDayATR`
    3.0 (2.213/6) исключён как вырожденный.
  - `volume` и `vol_window`: ряд из §3.4 с пометкой, что гейт даёт +0.33 PF ценой четверти сделок.
  - `risk`: ряды из §3.5 с прямой записью подписи капкана — от 0.8 и выше число сделок не меняется
    (136 → 135), и рост PF до 2.870 при стопе 2.0 куплен отодвинутым убытком, а не отбором входов;
    там же — потолок риск-гейта A (1.0, выживаемость 38.6%).
  - `exit` и `trail`: ряды из §3.6 с пометкой, что трейл от 1.2 побайтово равен его отсутствию.

- [ ] **Step 4: Прогнать тесты.**

Run: `go test ./internal/service/backtest/ -run 'TestVSMO|TestRSIPullback'`
Expected: PASS

- [ ] **Step 5: Коммит.**

```bash
git add data/params/rsi_pullback/vsmo internal/service/backtest/rsi_pullback_vsmo_grid_test.go
git commit -m "feat(rsi_pullback): каталог сеток VSMO"
```

---

### Task 2: Пакет `strategy/vsmo` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/vsmo/vsmo.go`, `vsmo_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params` и `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`; хелпер
  `rsiPullbackBindingFor(ticker string, params func() core.Params)` из
  `internal/service/backtest/rsi_pullback_registry.go`.
- Produces: `vsmo.Ticker` (константа `"VSMO"`) и `vsmo.DefaultParams() core.Params` — их используют
  Tasks 12, 13 и реестр бэктеста.

- [ ] **Step 1: Написать падающие тесты** в `vsmo_test.go`:

```go
package vsmo

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ПРОМЕЖУТОЧНОЕ состояние: пакет заведён под
// калибровку 2026-09-04, литерала ещё нет, и до вердикта он обязан отдавать ровно дефолты ядра.
// Тест снимается вместе с постановкой литерала (Task 12).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки VSMO обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsVSMO(t *testing.T) {
	if Ticker != "VSMO" {
		t.Fatalf("Ticker = %q, want VSMO", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/vsmo/`
Expected: FAIL — пакета ещё нет.

- [ ] **Step 3: Создать пакет** `vsmo.go` с честной шапкой в стиле соседних пакетов:

```go
// Package vsmo supplies the ticker and rsi_pullback Params for VSMO (ПАО «Корпорация
// ВСМПО-АВИСМА», обыкновенные акции, лот 1).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. DefaultParams() отдаёт baseline ядра до вердикта.
// ...
package vsmo

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "VSMO"

// DefaultParams returns the rsi_pullback parameters for VSMO.
func DefaultParams() core.Params { return core.DefaultParams() }
```

  В шапке записать: окно и схему прогонов; оба риск-гейта с числами (гейт A — потолок 1.0 при
  выживаемости 38.6%, гейт B — 7.14%); четыре пункта стоп-условия с прямым указанием, что четвёртый
  (издержки) заведён впервые в каталоге; априор скринера (24-е место, PFmed 1.39); особенности
  бумаги — шаг цены 20 ₽ при цене 22–24 тыс ₽ (круг 0.132–0.182% против моделируемых 0.1%, модель
  ОПТИМИСТИЧНА), ликвидность 27.2 млн ₽ по медиане 12 месяцев против гейта вселенной 50 млн, режим
  −53.0% за окно с одним растущим полугодием из шести, сессия расширилась 19 → 34 бара.

- [ ] **Step 4: Завести тикер в реестр бэктеста.** В `rsi_pullback_registry.go` добавить импорт
  `rsipullbackvsmo "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/vsmo"` и строку
  `rsipullbackvsmo.Ticker: rsiPullbackBindingFor(rsipullbackvsmo.Ticker, rsipullbackvsmo.DefaultParams),`
  — **обе в алфавитном порядке среди соседей** (`vsmo` идёт после `ugld`, перед `wush`/`ydex`
  согласно существующему порядку файла). В `rsi_pullback_registry_test.go` добавить
  `TestRSIPullbackVSMOTracksBaseline` по образцу соседнего теста TGKA: достать биндинг из
  `rsiPullbackRegistry[rsipullbackvsmo.Ticker]`, проверить `ok`, проверить, что параметры равны
  `rsipullbackvsmo.DefaultParams()`, и что алиас `rsipullbackvsmo` резолвится.

- [ ] **Step 5: Прогнать тесты.**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 6: Коммит.**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/vsmo internal/service/backtest
git commit -m "feat(rsi_pullback): пакет VSMO до калибровки"
```

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_screen.json -out ./reports/VSMO_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** проверить «Фолдов: 4» в шапке `*_walkforward.md` — иначе остановиться и доложить
  владельцу, остальные темы не запускать.
- [ ] **Step 3:** выписать pooled OOS PF, сделки пула, пофолдовые `UseDayATRGate` / `UseVolume`,
  пофолдовые OOS PF / сделки / MaxDD%. Ожидание из точечного замера: `UseDayATRGate=1` почти
  наверняка (без гейта 1.320/387), по `UseVolume` возможен свободный выбор в пользу единицы —
  точечно 1.704/110 против 1.658/144.
- [ ] **Step 4:** дописать строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: …` в `_comment` файла сетки.
- [ ] **Step 5: Коммит** `feat(rsi_pullback): VSMO, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_entry.json -out ./reports/VSMO_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему (504 прогона × 4 фолда).
- [ ] **Step 2:** проверить планку по критериям A (pooled OOS PF ≥ 1.5 при ≥ 20 сделках) и B
  (`RSILower` одинаков в ≥ 3 фолдах из 4). Записать «взят»/«провален» отдельно по каждому.
- [ ] **Step 3:** проверить края расширенной оси периода — победил ли `RSIPeriod` 10 (точечно
  1.072/30, порог `-min-trades 20` должен его отсечь внутри фолдов) или 2 (1.534/360). Победа
  любого края пишется прямым текстом.
- [ ] **Step 4:** отдельно записать, победил ли нижний край `RSILower` 10: точечно это 2.988 на
  21 сделке за три года, то есть ~7 сделок на обучающее окно — победа края означала бы вырожденный
  вход, и это предупреждение обязано попасть в `_comment`.
- [ ] **Step 5:** дописать результат в `_comment`, **Коммит** `feat(rsi_pullback): VSMO, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая ключевая и нижний угол

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_trend.json -out ./reports/VSMO_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_trend_low.json -out ./reports/VSMO_trend_low \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** планка проверяется **только по канонической** `cal_trend` (критерии A и B).
- [ ] **Step 3:** сравнить `trend_low` с `trend`: поле тренда идёт из `trend_low` **только** при
  большинстве ≥ 3/4 **и** превосходстве pooled OOS над канонической темой. Иначе — из `trend` (или
  дефолт ядра, если большинства нет). Решение пишется прямым текстом.
- [ ] **Step 4:** проверить края: победа `EMASlow` 50 (нижний край канонической оси) или `EMAFast` 3
  подтверждает точечный замер (`EMASlow` максимум 75 → 1.850, `EMAFast` максимум 8 → 1.841); победа
  верхних краёв (250, 40) идёт против него и пишется прямым текстом.
- [ ] **Step 5:** дописать результаты в оба `_comment`, **Коммит**
  `feat(rsi_pullback): VSMO, темы trend и trend_low`.

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_day.json -out ./reports/VSMO_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_day_spent.json -out ./reports/VSMO_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** сверить темы между собой: расхождение пофолдовых `SpentDayATR` — взаимодействие
  веток гейта, пишется прямым текстом.
- [ ] **Step 3:** проверить края: точечно `SpentDayATR` 1.0 → 1.922/102 и 1.25 → 1.857/60 — победа
  1.5 или 2.0 означала бы уход в вырожденную выборку (1.378/37 и 1.412/23) и записывается
  предупреждением.
- [ ] **Step 4:** дописать результаты, **Коммит** `feat(rsi_pullback): VSMO, темы day и day_spent`.

---

### Task 7: Тема `volume`

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_volume.json -out ./reports/VSMO_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** выписать пофолдовых победителей `VolMult` и `VolBaseDays`; проверить, победил ли
  край `VolBaseDays` 3 или 20 (точечно 20 — лучший: 1.2/20 → 1.990/107).
- [ ] **Step 3:** записать место VSMO в каталожной гипотезе «объём говорит на тонких бумагах»:
  оборот 27 млн ₽ по медиане 12 месяцев, соседи по гипотезе — NKHP (14 млн, единственный
  подтверждённый случай) и TGKA (22 млн, гейт не выбран).
- [ ] **Step 4:** дописать результат, **Коммит** `feat(rsi_pullback): VSMO, тема volume`.

---

### Task 8: Тема `vol_window`

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_vol_window.json -out ./reports/VSMO_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** сверить `VolMult` с темой `volume`; расхождение = форма гейта не откалибрована,
  `VolMult` идёт дефолтом ядра.
- [ ] **Step 3:** проверить верхний край `VolLookbackBars` 32 — насыщается ли ось (побайтовое
  равенство с 24 означает насыщение, как было на TGKA).
- [ ] **Step 4:** дописать результат, **Коммит** `feat(rsi_pullback): VSMO, тема vol_window`.

---

### Task 9: Тема `risk` — стоп, цель и обязательная проверка риск-гейта A

**Files:** также `data/params/rsi_pullback/vsmo/plateau_stop_10.json`, `plateau_stop_20.json`

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_risk.json -out ./reports/VSMO_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** применить риск-гейт A: если большинство фолдов выбрало `StopDailyATR` > 1.0,
  победитель темы **отвергается гейтом**, в точку идёт 1.0 (38.6% дней), цена решения в PF пишется
  прямым текстом.
- [ ] **Step 3:** снять два зонда капкана — файлы `plateau_stop_10.json` и `plateau_stop_20.json`
  (одноточечные сетки: дефолты ядра плюс поля, принятые темой, и `StopDailyATR` 1.0 и 2.0
  соответственно) на полном окне:

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/plateau_stop_10.json -out ./reports/VSMO_nb_stop_10 -months 36 -min-trades 1
```

  Выписать PF, сделки, **max DD**, выходы RSI/SL/TP с долей SL, удержание (медиана, p90, максимум),
  долю ночёвок, долю выходов в выходную сессию.
- [ ] **Step 4:** записать, воспроизводится ли на VSMO подпись капкана «PF вверх при неизменном
  пуле»: точечно 0.8 → 2.034/136, 1.0 → 2.224/136, 1.5 → 2.593/135, 2.0 → 2.870/135 против
  0.5 → 1.658/144. Проверить, растёт ли вместе с PF просадка — если нет, единственным работающим
  фильтром остаётся гейт A, и это пишется прямым текстом.
- [ ] **Step 5:** дописать результат, **Коммит**
  `feat(rsi_pullback): VSMO, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit` и `trail`

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_exit.json -out ./reports/VSMO_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/cal_trail.json -out ./reports/VSMO_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** проверить, не обходит ли трейл риск-гейт A (урок AFKS): дистанция трейла ниже
  принятого стопа делает её эффективной защитой, и гейт применяется к ней.
- [ ] **Step 3:** применить правило упрощения: дистанция ≥ 1.0 → `UseTrail 0`, `TrailDailyATR 0`
  (на VSMO точечно трейл 1.2 и 1.5 побайтово равны его отсутствию).
- [ ] **Step 4:** если тема выбрала `UseRSIExit 0` — записать отдельной строкой: точечно это
  1.346/129 против 1.658/144, то есть −0.31 PF.
- [ ] **Step 5:** дописать результаты, **Коммит** `feat(rsi_pullback): VSMO, темы exit и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и четыре пункта стоп-условия

**Files:** `data/params/rsi_pullback/vsmo/plateau_point.json`, соседи плато
`plateau_<поле>_<значение>.json`, `docs/superpowers/plans/task-11-report-vsmo.md`

- [ ] **Step 1:** собрать точку по правилу большинства (≥ 3 из 4; иначе дефолт ядра). Для каждого
  из восемнадцати полей записать голоса фолдов, решение и случайность совпадения с дефолтом.
- [ ] **Step 2:** применить риск-гейт A к собранной точке (эффективная защита
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, потолок 1.0).
- [ ] **Step 3:** прогнать точку схемой 36/12/6:

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/plateau_point.json -out ./reports/VSMO_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

  Выписать pooled OOS PF, сделки пула, пофолдовые in-sample → OOS PF / сделки / NetPnL% / MaxDD%.
- [ ] **Step 4:** проверить пункты 1 и 2 стоп-условия (PF ≥ 1.0, ≥ 20 сделок в пуле). Провал →
  **остановиться**, задачи 12–16 не выполнять, числа владельцу.
- [ ] **Step 5:** прогнать контрольную схему 24/12/3 (пункт 3):

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/plateau_point.json -out ./reports/VSMO_point_24 \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

  Baseline на том же куске даёт 2.681 на 46 сделках (с одним вырожденным фолдом) — если точка ниже
  baseline, это записывается прямым текстом.
- [ ] **Step 6:** проверить пункт 4 стоп-условия — тот же прогон схемой 36/12/6 с удвоенным кругом:

```bash
go run ./cmd/backtest -ticker VSMO -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/vsmo/plateau_point.json -out ./reports/VSMO_point_cost2 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.001
```

  pooled OOS PF < 1.0 → **остановиться**. Дополнительно снять строку `-commission 0.0015` как
  справочную (тройной круг, не пункт стоп-условия).
- [ ] **Step 7:** применить риск-гейт B на полной истории (одиночный прогон точки на 36 месяцах,
  `-out ./reports/VSMO_point_full`), потолок max DD **7.14%**.
- [ ] **Step 8:** снять анатомию и полугодия точки; сравнить с baseline по пяти числам третьего
  контура (доля SL 22.2%, удержание 8/21/47, ночёвки 46.5%, выходная сессия 6.2%, DD 5.49%).
- [ ] **Step 9:** снять соседей плато по каждому полю, принятому темой большинством (±один узел
  оси) — по одному файлу `plateau_<поле>_<значение>.json` на соседа; совпадение результата
  побайтово = инертная ось, и это пишется прямым текстом.
- [ ] **Step 10:** написать рабочую записку `docs/superpowers/plans/task-11-report-vsmo.md`: голоса
  по каждому полю, оба walk-forward, четыре пункта стоп-условия с числами, оба риск-гейта, соседи
  плато, полугодия, анатомия против baseline, вердикт по планке.
- [ ] **Step 11: Коммит** `feat(rsi_pullback): VSMO, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Задача выполняется только если ни один пункт стоп-условия не сработал.**

**Files:** `internal/service/trading_strategy/rsi_pullback/strategy/vsmo/vsmo.go`, `vsmo_test.go`,
`internal/service/backtest/rsi_pullback_registry_test.go`

- [ ] **Step 1:** заменить `TestParamsTrackTheBaselineUntilCalibrated` на
  `TestParamsAreTheAcceptedPoint` (снимок всех восемнадцати полей принятой точки, по образцу
  `nkhp_test.go`) плюс `TestPointDiffersFromTheCoreBaseline` (сторожит именно те поля, которыми
  точка отличается от дефолтов ядра, с их голосами в доке теста).
- [ ] **Step 2:** убедиться, что тесты падают.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/vsmo/`
Expected: FAIL — литерала ещё нет.

- [ ] **Step 3:** поставить литерал принятой точки в `DefaultParams()` (полный `core.Params{...}`,
  все восемнадцать полей явно).
- [ ] **Step 4:** заменить тест реестра `TestRSIPullbackVSMOTracksBaseline` на
  `TestRSIPullbackVSMOServesTheCalibratedPoint`.
- [ ] **Step 5:** тесты и линт.

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... && ./bin/golangci-lint run ./internal/...`
Expected: PASS

- [ ] **Step 6: Коммит** `feat(rsi_pullback): VSMO откалиброван — литерал вместо отслеживания baseline`.

---

### Task 13: Реестр живого раннера

**Files:** `internal/service/trading_strategy/rsi_pullback/live/registry.go`, `registry_test.go`

- [ ] **Step 1:** добавить импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/vsmo"`
  и строку `vsmo.Ticker: vsmo.DefaultParams(),` в карту `paramsByTicker` (алфавитный порядок).
- [ ] **Step 2:** дописать абзац комментария карты в том же англоязычном стиле, что у соседей:
  вердикт по планке, оба риск-гейта и их цена, четвёртый пункт стоп-условия по издержкам и принятые
  риски (круг 0.13–0.18% против моделируемых 0.1%, ликвидность 27 млн ₽ по медиане против гейта
  50 млн, выходная сессия 1.97 млн ₽, режим −53% за окно, расширившаяся сессия).
- [ ] **Step 3:** прогнать тесты.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/...`
Expected: PASS

- [ ] **Step 4: Коммит** `feat(rsi_pullback): VSMO в реестре живого раннера`.

---

### Task 14: Боевая вселенная

**Files:** `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`,
`env/prod.env`, `env/prod.env.example`, `env/local.env.example`, `docs/rsi_pullback/live.md`

- [ ] **Step 1:** перечитать числа Task 11 Steps 4–7 — ни один из четырёх пунктов стоп-условия не
  сработал, оба риск-гейта пройдены.
- [ ] **Step 2:** добавить `"VSMO"` в `want` теста конфига двадцать третьим.
- [ ] **Step 3:** убедиться, что тест падает.

Run: `go test ./internal/config/ -run RSIPullback`
Expected: FAIL

- [ ] **Step 4:** дописать `"VSMO"` в `Tickers` + комментарий-абзац в стиле соседних; дописать
  `,VSMO` в три env-файла (`env/prod.env`, `env/prod.env.example`, `env/local.env.example`).
- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  `docs/rsi_pullback/live.md` (только значение — пер-тикерного ничего).
- [ ] **Step 6:** прогнать тесты.

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 7: Коммит** `feat(rsi_pullback): завести VSMO в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:** `internal/service/trading_strategy/rsi_pullback/strategy/vsmo/vsmo.go`

- [ ] **Step 1:** переписать шапку пакета в итог калибровки: вердикт по планке по каждому критерию
  обеих ключевых тем, точка поле за полем с голосами фолдов, оба walk-forward, строки издержек
  (0.1% / 0.2% / 0.15%), соседи плато с пометкой инертных осей, шесть полугодий, анатомия против
  baseline, результат обоих риск-гейтов и их цена. Отдельными абзацами — принятые риски и условия
  пересмотра:
  1) **издержки**: круг 0.132–0.182% против моделируемых 0.1% (шаг 20 ₽), почему модель здесь
     оптимистична и как это закрыто четвёртым пунктом стоп-условия;
  2) **ликвидность**: медиана 27.2 млн ₽ за 12 месяцев против гейта вселенной 50 млн, падение
     50.9 → 22.7 млн по годам; **условие пересмотра: падение медианы ниже 15 млн ₽ — вывести тикер
     из боевой вселенной, не дожидаясь планового цикла**;
  3) **выходная сессия** с медианным оборотом 1.97 млн ₽ и долей выходов точки;
  4) **режим −53.0% за окно** при одном растущем полугодии из шести — стратегия проверена в
     основном против падения; **при первом растущем полугодии калибровку повторить вне планового
     цикла**;
  5) **сессия расширилась внутри окна** (19 → 34 бара) — отсюда контрольный прогон 24/12/3;
  6) **капкан широкого стопа** с неизменным пулом сделок и роль гейта A.
- [ ] **Step 2:** прогнать тесты.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 3:** механический вывод в `docs/rsi_pullback/strategy.md` — **только если** VSMO дал
  вывод о механике. Кандидаты названы заранее: объёмный гейт как второй подтверждённый случай
  сигнала на тонкой бумаге (§8.1) и капкан широкого стопа в форме «PF растёт при побайтово
  неизменном пуле сделок». Пер-тикерных чисел, дат и вердиктов в `docs/rsi_pullback/` не писать.
- [ ] **Step 4: Коммит** `docs(rsi_pullback): разбор калибровки VSMO и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** сверка живой сборки с бэктестом.

Run: `go run ./cmd/pullparity -tickers VSMO -months 36`
Expected: ноль расхождений

- [ ] **Step 2:** полный гейт.

Run: `./bin/mage ci`
Expected: зелёный

- [ ] **Step 3:** проверить, что отчёты не попали в индекс.

Run: `git status --porcelain | grep -c '^.. reports/'`
Expected: `0`

- [ ] **Step 4:** финальный коммит, если остались правки.

---

## Self-Review

**Покрытие спеки:** одиннадцать тем (§4.2) → Tasks 3–10; каноническая процедура без якоря (§4)
→ Global Constraints и Architecture; ширина сеток и исключённые вырожденные точки (§4.2) → Task 1 со
сторожевыми тестами; пакет и реестр бэктеста (§5) → Task 2; правило сборки точки (§4.3) → Task 11
Step 1; риск-гейт A (§4.4) → Task 9 Step 2, Task 10 Step 2, Task 11 Step 2; риск-гейт B (§4.5)
→ Task 11 Step 7; третий контур (§4.6) → Task 11 Step 8; планка (§4.7) → Tasks 4 и 5; четыре пункта
стоп-условия (§4.8) → Task 11 Steps 4–6; правило прода (§4.9) → Task 14 Step 1; литерал → Task 12;
реестр живого раннера → Task 13; боевая вселенная двадцать третьим тикером → Task 14; пер-тикерная
запись и принятые риски (§6) → Task 15; `pullparity` и `mage ci` (§5) → Task 16; проверка числа
фолдов (ловушка ASTR) → Task 3 Step 2; baseline и его числа (§2) → Global Constraints.

**Плейсхолдеры:** один — литерал в Task 12 (точка известна только после Task 11). Указано, откуда
берётся значение и по какому правилу.

**Согласованность имён:** `vsmo.Ticker` и `vsmo.DefaultParams()` заводятся в Task 2 и используются
в Tasks 12–14 под теми же именами; тест `TestParamsTrackTheBaselineUntilCalibrated` (Task 2)
заменяется на `TestParamsAreTheAcceptedPoint` в Task 12; `TestRSIPullbackVSMOTracksBaseline`
(Task 2) — на `TestRSIPullbackVSMOServesTheCalibratedPoint` (Task 12).

---

## Итог исполнения плана (2026-09-04)

**Все 16 задач выполнены, VSMO заведён в боевую вселенную.** Две поправки к тексту плана,
сделанные по факту:

1. **Порядковый номер — двадцать четвёртый, а не двадцать третий.** Двадцать третьим 2026-09-03 стал
   TGKA (на дефолтах ядра), и план был написан до этого факта.
2. **`pullparity` прогоняется на 24 месяцах, а не на 36.** Сам инструмент предупреждает, что при
   `-months 36` расхождения `Daily*` на ранних барах ожидаемы: окно `dailyFetchDays` живого
   сборщика короче, чем история, которую видит эталон (`maxDailyHorizonMonths`). На 24 месяцах —
   **0 расхождений на 20 730 барах**.

**Вердикт:** планка не взята двенадцатый раз подряд (`entry` — оба критерия, `trend` — только
устойчивость), но ни один из четырёх пунктов стоп-условия не сработал, оба риск-гейта пройдены.
Принятая точка отличается от дефолтов ядра двумя полями из восемнадцати: `EMAFast` 5 и `TPDailyATR`
0.3. Pooled OOS 1.814/116 (все четыре фолда прибыльны), контроль 24/12/3 — 3.558, удвоенный круг
издержек — 1.563, max DD на полном окне 4.92% против baseline 5.49%. `./bin/mage ci` — зелёный.
