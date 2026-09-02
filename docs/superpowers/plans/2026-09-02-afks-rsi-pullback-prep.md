# AFKS под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести AFKS (АФК «Система») до вердикта по стратегии `rsi_pullback`: каталог
максимально широких сеток, тематический walk-forward, принятая точка с риск-гейтом стопа, литерал
в пакете и заведение в боевую вселенную двадцать вторым тикером.

**Architecture:** Процедура тем **каноническая**: все десять тем идут поверх дефолтов ядра,
поэтому каталог сеток создаётся целиком одной задачей, якоря нет, поздних файлов нет. Схема
прогонов **каноническая 36/12/6** (четыре фолда, проверено контрольным прогоном до написания
спеки), плюс **обязательный контрольный прогон принятой точки по схеме 24/12/3** — внутри окна
расширилась сессия (29 → 35 баров в буднем дне), поэтому точка проверяется ещё и на однородном
куске. Четыре отступления от канона в осях, каждое посажено на замер: `RSIPeriod` вверх до 10 и
вниз до 2, `RSIUpper` в теме `exit` вниз до 35, `VolMult` вверх до 4.0, `EMASlow` вниз до 20 (с
обрезкой `EMAFast` сверху до 10 — инвариант `EMAFast < EMASlow` не разрешает держать обе оси
широкими). Новое против всего каталога — **риск-гейт стопа**: принимается только уровень,
достижимый ≥ 30% будних дней окна.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-02-afks-rsi-pullback-prep-design.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех без исключения прогонах. Флаг `-interval Minutes30` обязателен
  в каждой команде: дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Схема прогонов каноническая:** `-months 36 -train-months 12 -test-months 6 -min-trades 20
  -metric profit_factor`, четыре фолда встык (12 + 6·4 = 36). У темы `screen` — `-min-trades 1`.
  Расчётное окно — **2023-09-02 … 2026-09-02**.
- **Число фолдов сверяется на первой же теме.** Контрольный прогон 2026-09-02 уже дал «Фолдов: 4»
  (pooled OOS PF 0.987 на 164 сделках), но сверка повторяется: если в шапке отчёта темы `screen`
  окажется «Фолдов: 3» — **остановиться и доложить владельцу**, не запускать остальные темы.
  Правило сборки «≥ 3 фолда из 4» при трёх фолдах вырождается в «3 из 3» и меняет планку задним
  числом.
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш дотянут штатным top-up 2026-09-02:
  `AFKS_Minutes30.json` — 37 125 баров (2023-08-04 … 2026-09-02), в окне 36 397, из них 25 640
  будних; `AFKS_Day1.json` — 1468 свечей (2021-04-16 … 2026-09-01).
- **Дыр в серии нет.** Разрывов длиннее четырёх дней ноль, скачков «открытие против вчерашнего
  закрытия» глубже 12% ноль. Ни один шаг не расширяет окно за 36 месяцев.
- **Сетки держатся максимально широкими** (решение владельца 2026-09-02). Обрезок осей не
  делается; замеры, которые в узком каталоге были бы основанием вырезать край, идут в `_comment`
  как **предупреждения** о том, чего ждать на краю. Жёсткие инварианты: `RSILower ≤ 50`,
  `RSIPeriod ≥ 2`, `RSIUpper > RSILower` на всех парах, ось тренда не порождает пар
  `EMAFast ≥ EMASlow`, `StopDailyATR` нигде не равен нулю.
- **Правило сборки точки:** поле берётся из темы, которая его меряет, и принимается только если за
  значение высказались **не менее трёх фолдов из четырёх**; иначе поле остаётся на дефолте ядра.
  Ничья 2/2 большинством не считается.
- **РИСК-ГЕЙТ СТОПА (решение владельца 2026-09-02, объявлен до прогонов).** Принимается только
  `StopDailyATR`, чей уровень достижим **не менее чем в 30% будних дней окна**. Таблица
  выживаемости AFKS (n = 762 будних дня): 0.3 — 99.7%, 0.5 — 91.7%, 0.7 — 71.0%, **1.0 — 38.6%**,
  1.3 — 19.4%, 1.5 — 11.4%, 2.0 — 2.8%. **Потолок — 1.0.** Если тема `risk` голосует выше, в точку
  идёт ближайшее значение оси внутри гейта, а цена решения в PF пишется прямым текстом в
  `_comment` и в доку пакета. Порог 30% после начала прогонов не двигается.
- **Второй контур риск-гейта — риск-профиль точки против baseline.** Опорные числа baseline: доля
  SL-выходов **25.4%**, удержание медиана **10** / p90 **21** / максимум **39** баров, ночёвок
  **47.6%**, выходов в выходную сессию **8.7%**. Точка меряется по всем четырём. Если падение доли
  SL-выходов идёт вместе с ростом удержания и ночёвок — это подпись капкана, и точка берёт более
  узкий стоп с записью цены решения.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` обе дают pooled
  OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для `entry`, `EMASlow` для
  `trend`) выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд (ни одной убыточной сделки) в
  пользу тикера не засчитывается; счёт по дефолтной комиссии (круг 0.1%).
- **Стоп-условие из ТРЁХ пунктов** (решение владельца 2026-09-02). Работа останавливается, если
  принятая точка даёт: (1) pooled OOS PF < 1.0 на 36/12/6, **либо** (2) меньше 20 сделок в пуле
  OOS, **либо** (3) pooled OOS PF < 1.0 на контрольном прогоне 24/12/3. При срабатывании — числа
  владельцу, задачи 12–16 не выполняются.
- **Издержки — предупреждение, не стоп.** Счёт под `-commission 0.001` и `0.0015` снимается и
  пишется в доку, но работу не останавливает: настоящий круг AFKS 0.016% (шаг 0.001 ₽ при цене
  12.755 ₽) против моделируемых 0.1%, удвоенная модель завышает факт в 12 раз и её проваливает уже
  сам baseline (1.100 → 0.904 → 0.735 → 0.593 при 0.1/0.2/0.3/0.4%).
- **Правило прода:** литерал ставится и AFKS заводится в `RSI_PULLBACK_TICKERS` двадцать вторым
  **независимо от того, взята планка или нет**. Стоп-условие это правило перевешивает.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают ВСЕ темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`,
  `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`,
  `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`,
  `UseTrail 0`, `TrailDailyATR 0`. Контрольный прогон дефолтов на расчётном окне: **126 сделок,
  PF 1.100, net +6 426.61 ₽**, win rate 66.67%, max DD 13.88%, выходы RSI 84 / SL 32 / TP 10 —
  **предпоследнее место** в ряду baseline каталога (YDEX 1.778, NKHP 1.697, LSNGP 1.554,
  ELFV 1.546, SNGSP 1.535, IVAT 1.432, LENT 1.417, SVAV 1.368, BANEP 1.336, BSPB 1.114,
  **AFKS 1.100**, ASTR 1.093, SIBN 1.027, NVTK 0.984).
- **Полугодия расчётного окна** (нужны в Task 11): 2023-09-02…2024-03-02 (инструмент +0.3%,
  baseline 15 сделок, PF 0.844, net −592), 2024-03-02…2024-09-02 (−17.5%, 25, 1.090, +1 163),
  2024-09-02…2025-03-02 (**+14.2%**, 25, **0.775**, −5 129), 2025-03-02…2025-09-02 (−6.9%, 19,
  1.956, +4 700), 2025-09-02…2026-03-02 (−15.4%, 20, 2.298, +5 913), 2026-03-02…2026-09-02
  (**−46.8%**, 22, 1.024, +371). **Два убыточных из шести**, и убыточно единственное растущее
  полугодие. Итог окна: инструмент −60.5%, максимальная просадка инструмента −77.1%.
- **Ликвидность — лучшая во вселенной, исполнительного риска нет.** Медиана оборота будних дней
  1225.9 млн ₽ (36 мес), 968.2 млн (12 мес); выходная сессия — 314 дней, медиана 30.09 млн ₽.
  Дневной ATR(14) медиана 4.09% цены.
- **Сессия расширилась внутри окна**: медиана баров в буднем дне 29 (2023–2024) → 35 (2025–2026).
  Отсюда контрольный прогон 24/12/3 и пункт 3 стоп-условия.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; полную команду запуска с путём
  `data/params/rsi_pullback/afks/<файл>` (этого требует `TestRSIPullbackCalFilesValid`: тест падает,
  если `_comment` не называет путь собственного файла); место под строку
  `РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается.** Правило CLAUDE.md: `strategy.md`,
  `live.md`, `screener.md` описывают механику. Пер-тикерный разбор живёт в доке пакета
  `strategy/afks` и в `_comment` файлов сеток.
- **Каталог `reports/` — в `.gitignore`.** Отчёты остаются локальными; в репозиторий едут строки
  `РЕЗУЛЬТАТ ПРОГОНА` в `_comment`, рабочая записка Task 11 и код. Ни один шаг не должен пытаться
  закоммитить `reports/`.
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/afks-pullback-prep` от `feat/nkhp-pullback-prep` (`5b8c088`); спека уже
  закоммичена (`d87cc71`). Замеры инструмента — `reports/_analysis/afks_pullback_prep_measurements.md`
  (вне git).

---

### Task 1: Каталог десяти сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/afks/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`, `cal_vol_window.json`, `cal_risk.json`,
  `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_afks_grid_test.go`

**Interfaces:**
- Consumes: формат `Phase` из `internal/service/backtest/calibrate.go` (`ParsePhases`); хелпер
  `rsiPullbackTickerGrid(t, ticker, file) map[string][]float64` из
  `internal/service/backtest/rsi_pullback_grid_test.go:40`.
- Produces: десять файлов сеток, которые читают Tasks 3–10, и сторожевой тест
  `TestAFKSGridsStayWide`, который держит их ширину.

- [ ] **Step 1: Написать падающий сторожевой тест осей**

Создать `internal/service/backtest/rsi_pullback_afks_grid_test.go`:

```go
package backtest

import "testing"

// TestAFKSGridsStayWide держит НИЖНЮЮ границу ширины сеток AFKS. Владелец потребовал 2026-09-02
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет
// наличие обязательных значений и не запрещает лишних. Четыре отступления от канона §8.1 посажены
// на замеры (reports/_analysis/afks_pullback_prep_measurements.md) и пиньятся здесь, чтобы
// редактор сеток не «починил» их обратно к канону:
//
//   - RSIPeriod ВВЕРХ до 10 и ВНИЗ до 2 (канон 3..8). Зона 7-8 — лучшая связная на оси
//     (1.403/66 и 1.388/56 против 1.100/126 у дефолта 4), десятка ещё держит 38 сделок и
//     отделяет «максимум внутри» от «максимум на краю». Нижний край стоит ради симметрии
//     проверки: двойка убыточна (0.984 на 328 сделках), её победа означала бы переоптимизацию.
//   - RSIUpper в cal_exit ВНИЗ до 35 при сохранённом верхе 85. Замер говорит, что низ оси мёртв
//     (35 -> 0.759/155, 40 -> 0.806/150), а максимум стоит ровно на дефолте 70 (1.100). Ось
//     расширена ради доказуемости середины: на NKHP тот же приём дал обратный ответ, там
//     максимум и оказался на 40.
//   - VolMult ВВЕРХ до 4.0: максимум множителя стоит на крае (4.0 -> 1.348/39 против
//     1.2 -> 1.194/103). Предупреждение о крае: 39 сделок за окно — это около 13 на обучающее
//     окно, ниже -min-trades 20, и порог, скорее всего, отсечёт край внутри фолдов.
//   - EMASlow в cal_trend ВНИЗ до 20 при EMAFast, обрезанной сверху до 10. Оба края оси EMASlow
//     выше середины (20 -> 1.633/108, 250 -> 1.214/138 против 100 -> 1.100/126), но полная
//     канонической ширины пара осей породила бы пары EMAFast >= EMASlow — перевёрнутый трендовый
//     фильтр, способный на падающем на 60% инструменте выиграть тему in-sample. Инвариант не
//     разрешает держать обе оси широкими одновременно.
func TestAFKSGridsStayWide(t *testing.T) {
	type want struct {
		file   string
		field  string
		values []float64
	}
	cases := []want{
		{"cal_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_entry.json", "RSILower", []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8, 10}},
		{"cal_entry.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_trend.json", "EMAFast", []float64{3, 5, 10}},
		{"cal_trend.json", "EMASlow", []float64{20, 30, 50, 70, 100, 150, 200, 250}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.2, 0.3, 0.4, 0.5, 0.6}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 4.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24}},
		{"cal_vol_window.json", "VolMult", []float64{1.2, 2.5, 4.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "afks", c.file)
		got := grid[c.field]
		for _, v := range c.values {
			var found bool
			for _, g := range got {
				if g == v {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("afks/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestAFKSEntryGridKeepsRSIUpperAboveRSILower пинит жёсткий инвариант связки входа и выхода: в
// теме entry обе оси свипуются вместе, и пара, где уровень выхода не выше уровня входа, описывает
// сделку, которая закрывается в момент открытия. Инвариант держится конструкцией осей
// (RSILower <= 50 < 55 <= RSIUpper), и тест ловит его нарушение при любой правке файла.
func TestAFKSEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "afks", "cal_entry.json")
	var maxLower float64
	minUpper := 1e9
	for _, v := range grid["RSILower"] {
		if v > maxLower {
			maxLower = v
		}
	}
	for _, v := range grid["RSIUpper"] {
		if v < minUpper {
			minUpper = v
		}
	}
	if minUpper <= maxLower {
		t.Fatalf("cal_entry.json порождает пару RSIUpper=%v <= RSILower=%v: такая сделка закрывается в момент открытия", minUpper, maxLower)
	}
}

// TestAFKSTrendGridKeepsFastBelowSlow пинит вторую половину решения об осях тренда: EMAFast
// обрезана сверху до 10 ИМЕННО ПОТОМУ, что EMASlow расширена вниз до 20. Любая пара, где быстрая
// EMA не медленнее медленной, превращает трендовый фильтр в перевёрнутый и на падающем
// инструменте способна выиграть тему in-sample.
func TestAFKSTrendGridKeepsFastBelowSlow(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "afks", "cal_trend.json")
	for _, fast := range grid["EMAFast"] {
		for _, slow := range grid["EMASlow"] {
			if fast >= slow {
				t.Fatalf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: перевёрнутый трендовый фильтр", fast, slow)
			}
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run 'TestAFKS' -v`
Expected: FAIL — каталога `data/params/rsi_pullback/afks/` ещё нет.

- [ ] **Step 3: Создать десять файлов сеток**

Каждый файл — `{"_comment": "...", "phases": [{"name": "<тема>", "grid": {...}, "keepTop": 5}]}`.
`_comment` обязан называть **собственный** путь (`afks/<файл>`) в команде запуска — иначе падает
`TestRSIPullbackCalFilesValid`.

`cal_screen.json` — 4 прогона:

```json
{
  "phases": [{"name": "screen", "grid": {"UseDayATRGate": [0, 1], "UseVolume": [0, 1]}, "keepTop": 4}]
}
```

`_comment` темы `screen`: цена гейтов на точечном замере — дневной гейт включён 1.100/126,
выключен **0.991/474** (гейт несёт edge, в отличие от NKHP, где без него оставалось 1.440);
объёмный гейт на дефолтной форме 1.194/103 против 1.100/126. Команда:
`go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/afks/cal_screen.json -out ./reports/AFKS_screen -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor`.

`cal_entry.json` — 504 прогона, `"RSIUpper": [55,60,65,70,75,80,85]`,
`"RSIPeriod": [2,3,4,5,6,7,8,10]`, `"RSILower": [10,15,20,25,30,35,40,45,50]`.

`cal_trend.json` — 24 прогона, `"EMAFast": [3,5,10]`, `"EMASlow": [20,30,50,70,100,150,200,250]`.

`cal_day.json` — 54 прогона, `"FreshDayATR": [0,0.2,0.3,0.4,0.5,0.6]`,
`"SpentDayATR": [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5]`.

`cal_day_spent.json` — 9 прогонов, `"FreshDayATR": [0]`, `"SpentDayATR"` — та же ось.

`cal_volume.json` — 35 прогонов, `"UseVolume": [1]`, `"VolMult": [1.0,1.2,1.5,2.0,2.5,3.0,4.0]`,
`"VolBaseDays": [3,5,10,14,20]`.

`cal_vol_window.json` — 24 прогона, `"UseVolume": [1]`,
`"VolLookbackBars": [1,2,3,5,8,12,16,24]`, `"VolMult": [1.2,2.5,4.0]`.

`cal_risk.json` — 63 прогона, `"StopDailyATR": [0.3,0.5,0.7,1.0,1.3,1.5,2.0]`,
`"TPDailyATR": [0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.0,2.5]`. Строка `TPDailyATR 2.5` стоит как контроль
асимметрии, которого требует `TestRSIPullbackGridControlPoints` от файла, свипующего стоп до 2.0.
`_comment` обязан содержать **таблицу выживаемости стопа и объявление риск-гейта**: 0.3 — 99.7%,
0.5 — 91.7%, 0.7 — 71.0%, 1.0 — 38.6%, 1.3 — 19.4%, 1.5 — 11.4%, 2.0 — 2.8%; потолок принятого
значения — 1.0; верх оси стоит как контроль капкана, а не как кандидат.

`cal_exit.json` — 11 прогонов, `"RSIUpper": [35,40,45,50,55,60,65,70,75,80,85]`.

`cal_trail.json` — 10 прогонов, `"UseRSIExit": [0,1]`, `"UseTrail": [1]`,
`"TrailDailyATR": [0.3,0.5,0.7,1.0,1.5]`.

Замеры для `_comment` каждой темы — в `reports/_analysis/afks_pullback_prep_measurements.md`,
раздел «Рельефы осей»; переносить числа оттуда, а не пересчитывать.

- [ ] **Step 4: Запустить тесты сеток**

Run: `go test ./internal/service/backtest/ -run 'RSIPullback|TestAFKS' -v`
Expected: PASS — включая `TestRSIPullbackCalFilesValid`, `TestRSIPullbackGridControlPoints`,
`TestAFKSGridsStayWide`, `TestAFKSEntryGridKeepsRSIUpperAboveRSILower`,
`TestAFKSTrendGridKeepsFastBelowSlow`.

Примечание: `TestRSIPullbackPlateauFilesArePoints` требует, чтобы в поставке был хотя бы один
`plateau_*.json` — он уже есть у других тикеров, поэтому на этом шаге тест зелёный и без файлов
AFKS.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/afks internal/service/backtest/rsi_pullback_afks_grid_test.go
git commit -m "feat(rsi_pullback): каталог широких сеток AFKS и сторожевой тест осей"
```

---

### Task 2: Пакет `strategy/afks` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/afks/afks.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/afks/afks_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `afks.Ticker` (строка `"AFKS"`) и `afks.DefaultParams() core.Params` — их используют
  Task 12 (литерал) и Task 13 (реестр живого раннера).

**Внимание:** в репозитории уже есть два пакета с именем `afks` —
`internal/service/trading_strategy/reversion/strategy/afks` и
`.../scalping/strategy/afks`. Новый пакет живёт по третьему пути и в реестре бэктеста
импортируется под алиасом `rsipullbackafks`, как остальные пакеты `rsi_pullback` в
`rsi_pullback_registry.go`.

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/afks/afks_test.go`:

```go
package afks

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка AFKS ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 12, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("AFKS ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsAFKS(t *testing.T) {
	if Ticker != "AFKS" {
		t.Fatalf("Ticker = %q, want AFKS", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/afks/ -v`
Expected: FAIL — пакета не существует.

- [ ] **Step 3: Создать пакет**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/afks/afks.go` с
док-комментарием, который переносит априор и рамки из спеки:

```go
// Package afks supplies the ticker and rsi_pullback Params for AFKS (АФК «Система», обыкновенные
// акции, лот 100).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. Пакет возвращает core.DefaultParams() — baseline ядра, не
// подобранный под этот инструмент. Так и должно быть до конца калибровки: пакет заведён заранее,
// чтобы прогоны шли через тот же реестр, что и у остальных двадцати одного тикера, а не через
// generic-ветку. Состояние держит afks_test.go.
//
// ОКНО И СХЕМА КАНОНИЧЕСКИЕ. Расчётное окно — 2023-09-02 … 2026-09-02 (36 месяцев); в окне 36 397
// получасовых баров, из них 25 640 будних, 780 будних дней и 314 дней выходных сессий. Схема
// -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor, четыре фолда
// встык (12 + 6*4 = 36); число фолдов проверено контрольным прогоном ДО раскладки сеток. Дыр в
// серии нет: разрывов длиннее четырёх дней ноль, скачков «открытие против вчерашнего закрытия»
// глубже 12% ноль. -refresh во время калибровки НЕ запускать.
//
// СЕССИЯ РАСШИРИЛАСЬ ВНУТРИ ОКНА: медиана баров в буднем дне 29 (2023-2024) -> 35 (2025-2026).
// Поэтому кроме канонической схемы 36/12/6 точка обязана пройти контрольный прогон 24/12/3 на
// окне 2024-09-02 … 2026-09-02. Точка с pooled OOS PF ниже 1.0 на этом прогоне вердикта не
// получает — это пункт 3 стоп-условия.
//
// СЕТКИ ЭТОГО ТИКЕРА ДЕРЖАТСЯ МАКСИМАЛЬНО ШИРОКИМИ — решение владельца 2026-09-02. Обрезок осей
// нет; замеры, которые в узком каталоге были бы основанием вырезать край, живут в _comment сеток
// как предупреждения. Сторожевой тест TestAFKSGridsStayWide запрещает УРЕЗАТЬ ось, а не расширять
// её.
//
// РИСК-ГЕЙТ СТОПА — второе решение владельца того же дня, объявленное ДО прогонов: принимается
// только StopDailyATR, чей уровень достижим не менее чем в 30% будних дней окна. Таблица
// выживаемости (n = 762): 0.3 — 99.7%, 0.5 — 91.7%, 0.7 — 71.0%, 1.0 — 38.6%, 1.3 — 19.4%,
// 1.5 — 11.4%, 2.0 — 2.8%. Потолок принятого значения — 1.0. Причина: на точечном замере ось
// стопа даёт 2.0 -> 1.700 при НЕИЗМЕННЫХ 122 сделках, то есть убыток не исчезает, а утекает в
// RSI-выход; стоп в 2.0 дневных ATR это около 8.2% цены и защитой быть перестаёт.
//
// АПРИОР, записанный ДО прогонов, СЛАБЫЙ ПО ОБОИМ ЗАМЕРАМ. Контрольный прогон дефолтов ядра на
// расчётном окне: 126 сделок, PF 1.100, net +6 426.61 руб., win rate 66.67%, max DD 13.88%,
// выходы RSI 84 / SL 32 (25.4%) / TP 10 — ПРЕДПОСЛЕДНЕЕ место в ряду baseline каталога
// (YDEX 1.778, NKHP 1.697, LSNGP 1.554, ELFV 1.546, SNGSP 1.535, IVAT 1.432, LENT 1.417,
// SVAV 1.368, BANEP 1.336, BSPB 1.114, AFKS 1.100, ASTR 1.093, SIBN 1.027, NVTK 0.984).
// Априорный контрольный walk-forward темы screen: pooled OOS PF 0.987 на 164 сделках — свободный
// выбор гейтов вне in-sample даёт безубыток. Веса априору не придаётся: BSPB (1.114) и ASTR
// (1.093) с таким же априором дали принятые точки 2.336 и 2.740.
//
// ЧЕМ ЭТА БУМАГА ОТЛИЧАЕТСЯ ОТ ОСТАЛЬНОГО КАТАЛОГА: она самая ликвидная (медиана оборота будних
// дней 1226 млн руб., выходная сессия 30 млн) и самая дешёвая в исполнении (шаг 0.001 руб. при
// цене 12.755 руб. — круг 0.016% против моделируемых 0.1%, модель консервативна ВШЕСТЕРО). Это
// обратный полюс NKHP, и по этой причине пункт стоп-условия про удвоенные издержки понижен до
// предупреждения: он завышает факт в 12 раз и его проваливает уже сам baseline (0.904).
package afks

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the MOEX ticker this package parameterizes.
const Ticker = "AFKS"

// DefaultParams returns the rsi_pullback parameters for AFKS. Пока калибровка не проведена, это
// ровно baseline ядра — см. док-комментарий пакета.
func DefaultParams() core.Params { return core.DefaultParams() }
```

- [ ] **Step 4: Запустить тест и убедиться, что он проходит**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/afks/ -v`
Expected: PASS.

- [ ] **Step 5: Завести тикер в реестр бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт
`rsipullbackafks "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/afks"` (алфавитно
первым в блоке `rsipullback*`) и запись в `rsiPullbackRegistry`:

```go
	rsipullbackafks.Ticker: rsiPullbackBindingFor(rsipullbackafks.Ticker, rsipullbackafks.DefaultParams),
```

Если `internal/service/backtest/rsi_pullback_registry_test.go` содержит список ожидаемых тикеров —
добавить в него `"AFKS"`.

- [ ] **Step 6: Запустить тесты реестра**

Run: `go test ./internal/service/backtest/ -run 'RSIPullback' -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/afks internal/service/backtest
git commit -m "feat(rsi_pullback): пакет AFKS и реестр бэктеста"
```

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_screen.json` (строка результата в `_comment`)
- Отчёты: `reports/AFKS_screen/`

**Interfaces:**
- Consumes: `cal_screen.json` из Task 1.
- Produces: пофолдовых победителей `UseDayATRGate` и `UseVolume` для Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_screen.json \
  -out ./reports/AFKS_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Проверить число фолдов**

В шапке `*_walkforward.md` обязано стоять «Фолдов: 4». Если «Фолдов: 3» — **остановиться и
доложить владельцу**, остальные темы не запускать.

- [ ] **Step 3: Выписать результат**

Pooled OOS PF, число сделок пула, пофолдовые значения `UseDayATRGate` и `UseVolume`, пофолдовые
OOS PF / сделки / MaxDD%. Ожидание из точечного замера: `UseDayATRGate=1` единогласно (без гейта
0.991/474 — убыток), по `UseVolume` согласия может не быть (гейт даёт всего +0.09 PF).

- [ ] **Step 4: Дописать строку результата в `_comment`**

Формат: `РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: pooled OOS PF X.XXX на N сделках, фолдов 4, отчёт
reports/AFKS_screen/<файл>. Пофолдово: UseDayATRGate a,b,c,d; UseVolume a,b,c,d. …` плюс вывод: что
принято темой, что уходит в дефолт.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/afks/cal_screen.json
git commit -m "feat(rsi_pullback): AFKS, тема screen — цена гейтов"
```

---

### Task 4: Тема `entry` — ключевая, ось периода расширена вверх до 10 и вниз до 2

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_entry.json`
- Отчёты: `reports/AFKS_entry/`

**Interfaces:**
- Consumes: `cal_entry.json` из Task 1.
- Produces: пофолдовых победителей `RSILower` (ведущая ось планки), `RSIPeriod` и контекстного
  `RSIUpper` для Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_entry.json \
  -out ./reports/AFKS_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить планку по критериям A и B**

Критерий A: pooled OOS PF ≥ 1.5 при ≥ 20 сделках в пуле. Критерий B: `RSILower` выбран одинаково в
≥ 3 фолдах из 4. Оба записываются отдельно — «взят»/«провален», без сглаживания.

- [ ] **Step 3: Проверить края расширенной оси периода**

Отдельно записать, победил ли `RSIPeriod` 10 (верхний край) или 2 (нижний) хоть в одном фолде.
Победа 2 при точечном замере 0.984/328 — сигнал переоптимизации обучающего окна, пишется прямым
текстом. Победа 10 означает, что ось стоит на краю и её надо было тянуть дальше — тоже пишется.

- [ ] **Step 4: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/afks/cal_entry.json
git commit -m "feat(rsi_pullback): AFKS, тема entry — ключевая ось входа"
```

---

### Task 5: Тема `trend` — вторая ключевая, EMASlow расширена вниз, EMAFast обрезана инвариантом

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_trend.json`
- Отчёты: `reports/AFKS_trend/`

**Interfaces:**
- Consumes: `cal_trend.json` из Task 1.
- Produces: пофолдовых победителей `EMASlow` (ведущая ось планки) и `EMAFast` для Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_trend.json \
  -out ./reports/AFKS_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить планку и прочитать форму победы**

Критерии те же, ведущая ось — `EMASlow`. Отдельно записать: если побеждает край 20, это **выбор
режима, а не тренда** — на инструменте, падающем на 60%, короткая медленная EMA превращает фильтр
в моментум-фильтр. Если побеждает верхний край обрезанной оси `EMAFast` (10) — ось поставлена не
там, и это пишется прямым текстом, а не сглаживается.

- [ ] **Step 3: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/afks/cal_trend.json
git commit -m "feat(rsi_pullback): AFKS, тема trend — вторая ключевая ось"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт, носитель edge

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_day.json`, `cal_day_spent.json`
- Отчёты: `reports/AFKS_day/`, `reports/AFKS_day_spent/`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: пофолдовых победителей `FreshDayATR` и `SpentDayATR` для Task 11 (`day` — основная
  тема, `day_spent` — контекст).

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_day.json \
  -out ./reports/AFKS_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_day_spent.json \
  -out ./reports/AFKS_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сравнить темы между собой**

Расхождение победителей `SpentDayATR` в двух темах означает **взаимодействие веток**: открытая
свежая ветка сдвигает оптимум исчерпанной. Записать прямым текстом, в скольких фолдах темы
разошлись.

- [ ] **Step 3: Прочитать результат против точечного рельефа**

Опорные числа: исчерпанная ветка держит максимум ровно на дефолте 0.8 (1.100/126) и обваливается
по обе стороны (0.7 → 1.078/164, 0.9 → 0.930/84, 1.0 → 0.835/60); всплеск 1.5 → 1.198 стоит на 16
сделках и порогом `-min-trades 20` отсекается. Свежая ветка: 0.4 → 1.160/280 — лучший net (+22 178)
при PF на уровне дефолта.

- [ ] **Step 4: Дописать строки результата в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/afks/cal_day.json data/params/rsi_pullback/afks/cal_day_spent.json
git commit -m "feat(rsi_pullback): AFKS, темы day и day_spent"
```

---

### Task 7: Тема `volume` — гейт на самой ликвидной бумаге вселенной

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_volume.json`
- Отчёты: `reports/AFKS_volume/`

**Interfaces:**
- Consumes: `cal_volume.json` из Task 1.
- Produces: пофолдовых победителей `VolMult` и `VolBaseDays` для Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_volume.json \
  -out ./reports/AFKS_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить край оси `VolMult`**

Точечный максимум стоит на расширенном крае 4.0 (1.348/39). Записать: отсёк ли порог
`-min-trades 20` этот край внутри фолдов (39 сделок за окно — это ≈13 на обучающее окно). Если
край всё же победил — записать число сделок фолда прямым текстом, потому что редкий вход на
ликвиднейшей бумаге вселенной означает подгонку, а не отбор.

- [ ] **Step 3: Сопоставить с NKHP**

NKHP — единственный тикер каталога, где объёмный гейт нёс сигнал (+0.33 PF, единогласный выбор
всех четырёх фолдов), и объяснением была самая тонкая ликвидность вселенной. AFKS — обратный
полюс (1226 млн ₽/день против 14.4 млн), и дефолтная форма гейта даёт лишь +0.09 PF. Ответ темы
на AFKS — прямая проверка этого объяснения; записать, подтвердилось оно или нет.

- [ ] **Step 4: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/afks/cal_volume.json
git commit -m "feat(rsi_pullback): AFKS, тема volume"
```

---

### Task 8: Тема `vol_window` — окно объёмного гейта

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_vol_window.json`
- Отчёты: `reports/AFKS_vol_window/`

**Interfaces:**
- Consumes: `cal_vol_window.json` из Task 1.
- Produces: пофолдовых победителей `VolLookbackBars` (и контекстного `VolMult`) для Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_vol_window.json \
  -out ./reports/AFKS_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать исход гипотезы про окно**

Каталожная гипотеза «чем ликвиднее бумага, тем ближе оптимум `VolLookbackBars` к дефолту» имеет
три известных исхода: узкий край (ожидание), широкий край (HEAD), отсутствие устойчивого оптимума
вовсе (NKHP). Точечный замер AFKS даёт максимум на узком крае (2 → 1.227/95) при монотонном спаде
к 24 → 1.105/125. AFKS — самая ликвидная бумага каталога, то есть прямая проверка гипотезы на
верхнем конце шкалы ликвидности. Записать исход.

- [ ] **Step 3: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/afks/cal_vol_window.json
git commit -m "feat(rsi_pullback): AFKS, тема vol_window"
```

---

### Task 9: Тема `risk` — стоп, цель и обязательная проверка риск-гейта

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_risk.json`
- Отчёты: `reports/AFKS_risk/`, `reports/AFKS_risk_probe_stop<значение>/`

**Interfaces:**
- Consumes: `cal_risk.json` из Task 1.
- Produces: пофолдовых победителей `StopDailyATR` и `TPDailyATR` для Task 11 **и результат
  проверки риск-гейта** — принятое значение стопа плюс цена решения в PF.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_risk.json \
  -out ./reports/AFKS_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Применить риск-гейт к победителю темы**

Победитель проверяется по таблице выживаемости (доля будних дней окна, чей размах достаёт уровня,
n = 762): 0.3 — 99.7%, 0.5 — 91.7%, 0.7 — 71.0%, **1.0 — 38.6%**, 1.3 — 19.4%, 1.5 — 11.4%,
2.0 — 2.8%. Порог 30%, потолок принятого значения — **1.0**.

Если тема выбрала значение выше потолка (это ожидаемо: на точечном замере 2.0 даёт 1.700 при
неизменных 122 сделках), в точку идёт **1.0**, а разница PF между выбором темы и принятым
значением записывается прямым текстом как цена решения.

- [ ] **Step 3: Снять зонд принятого и отвергнутого стопа на полной истории**

Для принятого значения и для победителя темы (если они разные) прогнать одиночные конфигурации на
полном окне и сравнить анатомию:

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/plateau_stop_<значение>.json \
  -out ./reports/AFKS_risk_probe_stop<значение> -months 36
```

(файлы `plateau_stop_*.json` создаются здесь же, по одному значению на ключ — этого требует
`TestRSIPullbackPlateauFilesArePoints`.)

Сравнить с опорными числами baseline: доля SL-выходов 25.4%, удержание медиана 10 / p90 21 /
максимум 39, ночёвок 47.6%, выходов в выходную сессию 8.7%. **Подпись капкана — падение доли
SL-выходов вместе с ростом удержания и ночёвок**; если она есть, это фиксируется как
подтверждённая на самой бумаге, а не как теоретическое опасение.

- [ ] **Step 4: Дописать строку результата в `_comment` и закоммитить**

`_comment` обязан содержать: победителя темы по фолдам, применение риск-гейта, принятое значение,
цену решения в PF и анатомию обоих зондов.

```bash
git add data/params/rsi_pullback/afks/cal_risk.json data/params/rsi_pullback/afks/plateau_stop_*.json
git commit -m "feat(rsi_pullback): AFKS, тема risk и проверка риск-гейта стопа"
```

---

### Task 10: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/afks/cal_exit.json`, `cal_trail.json`
- Отчёты: `reports/AFKS_exit/`, `reports/AFKS_trail/`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: пофолдовых победителей `RSIUpper` (изолированно), `UseRSIExit`, `TrailDailyATR` для
  Task 11.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_exit.json \
  -out ./reports/AFKS_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/cal_trail.json \
  -out ./reports/AFKS_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Прочитать тему `exit` против расширенного края**

Точечный рельеф монотонно растёт до дефолта 70 (1.100), вся нижняя половина оси убыточна
(35 → 0.759, 40 → 0.806, 45 → 0.955). Победа значения ниже 55 в каком-либо фолде означает, что
фолд выбрал заведомо убыточную на полном окне зону, — записать прямым текстом. Верхний край
85 → 1.115 покупает +0.015 PF просадкой 25 567 против 15 055 у дефолта; при его победе это тоже
записывается.

- [ ] **Step 3: Применить правило упрощения трейла**

Прецедент ASTR: если тема `trail` выбрала дистанцию, на которой трейл не срабатывает (на AFKS это
1.0 и выше — там результат побайтово равен baseline), в точку идут `UseTrail 0` и
`TrailDailyATR 0`. Если тема выбрала `UseRSIExit 0` — сверить с точечным замером: замена
RSI-выхода трейлом 0.7 покупает +0.09 PF просадкой в 1.7 раза (25 975 против 15 055), и это идёт
на второй контур риск-гейта в Task 11.

- [ ] **Step 4: Дописать строки результата в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/afks/cal_exit.json data/params/rsi_pullback/afks/cal_trail.json
git commit -m "feat(rsi_pullback): AFKS, темы exit и trail"
```

---

### Task 11: Сборка точки, риск-гейт, два walk-forward и три пункта стоп-условия

**Files:**
- Create: `data/params/rsi_pullback/afks/plateau_point.json`
- Create: `data/params/rsi_pullback/afks/plateau_<поле>_<значение>.json` — соседи плато (Step 7)
- Create: `docs/superpowers/plans/task-11-report-afks.md` — рабочая записка со всеми выкладками
- Отчёты: `reports/AFKS_point_oos/`, `reports/AFKS_point_24/`, `reports/AFKS_point_comm/`,
  `reports/AFKS_point_comm3/`, `reports/AFKS_point_full/`, `reports/AFKS_plateau_*/`

**Interfaces:**
- Consumes: пофолдовых победителей всех десяти тем (Tasks 3–10).
- Produces: принятую точку `core.Params` — её литерал ставит Task 12.

- [ ] **Step 1: Собрать точку по правилу «≥ 3 фолда из 4»**

Поле за полем: `RSIPeriod`, `RSILower` — тема `entry`; `RSIUpper` — тема `exit` (изолированный
замер, не связка); `EMAFast`, `EMASlow` — тема `trend`; `DailyATRPeriod` — нигде не свипуется,
остаётся 14; `UseDayATRGate`, `UseVolume` — тема `screen`; `FreshDayATR`, `SpentDayATR` — тема
`day` (`day_spent` идёт контекстом); `VolMult`, `VolBaseDays` — тема `volume`; `VolLookbackBars` —
тема `vol_window`; `StopDailyATR`, `TPDailyATR` — тема `risk`; `UseRSIExit`, `UseTrail`,
`TrailDailyATR` — тема `trail` с правилом упрощения.

Значение принимается только при **≥ 3 голосах из 4**; иначе поле падает на дефолт ядра. Ничья 2/2
большинством не считается. Для каждого поля, упавшего на дефолт, записать причину и отметить
отдельно, **случайно ли** принятое значение совпало с дефолтом.

- [ ] **Step 2: Применить риск-гейт стопа к собранной точке**

`StopDailyATR` точки обязан лежать внутри гейта (уровень достижим ≥ 30% будних дней; потолок 1.0).
Если голос темы вывел поле выше потолка — заменить на ближайшее значение оси внутри гейта и
записать цену решения в PF. Замена делается **до** прогонов точки, а не после: гейт объявлен до
начала работы и не подстраивается под результат.

- [ ] **Step 3: Записать `plateau_point.json`**

Файл обязан пиньить **ровно одно значение на каждый ключ** (`TestRSIPullbackPlateauFilesArePoints`).
`_comment` содержит: полный разбор поля за полем (голоса фолдов → решение), применение риск-гейта
и его цену, оговорку о подглядывании (точку собирает исполнитель, видевший всю историю тем, поэтому
её числа нельзя сравнивать с pooled OOS самих тем как замену процедуре отбора), обе команды
прогонов.

- [ ] **Step 4: Прогнать точку канонической схемой 36/12/6**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/plateau_point.json \
  -out ./reports/AFKS_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Выписать pooled OOS PF, число сделок пула, пофолдовые OOS PF / сделки / NetPnL% / MaxDD%. Фолд с
MaxDD 0.00% при запредельном PF — **вырожденный**, отмечается прямым текстом.

**Проверить пункты 1 и 2 стоп-условия:** pooled OOS PF ≥ 1.0 и ≥ 20 сделок в пуле. При провале
любого — зафиксировать числа, остановить работу, доложить владельцу; Steps 5–9 и Tasks 12–16 не
выполняются.

- [ ] **Step 5: Прогнать контрольную схему 24/12/3 — пункт 3 стоп-условия**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/plateau_point.json \
  -out ./reports/AFKS_point_24 \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

Сверить шапку: «Фолдов: 4». Pooled OOS PF ≥ 1.0 — иначе стоп-условие сработало, работа
останавливается. Опорное число baseline на 24 месяцах: PF 1.205 на 85 сделках.

- [ ] **Step 6: Снять счёт под удвоенными и тройными издержками — ПРЕДУПРЕЖДЕНИЕ, не стоп**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/plateau_point.json \
  -out ./reports/AFKS_point_comm -commission 0.001 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/plateau_point.json \
  -out ./reports/AFKS_point_comm3 -commission 0.0015 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Числа записываются в записку и в доку пакета вместе с опорными потерями baseline: 1.100 → 0.904
(−17.8%) → 0.735 (−33.2%). Работу они **не останавливают** (решение владельца 2026-09-02:
настоящий круг 0.016%, модель завышает факт в 12 раз, baseline проваливает уже удвоенную строку).
Если точка теряет заметно большую долю, чем baseline, это записывается прямым текстом как
свойство конфигурации.

- [ ] **Step 7: Снять соседей плато**

Для **каждого поля, принятого темой** (не упавшего на дефолт), прогнать по схеме 36/12/6 файл
`plateau_<поле>_<значение>.json`, отличающийся от точки одним значением этого поля — по соседу с
каждой стороны, где ось это позволяет. Ось, чей сосед даёт результат, побайтово равный точке, —
**инертна**, и это записывается: значение поля тогда ничего не решает.

- [ ] **Step 8: Прогнать точку на полной истории и разложить по полугодиям**

```bash
go run ./cmd/backtest -ticker AFKS -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/afks/plateau_point.json \
  -out ./reports/AFKS_point_full -months 36
```

Из журнала сделок посчитать по шести полугодиям (границы — в Global Constraints) число сделок, PF
и net. **Точка, чей net держится одним полугодием, вердикта не получает** — требование спеки;
опорная концентрация baseline: полугодия 4 и 5 дают +10 613 при итоговых +6 427.

Там же снять анатомию точки: доля SL-выходов, удержание (медиана / p90 / максимум), доля ночёвок,
доля выходов в выходную сессию — и сравнить с опорными 25.4% / 10 / 21 / 39 / 47.6% / 8.7%. Это
второй контур риск-гейта: падение доли SL-выходов вместе с ростом удержания и ночёвок — подпись
капкана, и при ней точка берёт более узкий стоп с записью цены решения.

- [ ] **Step 9: Написать рабочую записку и закоммитить**

`docs/superpowers/plans/task-11-report-afks.md` — все выкладки: сборка поля за полем, применение
риск-гейта, обе таблицы фолдов, три строки издержек, соседи плато, полугодия, анатомия, вердикт по
планке и по каждому пункту стоп-условия.

```bash
git add data/params/rsi_pullback/afks docs/superpowers/plans/task-11-report-afks.md
git commit -m "feat(rsi_pullback): AFKS, принятая точка и её замеры"
```

---

### Task 12: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/afks/afks.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/afks/afks_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (если в нём есть список
  тикеров, отслеживающих baseline)

**Interfaces:**
- Consumes: принятую точку из Task 11.
- Produces: `afks.DefaultParams()`, возвращающий литерал точки — его читают Tasks 13 и 14.

- [ ] **Step 1: Заменить тест отслеживания baseline на снимок литерала**

В `afks_test.go` удалить `TestParamsTrackTheBaselineUntilCalibrated` и написать вместо него снимок
принятой точки (значения подставить из Task 11):

```go
// TestParamsAreTheAcceptedPoint пинит принятую точку целиком. Снимок стоит здесь, а не в
// комментарии, потому что любое поле, изменённое мимо калибровки, обязано валить тест: параметры
// этого пакета уходят в боевую вселенную живого раннера.
func TestParamsAreTheAcceptedPoint(t *testing.T) {
	want := core.Params{
		RSIPeriod:       0, // <- подставить принятые значения из Task 11
		RSILower:        0,
		RSIUpper:        0,
		EMAFast:         0,
		EMASlow:         0,
		DailyATRPeriod:  14,
		UseDayATRGate:   0,
		FreshDayATR:     0,
		SpentDayATR:     0,
		StopDailyATR:    0,
		TPDailyATR:      0,
		UseVolume:       0,
		VolBaseDays:     0,
		VolLookbackBars: 0,
		VolMult:         0,
		UseRSIExit:      0,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки AFKS разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}
```

Добавить отдельным тестом **пин риск-гейта** — он обязан пережить любую будущую правку литерала:

```go
// TestAcceptedStopStaysInsideTheRiskGate пинит решение владельца 2026-09-02: принятый стоп обязан
// быть уровнем, до которого цена реально доходит. Таблица выживаемости на расчётном окне
// (n = 762 будних дня): 0.3 — 99.7%, 0.5 — 91.7%, 0.7 — 71.0%, 1.0 — 38.6%, 1.3 — 19.4%,
// 1.5 — 11.4%, 2.0 — 2.8%. Порог 30% даёт потолок 1.0. Тест ловит попытку «улучшить» PF
// расширением стопа мимо процедуры.
func TestAcceptedStopStaysInsideTheRiskGate(t *testing.T) {
	const riskGateCeiling = 1.0
	if got := DefaultParams().StopDailyATR; got > riskGateCeiling {
		t.Fatalf("StopDailyATR = %v выше потолка риск-гейта %v: уровень достижим менее чем в 30%% будних дней и защитой не является", got, riskGateCeiling)
	}
	if got := DefaultParams().StopDailyATR; got <= 0 {
		t.Fatalf("StopDailyATR = %v: многодневная позиция без стопа не является принятой конфигурацией", got)
	}
}
```

- [ ] **Step 2: Запустить тесты и убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/afks/ -v`
Expected: FAIL — пакет всё ещё возвращает `core.DefaultParams()`.

- [ ] **Step 3: Поставить литерал**

В `afks.go` заменить тело `DefaultParams()` на литерал принятой точки:

```go
// DefaultParams returns the accepted rsi_pullback point for AFKS (см. док-комментарий пакета:
// вердикт, разбор поля за полем, риск-гейт и принятые риски).
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       0, // <- подставить принятые значения из Task 11
		RSILower:        0,
		RSIUpper:        0,
		EMAFast:         0,
		EMASlow:         0,
		DailyATRPeriod:  14,
		UseDayATRGate:   0,
		FreshDayATR:     0,
		SpentDayATR:     0,
		StopDailyATR:    0,
		TPDailyATR:      0,
		UseVolume:       0,
		VolBaseDays:     0,
		VolLookbackBars: 0,
		VolMult:         0,
		UseRSIExit:      0,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
}
```

- [ ] **Step 4: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -v`
Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/afks internal/service/backtest
git commit -m "feat(rsi_pullback): AFKS откалиброван — литерал точки и снимок"
```

---

### Task 13: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (если в нём есть
  список ожидаемых тикеров)

**Interfaces:**
- Consumes: `afks.Ticker`, `afks.DefaultParams()` из Task 12.
- Produces: запись в `paramsByTicker` — её читает `ParamsFor` живого раннера и `cmd/pullparity`.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/afks"` и строка в
`paramsByTicker` следом за `nkhp.Ticker`:

```go
	afks.Ticker:  afks.DefaultParams(),
```

- [ ] **Step 2: Дописать абзац комментария карты**

Перед картой стоят абзацы про каждый заведённый тикер, на английском. Дописать абзац про AFKS в
том же стиле: каноническая схема 36/12/6 плюс контрольный прогон 24/12/3 и зачем он нужен (сессия
расширилась внутри окна); решение о максимально широких сетках; **риск-гейт стопа и его потолок**;
вердикт по планке; предпоследний baseline каталога (1.100) и безубыточный априорный walk-forward
(0.987); снятые риски (лучшая ликвидность вселенной, круг издержек 0.016% против моделируемых
0.1%); **остаточные риски** — тонкий edge (баланс 1.100 съедается любым дополнительным трением),
падающий инструмент (−60.5% за окно, −46.8% за последнее полугодие), два убыточных полугодия
baseline из шести.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -v`
Expected: PASS. Два теста прямо сторожат эту пару задач: `TestEveryDefaultTickerIsRegistered`
(каждый тикер боевой вселенной обязан быть в карте) и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (тикер, всё ещё отслеживающий baseline, в
боевую вселенную попасть не может). Второй — причина, по которой Task 12 идёт раньше Task 14.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live
git commit -m "feat(rsi_pullback): AFKS в реестре живого раннера"
```

---

### Task 14: Боевая вселенная

**Files:**
- Modify: `internal/config/rsi_pullback.go` (комментарий-абзац + список `Tickers`, строка 314)
- Modify: `internal/config/rsi_pullback_test.go` (список `want`, строка 54)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Modify: `docs/rsi_pullback/live.md` (§8, значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  конфигурации — это механика, а не пер-тикерная запись)

**Interfaces:**
- Consumes: литерал из Task 12, реестр из Task 13.
- Produces: `RSI_PULLBACK_TICKERS` из двадцати двух тикеров.

- [ ] **Step 1: Проверить стоп-условие ещё раз**

Перечитать числа Task 11 Steps 4–5. Все ТРИ пункта стоп-условия не сработали — иначе эта задача не
выполняется вовсе.

- [ ] **Step 2: Обновить тест дефолта**

В `internal/config/rsi_pullback_test.go` в список `want` (строка 54) добавить `"AFKS"` двадцать
вторым, после `"NKHP"`.

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -v`
Expected: FAIL — дефолт ещё из двадцати одного тикера.

- [ ] **Step 4: Обновить дефолт и env-файлы**

В `internal/config/rsi_pullback.go` дописать `"AFKS"` в конец списка `Tickers` и добавить перед ним
комментарий-абзац в стиле соседних (NKHP, SNGSP, ASTR): когда заведён, вердикт по планке, числа
точки, риск-гейт и его цена, **тип риска** (у AFKS он не исполнительный, а сигнальный: тонкий
edge на самой ликвидной бумаге вселенной), каноническая процедура тем, схема 36/12/6 с контрольным
прогоном 24/12/3 и решение о широких сетках.

Во всех трёх env-файлах дописать `,AFKS` в конец `RSI_PULLBACK_TICKERS`. Текущее значение:
`UGLD,T,GAZP,DOMRF,FESH,WUSH,LENT,RENI,NVTK,LSNGP,IVAT,SVAV,SIBN,ELFV,DIAS,BSPB,YDEX,BANEP,ASTR,SNGSP,NKHP`.

- [ ] **Step 5: Обновить значение дефолта в `live.md` §8**

В таблице конфигурации строка `RSI_PULLBACK_TICKERS` содержит дефолтный список. Дописать `,AFKS`.
Ничего пер-тикерного (вердиктов, PF, дат) в `docs/rsi_pullback/` не добавлять — правило CLAUDE.md.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести AFKS в боевую вселенную"
```

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/afks/afks.go` (док-комментарий
  пакета)
- Modify: `docs/rsi_pullback/strategy.md` — **только если** AFKS дал механический вывод

**Interfaces:**
- Consumes: рабочую записку Task 11.
- Produces: пер-тикерную запись, которую читает следующий калибратор.

- [ ] **Step 1: Переписать шапку пакета в итог калибровки**

Заменить блок «СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ» на разбор: вердикт по планке (по каждому из
двух критериев каждой ключевой темы отдельно); точку поле за полем с голосами фолдов; **риск-гейт:
что выбрала тема, что принято, цена решения в PF**; обе таблицы фолдов (36/12/6 и 24/12/3); три
строки издержек с пометкой, что это предупреждение, а не стоп, и почему; соседей плато с пометкой
инертных осей; полугодия точки; анатомию против baseline; принятые риски и условие пересмотра.

- [ ] **Step 2: Записать принятые риски прямым текстом**

Минимум, который обязан быть назван: тонкий edge (baseline 1.100 — предпоследний каталога,
априорный walk-forward 0.987); падающий инструмент (−60.5% за окно, −46.8% за последнее полугодие,
максимальная просадка −77.1%); два убыточных полугодия baseline из шести, причём убыточно
единственное растущее полугодие инструмента; чувствительность к трению (0.2% круга уводит baseline
под единицу). Условие пересмотра сформулировать явно.

- [ ] **Step 3: Механический вывод — только если он есть**

Кандидаты на строку в `docs/rsi_pullback/strategy.md`: (а) исход проверки объяснения NKHP про
объёмный гейт на противоположном полюсе ликвидности (§8.1); (б) исход гипотезы про
`VolLookbackBars` на самой ликвидной бумаге каталога; (в) риск-гейт стопа как механизм — если
владелец решит распространить его на остальные тикеры, он описывается в `strategy.md` как
процедура, без чисел AFKS. Пер-тикерных чисел в `docs/rsi_pullback/` не появляется.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/afks docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки AFKS и принятый риск"
```

---

### Task 16: Финальная проверка

**Files:**
- Отчёты: `reports/AFKS_parity/` (каталог в `.gitignore`)

**Interfaces:**
- Consumes: всё, что сделали Tasks 12–15.
- Produces: доказательство, что живая сборка и бэктест торгуют одинаково, и зелёный CI.

- [ ] **Step 1: Сверить живую сборку с бэктестом**

```bash
go run ./cmd/pullparity -tickers AFKS -months 36
```

Expected: **ноль расхождений**. Ненулевое расхождение означает, что живой раннер и бэктест строят
разные сигналы на одних данных — останавливаться и разбираться, а не сглаживать.

- [ ] **Step 2: Прогнать полный гейт качества**

```bash
./bin/mage ci
```

Expected: зелёный (lint + `go test -race ./...` + проверка дрейфа моков). Если `./bin/mage`
отсутствует — сначала `./bin/mage tools`.

- [ ] **Step 3: Проверить, что `reports/` не попал в индекс**

```bash
git status --porcelain | grep -c '^.. reports/' || true
```

Expected: `0`.

- [ ] **Step 4: Финальный коммит, если остались правки**

```bash
git add -A ':!reports'
git commit -m "feat(rsi_pullback): AFKS — сверка живой сборки и зелёный CI"
```

---

## Self-Review

**Покрытие спеки:** десять тем спеки → Tasks 3–10; сетки и их ширина → Task 1 (сторожевой тест плюс
два теста инвариантов); пакет и реестр бэктеста → Task 2; правило сборки точки, риск-гейт и три
пункта стоп-условия → Task 11; издержки как предупреждение → Task 11 Step 6; литерал и пин
риск-гейта → Task 12; живой раннер → Task 13; боевая вселенная двадцать вторым → Task 14;
пер-тикерная дока и механический вывод → Task 15; pullparity и CI → Task 16.

**Плейсхолдеры:** значения литерала в Tasks 11–12 подставляются из результата прогонов — это
единственные «нули» в плане, и они помечены прямым указанием, откуда берутся. Все команды, оси
сеток и опорные числа приведены целиком.

**Согласованность типов и имён:** `afks.Ticker` / `afks.DefaultParams()` определены в Task 2 и
используются в Tasks 12–14 под теми же именами; алиас импорта `rsipullbackafks` — только в реестре
бэктеста (Task 2 Step 5), в живом раннере импорт без алиаса (Task 13), как у соседних тикеров;
хелпер `rsiPullbackTickerGrid(t, "afks", <файл>)` — существующий, из
`internal/service/backtest/rsi_pullback_grid_test.go:40`.
