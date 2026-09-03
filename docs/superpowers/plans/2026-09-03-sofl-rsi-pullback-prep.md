# SOFL под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести SOFL (ПАО «Софтлайн») до вердикта по стратегии `rsi_pullback`: каталог
максимально широких сеток, тематический walk-forward по гибридной схеме с якорем, принятая точка,
прошедшая два риск-гейта, литерал в пакете и заведение в боевую вселенную двадцать вторым тикером.

**Architecture:** Процедура тем **гибридная, с якорем** (прецедент BSPB): дефолты ядра на SOFL
убыточны (PF 0.987), поэтому ранние три темы (`screen`, `entry`, `trend`) идут поверх дефолтов —
только так вердикт сравним с каталогом и осмысленна планка, — а поздние семь тем идут поверх
**якоря**, собранного из победителей pooled OOS тем `entry` и `trend`. Файлы поздних тем создаются
после Task 5, потому что до прогона ранних тем якоря не существует. Схема прогонов **каноническая
36/12/6** (четыре фолда, проверено контрольным прогоном до написания спеки), плюс **обязательный
контрольный прогон принятой точки по схеме 24/12/3** — внутри окна расширилась сессия (29 → 35
баров в буднем дне). Новое против каталога — **два риск-гейта**: гейт выживаемости применяется к
эффективной защите `min(StopDailyATR, TrailDailyATR при UseTrail=1)` (урок AFKS, где трейл обошёл
гейт стопа), и гейт просадки — max DD точки ≤ 1.3 × max DD baseline.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-03-sofl-rsi-pullback-prep-design.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех без исключения прогонах. Флаг `-interval Minutes30` обязателен
  в каждой команде: дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Схема прогонов каноническая:** `-months 36 -train-months 12 -test-months 6 -min-trades 20
  -metric profit_factor`, четыре фолда встык. У темы `screen` — `-min-trades 1`.
- **Окно фактически 35.3 месяца:** история SOFL начинается днём IPO **2023-09-25**, поэтому движок
  обрезает запрошенные 36 месяцев по началу истории и печатает предупреждение
  «has no candles before 2023-09-25» — это **ожидаемое поведение, а не сбой**. Расчётное окно —
  **2023-09-25 … 2026-09-03**.
- **Число фолдов сверяется на первой же теме.** Контрольный прогон 2026-09-03 уже дал «Фолдов: 4»
  (pooled OOS PF 0.845 на схеме 36/12/6 и 0.946 на 24/12/3), но сверка повторяется: если в шапке
  отчёта темы `screen` окажется «Фолдов: 3» — **остановиться и доложить владельцу**, остальные темы
  не запускать. Правило сборки «≥ 3 фолда из 4» при трёх фолдах вырождается в «3 из 3».
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш дотянут штатным top-up 2026-09-03:
  `SOFL_Minutes30.json` — 33 976 баров (2023-09-25 … 2026-09-03), из них 24 644 будних;
  `SOFL_Day1.json` — 864 свечи (2023-09-25 … 2026-09-02).
- **Дыр в серии нет.** Разрыв длиннее четырёх дней ровно один — новогодние каникулы
  2023-12-29 → 2024-01-03 (4.4 дня), календарный. Скачков «открытие против вчерашнего закрытия»
  глубже 12% ноль. Ни один шаг не расширяет окно за 36 месяцев.
- **ГИБРИДНАЯ ПРОЦЕДУРА ТЕМ.** Ранние темы (`screen`, `entry`, `trend`) — поверх дефолтов ядра.
  Поздние семь (`day`, `day_spent`, `volume`, `vol_window`, `risk`, `exit`, `trail`) — поверх
  **якоря**, зафиксированного в Task 6. Якорь: `RSIPeriod`, `RSILower`, `RSIUpper` — из победителя
  pooled OOS темы `entry`; `EMAFast`, `EMASlow` — из победителя pooled OOS темы `trend`; все
  остальные поля — дефолты ядра. Если победитель `entry` даёт < 20 сделок на полной истории,
  якорь берёт второе место лидерборда, и оба кандидата с числами пишутся в `_comment` всех семи
  поздних файлов. Якорь фиксируется один раз и между поздними темами не меняется.
- **Сетки держатся максимально широкими** (решение владельца 2026-09-03). Обрезок осей не делается;
  замеры, которые в узком каталоге были бы основанием вырезать край, идут в `_comment` как
  **предупреждения**. Жёсткие инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower`
  на всех парах, ось тренда не порождает пар `EMAFast ≥ EMASlow`, `StopDailyATR` нигде не равен
  нулю.
- **Правило сборки точки:** поле берётся из темы, которая его меряет, и принимается только если за
  значение высказались **не менее трёх фолдов из четырёх**; иначе поле остаётся на **значении
  якоря** (поля входа и тренда) или на **дефолте ядра** (остальные). Ничья 2/2 большинством не
  считается.
- **РИСК-ГЕЙТ A — выживаемость эффективной защиты** (решение владельца 2026-09-03, объявлен до
  прогонов). Принимается только защита, чей уровень достижим **не менее чем в 30% будних дней
  окна**, и гейт применяется к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, а не к одному
  лишь стопу. Таблица выживаемости SOFL (n = 735 будних дней): 0.3 — 99.3%, 0.5 — 91.8%,
  0.7 — 74.1%, **1.0 — 46.1%**, 1.3 — 24.5%, 1.5 — 15.4%, 2.0 — 5.4%. **Потолок — 1.0.** Если тема
  голосует выше, в точку идёт ближайшее значение оси внутри гейта, а цена решения в PF пишется
  прямым текстом в `_comment` и в доку пакета. Порог 30% после начала прогонов не двигается.
  Основание расширения (урок AFKS): там гейт отверг стоп 2.0 и поставил 1.0, но тема `trail`
  независимо приняла 0.7 — трейл сел ближе стопа, дал 0 SL-выходов из 107 и просадку 18.50%
  против 13.88% у baseline. Гейт по одному стопу этот обход не ловит.
- **РИСК-ГЕЙТ B — просадка точки.** Max DD принятой точки на полном окне (одиночный прогон
  `-params`) **не превышает 1.3 × max DD baseline**: 15.84% × 1.3 = **20.6%**. Нарушение → поле,
  отвечающее за просадку (стоп или трейл), сдвигается к ближайшему значению оси, возвращающему
  точку под потолок; цена решения в PF пишется прямым текстом. Гейт измерим: стоп 2.0 даёт
  PF 1.226 при DD 17.45%, трейл 0.3 даёт PF 1.122 при DD 11.19%.
- **Третий контур — риск-профиль точки против baseline.** Опорные числа baseline: доля SL-выходов
  **30.7%**, удержание медиана **9** / p90 **21** / максимум **43** бара, ночёвок **48.8%**,
  выходов в выходную сессию **9.6%**, max DD **15.84%**. Точка меряется по всем пяти. Если падение
  доли SL-выходов идёт вместе с ростом удержания и ночёвок — это подпись капкана, и точка берёт
  более узкую защиту с записью цены решения.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` обе дают pooled
  OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для `entry`, `EMASlow` для `trend`)
  выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд (ни одной убыточной сделки) в пользу
  тикера не засчитывается; счёт по дефолтной комиссии (круг 0.1%).
- **Стоп-условие из ТРЁХ пунктов.** Работа останавливается, если принятая точка даёт: (1) pooled
  OOS PF < 1.0 на 36/12/6, **либо** (2) меньше 20 сделок в пуле OOS, **либо** (3) pooled OOS
  PF < 1.0 на контрольном прогоне 24/12/3. При срабатывании — числа владельцу, **задачи 13–17 не
  выполняются**.
- **Издержки — предупреждение, не стоп.** Счёт под `-commission 0.001` и `0.0015` снимается и
  пишется в доку, но работу не останавливает: настоящий круг SOFL 0.051% (шаг 0.02 ₽ при цене
  77.8 ₽) против моделируемых 0.1%, удвоенная модель завышает факт вчетверо и её проваливает уже
  сам baseline (0.987 → 0.812 → 0.662 → 0.535 при 0.1/0.2/0.3/0.4%).
- **Правило прода:** литерал ставится и SOFL заводится в `RSI_PULLBACK_TICKERS` двадцать вторым
  **независимо от того, взята планка или нет**. Стоп-условие это правило перевешивает.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают ранние темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`,
  `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`,
  `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`,
  `UseTrail 0`, `TrailDailyATR 0`. Контрольный прогон дефолтов на расчётном окне: **166 сделок,
  PF 0.987, net −1 134.65 ₽**, win rate 60.24%, max DD 15.84%, выходы RSI 90 / SL 51 (30.7%) /
  TP 25 — **предпоследнее место** в ряду baseline каталога (YDEX 1.778, NKHP 1.697, LSNGP 1.554,
  ELFV 1.546, SNGSP 1.535, IVAT 1.432, LENT 1.417, SVAV 1.368, BANEP 1.336, BSPB 1.114,
  AFKS 1.100, ASTR 1.093, SIBN 1.027, **SOFL 0.987**, NVTK 0.984). На 24 месяцах — 134 сделки,
  PF 1.001.
- **Полугодия расчётного окна** (нужны в Task 12): 2023-09-25…2024-03-03 (10 сделок, PF 0.829,
  net −1 140), 2024-03-03…2024-09-03 (инструмент −11.3%, 22, 0.990, −92),
  2024-09-03…2025-03-03 (−14.2%, 37, 1.081, +1 887), 2025-03-03…2025-09-03 (−14.0%, 38,
  **0.722**, −5 440), 2025-09-03…2026-03-03 (−24.9%, 33, **0.740**, −4 426),
  2026-03-03…2026-09-03 (−33.5%, 26, **1.698**, +8 076). **Четыре убыточных из шести**, и весь
  положительный net делает последнее полугодие. Итог окна: инструмент −67.1%, максимальная
  просадка инструмента −81.6%, **ни одного растущего полугодия**.
- **Ликвидность падает третий год.** Медиана оборота будних дней (лот 10): 134.5 млн ₽ (36 мес),
  100.0 (24 мес), 77.0 (12 мес); по годам 314 → 208 → 124 → **62 млн ₽ (2026)** при гейте вселенной
  50 млн. Выходная сессия — 266 дней с барами, медиана 6.0 млн ₽. Дневной ATR(14) медиана 3.12%
  цены.
- **Сессия расширилась внутри окна**: медиана баров в буднем дне 29 (2023–2024) → 35 (2025–2026).
  Отсюда контрольный прогон 24/12/3 и пункт 3 стоп-условия.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; **для поздних тем — якорь и то, что тема
  идёт поверх него**; полную команду запуска с путём `data/params/rsi_pullback/sofl/<файл>` (этого
  требует `TestRSIPullbackCalFilesValid`: тест падает, если `_comment` не называет путь
  собственного файла); место под строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-03: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается.** Правило CLAUDE.md: `strategy.md`,
  `live.md`, `screener.md` описывают механику. Пер-тикерный разбор живёт в доке пакета
  `strategy/sofl` и в `_comment` файлов сеток.
- **Каталог `reports/` — в `.gitignore`.** Отчёты остаются локальными; в репозиторий едут строки
  `РЕЗУЛЬТАТ ПРОГОНА` в `_comment`, рабочая записка Task 12 и код.
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/sofl-pullback-prep` от `feat/afks-pullback-prep` (`0d675cf`); спека уже
  закоммичена (`908b874`). Замеры инструмента — `reports/_analysis/sofl_pullback_prep_measurements.md`
  (вне git).

---

### Task 1: Каталог трёх ранних сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/sofl/cal_screen.json`, `cal_entry.json`, `cal_trend.json`
- Create: `internal/service/backtest/rsi_pullback_sofl_grid_test.go`

**Interfaces:**
- Consumes: формат `Phase` из `internal/service/backtest/calibrate.go` (`ParsePhases`); хелпер
  `rsiPullbackTickerGrid(t, ticker, file) map[string][]float64` из
  `internal/service/backtest/rsi_pullback_grid_test.go:40`.
- Produces: три файла ранних сеток (читают Tasks 3–5) и сторожевые тесты `TestSOFLGridsStayWide`,
  `TestSOFLEntryGridKeepsRSIUpperAboveRSILower`, `TestSOFLTrendGridKeepsFastBelowSlow`.

- [ ] **Step 1: Написать падающий сторожевой тест осей**

Создать `internal/service/backtest/rsi_pullback_sofl_grid_test.go`:

```go
package backtest

import "testing"

// TestSOFLGridsStayWide держит НИЖНЮЮ границу ширины сеток SOFL. Владелец потребовал 2026-09-03
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет
// наличие обязательных значений и не запрещает лишних. Три отступления от канона §8.1 посажены на
// замеры (reports/_analysis/sofl_pullback_prep_measurements.md) и пиньятся здесь, чтобы редактор
// сеток не «починил» их обратно к канону:
//
//   - RSIPeriod ВНИЗ до 2 и ВВЕРХ до 10 (канон 3..8). Максимум оси стоит ВНУТРИ неё: 5 -> 1.260/114
//     против дефолтной четвёрки 0.987/166, при этом 6 -> 0.983, 7 -> 1.059, а 8 и 10 мертвы
//     (0.800/57 и 0.803/43). Верхний край нужен, чтобы отделить «максимум внутри» от «максимум на
//     краю»; нижний — ради симметрии проверки (двойка убыточна: 0.965 на 415 сделках).
//     Четырнадцать в сетку не идут вовсе: 9 сделок за три года.
//   - RSIUpper в cal_exit ВНИЗ до 35 при сохранённом верхе 85. Рельеф рваный, и ВТОРАЯ живая зона
//     лежит целиком ниже канонической полосы: 45 -> 1.058/203 и 50 -> 1.058/193 против провала
//     55 -> 0.915 и 65 -> 0.926. Максимум оси (75 -> 1.115/162) канонической полосе принадлежит,
//     но оба её края (55 и 85 -> 0.803) — две худшие точки замера.
//   - EMASlow НЕ расширяется вниз до 20/30, а EMAFast держится канонически широкой (3..40). Это
//     ЗЕРКАЛО AFKS, где пришлось резать быструю ось: на SOFL 20 -> 0.957 и 30 -> 0.939 — две
//     ХУДШИЕ точки медленной оси, тогда как 20 -> 1.073 и 30 -> 1.070 — две ЛУЧШИЕ точки быстрой.
//     Инвариант EMAFast < EMASlow не разрешает держать обе оси широкими, и замер однозначно
//     говорит, какую отдавать.
func TestSOFLGridsStayWide(t *testing.T) {
	cases := []struct {
		file   string
		field  string
		values []float64
	}{
		{"cal_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_entry.json", "RSILower", []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8, 10}},
		{"cal_entry.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_trend.json", "EMAFast", []float64{3, 5, 10, 20, 30, 40}},
		{"cal_trend.json", "EMASlow", []float64{50, 70, 100, 150, 200, 250}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "sofl", c.file)
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
				t.Errorf("sofl/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestSOFLEntryGridKeepsRSIUpperAboveRSILower пинит жёсткий инвариант связки входа и выхода: в
// теме entry обе оси свипуются вместе, и пара, где уровень выхода не выше уровня входа, описывает
// сделку, которая закрывается в момент открытия.
func TestSOFLEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "sofl", "cal_entry.json")
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

// TestSOFLTrendGridKeepsFastBelowSlow пинит вторую половину решения об осях тренда: EMASlow
// начинается с 50 ИМЕННО ПОТОМУ, что EMAFast держится широкой до 40. Любая пара, где быстрая EMA
// не быстрее медленной, превращает трендовый фильтр в перевёрнутый и на падающем инструменте
// способна выиграть тему in-sample.
func TestSOFLTrendGridKeepsFastBelowSlow(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "sofl", "cal_trend.json")
	for _, fast := range grid["EMAFast"] {
		for _, slow := range grid["EMASlow"] {
			if fast >= slow {
				t.Fatalf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: перевёрнутый трендовый фильтр", fast, slow)
			}
		}
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/backtest/ -run 'TestSOFL' -v`
Expected: FAIL — каталога `data/params/rsi_pullback/sofl/` ещё нет.

- [ ] **Step 3: Создать три файла ранних сеток**

`cal_screen.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sofl/cal_screen.json — РАННЯЯ тема screen для SOFL, 4 прогона, поверх ДЕФОЛТОВ ЯДРА (core.DefaultParams(): UseDayATRGate=1, UseVolume=0). Тема мерит цену двух опциональных гейтов. ЗАМЕР 2026-09-03 (36 мес, in-sample): UseDayATRGate 0 -> 0.946/428 (net -11 777), 1 -> 0.987/166 — гейт режет 62% сделок и поднимает PF на 0.041, но ОБЕ строки убыточны: на SOFL дневной гейт не создаёт прибыль, он ограничивает убыток (отличие и от AFKS, где гейт носитель edge, и от NKHP, где без гейта остаётся 1.440). UseVolume в дефолтной форме (VolMult 1.2, VolBaseDays 14, VolLookbackBars 3) даёт 1.009/127 против 0.987/166 — плюс 0.022 PF. ЗАПУСК: go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sofl/cal_screen.json -out ./reports/SOFL_screen -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-03: <заполняется в Task 3>.",
  "phases": [
    {
      "name": "screen",
      "grid": {
        "UseDayATRGate": [0, 1],
        "UseVolume": [0, 1]
      },
      "keepTop": 4
    }
  ]
}
```

`cal_entry.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sofl/cal_entry.json — РАННЯЯ КЛЮЧЕВАЯ тема entry для SOFL, 504 прогона, поверх ДЕФОЛТОВ ЯДРА. Тема мерит вход целиком: глубину отката, период RSI и связанный уровень выхода. ОСЬ RSIPeriod РАСШИРЕНА ВНИЗ ДО 2 И ВВЕРХ ДО 10 (канон 3..8). ЗАМЕР 2026-09-03 (36 мес, in-sample) RSILower при RSI(4): 10 -> 1.276/22, 15 -> 1.363/56 (МАКСИМУМ ОСИ), 20 -> 1.262/95 (максимум net +13 746), 25 -> 1.057/125, 30 -> 0.987/166 (ДЕФОЛТ, УБЫТОЧЕН), 35 -> 0.939/192, 40 -> 1.007/224, 45 -> 1.080/259, 50 -> 1.009/281 — рабочая зона входа лежит ГЛУБОКО ПОД ДЕФОЛТОМ, дефолт стоит около локального минимума. ЭТО И ЕСТЬ ОСНОВАНИЕ ГИБРИДНОЙ СХЕМЫ С ЯКОРЕМ. RSIPeriod при уровне 30: 2 -> 0.965/415, 3 -> 0.995/239, 4 -> 0.987/166 (дефолт), 5 -> 1.260/114 (МАКСИМУМ ВНУТРИ ОСИ), 6 -> 0.983/93, 7 -> 1.059/76, 8 -> 0.800/57, 10 -> 0.803/43, 14 -> 0.729/9 (В СЕТКУ НЕ ИДЁТ: 9 сделок за три года). ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: победа двойки означала бы переоптимизацию обучающего окна (на полном окне она убыточна на 415 сделках); победа десятки означала бы, что ось надо было тянуть дальше вверх, хотя точечный замер там мёртв. RSIUpper здесь идёт КАНОНИЧЕСКОЙ полосой 55..85 и служит контекстом связки — изолированно её мерит тема exit. ЗАПУСК: go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sofl/cal_entry.json -out ./reports/SOFL_entry -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-03: <заполняется в Task 4>.",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIUpper": [55, 60, 65, 70, 75, 80, 85],
        "RSIPeriod": [2, 3, 4, 5, 6, 7, 8, 10],
        "RSILower": [10, 15, 20, 25, 30, 35, 40, 45, 50]
      },
      "keepTop": 5
    }
  ]
}
```

`cal_trend.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sofl/cal_trend.json — РАННЯЯ ВТОРАЯ КЛЮЧЕВАЯ тема trend для SOFL, 36 прогонов, поверх ДЕФОЛТОВ ЯДРА. Тема мерит трендовый фильтр. ЗАМЕР 2026-09-03 (36 мес, in-sample) EMASlow при EMAFast 10: 20 -> 0.957/152, 30 -> 0.939/158, 50 -> 0.998/160, 70 -> 0.944/163, 100 -> 0.987/166 (дефолт), 150 -> 1.050/162 (максимум), 200 -> 0.932/159, 250 -> 0.994/158 — размах 0.118 PF, САМАЯ ПЛОСКАЯ ОСЬ ТИКЕРА, вся она лежит около единицы. EMAFast при EMASlow 100: 3 -> 0.993/146, 5 -> 0.962/154, 10 -> 0.987/166 (дефолт), 20 -> 1.073/170, 30 -> 1.070/174, 40 -> 1.031/178 — ЛУЧШАЯ ЧАСТЬ БЫСТРОЙ ОСИ ЭТО ВЕРХ. РЕШЕНИЕ ОБ ОСЯХ (зеркало AFKS): EMAFast держится канонически широкой 3..40, а EMASlow НЕ расширяется вниз до 20/30, потому что 20 и 30 — две ХУДШИЕ точки медленной оси, тогда как 20 и 30 быстрой — две ЛУЧШИЕ. Инвариант EMAFast < EMASlow не разрешает держать обе оси широкими; ось выбрана по замеру, а не по инварианту. ЦЕНА РЕШЕНИЯ: пары EMASlow 20-30 не измеряются вовсе, и победа нижнего края медленной оси (50) будет означать, что ось поставлена не там — это пишется прямым текстом. ЗАПУСК: go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sofl/cal_trend.json -out ./reports/SOFL_trend -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-03: <заполняется в Task 5>.",
  "phases": [
    {
      "name": "trend",
      "grid": {
        "EMAFast": [3, 5, 10, 20, 30, 40],
        "EMASlow": [50, 70, 100, 150, 200, 250]
      },
      "keepTop": 5
    }
  ]
}
```

- [ ] **Step 4: Запустить тесты сеток**

Run: `go test ./internal/service/backtest/ -run 'TestSOFL|TestRSIPullback' -v`
Expected: PASS. `TestRSIPullbackCalFilesValid` проверит, что `_comment` каждого файла называет его
собственный путь; `TestRSIPullbackGridControlPoints` — что контроль `UseDayATRGate=0` и
`UseVolume=0` есть в наборе (он есть в `cal_screen.json`).

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sofl internal/service/backtest/rsi_pullback_sofl_grid_test.go
git commit -m "feat(rsi_pullback): каталог ранних сеток SOFL"
```

---

### Task 2: Пакет `strategy/sofl` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sofl/sofl.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sofl/sofl_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `sofl.Ticker` (= `"SOFL"`) и `sofl.DefaultParams()` — их читают Task 13 (литерал) и
  Task 14 (реестр живого раннера).

- [ ] **Step 1: Написать падающий тест пакета**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/sofl/sofl_test.go`:

```go
package sofl

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка SOFL ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 13, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("SOFL ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSOFL(t *testing.T) {
	if Ticker != "SOFL" {
		t.Fatalf("Ticker = %q, want SOFL", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sofl/ -v`
Expected: FAIL — пакета ещё нет.

- [ ] **Step 3: Создать пакет с честной шапкой**

Создать `sofl.go`. Док-комментарий пакета — состояние ДО калибровки; он полностью переписывается в
Task 16. Обязательные абзацы шапки: состояние «КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ», окно и схема, гибридная
процедура с якорем и её причина, оба риск-гейта с их числами, априор (baseline 0.987 и свободный
walk-forward 0.845), особенности бумаги (IPO 2023-09-25, падение 67%, ликвидность 62 млн и падает,
круг издержек 0.051%).

```go
// Package sofl supplies the ticker and rsi_pullback Params for SOFL (ПАО «Софтлайн», обыкновенные
// акции, лот 10).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. DefaultParams() возвращает ровно baseline ядра, и это
// сторожит sofl_test.go. Пакет заведён заранее, чтобы реестр бэктеста знал тикер до калибровки.
//
// ОКНО И СХЕМА. История SOFL начинается ДНЁМ IPO 2023-09-25, поэтому расчётное окно
// 2023-09-25 … 2026-09-03 — 35.3 месяца, а не 36: движок обрезает запрошенные -months 36 по началу
// истории и печатает предупреждение, и это ожидаемо. Схема прогонов каноническая -months 36
// -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor; число фолдов проверено
// контрольным прогоном ДО раскладки сеток (4 фолда, pooled OOS PF 0.845). Внутри окна расширилась
// сессия (29 -> 35 получасовых баров в буднем дне), поэтому принятая точка обязана пройти ещё и
// контрольный прогон 24/12/3, где сессия однородна; провал ниже 1.0 на нём — пункт 3
// стоп-условия.
//
// ПРОЦЕДУРА ТЕМ ГИБРИДНАЯ, С ЯКОРЕМ (прецедент BSPB), и причина названа замером: дефолты ядра на
// SOFL стоят в МЁРТВОЙ ЗОНЕ. Вход RSI(4)@30 даёт PF 0.987 и убыток, тогда как рабочая зона входа
// лежит глубоко в стороне (RSILower 15 -> 1.363/56, 20 -> 1.262/95; RSIPeriod 5 -> 1.260/114).
// Ранние темы screen, entry и trend идут поверх дефолтов — только так вердикт сравним с каталогом
// и осмысленна планка; поздние семь тем идут поверх якоря из победителей pooled OOS тем entry и
// trend. Цена решения: числа поздних тем сравнимы внутри SOFL, но не построчно с каталогом.
//
// ДВА РИСК-ГЕЙТА, объявленные владельцем 2026-09-03 ДО прогонов.
//   - ГЕЙТ A (выживаемость эффективной защиты): принимается только защита, достижимая не менее чем
//     в 30% будних дней окна, и гейт применяется к min(StopDailyATR, TrailDailyATR при
//     UseTrail=1), а не к одному стопу. Таблица выживаемости (n = 735): 0.3 — 99.3%, 0.5 — 91.8%,
//     0.7 — 74.1%, 1.0 — 46.1%, 1.3 — 24.5%, 1.5 — 15.4%, 2.0 — 5.4%. Потолок — 1.0. Расширение
//     против канона каталога принято по уроку AFKS, где гейт отверг стоп 2.0, но тема trail
//     независимо поставила 0.7 — трейл сел ближе стопа и дал 0 SL-выходов из 107 при просадке
//     18.50% против 13.88% у baseline.
//   - ГЕЙТ B (просадка точки): max DD принятой точки на полном окне не превышает 1.3 x max DD
//     baseline, то есть 20.6%. Гейт измерим именно на SOFL: стоп 2.0 даёт PF 1.226 при DD 17.45%,
//     а трейл 0.3 — PF 1.122 при DD 11.19%.
//
// АПРИОР, записанный ДО прогонов, СЛАБЕЙШИЙ В КАТАЛОГЕ ПО ОБОИМ ЗАМЕРАМ. Контрольный прогон
// дефолтов ядра: 166 сделок, PF 0.987, net -1 134.65 руб., win rate 60.24%, max DD 15.84%, выходы
// RSI 90 / SL 51 (30.7%) / TP 25 — предпоследнее место в ряду baseline каталога, дефолты ядра
// убыточны. Априорный walk-forward темы screen: pooled OOS PF 0.845 (36/12/6) и 0.946 (24/12/3) —
// свободный выбор гейтов вне in-sample убыточен на обеих схемах. Веса априору не придаётся: NVTK с
// baseline 0.984 дал принятую точку.
//
// ЧТО ЗНАЕМ ОБ ИНСТРУМЕНТЕ ДО ПРОГОНОВ (полностью —
// reports/_analysis/sofl_pullback_prep_measurements.md):
//
//   - РЕЖИМ: итог окна -67.1% (160.0 -> 52.7), максимальная просадка инструмента -81.6%, НИ ОДНОГО
//     РАСТУЩЕГО ПОЛУГОДИЯ. Четыре полугодия baseline убыточны из шести (0.829 / 0.990 / 1.081 /
//     0.722 / 0.740 / 1.698), и весь положительный net делает последнее.
//   - ЛИКВИДНОСТЬ ПАДАЕТ ТРЕТИЙ ГОД: медиана оборота будних дней 314 -> 208 -> 124 -> 62 млн руб.
//     (2026) при гейте вселенной 50 млн — запас сжался до 1.2 раза. Выходная сессия: 266 дней с
//     барами, медиана 6.0 млн руб.
//   - ИЗДЕРЖКИ: шаг 0.02 руб. при медианной цене 77.8 руб. — круг 0.051% против моделируемых 0.1%,
//     модель консервативна ВДВОЕ. Чувствительность baseline: 0.1% -> 0.987, 0.2% -> 0.812,
//     0.3% -> 0.662, 0.4% -> 0.535.
//   - ДНЕВНОЙ ГЕЙТ ограничивает убыток, но edge не создаёт: 0 -> 0.946/428, 1 -> 0.987/166 — обе
//     строки убыточны.
//   - КАПКАН ШИРОКОГО СТОПА, ТРИНАДЦАТЫЙ ТИКЕР ПОДРЯД, и здесь он виден ПО ПРОСАДКЕ: стоп
//     0.5 -> 0.987/166, 1.0 -> 1.150/158, 1.3 -> 1.166/156, 2.0 -> 1.226/156 при max DD 17.45%
//     против 15.84%. С 1.3 число сделок замирает.
//   - ЦЕЛЬ ПОЧТИ МЁРТВАЯ: размах оси 0.075 PF, выше 2.0 цель не достигается ни разу.
//   - ВЫХОД РВАНЫЙ: две живые зоны (45-50 и 75) разделены провалом 55-65; RSI-выход НЕЗАМЕНИМ (без
//     него 0.644/155 и net -44 444 руб.), а трейл 0.3 — единственная настройка тикера, которая
//     одновременно поднимает PF (+0.135) и снижает просадку (-4.65 п.п.).
package sofl

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the MOEX ticker this package parameterizes.
const Ticker = "SOFL"

// DefaultParams returns the rsi_pullback parameters for SOFL. Пока калибровка не проведена, это
// ровно baseline ядра — см. док-комментарий пакета.
func DefaultParams() core.Params { return core.DefaultParams() }
```

- [ ] **Step 4: Завести тикер в реестр бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт с алиасом
`rsipullbacksofl "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sofl"` (в
алфавитном порядке между `rsipullbacksngsp` и `rsipullbacksvav`) и строку в `rsiPullbackRegistry`:

```go
	rsipullbacksofl.Ticker:  rsiPullbackBindingFor(rsipullbacksofl.Ticker, rsipullbacksofl.DefaultParams),
```

В `internal/service/backtest/rsi_pullback_registry_test.go` добавить тест рядом с тестом AFKS:

```go
// TestRSIPullbackSOFLTracksBaseline держит честное состояние реестра: пока калибровка SOFL не
// проведена, реестр обязан отдавать ровно baseline ядра. Тест заменяется в Task 13 на проверку
// литерала.
func TestRSIPullbackSOFLTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbacksofl.Ticker]
	if !ok {
		t.Fatal("SOFL нет в реестре rsi_pullback")
	}
	p, err := b.ParseParams([]byte(`{}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if p != core.DefaultParams() {
		t.Fatalf("реестр отдаёт не baseline:\n got: %+v\nwant: %+v", p, core.DefaultParams())
	}
}
```

- [ ] **Step 5: Запустить тесты**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sofl internal/service/backtest
git commit -m "feat(rsi_pullback): пакет SOFL до калибровки"
```

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_screen.json` (строка результата в `_comment`)
- Отчёты: `reports/SOFL_screen/`

**Interfaces:**
- Consumes: `cal_screen.json` из Task 1.
- Produces: пофолдовых победителей `UseDayATRGate` и `UseVolume` для Task 12.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_screen.json \
  -out ./reports/SOFL_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Проверить число фолдов**

В шапке `*_walkforward.md` обязано стоять «Фолдов: 4». Если «Фолдов: 3» — **остановиться и
доложить владельцу**, остальные темы не запускать.

- [ ] **Step 3: Выписать результат**

Pooled OOS PF, число сделок пула, пофолдовые значения `UseDayATRGate` и `UseVolume`, пофолдовые
OOS PF / сделки / MaxDD%. Ожидание из точечного замера: `UseDayATRGate=1` вероятнее (без гейта
0.946/428 — убыток), по `UseVolume` согласия может не быть (гейт даёт всего +0.022 PF). Контрольный
прогон этой же темы до плана дал pooled OOS PF 0.845 — если число заметно другое, проверить, что
запуск шёл с `-min-trades 1` и без `-refresh`.

- [ ] **Step 4: Дописать строку результата в `_comment`**

Формат: `РЕЗУЛЬТАТ ПРОГОНА 2026-09-03: pooled OOS PF X.XXX на N сделках, фолдов 4, отчёт
reports/SOFL_screen/<файл>. Пофолдово: UseDayATRGate a,b,c,d; UseVolume a,b,c,d. …` плюс вывод: что
принято темой, что уходит в дефолт.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sofl/cal_screen.json
git commit -m "feat(rsi_pullback): SOFL, тема screen — цена гейтов"
```

---

### Task 4: Тема `entry` — ключевая, рабочая зона глубоко под дефолтом

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_entry.json`
- Отчёты: `reports/SOFL_entry/`

**Interfaces:**
- Consumes: `cal_entry.json` из Task 1.
- Produces: победителя pooled OOS (идёт в **якорь** Task 6) и пофолдовых победителей `RSILower`
  (ведущая ось планки), `RSIPeriod`, `RSIUpper` для Task 12.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_entry.json \
  -out ./reports/SOFL_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить планку по критериям A и B**

Критерий A: pooled OOS PF ≥ 1.5 при ≥ 20 сделках в пуле. Критерий B: `RSILower` выбран одинаково в
≥ 3 фолдах из 4. Оба записываются отдельно — «взят»/«провален», без сглаживания.

- [ ] **Step 3: Проверить края расширенной оси периода**

Отдельно записать, победил ли `RSIPeriod` 10 (верхний край) или 2 (нижний) хоть в одном фолде.
Победа 2 при точечном замере 0.965/415 — сигнал переоптимизации обучающего окна, пишется прямым
текстом. Победа 10 (точечно 0.803/43) означает, что ось стоит на краю — тоже пишется.

- [ ] **Step 4: Выписать победителя pooled OOS и его число сделок на полной истории**

Это будущий якорь. Записать тройку (`RSIPeriod`, `RSILower`, `RSIUpper`) победителя **pooled OOS**,
не лучшего фолда. Число сделок на полной истории проверяется в Task 6 Step 1 — от него зависит,
берётся ли первое место лидерборда или второе.

- [ ] **Step 5: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sofl/cal_entry.json
git commit -m "feat(rsi_pullback): SOFL, тема entry — ключевая ось входа"
```

---

### Task 5: Тема `trend` — вторая ключевая, EMAFast широкая, EMASlow от 50

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_trend.json`
- Отчёты: `reports/SOFL_trend/`

**Interfaces:**
- Consumes: `cal_trend.json` из Task 1.
- Produces: победителя pooled OOS (идёт в **якорь** Task 6) и пофолдовых победителей `EMASlow`
  (ведущая ось планки) и `EMAFast` для Task 12.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_trend.json \
  -out ./reports/SOFL_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить планку по критериям A и B**

Критерий A: pooled OOS PF ≥ 1.5 при ≥ 20 сделках. Критерий B: `EMASlow` выбран одинаково в ≥ 3
фолдах из 4. Точечный замер обещает провал критерия B: размах оси 0.118 PF, фильтр режимы не
разделяет.

- [ ] **Step 3: Проверить края обеих осей**

Записать отдельно: победил ли `EMASlow` 50 (нижний край) — это значит, что ось поставлена не там и
её надо было тянуть вниз, но тогда пришлось бы резать `EMAFast`, где стоит максимум замера; победил
ли `EMAFast` 40 (верхний край) — это подтверждение решения держать быструю ось широкой.

- [ ] **Step 4: Выписать победителя pooled OOS**

Пара (`EMAFast`, `EMASlow`) победителя **pooled OOS** — вторая половина якоря Task 6.

- [ ] **Step 5: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sofl/cal_trend.json
git commit -m "feat(rsi_pullback): SOFL, тема trend — вторая ключевая ось"
```

---

### Task 6: Фиксация якоря и семь поздних сеток

**Files:**
- Create: `data/params/rsi_pullback/sofl/cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Modify: `internal/service/backtest/rsi_pullback_sofl_grid_test.go` (дописать поздние оси в
  `TestSOFLGridsStayWide`)

**Interfaces:**
- Consumes: победителей pooled OOS тем `entry` (Task 4) и `trend` (Task 5).
- Produces: семь файлов поздних тем, которые читают Tasks 7–11.

- [ ] **Step 1: Проверить победителя `entry` на числе сделок**

Прогнать точку победителя `entry` на полной истории:

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -params <файл с тройкой победителя> -out ./reports/SOFL_anchor_probe -months 36
```

Если сделок **< 20**, якорь берёт **второе место лидерборда** темы `entry`, и оба кандидата с
числами записываются в `_comment` всех семи поздних файлов. Основание — рельеф входа SOFL: максимум
оси (`RSILower 15`) держится 56 сделками за три года, а край 10 — всего 22.

- [ ] **Step 2: Записать якорь**

Якорь = `RSIPeriod`, `RSILower`, `RSIUpper` из победителя pooled OOS `entry` + `EMAFast`, `EMASlow`
из победителя pooled OOS `trend` + **все остальные поля дефолтами ядра** (`DailyATRPeriod 14`,
`UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`,
`UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`, `UseTrail 0`,
`TrailDailyATR 0`). Подставлять в якорь точечные замеры поздних осей ЗАПРЕЩЕНО: это решило бы
вопрос темы до её прогона.

- [ ] **Step 3: Создать семь файлов поздних тем**

В каждом файле якорные поля записываются **однозначными списками** (`"RSIPeriod": [5]` — это
фиксация, а не свип, комбинации она не размножает), а свои оси — полностью. Ниже даны сетки; вместо
`<якорь>` подставляются значения Step 2.

`cal_day.json` — фаза `day`, grid: якорные поля + `FreshDayATR` [0, 0.2, 0.3, 0.4, 0.5, 0.6] ×
`SpentDayATR` [0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5] (54 прогона).
`_comment` обязан назвать путь файла, якорь и замеры: `SpentDayATR` 0.4 → 0.983/348, 0.5 → 1.024/298,
**0.6 → 1.113/250**, 0.7 → 1.091/207, 0.8 → 0.987/166 (дефолт в локальной яме), 0.9 → 1.048/137,
1.0 → 1.097/111, 1.25 → 1.011/63, **1.5 → 2.350/33 с предупреждением: 33 сделки за три года —
≈11 на обучающее окно, ниже `-min-trades 20`, край скорее всего отсечёт порог внутри фолдов**;
`FreshDayATR` 0 → 0.987 (максимум), дальше монотонно вниз до 0.6 → 0.907.

`cal_day_spent.json` — фаза `day_spent`, grid: якорные поля + `FreshDayATR` [0] × `SpentDayATR`
[0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5] (9 прогонов). Тема мерит исчерпанную ветку при
заведомо закрытой свежей — контроль взаимодействия веток.

`cal_volume.json` — фаза `volume`, grid: якорные поля + `UseVolume` [1] × `VolMult`
[1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 4.0] × `VolBaseDays` [3, 5, 10, 14, 20] (35 прогонов).
Замеры для `_comment`: `VolMult` 1.0 → 0.997/137, 1.2 → 1.009/127, 1.5 → 0.971/117, 2.0 → 1.101/96,
**2.5 → 1.133/82**, 3.0 → 1.017/69, 4.0 → 1.033/57; `VolBaseDays` **3 → 1.144/132**, 5 → 1.057/131,
10 → 1.051/133, 14 → 1.009/127, 20 → 0.915/129 — максимум на узком крае, ось монотонна.

`cal_vol_window.json` — фаза `vol_window`, grid: якорные поля + `UseVolume` [1] ×
`VolLookbackBars` [1, 2, 3, 5, 8, 12, 16, 24] × `VolMult` [1.2, 2.5, 4.0] (24 прогона).
Замеры: 1 → 1.011/106, 2 → 0.969/117, 3 → 1.009/127 (дефолт), 5 → 0.904/137, 8 → 0.889/143,
12 → 0.938/154, 16 → 0.971/163, 24 → 0.985/164 — максимум у узкого края, середина оси худшая.

`cal_risk.json` — фаза `risk`, grid: якорные поля + `StopDailyATR` [0.3, 0.5, 0.7, 1.0, 1.3, 1.5,
2.0] × `TPDailyATR` [0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5] (63 прогона).
`_comment` обязан содержать таблицу выживаемости и формулировку риск-гейта A: ось стопа стоит до
2.0 **как контроль капкана, а не как кандидат**; потолок принимаемого значения — 1.0 (46.1% будних
дней); строка `TPDailyATR 2.5` — контроль асимметрии, которого требует
`TestRSIPullbackGridControlPoints` от файла, свипующего стоп до 2.0.

`cal_exit.json` — фаза `exit`, grid: якорные поля + `RSIUpper` [35, 40, 45, 50, 55, 60, 65, 70, 75,
80, 85] (11 прогонов). Замеры рваного рельефа — в `_comment` целиком.
**Важно:** якорный `RSILower` в этом файле остаётся однозначным, и ось `RSIUpper` обязана целиком
лежать выше него; если якорный `RSILower` ≥ 35, нижние значения оси, нарушающие инвариант
`RSIUpper > RSILower`, из файла убираются, и это записывается в `_comment` как вынужденное сужение
с указанием причины.

`cal_trail.json` — фаза `trail`, grid: якорные поля + `UseRSIExit` [0, 1] × `UseTrail` [1] ×
`TrailDailyATR` [0.3, 0.5, 0.7, 1.0, 1.5] (10 прогонов).
Замеры: при `UseRSIExit 1` — **0.3 → 1.122/183 при max DD 11.19%** (baseline 15.84%), 0.5 → 1.039,
0.7 → 0.999, с 1.0 трейл не срабатывает вовсе; при `UseRSIExit 0` — 0.889 / 0.731 / 0.641 / 0.646 /
0.644, а `UseRSIExit 0` без трейла даёт 0.644/155 и net −44 444 ₽. RSI-выход незаменим.

- [ ] **Step 4: Дописать поздние оси в сторожевой тест**

В `TestSOFLGridsStayWide` добавить строки:

```go
		{"cal_day.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.2, 0.3, 0.4, 0.5, 0.6}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 4.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
```

Нижние значения оси `cal_exit.json` (35–50) в тест **не** пиньятся: они могут быть вынужденно
убраны инвариантом `RSIUpper > RSILower` от якоря (Step 3). Если якорный `RSILower` ≤ 30 и ось
осталась полной, дописать в тест ещё и `{35, 40, 45, 50}` отдельной строкой.

- [ ] **Step 5: Запустить тесты сеток**

Run: `go test ./internal/service/backtest/ -run 'TestSOFL|TestRSIPullback' -v`
Expected: PASS.

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/sofl internal/service/backtest/rsi_pullback_sofl_grid_test.go
git commit -m "feat(rsi_pullback): SOFL, якорь зафиксирован и семь поздних сеток"
```

---

### Task 7: Темы `day` и `day_spent` — дневной гейт поверх якоря

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_day.json`, `cal_day_spent.json`
- Отчёты: `reports/SOFL_day/`, `reports/SOFL_day_spent/`

**Interfaces:**
- Consumes: оба файла из Task 6.
- Produces: пофолдовых победителей `FreshDayATR` и `SpentDayATR` для Task 12 (`day` — основная
  тема, `day_spent` — контроль взаимодействия веток).

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_day.json -out ./reports/SOFL_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_day_spent.json -out ./reports/SOFL_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить темы между собой**

Если пофолдовые победители `SpentDayATR` в двух темах расходятся, это **взаимодействие веток**
(открытая свежая ветка сдвигает оптимум исчерпанной) — записывается прямым текстом, как на NKHP.

- [ ] **Step 3: Проверить край 1.5**

Отдельно записать, победил ли `SpentDayATR` 1.5 хоть в одном фолде. Точечно край даёт 2.350 на 33
сделках за три года — ниже порога `-min-trades 20` на обучающем окне; если порог его отсёк, это
ответ, а не сбой.

- [ ] **Step 4: Дописать строки результата в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sofl/cal_day.json data/params/rsi_pullback/sofl/cal_day_spent.json
git commit -m "feat(rsi_pullback): SOFL, темы day и day_spent"
```

---

### Task 8: Тема `volume` — объёмный гейт поверх якоря

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_volume.json`
- Отчёты: `reports/SOFL_volume/`

**Interfaces:**
- Consumes: `cal_volume.json` из Task 6.
- Produces: пофолдовых победителей `VolMult` и `VolBaseDays` для Task 12.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_volume.json -out ./reports/SOFL_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать результат и проверить край `VolBaseDays`**

Точечный максимум базы стоит на **узком крае** (3 → 1.144). Победа края в фолдах означает, что ось
надо было тянуть ниже, но ниже трёх дней базы нет — это записывается как упор в физическую границу,
а не как обрезанная ось.

- [ ] **Step 3: Записать место SOFL в каталожной гипотезе**

Каталог знает три исхода оси объёма: NKHP (оборот 14 млн — гейт несёт сигнал), AFKS (1226 млн — не
несёт), HEAD/NKHP по окну — расходятся. SOFL с оборотом 62–134 млн лежит между полюсами; результат
темы записывается как третья точка ряда, без обобщения на класс.

- [ ] **Step 4: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sofl/cal_volume.json
git commit -m "feat(rsi_pullback): SOFL, тема volume"
```

---

### Task 9: Тема `vol_window` — окно объёмного гейта

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_vol_window.json`
- Отчёты: `reports/SOFL_vol_window/`

**Interfaces:**
- Consumes: `cal_vol_window.json` из Task 6.
- Produces: пофолдовых победителей `VolLookbackBars` (и контекстного `VolMult`) для Task 12.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_vol_window.json -out ./reports/SOFL_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить `VolMult` с темой `volume`**

Если пофолдовые победители `VolMult` в двух темах расходятся, форма гейта **не откалибрована** —
это записывается прямым текстом, как на NKHP, и в точку `VolMult` идёт дефолтом ядра.

- [ ] **Step 3: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sofl/cal_vol_window.json
git commit -m "feat(rsi_pullback): SOFL, тема vol_window"
```

---

### Task 10: Тема `risk` — стоп, цель и обязательная проверка риск-гейта A

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_risk.json`
- Create: `data/params/rsi_pullback/sofl/plateau_stop_10.json`, `plateau_stop_20.json`
- Отчёты: `reports/SOFL_risk/`, `reports/SOFL_risk_probe_stop10/`, `reports/SOFL_risk_probe_stop20/`

**Interfaces:**
- Consumes: `cal_risk.json` из Task 6.
- Produces: пофолдовых победителей `StopDailyATR` и `TPDailyATR` для Task 12 **и результат
  применения риск-гейта A к победителю темы**.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_risk.json -out ./reports/SOFL_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Применить риск-гейт A к победителю темы**

Если большинство фолдов выбрало `StopDailyATR` > 1.0, победитель темы **отвергается гейтом**, и в
точку идёт 1.0 (последнее значение оси с выживаемостью ≥ 30%: 46.1%). Цена решения в PF считается
на точечном замере при одинаковом числе сделок и пишется прямым текстом в `_comment`.

- [ ] **Step 3: Снять два зонда капкана**

Создать два `plateau_*.json` — якорь + `TPDailyATR` победителя темы (или дефолт ядра, если
большинства нет) + `StopDailyATR` 1.0 и 2.0 соответственно, все ключи однозначными списками (этого
требует `TestRSIPullbackPlateauFilesArePoints`). Прогнать оба на полном окне:

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -params data/params/rsi_pullback/sofl/plateau_stop_10.json \
  -out ./reports/SOFL_risk_probe_stop10 -months 36

go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -params data/params/rsi_pullback/sofl/plateau_stop_20.json \
  -out ./reports/SOFL_risk_probe_stop20 -months 36
```

Для каждого зонда выписать: PF, число сделок, **max DD**, выходы RSI / SL / TP с долей SL, удержание
(медиана, p90, максимум), долю ночёвок, долю выходов в выходную сессию. Опорные числа baseline:
SL 30.7%, удержание 9 / 21 / 43, ночёвок 48.8%, выходных 9.6%, max DD 15.84%.

- [ ] **Step 4: Записать подпись капкана числами**

Если доля SL-выходов падает, а удержание, ночёвки и max DD растут — капкан подтверждён на самой
бумаге, и это пишется в `_comment` прямым текстом вместе с числами обоих зондов.

- [ ] **Step 5: Дописать строку результата в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sofl/cal_risk.json data/params/rsi_pullback/sofl/plateau_stop_*.json
git commit -m "feat(rsi_pullback): SOFL, тема risk и проверка риск-гейта стопа"
```

---

### Task 11: Темы `exit` и `trail` — выходы и проверка обхода гейта трейлом

**Files:**
- Modify: `data/params/rsi_pullback/sofl/cal_exit.json`, `cal_trail.json`
- Отчёты: `reports/SOFL_exit/`, `reports/SOFL_trail/`

**Interfaces:**
- Consumes: оба файла из Task 6.
- Produces: пофолдовых победителей `RSIUpper` (изолированно), `UseRSIExit` и `TrailDailyATR` для
  Task 12.

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_exit.json -out ./reports/SOFL_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/cal_trail.json -out ./reports/SOFL_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Проверить, не обходит ли трейл риск-гейт A**

Это **обязательный шаг, введённый по уроку AFKS**. Если тема `trail` выбрала дистанцию, которая
**меньше** принятого в Task 10 `StopDailyATR`, эффективная защита равна дистанции трейла, и
риск-гейт A применяется к ней: дистанция ниже потолка 1.0 гейт проходит по уровню, но точка обязана
пройти ещё и гейт B (просадка) в Task 12 Step 6. Если тема выбрала `UseRSIExit 0` — записать это
отдельной строкой: на SOFL отключение RSI-выхода обрушивает PF втрое (0.644/155, net −44 444 ₽), и
такой выбор темы против точечного замера требует прямого объяснения в `_comment`.

- [ ] **Step 3: Применить правило упрощения трейла**

Если тема выбрала дистанцию ≥ 1.0 (на SOFL трейл там не срабатывает вовсе — результат побайтово
равен baseline), в точку идут `UseTrail 0` и `TrailDailyATR 0` (прецедент ASTR).

- [ ] **Step 4: Дописать строки результата в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sofl/cal_exit.json data/params/rsi_pullback/sofl/cal_trail.json
git commit -m "feat(rsi_pullback): SOFL, темы exit и trail"
```

---

### Task 12: Сборка точки, два риск-гейта, два walk-forward и три пункта стоп-условия

**Files:**
- Create: `data/params/rsi_pullback/sofl/plateau_point.json`
- Create: соседи плато `data/params/rsi_pullback/sofl/plateau_<поле>_<значение>.json` — по одному
  на каждое поле, принятое темой большинством
- Create: `docs/superpowers/plans/task-12-report-sofl.md` (рабочая записка со всеми таблицами)
- Отчёты: `reports/SOFL_point_oos/`, `reports/SOFL_point_24/`, `reports/SOFL_point_full/`,
  `reports/SOFL_point_cost2/`, `reports/SOFL_point_cost3/`

**Interfaces:**
- Consumes: пофолдовых победителей всех десяти тем (Tasks 3–11), якорь (Task 6).
- Produces: принятую точку `core.Params` — её литерал ставит Task 13.

- [ ] **Step 1: Собрать точку по правилу большинства**

Поле за полем: голоса четырёх фолдов темы, которая это поле меряет. Принято — если ≥ 3 из 4;
иначе поле остаётся на **значении якоря** (`RSIPeriod`, `RSILower`, `RSIUpper`, `EMAFast`,
`EMASlow`) или на **дефолте ядра** (все остальные). Ничья 2/2 большинством не считается. Для
каждого поля записать: голоса, решение, и **случайно ли** принятое значение совпало с якорем или
дефолтом (совпадение по плюральности — не большинство).

- [ ] **Step 2: Применить риск-гейт A к собранной точке**

Посчитать эффективную защиту `min(StopDailyATR, TrailDailyATR при UseTrail=1)` и проверить её по
таблице выживаемости (потолок 1.0, 46.1%). Если защита выше потолка — сдвинуть к ближайшему
значению оси внутри гейта и записать цену решения в PF.

- [ ] **Step 3: Прогнать точку канонической схемой 36/12/6**

Создать `plateau_point.json` (все восемнадцать полей однозначными списками) и прогнать:

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/plateau_point.json -out ./reports/SOFL_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Выписать: pooled OOS PF, сделки пула, пофолдовые in-sample → OOS PF / сделки / NetPnL% / MaxDD%,
наличие вырожденных фолдов.

- [ ] **Step 4: Проверить пункты 1 и 2 стоп-условия**

Пункт 1: pooled OOS PF ≥ 1.0. Пункт 2: ≥ 20 сделок в пуле. При провале любого — **остановиться**,
принести числа владельцу, задачи 13–17 не выполнять.

- [ ] **Step 5: Прогнать контрольную схему 24/12/3 (пункт 3 стоп-условия)**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/plateau_point.json -out ./reports/SOFL_point_24 \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

Проверить «Фолдов: 4» в шапке. Pooled OOS PF < 1.0 → **остановиться**. Для контекста: baseline на
том же 24-месячном куске даёт 1.001 на 134 сделках — если точка ниже baseline при вчетверо меньшем
числе сделок, это записывается прямым текстом, как на AFKS.

- [ ] **Step 6: Применить риск-гейт B (просадка) на полной истории**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -params data/params/rsi_pullback/sofl/plateau_point.json -out ./reports/SOFL_point_full -months 36
```

Из отчёта взять **max DD**. Потолок — **20.6%** (1.3 × 15.84% у baseline). Превышение → сдвинуть
стоп или трейл к ближайшему значению оси, возвращающему точку под потолок, перезапустить Steps 3–6
и записать цену решения в PF прямым текстом.

- [ ] **Step 7: Снять анатомию и полугодия точки**

Из того же отчёта полной истории: выходы RSI / SL / TP с долей SL, удержание (медиана, p90,
максимум), доля ночёвок, доля выходов в выходную сессию, разбивка net по шести полугодиям.
Опорные числа baseline — в Global Constraints. **Точка, чей net держится одним полугодием, вердикта
не получает** (у baseline эта доля патологическая: +8 076 при итоге −1 135).

- [ ] **Step 8: Снять две строки издержек (предупреждение, не стоп)**

```bash
go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/plateau_point.json -out ./reports/SOFL_point_cost2 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.001

go run ./cmd/backtest -ticker SOFL -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sofl/plateau_point.json -out ./reports/SOFL_point_cost3 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.0015
```

Записать обе строки и сравнить относительную потерю с опорной у baseline (0.987 → 0.812 → 0.662,
то есть −17.7% и −32.9%). Работу эти числа не останавливают.

- [ ] **Step 9: Снять соседей плато**

По одному `plateau_*.json` на каждое поле, принятое темой большинством: то же значение ±один узел
оси, остальные поля точки без изменений. Прогнать каждого схемой 36/12/6 и выписать pooled OOS PF /
сделки. Сосед, совпавший с точкой **побайтово**, означает инертную ось (поле ничего не меняет) — это
записывается прямым текстом.

- [ ] **Step 10: Написать рабочую записку**

`docs/superpowers/plans/task-12-report-sofl.md`: якорь и его происхождение, таблица «поле → голоса
фолдов → решение», оба риск-гейта и их цена, два walk-forward, три пункта стоп-условия, полугодия,
анатомия против baseline, соседи плато, издержки, вердикт по планке по каждому критерию обеих
ключевых тем.

- [ ] **Step 11: Коммит**

```bash
git add data/params/rsi_pullback/sofl docs/superpowers/plans/task-12-report-sofl.md
git commit -m "feat(rsi_pullback): SOFL, принятая точка и её замеры"
```

---

### Task 13: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sofl/sofl.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sofl/sofl_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: принятую точку из Task 12.
- Produces: `sofl.DefaultParams()`, отдающий литерал, — его читают Task 14 и `cmd/pullparity`.

**Задача выполняется только если ни один пункт стоп-условия не сработал.**

- [ ] **Step 1: Заменить тест отслеживания baseline на снимок литерала**

В `sofl_test.go` удалить `TestParamsTrackTheBaselineUntilCalibrated` и написать снимок принятой
точки (значения подставить из Task 12):

```go
// TestDefaultParamsIsTheCalibratedSnapshot пинит принятую точку SOFL целиком. Литерал —
// результат процедуры Task 12 (гибридные темы, правило большинства 3 из 4, два риск-гейта), и
// любое его изменение обязано быть осознанным: тест падает на каждом поле.
func TestDefaultParamsIsTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       0, // <- подставить принятые значения из Task 12
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
		t.Fatalf("литерал SOFL разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestDefaultParamsDiffersFromCoreBaseline держит вторую половину смысла: калибровка проведена, и
// точка обязана отличаться от дефолтов ядра хотя бы одним полем.
func TestDefaultParamsDiffersFromCoreBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("литерал SOFL совпал с baseline ядра — значит калибровка не отражена в коде")
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sofl/ -v`
Expected: FAIL — `DefaultParams()` пока возвращает baseline.

- [ ] **Step 3: Поставить литерал**

Заменить `func DefaultParams() core.Params { return core.DefaultParams() }` на явный литерал
принятой точки (все восемнадцать полей перечислены явно, как в `nkhp.go`).

- [ ] **Step 4: Заменить тест реестра бэктеста**

В `rsi_pullback_registry_test.go` заменить `TestRSIPullbackSOFLTracksBaseline` на проверку, что
реестр отдаёт литерал пакета:

```go
func TestRSIPullbackSOFLServesTheCalibratedPoint(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbacksofl.Ticker]
	if !ok {
		t.Fatal("SOFL нет в реестре rsi_pullback")
	}
	p, err := b.ParseParams([]byte(`{}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if p != rsipullbacksofl.DefaultParams() {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, rsipullbacksofl.DefaultParams())
	}
}
```

- [ ] **Step 5: Запустить тесты и линтер**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.
Run: `./bin/golangci-lint run ./internal/...`
Expected: 0 issues.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sofl internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): SOFL откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 14: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (если тесты пакета
  требуют явной строки на тикер)

**Interfaces:**
- Consumes: `sofl.Ticker`, `sofl.DefaultParams()` из Task 13.
- Produces: запись в `paramsByTicker` — её читают `ParamsFor`, `StrategyFor` и `cmd/pullparity`.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sofl"` и строка в
`paramsByTicker` после `nkhp.Ticker`:

```go
	sofl.Ticker:  sofl.DefaultParams(),
```

- [ ] **Step 2: Дописать абзац комментария карты**

Перед картой в `registry.go` стоят англоязычные абзацы про каждый заведённый тикер. Дописать абзац
про SOFL в том же стиле: гибридная процедура с якорем и её причина (убыточные дефолты), вердикт по
планке, оба риск-гейта и их цена, и три принятых риска — падающая третий год ликвидность (62 млн ₽
в 2026 при гейте 50), режим без единого растущего полугодия (−67% за окно), выходная сессия с
медианным оборотом 6 млн ₽.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -v`
Expected: PASS. В пакете есть `TestEveryDefaultTickerIsRegistered` (каждый тикер боевой вселенной
обязан быть в карте) и `TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (тикер, всё ещё
отслеживающий baseline, в боевую вселенную попасть не может). Второй — причина, по которой Task 13
идёт раньше Task 15.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live
git commit -m "feat(rsi_pullback): SOFL в реестре живого раннера"
```

---

### Task 15: Боевая вселенная

**Files:**
- Modify: `internal/config/rsi_pullback.go` (комментарий-абзац + список `Tickers`, строка 314)
- Modify: `internal/config/rsi_pullback_test.go` (список `want`, строка 54)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Modify: `docs/rsi_pullback/live.md` (только значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  конфигурации — это механика, а не пер-тикерная запись)

**Interfaces:**
- Consumes: литерал из Task 13, реестр из Task 14.
- Produces: `RSI_PULLBACK_TICKERS` из двадцати двух тикеров.

- [ ] **Step 1: Проверить стоп-условие ещё раз**

Перечитать числа Task 12 Steps 4–6. Все три пункта стоп-условия не сработали и оба риск-гейта
пройдены — иначе эта задача не выполняется вовсе.

- [ ] **Step 2: Обновить тест дефолта**

В `internal/config/rsi_pullback_test.go:54` в список `want` добавить `"SOFL"` двадцать вторым,
после `"NKHP"`.

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -v`
Expected: FAIL — дефолт ещё из двадцати одного тикера.

- [ ] **Step 4: Обновить дефолт и env-файлы**

В `internal/config/rsi_pullback.go:314` дописать `"SOFL"` в конец списка `Tickers` и добавить перед
ним комментарий-абзац в стиле соседних (NKHP, SNGSP, ASTR): когда заведён, вердикт по планке, числа
точки, **тип риска** (у SOFL он двойной: исполнительный — ликвидность 62 млн ₽ и падает третий год
при гейте вселенной 50 — и режимный: инструмент −67% без единого растущего полугодия), гибридная
процедура тем и её цена, оба риск-гейта.

Во всех трёх env-файлах дописать `,SOFL` в конец `RSI_PULLBACK_TICKERS`. Текущее значение:
`UGLD,T,GAZP,DOMRF,FESH,WUSH,LENT,RENI,NVTK,LSNGP,IVAT,SVAV,SIBN,ELFV,DIAS,BSPB,YDEX,BANEP,ASTR,SNGSP,NKHP`.

- [ ] **Step 5: Обновить значение дефолта в `live.md`**

В таблице конфигурации строка `RSI_PULLBACK_TICKERS` содержит дефолтный список — дописать `,SOFL`.
Ничего пер-тикерного (вердиктов, PF, дат) в `docs/rsi_pullback/` не добавлять — правило CLAUDE.md.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести SOFL в боевую вселенную"
```

---

### Task 16: Дока пакета — разбор калибровки и принятый риск

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sofl/sofl.go` (док-комментарий
  пакета)
- Modify: `docs/rsi_pullback/strategy.md` — **только если** SOFL дал механический вывод

**Interfaces:**
- Consumes: рабочую записку Task 12.
- Produces: пер-тикерную запись, которую читает следующий калибратор.

- [ ] **Step 1: Переписать шапку пакета в итог калибровки**

Заменить блок «СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ» на разбор: вердикт по планке (по каждому из
двух критериев обеих ключевых тем), якорь и его происхождение, принятая точка поле за полем с
голосами фолдов, оба walk-forward, две строки издержек, соседи плато с пометкой инертных осей,
шесть полугодий, анатомия (SL, удержание, ночёвки, выходы в выходную сессию, max DD), результат
обоих риск-гейтов и их цена в PF.

Отдельными абзацами — **принятые риски и условие пересмотра**:

1. **ликвидность падает третий год**: 314 → 208 → 124 → 62 млн ₽ (2026) при гейте вселенной
   50 млн — запас 1.2×; риск исполнения, калибровкой не устраняемый;
2. **режим без единого растущего полугодия** (инструмент −67.1%, просадка −81.6%): стратегия
   проверена ТОЛЬКО против падения — **условие пересмотра: при первом же растущем полугодии
   калибровку повторить, не дожидаясь планового цикла**;
3. **выходная сессия**: доля выходов точки в дни с медианным оборотом 6.0 млн ₽ (у baseline 9.6%),
   проскальзывание там движок не моделирует;
4. **сессия расширилась внутри окна** (29 → 35 баров): первая треть окна структурно другая, отсюда
   контрольный прогон 24/12/3;
5. **гибридная процедура с якорем**: поздние семь тем условны от результата ранних двух, их числа
   не сравнимы построчно с каталогом;
6. **капкан широкого стопа** и его числа на принятой точке, включая max DD.

- [ ] **Step 2: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS (правка только комментария).

- [ ] **Step 3: Механический вывод в `docs/rsi_pullback/strategy.md` — если он есть**

Кандидат назван спекой заранее: **риск-гейт по эффективной защите `min(stop, trail)` и гейт
просадки** — это механика процедуры, а не свойство SOFL. Если оба гейта на SOFL сработали
осмысленно (отсекли конфигурацию, которую гейт по одному стопу пропустил бы), дописать в §8.1
две-три строки механики: гейт применяется к эффективной защите, а не к уровню стопа, и точка
дополнительно ограничена по max DD относительно baseline.

**Никаких пер-тикерных чисел, дат, вердиктов и PF в `docs/rsi_pullback/` не писать** — правило
CLAUDE.md. Если подтверждения нет — шаг пропускается целиком.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sofl docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки SOFL и принятый риск"
```

---

### Task 17: Финальная проверка

**Files:**
- Отчёты: `reports/SOFL_parity/` (каталог в `.gitignore`)

**Interfaces:**
- Consumes: всё, что сделали Tasks 13–16.
- Produces: доказательство, что живая сборка и бэктест торгуют одинаково, и зелёный CI.

- [ ] **Step 1: Сверить живую сборку с бэктестом**

```bash
go run ./cmd/pullparity -tickers SOFL -months 36
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
git commit -m "feat(rsi_pullback): SOFL — сверка живой сборки и зелёный CI"
```

---

## Self-Review

**Покрытие спеки:** десять тем → Tasks 3–5 (ранние) и 7–11 (поздние); гибридная схема и правило
якоря → Task 6; сетки и их ширина → Tasks 1 и 6 (+ сторожевые тесты); пакет и реестр бэктеста →
Task 2; правило сборки точки → Task 12 Step 1; риск-гейт A → Task 10 Step 2, Task 11 Step 2,
Task 12 Step 2; риск-гейт B → Task 12 Step 6; третий контур (риск-профиль) → Task 12 Step 7; три
пункта стоп-условия → Task 12 Steps 4 и 5; контрольный прогон 24/12/3 → Task 12 Step 5; издержки
как предупреждение → Task 12 Step 8; соседи плато → Task 12 Step 9; литерал → Task 13; реестр
живого раннера → Task 14; боевая вселенная двадцать вторым тикером → Task 15; пер-тикерная запись и
механический вывод → Task 16; `pullparity` и `mage ci` → Task 17; проверка числа фолдов (ловушка
ASTR) → Task 3 Step 2.

**Плейсхолдеры:** три места пусты по построению — якорь в Task 6 (существует только после Tasks 4–5),
сетки поздних тем в Task 6 Step 3 (зависят от якоря) и литерал в Task 13 Steps 1 и 3 (точка
известна только после Task 12). Везде указано, откуда берётся значение и по какому правилу.

**Согласованность имён:** `sofl.Ticker` / `sofl.DefaultParams()` (Task 2) используются в Tasks 13,
14 без переименований; `rsipullbacksofl` — алиас импорта в пакете `backtest` (Tasks 2, 13);
`TestSOFLGridsStayWide` (Task 1) дополняется в Task 6 Step 4 и больше нигде не переопределяется;
`TestRSIPullbackSOFLTracksBaseline` (Task 2) заменяется на
`TestRSIPullbackSOFLServesTheCalibratedPoint` (Task 13);
`TestParamsTrackTheBaselineUntilCalibrated` (Task 2) заменяется на
`TestDefaultParamsIsTheCalibratedSnapshot` (Task 13).
