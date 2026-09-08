# MVID под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести MVID (ПАО «М.видео») до вердикта по стратегии `rsi_pullback`: каталог
максимально широких сеток, канонический тематический walk-forward, принятая точка, прошедшая два
риск-гейта и пять пунктов стоп-условия, литерал в пакете и заведение в боевую вселенную двадцать
шестым тикером.

**Architecture:** Процедура **каноническая**: тринадцать тем поверх дефолтов ядра (baseline торгует
прибыльно — 149 сделок, PF 1.179, pooled OOS 1.326 — якорь не нужен), схема **36/12/6** (четыре
фолда, проверено до написания спеки), плюс обязательный контрольный прогон **24/12/3** — сессия
внутри окна расширялась один раз (29 → 35 получасовых баров в буднем дне). Два риск-гейта: гейт
выживаемости применяется к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, потолок 1.0; гейт
просадки — **жёсткий**, max DD точки ≤ max DD baseline 11.10%. **Особенности MVID:** модель издержек
оптимистична в 2.2 раза (настоящий круг 0.217% при шаге 0.05 ₽ и цене 46.25 ₽), поэтому добавлен
**пятый пункт стоп-условия** по реальному кругу; ось входа несёт сильнейший рельеф каталога
(размах 1.6 PF, монотонный), поэтому заведена уплотняющая тема `entry_deep`; горб тренда и зона
раннего выхода вынесены в темы `trend_hump` и `exit_low`, потому что каноническим темам они
недостижимы без вырожденных пар.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-07-mvid-rsi-pullback-prep-design.md`
**Замеры:** `reports/_analysis/mvid_pullback_prep_measurements.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех прогонах; `-interval Minutes30` обязателен в каждой команде
  (дефолт CLI — `Hour1`, забытый флаг даёт чужие числа, а не ошибку).
- **Схема:** `-months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor`;
  у темы `screen` — `-min-trades 1`.
- **Расчётное окно 2023-09-07 … 2026-09-07, 36 месяцев, без обрезки.** Сплитов, консолидаций и
  остановок торгов внутри окна нет; провал июля 2026 (57.65 → 36.40 ₽) — непрерывная распродажа без
  гэпов, ряд корректировать не нужно.
- **Число фолдов сверяется на первой же теме:** «Фолдов: 3» в шапке отчёта темы `screen` —
  остановиться и доложить владельцу (ловушка ASTR).
- **`-refresh` НЕ запускать:** кэш дотянут 2026-09-07 (`MVID_Minutes30.json`, `MVID_Day1.json`).
- **Сетки максимально широкие**; обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
  инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower` на всех парах тем
  `entry_deep` и `exit_low`, ни одна из двух трендовых тем не порождает пар `EMAFast ≥ EMASlow`,
  `StopDailyATR` нигде не ноль, ось `RSILower` темы `entry` содержит 5, ось `RSIUpper` темы
  `exit_low` содержит 20, ось `VolBaseDays` содержит 30, ось `EMASlow` темы `trend_hump` содержит
  25, 30, 35, 40, 45, ось `EMASlow` темы `trend` содержит 250.
- **Исключены как вырожденные или доказанно инертные:** `TPDailyATR` 3.0 и выше (ось инертна с 1.5:
  145 сделок и PF 1.050 побайтово на 1.5, 2.0, 2.5, 3.0), `EMASlow` 12 (1.090/110 — ниже горба и
  ниже дефолта), `RSIPeriod` 14 и выше (на 12 уже 24 сделки за три года).
- **Правило сборки точки:** ≥ 3 фолда из 4 за значение, иначе дефолт ядра; ничья 2/2 не считается;
  `entry_deep` побеждает только при большинстве ≥ 3/4 И превосходстве pooled OOS над `entry`;
  `trend_hump` — только при большинстве ≥ 3/4 И превосходстве pooled OOS над `trend`; голоса
  `RSIUpper` из обеих тем входа в сборку не идут; голоса `RSIUpper` из `exit_low` идут только при
  принятом `RSILower ≤ 20`, и при конфликте с канонической `exit` побеждает тема с большим pooled
  OOS.
- **РИСК-ГЕЙТ A:** защита достижима ≥ 30% будних дней; применяется к
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`. Таблица (n = 762): 0.3 — 98.3%, 0.4 — 94.0%,
  0.5 — 86.2%, 0.6 — 75.9%, 0.7 — 65.4%, 0.8 — 54.6%, 0.9 — 44.6%, **1.0 — 36.0%**, 1.2 — 21.0%,
  1.5 — 12.3%, 2.0 — 4.6%. **Потолок — 1.0.**
- **РИСК-ГЕЙТ B (жёсткий):** max DD точки на полном окне ≤ **11.10%** (max DD baseline), а не
  1.3 × baseline. Основание: капкан широкого стопа начинается на 0.7, то есть внутри зоны,
  разрешённой гейтом A.
- **ПРАВИЛО НИЧЬИХ:** при разнице pooled OOS < 0.05 в точку идёт вариант с меньшей просадкой.
- **Третий контур:** опорные числа baseline — доля SL **28.2%**, удержание **9 / 20 / 56** баров,
  ночёвок **40.3%**, выходов в выходную сессию **6.7%**, max DD **11.10%**, expectancy
  **127.42 ₽**. Замер дивидендной экспозиции **не требуется** — дивидендов у MVID в окне нет.
- **Планка:** `entry` и `trend` (канонические, не `entry_deep` и не `trend_hump`) обе дают pooled
  OOS PF ≥ 1.5 при ≥ 20 сделках; ведущая ось (`RSILower` для `entry`, `EMASlow` для `trend`)
  выбрана одинаково в ≥ 3 фолдах из 4. Ожидание записано заранее: ось входа несёт сильнейший рельеф
  каталога (размах 1.6 PF, монотонный), поэтому шанс взять планку темой `entry` у MVID выше, чем у
  любого из последних тикеров; ось тренда — горб шириной 0.16 PF, по ней планка менее вероятна.
- **Стоп-условие из ПЯТИ пунктов:** (1) pooled OOS PF < 1.0 на 36/12/6; (2) < 20 сделок в пуле;
  (3) pooled OOS PF < 1.0 на 24/12/3; (4) то же при `-commission 0.001` (круг 0.2%); (5) то же при
  **`-commission 0.0011`** (реальный круг MVID 0.217%) — решение владельца, принятое до прогонов.
- **ДОПОЛНИТЕЛЬНОЕ УСЛОВИЕ MVID:** точка с pooled OOS PF на 36/12/6 **ниже 1.326** (уровень
  дефолтов ядра) литералом в прод не заводится. Маршрут «дефолты ядра» (прецедент TGKA) формально
  существует, но закрыт: дефолты проваливают пункт 5 (PF 0.995 при реальном круге).
- При срабатывании любого пункта или дополнительного условия — числа владельцу, **задачи 12–16 не
  выполняются**; маршрут (второй круг по прецеденту SOFL или закрытие по прецедентам HEAD, AFKS,
  UWGN, RTKMP, TRNFP) выбирает владелец.
- **Дефолты ядра:** `RSIPeriod 4`, `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`,
  `DailyATRPeriod 14`, `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`,
  `TPDailyATR 0.6`, `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`,
  `UseRSIExit 1`, `UseTrail 0`, `TrailDailyATR 0`. Baseline на окне: **149 сделок, PF 1.179, net
  +18 985.36 ₽ (+18.99%)**, win rate 62.42%, max DD **12 169.24 ₽ (11.10%)**, exposure 4.30%,
  CAGR 5.96%, expectancy **127.42 ₽ = 0.127% капитала**, выходы RSI 84 (56.4%) / SL 42 (28.2%) /
  TP 23 (15.4%), удержание 9 / 20 / 56, ночёвок 40.3%, входов в выходную сессию 0, выходов 6.7%.
  Walk-forward дефолтов: 36/12/6 → **1.326/100** (фолды 1.708/28, **0.409**/16, 2.048/27,
  1.098/29); 24/12/3 → **1.469/56** (фолды 1.529/10, 2.560/17, **0.784**/14, 1.308/15).
- **Полугодия baseline:** **0.864** (29 сделок, −1 594 ₽), 1.008 (19, +101), 1.413 (29, +9 440),
  **0.409** (16, −9 088), **2.017** (26, +16 969), 1.117 (30, +3 157).
- **Издержки:** шаг **0.05 ₽**; круг **0.217%** при цене 46.25 ₽, 0.153% при медиане 12 месяцев
  65.55 ₽, 0.104% при медиане окна 96.1 ₽; модель считает 0.1%. Чувствительность baseline:
  0.1% → 1.179; 0.2% → 1.024; **0.217% → 0.995**; 0.3% → 0.884.
- **Ликвидность:** медиана оборота будних дней 83.93 млн ₽ (36 мес), 65.52 (24), **41.40 (12)**,
  **32.64 (6)**; по годам 89.80 → 165.38 → 63.63 → **33.48 (2026)**. Выходная сессия: 314 дней,
  медиана **3.25 млн ₽**. Дневной ATR(14) медиана **3.54%** (p10 1.78%, p90 7.75%).
- **Априор скринера не снимался:** кандидат выбран владельцем, а не шортлистом `cmd/pullscreen`.
  Ссылок на априор в `_comment` и доке пакета не делать.
- **Режим:** buy&hold **−76.8%** (198.5 → 46.25 ₽); полугодия +1.9, −49.6, +9.7, −18.2, −22.7,
  **−35.9**; растущих два из шести. Дивидендов в окне нет; худшие открытия будних дней −4.58%,
  −4.06%, −2.89%, экспозиция baseline через каждое — ноль.
- **Каждый `_comment`** обязан содержать: что тема меряет и сколько прогонов; замеры осей с
  предупреждением о крае; полную команду запуска с путём `data/params/rsi_pullback/mvid/<файл>`;
  место под строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-07: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md).
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле истории.
- **Ветка:** `feat/mvid-pullback-prep` от `main` (`a362cc6`); спека закоммичена (`092c200`).

---

### Task 1: Каталог тринадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/mvid/cal_screen.json`, `cal_entry.json`, `cal_entry_deep.json`,
  `cal_trend.json`, `cal_trend_hump.json`, `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_exit_low.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_mvid_grid_test.go`

**Interfaces:**
- Produces: тринадцать путей `data/params/rsi_pullback/mvid/cal_*.json`, которые читают задачи 3–10;
  тест `TestMVIDGridsStayWide`.

- [ ] **Step 1:** написать падающий сторожевой тест осей по образцу
  `internal/service/backtest/rsi_pullback_trnfp_grid_test.go`. Тест `TestMVIDGridsStayWide` читает
  файлы каталога и проверяет **все** инварианты из Global Constraints: `RSILower ≤ 50` и наличие
  узла 5 в `cal_entry.json`; `RSIPeriod ≥ 2` везде; `RSIUpper > RSILower` на всех парах
  `cal_entry_deep.json` и `cal_exit_low.json`; отсутствие пар `EMAFast ≥ EMASlow` в
  `cal_trend.json` и `cal_trend_hump.json`; `StopDailyATR != 0` везде; наличие узла 20 на оси
  `RSIUpper` в `cal_exit_low.json`; наличие узла 30 на оси `VolBaseDays` в `cal_volume.json`;
  наличие узлов 25, 30, 35, 40, 45 на оси `EMASlow` в `cal_trend_hump.json`; наличие узла 250 на
  оси `EMASlow` в `cal_trend.json`.
- [ ] **Step 2:** убедиться, что тест падает.
  Run: `go test ./internal/service/backtest/ -run TestMVIDGridsStayWide` → FAIL (файлов нет)
- [ ] **Step 3:** создать тринадцать файлов сеток по таблице §5.1 спеки. Оси дословно:
  - `cal_screen.json`: `UseDayATRGate` [0,1] × `UseVolume` [0,1] — 4 прогона.
  - `cal_entry.json`: `RSIPeriod` [2,3,4,5,6,7,8,10,12] × `RSILower` [5,10,15,20,25,30,35,40,45,50]
    — 90.
  - `cal_entry_deep.json`: `RSIPeriod` [2,3,4,5,6] × `RSILower` [8,10,12,15,18,20] × `RSIUpper`
    [40,55,65,70] — 120.
  - `cal_trend.json`: `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] — 48.
  - `cal_trend_hump.json`: `EMAFast` [3,5,8,10] × `EMASlow` [15,20,25,30,35,40,45] — 28.
  - `cal_day.json`: `FreshDayATR` [0,0.1,0.2,0.3,0.4,0.5] × `SpentDayATR`
    [0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0] — 54.
  - `cal_day_spent.json`: `FreshDayATR` [0] × `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0]
    — 10.
  - `cal_volume.json`: `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0] × `VolBaseDays`
    [3,5,10,14,20,30] — 36.
  - `cal_vol_window.json`: `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult`
    [1.2,2.0] — 18.
  - `cal_risk.json`: `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.2,1.5,2.0] × `TPDailyATR`
    [0.2,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.0,2.5] — 100.
  - `cal_exit.json`: `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90,95] — 13.
  - `cal_exit_low.json`: `RSILower` [15] × `RSIUpper` [20,25,30,35,40,45] — 6.
  - `cal_trail.json`: `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.3,0.5,0.7,1.0,1.5] — 10.

  В `_comment` каждого файла — что тема меряет, число прогонов, замер оси из §4 спеки с
  предупреждением о крае, полная команда запуска, называющая **этот же** файл (этого требует
  `TestRSIPullbackCalFilesValid`), и место под `РЕЗУЛЬТАТ ПРОГОНА 2026-09-07: …`. Обязательные
  предупреждения о краях: узел `RSILower` 5 вырожден (10 сделок за три года); узел `RSIPeriod` 12
  вырожден (24 сделки); `VolBaseDays` 30 — зонд за каноническим краем 20, взявшим максимум;
  `TPDailyATR` 1.5–2.5 — инертная зона (145 сделок и 1.050 побайтово).
- [ ] **Step 4:** Run: `go test ./internal/service/backtest/ -run 'TestMVID|TestRSIPullback'` → PASS
- [ ] **Step 5:** коммит `feat(rsi_pullback): каталог сеток MVID`.

---

### Task 2: Пакет `strategy/mvid` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/mvid/mvid.go`,
  `internal/service/trading_strategy/rsi_pullback/strategy/mvid/mvid_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Produces: `mvid.Ticker` (константа `"MVID"`) и `mvid.DefaultParams() core.Params` — их используют
  задачи 12 и 13.

- [ ] **Step 1:** написать падающие тесты по образцу пакета `trnfp`:
  `TestParamsTrackTheBaselineUntilCalibrated` (`DefaultParams()` равен `core.DefaultParams()` поле
  за полем) и `TestTickerIsMVID`.
- [ ] **Step 2:** убедиться, что падают.
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/mvid/` → FAIL (пакета нет)
- [ ] **Step 3:** создать пакет. В док-комментарии (англоязычном, в стиле соседей) — расчётное окно
  и схема; baseline с анатомией (149 сделок, PF 1.179, max DD 11.10%, выходы RSI 56.4% / SL 28.2% /
  TP 15.4%, удержание 9 / 20 / 56, ночёвок 40.3%, выходов в выходную сессию 6.7%); walk-forward
  дефолтов (1.326/100 и 1.469/56); издержки (шаг 0.05 ₽, круг 0.217% против 0.1% модели,
  чувствительность 1.179 / 1.024 / 0.995 / 0.884); состояние «калибровка не проводилась».
- [ ] **Step 4:** завести тикер в реестр бэктеста (импорт `rsipullbackmvid` в алфавитном порядке,
  строка в `rsiPullbackRegistry`) + тест `TestRSIPullbackMVIDTracksBaseline`.
- [ ] **Step 5:** Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 6:** коммит `feat(rsi_pullback): пакет MVID до калибровки`.

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/mvid/cal_screen.json` (строка результата)

- [ ] **Step 1:** прогнать тему.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_screen.json -out ./reports/MVID_screen \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** проверить «Фолдов: 4» в шапке отчёта — иначе остановиться и доложить владельцу
  (ловушка ASTR).
- [ ] **Step 3:** выписать pooled OOS PF, сделки пула, пофолдовых победителей `UseDayATRGate` и
  `UseVolume`, число сделок каждой из четырёх комбинаций. Ожидание §4.4: весь массив темы `volume`
  лежит выше baseline, поэтому единогласие за `UseVolume = 1` здесь вероятно.
- [ ] **Step 4:** дописать `РЕЗУЛЬТАТ ПРОГОНА 2026-09-07: …` в `_comment`.
- [ ] **Step 5:** коммит `feat(rsi_pullback): MVID, тема screen — цена гейтов`.

---

### Task 4: Темы `entry` и `entry_deep` — ключевые

**Files:** Modify `cal_entry.json`, `cal_entry_deep.json` (строки результата)

- [ ] **Step 1:** прогнать обе темы (90 и 120 прогонов × 4 фолда).

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_entry.json -out ./reports/MVID_entry \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_entry_deep.json -out ./reports/MVID_entry_deep \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** планка — **только по канонической `cal_entry.json`**, по критериям A (pooled OOS
  PF ≥ 1.5 при ≥ 20 сделках в пуле) и B (`RSILower` выбран одинаково в ≥ 3 фолдах из 4). Вырожденный
  фолд в пользу тикера не засчитывается.
- [ ] **Step 3:** сравнить `entry_deep` с `entry` по pooled OOS; решение записать прямым текстом
  (поля входа идут из `entry_deep` только при большинстве ≥ 3/4 И превосходстве pooled OOS).
  Голоса `RSIUpper` из `entry_deep` в сборку **не идут** — там ось контекстная.
- [ ] **Step 4:** проверить края: победа `RSILower` 5 (точечно 1.119 на **10** сделках за три года)
  или `RSIPeriod` 12 (1.292 на **24**) — вырожденный узел, записать предупреждением; победа
  `RSILower` 50 (0.989/269) означала бы разворот измеренного рельефа на фолдах и требует отдельной
  записи.
- [ ] **Step 5:** сверить с ожиданием §4.1: ось `RSILower` монотонна от 10 до 50 с размахом 1.6 PF,
  `RSIPeriod` структуры не несёт. Записать, подтвердили ли фолды монотонность или разошлись с ней.
- [ ] **Step 6:** дописать результаты, коммит `feat(rsi_pullback): MVID, темы entry и entry_deep`.

---

### Task 5: Темы `trend` и `trend_hump`

**Files:** Modify `cal_trend.json`, `cal_trend_hump.json` (строки результата)

- [ ] **Step 1:** прогнать обе темы.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_trend.json -out ./reports/MVID_trend \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_trend_hump.json -out ./reports/MVID_trend_hump \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** планка — только по канонической `cal_trend.json` (ведущая ось `EMASlow`).
- [ ] **Step 3:** сравнить `trend_hump` с `trend` по pooled OOS; поле тренда идёт из `trend_hump`
  только при большинстве ≥ 3/4 И превосходстве pooled OOS. Решение — прямым текстом.
- [ ] **Step 4:** проверить края: победа `EMASlow` 15 (точечно 1.114/119 — ниже дефолта) означает
  неисчерпанную ось вниз, но узел 12 измерен и хуже (1.090/110) — записать это как ответ на
  вопрос о нижнем крае; победа `EMASlow` 250 (1.048/160) означает неисчерпанную ось вверх и требует
  зонда за краем.
- [ ] **Step 5:** записать сверку с ожиданием §4.2: точечно горб стоит на 30–40 (1.309 и 1.338) при
  дефолте 100 (1.179), ось `EMAFast` пологая с размахом 0.12 PF.
- [ ] **Step 6:** дописать результаты, коммит `feat(rsi_pullback): MVID, темы trend и trend_hump`.

---

### Task 6: Темы `day` и `day_spent`

**Files:** Modify `cal_day.json`, `cal_day_spent.json` (строки результата)

- [ ] **Step 1:** прогнать обе темы.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_day.json -out ./reports/MVID_day \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_day_spent.json -out ./reports/MVID_day_spent \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** сверить темы между собой; расхождение пофолдовых `SpentDayATR` означает
  взаимодействие веток гейта — записать прямым текстом, какая тема в каком фолде что выбрала.
- [ ] **Step 3:** проверить края: победа `SpentDayATR` 1.5 (точечно 1.451 на **35** сделках за три
  года) или 2.0 (1.418 на **16**) — уход в редкость, записать предупреждением; узел 1.0
  (1.416 на 96) — тот же уровень PF при втрое большем пуле, и правило ничьих (разница < 0.05)
  разрешается в его пользу.
- [ ] **Step 4:** записать исход ранней ветки: точечно `FreshDayATR` в изоляции убыточен почти
  везде (0.3 → 0.904/90, 0.4 → 0.972/149), прибыль только на вырожденных узлах (0.1 → 1.176/30).
  Победа `FreshDayATR > 0` в трёх фолдах требует отдельной записи как расхождение с замером.
- [ ] **Step 5:** дописать результаты, коммит `feat(rsi_pullback): MVID, темы day и day_spent`.

---

### Task 7: Тема `volume`

**Files:** Modify `cal_volume.json` (строка результата)

- [ ] **Step 1:** прогнать тему.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_volume.json -out ./reports/MVID_volume \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** выписать пофолдовых победителей `VolMult` и `VolBaseDays`; проверить победу краёв
  (`VolBaseDays` 3 — база короче недели; `VolBaseDays` 30 — зонд за каноническим краем;
  `VolMult` 3.0 — верхний край).
- [ ] **Step 3:** записать место MVID в каталожном признаке «весь массив оси выше baseline без
  гейта»: точечно он выполнен (худший узел темы 1.240 против baseline 1.179, лучший 1.497), то есть
  гейт объёмов — носитель сигнала, а не только фильтр числа сделок (прецедент NKHP). Сверить с
  тем, что выбрали фолды.
- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): MVID, тема volume`.

---

### Task 8: Тема `vol_window`

**Files:** Modify `cal_vol_window.json` (строка результата)

- [ ] **Step 1:** прогнать тему.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_vol_window.json -out ./reports/MVID_vol_window \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** сверить `VolMult` с темой `volume`: расхождение означает, что форма гейта не
  откалибрована, и тогда **и `VolMult`, и `VolLookbackBars` идут в точку дефолтами ядра** —
  записать это решение прямым текстом (прецедент TRNFP).
- [ ] **Step 3:** проверить края оси `VolLookbackBars` (1 и 32). Каталожное правило: связь «чем
  ликвиднее бумага, тем ближе оптимум окна к дефолту» проверена на десяти тикерах и не
  подтверждена — ось общего правила не даёт, поэтому победа любого края не является аномалией, но
  требует записи.
- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): MVID, тема vol_window`.

---

### Task 9: Тема `risk`, риск-гейт A и зонды капкана

**Files:** Modify `cal_risk.json`; Create `data/params/rsi_pullback/mvid/plateau_stop_07.json`,
`plateau_stop_10.json`, `plateau_tp_02.json`

- [ ] **Step 1:** прогнать тему.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_risk.json -out ./reports/MVID_risk \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** применить риск-гейт A: победитель `StopDailyATR` > 1.0 отвергается, в точку идёт
  ближайшее значение оси внутри гейта, цена решения в PF пишется прямым текстом.
- [ ] **Step 3:** снять три зонда одиночными прогонами на полном окне и выписать для каждого число
  сделок, долю выходов по `SL`, медиану удержания, долю ночёвок и max DD:

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_stop_07.json -out ./reports/MVID_stop07 \
  -months 36 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_stop_10.json -out ./reports/MVID_stop10 \
  -months 36 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_tp_02.json -out ./reports/MVID_tp02 \
  -months 36 -min-trades 20 -metric profit_factor
```

  Файлы — одноточечные сетки: дефолты ядра со `StopDailyATR` 0.7, со `StopDailyATR` 1.0 и с
  `TPDailyATR` 0.2 соответственно.
- [ ] **Step 4:** записать форму капкана прямым текстом: точечно пул сделок встаёт после 0.7
  (142 → 142 → 141 → 141 → 141 → 140 на 0.7 … 2.0) при росте PF с 1.233 (0.6) до 1.661 (0.8).
  Гейт A пропускает всё до 1.0, поэтому фильтруют гейт B и анатомия: падение доли `SL`-выходов
  вместе с ростом удержания и ночёвок — подпись капкана, такой узел отвергается независимо от PF.
- [ ] **Step 5:** записать поведение оси цели: точечно максимум на `TPDailyATR` 0.2 (1.364/173) —
  цель ≈ 0.7% цены, то есть **втрое реальный круг издержек 0.217%**; выше 1.5 ось инертна
  (145 сделок и 1.050 побайтово на 1.5, 2.0, 2.5). Зонд `plateau_tp_02.json` меряет цену этого
  выбора; выбор цели 0.2 в точке допустим только если она проходит пункт 5 стоп-условия.
- [ ] **Step 6:** дописать результат, коммит
  `feat(rsi_pullback): MVID, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit`, `exit_low` и `trail`

**Files:** Modify `cal_exit.json`, `cal_exit_low.json`, `cal_trail.json` (строки результата)

- [ ] **Step 1:** прогнать три темы.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_exit.json -out ./reports/MVID_exit \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_exit_low.json -out ./reports/MVID_exit_low \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/cal_trail.json -out ./reports/MVID_trail \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2:** проверить нижний край канонической `exit`: точечно её глобальный максимум стоит
  на **40** — нижнем крае оси (1.226/193), поэтому победа 35 или 40 в фолдах означает, что ось не
  исчерпана вниз, и тогда голос темы `exit_low` становится решающим.
- [ ] **Step 3:** применить правило голосов `exit_low`: они идут в сборку **только если** принятая
  тема входа дала `RSILower ≤ 20`; при конфликте с канонической `exit` побеждает тема с большим
  pooled OOS. Решение записать прямым текстом.
- [ ] **Step 4:** проверить, не обходит ли трейл риск-гейт A (урок AFKS): гейт применяется к
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`. Трейл ближе стопа делает эффективной защитой
  себя и обязан проходить порог 30% наравне со стопом.
- [ ] **Step 5:** записать исход `UseRSIExit`: точечно вся ось при `UseRSIExit = 0` лежит ниже
  соответствующих узлов с включённым выходом (0.5 → 1.147 против 1.234; 1.0 → 1.039 против 1.179),
  то есть RSI-выход отключать нельзя. Победа `UseRSIExit = 0` в трёх фолдах — расхождение с замером
  и требует отдельной записи.
- [ ] **Step 6:** записать исход трейла: точечно 0.5 даёт 1.234 против 1.179 у baseline (+0.055 —
  на грани правила «плато шириной меньше 0.05 PF читается как отсутствие сигнала»), с 1.0 ось
  инертна (трейл шире стопа 0.5 и уровень не связывает).
- [ ] **Step 7:** дописать результаты, коммит `feat(rsi_pullback): MVID, темы exit, exit_low и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и пять пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/mvid/plateau_point.json`, файлы соседей плато
(`plateau_<поле>_<значение>.json`), `docs/superpowers/plans/task-11-report-mvid.md`

- [ ] **Step 1:** собрать точку правилом большинства (≥ 3 из 4; иначе дефолт ядра); для каждого из
  восемнадцати полей записать голоса фолдов и решение, отмечая случайные совпадения с дефолтом.
  Поля входа — из `entry` либо `entry_deep` по правилу Task 4 Step 3; поле тренда — из `trend` либо
  `trend_hump` по правилу Task 5 Step 3; `RSIUpper` — из `exit` либо `exit_low` по правилу Task 10
  Step 3.
- [ ] **Step 2:** применить риск-гейт A к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`,
  потолок 1.0.
- [ ] **Step 3:** прогнать точку схемой 36/12/6; выписать pooled OOS PF, сделки пула, пофолдовые
  in-sample → OOS PF / сделки / NetPnL% / MaxDD%. Baseline для сравнения — **1.326/100**.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_point.json -out ./reports/MVID_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 4:** проверить пункты 1 и 2 стоп-условия **и дополнительное условие MVID (pooled OOS
  ≥ 1.326)**. Провал → **остановиться**, задачи 12–16 не выполнять, числа владельцу.
- [ ] **Step 5:** контрольная схема 24/12/3 — пункт 3. Baseline на том же куске даёт **1.469**:
  точка обязана быть не хуже дефолтов, а не просто выше единицы.

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_point.json -out ./reports/MVID_point_24 \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 6:** пункты 4 и 5 — та же схема 36/12/6 при удвоенном и реальном круге издержек:

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_point.json -out ./reports/MVID_point_cost2 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.001
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_point.json -out ./reports/MVID_point_cost_real \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.0011
```

  Пункт 5 — главный на MVID: baseline при `-commission 0.0011` даёт 0.995. Дополнительно снять
  справочную строку `-commission 0.0015` (`-out ./reports/MVID_point_cost3`) — это круг 0.3%, то
  есть цена бумаги около 33 ₽ при том же шаге 0.05 ₽.
- [ ] **Step 7:** риск-гейт B на полной истории, потолок max DD **11.10%** (жёсткий):

```bash
go run ./cmd/backtest -ticker MVID -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/mvid/plateau_point.json -out ./reports/MVID_point_full \
  -months 36 -min-trades 20 -metric profit_factor
```

- [ ] **Step 8:** анатомия и полугодия точки против шести чисел третьего контура (доля SL 28.2%,
  удержание 9 / 20 / 56, ночёвок 40.3%, выходов в выходную сессию 6.7%, max DD 11.10%, expectancy
  127.42 ₽). Проверить подпись капкана (падение доли SL при росте удержания и ночёвок → более узкий
  стоп). **Обязательный замер, специфичный для MVID:** доля выходов в выходную сессию — медиана
  оборота выходного дня 3.25 млн ₽, и рост этой доли записывается как реализованный риск
  проскальзывания.
- [ ] **Step 9:** соседи плато по каждому полю, ушедшему от дефолта (± один узел оси); край сетки
  дополнительно проверяется зондом за краем; соседа по оси стопа при включённом трейле проверять
  **в сторону уменьшения**. Плато шириной меньше 0.05 PF записывается как отсутствие сигнала;
  правило ничьих (разница pooled OOS < 0.05 → меньшая просадка) применяется механически.
- [ ] **Step 10:** написать `docs/superpowers/plans/task-11-report-mvid.md` со всеми выкладками,
  командами и таблицами.
- [ ] **Step 11:** коммит `feat(rsi_pullback): MVID, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Задача выполняется только если не сработал ни один из пяти пунктов стоп-условия и pooled OOS
точки ≥ 1.326.**

Пункт 2 сработал (15 сделок < 20); задачи выполняются по решению владельца §5.8.1 спеки, коммит
9e0fecc.

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/mvid/mvid.go`,
`mvid_test.go`, `internal/service/backtest/rsi_pullback_registry_test.go`

- [ ] **Step 1:** заменить `TestParamsTrackTheBaselineUntilCalibrated` на
  `TestParamsAreTheAcceptedPoint` (снимок восемнадцати полей принятой точки) плюс
  `TestPointDiffersFromTheCoreBaseline`.
- [ ] **Step 2:** убедиться, что тесты падают.
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/mvid/` → FAIL
- [ ] **Step 3:** поставить литерал принятой точки в `DefaultParams()`.
- [ ] **Step 4:** заменить тест реестра `TestRSIPullbackMVIDTracksBaseline` на
  `TestRSIPullbackMVIDServesTheCalibratedPoint`.
- [ ] **Step 5:** Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... && ./bin/golangci-lint run ./internal/...` → PASS
- [ ] **Step 6:** коммит `feat(rsi_pullback): MVID откалиброван — литерал вместо отслеживания baseline`.

---

### Task 13: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`

- [ ] **Step 1:** добавить импорт и строку `mvid.Ticker: mvid.DefaultParams(),` в `paramsByTicker`.
- [ ] **Step 2:** дописать абзац комментария карты в англоязычном стиле соседей: вердикт по планке,
  оба риск-гейта и их цена, **пятый пункт стоп-условия и его исход** (реальный круг 0.217% при шаге
  0.05 ₽), и принятые риски: круг издержек растёт по мере падения цены (условие пересмотра — цена
  ниже 40 ₽), оборот ниже гейта вселенной скринера (41 млн ₽ за 12 месяцев, условие пересмотра —
  ниже 20 млн ₽ за 6 месяцев), односторонне падающий инструмент (−76.8% за окно), тонкая выходная
  сессия (3.25 млн ₽ при 6.7% выходов baseline).
- [ ] **Step 3:** Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/...` → PASS
- [ ] **Step 4:** коммит `feat(rsi_pullback): MVID в реестре живого раннера`.

---

### Task 14: Боевая вселенная

**Files:** Modify `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`,
`env/prod.env`, `env/prod.env.example`, `env/local.env.example`, `docs/rsi_pullback/live.md`

- [ ] **Step 1:** перечитать числа Task 11 — пункт 2 сработал (15 сделок < 20); задачи выполняются
  по решению владельца §5.8.1 спеки, коммит 9e0fecc. Дополнительное условие (≥ 1.326) выполнено,
  оба гейта пройдены.
- [ ] **Step 2:** добавить `"MVID"` в `want` теста конфига двадцать шестым.
- [ ] **Step 3:** Run: `go test ./internal/config/ -run RSIPullback` → FAIL
- [ ] **Step 4:** дописать `"MVID"` в `Tickers` + комментарий-абзац; дописать `,MVID` в
  `env/prod.env`, `env/prod.env.example`, `env/local.env.example`.
- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  `docs/rsi_pullback/live.md`.
- [ ] **Step 6:** Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 7:** коммит `feat(rsi_pullback): завести MVID в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/mvid/mvid.go`
(док-комментарий), возможно `docs/rsi_pullback/strategy.md`

- [ ] **Step 1:** переписать шапку пакета в итог калибровки: вердикт по планке по каждому критерию
  обеих ключевых тем; точка поле за полем с голосами фолдов; оба walk-forward со сравнением с
  baseline (1.326/100 и 1.469/56); **четыре строки издержек** (0.1% модели, 0.2% пункта 4, 0.217%
  реального круга пункта 5, 0.3% справочно); соседи плато; шесть полугодий; анатомия против
  baseline, включая долю выходов в выходную сессию; результат обоих риск-гейтов и их цена.
  Отдельными абзацами — принятые риски и условия пересмотра из §7 спеки: реальный круг издержек и
  его рост при падении цены (пересмотр при цене ниже 40 ₽); падающий оборот (пересмотр при медиане
  6 месяцев ниже 20 млн ₽); односторонне падающий инструмент и некомпенсируемый корпоративный риск;
  капкан широкого стопа внутри зоны гейта A; тяга темы `risk` к цели 0.2 ATR; тонкая выходная
  сессия; расширение сессии внутри окна.
- [ ] **Step 2:** Run: `go test ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 3:** механический вывод в `docs/rsi_pullback/strategy.md` — **только если** MVID дал
  вывод о механике. Кандидат назван заранее: **шаг цены как критерий отбора вселенной наравне с
  оборотом** (лучший рельеф входа в каталоге при круге издержек, съедающем весь edge дефолтов).
  Пер-тикерных чисел, дат и вердиктов в `docs/rsi_pullback/` не писать.
- [ ] **Step 4:** коммит `docs(rsi_pullback): разбор калибровки MVID и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** Run: `go run ./cmd/pullparity -tickers MVID -months 24` → ноль расхождений
  (на 24 месяцах, не на 36 — урок VSMO).
- [ ] **Step 2:** Run: `./bin/mage ci` → зелёный
- [ ] **Step 3:** Run: `git status --porcelain | grep -c '^.. reports/'` → `0`
- [ ] **Step 4:** финальный коммит, если остались правки.

---

## Self-Review

**Покрытие спеки:** §1 (инструмент, шаг цены, ликвидность, волатильность, отсутствие дивидендов) →
Global Constraints и Task 2 Step 3; §2 (окно и число фолдов) → Global Constraints и Task 3 Step 2;
§3 (baseline и маршрут TGKA) → Global Constraints и Task 11 Steps 3–5; §4 (рельеф осей) →
`_comment` в Task 1 Step 3 и ожидания Tasks 3–10; §5.1 (тринадцать тем и пять отличий сеток) →
Task 1 и Tasks 3–10; жёсткие инварианты → Task 1 Step 1; §5.2 (правило сборки) → Task 11 Step 1 и
правила в Tasks 4, 5, 10; §5.3 (соседи плато) → Task 11 Step 9; §5.4 (гейт A) → Task 9 Step 2,
Task 10 Step 4, Task 11 Step 2; §5.5 (гейт B жёсткий и правило ничьих) → Task 11 Steps 7 и 9;
§5.6 (третий контур, замер выходной сессии) → Task 11 Step 8; §5.7 (планка) → Task 4 Step 2 и
Task 5 Step 2; §5.8 (пять пунктов и порог 1.326) → Task 11 Steps 4–6; §5.9 (правило прода) →
Task 14 Step 1; §6 (артефакты) → Tasks 1, 2, 9, 11–15; §7 (принятые риски) → Task 13 Step 2 и
Task 15 Step 1; `pullparity` и `mage ci` → Task 16.

**Плейсхолдеры:** один — литерал в Task 12 (точка известна только после Task 11), и он неизбежен
по построению процедуры. Все прочие шаги несут конкретные оси, команды, пороги и числа сравнения.

**Согласованность имён:** `mvid.Ticker` и `mvid.DefaultParams()` заводятся в Task 2 и используются
в Tasks 12 и 13; тринадцать файлов сеток из Task 1 читаются задачами 3–10 по тем же путям;
`plateau_stop_07.json`, `plateau_stop_10.json`, `plateau_tp_02.json` заводятся в Task 9 и
упоминаются только там; `plateau_point.json` — Task 11; тесты `TestMVIDGridsStayWide` (Task 1),
`TestParamsTrackTheBaselineUntilCalibrated` и `TestRSIPullbackMVIDTracksBaseline` (Task 2) →
`TestParamsAreTheAcceptedPoint`, `TestPointDiffersFromTheCoreBaseline` и
`TestRSIPullbackMVIDServesTheCalibratedPoint` (Task 12).
