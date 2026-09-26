# RAGR под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести RAGR (ПАО «Русагро», обыкновенные акции) до вердикта по стратегии
`rsi_pullback`. Нужно собрать двенадцать максимально широких сеток, прогнать тематический
walk-forward на окне после рестарта торгов и получить точку внутри риск-гейтов A и B, которая
проходит семь пунктов стоп-условия. Если первый круг провален — второй круг по правилам владельца.
Результат — либо литерал в пакете и заведение в боевую вселенную тридцать четвёртым тикером, либо
протокол отказа.

**Architecture:** Процедура каноническая (спека §5). Шесть отличий от каталога.

1. **Окно — только после рестарта торгов:** `-months 19`, схема сборки **19/7/3**, контроль
   **19/10/3**. Пауза редомициляции 2024-12-02 → 2025-02-17 в окно не попадает.
2. **`-min-trades 10`** вместо 20 (train семь месяцев).
3. **Главный критерий владельца в стоп-условии:** годы 2025 и 2026 в плюс.
4. **Жёсткий пункт устойчивости:** хвост 6 месяцев против хвоста 12 месяцев по дате входа.
5. **Потолок стопа 0.8** — строжайшее из двух правил гейта A.
6. **Гейт D меряет пять событийных шоков**, планка — не хуже baseline. Арбитры — `exit_target` и
   `trend_day`.

Разведка показала: дефолты прибыльны (PF 1.231/77), но проваливают 2025 год. Вход и стоп на RAGR
инертны или вредны; рычаги — выход (`RSIUpper` 35, `TPDailyATR` 0.15–0.2, трейл 0.5), тренд
(`EMAFast` 3, `EMASlow` 25) и свежий день (`FreshDayATR` 0.05). Каждый одиночный рычаг на полном
окне даёт оба года в плюс. Главный риск — рассыпание голосов на семимесячных train и фолде 4 из
семи сделок.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков), `python3` для
разбора журналов сделок.

**Spec:** `docs/superpowers/specs/2026-09-26-ragr-rsi-pullback-prep-design.md` — план ссылается на её
параграфы (§N), исполнитель читает оба документа.
**Сырые отчёты разведки:** `reports/RAGR_prep/` (вне git): `base19/` — baseline на полном окне
(`RAGR_rsi_pullback_Minutes30_20260926_091823.md`), `wf/1973/` — walk-forward дефолтов 19/7/3,
`ax/` — оси, `pt/` — точки.

## Global Constraints

**Ветка, кэш и окно**

- **Ветка:** `feat/ragr-pullback-prep` от `main` `5baebae` (в боевой вселенной 33 тикера, последний —
  `SVCB`). Спека закоммичена (`0f7917c`).
- **Таймфрейм `Minutes30` во всех прогонах.** Флаг **`-interval Minutes30` обязателен в каждой
  команде `cmd/backtest`**. Дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Флаги окна `-months 19 -train-months 7 -test-months 3` обязательны в каждой команде
  walk-forward**, `-months 19` — в каждом одиночном прогоне. Дефолт `-months` — 12; пропуск любого
  флага молча даёт чужие числа (§4 спеки).
- **`-refresh` НЕ запускать.** Кэш дотянут `-refresh` 2026-09-26 09:15 МСК:
  `data/candles/RAGR_Minutes30.json` (33 042 бара, 2023-09-26 09:30 … 2026-09-26 08:30),
  `RAGR_Day1.json` (1 083 свечи).
- **Хвост кэша дотягивается при каждом прогоне** (`internal/service/backtest/candles.go` —
  `CandleProvider.Load` добирает бары от последнего кэшированного до момента запуска), а окно
  `-months 19` отсчитывается от момента запуска. **В отличие от SVCB, сдвигаются оба конца окна:**
  история RAGR длиннее окна, и прогон на день позже теряет день в начале. Следствия:
  - границы фолдов walk-forward сдвигаются вместе с датой запуска;
  - **все прогоны одной задачи делаются в один календарный день**; дата прогона пишется в отчёт
    задачи и в строку `РЕЗУЛЬТАТ ПРОГОНА` `_comment`;
  - годы (§5.9) и хвосты (§5.10) режутся **фиксированными датами** — так уже делает скрипт журнала;
  - год 2025 считается с фактического начала окна прогона; оно пишется в отчёт.
- **Прогоны `cmd/backtest` — строго последовательно, никогда параллельно.** Параллельные запуски
  одновременно дописывают один файл кэша и роняют друг друга (урок SVCB).
- **Прогон, упавший на ошибке API** (`rpc error: code = Internal desc = unexpected EOF`),
  перезапускается; его числа не пишутся.

**Схемы прогонов**

- **Тематические прогоны:** `-months 19 -train-months 7 -test-months 3 -min-trades 10
  -metric profit_factor`. У темы `screen` — `-min-trades 1`. Все двенадцать тем идут только по этой
  схеме.
- **Схемы проверки точки** (не тем):

  | Схема | Флаги | Фолдов | Роль | Дефолты (2026-09-26) |
  |---|---|---|---|---|
  | **19/7/3** | `-months 19 -train-months 7 -test-months 3` | 4 | сборка и вердикт | **1.339** / пул 57; фолды 0.846/14, 1.057/17, 1.448/19, 2.412/**7** |
  | **19/10/3** | `-months 19 -train-months 10 -test-months 3` | 3 | **пункт 3 стоп-условия** | **1.544** / пул 43; фолды 1.058/17, 1.448/19, 2.411/7 |
  | 19/7/3 + `-commission 0.001` | как 19/7/3 | 4 | **пункт 4 стоп-условия** | **1.090** / пул 57 |

- **Вырожденный фолд** — меньше пяти сделок OOS или ни одной убыточной сделки. В пользу тикера он не
  засчитывается: ни как PF, ни как **голос в большинство 3/4** (§4 спеки). Вердикт по схеме
  выносится по pooled OOS, оговорка записывается.
- **Число фолдов сверяется на первой же теме** (ловушка ASTR): в шапке отчёта темы `screen` должно
  стоять «Фолдов: 4», а train первого фолда начинаться не раньше 2025-02-17. Иначе остановиться и
  доложить владельцу.

**Инструмент**

- **Шаг цены 0.02 ₽.** Реальный круг двух шагов при цене 69 ₽ — **0.058%**. Модель
  (`-commission 0.0005`, круг 0.1%) **пессимистична в 1.7 раза**. Пункт 4 стоп-условия
  (`-commission 0.001`, круг 0.2%) — в 3.4 раза выше реального круга.
- **Пять событийных шоков — прокси гейта D** (§5.7 спеки), объявлены до прогонов и не меняются:

  | Дата | Время бара шока | Тип | Размер |
  |---|---|---|---|
  | 2025-03-26 | 15:30 | внутридневной | −15.7% |
  | 2025-04-07 | — | дневной гэп | −8.1% |
  | 2025-05-14 | 10:00 | внутридневной | −13.8% |
  | 2026-06-22 | 17:00 | внутридневной | −9.7% |
  | 2026-08-06 | — | дневной гэп | −15.5% |

  Время бара — в той же шкале, что журнал сделок отчёта (спека §1 снята тем же скриптом).

**Сетки**

Сетки максимально широкие: обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
инварианты (§5.1 спеки):

- `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `StopDailyATR` нигде не ноль;
- ни один файл не порождает пар `EMAFast ≥ EMASlow` (файл без оси `EMAFast` живёт на дефолте 10);
- ось `RSIPeriod` в `cal_entry.json` содержит 2 и 14, ось `RSILower` — 5 и 50;
- ось `EMAFast` в `cal_trend_low.json` содержит 2, ось `EMASlow` — 12 и 45; в `cal_trend.json`
  `EMASlow` содержит 250;
- ось `RSIUpper` в `cal_exit.json` содержит 30 и 95;
- ось `StopDailyATR` в `cal_risk.json` содержит 0.3 и 2.0, ось `TPDailyATR` — 0.1 и 2.5;
- ось `FreshDayATR` в `cal_day.json` содержит 0.05 и 0.5; ось `SpentDayATR` — 0.4 и 2.0;
- ось `VolLookbackBars` в `cal_vol_window.json` содержит 32;
- ось `TrailDailyATR` в `cal_trail.json` содержит 0.2 и 1.5;
- **ни один файл первого круга, кроме `cal_risk.json`, не свипует `StopDailyATR`.**

**Риск-гейты**

- **Гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **0.8**. Порог объявлен до
  прогонов и не двигается. Срабатывания стопа на дефолтах (полное окно): 0.5 → 18 SL (23.4%),
  0.6 → 16, 0.7 → 15, **0.8 → 10 (13.3%)**, 1.0 → 6 (8.0%), 1.3 → 3, 1.5 → 2, 2.0 → 2.
- **Гейт B:** max DD точки на полном окне ≤ **9 642.93 ₽ и ≤ 9.03%** (не выше дефолтов; если
  baseline переснят в Task 1 — переснятые числа). Проверяются обе формы; пробита любая — гейт не
  пройден. Ничья в пользу риска: при разнице pooled OOS < 0.05 PF берётся вариант с меньшей
  просадкой; при разнице просадок < 0.5 п.п. — с меньшим числом убыточных полугодий.
- **Гейт D:** сделок через пять шоков не больше **одной**, их суммарный PnL не хуже
  **−2 822.39 ₽** (baseline: вход 2025-05-13 23:00, выход по SL). Провал — пункт 7.

**Семь пунктов стоп-условия** (§5.13 спеки). Круг останавливается, если точка даёт:

1. pooled OOS PF < 1.0 на 19/7/3;
2. меньше 20 сделок в пуле OOS 19/7/3;
3. pooled OOS PF < 1.0 на **19/10/3**;
4. pooled OOS PF < 1.0 на 19/7/3 при `-commission 0.001`;
5. хотя бы один из годов **2025** (с начала окна), **2026** убыточен (net сделок, закрытых в году,
   ≤ 0);
6. **хвост 6** (вход с 2026-03-26) с PF < 1.0, **или хвост 12** (вход с 2025-09-26) с PF < 1.0,
   **или** в хвосте 6 меньше пяти сделок. Дополнительно знак хвоста 6 (PF ≥ 1 или < 1) сверяется со
   знаком объединённого OOS PF фолдов 3 и 4 одноточечного walk-forward 19/7/3. Расхождение —
   остановиться и доложить методику;
7. провал гейта D **или** вход в выходные / в часы метки 02–06, не устранённый уводом
   `FreshDayATR` на 0.

Гейты A и B — не пункты, а ограничения сборки. Точка, которую нельзя собрать внутри них, считается
сработавшей по пункту 1.

**Числа дефолтов** (полное окно на 2026-09-26 09:18, против них меряется всё)

| Показатель | Значение |
|---|---|
| Окно | 2025-02-26 09:18 — 2026-09-26 09:18 |
| Сделок | **77** |
| PF | **1.231** |
| Net | **+8 523.21 ₽ (+8.52%)** |
| Max DD | **9 642.93 ₽ (9.03%)** |
| Win rate | 63.64% |
| Expectancy | **+110.69 ₽** |
| Выходы | RSI 51 (66.2%), **SL 18 (23.4%)**, TP 8 (10.4%) |
| Удержание медиана / p90 / максимум | **9 / 21 / 39** баров |
| Ночёвок | **32 (41.6%)**, переносов через 2+ дня **1** |
| Выходная сессия | входов **0**, выходов **1** (+66.25 ₽) |
| Часы метки 02–06 | входов **0**, выходов **3** (+123.11 ₽) |
| Гейт D | **1** сделка (2025-05-14), **−2 822.39 ₽** |

Календарные годы (по дате выхода): **2025 — 36 сделок, −2 324.97 ₽, PF 0.893**; 2026 — 41 сделка,
+10 848.15 ₽, PF 1.717. Хвосты: **хвост 6 — 26 сделок, +8 874 ₽, PF 1.806**; **хвост 12 — 57 сделок,
+8 097 ₽, PF 1.338**. Дефолты проваливают только пункт 5; маршрут «завести на дефолтах ядра» закрыт
(§3 спеки).

**Скрипты протокола лежат в `reports/_analysis/`, в git не попадают** (`reports/` в `.gitignore`).
Не удалять:

- `ragr_journal.py` — разбор журнала одиночного прогона (годы, полугодия, выходы, удержание,
  ночёвки, выходные, «ГЕЙТ C» — часы 02–06, «ГЕЙТ D» — шоки, «ПУНКТ 6» — хвосты, пять худших
  сделок). Запуск: `python3 reports/_analysis/ragr_journal.py <отчёт>_best.md`. Task 1 правит список
  шоков;
- `sfin_sweep.py` — печать свипа из калибровочного отчёта:
  `python3 reports/_analysis/sfin_sweep.py <ось[,ось2]> <отчёт>_calibration.md`;
- `ragr_recon.py`, `ragr_axes.py`, `ragr_brief.py`, `ragr_points.sh` — протокол разведки.

**Коммиты.** Каждое сообщение коммита кончается строкой
`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

**Задачи прода (14–16) не выполняются, пока вердикт не вынесен** (§5.15 спеки).

**Правило CLAUDE.md:** `docs/rsi_pullback/` не трогается; пер-тикерный разбор живёт в doc-comment
пакета `strategy/ragr/` и в `_comment` сеток.

## Review Focus

Пять режимов отказа, которые спека подразумевает, но которые легче всего пропустить. Каждый закрыт
шагом сверки или тестом в задаче-владельце.

1. **Скользящее НАЧАЛО окна.** В отличие от SVCB прогон на день позже теряет сделки в начале окна,
   и исполнитель примет это за порчу кэша или пропустит настоящую порчу. Task 1 Step 7 сверяет
   журнал переснятого baseline с эталонным отчётом разведки **сделка в сделку на пересечении окон**
   и записывает выпавшие в начале и добавленные в конце сделки отдельно.
2. **Правка скрипта журнала сломала гейт D.** Внутридневной шок с неверным условием либо пропустит
   сделку, открытую утром в день шока, либо засчитает сделку, закрытую до бара шока. Task 1 Step 5
   проверяет правило на синтетических сделках, Step 7 — что baseline по-прежнему даёт ровно одну
   сделку через шок и −2 822.39 ₽.
3. **Забытый флаг окна.** `-months` без `-train-months 7 -test-months 3` или без `-months 19` молча
   даёт чужую схему. Task 3 Step 2 сверяет «Фолдов: 4» и даты фолда 1; Task 11 Step 9 — «Фолдов:
   4» и «Фолдов: 3» для двух схем точки.
4. **Голос вырожденного фолда засчитан в большинство.** Фолд 4 держится на семи сделках у
   дефолтов. Task 11 Step 1 требует в таблице сборки столбец «вырожденные фолды» и пересчёт
   большинства без них.
5. **Знак хвоста 6 сверяется с одним фолдом вместо двух.** Хвост 6 покрывает OOS фолдов 3 и 4 вместе,
   а отчёт walk-forward не печатает сделки по фолдам. Task 11 Step 13 даёт формулу объединённого PF
   из пофолдовых чисел и проверяет её на дефолтах, где ответ известен.

---

### Task 1: Пакет `strategy/ragr` до калибровки, правка скрипта журнала и пересъёмка baseline

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт в блоке рядом со строкой 29 —
  сосед `rsipullbackrtkm`; запись в карте рядом со строкой 95)
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`
- Modify (вне git): `reports/_analysis/ragr_journal.py`
- Create: `docs/superpowers/plans/task-1-report-ragr.md`

**Interfaces:**
- Produces: `ragr.Ticker` (константа `"RAGR"`) и `ragr.DefaultParams() core.Params` — их читают
  задачи 14–16; исправленный `ragr_journal.py` — его читают задачи 8, 11, 12; переснятые числа
  baseline — гейт B в Task 11.

- [ ] **Step 1: Написать падающий тест пакета.**

```go
package ragr

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated держит состояние «калибровка не проводилась»: до
// вердикта по спеке docs/superpowers/specs/2026-09-26-ragr-rsi-pullback-prep-design.md пакет
// отдаёт ровно дефолты ядра. Тест заменяется снимком литерала (Task 14) или остаётся как протокол
// отказа (Task 13).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки RAGR обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsRAGR(t *testing.T) {
	if Ticker != "RAGR" {
		t.Fatalf("Ticker = %q, want RAGR", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/ragr/ -v`
  Expected: FAIL — пакета нет.

- [ ] **Step 3: Написать пакет.**

```go
// Package ragr supplies the ticker and rsi_pullback Params for RAGR (ПАО «Русагро»,
// обыкновенные акции после редомициляции; до неё — расписки AGRO).
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-26-ragr-rsi-pullback-prep-design.md.
package ragr

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "RAGR"

// DefaultParams returns the rsi_pullback parameters for RAGR.
func DefaultParams() core.Params { return core.DefaultParams() }
```

- [ ] **Step 4: Зарегистрировать пакет в реестре бэктеста.** В
  `internal/service/backtest/rsi_pullback_registry.go`:
  - импорт
    `rsipullbackragr "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ragr"`
    в алфавитном порядке блока импортов;
  - строка карты
    `rsipullbackragr.Ticker: rsiPullbackBindingFor(rsipullbackragr.Ticker, rsipullbackragr.DefaultParams),`
    после строки `rsipullbacksvcb`.

  Выравнивание столбцов поправит `gofmt -w`. В `rsi_pullback_registry_test.go` добавить тест рядом с
  `TestRSIPullbackSVCBIsRegisteredAndCalibrated` и тот же импорт:

```go
// TestRSIPullbackRAGRTracksBaseline сторожит ЧЕСТНОЕ состояние: RAGR заведён в реестр до
// калибровки, чтобы прогоны шли через реестр, а не через generic-ветку, и обязан возвращать ровно
// baseline ядра. Заменяется снимком литерала (Task 14 плана) при положительном вердикте.
func TestRSIPullbackRAGRTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackragr.Ticker]
	if !ok {
		t.Fatal("RAGR отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("RAGR: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("RAGR ещё не откалиброван, params обязаны совпадать с baseline:\n got: %+v\nwant: %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "RAGR" {
		t.Fatalf("Ticker() = %q, want RAGR", got)
	}
}
```

  Run: `gofmt -l internal/ && go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'RAGR|RSIPullback' -v`
  Expected: `gofmt -l` ничего не печатает, тесты PASS.

- [ ] **Step 5: Привести гейт D скрипта журнала к §5.7 спеки.** В `reports/_analysis/ragr_journal.py`:
  - заменить `EX_DATES` на пять шоков с временем бара для внутридневных:

```python
# Событийные шоки внутри окна (§5.7 спеки): дата, время бара шока (None — дневной гэп
# на открытии), размер. Прокси событийного риска, а не дивидендные отсечки.
EX_DATES = [
    (date(2025, 3, 26), time(15, 30), -15.7),
    (date(2025, 4, 7), None, -8.1),
    (date(2025, 5, 14), time(10, 0), -13.8),
    (date(2026, 6, 22), time(17, 0), -9.7),
    (date(2026, 8, 6), None, -15.5),
]


def held_through(t, day, bar):
    """Сделка держится через шок: вошла до дня шока и вышла в день шока или позже; для
    внутридневного шока — также вошла в день шока до бара шока и вышла на баре шока или позже."""
    if t["entry"].date() < day <= t["exit"].date():
        return True
    if bar is None or t["entry"].date() != day:
        return False
    shock = datetime.combine(day, bar)
    return t["entry"] < shock <= t["exit"]
```

  - импорт: `from datetime import date, datetime, time`;
  - цикл блока «ГЕЙТ D» переписать под тройку и функцию:

```python
    print("\n  ГЕЙТ D: сделки, удерживаемые через событийный шок:")
    held = []
    for ex, bar, gap in EX_DATES:
        hit = [t for t in trades if held_through(t, ex, bar)]
        held.extend(hit)
        print("    %s %s (шок %.1f%%): сделок %d, PnL %+.2f ₽" % (
            ex, bar.strftime("%H:%M") if bar else "гэп", gap, len(hit), sum(t["pnl"] for t in hit)))
        for t in hit:
            print("      #%-3d %s %.2f -> %s %.2f (%s) %+.2f ₽" % (
                t["idx"], t["entry"], t["entry_price"], t["exit"], t["exit_price"],
                t["reason"], t["pnl"]))
    print("    итого через шоки: сделок %d, PnL %+.2f ₽ (планка: ≤ 1 сделки и ≥ −2822.39 ₽)" % (
        len(held), sum(t["pnl"] for t in held)))
    if held:
        rest = [t for t in trades if t not in held]
        print("    без них: сделок %d, net %+.2f ₽, PF %.3f" % (
            len(rest), sum(t["pnl"] for t in rest), pf_of(rest)))
```

  - в докстринге файла заменить «дивидендным отсечкам» на «событийным шокам» и `svcb_journal.py` на
    `ragr_journal.py`.

  Проверить правило на синтетических сделках:

```bash
python3 - <<'EOF'
import importlib.util
from datetime import date, datetime, time
spec = importlib.util.spec_from_file_location("j", "reports/_analysis/ragr_journal.py")
j = importlib.util.module_from_spec(spec); spec.loader.exec_module(j)
def tr(a, b): return {"entry": datetime.fromisoformat(a), "exit": datetime.fromisoformat(b)}
d, bar = date(2025, 3, 26), time(15, 30)
cases = [
    (tr("2025-03-25 12:00", "2025-03-26 10:00"), True),   # вошла накануне, вышла в день шока до бара — держала через дату
    (tr("2025-03-26 11:00", "2025-03-26 16:00"), True),   # вошла утром в день шока, вышла после бара
    (tr("2025-03-26 11:00", "2025-03-26 15:30"), True),   # вышла на самом баре шока
    (tr("2025-03-26 11:00", "2025-03-26 15:00"), False),  # закрылась до бара шока
    (tr("2025-03-26 16:00", "2025-03-27 10:00"), False),  # вошла после бара шока
    (tr("2025-03-24 12:00", "2025-03-25 10:00"), False),  # закрылась до дня шока
]
for t, want in cases:
    got = j.held_through(t, d, bar)
    assert got == want, (t, got, want)
assert j.held_through(tr("2025-04-06 12:00", "2025-04-07 10:00"), date(2025, 4, 7), None)
assert not j.held_through(tr("2025-04-07 10:00", "2025-04-07 12:00"), date(2025, 4, 7), None)
assert len(j.EX_DATES) == 5
print("OK")
EOF
```

  Expected: `OK`.

- [ ] **Step 6: Прогнать скрипт на эталонном baseline разведки.**
  Run: `python3 reports/_analysis/ragr_journal.py reports/RAGR_prep/base19/RAGR_rsi_pullback_Minutes30_20260926_091823.md`
  Expected: блок «ГЕЙТ D» — ровно одна сделка через 2025-05-14 10:00 (вход 2025-05-13 23:00, SL),
  итого **1 сделка, −2 822.39 ₽**; остальные четыре шока — 0. Любое другое число — ошибка правки
  Step 5, исправить до перехода дальше.

- [ ] **Step 7: Переснять baseline на дату исполнения и сверить с эталоном сделка в сделку.**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -months 19 -metric profit_factor -out ./reports/RAGR_base
python3 reports/_analysis/ragr_journal.py $(ls reports/RAGR_base/RAGR_rsi_pullback_Minutes30_*.md | grep -v _equity | tail -1)
python3 - <<'EOF'
import glob, importlib.util
spec = importlib.util.spec_from_file_location("j", "reports/_analysis/ragr_journal.py")
j = importlib.util.module_from_spec(spec); spec.loader.exec_module(j)
ref, _ = j.parse("reports/RAGR_prep/base19/RAGR_rsi_pullback_Minutes30_20260926_091823.md")
path = sorted(p for p in glob.glob("reports/RAGR_base/RAGR_rsi_pullback_Minutes30_*.md"))[-1]
new, summary = j.parse(path)
print("новое окно:", summary.get("Период"))
key = lambda t: (t["entry"], t["exit"], round(t["pnl"], 2))
start = min(t["entry"] for t in new)
end = max(t["entry"] for t in ref)
ref_overlap = [t for t in ref if t["entry"] >= start]
new_overlap = [t for t in new if t["entry"] <= end]
dropped = [t for t in ref if t["entry"] < start]
added = [t for t in new if t["entry"] > end]
print("эталон на пересечении %d, новый %d" % (len(ref_overlap), len(new_overlap)))
mism = [(a, b) for a, b in zip(ref_overlap, new_overlap) if key(a) != key(b)]
print("расхождений на пересечении: %d (длина %s)" % (len(mism), "равна" if len(ref_overlap) == len(new_overlap) else "РАЗНАЯ"))
for a, b in mism[:5]:
    print("  эталон", key(a), "\n  новый ", key(b))
print("выпали в начале: %d, net %+.2f" % (len(dropped), sum(t["pnl"] for t in dropped)))
print("добавились в конце: %d, net %+.2f" % (len(added), sum(t["pnl"] for t in added)))
EOF
```

  **Решающее правило.**
  - Запуск 2026-09-26 до 09:30: ожидается «расхождений 0», длины равны, выпавших и добавленных 0 —
    в отчёт пишется «baseline совпал со спекой».
  - Запуск позже: расхождения допустимы **только** у сделок первых двух недель нового окна (прогрев
    дневного ATR и EMA на другой стартовой точке). Любое расхождение дальше — порча кэша или забытый
    флаг: **остановиться и доложить владельцу**.
  - Если выпавшие или добавленные сделки есть, в `task-1-report-ragr.md` записываются **переснятые**
    числа полного окна: окно, сделок, PF, max DD в обеих формах, годы, хвосты, выходы, гейт D. С
    этого момента решающими считаются они, и гейт B пересчитывается по переснятому max DD (§5.5
    спеки). Отдельно записать, изменился ли вердикт дефолтов по пункту 5 (2025 год в минусе).

- [ ] **Step 8: Коммит** `feat(rsi_pullback): пакет RAGR до калибровки` (пакет, реестр, тест реестра,
  отчёт `task-1-report-ragr.md`; скрипт в `reports/` в git не идёт).

---

### Task 2: Каталог двенадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/ragr/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_volume.json`, `cal_vol_window.json`, `cal_risk.json`,
  `cal_exit.json`, `cal_trail.json`, `cal_exit_target.json`, `cal_trend_day.json`
- Create: `internal/service/backtest/rsi_pullback_ragr_grid_test.go`

**Interfaces:**
- Consumes: хелперы пакета `backtest` — `rsiPullbackTickerGrid(t, ticker, file string)
  map[string][]float64` (`rsi_pullback_grid_test.go:40`) и `containsFloat(values []float64, want
  float64) bool` (`rsi_pullback_cnru_grid_test.go:121`).
- Produces: двенадцать путей `data/params/rsi_pullback/ragr/cal_*.json` для задач 3–10; тест
  `TestRAGRGridsStayWide`; переменная `ragrGridFiles` и константа `ragrStopCeiling` (их расширяет
  Task 12).

- [ ] **Step 1: Написать падающий сторожевой тест.**

```go
package backtest

import "testing"

// ragrGridFiles перечисляет сетки RAGR ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN, RTKM, SVCB.
var ragrGridFiles = []string{
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
	"cal_exit_target.json",
	"cal_trend_day.json",
}

// ragrCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast (trend_day), живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const ragrCoreEMAFast = 10

// ragrStopCeiling — потолок риск-гейта A (§5.4 спеки). Правила каталога на RAGR расходятся:
// выживаемость 30% даёт 1.0, частота срабатывания не реже 10% сделок — 0.8; берётся строжайшее.
const ragrStopCeiling = 0.8

// TestRAGRGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-26-ragr-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestRAGRGridsStayWide(t *testing.T) {
	for _, file := range ragrGridFiles {
		grid := rsiPullbackTickerGrid(t, "ragr", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("ragr/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("ragr/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("ragr/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{ragrCoreEMAFast}
		}
		for _, f := range fast {
			for _, s := range grid["EMASlow"] {
				if f >= s {
					t.Errorf("ragr/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
		// Стоп меряет только тема risk (§5.1 спеки): арбитры первого круга RAGR — связки выхода и
		// тренда, а не связки со стопом.
		if file != "cal_risk.json" && len(grid["StopDailyATR"]) > 0 {
			t.Errorf("ragr/%s: свипует StopDailyATR %v — стоп меряет только cal_risk.json", file, grid["StopDailyATR"])
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSIPeriod", []float64{2, 14}},
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMAFast", []float64{2}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{30, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.1, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0.05, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 2.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.2, 1.5}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "ragr", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("ragr/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/backtest/ -run RAGRGridsStayWide -v`
  Expected: FAIL — файлов сеток ещё нет.

- [ ] **Step 3: Написать двенадцать файлов сеток.** Формат (образец —
  `data/params/rsi_pullback/svcb/cal_entry.json`):

```json
{
  "_comment": "data/params/rsi_pullback/ragr/cal_entry.json — тема entry для RAGR, 100 прогонов, поверх ДЕФОЛТОВ ЯДРА. <что меряет>. ОКНО: только после рестарта торгов 2025-02-17 (-months 19, схема 19/7/3, -min-trades 10), числа несравнимы с 36-месячным каталогом. ЗАМЕР 2026-09-26 (in-sample, §6.1 спеки docs/superpowers/specs/2026-09-26-ragr-rsi-pullback-prep-design.md): <числа оси>. ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: <вырожденные узлы>. ЗАПУСК: go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ragr/cal_entry.json -out ./reports/RAGR_entry -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА: (заполняется задачей плана).",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIPeriod": [2, 3, 4, 5, 6, 7, 8, 10, 12, 14],
        "RSILower": [5, 10, 15, 20, 25, 30, 35, 40, 45, 50]
      },
      "keepTop": 5
    }
  ]
}
```

  Угловые скобки в образце заполняются реальным текстом из §6 спеки — в файле их не остаётся. Имя
  фазы = имя темы. Оси всех файлов:

| Файл | Тема | Оси | Прогонов | `-min-trades` | `keepTop` |
|---|---|---|---|---|---|
| `cal_screen.json` | `screen` | `UseDayATRGate` [0,1] × `UseVolume` [0,1] | 4 | 1 | 4 |
| `cal_entry.json` | `entry` | `RSIPeriod` [2,3,4,5,6,7,8,10,12,14] × `RSILower` [5,10,15,20,25,30,35,40,45,50] | 100 | 10 | 5 |
| `cal_trend.json` | `trend` | `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] | 48 | 10 | 5 |
| `cal_trend_low.json` | `trend_low` | `EMAFast` [2,3,5,8,10] × `EMASlow` [12,15,20,25,30,35,40,45] | 40 | 10 | 5 |
| `cal_day.json` | `day` | `FreshDayATR` [0,0.05,0.1,0.15,0.2,0.3,0.4,0.5] × `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0] | 80 | 10 | 5 |
| `cal_volume.json` | `volume` | `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0] × `VolBaseDays` [3,5,10,14,20,30] | 36 | 10 | 5 |
| `cal_vol_window.json` | `vol_window` | `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult` [1.0,1.2,2.0] | 27 | 10 | 5 |
| `cal_risk.json` | `risk` | `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.3,1.5,2.0] × `TPDailyATR` [0.1,0.15,0.2,0.25,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.5] | 120 | 10 | 5 |
| `cal_exit.json` | `exit` | `RSIUpper` [30,35,40,45,50,55,60,65,70,75,80,85,90,95] | 14 | 10 | 5 |
| `cal_trail.json` | `trail` | `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.2,0.3,0.4,0.5,0.7,1.0,1.5] | 14 | 10 | 5 |
| `cal_exit_target.json` | `exit_target` | `RSIUpper` [30,35,40,45,50,55,60,65,70] × `TPDailyATR` [0.1,0.15,0.2,0.25,0.3,0.4,0.6,0.8] | 72 | 10 | 5 |
| `cal_trend_day.json` | `trend_day` | `EMASlow` [15,20,25,30,40,50,75,100] × `FreshDayATR` [0,0.05,0.1,0.15,0.2] | 40 | 10 | 5 |

  **Что обязательно написать в `_comment` каждого файла:**
  - что тема меряет и сколько в ней прогонов;
  - замер из §6 спеки, из которого получена каждая ось;
  - оговорку об окне (только после рестарта, 19/7/3, `-min-trades 10`, несравнимость с каталогом);
  - предупреждения о краях, по файлам:
    - `cal_entry.json`: `RSILower` 5 — 0 сделок за окно, 10 — 17; `RSIPeriod` 12 — 15 сделок, 14 —
      8; дефолт 4×30 — пик обеих осей, глубокий вход на новостной бумаге убыточен (§6.1);
    - `cal_trend_low.json`: `EMAFast` 2 — узел за краем SVCB, ось монотонно растёт к нижнему краю
      (3 → 1.544/65); `EMASlow` ниже 12 и пары `EMAFast ≥ EMASlow` намеренно не входят (§5.1);
    - `cal_risk.json`: узел `TPDailyATR` 2.5 — контрольная строка, которую требует
      `TestRSIPullbackGridControlPoints` (цель строго выше самого широкого стопа 2.0), кандидатом не
      считается (2.5 → 0.971/76); стоп выше 0.8 режет гейт A — ось остаётся широкой, чтобы решение
      было видно в отчёте (§5.1 п.5);
    - `cal_exit.json`: `RSIUpper` 30 не мерялся — новый край, максимум SVCB-сетки на нижнем узле
      (35 → 1.729/108);
    - `cal_trail.json`: `TrailDailyATR` 0.2 — новый нижний край; 1.0 и 1.5 побайтово равны
      дефолтам (трейл не срабатывает); при `UseRSIExit` 0 трейл вреден (0.87–1.02);
    - `cal_day.json`: `FreshDayATR` 0.05 открывает вход на первом баре дня (урок X5), проверка
      входов в выходные и 02–06 обязательна; `SpentDayATR` 2.0 — 7 сделок за окно, горб 1.25–1.5
      проваливает 2025 год;
    - `cal_exit_target.json`, `cal_trend_day.json`: тема — **арбитр**, а не источник полей (§5.2
      спеки); голоса в сборку не идут, но записываются всегда (по ним выбираются зоны второго
      круга);
  - полную команду запуска с путём самого файла (`TestRSIPullbackCalFilesValid` требует, чтобы
    `_comment` содержал `ragr/<имя файла>`); `-min-trades` — по таблице;
  - место под результат прогона.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ -run 'RAGRGridsStayWide|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints|RSIPullbackPointFilesArePoints' -v`
  Expected: PASS.

- [ ] **Step 5: Мутационная проверка сторожа.** Временно добавить `"StopDailyATR": [0.5, 0.8]` в
  `grid` файла `cal_exit_target.json` и прогнать `go test ./internal/service/backtest/ -run RAGRGridsStayWide`
  — Expected: FAIL с текстом `стоп меряет только cal_risk.json`. Затем временно убрать узел 2 из
  `EMAFast` в `cal_trend_low.json` — Expected: FAIL `потеряла обязательный узел 2`. Вернуть обе
  правки, прогнать снова — PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): каталог сеток RAGR`.

---

### Task 3: Тема `screen` — цена двух гейтов и сверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/ragr/cal_screen.json` (`_comment`); Create
`docs/superpowers/plans/task-3-report-ragr.md`

**Interfaces:**
- Consumes: `cal_screen.json` из Task 2.
- Produces: голоса фолдов по `UseDayATRGate` и `UseVolume` — их читает Task 11; фактические
  train/test-окна фолдов на дату запуска — их читает Task 11 Step 13.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_screen.json -out ./reports/RAGR_screen \
  -months 19 -train-months 7 -test-months 3 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Сверить число фолдов и окно.** В шапке `*_walkforward.md` должно стоять
  «Train-окно: 7 мес; OOS-фолд: 3 мес» и «Фолдов: 4»; train фолда 1 начинается не раньше
  2025-02-17. Иначе **остановиться и доложить владельцу**. Записать фактические train/test-окна
  четырёх фолдов.

- [ ] **Step 3: Записать** pooled OOS PF, размер пула и голоса всех четырёх фолдов по обеим осям;
  пометить вырожденные фолды.

- [ ] **Step 4: Сверить с ожиданием** (§6.3 спеки): **ожидается 1×0** — дневной гейт включён,
  объёмный выключен. Точечный замер in-sample: 1×0 → 1.231/77, 0×1 → 1.314/136 (DD 12 937 ₽ — выше
  baseline, гейт B его отсекает), 0×0 → 1.129/240, 1×1 → 1.203/61. Если фолды голосуют за 0×1,
  записать прямым текстом, что узел в точку не пройдёт гейт B, — решение принимает Task 11.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-3-report-ragr.md` (образец —
  `docs/superpowers/plans/task-3-report-svcb.md`) и дописать результат в `_comment` сетки строкой
  `РЕЗУЛЬТАТ ПРОГОНА <дата>: …` с pooled OOS, пулом, фолдами и голосами.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): RAGR, тема screen`.

---

### Task 4: Тема `entry` — первая половина планки

**Files:** Modify `data/params/rsi_pullback/ragr/cal_entry.json` (`_comment`); Create
`docs/superpowers/plans/task-4-report-ragr.md`

**Interfaces:**
- Consumes: `cal_entry.json` из Task 2.
- Produces: голоса по `RSIPeriod` и `RSILower`, вердикт первой половины планки — их читают Task 11 и
  Task 12.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_entry.json -out ./reports/RAGR_entry \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS PF, пул и голоса четырёх фолдов (`RSILower` — ведущая ось);
  пометить вырожденные фолды.

- [ ] **Step 3: Проверить первую половину планки** (§5.12): pooled OOS ≥ **1.5** при ≥ **20**
  сделках **и** `RSILower` одинаков в ≥ 3 фолдах из 4. Вырожденный фолд в пользу тикера не
  засчитывается.

- [ ] **Step 4: Сверить с ожиданием:** **планку тема НЕ возьмёт** (§6.1): дефолт 4×30 — пик обеих
  осей, ни одного узла выше дефолта. Голос за `RSILower` 45 (in-sample 1.213/126, 2025 +3 269 ₽)
  записать отдельно — это единственный узел оси входа, лечащий 2025 год.

- [ ] **Step 5: Отметить вырожденные узлы, если фолды их выбрали:** `RSILower` 5 (0 сделок),
  10 (17 сделок за окно), `RSIPeriod` 14 (8 сделок), 12 (15 сделок) — на семимесячном train такой
  узел почти наверняка ниже `-min-trades 10` и тонет в рейтинге; если всё же выбран, записать.

- [ ] **Step 6: Написать отчёт** `task-4-report-ragr.md` и дописать результат в `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): RAGR, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая половина планки

**Files:** Modify `data/params/rsi_pullback/ragr/cal_trend.json`, `cal_trend_low.json`
(`_comment`); Create `docs/superpowers/plans/task-5-report-ragr.md`

**Interfaces:**
- Consumes: оба файла из Task 2.
- Produces: голоса по `EMAFast` и `EMASlow` из обеих тем и их pooled OOS — их читают Task 10 и
  Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_trend.json -out ./reports/RAGR_trend \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_trend_low.json -out ./reports/RAGR_trend_low \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов (`EMASlow` —
  ведущая ось); пометить вырожденные фолды и отдельно — OOS PF фолда 4.

- [ ] **Step 3: Проверить вторую половину планки** по **канонической** `trend`, не по `trend_low`.

- [ ] **Step 4: Сверить с ожиданиями** (§6.2 спеки).
  - `trend`: каноническая ось `EMASlow` 50–250 лежит в 1.155–1.375 in-sample; ось `EMAFast` —
    максимум на 3 (1.544/65). Планку тема вряд ли возьмёт.
  - `trend_low`: in-sample пик `EMASlow` 25 → 1.526/69 (оба года в плюс, DD 5.10%); 15 → 1.434/67,
    но 2025 −791 ₽. Короткий тренд на RAGR свежий год **не ломает** (в отличие от SVCB). Голос за
    `EMAFast` 2 (новый край) записать отдельно: если он соберёт большинство, Task 11 Step 7 снимает
    зонд за краем.

- [ ] **Step 5: Записать, какая тема дала больший pooled OOS**, и предварительный вывод по полю
  тренда для Task 11 (правило §5.2: `trend_low` — только при большинстве ≥ 3/4 **и** pooled OOS
  выше, чем у `trend`).

- [ ] **Step 6: Написать отчёт** `task-5-report-ragr.md` и дописать результаты в оба `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): RAGR, темы trend и trend_low`.

---

### Task 6: Тема `day`

**Files:** Modify `data/params/rsi_pullback/ragr/cal_day.json` (`_comment`); Create
`docs/superpowers/plans/task-6-report-ragr.md`

**Interfaces:**
- Consumes: `cal_day.json` из Task 2.
- Produces: голоса по `FreshDayATR` и `SpentDayATR` и pooled OOS — их читают Task 10 и Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_day.json -out ./reports/RAGR_day \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям; пометить
  вырожденные фолды.

- [ ] **Step 3: Записать опорный замер `FreshDayATR`** (§6.4): 0 → 1.231 (дефолт), **0.05 →
  1.514/84**, 0.1 → 1.513/85, 0.15 → 1.319, 0.2 → 1.267, 0.3 → 1.269/125, 0.4 → 1.293/154
  (DD 16 352 ₽ — выше гейта B), 0.5 → 1.222/183. Ненулевое значение заводит вход на первый бар дня —
  кандидат на провал запрета входов в выходные и 02–06 в Task 11 Step 6.

- [ ] **Step 4: Записать опорный замер `SpentDayATR`:** рваная ось, горб 1.25 → 1.682/34 (2025
  −4 543 ₽), 1.5 → 1.589/24, 2.0 → 0.822/**7**. Голос за 1.25–1.5 записать как рычаг, проваливающий
  пункт 5 in-sample; голос за 2.0 — как подгонку под фолд на вырожденной выборке.

- [ ] **Step 5: Написать отчёт** `task-6-report-ragr.md` и дописать результат в `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): RAGR, тема day`.

---

### Task 7: Темы `volume` и `vol_window`

**Files:** Modify `data/params/rsi_pullback/ragr/cal_volume.json`, `cal_vol_window.json`
(`_comment`); Create `docs/superpowers/plans/task-7-report-ragr.md`

**Interfaces:**
- Consumes: оба файла из Task 2.
- Produces: голоса по `VolMult`, `VolBaseDays`, `VolLookbackBars` и pooled OOS обеих тем — их читает
  Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_volume.json -out ./reports/RAGR_volume \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_vol_window.json -out ./reports/RAGR_vol_window \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с ожиданием.** Обе темы идут с `UseVolume` 1 и **поле `UseVolume` не
  решают** — его решает `screen` (Task 3). Опорный замер `vol_window` in-sample: лучший узел
  1 × 2.0 → 1.519/43 (2025 +4 773 ₽, хвост 6 1.442/**13**), остальные 1.18–1.40. Если объёмный гейт
  в точку не войдёт (`screen` не голосует за `UseVolume` 1), голоса этих тем пишутся справочно.

- [ ] **Step 4: Написать отчёт** `task-7-report-ragr.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): RAGR, темы volume и vol_window`.

---

### Task 8: Тема `risk`, потолок гейта A и зонды капкана

**Files:** Modify `data/params/rsi_pullback/ragr/cal_risk.json` (`_comment`); Create
`data/params/rsi_pullback/ragr/probe_stop_10.json`, `probe_stop_15.json`; Create
`docs/superpowers/plans/task-8-report-ragr.md`

**Interfaces:**
- Consumes: `cal_risk.json` из Task 2; `ragr_journal.py` из Task 1.
- Produces: голоса по `StopDailyATR` (уже пропущенные через потолок 0.8) и по `TPDailyATR`, pooled
  OOS — их читают Task 10 и Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_risk.json -out ./reports/RAGR_risk \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям.

- [ ] **Step 3: Применить потолок гейта A.** Голос выше **0.8** заменяется на 0.8, цена решения в PF
  записывается прямым текстом. Обоснование повторить в отчёте (§5.4 спеки): правила каталога
  расходятся — выживаемость 30% даёт 1.0 (31.9% дней), частота срабатывания ≥ 10% сделок даёт 0.8
  (13.3%; 1.0 → 8.0%); берётся строжайшее. С 0.8 пул застывает на 75 сделках — подпись капкана.

- [ ] **Step 4: Снять два зонда капкана на полном окне.** Каждый файл — одна комбинация поверх
  дефолтов ядра:
  - `probe_stop_10.json`: `{"_comment":"…","phases":[{"name":"probe","grid":{"StopDailyATR":[1.0]},"keepTop":1}]}`;
  - `probe_stop_15.json` — то же со `StopDailyATR` [1.5].

  В `_comment` — назначение зонда («цена потолка 0.8, за потолком — капкан»), опорный замер и
  команда запуска. Команда (последовательно):

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/probe_stop_10.json -out ./reports/RAGR_probe_stop_10 \
  -months 19 -min-trades 1 -metric profit_factor
python3 reports/_analysis/ragr_journal.py reports/RAGR_probe_stop_10/*_best.md
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/probe_stop_15.json -out ./reports/RAGR_probe_stop_15 \
  -months 19 -min-trades 1 -metric profit_factor
python3 reports/_analysis/ragr_journal.py reports/RAGR_probe_stop_15/*_best.md
```

  По каждому зонду записать PF, сделок, число SL-выходов, обе формы DD и годы. Опорные числа:
  1.0 → 1.171/75, 6 SL (8.0%), DD 8.32%, 2025 +50 ₽; 1.5 → 1.366/75, 2 SL, DD 10.27%. **Главный
  вывод для отчёта:** 2025 год в плюс по оси стопа дают только номинальные стопы за потолком;
  внутри потолка (0.8 → 2025 −1 474 ₽) стоп год не лечит (§5.9).

- [ ] **Step 5: Записать опорный замер `TPDailyATR`** (§6.5): 0.1 → 1.574/103, **0.15 → 1.545/95**,
  **0.2 → 1.497/90**, 0.25 → 1.269, 0.6 → 1.231 (дефолт), 2.5 → 0.971/76 (контрольная строка).
  Близкая цель режет DD до 5.10%, но хвост 6 слабеет (0.15 → 1.077/28). Голос за 0.1–0.2 — кандидат
  на конфликт с `exit` (Task 10).

- [ ] **Step 6: Прогнать тесты** (файлы зондов — точки, они проходят `TestRSIPullbackCalFilesValid`
  и не участвуют в асимметрии): `go test ./internal/service/backtest/ -run 'RAGR|RSIPullback'`.
  Expected: PASS.

- [ ] **Step 7: Написать отчёт** `task-8-report-ragr.md` и дописать результат в `_comment`.

- [ ] **Step 8: Коммит** `docs(rsi_pullback): RAGR, тема risk и зонды капкана`.

---

### Task 9: Темы `exit` и `trail`

**Files:** Modify `data/params/rsi_pullback/ragr/cal_exit.json`, `cal_trail.json` (`_comment`);
Create `docs/superpowers/plans/task-9-report-ragr.md`

**Interfaces:**
- Consumes: оба файла из Task 2.
- Produces: голоса по `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` и pooled OOS — их читают
  Task 10 и Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_exit.json -out ./reports/RAGR_exit \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_trail.json -out ./reports/RAGR_trail \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с опорными замерами** (§6.6 спеки).
  - `RSIUpper`: ось монотонно растёт к нижнему краю, 35 → 1.729/108 (2025 +6 942 / 2026 +10 196 ₽,
    хвост 6 1.619/36, DD 6.19%), 40 → 1.579/100; 30 не мерялся. Голос за 30 (новый край)
    записать отдельно — Task 11 Step 7 снимает для него зонд за краем (`RSIUpper` 25).
  - Трейл: при `UseRSIExit` 1 — 0.5 → 1.441/78 (2025 +1 329 ₽, DD 6.68%), 0.3–0.4 — провал,
    1.0 и 1.5 равны дефолтам. Если тема голосует за `UseTrail` 1, гейт A применяется к
    `min(StopDailyATR, TrailDailyATR)` (урок AFKS): трейл 1.0 и 1.5 при стопе 0.5 гейт проходят, но
    как рычаг номинальны — записать.

- [ ] **Step 4: Написать отчёт** `task-9-report-ragr.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): RAGR, темы exit и trail`.

---

### Task 10: Темы-арбитры `exit_target` и `trend_day`

**Files:** Modify `data/params/rsi_pullback/ragr/cal_exit_target.json`, `cal_trend_day.json`
(`_comment`); Create `docs/superpowers/plans/task-10-report-ragr.md`

**Interfaces:**
- Consumes: оба файла из Task 2; pooled OOS и голоса `exit` (Task 9), `risk` по оси цели (Task 8),
  `trend`/`trend_low` (Task 5), `day` по оси `FreshDayATR` (Task 6).
- Produces: вердикты «рычаги складываются / конкурируют / арбитр справочный» по двум парам и голоса
  арбитров — их читают Task 11 Step 1 и Task 12 Step 1.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_exit_target.json -out ./reports/RAGR_exit_target \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal_trend_day.json -out ./reports/RAGR_trend_day \
  -months 19 -train-months 7 -test-months 3 -min-trades 10 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов по обеим осям;
  пометить вырожденные фолды.

- [ ] **Step 3: Вынести вердикт каждого арбитра** (§5.2 спеки). Пара `exit` × `risk` (ось
  `TPDailyATR`) — судит `exit_target`; пара «тема тренда, давшая поле» × `day` (ось `FreshDayATR`) —
  судит `trend_day`:
  - обе темы пары голосуют большинством ≥ 3/4 за уход от дефолта, и pooled OOS арбитра **не ниже**
    обеих — рычаги **складываются**, оба поля идут в точку по своим темам;
  - pooled OOS арбитра **ниже любой** из двух — рычаги **конкурируют**: в точку идёт один рычаг,
    тот, чья тема дала больший pooled OOS; второе поле остаётся на дефолте ядра; цена решения пишется
    прямым текстом;
  - большинство за уход от дефолта есть не больше чем у одной темы пары — арбитр **справочный**.

  Голоса `RSIUpper`, `TPDailyATR`, `EMASlow`, `FreshDayATR` **из арбитров в сборку не идут** —
  записываются для второго круга.

- [ ] **Step 4: Записать опорные числа разведки** (§6.5–§6.7): одиночные рычаги `RSIUpper` 35
  (1.729/108), `TPDailyATR` 0.2 (1.497/90), `EMASlow` 25 (1.526/69), `FreshDayATR` 0.05 (1.514/84)
  каждый проходит пункты 5, 6 и гейт B; сумма `RSIUpper` и близкой цели меняет одни и те же сделки и
  может оказаться хуже каждого по отдельности.

- [ ] **Step 5: Написать отчёт** `task-10-report-ragr.md` и дописать результаты в оба `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): RAGR, темы-арбитры exit_target и trend_day`.

---

### Task 11: Сборка точки, гейты, walk-forward и семь пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/ragr/plateau_point.json` и соседи плато
`plateau_<поле>_<значение>.json`; Create `docs/superpowers/plans/task-11-report-ragr.md`

**Interfaces:**
- Consumes: голоса и вердикты задач 3–10; `ragr_journal.py` (Task 1), `sfin_sweep.py`; переснятые
  числа baseline (Task 1 Step 7).
- Produces: точку первого круга (восемнадцать полей) и вердикт — их читают Task 12 и Task 14.

- [ ] **Step 1: Собрать точку правилом большинства** (§5.2 спеки):
  - поле берётся из темы, которая его меряет, только при ≥ 3 голосах из 4 **невырожденных**
    фолдов; иначе — дефолт ядра; ничья 2/2 большинством не считается;
  - источники: `RSIPeriod`, `RSILower` — `entry`; `StopDailyATR`, `TPDailyATR` — `risk`;
    `FreshDayATR`, `SpentDayATR` — `day`; `RSIUpper` — `exit`; `UseRSIExit`, `UseTrail`,
    `TrailDailyATR` — `trail`; `UseDayATRGate`, `UseVolume` — `screen`;
  - поле тренда: из `trend_low` только при большинстве ≥ 3/4 **и** pooled OOS выше `trend`; иначе из
    `trend`;
  - поля объёма: из `vol_window` при превосходстве над `volume` и только если `screen` не
    проголосовала большинством за `UseVolume` 0;
  - вердикты арбитров применяются по Task 10 Step 3.

  Записать таблицу для всех восемнадцати полей: «поле → тема-источник → голоса по фолдам →
  вырожденные фолды → большинство без вырожденных → принятое значение → дефолт ядра».

- [ ] **Step 2: Применить гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **0.8**.

- [ ] **Step 3: Написать `plateau_point.json`** — одна фаза, все восемнадцать полей по одному
  значению (образец — `data/params/rsi_pullback/svcb/plateau_point.json`). Снять одиночный прогон:

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/plateau_point.json -out ./reports/RAGR_point \
  -months 19 -min-trades 1 -metric profit_factor
python3 reports/_analysis/ragr_journal.py reports/RAGR_point/*_best.md
```

- [ ] **Step 4: Применить гейт B:** max DD ≤ **9 642.93 ₽ и ≤ 9.03%** (или переснятые числа Task 1
  Step 7), обе формы. При пробое в точку идёт ближайший вариант плато с меньшей просадкой, цена
  пишется прямым текстом.

- [ ] **Step 5: Гейт D:** из блока «ГЕЙТ D» — сделок через шоки ≤ **1** и итог PnL ≥
  **−2 822.39 ₽**. Иначе сработал пункт 7.

- [ ] **Step 6: Запрет входов в выходные и в часы 02–06:** входов в выходные 0 и входов 02–06
  (блок «ГЕЙТ C») 0. При провале увести `FreshDayATR` на 0 и пересчитать с Step 3. Если провал
  остался — сработал пункт 7.

- [ ] **Step 7: Снять соседей плато.** Для каждого поля, ушедшего от дефолта, — прогон самого
  значения и двух соседей по оси, команда как в Step 3, файлы `plateau_<поле>_<значение>.json`
  (значение без точки: `plateau_tp_015.json`, `plateau_rsiupper_35.json`).
  - Край сетки дополнительно проверяется **зондом за краем** (`RSIUpper` 25 за краем 30, `EMAFast`
    1 не допускается ядром — тогда записать «край = минимум ядра»; `TrailDailyATR` 0.15 за краем
    0.2; `TPDailyATR` 0.05 за краем 0.1).
  - Соседа по оси стопа при включённом трейле проверяют в сторону уменьшения.
  - **Для каждого соседа снять годы и хвосты** скриптом `ragr_journal.py` (§5.3 спеки).
  - Разница pooled OOS < 0.05 решается в пользу меньшей просадки, затем меньшего числа убыточных
    полугодий (§5.5).
  - Плато уже 0.05 PF записывается как отсутствие сигнала.

- [ ] **Step 8: Третий контур.** Сравнить с дефолтами: SL 23.4%, удержание 9/21/39, ночёвок 41.6%,
  переносов 1, DD 9.03%, expectancy +110.69 ₽, выходов в выходные 1, выходов 02–06 3. **Падение
  доли SL вместе с ростом удержания и ночёвок — капкан**, тогда точка берёт более узкий стоп в
  пределах плато. Падение доли SL вместе с **сокращением** удержания (близкая цель) — не капкан
  (§6.5), записать как второй случай из `docs/rsi_pullback/strategy.md`.

- [ ] **Step 9: Два walk-forward точки (последовательно).**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/plateau_point.json -out ./reports/RAGR_point_wf73 \
  -months 19 -train-months 7 -test-months 3 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/plateau_point.json -out ./reports/RAGR_point_wf103 \
  -months 19 -train-months 10 -test-months 3 -min-trades 1 -metric profit_factor
```

  Ожидаемое число фолдов: **4 и 3**; расхождение — остановиться и доложить (ловушка ASTR). Записать
  pooled OOS, пул и пофолдовые числа, пометить вырожденные фолды.

- [ ] **Step 10: Утяжелённые издержки.**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/plateau_point.json -out ./reports/RAGR_point_c001 \
  -months 19 -train-months 7 -test-months 3 -min-trades 1 -metric profit_factor -commission 0.001
```

- [ ] **Step 11: Календарные годы.** Из блока «календарные годы» Step 3: 2025 (с начала окна прогона)
  и 2026 — net > 0.

- [ ] **Step 12: Хвосты.** Из блока «ПУНКТ 6» Step 3 записать хвост 6 и хвост 12 (сделок, net, PF) и
  долю крупнейшей сделки в net хвоста 6 — сделку с максимальным |PnL| среди входов с 2026-03-26.

- [ ] **Step 13: Сверка знака хвоста 6 с фолдами 3–4.** Отчёт walk-forward не печатает сделки по
  фолдам, поэтому объединённый OOS PF фолдов 3 и 4 восстанавливается из таблицы «Результаты по
  фолдам» (каждый фолд стартует с капитала 100 000 ₽: `OOS NetPnL%` × 1000 = net в ₽; тогда
  `GL = net / (PF − 1)`, `GP = PF × GL`):

```bash
python3 - reports/RAGR_point_wf73/*_walkforward.md <<'EOF'
import re, sys
rows = {}
for line in open(sys.argv[1], encoding="utf-8"):
    c = [x.strip() for x in line.strip().strip("|").split("|")]
    if len(c) == 8 and c[0].isdigit():
        rows[int(c[0])] = (c[2], float(c[4]), int(c[5]), float(c[6].rstrip("%")))
gp = gl = 0.0
for k in (3, 4):
    win, pf, n, netpct = rows[k]
    net = netpct * 1000
    if abs(pf - 1) < 1e-3 or pf == 0:
        sys.exit("фолд %d: PF %.3f — формула неприменима, доложить владельцу" % (k, pf))
    loss = net / (pf - 1)
    gl += loss; gp += pf * loss
    print("фолд %d: test %s, PF %.3f, сделок %d, net %+.0f ₽" % (k, win, pf, n, net))
print("объединённый OOS PF фолдов 3-4: %.3f" % (gp / gl))
EOF
```

  **Проверка формулы на дефолтах до применения к точке:** на
  `reports/RAGR_prep/wf/1973/RAGR_rsi_pullback_Minutes30_20260926_092058_walkforward.md` скрипт
  обязан дать **1.813 ± 0.005** (фолд 3: 1.448, +3 030 ₽; фолд 4: 2.412, +5 830 ₽), что совпадает с
  хвостом 6 дефолтов 1.806 не только по знаку, но и по величине. Иное число — ошибка разбора,
  исправить до применения к точке. Если фолд без убыточных сделок (PF «+Inf» или ошибка разбора) —
  вырожденный, сверка по одному оставшемуся фолду с оговоркой.
  Знак объединённого PF (≥ 1 или < 1) обязан совпасть со знаком хвоста 6. Расхождение —
  **остановиться и доложить владельцу** оба числа и фактические тест-окна фолдов 3–4 (они
  начинаются от даты запуска, а не ровно 2026-03-26).

- [ ] **Step 14: Применить семь пунктов стоп-условия** (Global Constraints). По каждому пункту
  записать число и вердикт.

- [ ] **Step 15: Проверить планку** (Task 4 Step 3, Task 5 Step 3) и сверить с ожиданием «не
  возьмёт ни одна из двух тем».

- [ ] **Step 16: Прогнать тесты** (файлы-точки обязаны нести ровно одно значение на ключ):
  `go test ./internal/service/backtest/ -run 'RAGR|RSIPullback'`. Expected: PASS.

- [ ] **Step 17: Написать отчёт** `task-11-report-ragr.md`. Содержание:
  - таблица сборки с вырожденными фолдами;
  - гейты A, B, D и запрет входов с ценой решений;
  - соседи плато с годами и хвостами;
  - третий контур;
  - два walk-forward и издержки;
  - годы, хвосты, доля крупнейшей сделки и сверка знака;
  - семь пунктов и планка.

- [ ] **Step 18: Вердикт.**
  - Ни один пункт не сработал — переход к Task 14, задачи 12 и 13 пропускаются.
  - Иначе — переход к Task 12.

  **Ожидание, записанное до прогонов (§5.14): первый круг может пройти**; главный риск — рассыпание
  голосов, тогда точка остаётся на дефолтах и проваливает пункт 5.

- [ ] **Step 19: Коммит** `feat(rsi_pullback): RAGR, точка первого круга и вердикт`.

---

### Task 12: Второй круг (только при провале первого)

**Files:** Create `data/params/rsi_pullback/ragr/cal2_<тема>.json` (не больше шести),
`plateau_point2.json`, соседи `plateau_r2_<поле>_<значение>.json`; Modify
`internal/service/backtest/rsi_pullback_ragr_grid_test.go`; Create
`docs/superpowers/plans/task-12-report-ragr.md`

**Задача выполняется ТОЛЬКО при срабатывании хотя бы одного пункта стоп-условия в Task 11.**

**Interfaces:**
- Consumes: голоса задач 3–10 (включая арбитров), вердикт Task 11, `ragrGridFiles` и
  `ragrStopCeiling` из Task 2.
- Produces: точку второго круга — её читают Task 13 и Task 14.

- [ ] **Step 1: Выбрать зоны по фактическим голосам фолдов первого круга**, включая голоса
  арбитров `exit_target` и `trend_day`, а не по in-sample рельефу. Шаг — половина шага первого
  круга. Не больше шести тем. **Каждое поле свипует ровно одна тема** второго круга, и она — его
  источник (§5.14): тема-связка (например, `cal2_exit_target`) забирает оба своих поля, одиночные
  темы тех же полей не заводятся. В `_comment` каждого файла записать, из каких голосов какого фолда
  получена зона, и команду запуска с путём `ragr/cal2_<тема>.json`.
  - Кандидат номер один — `cal2_exit_target`, если голоса `exit` / `exit_target` / `risk` по цели
    рассыпались по нижней зоне: `RSIUpper` [27.5,30,32.5,35,37.5,40,42.5,45] × `TPDailyATR`
    [0.1,0.125,0.15,0.175,0.2,0.225,0.25,0.6].
    Узел 0.6 (дефолт) нужен как контроль «цель выключена близким RSI-выходом».
  - Кандидат номер два — `cal2_trend_day`: `EMASlow` [20,22,24,25,26,28,30] × `FreshDayATR`
    [0,0.025,0.05,0.075,0.1] — только если `EMASlow` и `FreshDayATR` не свипует другая тема второго
    круга. `EMASlow` целочисленный, поэтому половинный шаг округлён до целых узлов.
  - Стоп, если его свипует тема второго круга, — не выше **0.8**; при свипе стопа файл обязан
    содержать цель строго выше самого широкого стопа (`TestRSIPullbackGridControlPoints`).
  - Целочисленные поля `core.Params` (`RSIPeriod`, `EMAFast`, `EMASlow`, `VolBaseDays`,
    `VolLookbackBars` и флаги) берут только целые узлы — `applyField` отвергает дробные.

- [ ] **Step 2: Расширить сторожевой тест.** В `rsi_pullback_ragr_grid_test.go` добавить переменную
  `ragrRound2GridFiles` с поимённым списком файлов `cal2_*.json` и функцию:

```go
// ragrAllGridFiles склеивает сетки обоих кругов: жёсткие инварианты §5.1 держатся на обоих.
func ragrAllGridFiles() []string {
	out := make([]string, 0, len(ragrGridFiles)+len(ragrRound2GridFiles))
	out = append(out, ragrGridFiles...)
	return append(out, ragrRound2GridFiles...)
}
```

  В первом цикле `TestRAGRGridsStayWide` заменить `for _, file := range ragrGridFiles` на
  `for _, file := range ragrAllGridFiles()`, а проверку «стоп меряет только `cal_risk.json`» оставить
  только для файлов первого круга:

```go
		if !strings.HasPrefix(file, "cal2_") && file != "cal_risk.json" && len(grid["StopDailyATR"]) > 0 {
```

  (добавить импорт `strings`). В конец теста — потолок второго круга:

```go
	// Потолок гейта A во втором круге не двигается (§5.14 спеки).
	for _, file := range ragrRound2GridFiles {
		for _, v := range rsiPullbackTickerGrid(t, "ragr", file)["StopDailyATR"] {
			if v > ragrStopCeiling {
				t.Errorf("ragr/%s: StopDailyATR=%v выше потолка гейта A %v", file, v, ragrStopCeiling)
			}
		}
	}
```

  И тест «одно поле — одна тема»:

```go
// TestRAGRRound2FieldsHaveOneSource держит правило §5.14 спеки: во втором круге каждое поле
// свипует ровно одна тема, иначе две темы проголосуют за одно поле по-разному.
func TestRAGRRound2FieldsHaveOneSource(t *testing.T) {
	owner := map[string]string{}
	for _, file := range ragrRound2GridFiles {
		for field, values := range rsiPullbackTickerGrid(t, "ragr", file) {
			if len(values) < 2 {
				continue // зафиксированное поле не голосует
			}
			if prev, ok := owner[field]; ok {
				t.Errorf("ragr: поле %s свипуют две темы второго круга: %s и %s", field, prev, file)
			}
			owner[field] = file
		}
	}
}
```

  Run: `go test ./internal/service/backtest/ -run 'RAGR|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints' -v`
  Expected: PASS.

- [ ] **Step 3: Прогнать каждую узкую тему (последовательно).**

```bash
go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ragr/cal2_<тема>.json -out ./reports/RAGR_r2_<тема> \
  -months 19 -train-months 7 -test-months 3 -min-trades 1 -metric profit_factor
```

  `-min-trades 1` — число сделок не критерий (прецедент SFIN).

- [ ] **Step 4: Собрать точку второго круга** правилом ≥ 3/4 невырожденных фолдов **в зоне ±1 шаг
  второго круга** (§5.14): голоса соседних узлов складываются в пользу центрального, если
  центральный сам набрал хотя бы один голос; при двух претендентах берётся тот, у кого больше
  собственных голосов, при равенстве — узел с большим pooled OOS одноточечного прогона. Правило
  записать в отчёт с разбивкой голосов. Поля, которые второй круг не свипует, берутся из точки
  первого круга. Прогнать точку через Task 11 Steps 2–16 без изменений, с файлами
  `plateau_point2.json` / `plateau_r2_*` и каталогами `RAGR_point2*`.

- [ ] **Step 5: Стоп-условие второго круга.**
  - Пункт 2 снят; фактический размер пула записывается.
  - **Пул OOS меньше десяти сделок закрывает работу как непредставительный.**
  - Пункты 5 и 6 — главные критерии, не смягчаются.
  - Пункты 1, 3, 4, 7 — в силе полностью. Потолок стопа 0.8 не двигается.

- [ ] **Step 6: Отчёт** `task-12-report-ragr.md` той же структуры, что Task 11, плюс таблица «зона
  узкой сетки → голоса первого круга». Прогнать тесты:
  `go test ./internal/service/backtest/ -run 'RAGR|RSIPullback'`. Expected: PASS.

- [ ] **Step 7: Вердикт.** Точка прошла — переход к Task 14. Иначе — Task 13.

- [ ] **Step 8: Коммит** `feat(rsi_pullback): RAGR, второй круг и вердикт`.

---

### Task 13: Протокол отказа (только при провале обоих кругов)

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr.go`
(doc-comment), `ragr_test.go` (комментарий теста), `internal/service/backtest/rsi_pullback_registry_test.go`
(комментарий `TestRSIPullbackRAGRTracksBaseline`); Create
`docs/superpowers/plans/task-13-report-ragr.md`

**Задача выполняется ТОЛЬКО при провале обоих кругов. Задачи 14–17 не выполняются.**

- [ ] **Step 1: Написать разбор отказа в doc-comment пакета.** Образец —
  `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm.go`. Содержание:
  - всё из §5.16 спеки;
  - какой пункт сработал в каждом круге и с какими числами;
  - что на бумаге этому причиной.

  В doc-comment `TestParamsTrackTheBaselineUntilCalibrated` и `TestRSIPullbackRAGRTracksBaseline`
  записать окончательное состояние «калибровка закрыта отказом <дата>, RAGR в боевую вселенную не
  заводится» (образец — `TestRSIPullbackRTKMPTracksBaseline`). Литерал не ставится.

- [ ] **Step 2: Полный гейт.** Run: `./bin/mage ci`. Expected: PASS.

- [ ] **Step 3: Доложить владельцу** числа обоих кругов и причину отказа.

- [ ] **Step 4: Коммит** `docs(rsi_pullback): RAGR, протокол отказа`.

---

### Task 14: Литерал в пакете и снимок

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr.go`,
`ragr_test.go`, `internal/service/backtest/rsi_pullback_registry_test.go`

**Выполняется ТОЛЬКО при положительном вердикте Task 11 Step 18 или Task 12 Step 7.**

**Interfaces:**
- Consumes: принятую точку.
- Produces: `ragr.DefaultParams()` возвращает литерал — его читают Task 15 и Task 16.

- [ ] **Step 1: Заменить тест пакета снимком.** Удалить `TestParamsTrackTheBaselineUntilCalibrated`,
  написать `TestParamsMatchTheCalibratedSnapshot`. Все восемнадцать полей выписаны литералами.
  Образец — `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb_test.go`. Скелет:

```go
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod: 4, RSILower: 30, RSIUpper: 35,
		EMAFast: 10, EMASlow: 100,
		DailyATRPeriod: 14, UseDayATRGate: 1, FreshDayATR: 0, SpentDayATR: 0.8,
		StopDailyATR: 0.5, TPDailyATR: 0.6,
		UseVolume: 0, VolBaseDays: 14, VolLookbackBars: 3, VolMult: 1.2,
		UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал RAGR разошёлся со снимком калибровки:\n got: %+v\nwant: %+v", got, want)
	}
}
```

  Значения в скелете — **пример формы** (дефолты + `RSIUpper` 35 из §6.6 спеки). Перед записью они
  заменяются числами принятой точки поле за полем по таблице отчёта Task 11 или 12. Целочисленные
  поля `core.Params` (`core/core.go`): `RSIPeriod`, `EMAFast`, `EMASlow`, `DailyATRPeriod`,
  `UseDayATRGate`, `UseVolume`, `VolBaseDays`, `VolLookbackBars`, `UseRSIExit`, `UseTrail`; остальные
  — `float64`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/ragr/ -v`
  Expected: FAIL.

- [ ] **Step 3: Поставить литерал.** `DefaultParams()` возвращает `core.Params` со всеми восемнадцатью
  полями **явно**, даже совпавшими с ядром. Строку «СОСТОЯНИЕ: калибровка не проводилась» в
  doc-comment заменить на «СОСТОЯНИЕ: откалиброван <дата>, разбор ниже» (разбор пишет Task 17).

- [ ] **Step 4: Обновить тест реестра.** `TestRSIPullbackRAGRTracksBaseline` заменить на
  `TestRSIPullbackRAGRIsRegisteredAndCalibrated` по образцу
  `TestRSIPullbackSVCBIsRegisteredAndCalibrated` (`rsi_pullback_registry_test.go:1053`): реестр отдаёт
  `rsipullbackragr.DefaultParams()`, и это не `core.DefaultParams()`.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'RAGR|RSIPullback' -v`
  Expected: PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): литерал RAGR`.

---

### Task 15: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`,
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `ragr.Ticker`, `ragr.DefaultParams()` из Task 14.
- Produces: `ParamsFor("RAGR")` — его проверяет Task 18 через `cmd/pullparity`.

- [ ] **Step 1: Написать падающий тест** рядом с `TestRegistryHasSVCB` (`registry_test.go:227`):

```go
// TestRegistryHasRAGR держит связку «пакет — реестр живого раннера» для RAGR: раннер обязан
// торговать ровно литерал пакета.
func TestRegistryHasRAGR(t *testing.T) {
	p, ok := ParamsFor(ragr.Ticker)
	if !ok {
		t.Fatal("RAGR нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := ragr.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}
```

  Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ragr"` добавить в тестовый
  файл.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run RAGR -v`
  Expected: FAIL.

- [ ] **Step 3: Добавить в реестр.** Импорт
  `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ragr"` (в алфавитном порядке
  блока, перед `.../strategy/reni`) и строка `ragr.Ticker: ragr.DefaultParams(),` после
  `svcb.Ticker: svcb.DefaultParams(),` (`registry.go:714`). `gofmt -w` выровняет столбцы.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
  Expected: PASS.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): RAGR в реестре живого раннера`.

---

### Task 16: Боевая вселенная

**Files:** Modify `internal/config/rsi_pullback.go:605`, `internal/config/rsi_pullback_test.go:56-58`,
`env/prod.env:20`, `env/prod.env.example:30`, `env/local.env.example:28`

Сторож `TestRSIPullbackTickersMatchEnvFiles` (`internal/config/rsi_pullback_test.go:79`) уже есть в
репозитории (заведён на SVCB) и читает все три env-файла.

- [ ] **Step 1: Написать падающий тест.** В `TestNewRSIPullbackConfig_Defaults`
  (`rsi_pullback_test.go:54`) дописать `"RAGR"` последним в `want`, заменить `len(want) != 33` на
  `len(want) != 34` и сообщение `want 33: SVCB заведён тридцать третьим 2026-09-25` на
  `want 34: RAGR заведён тридцать четвёртым <дата исполнения>`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/config/ -run RSIPullback -v`
  Expected: FAIL в `TestNewRSIPullbackConfig_Defaults` (Go-дефолт ещё без RAGR).
  `TestRSIPullbackTickersMatchEnvFiles` пока PASS — он сторожит расхождение файлов, а не состав.

- [ ] **Step 3: Добавить `RAGR` последним элементом во все четыре места сразу:**
  - `internal/config/rsi_pullback.go:605` — `"RAGR"` после `"SVCB"`;
  - `env/prod.env:20` — `,RAGR` в конец строки `RSI_PULLBACK_TICKERS=`;
  - `env/prod.env.example:30` — то же;
  - `env/local.env.example:28` — то же.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/config/ -v`
  Expected: PASS.

- [ ] **Step 5: Мутационная проверка сторожа.** Временно убрать `,RAGR` из `env/prod.env`, прогнать
  `go test ./internal/config/ -run RSIPullbackTickersMatchEnvFiles` — Expected: FAIL с именем
  `env/prod.env`. Восстановить строку вручную (`git checkout` не годится — правка ещё не
  закоммичена) и снова прогнать — PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): завести RAGR в боевую вселенную`.

---

### Task 17: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr.go`
(doc-comment)

- [ ] **Step 1: Написать разбор в doc-comment пакета.** Образец формы —
  `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb.go`. Обязательное содержимое
  §5.16 спеки:
  - пауза редомициляции 2024-12-02 → 2025-02-17, окно только после рестарта и сделка +33 414 ₽,
    из-за которой 36/12/6 не годится;
  - схема 19/7/3, `-min-trades 10` и несравнимость с 36-месячным каталогом;
  - шаг цены 0.02 ₽ и реальный круг 0.058% против модельных 0.1%;
  - таблица сборки точки;
  - годы 2025–2026 и оба хвоста с долей крупнейшей сделки хвоста 6;
  - сверка знака хвоста 6 с фолдами 3–4;
  - таблица срабатываний стопа (§5.4), голос темы `risk` и цена урезания до 0.8;
  - гейт B в обеих формах;
  - третий контур против дефолтов;
  - гейт D с оговоркой «даты — прокси событийного риска, движок новости не моделирует»;
  - два walk-forward, издержки 0.2%, фактический пул OOS и вырожденные фолды;
  - семь пунктов и планка;
  - прогрев дневного ATR через паузу — сделки конца февраля — начала марта 2025 года перечислить
    (§8 №7 спеки);
  - числа выходной сессии и часов 02–06 точки (§8 №8);
  - принятые риски §8: короткое окно; **корпоративно-событийный риск — ручной контроль владельца,
    при новостях по корпоративному управлению «Русагро» тикер выводится из вселенной вручную**;
    условие пересмотра после прода — хвост 6 месяцев с PF < 1.0 на ≥ 10 сделках, вывести из
    вселенной вне планового цикла.

  **Правило CLAUDE.md: `docs/rsi_pullback/` не трогается.**

- [ ] **Step 2: Линтер.** Run: `./bin/mage lint`. Expected: PASS.

- [ ] **Step 3: Коммит** `docs(rsi_pullback): разбор калибровки RAGR и принятый риск`.

---

### Task 18: Финальная проверка

**Files:** нет (только прогоны), кроме правок по найденному.

- [ ] **Step 1: Полный гейт.** Run: `./bin/mage ci`. Expected: PASS (lint + `go test -race ./...` +
  дрейф моков).

- [ ] **Step 2: Сверка живой сборки с бэктестом.**

```bash
go run ./cmd/pullparity -tickers RAGR -months 24
```

  Флаг — **`-tickers`** (множественное число). Без него команда молча возьмёт `UGLD,T,GAZP`.
  **`-months 24`, не больше** (`maxDailyHorizonMonths` в `cmd/pullparity/main.go`; прецедент YDEX —
  тоже редомициляция с паузой, сверка прошла на 24). Expected: ноль расхождений. Если расхождения
  есть и все лежат на барах паузы 2024-12-02 → 2025-02-17 или в первые две недели после рестарта —
  **остановиться и доложить владельцу** с датами расхождений, не подбирать окно.

- [ ] **Step 3: Сверить вселенную.** Run: `go test ./internal/config/ -run 'RSIPullbackTickersMatchEnvFiles|NewRSIPullbackConfig_Defaults' -v`.
  Expected: PASS, в списке 34 тикера, RAGR последний.

- [ ] **Step 4: Доложить владельцу:**
  - точку;
  - два walk-forward;
  - издержки 0.2%;
  - годы;
  - хвосты и сверку знака;
  - гейты A, B, D;
  - семь пунктов;
  - планку;
  - ручной контроль корпоративно-событийного риска.

- [ ] **Step 5: Коммит** правок финальной проверки, если они были. Мерж делает владелец.

---

## Self-Review

**Покрытие спеки:**

| Раздел спеки | Задача |
|---|---|
| §1 инструмент, кэш, скользящий конец окна, шоки | Global Constraints; Task 1 Steps 5–7 |
| §2 отношение к каталогу, несравнимость | Task 2 Step 3 (`_comment`); Task 17 |
| §3 baseline, маршрут дефолтов закрыт | Global Constraints; Task 1 Step 7 |
| §4 окно 19/7/3, 19/10/3, ASTR, вырожденные фолды, обязательные флаги | Global Constraints; Task 3 Step 2; Task 11 Steps 1, 9 |
| §5.1 двенадцать тем, инварианты, отличия от SVCB | Task 2; задачи 3–10 |
| §5.2 сборка, `trend_low`, `vol_window`, арбитры | Task 10 Step 3; Task 11 Step 1 |
| §5.3 соседи плато, зонд за краем, годы и хвосты | Task 11 Step 7 |
| §5.4 гейт A (0.8) | Task 2 (константа); Task 8 Steps 3–4; Task 11 Step 2 |
| §5.5 гейт B и ничьи | Task 11 Steps 4, 7 |
| §5.6 третий контур, запрет входов | Task 11 Steps 6, 8 |
| §5.7 гейт D (пять шоков, планка baseline) | Task 1 Steps 5–6; Task 11 Step 5 |
| §5.8 скрипт журнала и его правка | Global Constraints; Task 1 Step 5 |
| §5.9 календарные годы | Task 11 Step 11 |
| §5.10 хвосты и сверка знака с фолдами 3–4 | Task 11 Steps 12–13 |
| §5.11 схемы проверки | Task 11 Steps 9–10 |
| §5.12 планка | Task 4 Step 3; Task 5 Step 3; Task 11 Step 15 |
| §5.13 стоп-условие | Task 11 Step 14 |
| §5.14 второй круг, одно поле — одна тема, зона ±1 шаг, пул < 10 | Task 12 |
| §5.15 правило прода | Task 11 Step 18; задачи 14–16, 18 |
| §5.16 что записывается | Task 17; Task 13 (при отказе) |
| §6 свойства бумаги | опорные замеры в задачах 3–10 и `_comment` сеток |
| §7 артефакты | задачи 1, 2, 14–16 |
| §8 риски | Task 17 Step 1 |

**Плейсхолдеры.** Угловые скобки остались только там, где исполнитель подставляет значения из
отчётов прогонов (`<тема>`, `<поле>`, `<значение>`, `<дата исполнения>`) и в образце `_comment`, где
явно сказано заполнить их текстом §6. Скелет литерала в Task 14 помечен как пример формы.

**Согласованность имён:**
- `ragrGridFiles`, `ragrStopCeiling`, `ragrCoreEMAFast` — Task 2; расширяются в Task 12 через
  `ragrAllGridFiles`, `ragrRound2GridFiles`;
- `ragr.Ticker`, `ragr.DefaultParams()` — Task 1, используются в задачах 14–16;
- `TestRSIPullbackRAGRTracksBaseline` — Task 1, заменяется в Task 14 на
  `TestRSIPullbackRAGRIsRegisteredAndCalibrated` или получает окончательный комментарий в Task 13;
- `held_through`, `EX_DATES` (тройки) — Task 1 Step 5, читаются скриптом во всех прогонах точки;
- каталоги отчётов `RAGR_point`, `RAGR_point_wf73`, `RAGR_point_wf103`, `RAGR_point_c001` — Task 11,
  второй круг — `RAGR_point2*`;
- номера отчётов совпадают с номерами задач (`task-N-report-ragr.md`).

**Развилки:**
- Task 11 Step 18: точка прошла — задачи 12 и 13 пропускаются;
- иначе Task 12; при её провале Task 13, и задачи 14–18 (кроме `mage ci` в Task 13) не
  выполняются;
- Task 10 Step 3: при конфликте рычагов в точку идёт один из двух.
