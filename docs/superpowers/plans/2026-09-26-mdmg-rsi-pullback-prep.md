# MDMG под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести MDMG («Мать и дитя», обыкновенные акции ПАО) до вердикта по стратегии
`rsi_pullback`. Нужно собрать двенадцать максимально широких сеток, прогнать тематический
walk-forward на 24-месячном окне после редомициляции и получить точку внутри риск-гейтов A и B,
которая проходит семь пунктов стоп-условия. Если первый круг провален — второй круг по правилам
владельца. Затем выбрать победителя между точкой и дефолтами ядра (§5.15 спеки) и завести его в
боевую вселенную тридцать пятым тикером.

**Architecture:** Процедура каноническая (спека §5). Пять отличий от каталога.

1. **Окно — 24 месяца:** `-months 24`, схема сборки **24/12/3**, контроль **24/15/3**. Пауза
   редомициляции 2024-05-22 → 2024-06-17 и сессия расписок (19 баров) в окно не попадают.
2. **Главный критерий владельца в стоп-условии:** годы 2025 и 2026 в плюс. Огрызок 2024Q4 пишется, но
   пунктом не считается.
3. **Жёсткий пункт устойчивости:** хвост 6 месяцев против хвоста 12 месяцев по дате входа.
4. **Потолок стопа 0.8** — строжайшее из двух правил гейта A.
5. **Дефолты ядра уже проходят стоп-условие** — они полноправный кандидат в прод; точка должна их
   обойти. Арбитры — `trend_spent` и `trend_day`.

Разведка показала: дефолты сильные (PF 1.546/134, pooled OOS 1.659/52). Рычаги — `SpentDayATR` 0.9
(2.243/107), короткая быстрая EMA (`EMAFast` 1–3), короткая медленная (`EMASlow` 20–30), строгий
объём (`VolMult` 3.0). Выход и цель — не рычаги. Капкан широкого стопа выражен чисто. Главный риск —
точка не обойдёт дефолты вне выборки.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков), `python3` для
разбора журналов сделок.

**Spec:** `docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md` — план ссылается на её
параграфы (§N), исполнитель читает оба документа.
**Сырые отчёты разведки:** `reports/MDMG_prep/` (вне git): `wf/wf_24_0_0_0.0005/` — baseline на полном
окне (`MDMG_rsi_pullback_Minutes30_20260926_230928_best.md`), `wf/wf_24_12_3_0.0005/`,
`wf/wf_24_15_3_0.0005/`, `wf/wf_24_12_3_0.001/` — walk-forward дефолтов, `ax/` — оси, `pt/` — точки.

## Global Constraints

**Ветка, кэш и окно**

- **Ветка:** `feat/mdmg-pullback-prep` от `main` `3086f36` (в боевой вселенной 34 тикера, последний —
  `RAGR`). Спека закоммичена (`6e136f2`, поправка дат отсечек `c41129d`).
- **Таймфрейм `Minutes30` во всех прогонах.** Флаг **`-interval Minutes30` обязателен в каждой
  команде `cmd/backtest`**. Дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Флаги окна `-months 24 -train-months 12 -test-months 3` обязательны в каждой команде
  walk-forward**, `-months 24` — в каждом одиночном прогоне. Дефолт `-months` — 12; пропуск любого
  флага молча даёт чужие числа (§4 спеки).
- **`-refresh` НЕ запускать.** Кэш дотянут `-refresh` 2026-09-26 23:07 МСК:
  `data/candles/MDMG_Minutes30.json` (32 383 бара, 2023-09-27 09:30 … 2026-09-26 22:30),
  `MDMG_Day1.json` (1 120 свечей).
- **Хвост кэша дотягивается при каждом прогоне** (`internal/service/backtest/candles.go` —
  `CandleProvider.Load` добирает бары от последнего кэшированного до момента запуска), а окно
  `-months 24` отсчитывается от момента запуска. Сдвигаются оба конца окна. Следствия:
  - границы фолдов walk-forward сдвигаются вместе с датой запуска;
  - **все прогоны одной задачи делаются в один календарный день**; дата прогона пишется в отчёт
    задачи и в строку `РЕЗУЛЬТАТ ПРОГОНА` `_comment`;
  - годы (§5.9) и хвосты (§5.10) режутся **фиксированными датами** — так уже делает скрипт журнала.
- **Прогоны `cmd/backtest` — строго последовательно, никогда параллельно.** Параллельные запуски
  одновременно дописывают один файл кэша и роняют друг друга (урок SVCB).
- **Прогон, упавший на ошибке API** (`rpc error: code = Internal desc = unexpected EOF` и подобные),
  перезапускается; его числа не пишутся.

**Схемы прогонов**

- **Тематические прогоны:** `-months 24 -train-months 12 -test-months 3 -min-trades 20
  -metric profit_factor`. У темы `screen` — `-min-trades 1`. Все двенадцать тем идут только по этой
  схеме.
- **Схемы проверки точки** (не тем):

  | Схема | Флаги | Фолдов | Роль | Дефолты (2026-09-26) |
  |---|---|---|---|---|
  | **24/12/3** | `-months 24 -train-months 12 -test-months 3` | 4 | сборка и вердикт | **1.659** / пул 52; фолды 1.086/17, 2.650/13, 0.927/9, 2.228/13 |
  | **24/15/3** | `-months 24 -train-months 15 -test-months 3` | 3 | **пункт 3 стоп-условия** | **1.903** / пул 35; фолды 2.661/13, 0.922/9, 2.216/13 |
  | 24/12/3 + `-commission 0.001` | как 24/12/3 | 4 | **пункт 4 стоп-условия** | **1.324** / пул 52 |

- **Вырожденный фолд** — меньше пяти сделок OOS или ни одной убыточной сделки. В пользу тикера он не
  засчитывается: ни как PF, ни как **голос в большинство 3/4** (§4 спеки). Вердикт по схеме
  выносится по pooled OOS, оговорка записывается.
- **Число фолдов сверяется на первой же теме** (ловушка ASTR): в шапке отчёта темы `screen` должно
  стоять «Фолдов: 4», а train первого фолда начинаться не раньше 2024-09-01. Иначе остановиться и
  доложить владельцу.

**Инструмент**

- **Шаг цены 0.1 ₽.** Реальный круг двух шагов при цене 1 295 ₽ — **0.015%**. Модель
  (`-commission 0.0005`, круг 0.1%) **пессимистична в 6.5 раза**. Пункт 4 стоп-условия
  (`-commission 0.001`, круг 0.2%) — в 13 раз выше реального круга.
- **Четыре дивидендные отсечки — даты гейта D** (§5.7 спеки, из T-Invest `GetDividends`), объявлены
  до прогонов и не меняются: **2024-11-29** (20 ₽), **2025-05-19** (22 ₽), **2025-10-20** (42 ₽),
  **2026-07-14** (47 ₽). Правило удержания: `дата входа < дата отсечки ≤ дата выхода`.

**Сетки**

Сетки максимально широкие: обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
инварианты (§5.1 спеки):

- `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `EMAFast ≥ 1`, `StopDailyATR` нигде не ноль;
- ни один файл не порождает пар `EMAFast ≥ EMASlow` (файл без оси `EMAFast` живёт на дефолте 10);
- ось `RSIPeriod` в `cal_entry.json` содержит 2 и 14, ось `RSILower` — 5 и 50;
- ось `EMAFast` в `cal_trend_low.json` содержит 1, ось `EMASlow` — 12 и 45; в `cal_trend.json`
  `EMASlow` содержит 250;
- ось `RSIUpper` в `cal_exit.json` содержит 30 и 95;
- ось `StopDailyATR` в `cal_risk.json` содержит 0.3 и 2.0, ось `TPDailyATR` — 0.1 и 2.5;
- ось `FreshDayATR` в `cal_day.json` содержит 0.05 и 0.5; ось `SpentDayATR` — 0.4 и 2.5;
- ось `VolMult` в `cal_volume.json` содержит 4.0;
- ось `VolLookbackBars` в `cal_vol_window.json` содержит 32;
- ось `TrailDailyATR` в `cal_trail.json` содержит 0.2 и 1.5;
- **ни один файл первого круга, кроме `cal_risk.json`, не свипует `StopDailyATR`.**

**Риск-гейты**

- **Гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **0.8**. Порог объявлен до
  прогонов и не двигается. Срабатывания стопа на дефолтах (полное окно, 134 сделки начиная с 0.5):
  0.3 → 57 SL (39.9%), 0.4 → 43, 0.5 → 33 (24.6%), 0.6 → 27, 0.7 → 25, **0.8 → 22 (16.4%)**,
  1.0 → 10 (7.5%), 1.2 → 6, 1.5 → 6, 2.0 → 3.
- **Гейт B:** max DD точки на полном окне ≤ **9 153.17 ₽ и ≤ 7.54%** (не выше дефолтов; если
  baseline переснят в Task 1 — переснятые числа). Проверяются обе формы; пробита любая — гейт не
  пройден. Ничья в пользу риска: при разнице pooled OOS < 0.05 PF берётся вариант с меньшей
  просадкой; при разнице просадок < 0.5 п.п. — с меньшим числом убыточных полугодий.
- **Гейт C:** входов в выходные — **0**, входов в часы метки 02–06 — не больше **2** (как у
  дефолтов). Провал уводит `FreshDayATR` на 0; не устранённый провал — пункт 7.
- **Гейт D:** сделок через четыре отсечки — **0**. Провал — пункт 7.

**Семь пунктов стоп-условия** (§5.13 спеки). Круг останавливается, если точка даёт:

1. pooled OOS PF < 1.0 на 24/12/3;
2. меньше 20 сделок в пуле OOS 24/12/3;
3. pooled OOS PF < 1.0 на **24/15/3**;
4. pooled OOS PF < 1.0 на 24/12/3 при `-commission 0.001`;
5. хотя бы один из годов **2025**, **2026** убыточен (net сделок, закрытых в году, ≤ 0); огрызок 2024
   не пункт;
6. **хвост 6** (вход с 2026-03-26) с PF < 1.0, **или хвост 12** (вход с 2025-09-26) с PF < 1.0,
   **или** в хвосте 6 меньше пяти сделок. Дополнительно знак хвоста 6 (PF ≥ 1 или < 1) сверяется со
   знаком объединённого OOS PF фолдов 3 и 4 одноточечного walk-forward 24/12/3. Расхождение —
   остановиться и доложить методику;
7. провал гейта D **или** гейта C, не устранённый уводом `FreshDayATR` на 0.

Гейты A и B — не пункты, а ограничения сборки. Точка, которую нельзя собрать внутри них, считается
сработавшей по пункту 1.

**Числа дефолтов** (полное окно на 2026-09-26 23:09, против них меряется всё)

| Показатель | Значение |
|---|---|
| Окно | 2024-09-26 23:09 — 2026-09-26 23:09 |
| Сделок | **134** |
| PF | **1.546** |
| Net | **+28 648.01 ₽ (+28.65%)** |
| Max DD | **9 153.17 ₽ (7.54%)** |
| Win rate | 69.40% |
| Expectancy | **+213.79 ₽** |
| Выходы | RSI 82 (61.2%), **SL 33 (24.6%)**, TP 19 (14.2%) |
| Удержание медиана / p90 / максимум | **9 / 19 / 33** бара |
| Ночёвок | **46 (34.3%)**, переносов через 2+ дня **2** |
| Выходная сессия | входов **0**, выходов **5** (+716.46 ₽) |
| Часы метки 02–06 | входов **2**, выходов **6** (+1 407.87 ₽) |
| Гейт D | **0** сделок |

Календарные годы (по дате выхода): огрызок 2024 — 19 сделок, −2 348.11 ₽, PF 0.735 (не пункт);
**2025 — 83 сделки, +19 457.38 ₽, PF 1.627**; **2026 — 32 сделки, +11 538.81 ₽, PF 1.914**. Хвосты:
**хвост 6 — 22 сделки, PF 1.727**; **хвост 12 — 52 сделки, PF 1.663**. Объединённый OOS PF фолдов 3–4
дефолтов — **1.729**. Дефолты проходят все семь пунктов (§3 спеки) — кандидат в прод.

**Скрипты протокола лежат в `reports/_analysis/`, в git не попадают** (`reports/` в `.gitignore`).
Не удалять:

- `mdmg_journal.py` — разбор журнала одиночного прогона (годы, полугодия, выходы, удержание,
  ночёвки, выходные, «ГЕЙТ C» — часы 02–06, «ГЕЙТ D» — четыре отсечки, «ПУНКТ 6» — хвосты, пять
  худших сделок). Запуск: `python3 reports/_analysis/mdmg_journal.py <отчёт>_best.md`;
- `sfin_sweep.py` — печать свипа из калибровочного отчёта:
  `python3 reports/_analysis/sfin_sweep.py <ось[,ось2]> <отчёт>_calibration.md`;
- `mdmg_recon.py`, `mdmg_axes_run.sh`, `mdmg_axes.py`, `mdmg_brief.py`, `mdmg_points.sh`,
  `divprobe/` — протокол разведки.

**Коммиты.** Каждое сообщение коммита кончается строкой
`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

**Задачи прода (14–16) не выполняются, пока победитель не выбран** (Task 13, §5.15 спеки).

**Правило CLAUDE.md:** `docs/rsi_pullback/` не трогается; пер-тикерный разбор живёт в doc-comment
пакета `strategy/mdmg/` и в `_comment` сеток.

**Пакет `rsi_pullback/strategy/mdmg` не путать с `reversion/strategy/mdmg`** — разные деревья, второй
не трогается.

## Review Focus

Пять режимов отказа, которые спека подразумевает, но которые легче всего пропустить. Каждый закрыт
шагом сверки или тестом в задаче-владельце.

1. **Точка «прошла», но хуже дефолтов.** Исполнитель, привыкший к каталогу, заведёт точку, как только
   стоп-условие не сработало. Task 13 сравнивает кандидатов по правилу §5.15 механически и требует
   таблицу «точка против дефолтов» на одинаковых схемах одного дня.
2. **Скользящее начало окна.** Прогон на день позже теряет сделки в начале окна. Task 1 Step 6 сверяет
   журнал переснятого baseline с эталоном разведки сделка в сделку на пересечении окон.
3. **Забытый флаг окна.** `-months` без `-train-months 12 -test-months 3` молча даёт чужую схему.
   Task 3 Step 2 сверяет «Фолдов: 4» и даты фолда 1; Task 11 Step 9 — «Фолдов: 4» и «Фолдов: 3» для
   двух схем точки.
4. **Узел `FreshDayATR` пускает ночные входы.** 0.05 даёт 19 входов в 02–06 против двух у дефолтов.
   Task 11 Step 6 проверяет гейт C и уводит поле на 0 до walk-forward.
5. **Вырожденный фолд голосует.** Фолд 3 дефолтов — девять сделок, а узлы `SpentDayATR` 1.25–1.5 и
   `VolMult` 3.0+ режут пул втрое. Task 11 Step 1 требует столбец «вырожденные фолды» и пересчёт
   большинства без них.

---

### Task 1: Пакет `strategy/mdmg` до калибровки, проверка скрипта журнала и пересъёмка baseline

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт в блоке рядом со строкой 28 —
  сосед `rsipullbackragr`; запись в карте рядом со строкой 95)
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`
- Create: `docs/superpowers/plans/task-1-report-mdmg.md`

**Interfaces:**
- Produces: `mdmg.Ticker` (константа `"MDMG"`) и `mdmg.DefaultParams() core.Params` — их читают
  задачи 13–16; переснятые числа baseline (полное окно и три схемы walk-forward) — гейт B в Task 11
  и сравнение кандидатов в Task 13.

- [ ] **Step 1: Написать падающий тест пакета.**

```go
package mdmg

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated держит состояние «калибровка не проводилась»: до
// вердикта по спеке docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md пакет
// отдаёт ровно дефолты ядра. Тест заменяется снимком литерала (Task 14, маршрут точки) или
// решением «дефолты по выбору» (Task 14, маршрут дефолтов), либо остаётся протоколом отказа.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки MDMG обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsMDMG(t *testing.T) {
	if Ticker != "MDMG" {
		t.Fatalf("Ticker = %q, want MDMG", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/mdmg/ -v`
  Expected: FAIL — пакета нет.

- [ ] **Step 3: Написать пакет.**

```go
// Package mdmg supplies the ticker and rsi_pullback Params for MDMG (МКПАО «МД Медикал Груп»,
// «Мать и дитя», обыкновенные акции после редомициляции; до неё — расписки).
//
// Не путать с пакетом reversion/strategy/mdmg: другая стратегия, другое дерево.
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md.
package mdmg

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "MDMG"

// DefaultParams returns the rsi_pullback parameters for MDMG.
func DefaultParams() core.Params { return core.DefaultParams() }
```

- [ ] **Step 4: Зарегистрировать пакет в реестре бэктеста.** В
  `internal/service/backtest/rsi_pullback_registry.go`:
  - импорт
    `rsipullbackmdmg "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg"`
    в алфавитном порядке блока импортов;
  - строка карты
    `rsipullbackmdmg.Ticker: rsiPullbackBindingFor(rsipullbackmdmg.Ticker, rsipullbackmdmg.DefaultParams),`
    после строки `rsipullbackragr`.

  Выравнивание столбцов поправит `gofmt -w`. В `rsi_pullback_registry_test.go` добавить тест рядом с
  `TestRSIPullbackRAGRIsRegisteredAndCalibrated` (строка ~1081) и тот же импорт:

```go
// TestRSIPullbackMDMGTracksBaseline сторожит ЧЕСТНОЕ состояние: MDMG заведён в реестр до
// калибровки, чтобы прогоны шли через реестр, а не через generic-ветку, и обязан возвращать ровно
// baseline ядра. Заменяется в Task 14 плана снимком литерала или решением «дефолты по выбору».
func TestRSIPullbackMDMGTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackmdmg.Ticker]
	if !ok {
		t.Fatal("MDMG отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("MDMG: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("MDMG ещё не откалиброван, params обязаны совпадать с baseline:\n got: %+v\nwant: %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "MDMG" {
		t.Fatalf("Ticker() = %q, want MDMG", got)
	}
}
```

  Run: `gofmt -l internal/ && go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'MDMG|RSIPullback' -v`
  Expected: `gofmt -l` ничего не печатает, тесты PASS.

- [ ] **Step 5: Проверить гейт D скрипта журнала на синтетике и на эталоне.** Скрипт уже содержит
  четыре даты отсечек (`EX_DATES`) и функцию `held_through`. Проверить:

```bash
python3 - <<'EOF'
import importlib.util
from datetime import date, datetime
spec = importlib.util.spec_from_file_location("j", "reports/_analysis/mdmg_journal.py")
j = importlib.util.module_from_spec(spec); spec.loader.exec_module(j)
def tr(a, b): return {"entry": datetime.fromisoformat(a), "exit": datetime.fromisoformat(b)}
d = date(2025, 10, 20)
assert j.held_through(tr("2025-10-17 18:00", "2025-10-20 10:00"), d, None)       # через выходные в день отсечки
assert j.held_through(tr("2025-10-17 18:00", "2025-10-21 10:00"), d, None)       # через отсечку дальше
assert not j.held_through(tr("2025-10-20 10:00", "2025-10-20 12:00"), d, None)   # вошла в день отсечки
assert not j.held_through(tr("2025-10-16 10:00", "2025-10-17 22:00"), d, None)   # закрылась до отсечки
assert [x[0] for x in j.EX_DATES] == [date(2024, 11, 29), date(2025, 5, 19), date(2025, 10, 20), date(2026, 7, 14)]
print("OK")
EOF
python3 reports/_analysis/mdmg_journal.py reports/MDMG_prep/wf/wf_24_0_0_0.0005/MDMG_rsi_pullback_Minutes30_20260926_230928_best.md
```

  Expected: `OK`; на эталоне — 134 сделки, PF 1.546, «ГЕЙТ C: входов 02-06 2», все четыре отсечки
  по 0 сделок, хвост 6 — 22/1.727, хвост 12 — 52/1.663. Иное — остановиться и доложить.

- [ ] **Step 6: Переснять baseline на дату исполнения и сверить с эталоном сделка в сделку.**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -months 24 -metric profit_factor -out ./reports/MDMG_base
python3 reports/_analysis/mdmg_journal.py $(ls reports/MDMG_base/MDMG_rsi_pullback_Minutes30_*.md | grep -v _equity | tail -1)
python3 - <<'EOF'
import glob, importlib.util
spec = importlib.util.spec_from_file_location("j", "reports/_analysis/mdmg_journal.py")
j = importlib.util.module_from_spec(spec); spec.loader.exec_module(j)
ref, _ = j.parse("reports/MDMG_prep/wf/wf_24_0_0_0.0005/MDMG_rsi_pullback_Minutes30_20260926_230928_best.md")
path = sorted(p for p in glob.glob("reports/MDMG_base/MDMG_rsi_pullback_Minutes30_*.md") if "_equity" not in p)[-1]
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
  - Расхождения PnL на пересечении допустимы **только** у сделок первых двух недель нового окна
    (прогрев EMA на другой стартовой точке; капитал сделки зависит от предыдущих, поэтому сдвиг PnL
    после выпавшей прибыльной или убыточной сделки ожидаем — сравнивать ещё и по `(entry, exit)` без
    PnL, там расхождений быть не должно). Расхождение дат входа/выхода дальше двух недель — порча кэша
    или забытый флаг: **остановиться и доложить владельцу**.
  - В `task-1-report-mdmg.md` записать **переснятые** числа полного окна: окно, сделок, PF, max DD в
    обеих формах, годы, хвосты, выходы, гейты C и D. С этого момента решающими считаются они; гейт B
    пересчитывается по переснятому max DD (§5.5 спеки).

- [ ] **Step 7: Переснять три схемы walk-forward дефолтов (последовательно).** Файл одной точки —
  дефолты ядра:

```bash
mkdir -p reports/MDMG_base
echo '{"_comment":"baseline","phases":[{"name":"base","grid":{"RSILower":[30]},"keepTop":1}]}' > reports/MDMG_base/base.json
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate reports/MDMG_base/base.json -out ./reports/MDMG_base_wf123 \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate reports/MDMG_base/base.json -out ./reports/MDMG_base_wf153 \
  -months 24 -train-months 15 -test-months 3 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate reports/MDMG_base/base.json -out ./reports/MDMG_base_c001 \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor -commission 0.001
```

  Записать pooled OOS, пул и фолды каждой схемы. Проверить семь пунктов стоп-условия **для
  дефолтов** (годы и хвосты — из Step 6). Записать вердикт «дефолты допущены / не допущены
  кандидатом» (§5.15 спеки). Опорные числа 2026-09-26: 1.659/52, 1.903/35, 1.324/52.

- [ ] **Step 8: Коммит** `feat(rsi_pullback): пакет MDMG до калибровки` (пакет, реестр, тест реестра,
  отчёт `task-1-report-mdmg.md`; `reports/` в git не идёт).

---

### Task 2: Каталог двенадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/mdmg/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_volume.json`, `cal_vol_window.json`, `cal_risk.json`,
  `cal_exit.json`, `cal_trail.json`, `cal_trend_spent.json`, `cal_trend_day.json`
- Create: `internal/service/backtest/rsi_pullback_mdmg_grid_test.go`

**Interfaces:**
- Consumes: хелперы пакета `backtest` — `rsiPullbackTickerGrid(t, ticker, file string)
  map[string][]float64` (`rsi_pullback_grid_test.go:40`) и `containsFloat(values []float64, want
  float64) bool` (`rsi_pullback_cnru_grid_test.go`).
- Produces: двенадцать путей `data/params/rsi_pullback/mdmg/cal_*.json` для задач 3–10; тест
  `TestMDMGGridsStayWide`; переменная `mdmgGridFiles` и константа `mdmgStopCeiling` (их расширяет
  Task 12).

- [ ] **Step 1: Написать падающий сторожевой тест.**

```go
package backtest

import "testing"

// mdmgGridFiles перечисляет сетки MDMG ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN, RTKM, SVCB, RAGR.
var mdmgGridFiles = []string{
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
	"cal_trend_spent.json",
	"cal_trend_day.json",
}

// mdmgCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast (trend_day), живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const mdmgCoreEMAFast = 10

// mdmgCoreEMASlow — дефолт ядра core.DefaultParams().EMASlow. Сетка, свипующая EMAFast без
// EMASlow (trend_spent), живёт на этом значении.
const mdmgCoreEMASlow = 100

// mdmgStopCeiling — потолок риск-гейта A (§5.4 спеки). Правила каталога на MDMG расходятся:
// выживаемость 30% даёт 1.0, частота срабатывания не реже 10% сделок — 0.8; берётся строжайшее.
const mdmgStopCeiling = 0.8

// TestMDMGGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestMDMGGridsStayWide(t *testing.T) {
	for _, file := range mdmgGridFiles {
		grid := rsiPullbackTickerGrid(t, "mdmg", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("mdmg/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("mdmg/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["EMAFast"] {
			if v < 1 {
				t.Errorf("mdmg/%s: EMAFast=%v нарушает EMAFast >= 1", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("mdmg/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{mdmgCoreEMAFast}
		}
		slow := grid["EMASlow"]
		if len(slow) == 0 {
			slow = []float64{mdmgCoreEMASlow}
		}
		for _, f := range fast {
			for _, s := range slow {
				if f >= s {
					t.Errorf("mdmg/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
		// Стоп меряет только тема risk (§5.1 спеки): арбитры первого круга MDMG — связки тренда и
		// дневного гейта, а не связки со стопом.
		if file != "cal_risk.json" && len(grid["StopDailyATR"]) > 0 {
			t.Errorf("mdmg/%s: свипует StopDailyATR %v — стоп меряет только cal_risk.json", file, grid["StopDailyATR"])
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSIPeriod", []float64{2, 14}},
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMAFast", []float64{1}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{30, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.1, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0.05, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 2.5}},
		{"cal_volume.json", "VolMult", []float64{4.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.2, 1.5}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "mdmg", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("mdmg/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/backtest/ -run MDMGGridsStayWide -v`
  Expected: FAIL — файлов сеток ещё нет.

- [ ] **Step 3: Написать двенадцать файлов сеток.** Формат (образец —
  `data/params/rsi_pullback/ragr/cal_entry.json`):

```json
{
  "_comment": "data/params/rsi_pullback/mdmg/cal_entry.json — тема entry для MDMG, 100 прогонов, поверх ДЕФОЛТОВ ЯДРА. <что меряет>. ОКНО: 24 месяца после редомициляции (-months 24, схема 24/12/3, -min-trades 20), числа несравнимы с 36-месячным каталогом. ЗАМЕР 2026-09-26 (in-sample, §6.1 спеки docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md): <числа оси>. ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: <вырожденные узлы>. ЗАПУСК: go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/mdmg/cal_entry.json -out ./reports/MDMG_entry -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА: (заполняется задачей плана).",
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
| `cal_entry.json` | `entry` | `RSIPeriod` [2,3,4,5,6,7,8,10,12,14] × `RSILower` [5,10,15,20,25,30,35,40,45,50] | 100 | 20 | 5 |
| `cal_trend.json` | `trend` | `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] | 48 | 20 | 5 |
| `cal_trend_low.json` | `trend_low` | `EMAFast` [1,2,3,5,8,10] × `EMASlow` [12,15,20,25,30,35,40,45] | 48 | 20 | 5 |
| `cal_day.json` | `day` | `FreshDayATR` [0,0.05,0.1,0.15,0.2,0.3,0.4,0.5] × `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.1,1.25,1.5,2.0,2.5] | 96 | 20 | 5 |
| `cal_volume.json` | `volume` | `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0,3.5,4.0] × `VolBaseDays` [3,5,10,14,20,30] | 48 | 20 | 5 |
| `cal_vol_window.json` | `vol_window` | `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult` [1.0,1.2,2.0,3.0] | 36 | 20 | 5 |
| `cal_risk.json` | `risk` | `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.2,1.3,1.5,2.0] × `TPDailyATR` [0.1,0.15,0.2,0.25,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.5] | 132 | 20 | 5 |
| `cal_exit.json` | `exit` | `RSIUpper` [30,35,40,45,50,55,60,65,70,75,80,85,90,95] | 14 | 20 | 5 |
| `cal_trail.json` | `trail` | `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.2,0.3,0.4,0.5,0.7,1.0,1.5] | 14 | 20 | 5 |
| `cal_trend_spent.json` | `trend_spent` | `EMAFast` [1,2,3,5,10] × `SpentDayATR` [0.7,0.8,0.9,1.0,1.1,1.25,1.5] | 35 | 20 | 5 |
| `cal_trend_day.json` | `trend_day` | `EMASlow` [15,20,25,30,40,50,75,100] × `FreshDayATR` [0,0.05,0.1,0.15,0.2] | 40 | 20 | 5 |

  **Что обязательно написать в `_comment` каждого файла:**
  - что тема меряет и сколько в ней прогонов;
  - замер из §6 спеки, из которого получена каждая ось;
  - оговорку об окне (24 месяца после редомициляции, 24/12/3, `-min-trades 20`, несравнимость с
    каталогом);
  - предупреждения о краях, по файлам:
    - `cal_entry.json`: `RSILower` 5 — 3 сделки за окно, 10 — 27 (PF 2.286, но хвост 6 на восьми
      сделках); `RSIPeriod` 12 — 17 сделок, 14 — 11; узлы 7–8 проваливают хвост 6 (§6.1);
    - `cal_trend_low.json`: `EMAFast` 1 — узел за краем RAGR, это само закрытие против медленной EMA
      (1 → 2.020/86, 2 → 2.003/106); `EMASlow` ниже 12 и пары `EMAFast ≥ EMASlow` намеренно не
      входят (§5.1);
    - `cal_risk.json`: узел `TPDailyATR` 2.5 — контрольная строка, которую требует
      `TestRSIPullbackGridControlPoints` (цель строго выше самого широкого стопа 2.0); стоп выше 0.8
      режет гейт A — капкан: пул застывает на 134 сделках с 0.5, PF 2.0–2.3 на 1.0+ при доле SL
      2–7.5%; стопы 0.6–0.8 пробивают гейт B (DD 9.5–10.7%); ось остаётся широкой, чтобы решение
      было видно в отчёте (§5.1 п.5);
    - `cal_exit.json`: дефолт 70 — пик оси, все прочие узлы дают DD 11–19 тыс. ₽ (§6.6);
    - `cal_trail.json`: 1.5 побайтово равен дефолтам (трейл не срабатывает); при `UseRSIExit` 0
      трейл вреден (0.80–1.15);
    - `cal_day.json`: `FreshDayATR` 0.05 открывает вход на первом баре дня (урок X5) — 19 входов в
      02–06 против двух у дефолтов, гейт C (§5.6); узлы 0.05, 0.1, 0.15 in-sample побайтово равны;
      `SpentDayATR` 2.0 — 11 сделок, 2.5 — 5 (край, PF 0.449); горб 1.25–1.5 держит хвост 6 на 4–7
      сделках;
    - `cal_volume.json`: 3.5 и 4.0 — узлы за краем RAGR, лучший in-sample узел 3.0 × 20 (2.297/43)
      лежал на краю; 4.0 × 20 → 2.171/31;
    - `cal_trend_spent.json`, `cal_trend_day.json`: тема — **арбитр**, а не источник полей (§5.2
      спеки); голоса в сборку не идут, но записываются всегда (по ним выбираются зоны второго круга);
  - полную команду запуска с путём самого файла (`TestRSIPullbackCalFilesValid` требует, чтобы
    `_comment` содержал `mdmg/<имя файла>`); `-min-trades` — по таблице;
  - место под результат прогона.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ -run 'MDMGGridsStayWide|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints|RSIPullbackPointFilesArePoints' -v`
  Expected: PASS.

- [ ] **Step 5: Мутационная проверка сторожа.** Временно добавить `"StopDailyATR": [0.5, 0.8]` в
  `grid` файла `cal_trend_spent.json` и прогнать `go test ./internal/service/backtest/ -run MDMGGridsStayWide`
  — Expected: FAIL с текстом `стоп меряет только cal_risk.json`. Затем временно убрать узел 1 из
  `EMAFast` в `cal_trend_low.json` — Expected: FAIL `потеряла обязательный узел 1`. Затем временно
  добавить узел 100 в `EMAFast` файла `cal_trend_spent.json` — Expected: FAIL
  `нарушает EMAFast < EMASlow` (пара с дефолтной `EMASlow` 100). Вернуть все правки, прогнать снова
  — PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): каталог сеток MDMG`.

---

### Task 3: Тема `screen` — цена двух гейтов и сверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_screen.json` (`_comment`); Create
`docs/superpowers/plans/task-3-report-mdmg.md`

**Interfaces:**
- Consumes: `cal_screen.json` из Task 2.
- Produces: голоса фолдов по `UseDayATRGate` и `UseVolume` — их читает Task 11; фактические
  train/test-окна фолдов на дату запуска — их читает Task 11 Step 13.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_screen.json -out ./reports/MDMG_screen \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Сверить число фолдов и окно.** В шапке `*_walkforward.md` должно стоять
  «Train-окно: 12 мес; OOS-фолд: 3 мес» и «Фолдов: 4»; train фолда 1 начинается не раньше
  2024-09-01. Иначе **остановиться и доложить владельцу**. Записать фактические train/test-окна
  четырёх фолдов.

- [ ] **Step 3: Записать** pooled OOS PF, размер пула и голоса всех четырёх фолдов по обеим осям;
  пометить вырожденные фолды.

- [ ] **Step 4: Сверить с ожиданием** (§6.3 спеки): **ожидается 1×0** — дневной гейт включён,
  объёмный при дефолтном множителе 1.2 выключен. Замер in-sample: 1×0 → 1.546/134, 1×1 → 1.349/101,
  0×1 → 1.319/247, 0×0 → 1.298/376 (DD 24 447 ₽). Голос за `UseVolume` 1 здесь меряет объём при
  `VolMult` 1.2 — строгий множитель решают `volume` и `vol_window` (Task 7); записать это.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-3-report-mdmg.md` (образец —
  `docs/superpowers/plans/task-3-report-ragr.md`) и дописать результат в `_comment` сетки строкой
  `РЕЗУЛЬТАТ ПРОГОНА <дата>: …` с pooled OOS, пулом, фолдами и голосами.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): MDMG, тема screen`.

---

### Task 4: Тема `entry` — первая половина планки

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_entry.json` (`_comment`); Create
`docs/superpowers/plans/task-4-report-mdmg.md`

**Interfaces:**
- Consumes: `cal_entry.json` из Task 2.
- Produces: голоса по `RSIPeriod` и `RSILower`, вердикт первой половины планки — их читают Task 11 и
  Task 12.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_entry.json -out ./reports/MDMG_entry \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS PF, пул и голоса четырёх фолдов (`RSILower` — ведущая ось);
  пометить вырожденные фолды.

- [ ] **Step 3: Проверить первую половину планки** (§5.12): pooled OOS ≥ **1.5** при ≥ **20**
  сделках **и** `RSILower` одинаков в ≥ 3 фолдах из 4. Вырожденный фолд в пользу тикера не
  засчитывается.

- [ ] **Step 4: Сверить с ожиданием** (§6.1, §5.12): ось `RSILower` рваная (10 → 2.286/27,
  20 → 1.621, 25 → 1.792, 30 → 1.546, 35 → 1.717), большинство 3/4 маловероятно. `RSIPeriod` 7–8
  проваливают хвост 6 in-sample — голос за них записать с этой пометкой.

- [ ] **Step 5: Отметить вырожденные узлы, если фолды их выбрали:** `RSILower` 5 (3 сделки за
  окно), `RSIPeriod` 12 и 14 (17 и 11) — на 12-месячном train такой узел почти наверняка ниже
  `-min-trades 20` и тонет в рейтинге; если всё же выбран, записать.

- [ ] **Step 6: Написать отчёт** `task-4-report-mdmg.md` и дописать результат в `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): MDMG, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая половина планки

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_trend.json`, `cal_trend_low.json`
(`_comment`); Create `docs/superpowers/plans/task-5-report-mdmg.md`

**Interfaces:**
- Consumes: оба файла из Task 2.
- Produces: голоса по `EMAFast` и `EMASlow` из обеих тем и их pooled OOS — их читают Task 10 и
  Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_trend.json -out ./reports/MDMG_trend \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_trend_low.json -out ./reports/MDMG_trend_low \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов (`EMASlow` —
  ведущая ось); пометить вырожденные фолды.

- [ ] **Step 3: Проверить вторую половину планки** по **канонической** `trend`, не по `trend_low`.

- [ ] **Step 4: Сверить с ожиданиями** (§6.2 спеки).
  - `trend`: каноническая ось `EMASlow` 50–250 лежит в 1.389–1.700 in-sample; ось `EMAFast` —
    максимум на 3 (1.758/118). Планку тема может взять по PF, но не по устойчивости ведущей оси.
  - `trend_low`: in-sample пик `EMASlow` 20–25 → 1.900 (оба года в плюс, DD 3.5–4.7%); `EMAFast` 1–2
    → 2.0. Голос за `EMAFast` 1 записать отдельно: это минимум оси, зонда за краем нет.

- [ ] **Step 5: Записать, какая тема дала больший pooled OOS**, и предварительный вывод по полю
  тренда для Task 11 (правило §5.2: `trend_low` — только при большинстве ≥ 3/4 **и** pooled OOS
  выше, чем у `trend`).

- [ ] **Step 6: Написать отчёт** `task-5-report-mdmg.md` и дописать результаты в оба `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): MDMG, темы trend и trend_low`.

---

### Task 6: Тема `day`

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_day.json` (`_comment`); Create
`docs/superpowers/plans/task-6-report-mdmg.md`

**Interfaces:**
- Consumes: `cal_day.json` из Task 2.
- Produces: голоса по `FreshDayATR` и `SpentDayATR` и pooled OOS — их читают Task 10 и Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_day.json -out ./reports/MDMG_day \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям; пометить
  вырожденные фолды.

- [ ] **Step 3: Записать опорный замер `SpentDayATR`** (§6.4) — главный рычаг: 0.8 → 1.546
  (дефолт), **0.9 → 2.243/107** (DD 4.46%, хвост 6 2.303/20), 1.0 → 2.129/77, 1.25 → 3.587/46,
  1.5 → 3.835/31, 2.0 → 1.658/11, 2.5 → 0.449/5. Голос за 1.25–1.5 записать с пометкой «хвост 6 на
  4–7 сделках» — кандидат на провал пункта 6 (меньше пяти сделок).

- [ ] **Step 4: Записать опорный замер `FreshDayATR`:** 0.05 = 0.1 = 0.15 → 1.674/151,
  0.2 → 1.690/154, 0.4 → 1.474 (DD 14 181 ₽ — выше гейта B). Ненулевое значение заводит вход на
  первый бар дня — **19 входов в 02–06**, кандидат на провал гейта C в Task 11 Step 6.

- [ ] **Step 5: Написать отчёт** `task-6-report-mdmg.md` и дописать результат в `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): MDMG, тема day`.

---

### Task 7: Темы `volume` и `vol_window`

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_volume.json`, `cal_vol_window.json`
(`_comment`); Create `docs/superpowers/plans/task-7-report-mdmg.md`

**Interfaces:**
- Consumes: оба файла из Task 2.
- Produces: голоса по `VolMult`, `VolBaseDays`, `VolLookbackBars` и pooled OOS обеих тем — их читает
  Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_volume.json -out ./reports/MDMG_volume \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_vol_window.json -out ./reports/MDMG_vol_window \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с ожиданием.** Обе темы идут с `UseVolume` 1. Поле `UseVolume` в точку
  включается, **только если** тема объёма (`volume` или `vol_window` по правилу §5.2) большинством
  3/4 держит одно значение множителя **и** её pooled OOS выше pooled OOS `screen` на узле 1×0
  (дефолты); голос `screen` за `UseVolume` 0 большинством 3/4 объём закрывает (§5.2). Опорный замер
  (§6.3): `VolMult` 3.0 × `VolBaseDays` 20 → 2.297/43 (хвост 6 — 6 сделок без убытков),
  4.0 × 20 → 2.171/31; `vol_window` in-sample 1.37–1.56. При `-min-trades 20` на 12-месячном train
  узлы 3.0+ дают около 20–25 сделок — на границе порога; записать фактическое число сделок train
  у выбранных узлов.

- [ ] **Step 4: Написать отчёт** `task-7-report-mdmg.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): MDMG, темы volume и vol_window`.

---

### Task 8: Тема `risk`, потолок гейта A и зонды капкана

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_risk.json` (`_comment`); Create
`data/params/rsi_pullback/mdmg/probe_stop_10.json`, `probe_stop_12.json`; Create
`docs/superpowers/plans/task-8-report-mdmg.md`

**Interfaces:**
- Consumes: `cal_risk.json` из Task 2; `mdmg_journal.py`.
- Produces: голоса по `StopDailyATR` (уже пропущенные через потолок 0.8) и по `TPDailyATR`, pooled
  OOS — их читает Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_risk.json -out ./reports/MDMG_risk \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям.

- [ ] **Step 3: Применить потолок гейта A.** Голос выше **0.8** заменяется на 0.8, цена решения в PF
  записывается прямым текстом. Обоснование повторить в отчёте (§5.4 спеки): выживаемость 30% даёт
  1.0 (38.0% дней), частота срабатывания ≥ 10% сделок даёт 0.8 (16.4%; 1.0 → 7.5%); берётся
  строжайшее. Пул застывает на 134 сделках уже с 0.5, PF 2.0–2.3 на 1.0+ — подпись капкана.
  **Ожидание:** фолды проголосуют за 1.0+; после потолка 0.8 гейт B (Task 11 Step 4), вероятно,
  вернёт стоп на 0.5 (0.6–0.8 дают DD 9.5–10.7% при планке 7.54%).

- [ ] **Step 4: Снять два зонда капкана на полном окне.** Каждый файл — одна комбинация поверх
  дефолтов ядра:
  - `probe_stop_10.json`: `{"_comment":"…","phases":[{"name":"probe","grid":{"StopDailyATR":[1.0]},"keepTop":1}]}`;
  - `probe_stop_12.json` — то же со `StopDailyATR` [1.2].

  В `_comment` — назначение зонда («цена потолка 0.8, за потолком — капкан»), опорный замер и
  команда запуска с путём файла. Команда (последовательно):

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/probe_stop_10.json -out ./reports/MDMG_probe_stop_10 \
  -months 24 -min-trades 1 -metric profit_factor
python3 reports/_analysis/mdmg_journal.py reports/MDMG_probe_stop_10/*_best.md
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/probe_stop_12.json -out ./reports/MDMG_probe_stop_12 \
  -months 24 -min-trades 1 -metric profit_factor
python3 reports/_analysis/mdmg_journal.py reports/MDMG_probe_stop_12/*_best.md
```

  По каждому зонду записать PF, сделок, число SL-выходов, обе формы DD, удержание, ночёвки и годы.
  Опорные числа: 1.0 → 2.009/134, 10 SL (7.5%), DD 8.30%; 1.2 → 2.320/134, 6 SL (4.5%), DD 7.73%.

- [ ] **Step 5: Записать опорный замер `TPDailyATR`** (§6.5): 0.3 → 1.659/154 (хвост 6 1.369/24,
  DD 7.94% — выше гейта B), 0.6 → 1.546 (дефолт), 0.1 → 1.099. Цель на MDMG — не рычаг.

- [ ] **Step 6: Прогнать тесты:** `go test ./internal/service/backtest/ -run 'MDMG|RSIPullback'`.
  Expected: PASS.

- [ ] **Step 7: Написать отчёт** `task-8-report-mdmg.md` и дописать результат в `_comment`.

- [ ] **Step 8: Коммит** `docs(rsi_pullback): MDMG, тема risk и зонды капкана`.

---

### Task 9: Темы `exit` и `trail`

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_exit.json`, `cal_trail.json` (`_comment`);
Create `docs/superpowers/plans/task-9-report-mdmg.md`

**Interfaces:**
- Consumes: оба файла из Task 2.
- Produces: голоса по `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` и pooled OOS — их читает
  Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_exit.json -out ./reports/MDMG_exit \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_trail.json -out ./reports/MDMG_trail \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с опорными замерами** (§6.6 спеки).
  - `RSIUpper`: дефолт 70 — пик оси; прочие узлы дают DD 11–19 тыс. ₽ и не пройдут гейт B.
  - Трейл: при `UseRSIExit` 1 лучший 0.7 → 1.566/137, прочие ниже дефолта. Если тема голосует за
    `UseTrail` 1, гейт A применяется к `min(StopDailyATR, TrailDailyATR)` (урок AFKS) — записать.

- [ ] **Step 4: Написать отчёт** `task-9-report-mdmg.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): MDMG, темы exit и trail`.

---

### Task 10: Темы-арбитры `trend_spent` и `trend_day`

**Files:** Modify `data/params/rsi_pullback/mdmg/cal_trend_spent.json`, `cal_trend_day.json`
(`_comment`); Create `docs/superpowers/plans/task-10-report-mdmg.md`

**Interfaces:**
- Consumes: оба файла из Task 2; pooled OOS и голоса `trend`/`trend_low` (Task 5), `day` (Task 6).
- Produces: вердикты «рычаги складываются / конкурируют / арбитр справочный» по двум парам и голоса
  арбитров — их читают Task 11 Step 1 и Task 12 Step 1.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_trend_spent.json -out ./reports/MDMG_trend_spent \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal_trend_day.json -out ./reports/MDMG_trend_day \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов по обеим осям;
  пометить вырожденные фолды.

- [ ] **Step 3: Вынести вердикт каждого арбитра** (§5.2 спеки). Пара «тема тренда, давшая `EMAFast`»
  × `day` (ось `SpentDayATR`) — судит `trend_spent`; пара «тема тренда, давшая `EMASlow`» × `day`
  (ось `FreshDayATR`) — судит `trend_day`:
  - обе темы пары голосуют большинством ≥ 3/4 за уход от дефолта, и pooled OOS арбитра **не ниже**
    обеих — рычаги **складываются**, оба поля идут в точку по своим темам;
  - pooled OOS арбитра **ниже любой** из двух — рычаги **конкурируют**: в точку идёт один рычаг,
    тот, чья тема дала больший pooled OOS; второе поле остаётся на дефолте ядра; цена решения пишется
    прямым текстом;
  - большинство за уход от дефолта есть не больше чем у одной темы пары — арбитр **справочный**.

  Голоса `EMAFast`, `SpentDayATR`, `EMASlow`, `FreshDayATR` **из арбитров в сборку не идут** —
  записываются для второго круга.

- [ ] **Step 4: Записать опорные числа разведки** (§6.2, §6.4): одиночные рычаги `EMAFast` 2
  (2.003/106), `SpentDayATR` 0.9 (2.243/107), `EMASlow` 25 (1.900/107) каждый проходит пункты 5, 6 и
  гейт B; `EMAFast` и `SpentDayATR` режут одни и те же сделки, и их сумма может оказаться хуже
  каждого по отдельности. `FreshDayATR` ≠ 0 почти наверняка упрётся в гейт C.

- [ ] **Step 5: Написать отчёт** `task-10-report-mdmg.md` и дописать результаты в оба `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): MDMG, темы-арбитры trend_spent и trend_day`.

---

### Task 11: Сборка точки, гейты, walk-forward и семь пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/mdmg/plateau_point.json` и соседи плато
`plateau_<поле>_<значение>.json`; Create `docs/superpowers/plans/task-11-report-mdmg.md`

**Interfaces:**
- Consumes: голоса и вердикты задач 3–10; `mdmg_journal.py`, `sfin_sweep.py`; переснятые числа
  baseline (Task 1 Steps 6–7).
- Produces: точку первого круга (восемнадцать полей), её числа по всем схемам и вердикт — их читают
  Task 12 и Task 13.

- [ ] **Step 1: Собрать точку правилом большинства** (§5.2 спеки):
  - поле берётся из темы, которая его меряет, только при ≥ 3 голосах из 4 **невырожденных**
    фолдов; иначе — дефолт ядра; ничья 2/2 большинством не считается;
  - источники: `RSIPeriod`, `RSILower` — `entry`; `StopDailyATR`, `TPDailyATR` — `risk`;
    `FreshDayATR`, `SpentDayATR` — `day`; `RSIUpper` — `exit`; `UseRSIExit`, `UseTrail`,
    `TrailDailyATR` — `trail`; `UseDayATRGate`, `UseVolume` — `screen` (с уточнением по объёму из
    Task 7 Step 3);
  - поле тренда: из `trend_low` только при большинстве ≥ 3/4 **и** pooled OOS выше `trend`; иначе из
    `trend`;
  - поля объёма: из `vol_window` при превосходстве над `volume`, и только если `screen` не
    проголосовала большинством за `UseVolume` 0;
  - вердикты арбитров применяются по Task 10 Step 3.

  Записать таблицу для всех восемнадцати полей: «поле → тема-источник → голоса по фолдам →
  вырожденные фолды → большинство без вырожденных → принятое значение → дефолт ядра».

- [ ] **Step 2: Применить гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **0.8**.

- [ ] **Step 3: Написать `plateau_point.json`** — одна фаза, все восемнадцать полей по одному
  значению (образец — `data/params/rsi_pullback/ragr/plateau_point2.json`). Снять одиночный прогон:

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/plateau_point.json -out ./reports/MDMG_point \
  -months 24 -min-trades 1 -metric profit_factor
python3 reports/_analysis/mdmg_journal.py reports/MDMG_point/*_best.md
```

  **Если все восемнадцать полей совпали с дефолтами ядра**, точка тождественна baseline: записать
  это (маршрут CNRU), пропустить Steps 4–13 и в Step 14 взять числа baseline из Task 1.

- [ ] **Step 4: Применить гейт B:** max DD ≤ **9 153.17 ₽ и ≤ 7.54%** (или переснятые числа Task 1
  Step 6), обе формы. При пробое в точку идёт ближайший вариант плато с меньшей просадкой, цена
  пишется прямым текстом. Типичный случай — стоп 0.6–0.8: вернуть стоп на 0.5 и пересчитать с Step 3.

- [ ] **Step 5: Гейт D:** из блока «ГЕЙТ D» — сделок через отсечки **0**. Иначе сработал пункт 7.

- [ ] **Step 6: Гейт C:** входов в выходные **0** и входов 02–06 (блок «ГЕЙТ C») **≤ 2**. При провале
  увести `FreshDayATR` на 0 и пересчитать с Step 3. Если провал остался — сработал пункт 7.

- [ ] **Step 7: Снять соседей плато.** Для каждого поля, ушедшего от дефолта, — прогон самого
  значения и двух соседей по оси, команда как в Step 3, файлы `plateau_<поле>_<значение>.json`
  (значение без точки: `plateau_spent_09.json`, `plateau_emafast_2.json`).
  - Край сетки дополнительно проверяется **зондом за краем**: `SpentDayATR` 2.75 за краем 2.5,
    `VolMult` 5.0 за краем 4.0, `EMASlow` 300 за краем 250. Два края зонда не имеют, и это
    записывается: `RSILower` 50 (выше запрещено инвариантом `RSILower ≤ 50`) и `EMAFast` 1 (минимум
    оси).
  - Соседа по оси стопа при включённом трейле проверяют в сторону уменьшения.
  - **Для каждого соседа снять годы и хвосты** скриптом `mdmg_journal.py` (§5.3 спеки).
  - Разница PF < 0.05 решается в пользу меньшей просадки, затем меньшего числа убыточных полугодий
    (§5.5).
  - Плато уже 0.05 PF записывается как отсутствие сигнала.

- [ ] **Step 8: Третий контур.** Сравнить с дефолтами: SL 24.6%, удержание 9/19/33, ночёвок 34.3%,
  переносов 2, DD 7.54%, expectancy +213.79 ₽, выходов в выходные 5, выходов 02–06 6. **Падение
  доли SL вместе с ростом удержания и ночёвок — капкан**, тогда точка берёт более узкий стоп в
  пределах плато.

- [ ] **Step 9: Два walk-forward точки (последовательно).**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/plateau_point.json -out ./reports/MDMG_point_wf123 \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/plateau_point.json -out ./reports/MDMG_point_wf153 \
  -months 24 -train-months 15 -test-months 3 -min-trades 1 -metric profit_factor
```

  Ожидаемое число фолдов: **4 и 3**; расхождение — остановиться и доложить (ловушка ASTR). Записать
  pooled OOS, пул и пофолдовые числа, пометить вырожденные фолды.

- [ ] **Step 10: Утяжелённые издержки.**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/plateau_point.json -out ./reports/MDMG_point_c001 \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor -commission 0.001
```

- [ ] **Step 11: Календарные годы.** Из блока «календарные годы» Step 3: 2025 и 2026 — net > 0.
  Огрызок 2024 записать отдельно, пунктом не считать.

- [ ] **Step 12: Хвосты.** Из блока «ПУНКТ 6» Step 3 записать хвост 6 и хвост 12 (сделок, net, PF) и
  долю крупнейшей сделки в net хвоста 6 — сделку с максимальным |PnL| среди входов с 2026-03-26.

- [ ] **Step 13: Сверка знака хвоста 6 с фолдами 3–4.** Отчёт walk-forward не печатает сделки по
  фолдам, поэтому объединённый OOS PF фолдов 3 и 4 восстанавливается из таблицы «Результаты по
  фолдам» (каждый фолд стартует с капитала 100 000 ₽: `OOS NetPnL%` × 1000 = net в ₽; тогда
  `GL = net / (PF − 1)`, `GP = PF × GL`):

```bash
python3 - reports/MDMG_point_wf123/*_walkforward.md <<'EOF'
import sys
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
  `reports/MDMG_prep/wf/wf_24_12_3_0.0005/MDMG_rsi_pullback_Minutes30_20260926_230859_walkforward.md`
  скрипт обязан дать **1.729 ± 0.005** (фолд 3: 0.927, −260 ₽; фолд 4: 2.228, +7 040 ₽), что
  совпадает с хвостом 6 дефолтов 1.727. Иное число — ошибка разбора, исправить до применения к точке.
  Если фолд без убыточных сделок (PF «+Inf» или ошибка разбора) — вырожденный, сверка по одному
  оставшемуся фолду с оговоркой. Знак объединённого PF (≥ 1 или < 1) обязан совпасть со знаком
  хвоста 6. Расхождение — **остановиться и доложить владельцу** оба числа и фактические тест-окна
  фолдов 3–4.

- [ ] **Step 14: Применить семь пунктов стоп-условия** (Global Constraints). По каждому пункту
  записать число и вердикт.

- [ ] **Step 15: Проверить планку** (Task 4 Step 3, Task 5 Step 3). Провал планки на заведение не
  влияет (§5.12).

- [ ] **Step 16: Прогнать тесты** (файлы-точки обязаны нести ровно одно значение на ключ):
  `go test ./internal/service/backtest/ -run 'MDMG|RSIPullback'`. Expected: PASS.

- [ ] **Step 17: Написать отчёт** `task-11-report-mdmg.md`. Содержание:
  - таблица сборки с вырожденными фолдами;
  - гейты A, B, C, D с ценой решений;
  - соседи плато с годами и хвостами;
  - третий контур;
  - два walk-forward и издержки;
  - годы, хвосты, доля крупнейшей сделки и сверка знака;
  - семь пунктов и планка;
  - строка «точка против дефолтов» на 24/12/3 (pooled OOS обоих, DD обоих) — для Task 13.

- [ ] **Step 18: Вердикт первого круга.**
  - Ни один пункт не сработал — точка допущена кандидатом; Task 12 пропускается, переход к Task 13.
  - Иначе — переход к Task 12.

- [ ] **Step 19: Коммит** `feat(rsi_pullback): MDMG, точка первого круга и вердикт`.

---

### Task 12: Второй круг (только при провале первого)

**Files:** Create `data/params/rsi_pullback/mdmg/cal2_<тема>.json` (не больше шести),
`plateau_point2.json`, соседи `plateau_r2_<поле>_<значение>.json`; Modify
`internal/service/backtest/rsi_pullback_mdmg_grid_test.go`; Create
`docs/superpowers/plans/task-12-report-mdmg.md`

**Задача выполняется ТОЛЬКО при срабатывании хотя бы одного пункта стоп-условия в Task 11.**

**Interfaces:**
- Consumes: голоса задач 3–10 (включая арбитров), вердикт Task 11, `mdmgGridFiles` и
  `mdmgStopCeiling` из Task 2.
- Produces: точку второго круга и её вердикт — их читает Task 13.

- [ ] **Step 1: Выбрать зоны по фактическим голосам фолдов первого круга**, включая голоса
  арбитров `trend_spent` и `trend_day`, а не по in-sample рельефу. Шаг — половина шага первого
  круга. Не больше шести тем. **Каждое поле свипует ровно одна тема** второго круга, и она — его
  источник (§5.14): тема-связка (например, `cal2_trend_spent`) забирает оба своих поля, одиночные
  темы тех же полей не заводятся. В `_comment` каждого файла записать, из каких голосов какого фолда
  получена зона, и команду запуска с путём `mdmg/cal2_<тема>.json`.
  - Кандидат номер один — `cal2_trend_spent`, если голоса `trend_low` / `trend_spent` / `day`
    рассыпались по зоне короткой EMA и строгого дня: `EMAFast` [1,2,3,4] × `SpentDayATR`
    [0.8,0.85,0.9,0.95,1.0,1.05,1.1].
  - Кандидат номер два — `cal2_trend_day`: `EMASlow` [18,20,22,24,25,26,28,30] × `FreshDayATR` [0]
    (фиксирован — гейт C) — только если `EMASlow` не свипует другая тема второго круга. `EMASlow`
    целочисленный, поэтому половинный шаг округлён до целых узлов.
  - Стоп, если его свипует тема второго круга, — не выше **0.8**; при свипе стопа файл обязан
    содержать цель строго выше самого широкого стопа (`TestRSIPullbackGridControlPoints`).
  - Целочисленные поля `core.Params` (`RSIPeriod`, `EMAFast`, `EMASlow`, `VolBaseDays`,
    `VolLookbackBars` и флаги) берут только целые узлы — `applyField` отвергает дробные.

- [ ] **Step 2: Расширить сторожевой тест.** В `rsi_pullback_mdmg_grid_test.go` добавить переменную
  `mdmgRound2GridFiles` с поимённым списком файлов `cal2_*.json` и функцию:

```go
// mdmgAllGridFiles склеивает сетки обоих кругов: жёсткие инварианты §5.1 держатся на обоих.
func mdmgAllGridFiles() []string {
	out := make([]string, 0, len(mdmgGridFiles)+len(mdmgRound2GridFiles))
	out = append(out, mdmgGridFiles...)
	return append(out, mdmgRound2GridFiles...)
}
```

  В первом цикле `TestMDMGGridsStayWide` заменить `for _, file := range mdmgGridFiles` на
  `for _, file := range mdmgAllGridFiles()`, а проверку «стоп меряет только `cal_risk.json`» оставить
  только для файлов первого круга:

```go
		if !strings.HasPrefix(file, "cal2_") && file != "cal_risk.json" && len(grid["StopDailyATR"]) > 0 {
```

  (добавить импорт `strings`). В конец теста — потолок второго круга:

```go
	// Потолок гейта A во втором круге не двигается (§5.14 спеки).
	for _, file := range mdmgRound2GridFiles {
		for _, v := range rsiPullbackTickerGrid(t, "mdmg", file)["StopDailyATR"] {
			if v > mdmgStopCeiling {
				t.Errorf("mdmg/%s: StopDailyATR=%v выше потолка гейта A %v", file, v, mdmgStopCeiling)
			}
		}
	}
```

  И тест «одно поле — одна тема»:

```go
// TestMDMGRound2FieldsHaveOneSource держит правило §5.14 спеки: во втором круге каждое поле
// свипует ровно одна тема, иначе две темы проголосуют за одно поле по-разному.
func TestMDMGRound2FieldsHaveOneSource(t *testing.T) {
	owner := map[string]string{}
	for _, file := range mdmgRound2GridFiles {
		for field, values := range rsiPullbackTickerGrid(t, "mdmg", file) {
			if len(values) < 2 {
				continue // зафиксированное поле не голосует
			}
			if prev, ok := owner[field]; ok {
				t.Errorf("mdmg: поле %s свипуют две темы второго круга: %s и %s", field, prev, file)
			}
			owner[field] = file
		}
	}
}
```

  Run: `go test ./internal/service/backtest/ -run 'MDMG|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints' -v`
  Expected: PASS.

- [ ] **Step 3: Прогнать каждую узкую тему (последовательно).**

```bash
go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mdmg/cal2_<тема>.json -out ./reports/MDMG_r2_<тема> \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor
```

  `-min-trades 1` — число сделок не критерий (прецедент SFIN).

- [ ] **Step 4: Собрать точку второго круга** правилом ≥ 3/4 невырожденных фолдов **в зоне ±1 шаг
  второго круга** (§5.14): голоса соседних узлов складываются в пользу центрального, если
  центральный сам набрал хотя бы один голос; при двух претендентах берётся тот, у кого больше
  собственных голосов, при равенстве — узел с большим pooled OOS одноточечного прогона. Правило
  записать в отчёт с разбивкой голосов. Поля, которые второй круг не свипует, берутся из точки
  первого круга. Прогнать точку через Task 11 Steps 2–16 без изменений, с файлами
  `plateau_point2.json` / `plateau_r2_*` и каталогами `MDMG_point2*`.

- [ ] **Step 5: Стоп-условие второго круга.**
  - Пункт 2 снят; фактический размер пула записывается.
  - **Пул OOS меньше десяти сделок закрывает работу второго круга как непредставительный.**
  - Пункты 5 и 6 — главные критерии, не смягчаются.
  - Пункты 1, 3, 4, 7 — в силе полностью. Потолок стопа 0.8 не двигается.

- [ ] **Step 6: Отчёт** `task-12-report-mdmg.md` той же структуры, что Task 11, плюс таблица «зона
  узкой сетки → голоса первого круга». Прогнать тесты:
  `go test ./internal/service/backtest/ -run 'MDMG|RSIPullback'`. Expected: PASS.

- [ ] **Step 7: Вердикт.** Точка второго круга прошла — она кандидат. Иначе кандидат только дефолты.
  В обоих случаях — переход к Task 13.

- [ ] **Step 8: Коммит** `feat(rsi_pullback): MDMG, второй круг и вердикт`.

---

### Task 13: Выбор победителя — точка против дефолтов (§5.15), или протокол отказа

**Files:** Create `docs/superpowers/plans/task-13-report-mdmg.md`. Только при отказе: Modify
`internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg.go` (doc-comment), `mdmg_test.go`
(комментарий теста), `internal/service/backtest/rsi_pullback_registry_test.go` (комментарий
`TestRSIPullbackMDMGTracksBaseline`)

**Interfaces:**
- Consumes: вердикт дефолтов (Task 1 Step 7), точка и её числа (Task 11 или Task 12).
- Produces: победитель — «точка» (литерал) или «дефолты» — либо «отказ»; его читают задачи 14–18.

- [ ] **Step 1: Составить список допущенных кандидатов.** Дефолты — если Task 1 Step 7 их допустил.
  Точка — если её круг не сработал ни по одному пункту и она собрана внутри гейтов A и B.

- [ ] **Step 2: Если допущены оба — пересчитать дефолты на дату выбора.** Числа точки и дефолтов
  обязаны быть сняты в один день (скользящее окно). Если Task 1 выполнялся в другой день, чем
  прогоны точки, повторить Task 1 Step 7 (схема 24/12/3) и Task 1 Step 6 (полное окно) в день
  выбора, в каталоги `reports/MDMG_base_final_wf123` и `reports/MDMG_base_final`.

- [ ] **Step 3: Применить правило выбора механически** (§5.15 спеки) и записать таблицу:

  | | Точка | Дефолты |
  |---|---|---|
  | pooled OOS PF 24/12/3 / пул | | |
  | pooled OOS PF 24/15/3 | | |
  | pooled OOS PF при `-commission 0.001` | | |
  | max DD полного окна (₽ / %) | | |
  | убыточных полугодий | | |
  | 2025 / 2026 net | | |
  | хвост 6 / хвост 12 | | |

  1. больший pooled OOS PF на 24/12/3;
  2. при разнице меньше 0.05 — меньшая max DD на полном окне (в процентах);
  3. при разнице DD меньше 0.5 п.п. — меньше убыточных полугодий;
  4. при полном равенстве — дефолты.

- [ ] **Step 4: Ветвление.**
  - Победила точка — Task 14 по маршруту A (литерал).
  - Победили дефолты — Task 14 по маршруту B (дефолты по выбору, прецедент TGKA/CNRU).
  - **Не допущен ни один кандидат** — протокол отказа: Step 5, затем задачи 14–17 не выполняются.

- [ ] **Step 5 (только при отказе): Протокол отказа.** Написать разбор в doc-comment пакета (образец —
  `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm.go`): всё из §5.16 спеки, какой
  пункт сработал у каждого кандидата и с какими числами. В doc-comment
  `TestParamsTrackTheBaselineUntilCalibrated` и `TestRSIPullbackMDMGTracksBaseline` записать
  окончательное состояние «калибровка закрыта отказом <дата>, MDMG в боевую вселенную не заводится».
  Run: `./bin/mage ci`. Expected: PASS. Доложить владельцу.

- [ ] **Step 6: Написать отчёт** `task-13-report-mdmg.md` с таблицей Step 3 и решением.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): MDMG, выбор победителя` (или `…, протокол отказа`).

---

### Task 14: Параметры победителя в пакете

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg.go`,
`mdmg_test.go`, `internal/service/backtest/rsi_pullback_registry_test.go`

**Выполняется ТОЛЬКО если Task 13 выбрал победителя.**

**Interfaces:**
- Consumes: победителя Task 13.
- Produces: `mdmg.DefaultParams()` — литерал (маршрут A) или `core.DefaultParams()` по решению
  (маршрут B); его читают Task 15 и Task 16.

**Маршрут A — победила точка.**

- [ ] **Step A1: Заменить тест пакета снимком.** Удалить `TestParamsTrackTheBaselineUntilCalibrated`,
  написать `TestParamsMatchTheCalibratedSnapshot`. Все восемнадцать полей выписаны литералами.
  Образец — `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr_test.go`. Скелет:

```go
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod: 4, RSILower: 30, RSIUpper: 70,
		EMAFast: 2, EMASlow: 100,
		DailyATRPeriod: 14, UseDayATRGate: 1, FreshDayATR: 0, SpentDayATR: 0.9,
		StopDailyATR: 0.5, TPDailyATR: 0.6,
		UseVolume: 0, VolBaseDays: 14, VolLookbackBars: 3, VolMult: 1.2,
		UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал MDMG разошёлся со снимком калибровки:\n got: %+v\nwant: %+v", got, want)
	}
}
```

  Значения в скелете — **пример формы** (дефолты + `EMAFast` 2 и `SpentDayATR` 0.9 из §6 спеки).
  Перед записью они заменяются числами принятой точки поле за полем по таблице отчёта Task 11 или
  12. Целочисленные поля `core.Params` (`core/core.go`): `RSIPeriod`, `EMAFast`, `EMASlow`,
  `DailyATRPeriod`, `UseDayATRGate`, `UseVolume`, `VolBaseDays`, `VolLookbackBars`, `UseRSIExit`,
  `UseTrail`; остальные — `float64`.

- [ ] **Step A2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/mdmg/ -v`
  Expected: FAIL.

- [ ] **Step A3: Поставить литерал.** `DefaultParams()` возвращает `core.Params` со всеми
  восемнадцатью полями **явно**, даже совпавшими с ядром. Строку «СОСТОЯНИЕ: калибровка не
  проводилась» в doc-comment заменить на «СОСТОЯНИЕ: откалиброван <дата>, разбор ниже» (разбор пишет
  Task 17).

- [ ] **Step A4: Обновить тест реестра.** `TestRSIPullbackMDMGTracksBaseline` заменить на
  `TestRSIPullbackMDMGIsRegisteredAndCalibrated` по образцу
  `TestRSIPullbackRAGRIsRegisteredAndCalibrated` (`rsi_pullback_registry_test.go:1081`): реестр отдаёт
  `rsipullbackmdmg.DefaultParams()`, и это не `core.DefaultParams()`.

**Маршрут B — победили дефолты.**

- [ ] **Step B1: Переименовать тест пакета в решение.** `TestParamsTrackTheBaselineUntilCalibrated`
  заменить на `TestParamsAreTheCoreBaselineByDesign` по образцу
  `internal/service/trading_strategy/rsi_pullback/strategy/cnru/cnru_test.go`: doc-comment говорит,
  что калибровка проведена <дата>, точка проиграла дефолтам по правилу §5.15 (или не собралась), и
  литерал без новой калибровки не появится. Тест проверяет `DefaultParams() == core.DefaultParams()`.

- [ ] **Step B2: Тест реестра бэктеста.** `TestRSIPullbackMDMGTracksBaseline` заменить на
  `TestRSIPullbackMDMGServesTheCoreBaselineByDesign` по образцу
  `TestRSIPullbackCNRUServesTheCoreBaselineByDesign` (`rsi_pullback_registry_test.go:1177`).

- [ ] **Step B3: Doc-comment пакета.** Строку «СОСТОЯНИЕ: калибровка не проводилась» заменить на
  «СОСТОЯНИЕ: калибровка проведена <дата>; в прод идут дефолты ядра по правилу §5.15 спеки, разбор
  ниже» (разбор пишет Task 17). Doc-comment `DefaultParams` — по образцу `cnru.DefaultParams`.

**Оба маршрута.**

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'MDMG|RSIPullback' -v`
  Expected: PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): литерал MDMG` (маршрут A) или
  `feat(rsi_pullback): MDMG на дефолтах ядра по выбору` (маршрут B).

---

### Task 15: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`,
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `mdmg.Ticker`, `mdmg.DefaultParams()` из Task 14.
- Produces: `ParamsFor("MDMG")` — его проверяет Task 18 через `cmd/pullparity`.

- [ ] **Step 1: Написать падающий тест** рядом с `TestRegistryHasRAGR` (`registry_test.go:~240`):

```go
// TestRegistryHasMDMG держит связку «пакет — реестр живого раннера» для MDMG: раннер обязан
// торговать ровно параметры пакета.
func TestRegistryHasMDMG(t *testing.T) {
	p, ok := ParamsFor(mdmg.Ticker)
	if !ok {
		t.Fatal("MDMG нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := mdmg.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не параметры пакета:\n got: %+v\nwant: %+v", p, want)
	}
}
```

  Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg"` добавить в тестовый
  файл. **Маршрут B:** дополнительно добавить запись `"MDMG": "<причина, дата>"` в карту
  `baselineByDesignTickers` (`registry_test.go:~86`, рядом с `"CNRU"` и `"TGKA"`) с комментарием в том
  же стиле: калибровка проведена, точка проиграла дефолтам по §5.15 (числа из Task 13).

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run MDMG -v`
  Expected: FAIL.

- [ ] **Step 3: Добавить в реестр.** Импорт
  `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg"` в алфавитном порядке блока
  и строка `mdmg.Ticker: mdmg.DefaultParams(),` после `ragr.Ticker: ragr.DefaultParams(),`
  (`registry.go:716`). `gofmt -w` выровняет столбцы.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
  Expected: PASS (включая сторож вселенной с `baselineByDesignTickers` на маршруте B).

- [ ] **Step 5: Коммит** `feat(rsi_pullback): MDMG в реестре живого раннера`.

---

### Task 16: Боевая вселенная

**Files:** Modify `internal/config/rsi_pullback.go:605`, `internal/config/rsi_pullback_test.go:56-58`,
`env/prod.env:20`, `env/prod.env.example:30`, `env/local.env.example:28`

Сторож `TestRSIPullbackTickersMatchEnvFiles` (`internal/config/rsi_pullback_test.go`) уже есть и
читает все три env-файла.

- [ ] **Step 1: Написать падающий тест.** В `TestNewRSIPullbackConfig_Defaults`
  (`rsi_pullback_test.go:54`) дописать `"MDMG"` последним в `want`, заменить `len(want) != 34` на
  `len(want) != 35` и сообщение `want 34: RAGR заведён тридцать четвёртым 2026-09-26` на
  `want 35: MDMG заведён тридцать пятым <дата исполнения>`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/config/ -run RSIPullback -v`
  Expected: FAIL в `TestNewRSIPullbackConfig_Defaults`.

- [ ] **Step 3: Добавить `MDMG` последним элементом во все четыре места сразу:**
  - `internal/config/rsi_pullback.go:605` — `"MDMG"` после `"RAGR"`;
  - `env/prod.env:20` — `,MDMG` в конец строки `RSI_PULLBACK_TICKERS=`;
  - `env/prod.env.example:30` — то же;
  - `env/local.env.example:28` — то же.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/config/ -v`
  Expected: PASS.

- [ ] **Step 5: Мутационная проверка сторожа.** Временно убрать `,MDMG` из `env/prod.env`, прогнать
  `go test ./internal/config/ -run RSIPullbackTickersMatchEnvFiles` — Expected: FAIL с именем
  `env/prod.env`. Восстановить строку вручную (`git checkout` не годится — правка ещё не
  закоммичена) и снова прогнать — PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): завести MDMG в боевую вселенную`.

---

### Task 17: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg.go`
(doc-comment)

- [ ] **Step 1: Написать разбор в doc-comment пакета.** Образец формы — маршрут A:
  `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr.go`; маршрут B:
  `.../strategy/cnru/cnru.go`. Обязательное содержимое §5.16 спеки:
  - пауза редомициляции 2024-05-22 → 2024-06-17, смена сессии 19 → 35 баров, окно 24 месяца после
    рестарта;
  - схема 24/12/3 и несравнимость с 36-месячным каталогом;
  - шаг цены 0.1 ₽ и реальный круг 0.015% против модельных 0.1%;
  - таблица сборки точки;
  - годы 2025–2026 (и огрызок 2024) и оба хвоста с долей крупнейшей сделки хвоста 6;
  - сверка знака хвоста 6 с фолдами 3–4;
  - таблица срабатываний стопа (§5.4), голос темы `risk` и цена урезания до 0.8;
  - гейт B в обеих формах;
  - третий контур против дефолтов;
  - гейты C и D (четыре отсечки из API);
  - два walk-forward, издержки 0.2%, фактический пул OOS и вырожденные фолды;
  - семь пунктов и планка;
  - **таблица «точка против дефолтов» из Task 13 и почему победил победитель**;
  - числа выходной сессии и часов 02–06 победителя (§8 №7);
  - принятые риски §8: короткое окно; режим окна — сильный рост 2025 года; растущие дивиденды;
    условие пересмотра после прода — хвост 6 месяцев с PF < 1.0 на ≥ 10 сделках, вывести из
    вселенной вне планового цикла.

  **Правило CLAUDE.md: `docs/rsi_pullback/` не трогается.**

- [ ] **Step 2: Линтер.** Run: `./bin/mage lint`. Expected: PASS.

- [ ] **Step 3: Коммит** `docs(rsi_pullback): разбор калибровки MDMG и принятый риск`.

---

### Task 18: Финальная проверка

**Files:** нет (только прогоны), кроме правок по найденному.

- [ ] **Step 1: Полный гейт.** Run: `./bin/mage ci`. Expected: PASS (lint + `go test -race ./...` +
  дрейф моков).

- [ ] **Step 2: Сверка живой сборки с бэктестом.**

```bash
go run ./cmd/pullparity -tickers MDMG -months 24
```

  Флаг — **`-tickers`** (множественное число). Без него команда молча возьмёт `UGLD,T,GAZP`.
  **`-months 24`, не больше** (`maxDailyHorizonMonths` в `cmd/pullparity/main.go`). Окно 24 месяца
  начинается 2024-09-26 — пауза 2024-05-22 → 2024-06-17 в него не попадает. Expected: ноль
  расхождений. Если расхождения есть — **остановиться и доложить владельцу** с датами расхождений,
  не подбирать окно.

- [ ] **Step 3: Сверить вселенную.** Run: `go test ./internal/config/ -run 'RSIPullbackTickersMatchEnvFiles|NewRSIPullbackConfig_Defaults' -v`.
  Expected: PASS, в списке 35 тикеров, MDMG последний.

- [ ] **Step 4: Доложить владельцу:**
  - победителя (точка или дефолты) и таблицу Task 13;
  - два walk-forward;
  - издержки 0.2%;
  - годы;
  - хвосты и сверку знака;
  - гейты A, B, C, D;
  - семь пунктов;
  - планку.

- [ ] **Step 5: Коммит** правок финальной проверки, если они были. Мерж делает владелец.

---

## Self-Review

**Покрытие спеки:**

| Раздел спеки | Задача |
|---|---|
| §1 инструмент, кэш, скользящий конец окна, дивиденды из API | Global Constraints; Task 1 Steps 5–7 |
| §2 отношение к каталогу, несравнимость | Task 2 Step 3 (`_comment`); Task 17 |
| §3 baseline, дефолты — кандидат | Global Constraints; Task 1 Steps 6–7; Task 13 |
| §4 окно 24/12/3, 24/15/3, ASTR, вырожденные фолды, обязательные флаги | Global Constraints; Task 3 Step 2; Task 11 Steps 1, 9 |
| §5.1 двенадцать тем, инварианты, отличия от RAGR | Task 2; задачи 3–10 |
| §5.2 сборка, `trend_low`, `vol_window`, арбитры | Task 7 Step 3; Task 10 Step 3; Task 11 Step 1 |
| §5.3 соседи плато, зонд за краем, годы и хвосты | Task 11 Step 7 |
| §5.4 гейт A (0.8) | Task 2 (константа); Task 8 Steps 3–4; Task 11 Step 2 |
| §5.5 гейт B и ничьи | Task 11 Steps 4, 7 |
| §5.6 третий контур, гейт C | Task 11 Steps 6, 8 |
| §5.7 гейт D (четыре отсечки) | Task 1 Step 5; Task 11 Step 5 |
| §5.8 скрипты журнала | Global Constraints; Task 1 Step 5 |
| §5.9 календарные годы, огрызок 2024 | Task 11 Step 11 |
| §5.10 хвосты и сверка знака с фолдами 3–4 | Task 11 Steps 12–13 |
| §5.11 схемы проверки | Task 11 Steps 9–10 |
| §5.12 планка | Task 4 Step 3; Task 5 Step 3; Task 11 Step 15 |
| §5.13 стоп-условие | Task 11 Step 14 |
| §5.14 второй круг, одно поле — одна тема, зона ±1 шаг, пул < 10 | Task 12 |
| §5.15 правило прода — точка против дефолтов | Task 1 Step 7; Task 13; задачи 14–16, 18 |
| §5.16 что записывается | Task 17; Task 13 Step 5 (при отказе) |
| §6 свойства бумаги | опорные замеры в задачах 3–10 и `_comment` сеток |
| §7 артефакты | задачи 1, 2, 14–16 |
| §8 риски | Task 17 Step 1 |

**Плейсхолдеры.** Угловые скобки остались только там, где исполнитель подставляет значения из
отчётов прогонов (`<тема>`, `<поле>`, `<значение>`, `<дата исполнения>`, `<причина, дата>`) и в
образце `_comment`, где явно сказано заполнить их текстом §6. Скелет литерала в Task 14 помечен как
пример формы.

**Согласованность имён:**
- `mdmgGridFiles`, `mdmgStopCeiling`, `mdmgCoreEMAFast`, `mdmgCoreEMASlow` — Task 2; расширяются в
  Task 12 через `mdmgAllGridFiles`, `mdmgRound2GridFiles`;
- `mdmg.Ticker`, `mdmg.DefaultParams()` — Task 1, используются в задачах 14–16;
- `TestRSIPullbackMDMGTracksBaseline` — Task 1, заменяется в Task 14 на
  `TestRSIPullbackMDMGIsRegisteredAndCalibrated` (маршрут A) или
  `TestRSIPullbackMDMGServesTheCoreBaselineByDesign` (маршрут B), либо получает окончательный
  комментарий в Task 13 (отказ);
- каталоги отчётов `MDMG_point`, `MDMG_point_wf123`, `MDMG_point_wf153`, `MDMG_point_c001` — Task 11,
  второй круг — `MDMG_point2*`; baseline — `MDMG_base`, `MDMG_base_wf123`, `MDMG_base_wf153`,
  `MDMG_base_c001`;
- номера отчётов совпадают с номерами задач (`task-N-report-mdmg.md`).

**Развилки:**
- Task 11 Step 18: точка прошла — Task 12 пропускается;
- Task 13 Step 4: точка / дефолты / отказ; при отказе задачи 14–17 не выполняются, Task 18 — только
  `mage ci` уже в Task 13 Step 5;
- Task 10 Step 3: при конфликте рычагов в точку идёт один из двух;
- Task 14: маршрут A или B.
