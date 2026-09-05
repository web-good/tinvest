# TRNFP под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести TRNFP (ПАО «Транснефть», привилегированные акции) до вердикта по стратегии
`rsi_pullback`: каталог максимально широких сеток, канонический тематический walk-forward, принятая
точка, прошедшая два риск-гейта, четыре пункта стоп-условия и дополнительный порог pooled OOS 1.10,
литерал в пакете и заведение в боевую вселенную двадцать шестым тикером.

**Architecture:** Процедура **каноническая**: одиннадцать тем поверх дефолтов ядра (baseline
торгует прибыльно — 155 сделок, PF 1.075 — поэтому якорь не нужен), схема **36/12/6** (четыре
фолда, проверено до написания спеки), плюс обязательный контрольный прогон **24/12/3** — сессия
внутри окна расширялась один раз (29 → 35 получасовых баров в буднем дне). Два риск-гейта: гейт
выживаемости применяется к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, потолок 1.0; гейт
просадки — max DD точки ≤ 1.3 × 7.30% = 9.49%. **Особенности TRNFP:** модель издержек впервые в
каталоге **консервативна** (настоящий круг 0.038% против 0.1% в модели), поэтому пятого пункта
стоп-условия нет; зато baseline стоит на нуле (pooled OOS 1.036 и 1.004 при expectancy 0.025%
капитала), поэтому добавлен **порог 1.10** на pooled OOS точки и обязательный замер экспозиции в
июльский дивидендный гэп.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-05-trnfp-rsi-pullback-prep-design.md`
**Замеры:** `reports/_analysis/trnfp_pullback_prep_measurements.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех прогонах; `-interval Minutes30` обязателен в каждой команде
  (дефолт CLI — `Hour1`, забытый флаг даёт чужие числа, а не ошибку).
- **Схема:** `-months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor`;
  у темы `screen` — `-min-trades 1`.
- **Расчётное окно 2023-09-05 … 2026-09-05, 36 месяцев, без обрезки** (сплит 1:100 внутри окна уже
  скорректирован в ряду и по цене, и по объёму).
- **Число фолдов сверяется на первой же теме:** «Фолдов: 3» в шапке отчёта темы `screen` —
  остановиться и доложить владельцу (ловушка ASTR).
- **`-refresh` НЕ запускать:** кэш дотянут 2026-09-05 (`TRNFP_Minutes30.json` — 36 439 баров,
  `TRNFP_Day1.json` — 1 134 свечи).
- **Сетки максимально широкие**; обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
  инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower`, ось тренда не порождает пар
  `EMAFast ≥ EMASlow`, `StopDailyATR` нигде не ноль, ось `RSIUpper` темы `exit` содержит 95, ось
  `EMASlow` темы `trend_low` содержит 12.
- **Исключены как вырожденные или доказанно отрицательные:** `RSILower` 5 (6 сделок за 36 месяцев),
  `RSIPeriod` 12 и 14 (31 и 19), `EMASlow` 8 (1.122/299 при убыточных соседях) и 10 (ноль сделок
  при равных периодах), `TPDailyATR` 0.1 (0.721/189) и 0.15 (0.828/176).
- **Правило сборки точки:** ≥ 3 фолда из 4 за значение, иначе дефолт ядра; ничья 2/2 не считается;
  `trend_low` побеждает только при большинстве ≥ 3/4 И превосходстве pooled OOS над `trend`; голоса
  `RSIUpper` из темы `entry` в сборку не идут.
- **РИСК-ГЕЙТ A:** защита достижима ≥ 30% будних дней; применяется к
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`. Таблица (n = 760): 0.3 — 99.9%, 0.4 — 96.7%,
  0.5 — 91.3%, 0.6 — 85.4%, 0.7 — 77.5%, 0.8 — 66.7%, **1.0 — 44.2%**, 1.2 — 27.8%, 1.3 — 21.8%,
  1.5 — 13.8%, 2.0 — 4.5%. **Потолок — 1.0.**
- **РИСК-ГЕЙТ B:** max DD точки на полном окне ≤ **9.49%** (1.3 × 7.30%).
- **ПРАВИЛО НИЧЬИХ:** при разнице pooled OOS < 0.05 в точку идёт вариант с меньшей просадкой.
- **Третий контур:** опорные числа baseline — доля SL 21.9%, удержание 9 / 21 / 45 баров, ночёвок
  52.3%, выходов в выходную сессию 11.0%, max DD 7.30%, **экспозиция в дивидендный гэп — ноль**.
- **Планка:** `entry` и `trend` (каноническая) обе дают pooled OOS PF ≥ 1.5 при ≥ 20 сделках;
  ведущая ось выбрана одинаково в ≥ 3 фолдах из 4. Ожидание записано заранее: ось входа рельефа не
  несёт (размах рабочей части 0.12 PF), планка почти наверняка провалится темой `entry`.
- **Стоп-условие из ЧЕТЫРЁХ пунктов:** (1) pooled OOS PF < 1.0 на 36/12/6; (2) < 20 сделок в пуле;
  (3) pooled OOS PF < 1.0 на 24/12/3; (4) то же при `-commission 0.001` (круг 0.2%). Пятого пункта
  по реальному кругу нет: настоящий круг TRNFP 0.038%, модель считает 0.1%, пункт 4 стрессует до
  0.2% — пятикратный запас над реальностью.
- **ДОПОЛНИТЕЛЬНОЕ УСЛОВИЕ TRNFP:** точка с pooled OOS PF на 36/12/6 **ниже 1.10** вердикта «в
  прод» не получает даже при непровалившемся стоп-условии. Основание: дефолты дают 1.036 при
  expectancy 13.36 ₽ на сделку. Порог объявлен до прогонов и не двигается.
- При срабатывании любого пункта или дополнительного условия — числа владельцу, **задачи 12–16 не
  выполняются**. Маршрут «дефолты ядра» (прецедент TGKA) недоступен: у дефолтов TRNFP нулевое
  математическое ожидание.
- **Дефолты ядра:** `RSIPeriod 4`, `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`,
  `DailyATRPeriod 14`, `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`,
  `TPDailyATR 0.6`, `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`,
  `UseRSIExit 1`, `UseTrail 0`, `TrailDailyATR 0`. Baseline на окне: **155 сделок, PF 1.075, net
  +3 863.85 ₽ (+3.86%)**, win rate 68.39%, max DD **7 596.98 ₽ (7.30%)**, exposure 4.61%,
  expectancy **24.93 ₽ = 0.025% капитала**, выходы RSI 113 / SL 34 (21.9%) / TP 8, удержание
  9 / 21 / 45, ночёвок 52.3%, выходов в выходную сессию 11.0%. Walk-forward дефолтов: 36/12/6 →
  **1.036/107** (фолды 0.898/26, 1.369/27, 1.097/35, 0.862/19); 24/12/3 → **1.004/54** (фолды
  0.748/18, 3.234/17, 1.150/13, 0.648/6).
- **Полугодия baseline:** 1.259 (22 сделки, +1 019 ₽), 1.489 (24, +2 892), **0.802** (27, −3 506),
  1.370 (27, +3 152), 1.136 (36, +1 194), **0.864** (19, −888).
- **Издержки:** шаг 0.2 ₽; круг **0.038%** при текущей цене 1062.6 ₽, 0.030% при медиане окна
  1352 ₽. Чувствительность baseline: 0.1% → 1.075, 0.2% → **0.791**, 0.3% → 0.569, 0.4% → 0.400.
- **Ликвидность:** медиана оборота будних дней 1008.23 млн ₽ (36 мес), 811.36 (24), **613.04 (12)**;
  по годам 1266.05 → 1218.85 → 943.01 → **624.52 (2026)**. Выходная сессия: 311 дней, медиана
  27.21 млн ₽ (p90 120.49). Дневной ATR(14) медиана **2.05%**.
- **Априор скринера** (2026-09-05): PFmed **1.28**, **Plateau 25%**, зажатых 0/24, молчащих 0/24,
  holdout PFmed 6.02 на 10 сделках, оборот 1010 млн ₽, ATR 2.54%, лучшая конфигурация RSI 6/10,
  EMA 20/100, TP 1.0.
- **Режим:** buy&hold −24.1%; полугодия +13.3, −20.5, −10.5, +16.0, +5.5, **−25.4**; просадка
  инструмента **−47.4%**. Дивидендные отсечки: −10.23% (2024-07-18), −10.61% (2025-07-17),
  **−15.22% (2026-07-18)**.
- **Каждый `_comment`** обязан содержать: что тема меряет и сколько прогонов; замеры осей с
  предупреждением о крае; полную команду запуска с путём `data/params/rsi_pullback/trnfp/<файл>`;
  место под строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-05: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md).
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле истории.
- **Ветка:** `feat/trnfp-pullback-prep` от `main` (`8208ab5`); спека закоммичена (`f831211`).

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/trnfp/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Test: `internal/service/backtest/rsi_pullback_trnfp_grid_test.go`

**Interfaces:**
- Produces: одиннадцать файлов сеток, которые читают задачи 3–10 и валидирует существующий
  `TestRSIPullbackCalFilesValid`.

- [ ] **Step 1:** написать падающий сторожевой тест осей по образцу
  `internal/service/backtest/rsi_pullback_spbe_grid_test.go`: `TestTRNFPGridsStayWide` (наличие
  обязательных значений по таблице §5.1 спеки), `TestTRNFPEntryGridKeepsRSIUpperAboveRSILower`,
  `TestTRNFPTrendGridsKeepFastBelowSlow`. В доке теста — четыре отступления от канона с их
  замерами: ось `exit` расширена **вверх** до 95 (максимум рельефа на 65–75, зона ниже 60
  убыточна); `trend_low` вниз до 12 (там максимум оси `EMASlow`, 1.372/129); ось цели не
  расширяется ниже 0.2 (ось инертна сверху, размах 0.04 PF, и валится снизу: 0.1 → 0.721/189);
  стоп широкий до 2.0, режет его гейт A, а не сетка.
- [ ] **Step 2:** убедиться, что тест падает.
  Run: `go test ./internal/service/backtest/ -run TestTRNFP` → FAIL (файлов сеток нет)
- [ ] **Step 3:** создать одиннадцать файлов сеток по таблице §5.1 спеки; замеры для `_comment`
  берутся из §4 спеки и протокола замеров. Оси:
  - `cal_screen.json`: `UseDayATRGate` [0,1] × `UseVolume` [0,1] — 4 прогона;
  - `cal_entry.json`: `RSIPeriod` [2,3,4,5,6,7,8,10] × `RSILower` [10,15,20,25,30,35,40,45,50] ×
    `RSIUpper` [55,60,65,70,75,80,85] — 504, `keepTop: 5`;
  - `cal_trend.json`: `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] — 48;
  - `cal_trend_low.json`: `EMAFast` [3,5,8,10] × `EMASlow` [12,15,20,30,40] — 20;
  - `cal_day.json`: `FreshDayATR` [0,0.1,0.2,0.3,0.4,0.5] × `SpentDayATR`
    [0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5] — 48;
  - `cal_day_spent.json`: `FreshDayATR` [0] × `SpentDayATR`
    [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0] — 10;
  - `cal_volume.json`: `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0] × `VolBaseDays`
    [3,5,10,14,20] — 30;
  - `cal_vol_window.json`: `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult`
    [1.0,1.2,2.0] — 27;
  - `cal_risk.json`: `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.2,1.3,1.5,2.0] × `TPDailyATR`
    [0.2,0.25,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.5] — 110;
  - `cal_exit.json`: `RSIUpper` [40,45,50,55,60,65,70,75,80,85,90,95] — 12;
  - `cal_trail.json`: `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.3,0.5,0.7,1.0,1.5]
    — 10.
- [ ] **Step 4:** Run: `go test ./internal/service/backtest/ -run 'TestTRNFP|TestRSIPullback'` → PASS
- [ ] **Step 5:** коммит `feat(rsi_pullback): каталог сеток TRNFP`.

---

### Task 2: Пакет `strategy/trnfp` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/trnfp/trnfp.go`, `trnfp_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Produces: `trnfp.Ticker` (константа `"TRNFP"`), `trnfp.DefaultParams() core.Params`,
  алиас реестра `rsipullbacktrnfp`. Задачи 12 и 13 опираются на эти два имени.

- [ ] **Step 1:** написать падающие тесты `TestParamsTrackTheBaselineUntilCalibrated` (все
  восемнадцать полей равны `core.DefaultParams()`) и `TestTickerIsTRNFP` по образцу пакета
  `strategy/spbe`.
- [ ] **Step 2:** убедиться, что падают.
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/trnfp/` → FAIL
- [ ] **Step 3:** создать пакет; в шапке (док-комментарий) — окно и схема; baseline с анатомией;
  walk-forward дефолтов (**1.036/107** и **1.004/54**) с прямой записью, что обе схемы стоят на
  нуле; оба риск-гейта с числами; четыре пункта стоп-условия и порог 1.10; априор скринера
  (PFmed 1.28, Plateau 25%); шаг цены 0.2 ₽ и круг 0.038% против 0.1% в модели; ликвидность
  1008 млн ₽ и её падение вдвое за окно; режим −24.1% с тремя растущими полугодиями; сплит 1:100
  внутри окна и проверка непрерывности объёма; дивидендные отсечки −10.23 / −10.61 / −15.22%.
- [ ] **Step 4:** завести тикер в реестр бэктеста (алфавитный порядок) + тест
  `TestRSIPullbackTRNFPTracksBaseline`.
- [ ] **Step 5:** Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 6:** коммит `feat(rsi_pullback): пакет TRNFP до калибровки`.

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_screen.json -out ./reports/TRNFP_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** проверить «Фолдов: 4» — иначе остановиться и доложить владельцу (ловушка ASTR).
- [ ] **Step 3:** выписать pooled OOS PF, сделки пула, пофолдовые `UseDayATRGate` / `UseVolume`,
  пофолдовые OOS PF / сделки / MaxDD%. Ожидание: `UseDayATRGate=1` почти наверняка (без гейта
  0.816/510, убыток); по `UseVolume` выбор свободный (лучшая форма даёт +0.15 PF ценой 23% сделок).
- [ ] **Step 4:** дописать `РЕЗУЛЬТАТ ПРОГОНА 2026-09-05: …` в `_comment`.
- [ ] **Step 5:** коммит `feat(rsi_pullback): TRNFP, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_entry.json -out ./reports/TRNFP_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать (504 прогона × 4 фолда).
- [ ] **Step 2:** планка по критериям A (pooled OOS PF ≥ 1.5 при ≥ 20 сделках) и B (`RSILower`
  одинаков в ≥ 3 фолдах). Записать «взят»/«провален» отдельно по каждому критерию.
- [ ] **Step 3:** проверить края: победа `RSIPeriod` 2 (0.916/404 точечно — убыточный край) или 10
  (1.143/**38**), победа `RSILower` 10 (1.770 на **29 сделках за три года**) пишется
  предупреждением прямым текстом.
- [ ] **Step 4:** отдельно записать, ушла ли пара от дефолта, и сверить с ожиданием §4.1: ось
  входа рельефа не несёт, размах рабочей части 0.12 PF, все максимумы поверхности стоят на
  6–34 сделках. Если тема выдаёт pooled OOS ≥ 1.5 — проверить, не сделан ли он одним фолдом.
- [ ] **Step 5:** дописать результат, коммит `feat(rsi_pullback): TRNFP, тема entry`.

---

### Task 5: Темы `trend` и `trend_low`

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_trend.json -out ./reports/TRNFP_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_trend_low.json -out ./reports/TRNFP_trend_low \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** планка — только по канонической `cal_trend`.
- [ ] **Step 3:** сравнить `trend_low` с `trend` по pooled OOS; решение прямым текстом (поле идёт
  из `trend_low` только при большинстве ≥ 3/4 И превосходстве pooled OOS).
- [ ] **Step 4:** проверить нижний край: победа `EMASlow` 12 означает неисчерпанную ось — снять
  зонд за краем одноточечной сеткой при `EMAFast` 3 (узел 10 при `EMAFast` 10 вырожден: равные
  периоды дают ноль сделок). Верхний край: победа 250 — снять зонд 300.
- [ ] **Step 5:** записать, что ось двугорбая (максимумы точечно на 12 и 200, дефолт во впадине),
  и какой горб выбрали фолды.
- [ ] **Step 6:** дописать результаты, коммит `feat(rsi_pullback): TRNFP, темы trend и trend_low`.

---

### Task 6: Темы `day` и `day_spent`

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_day.json -out ./reports/TRNFP_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_day_spent.json -out ./reports/TRNFP_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** сверить темы между собой; расхождение пофолдовых `SpentDayATR` — взаимодействие
  веток гейта, пишется прямым текстом.
- [ ] **Step 3:** проверить края: победа `SpentDayATR` 1.25 (38 сделок) или 1.5 (18) — уход в
  вырожденную выборку, записывается предупреждением. По `FreshDayATR` ось монотонно валится от
  дефолта (0 → 1.075, 0.6 → 0.762), победа любого ненулевого узла — сигнал переоптимизации фолда.
- [ ] **Step 4:** **отдельно записать связь с дивидендным гэпом:** ослабление `SpentDayATR` вверх
  (0.9 и выше) открывает вход в день с гэпом; это идёт в Task 11 Step 8 как предупреждение для
  замера гэповой экспозиции.
- [ ] **Step 5:** дописать результаты, коммит `feat(rsi_pullback): TRNFP, темы day и day_spent`.

---

### Task 7: Тема `volume`

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_volume.json -out ./reports/TRNFP_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** выписать пофолдовых победителей `VolMult` / `VolBaseDays`; проверить победу краёв
  (3 или 20 по `VolBaseDays`, 1.0 или 3.0 по `VolMult`).
- [ ] **Step 3:** записать место TRNFP в каталожной гипотезе «объём говорит на тонких бумагах»:
  **TRNFP — самая ликвидная бумага каталога** (медиана 613 млн ₽ за 12 месяцев против 14 млн у
  NKHP, где гейт побеждал единогласно). Гипотеза предсказывает, что объёмный гейт здесь проиграет;
  результат записать как подтверждение или опровержение.
- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): TRNFP, тема volume`.

---

### Task 8: Тема `vol_window`

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_vol_window.json -out ./reports/TRNFP_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** сверить `VolMult` с темой `volume`; расхождение = форма гейта не откалибрована,
  `VolMult` идёт дефолтом ядра.
- [ ] **Step 3:** проверить края. Точечно максимум оси стоит на `VolLookbackBars` **2** — не на
  краю, в отличие от RTKMP и NKHP; победа края 1 или 32 записывается предупреждением прямым
  текстом.
- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): TRNFP, тема vol_window`.

---

### Task 9: Тема `risk` и обязательная проверка риск-гейта A

**Files:** также `data/params/rsi_pullback/trnfp/plateau_stop_10.json`,
`data/params/rsi_pullback/trnfp/plateau_stop_15.json`

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_risk.json -out ./reports/TRNFP_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** применить риск-гейт A: победитель `StopDailyATR` > 1.0 отвергается, в точку идёт
  1.0 (44.2% дней), цена решения в PF — прямым текстом. Ожидание высокое: точечно максимум оси
  стоит на 2.0 (1.757/151), а 1.5 даёт 1.756/151 при выживаемости 13.8%.
- [ ] **Step 3:** снять два зонда капкана (`plateau_stop_10.json` — стоп 1.0, `plateau_stop_15.json`
  — стоп 1.5, остальные поля дефолтные) одиночными прогонами на полном окне:
  `-months 36` без `-train-months`; выписать PF, сделки, max DD, выходы с долей SL, удержание,
  ночёвки, выходы в выходную сессию.
- [ ] **Step 4:** записать форму капкана прямым текстом: пул стоит с 0.5 (155 → 151), PF растёт в
  1.6 раза. **Проверить, растёт ли просадка**: если падает, гейт B бессилен (случай RTKMP и SPBE) и
  режет только гейт A; если растёт — гейт B работает как второй фильтр.
- [ ] **Step 5:** записать поведение оси цели: точечно она инертна сверху (0.4…2.5 в полосе
  1.036–1.075). Победа далёкой цели (1.5 или 2.5) означает, что цель фактически отключена — это
  записывается как «цель не работает», а не как выбор.
- [ ] **Step 6:** дописать результат, коммит
  `feat(rsi_pullback): TRNFP, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit` и `trail`

```bash
go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_exit.json -out ./reports/TRNFP_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker TRNFP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/trnfp/cal_trail.json -out ./reports/TRNFP_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы. `exit` на TRNFP — сильнейшая ось бумаги (размах 0.58 PF).
- [ ] **Step 2:** проверить верхний край: победа 90 или 95 означает, что ось не исчерпана вверх —
  снять зонд 100 одноточечной сеткой. Победа узла ниже 60 противоречит точечному рельефу (вся зона
  убыточна, 32 → 0.594) и записывается как переоптимизация фолда.
- [ ] **Step 3:** проверить, не обходит ли трейл риск-гейт A (урок AFKS): гейт применяется к
  `min(StopDailyATR, TrailDailyATR)`.
- [ ] **Step 4:** правило упрощения: дистанция ≥ 1.0 → `UseTrail 0`, `TrailDailyATR 0` (точечно
  трейл 1.0 и 1.5 побайтово равны его отсутствию).
- [ ] **Step 5:** если тема выбрала `UseRSIExit 0` — записать отдельной строкой: точечно
  0.933/137 против 1.075/155.
- [ ] **Step 6:** записать ожидание §4.6 и его исход: трейл на TRNFP точечно только вредит
  (0.3 → 0.916, 0.5 → 1.014, 0.7 → 1.048 против 1.075 у дефолта).
- [ ] **Step 7:** дописать результаты, коммит `feat(rsi_pullback): TRNFP, темы exit и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и четыре пункта стоп-условия

**Files:** `data/params/rsi_pullback/trnfp/plateau_point.json`, файлы соседей плато
(`plateau_<поле>_<значение>.json`), `docs/superpowers/plans/task-11-report-trnfp.md`

- [ ] **Step 1:** собрать точку правилом большинства (≥ 3 из 4; иначе дефолт ядра); для каждого из
  восемнадцати полей записать голоса фолдов и решение, отмечая случайные совпадения с дефолтом.
- [ ] **Step 2:** применить риск-гейт A (эффективная защита, потолок 1.0).
- [ ] **Step 3:** прогнать точку схемой 36/12/6 (`-out ./reports/TRNFP_point_oos`); выписать pooled
  OOS PF, сделки, пофолдовые in-sample → OOS PF / сделки / NetPnL% / MaxDD%. Baseline — 1.036/107.
- [ ] **Step 4:** проверить пункты 1 и 2 стоп-условия **и дополнительное условие TRNFP
  (pooled OOS ≥ 1.10)**. Провал → **остановиться**, задачи 12–16 не выполнять.
- [ ] **Step 5:** контрольная схема 24/12/3 (`-months 24 -train-months 12 -test-months 3`,
  `-out ./reports/TRNFP_point_24`) — пункт 3. Baseline на том же куске даёт **1.004**: точка
  обязана быть заметно лучше дефолтов, а не «не хуже».
- [ ] **Step 6:** пункт 4 — тот же прогон 36/12/6 при `-commission 0.001`
  (`-out ./reports/TRNFP_point_cost2`). Дополнительно снять справочную строку `-commission 0.0015`
  (`-out ./reports/TRNFP_point_cost3`) — при настоящем круге 0.038% это запас в четыре раза.
- [ ] **Step 7:** риск-гейт B на полной истории (`-out ./reports/TRNFP_point_full`), потолок max DD
  **9.49%**.
- [ ] **Step 8:** анатомия и полугодия точки против шести чисел третьего контура. **Обязательный
  замер дивидендной экспозиции:** по журналу сделок прогона полной истории проверить каждую из
  трёх ночей отсечки (2024-07-18, 2025-07-17, 2026-07-18) — держала ли точка позицию через ночь и
  входила ли в сам день отсечки. У baseline: держала ноль раз, входила один раз (2025-07-17).
  Удержание хоть через одну ночь → записать как реализованный риск и взять более узкий стоп.
  Проверить подпись капкана (падение доли SL при росте удержания и ночёвок → более узкий стоп).
- [ ] **Step 9:** соседи плато по каждому полю, ушедшему от дефолта (±один узел оси), плюс правило
  разрешения ничьих (разница pooled OOS < 0.05 → меньшая просадка).
- [ ] **Step 10:** написать `docs/superpowers/plans/task-11-report-trnfp.md` со всеми выкладками,
  командами и таблицами.
- [ ] **Step 11:** коммит `feat(rsi_pullback): TRNFP, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Задача выполняется только если ни один пункт стоп-условия не сработал и pooled OOS точки ≥ 1.10.**

- [ ] **Step 1:** заменить `TestParamsTrackTheBaselineUntilCalibrated` на
  `TestParamsAreTheAcceptedPoint` (снимок восемнадцати полей) плюс
  `TestPointDiffersFromTheCoreBaseline`.
- [ ] **Step 2:** убедиться, что тесты падают.
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/trnfp/` → FAIL
- [ ] **Step 3:** поставить литерал принятой точки в `DefaultParams()`.
- [ ] **Step 4:** заменить тест реестра на `TestRSIPullbackTRNFPServesTheCalibratedPoint`.
- [ ] **Step 5:** Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... && ./bin/golangci-lint run ./internal/...` → PASS
- [ ] **Step 6:** коммит `feat(rsi_pullback): TRNFP откалиброван — литерал вместо отслеживания baseline`.

---

### Task 13: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`

- [ ] **Step 1:** добавить импорт и строку `trnfp.Ticker: trnfp.DefaultParams(),` в `paramsByTicker`.
- [ ] **Step 2:** дописать абзац комментария карты в англоязычном стиле соседей: вердикт по планке,
  оба риск-гейта и их цена, отсутствие пятого пункта стоп-условия (модель издержек консервативна
  втрое), порог 1.10 и принятые риски: дивидендный гэп −10…−15% три года подряд в середине июля,
  падение оборота вдвое за окно (1266 → 625 млн ₽), выходная сессия 11.0% выходов при обороте
  27 млн ₽, сплит 1:100 внутри расчётного окна.
- [ ] **Step 3:** Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/...` → PASS
- [ ] **Step 4:** коммит `feat(rsi_pullback): TRNFP в реестре живого раннера`.

---

### Task 14: Боевая вселенная

**Files:** Modify `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`,
`env/prod.env`, `env/prod.env.example`, `env/local.env.example`, `docs/rsi_pullback/live.md`

- [ ] **Step 1:** перечитать числа Task 11 — ни один из четырёх пунктов не сработал, порог 1.10
  выполнен, оба гейта пройдены.
- [ ] **Step 2:** добавить `"TRNFP"` в `want` теста конфига двадцать шестым.
- [ ] **Step 3:** Run: `go test ./internal/config/ -run RSIPullback` → FAIL
- [ ] **Step 4:** дописать `"TRNFP"` в `Tickers` + комментарий-абзац; дописать `,TRNFP` в
  `env/prod.env`, `env/prod.env.example`, `env/local.env.example`.
- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  `docs/rsi_pullback/live.md`.
- [ ] **Step 6:** Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 7:** коммит `feat(rsi_pullback): завести TRNFP в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

- [ ] **Step 1:** переписать шапку пакета в итог калибровки: вердикт по планке по каждому критерию
  обеих ключевых тем, точка поле за полем с голосами, оба walk-forward со сравнением с baseline
  (1.036/107 и 1.004/54), строки издержек (0.1% / 0.2% / 0.15% справочно / настоящий круг 0.038%),
  соседи плато, шесть полугодий, анатомия против baseline, **замер дивидендной экспозиции по трём
  ночам отсечки**, результат обоих риск-гейтов и их цена. Отдельными абзацами — принятые риски и
  условия пересмотра из §7 спеки (нулевой baseline и порог 1.10; отсутствие рельефа на оси входа;
  дивидендный гэп; капкан широкого стопа; сплит внутри окна и обязательная проверка непрерывности
  объёма при любой перекалибровке; падение оборота с порогом пересмотра 100 млн ₽ за 6 месяцев;
  выходная сессия; расширение сессии).
- [ ] **Step 2:** Run: `go test ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 3:** механический вывод в `docs/rsi_pullback/strategy.md` — **только если** TRNFP дал
  вывод о механике. Кандидат назван заранее: **ликвидность не создаёт edge** (оборот на порядок
  выше каталога при самом слабом baseline в нём). Пер-тикерных чисел, дат и вердиктов в
  `docs/rsi_pullback/` не писать.
- [ ] **Step 4:** коммит `docs(rsi_pullback): разбор калибровки TRNFP и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** Run: `go run ./cmd/pullparity -tickers TRNFP -months 24` → ноль расхождений
  (на 24 месяцах, не на 36 — урок VSMO).
- [ ] **Step 2:** Run: `./bin/mage ci` → зелёный
- [ ] **Step 3:** Run: `git status --porcelain | grep -c '^.. reports/'` → `0`
- [ ] **Step 4:** финальный коммит, если остались правки.

---

## Self-Review

**Покрытие спеки:** §1 → Global Constraints и Task 2 Step 3; §2 → Global Constraints и
Architecture; §3 → Global Constraints и Task 11 Step 8; §4 → `_comment` в Task 1 Step 3 и ожидания
Tasks 3–10; §5.1 → Tasks 3–10 и Task 1 (сторожевые тесты); §5.2 → Task 11 Step 1; §5.3 →
Task 11 Step 9; §5.4 → Task 9 Step 2, Task 10 Step 3, Task 11 Step 2; §5.5 → Task 11 Steps 7 и 9;
§5.6 (включая дивидендную экспозицию) → Task 11 Step 8 и Task 6 Step 4; §5.7 → Tasks 4 и 5;
§5.8 (четыре пункта + порог 1.10) → Task 11 Steps 4–6; §5.9 → Task 14 Step 1; §6 → Tasks 1, 2,
11–15; §7 → Task 15 Step 1; `pullparity` и `mage ci` → Task 16; ловушка ASTR → Task 3 Step 2.

**Плейсхолдеры:** один — литерал в Task 12 (точка известна только после Task 11). Все прочие шаги
несут конкретные оси, команды и пороги.

**Согласованность имён:** `trnfp.Ticker` и `trnfp.DefaultParams()` заводятся в Task 2 и
используются в Tasks 12 и 13; файлы сеток из Task 1 читаются задачами 3–10 по тем же путям;
`plateau_stop_10.json` и `plateau_stop_15.json` заводятся в Task 9 и упоминаются только там;
`plateau_point.json` — Task 11; тесты `TestTRNFPGridsStayWide`,
`TestRSIPullbackTRNFPTracksBaseline`, `TestParamsTrackTheBaselineUntilCalibrated` (Task 2) →
`TestParamsAreTheAcceptedPoint` и `TestRSIPullbackTRNFPServesTheCalibratedPoint` (Task 12).
