# CNRU под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести CNRU (МКПАО «ЦИАН») до вердикта по стратегии `rsi_pullback`: каталог максимально
широких сеток, канонический тематический walk-forward, принятая точка, прошедшая два риск-гейта,
пять пунктов стоп-условия и порог дефолтов ядра, литерал в пакете и заведение в боевую вселенную
двадцать седьмым тикером. Если точка порога не берёт — прод на дефолтах ядра по прецеденту TGKA.

**Architecture:** Процедура каноническая, но задача у неё обратная привычной. **Дефолты ядра на CNRU
уже проходят всё**: pooled OOS PF 1.989 на 122 сделках при четырёх прибыльных фолдах из четырёх,
каждый календарный год окна в плюсе, живучесть до круга 0.3%. Значит калибровка обязана не найти
edge, а не испортить найденный, и её результат оценивается **против уровня дефолтов 1.989**, а не
против единицы. Одиннадцать тем поверх дефолтов ядра, схема **36/12/6** (четыре фолда, проверено до
написания спеки) плюс обязательный контроль **24/12/3** — сессия внутри окна расширялась вдвое
(19 → 34 получасовых бара в буднем дне, 2024H2). Два риск-гейта: гейт выживаемости применяется к
`min(StopDailyATR, TrailDailyATR при UseTrail=1)`, потолок 1.0; гейт просадки — **жёсткий**, max DD
точки ≤ max DD baseline 9.15%.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков), `python3` для
разбора журналов сделок.

**Spec:** `docs/superpowers/specs/2026-09-08-cnru-rsi-pullback-prep-design.md`
**Замеры:** `reports/_analysis/cnru_pullback_prep_measurements.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех прогонах; `-interval Minutes30` обязателен в каждой команде
  (дефолт CLI — `Hour1`, забытый флаг даёт чужие числа, а не ошибку).
- **Схема:** `-months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor`;
  у темы `screen` — `-min-trades 1`.
- **Расчётное окно 2023-09-08 … 2026-09-08, 36 месяцев, без обрезки.** Получасовой ряд начинается
  2023-08-04, то есть окно помещается с запасом в один месяц; расширять окно назад **некуда**.
- **Внутри окна есть остановка торгов 2025-02-05 → 2025-04-03 (57 дней)** — редомициляция
  CIAN → МКПАО «ЦИАН». Сплита, консолидации и обмена с коэффициентом нет (575.0 ₽ до, 580.0 ₽
  в день возобновления), **ряд корректировать не нужно**. Дыра остаётся дырой и порождает
  обязательный замер (Task 11 Step 9).
- **Число фолдов сверяется на первой же теме:** «Фолдов: 4» в шапке отчёта темы `screen` — иначе
  остановиться и доложить владельцу (ловушка ASTR). Дефолты дают 4 фолда на обеих схемах, проверено.
- **`-refresh` НЕ запускать:** кэш дотянут 2026-09-08 (`CNRU_Minutes30.json`, `CNRU_Day1.json`).
- **Сетки максимально широкие**; обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
  инварианты: `RSILower ≤ 50`; `RSIPeriod ≥ 2`; `StopDailyATR` нигде не ноль; ни одна из двух
  трендовых тем не порождает пар `EMAFast ≥ EMASlow`; ось `RSILower` темы `entry` содержит 5 и все
  узлы плато 22, 24, 25, 26, 28; ось `EMASlow` темы `trend_hump` содержит 30, 35, 40, 45; ось
  `EMASlow` темы `trend` содержит 250; ось `VolBaseDays` темы `volume` содержит 50; ось `VolMult`
  темы `volume` содержит 4.0; ось `RSIUpper` темы `exit` содержит оба края 35 и 95.
- **Исключены как вырожденные или доказанно инертные:** `TPDailyATR` 3.0 и выше (ось инертна с 1.5:
  163 сделки и PF 1.702 побайтово на 1.5, 2.0, 2.5, 3.0); `RSIPeriod` 14 и выше (на 12 уже 20
  сделок за три года); `EMASlow` ниже 12 (за краем измеренного горба).
- **Темы `entry_deep` и `exit_low` НЕ заводятся** — расхождение с непосредственным прецедентом MVID,
  обоснованное §5.1 отличием 4 спеки: максимум оси входа (25) и максимум оси выхода (45) — оба
  внутренние, обе зоны накрываются каноническими темами.
- **Правило сборки точки:** ≥ 3 фолда из 4 за значение, иначе дефолт ядра; ничья 2/2 не считается;
  `trend_hump` побеждает только при большинстве ≥ 3/4 И превосходстве pooled OOS над `trend`;
  `vol_window` забирает поля объёмного гейта только при большинстве ≥ 3/4 И превосходстве pooled OOS
  над первичной `volume`.
- **РИСК-ГЕЙТ A:** защита достижима ≥ 30% будних дней; применяется к
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`. Таблица (n = 722 будних дня): 0.3 — 98.8%,
  0.4 — 94.6%, 0.5 — 88.5%, 0.6 — 80.1%, 0.7 — 70.4%, 0.8 — 60.8%, 0.9 — 48.5%, **1.0 — 37.3%**,
  1.2 — 22.4%, 1.5 — 9.7%, 2.0 — 2.8%. **Потолок — 1.0.**
- **РИСК-ГЕЙТ B (жёсткий):** max DD точки на полном окне ≤ **9.15%** (max DD baseline), а не
  1.3 × baseline. Основание: капкан широкого стопа начинается **на самом дефолте** 0.5, то есть
  весь он лежит внутри зоны, разрешённой гейтом A.
- **ПРАВИЛО НИЧЬИХ:** при разнице pooled OOS < 0.05 в точку идёт вариант с меньшей просадкой.
- **Третий контур:** опорные числа baseline — доля SL **21.3%**, удержание **9 / 20 / 35** баров,
  ночёвок **48.8%**, выходов в выходную сессию **6.1%**, max DD **9.15%**, expectancy **504.37 ₽**.
- **ДВА ОБЯЗАТЕЛЬНЫХ ЗАМЕРА, СПЕЦИФИЧНЫХ ДЛЯ CNRU** (Task 11 Step 9):
  1. **Экспозиция точки через дивидендный гэп 2025-12-12 (−16.18%)** — крупнейший в каталоге
     (спецдивиденд ~104 ₽ при цене ~640 ₽). У baseline ноль. Каждая сделка точки, удерживаемая через
     эту ночь, записывается в доку пакета с её PnL.
  2. **Экспозиция точки через остановку торгов 2025-02-05 → 2025-04-03.** У baseline ноль. Сделка,
     открытая до 2025-02-05 и закрытая после 2025-04-03, означает **два месяца запертой позиции без
     возможности выйти по стопу**; такая точка **не принимается независимо от PF**, и в неё идёт
     ближайший вариант плато без этого свойства. Прецедента в каталоге нет.
- **Планка:** `entry` и `trend` (канонические, не `trend_hump`) обе дают pooled OOS PF ≥ 1.5 при
  ≥ 20 сделках; ведущая ось (`RSILower` для `entry`, `EMASlow` для `trend`) выбрана одинаково в ≥ 3
  фолдах из 4; вырожденный фолд в пользу тикера не засчитывается. Ожидание записано заранее: дефолты
  уже дают 1.989 при четырёх прибыльных фолдах, а ось входа несёт измеренное плато шириной 5
  пунктов — **шанс взять планку выше, чем у любого тикера каталога, включая SVAV** (единственного,
  кто её взял).
- **Стоп-условие из ПЯТИ пунктов:** (1) pooled OOS PF < 1.0 на 36/12/6; (2) < 20 сделок в пуле OOS;
  (3) pooled OOS PF < 1.0 на 24/12/3; (4) то же при `-commission 0.001` (круг 0.2%); (5) **хотя бы
  один календарный год окна убыточен** на одиночном прогоне точки по полному окну — критерий
  владельца, добавленный 2026-09-08. Годы 2024 и 2025 полные и триггерят безусловно; неполные 2023
  (с 08 сентября) и 2026 (до 08 сентября) триггерят, только если убыточны **и** содержат ≥ 10
  сделок. Порог в 10 сделок объявлен до прогонов (на дефолтах в 2023 всего шесть сделок).
- **Пункта по реальному кругу издержек, который был у MVID, здесь НЕТ**: круг CNRU **0.064%** при
  шаге 0.2 ₽ и цене 627 ₽ — **ниже** модельных 0.1%. Модель пессимистична в 1.6 раза, пункт 4 уже
  меряет втрое худшие условия, чем реальность.
- **ДОПОЛНИТЕЛЬНОЕ УСЛОВИЕ CNRU (§5.9 спеки):** точка с pooled OOS PF на 36/12/6 **ниже 1.989**
  (уровень дефолтов ядра) литералом в прод не заводится. В этом случае — **маршрут TGKA**: прод на
  `core.DefaultParams()` с именным исключением в сторожевом тесте. Маршрут открыт и проверен
  заранее: дефолты проходят все пять пунктов.
- **Исхода без прода нет.** При срабатывании пункта стоп-условия или недоборе порога 1.989 задачи
  12–16 выполняются **по маршруту TGKA**, а не отменяются. Отменяются они только по прямому решению
  владельца после доклада чисел.
- **Дефолты ядра:** `RSIPeriod 4`, `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`,
  `DailyATRPeriod 14`, `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`,
  `TPDailyATR 0.6`, `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`,
  `UseRSIExit 1`, `UseTrail 0`, `TrailDailyATR 0`. Baseline на окне: **164 сделки, PF 1.814, net
  +82 716.98 ₽ (+82.72%)**, win rate **75.00%**, max DD **12 081.22 ₽ (9.15%)**, exposure 5.72%,
  CAGR 22.23%, expectancy **504.37 ₽ = 0.504% капитала**, выходы RSI 109 (66.5%) / SL 35 (21.3%) /
  TP 20 (12.2%), удержание 9 / 20 / 35, ночёвок 48.8%, выходов в выходную сессию 6.1%.
  Walk-forward дефолтов: 36/12/6 → **1.989/122** (compounded +84.21%, win rate 78.69%, expectancy
  552.11, Sortino 0.385; фолды 1.863/29, 2.535/37, 3.180/21, 1.214/35 — **ни одного убыточного**);
  24/12/3 → **1.705** (compounded +21.74%).
- **Календарные годы baseline:** 2023 (с 09-08) **+2 528 ₽ / PF 1.880 / 6 сделок**, 2024
  **+22 942 / 1.701 / 56**, 2025 **+45 937 / 2.219 / 60**, 2026 (до 09-08) **+11 310 / 1.399 / 42**.
- **Издержки:** шаг **0.2 ₽**; круг **0.064%** при цене 627 ₽; модель считает 0.1%. Чувствительность
  walk-forward дефолтов: 0.1% → **1.989**; 0.2% → **1.705**; 0.3% → **1.448**.
- **Ликвидность растёт:** медиана оборота будних дней 61.75 млн ₽ (36 мес), 68.72 (24),
  **118.49 (12)**, 102.91 (6) — первый такой тикер каталога. Выходная сессия: 256 дней, медиана
  **3.92 млн ₽**. Дневной ATR(14) медиана **3.95%** (p10 1.85%, p90 5.91%) — верх каталога.
- **Режим:** buy&hold **−26.9%**; полугодия −26.8, +25.6, −36.3, +11.4, +7.8, −15.8, +25.0;
  растущих четыре из семи. Худшие открытия будних дней **−16.18% (2025-12-12, дивидендная
  отсечка)**, −9.97% (2025-04-07), −7.47% (2026-06-22), −5.19%, −3.74%, −3.51%; экспозиция baseline
  через каждое — **ноль**.
- **Априор скринера не снимался:** кандидат выбран владельцем, а не шортлистом `cmd/pullscreen`.
  Ссылок на априор в `_comment` и доке пакета не делать.
- **Каждый `_comment`** обязан содержать: что тема меряет и сколько прогонов; замеры осей с
  предупреждением о крае; полную команду запуска с путём `data/params/rsi_pullback/cnru/<файл>`
  (этого требует `TestRSIPullbackCalFilesValid`); место под строку
  `РЕЗУЛЬТАТ ПРОГОНА 2026-09-08: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md).
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле истории.
- **Ветка:** `feat/cnru-pullback-prep` от `feat/mvid-pullback-prep` (`92a56a7`); спека закоммичена
  (`8b94ebd`). MVID в `main` ещё не смержен, поэтому CNRU строится поверх ветки MVID и становится
  **двадцать седьмым** тикером вселенной.

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/cnru/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_hump.json`, `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_cnru_grid_test.go`

**Interfaces:**
- Produces: одиннадцать путей `data/params/rsi_pullback/cnru/cal_*.json`, которые читают задачи
  3–10; тест `TestCNRUGridsStayWide`.

- [ ] **Step 1: Написать падающий сторожевой тест осей.** Образец —
  `internal/service/backtest/rsi_pullback_mvid_grid_test.go`. Тест `TestCNRUGridsStayWide` читает
  файлы каталога и проверяет **все** инварианты из Global Constraints:
  - `RSILower ≤ 50` во всех файлах, и ось `RSILower` в `cal_entry.json` содержит узлы
    `5, 22, 24, 25, 26, 28`;
  - `RSIPeriod ≥ 2` везде;
  - `StopDailyATR != 0` везде, где ось присутствует;
  - ни `cal_trend.json`, ни `cal_trend_hump.json` не порождают пар `EMAFast ≥ EMASlow`
    (перебрать декартово произведение осей внутри каждого файла);
  - ось `EMASlow` в `cal_trend_hump.json` содержит `30, 35, 40, 45`;
  - ось `EMASlow` в `cal_trend.json` содержит `250`;
  - ось `VolBaseDays` в `cal_volume.json` содержит `50`, ось `VolMult` содержит `4.0`;
  - ось `RSIUpper` в `cal_exit.json` содержит `35` и `95`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/backtest/ -run TestCNRUGridsStayWide -v`
  Expected: FAIL — файлов сеток нет.

- [ ] **Step 3: Создать одиннадцать файлов сеток.** Оси дословно (§5.1 спеки):
  - `cal_screen.json`: `UseDayATRGate` [0,1] × `UseVolume` [0,1] — **4** прогона.
  - `cal_entry.json`: `RSIPeriod` [2,3,4,5,6,7,8,10,12] × `RSILower`
    [5,10,15,20,22,24,25,26,28,30,35,40,45,50] — **126**.
  - `cal_trend.json`: `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] — **48**.
  - `cal_trend_hump.json`: `EMAFast` [3,5,8,10] × `EMASlow` [12,15,20,25,30,35,40,45] — **32**.
  - `cal_day.json`: `FreshDayATR` [0,0.1,0.2,0.3,0.4,0.5] × `SpentDayATR`
    [0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0] — **54**.
  - `cal_day_spent.json`: `FreshDayATR` [0] × `SpentDayATR`
    [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0] — **10**.
  - `cal_volume.json`: `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0,3.5,4.0] ×
    `VolBaseDays` [3,5,10,14,20,30,40,50] — **64**.
  - `cal_vol_window.json`: `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult`
    [1.2,2.5] — **18**.
  - `cal_risk.json`: `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.2,1.5,2.0] × `TPDailyATR`
    [0.2,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.0,2.5] — **100**.
  - `cal_exit.json`: `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90,95] — **13**.
  - `cal_trail.json`: `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.3,0.5,0.7,1.0,1.5]
    — **10**.

  Формат файла — как у `data/params/rsi_pullback/mvid/cal_screen.json`: объект с `_comment` и
  массивом `phases`, каждая фаза — `{"name": ..., "grid": {...}, "keepTop": N}`.

  **Обязательные предупреждения о краях в `_comment`:**
  - `cal_entry.json`: узлы `RSILower` 5 и 10 вырождены (3 и 30 сделок за три года); узел
    `RSIPeriod` 12 вырожден (20 сделок); плато 22–26 измерено шагом 1 до написания сеток
    (21 → 1.838/104, 22 → 2.555/109, 23 → 2.464/116, 24 → 2.651/122, 25 → 2.543/131,
    26 → 2.388/135, 27 → 2.046/144, 28 → 2.078/152, 29 → 1.769/156) и накрыто шагом 2 внутри
    двумерной темы, потому что оси взаимодействуют (4×25 → 2.543, 5×30 → 2.071, 5×25 → 1.640).
  - `cal_trend.json` / `cal_trend_hump.json`: горб `EMASlow` 40–75 (2.002 / 1.906 / 1.927) лежит
    **ниже** нижнего края канонической темы 50, отсюда разделение; узел 12 (1.557/125) —
    нижний зонд горба.
  - `cal_volume.json`: максимум оси 3.0×40 → 3.361/**60** стоит вдвое дальше канонического края 20,
    рост по `VolBaseDays` не прекращается до 50 (2.5×50 → 3.235/68, 3.0×50 → 3.275/60,
    4.0×50 → 3.160/43); зона 40–50 дней — **зона редкости**, 43–70 сделок за три года, и её
    удерживает только порог в 20 сделок OOS. Низ темы уходит под baseline 1.814
    (1.5×10 → 1.582/114, 1.0×10 → 1.731/131, 1.2×5 → 1.743/127, 1.2×10 → 1.746/127) — признак §8.1
    доки стратегии НЕ выполнен.
  - `cal_risk.json`: пул сделок встаёт уже на дефолте 0.5 (164, 164, 164, 162, 162, 162, 162, 162
    на 0.5 … 2.0) при росте PF с 1.814 до 2.763 — капкан начинается на дефолтном значении, чего не
    было ни у одного тикера каталога; гейт A пропускает всё до 1.0, поэтому весь капкан внутри
    разрешённой зоны. `TPDailyATR` выше 1.5 инертна (163 сделки и 1.702 побайтово).
  - `cal_exit.json`: максимум 45 (2.120/209) внутренний, вся зона 35–55 выше baseline, вся зона
    75–95 ниже; нижний край 35 даёт 236 сделок — вырождения нет, отдельная тема `exit_low` не нужна.
  - `cal_trail.json`: лучший трейл 0.5 даёт +0.010 PF над baseline — меньше правила 0.05, сигнала
    нет; при `UseRSIExit = 0` вся ось ниже соответствующих узлов с включённым выходом.
  - `cal_day_spent.json`: пик оси стоит **ровно на дефолте 0.8** — единственный такой случай в
    каталоге.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ -run 'TestCNRU|TestRSIPullback' -v`
  Expected: PASS (включая `TestRSIPullbackCalFilesValid`, который требует полную команду запуска в
  `_comment` с путём именно этого файла).

- [ ] **Step 5: Коммит.**

```bash
git add data/params/rsi_pullback/cnru internal/service/backtest/rsi_pullback_cnru_grid_test.go
git commit -m "feat(rsi_pullback): каталог сеток CNRU"
```

---

### Task 2: Пакет `strategy/cnru` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/cnru/cnru.go`,
  `internal/service/trading_strategy/rsi_pullback/strategy/cnru/cnru_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Produces: `cnru.Ticker` (константа `"CNRU"`) и `cnru.DefaultParams() core.Params` — их используют
  задачи 12 и 13.

- [ ] **Step 1: Написать падающие тесты** по образцу пакета `mvid`:
  `TestParamsTrackTheBaselineUntilCalibrated` (`DefaultParams()` равен `core.DefaultParams()` поле
  за полем) и `TestTickerIsCNRU`.

- [ ] **Step 2: Убедиться, что падают.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/cnru/ -v`
  Expected: FAIL — пакета нет.

- [ ] **Step 3: Создать пакет.** Док-комментарий англоязычный, в стиле соседей, и содержит:
  - расчётное окно 2023-09-08 … 2026-09-08 (36 месяцев), интервал Minutes30;
  - **остановку торгов 2025-02-05 → 2025-04-03 (57 дней, редомициляция CIAN → МКПАО «ЦИАН»)**,
    вывод «сплита и обмена нет, ряд не корректируется» и вывод «дыра остаётся риском»;
  - расширение сессии 19 → 34 бара в 2024H2;
  - baseline с анатомией (164 сделки, PF 1.814, net +82 716.98 ₽, win rate 75.00%, max DD 9.15%,
    expectancy 504.37 ₽, выходы RSI 66.5% / SL 21.3% / TP 12.2%, удержание 9 / 20 / 35, ночёвок
    48.8%, выходов в выходную сессию 6.1%);
  - walk-forward дефолтов (1.989/122 при фолдах 1.863/29, 2.535/37, 3.180/21, 1.214/35 — ни одного
    убыточного; 24/12/3 → 1.705);
  - календарные годы дефолтов (+2 528 / +22 942 / +45 937 / +11 310);
  - издержки (шаг 0.2 ₽, круг 0.064% против 0.1% модели — **модель пессимистична**;
    чувствительность 1.989 / 1.705 / 1.448);
  - состояние «калибровка не проводилась».

- [ ] **Step 4: Завести тикер в реестр бэктеста** — импорт `rsipullbackcnru` в алфавитном порядке,
  строка в `rsiPullbackRegistry`, тест `TestRSIPullbackCNRUTracksBaseline` в
  `rsi_pullback_registry_test.go`.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...`
  Expected: PASS

- [ ] **Step 6: Коммит.**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/cnru internal/service/backtest
git commit -m "feat(rsi_pullback): пакет CNRU до калибровки"
```

---

### Task 3: Тема `screen` — цена двух гейтов и сверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/cnru/cal_screen.json` (строка результата)

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_screen.json -out ./reports/CNRU_screen \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Проверить «Фолдов: 4»** в шапке отчёта — иначе **остановиться** и доложить владельцу
  (ловушка ASTR).

- [ ] **Step 3: Выписать** pooled OOS PF, сделки пула, пофолдовых победителей `UseDayATRGate` и
  `UseVolume`, число сделок каждой из четырёх комбинаций. **Ожидание, записанное заранее (§4.4
  спеки): 1×1.** На полном окне комбинации дают 0×0 → 1.299/400, 0×1 → 1.263/250,
  1×0 → 1.814/164 (baseline), **1×1 → 2.130/126**: оба гейта окупаются, и объёмный — уже на
  дефолтной форме, в отличие от MVID.

- [ ] **Step 4:** дописать `РЕЗУЛЬТАТ ПРОГОНА 2026-09-08: …` в `_comment`.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): CNRU, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая

**Files:** Modify `data/params/rsi_pullback/cnru/cal_entry.json` (строка результата)

- [ ] **Step 1: Прогнать тему** (126 прогонов × 4 фолда).

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_entry.json -out ./reports/CNRU_entry \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Проверить планку по этой теме** (критерий A: pooled OOS PF ≥ 1.5 при ≥ 20 сделках;
  критерий B: ведущая ось `RSILower` выбрана одинаково в ≥ 3 фолдах из 4). Записать результат по
  каждому критерию отдельно; вырожденный фолд в пользу тикера не засчитывается.

- [ ] **Step 3: Проверить, попали ли победители фолдов внутрь измеренного плато 22–26.** Плато —
  главный рычаг CNRU (§4.1 спеки). Если победители легли на 24, 25 или 26 — записать, что тема
  подтвердила плато. Если легли на 20, 28 или 30 — записать расхождение с замером прямым текстом.

- [ ] **Step 4: Записать взаимодействие осей.** Точечно: при `RSIPeriod` 4 оптимум входа 25
  (2.543/131), при `RSIPeriod` 5 — 30 (2.071/126), при `RSIPeriod` 3 — 20 (1.776/157). Если фолды
  голосуют за пары, не совпадающие с этим рельефом, это записывается отдельно.

- [ ] **Step 5:** дописать результат, коммит `feat(rsi_pullback): CNRU, тема entry`.

---

### Task 5: Темы `trend` и `trend_hump`

**Files:** Modify `cal_trend.json`, `cal_trend_hump.json` (строки результата)

- [ ] **Step 1: Прогнать обе темы.**

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_trend.json -out ./reports/CNRU_trend \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_trend_hump.json -out ./reports/CNRU_trend_hump \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Проверить планку по канонической `trend`** (не по `trend_hump`): pooled OOS PF ≥ 1.5
  при ≥ 20 сделках; ведущая ось `EMASlow` выбрана одинаково в ≥ 3 фолдах из 4.

- [ ] **Step 3: Применить правило приоритета тем.** Поле тренда идёт из `trend_hump` **только если**
  у неё большинство ≥ 3/4 **и** её pooled OOS выше pooled OOS канонической `trend`. Решение
  записать прямым текстом с обоими числами.

- [ ] **Step 4: Записать положение победителей относительно измеренного горба.** Точечно `EMASlow`:
  12 → 1.557/125, 15 → 1.658/130, 20 → 1.529/137, 30 → 1.777/140, **40 → 2.002/145**,
  50 → 1.906/146, 75 → 1.927/158, 100 → 1.814/164 (дефолт), 150 → 1.575/176, 200 → 1.508/178,
  250 → 1.480/183. Вершина горба 40 лежит **ниже** нижнего края канонической темы — если
  каноническая `trend` голосует за 50, это её край, и он обязан быть сверен с голосом `trend_hump`.

- [ ] **Step 5:** дописать результаты, коммит `feat(rsi_pullback): CNRU, темы trend и trend_hump`.

---

### Task 6: Темы `day` и `day_spent`

**Files:** Modify `cal_day.json`, `cal_day_spent.json` (строки результата)

- [ ] **Step 1: Прогнать обе темы.**

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_day.json -out ./reports/CNRU_day \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_day_spent.json -out ./reports/CNRU_day_spent \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать исход `SpentDayATR` против замера.** Точечно ось одногорбая с максимумом
  **ровно на дефолте 0.8** (0.4 → 1.253/335, 0.5 → 1.299/294, 0.6 → 1.359/246, 0.7 → 1.602/194,
  **0.8 → 1.814/164**, 0.9 → 1.440/126, 1.0 → 1.486/92, 1.25 → 1.223/40, 1.5 → 0.729/14,
  2.0 → 0.243/4). Это единственный такой случай в каталоге; победа фолдов над дефолтом здесь —
  расхождение с замером и требует отдельной записи.

- [ ] **Step 3: Записать исход `FreshDayATR`.** Точечно вся ранняя ветка ниже baseline
  (0.1 → 1.576/25, 0.2 → 1.516/31, 0.3 → 1.421/49, 0.4 → 1.402/96, 0.5 → 1.169/142), ожидаемый
  исход — дефолт 0.

- [ ] **Step 4:** дописать результаты, коммит `feat(rsi_pullback): CNRU, темы day и day_spent`.

---

### Task 7: Тема `volume`

**Files:** Modify `cal_volume.json` (строка результата)

- [ ] **Step 1: Прогнать тему** (64 прогона × 4 фолда).

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_volume.json -out ./reports/CNRU_volume \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Проверить, не ушли ли победители в зону редкости.** Точечно узлы с `VolBaseDays`
  40–50 дают 43–70 сделок за три года: 2.5×40 → 2.785/70, 2.5×50 → 3.235/68, **3.0×40 → 3.361/60**,
  3.0×50 → 3.275/60, 3.5×50 → 3.239/55, 4.0×50 → 3.160/43. Если фолды голосуют туда, записать
  прямым текстом, что порог в 20 сделок OOS (пункт 2 стоп-условия) — **единственный** фильтр против
  этой зоны, и он не снимается.

- [ ] **Step 3: Записать форму гейта против факта включения.** Точечно низ темы уходит под baseline
  1.814 (1.5×10 → 1.582/114, 1.0×10 → 1.731/131, 1.2×5 → 1.743/127, 1.2×10 → 1.746/127) — признак
  §8.1 доки стратегии («весь массив выше baseline») **не выполнен**, как и у MVID. Отличие от MVID:
  дефолтная форма 1.2×14 у CNRU уже даёт 2.130/126, то есть +0.32 PF.

- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): CNRU, тема volume`.

---

### Task 8: Тема `vol_window`

**Files:** Modify `cal_vol_window.json` (строка результата)

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_vol_window.json -out ./reports/CNRU_vol_window \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Применить правило приоритета тем.** Поля объёмного гейта идут из `vol_window`
  **только если** у неё большинство ≥ 3/4 **и** её pooled OOS выше pooled OOS первичной `volume`,
  которая структурно владеет формой гейта. Решение записать с обоими числами.

- [ ] **Step 3: Записать исход `VolLookbackBars` против замера.** Точечно при `VolMult = 1.2`:
  1 → 1.780/102, 2 → 2.127/120, **3 → 2.130/126 (дефолт)**, 5 → 1.965/135, 8 → 1.997/146,
  12 → 2.038/149, 16 → 1.807/155, 24 → 1.857/157, 32 → 1.861/158. Пик на дефолте, плато 2–3, размах
  0.35 PF.

- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): CNRU, тема vol_window`.

---

### Task 9: Тема `risk`, риск-гейт A и зонды капкана

**Files:** Modify `cal_risk.json`; Create `data/params/rsi_pullback/cnru/plateau_stop_06.json`,
`plateau_stop_07.json`, `plateau_stop_10.json`, `plateau_tp_03.json`

- [ ] **Step 1: Прогнать тему** (100 прогонов × 4 фолда).

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_risk.json -out ./reports/CNRU_risk \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Применить риск-гейт A.** Победитель `StopDailyATR` > 1.0 отвергается (выживаемость
  1.2 — 22.4% < 30%), в точку идёт ближайшее значение оси внутри гейта, цена решения в PF пишется
  прямым текстом.

- [ ] **Step 3: Снять четыре зонда** одиночными прогонами на полном окне. Файлы — одноточечные
  сетки: дефолты ядра со `StopDailyATR` 0.6, 0.7, 1.0 и с `TPDailyATR` 0.3 соответственно.

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_stop_06.json -out ./reports/CNRU_stop06 \
  -months 36 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_stop_07.json -out ./reports/CNRU_stop07 \
  -months 36 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_stop_10.json -out ./reports/CNRU_stop10 \
  -months 36 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_tp_03.json -out ./reports/CNRU_tp03 \
  -months 36 -min-trades 20 -metric profit_factor
```

  Для каждого зонда выписать: число сделок, долю выходов по `SL`, медиану удержания, долю ночёвок,
  max DD. Разбор журнала — скриптом из Task 11 Step 8.

- [ ] **Step 4: Записать форму капкана прямым текстом.** Точечно пул встаёт **уже на дефолте 0.5**:
  0.3 → 1.745/175, 0.4 → 1.787/168, 0.5 → 1.814/**164**, 0.6 → 1.975/**164**, 0.7 → 2.007/**164**,
  0.8 → 2.414/**162**, 1.0 → 2.484/**162**, 1.2 → 2.720/**162**, 1.5 → 2.535/**162**,
  2.0 → 2.763/**162**. Ни один тикер каталога не показывал капкан, начинающийся на дефолтном
  значении. Гейт A пропускает всё до 1.0, поэтому **весь капкан лежит внутри разрешённой зоны**;
  фильтруют гейт B (потолок 9.15%) и анатомия: падение доли `SL`-выходов вместе с ростом удержания
  и ночёвок — подпись капкана, такой узел отвергается независимо от PF.

- [ ] **Step 5: Записать поведение оси цели.** Точечно максимум на `TPDailyATR` 0.3 (2.039/175);
  выше 1.5 ось инертна (163 сделки и 1.702 побайтово на 1.5, 2.0, 2.5). Цель 0.3 ATR ≈ 1.2% цены
  против реального круга 0.064% — запас в 18 раз, хрупкости к издержкам нет. Двумерный зонд внутри
  зоны гейта A: вершина 0.7×0.4 → 2.208/170, рядом 0.7×0.3 → 2.171/175, 0.6×0.3 → 2.152/175,
  0.6×0.4 → 2.148/170.

- [ ] **Step 6:** дописать результат, коммит
  `feat(rsi_pullback): CNRU, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit` и `trail`

**Files:** Modify `cal_exit.json`, `cal_trail.json` (строки результата)

- [ ] **Step 1: Прогнать обе темы.**

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_exit.json -out ./reports/CNRU_exit \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/cal_trail.json -out ./reports/CNRU_trail \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Проверить, не встал ли победитель `RSIUpper` на нижний край 35.** Точечно максимум
  оси внутренний (45 → 2.120/209), край 35 даёт 1.917/236 — то есть край **не** является максимумом,
  и темы `exit_low` не требуется. Если фолды всё же голосуют за 35, это расхождение с замером: ось
  оказалась не исчерпана вниз, и факт записывается прямым текстом с рекомендацией владельцу
  рассмотреть `exit_low` во втором круге.

- [ ] **Step 3: Проверить, не обходит ли трейл риск-гейт A** (урок AFKS): гейт применяется к
  `min(StopDailyATR, TrailDailyATR при UseTrail = 1)`. Трейл ближе стопа делает эффективной защитой
  себя и обязан проходить порог 30% наравне со стопом.

- [ ] **Step 4: Записать исход `UseRSIExit`.** Точечно при `UseRSIExit = 0` вся ось ниже
  соответствующих узлов с включённым выходом (0.3 → 1.353 против 1.663; 0.5 → 1.622 против 1.824;
  0.7 → 1.523 против 1.790; 1.0 → 1.486 против 1.814), то есть RSI-выход отключать нельзя. Победа
  `UseRSIExit = 0` в трёх фолдах — расхождение с замером и требует отдельной записи.

- [ ] **Step 5: Записать исход трейла.** Точечно лучший узел 0.5 даёт 1.824 против 1.814 у baseline
  — **+0.010 PF**, то есть по правилу «плато шириной меньше 0.05 PF читается как отсутствие
  сигнала» сигнала нет; с 1.0 ось инертна (трейл шире стопа 0.5 и уровень не связывает). Ожидаемый
  исход темы — дефолт `UseTrail = 0`.

- [ ] **Step 6:** дописать результаты, коммит `feat(rsi_pullback): CNRU, темы exit и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и пять пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/cnru/plateau_point.json`, файлы соседей плато
(`plateau_<поле>_<значение>.json`), `docs/superpowers/plans/task-11-report-cnru.md`

- [ ] **Step 1: Собрать точку правилом большинства** (≥ 3 из 4; иначе дефолт ядра); для каждого из
  восемнадцати полей записать голоса фолдов и решение, отмечая случайные совпадения с дефолтом.
  Поле тренда — из `trend` либо `trend_hump` по правилу Task 5 Step 3; поля объёмного гейта — из
  `volume` либо `vol_window` по правилу Task 8 Step 2.

- [ ] **Step 2: Применить риск-гейт A** к `min(StopDailyATR, TrailDailyATR при UseTrail = 1)`,
  потолок 1.0 (выживаемость 37.3%).

- [ ] **Step 3: Прогнать точку схемой 36/12/6.** Выписать pooled OOS PF, сделки пула, пофолдовые
  in-sample → OOS PF / сделки / NetPnL% / MaxDD%. **Baseline для сравнения — 1.989/122 при фолдах
  1.863/29, 2.535/37, 3.180/21, 1.214/35.**

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_point.json -out ./reports/CNRU_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 4: Проверить пункты 1 и 2 стоп-условия и дополнительное условие §5.9 (pooled OOS
  ≥ 1.989).** Провал любого — **не остановка работы**, а переключение задач 12–16 на **маршрут
  TGKA** (прод на дефолтах ядра). Числа доложить владельцу до продолжения.

- [ ] **Step 5: Контрольная схема 24/12/3 — пункт 3.** Baseline на том же куске даёт **1.705**:
  точка обязана быть не хуже дефолтов, а не просто выше единицы. Этот прогон дополнительно важен
  тем, что его окно (2024-09-08 …) целиком лежит на однородной сессии 34–35 баров.

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_point.json -out ./reports/CNRU_point_24 \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 6: Пункт 4 — та же схема 36/12/6 при удвоенном круге издержек**, плюс справочная
  строка при круге 0.3%. Baseline даёт 1.705 и 1.448 соответственно.

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_point.json -out ./reports/CNRU_point_cost2 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.001
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_point.json -out ./reports/CNRU_point_cost3 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.0015
```

- [ ] **Step 7: Риск-гейт B на полном окне**, потолок max DD **9.15%** (жёсткий). Этот же отчёт —
  источник журнала сделок для Steps 8 и 9.

```bash
go run ./cmd/backtest -ticker CNRU -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cnru/plateau_point.json -out ./reports/CNRU_point_full \
  -months 36 -min-trades 20 -metric profit_factor
```

- [ ] **Step 8: Анатомия точки против шести чисел третьего контура и пункт 5 стоп-условия.**
  Опорные числа baseline: доля SL **21.3%**, удержание **9 / 20 / 35**, ночёвок **48.8%**, выходов
  в выходную сессию **6.1%**, max DD **9.15%**, expectancy **504.37 ₽**. Проверить подпись капкана
  (падение доли SL при росте удержания и ночёвок → в точку идёт более узкий стоп).

  Разобрать журнал сделок отчёта Step 7 этим скриптом (подставить фактический путь отчёта):

```bash
python3 - <<'PY'
import collections, datetime
p = "reports/CNRU_point_full/<файл отчёта>.md"
rows = [l for l in open(p) if l.startswith("| ") and l.count("|") > 12]
byy = collections.defaultdict(lambda: [0, 0.0, 0.0, 0.0])
reasons = collections.Counter(); bars = []; nights = 0; wknd = 0; tot = 0
for l in rows:
    c = [x.strip() for x in l.strip().strip("|").split("|")]
    if not c[0].isdigit():
        continue
    entry, ex, reason, nb, pnl = c[1], c[3], c[5], int(c[6]), float(c[7])
    y = entry[:4]
    byy[y][0] += 1; byy[y][1] += pnl
    if pnl > 0: byy[y][2] += pnl
    else: byy[y][3] += -pnl
    reasons[reason] += 1; bars.append(nb); tot += 1
    if entry[:10] != ex[:10]: nights += 1
    if datetime.date.fromisoformat(ex[:10]).weekday() >= 5: wknd += 1
print("ГОД | СДЕЛОК | NET | PF")
for y in sorted(byy):
    n, net, gp, gl = byy[y]
    print(f"{y} | {n} | {net:+.0f} | {(gp/gl if gl else float('inf')):.3f}")
bars.sort()
print("выходы:", dict(reasons), "из", tot)
print("баров: медиана", bars[len(bars)//2], "p90", bars[len(bars)*9//10], "max", bars[-1])
print("ночёвок %.1f%%" % (100*nights/tot), "выходов в выходные %.1f%%" % (100*wknd/tot))
PY
```

  **Пункт 5 стоп-условия:** годы 2024 и 2025 обязаны быть прибыльны безусловно; годы 2023 и 2026
  неполные — убыточный неполный год триггерит пункт, только если в нём ≥ 10 сделок. Сравнение с
  baseline: 2023 +2 528 / 6 сделок, 2024 +22 942 / 56, 2025 +45 937 / 60, 2026 +11 310 / 42.

  **Обязательный замер выходов в выходную сессию:** медиана оборота выходного дня 3.92 млн ₽, доля
  выходов baseline 6.1%; рост доли записывается как реализованный риск проскальзывания.

- [ ] **Step 9: Два обязательных замера, специфичных для CNRU.** Тот же журнал, тот же отчёт.

```bash
python3 - <<'PY'
import datetime
p = "reports/CNRU_point_full/<файл отчёта>.md"
rows = [l for l in open(p) if l.startswith("| ") and l.count("|") > 12]
risky = {"2025-12-12", "2025-04-07", "2026-06-22", "2024-06-13", "2024-09-02", "2023-12-05"}
h0, h1 = datetime.date(2025, 2, 5), datetime.date(2025, 4, 3)
gap_hits, halt_hits = [], []
for l in rows:
    c = [x.strip() for x in l.strip().strip("|").split("|")]
    if not c[0].isdigit():
        continue
    e = datetime.date.fromisoformat(c[1][:10]); x = datetime.date.fromisoformat(c[3][:10])
    for day in risky:
        d = datetime.date.fromisoformat(day)
        if e < d <= x:
            gap_hits.append((day, c[1], c[3], c[5], c[7]))
    if e <= h1 and x >= h0 and (x - e).days > 3:
        halt_hits.append((c[1], c[3], c[5], c[7], c[6]))
print("ЭКСПОЗИЦИЯ ЧЕРЕЗ ГЭПЫ:", gap_hits or "НОЛЬ")
print("ЭКСПОЗИЦИЯ ЧЕРЕЗ ОСТАНОВКУ ТОРГОВ:", halt_hits or "НОЛЬ")
PY
```

  **Замер 1 — дивидендный гэп 2025-12-12 (−16.18%).** У baseline экспозиция нулевая. Каждая сделка
  точки, удерживаемая через эту ночь, записывается в доку пакета с её PnL как реализованный риск.

  **Замер 2 — остановка торгов 2025-02-05 → 2025-04-03.** У baseline экспозиция нулевая.
  **Ненулевая экспозиция точки — запрет:** такая точка не принимается независимо от PF, и в неё
  идёт ближайший вариант плато без этого свойства (Step 10). Если запрет сработал, факт и цена
  замены в PF пишутся прямым текстом.

- [ ] **Step 10: Соседи плато** по каждому полю, ушедшему от дефолта (± один узел оси); край сетки
  дополнительно проверяется зондом за краем; соседа по оси стопа при включённом трейле проверять
  **в сторону уменьшения**. Плато шириной меньше 0.05 PF записывается как отсутствие сигнала;
  правило ничьих (разница pooled OOS < 0.05 → меньшая просадка) применяется механически. Файлы
  соседей — одноточечные сетки `plateau_<поле>_<значение>.json` в том же каталоге, команда как в
  Task 9 Step 3.

- [ ] **Step 11: Написать `docs/superpowers/plans/task-11-report-cnru.md`** со всеми выкладками,
  командами и таблицами: голоса фолдов по каждому из восемнадцати полей; оба walk-forward против
  baseline; три строки издержек; календарные годы; анатомия; оба риск-гейта; оба специфичных замера;
  соседи плато; вердикт по планке; вердикт по пяти пунктам и по порогу 1.989.

- [ ] **Step 12: Коммит** `feat(rsi_pullback): CNRU, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Выбор маршрута делается по числам Task 11 Step 4 и не требует нового решения:**
- **Маршрут «литерал»** — если не сработал ни один из пяти пунктов стоп-условия, оба риск-гейта
  пройдены, замер Step 9 не дал экспозиции через остановку торгов, и pooled OOS точки **≥ 1.989**.
- **Маршрут TGKA** — во всех остальных случаях: `DefaultParams()` остаётся равным
  `core.DefaultParams()`. Прод при этом выполняется полностью (задачи 13–16), вердикт — «калибровка
  ничего не нашла, тикер заведён на дефолтах ядра».

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/cnru/cnru.go`,
`cnru_test.go`, `internal/service/backtest/rsi_pullback_registry_test.go`; маршрут TGKA
дополнительно — `internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

- [ ] **Step 1 (маршрут «литерал»): написать падающие тесты.** Заменить
  `TestParamsTrackTheBaselineUntilCalibrated` на `TestParamsAreTheAcceptedPoint` (снимок
  восемнадцати полей принятой точки) плюс `TestPointDiffersFromTheCoreBaseline`.

  **(маршрут TGKA): тесты пакета не меняются.** Вместо этого: `CNRU` добавляется в список
  `baselineByDesignTickers` в
  `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (именно там живёт именное
  исключение из сторожевого теста вселенной, требующего расхождения литерала с дефолтами), с
  комментарием-обоснованием по образцу строки TGKA.

- [ ] **Step 2: Убедиться, что тесты падают.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/cnru/ -v`
  Expected: FAIL (маршрут «литерал»); маршрут TGKA этот шаг пропускает.

- [ ] **Step 3 (маршрут «литерал»): поставить литерал принятой точки в `DefaultParams()`.**

- [ ] **Step 4: Заменить тест реестра** `TestRSIPullbackCNRUTracksBaseline` на
  `TestRSIPullbackCNRUServesTheCalibratedPoint` (маршрут «литерал») либо на
  `TestRSIPullbackCNRUServesTheCoreBaselineByDesign` по образцу
  `TestRSIPullbackTGKAServesTheCoreBaselineByDesign`
  (`internal/service/backtest/rsi_pullback_registry_test.go:889`) — маршрут TGKA.

- [ ] **Step 5: Прогнать тесты и линтер.**
  Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... && ./bin/golangci-lint run ./internal/...`
  Expected: PASS

- [ ] **Step 6: Коммит** `feat(rsi_pullback): CNRU откалиброван — литерал вместо отслеживания baseline`
  либо `feat(rsi_pullback): CNRU заводится на дефолтах ядра — маршрут TGKA`.

---

### Task 13: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`

- [ ] **Step 1:** добавить импорт и строку `cnru.Ticker: cnru.DefaultParams(),` в `paramsByTicker`
  (в алфавитном порядке).

- [ ] **Step 2:** дописать абзац комментария карты в англоязычном стиле соседей: вердикт по планке;
  оба риск-гейта и их цена; исход пяти пунктов стоп-условия и порога 1.989; и **принятые риски**:
  - **остановка торгов на 57 дней внутри окна** (редомициляция) — многодневная стратегия без
    тайм-стопа может оказаться в запертой позиции, биржевая стоп-заявка в приостановленном
    инструменте не исполняется, механизма выхода у раннера нет; экспозиция baseline и точки нулевая,
    но это факт истории, а не механизм;
  - **дивидендный гэп −16.18% (2025-12-12)** при 48.8% ночёвок; компания перешла к выплатам после
    редомициляции, отсечки будут повторяться;
  - капкан широкого стопа начинается на самом дефолте, гейт A против него бессилен;
  - расширение сессии вдвое внутри окна (19 → 34 бара);
  - тонкая выходная сессия (3.92 млн ₽ при 6.1% выходов baseline).

  **Условия пересмотра, которые надо записать явно:** медиана оборота будних дней за 6 месяцев ниже
  50 млн ₽ (сейчас 102.91, гейт вселенной скринера); цена ниже 200 ₽ (тогда круг из двух шагов по
  0.2 ₽ превысит 0.2%, то есть уровень пункта 4 стоп-условия).

- [ ] **Step 3: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/...`
  Expected: PASS

- [ ] **Step 4: Коммит** `feat(rsi_pullback): CNRU в реестре живого раннера`.

---

### Task 14: Боевая вселенная

**Files:** Modify `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`,
`env/prod.env`, `env/prod.env.example`, `env/local.env.example`, `docs/rsi_pullback/live.md`

- [ ] **Step 1: Перечитать числа Task 11** и записать в сообщении коммита, каким маршрутом заведён
  тикер («литерал» или TGKA), с pooled OOS точки и порогом 1.989.

- [ ] **Step 2:** добавить `"CNRU"` в `want` теста конфига **двадцать седьмым** (после `MVID`).

- [ ] **Step 3: Убедиться, что тест падает.**
  Run: `go test ./internal/config/ -run RSIPullback -v`
  Expected: FAIL

- [ ] **Step 4:** дописать `"CNRU"` в `Tickers` + комментарий-абзац; дописать `,CNRU` в
  `env/prod.env`, `env/prod.env.example`, `env/local.env.example`.

- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  `docs/rsi_pullback/live.md` (только значение дефолта — пер-тикерных записей туда не делать).

- [ ] **Step 6: Прогнать тесты.**
  Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...`
  Expected: PASS

- [ ] **Step 7: Коммит** `feat(rsi_pullback): завести CNRU в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/cnru/cnru.go`
(док-комментарий), возможно `docs/rsi_pullback/strategy.md`

- [ ] **Step 1: Переписать шапку пакета в итог калибровки:**
  - вердикт по планке по каждому критерию обеих ключевых тем (`entry` и `trend`);
  - точка поле за полем с голосами фолдов, с пометкой случайных совпадений с дефолтом;
  - оба walk-forward со сравнением с baseline (1.989/122 и 1.705);
  - три строки издержек (0.1% модели, 0.2% пункта 4, 0.3% справочно) и отдельная запись о том, что
    **реальный круг 0.064% ниже модельного** — модель пессимистична в 1.6 раза;
  - календарные годы точки против годов baseline и исход пункта 5;
  - соседи плато; анатомия против шести чисел baseline, включая долю выходов в выходную сессию;
  - результат обоих риск-гейтов и их цена;
  - **оба специфичных замера — экспозиция через дивидендный гэп и через остановку торгов.**

  Отдельными абзацами — принятые риски и условия пересмотра из §7 спеки: остановка торгов внутри
  окна и отсутствие механизма выхода; дивидендный гэп и повторяемость отсечек; капкан широкого
  стопа на самом дефолте; зона редкости объёмной оси; расширение сессии вдвое; получасовой ряд
  короче окна на месяц; слабость выигрыша калибровки над сильным baseline; свежая редомициляция.

- [ ] **Step 2: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/...`
  Expected: PASS

- [ ] **Step 3: Механический вывод в `docs/rsi_pullback/strategy.md` — только если CNRU дал вывод о
  механике.** Кандидат назван заранее: **остановка торгов как риск многодневной стратегии**.
  Каталог не имел ни одного тикера с многомесячной остановкой внутри окна, и ядро не содержит
  механизма, который закрыл бы позицию перед приостановкой. Если замер Task 11 Step 9 показал, что
  точка удерживала позицию через дыру, это вывод о механике стратегии, а не о CNRU, и он идёт в доку
  стратегии. Пер-тикерных чисел, дат и вердиктов в `docs/rsi_pullback/` не писать.

- [ ] **Step 4: Коммит** `docs(rsi_pullback): разбор калибровки CNRU и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** Run: `go run ./cmd/pullparity -tickers CNRU -months 24`
  Expected: ноль расхождений (на 24 месяцах, не на 36 — урок VSMO).

- [ ] **Step 2:** Run: `./bin/mage ci`
  Expected: зелёный.

- [ ] **Step 3:** Run: `git status --porcelain | grep -c '^.. reports/'`
  Expected: `0` (каталог `reports/` в `.gitignore`).

- [ ] **Step 4:** финальный коммит, если остались правки.

---

## Self-Review

**Покрытие спеки.** §1 (инструмент, шаг цены, остановка торгов, расширение сессии, ликвидность,
волатильность, дивидендный гэп, режим) → Global Constraints и Task 2 Step 3; §2 (окно и число
фолдов) → Global Constraints и Task 3 Step 2; §3 (baseline и открытый маршрут TGKA) → Global
Constraints, Task 11 Steps 3–6 и Task 12; §4 (рельеф осей) → `_comment` в Task 1 Step 3 и ожидания
Tasks 3–10; §5.1 (одиннадцать тем и четыре отличия сеток) → Task 1 и Tasks 3–10; жёсткие
инварианты → Task 1 Step 1; §5.2 (правило сборки и приоритет тем) → Task 11 Step 1, Task 5 Step 3,
Task 8 Step 2; §5.3 (соседи плато) → Task 11 Step 10; §5.4 (гейт A) → Task 9 Step 2, Task 10 Step 3,
Task 11 Step 2; §5.5 (гейт B жёсткий и правило ничьих) → Task 11 Steps 7 и 10; §5.6 (третий контур
и два специфичных замера) → Task 11 Steps 8 и 9; §5.7 (планка) → Task 4 Step 2 и Task 5 Step 2;
§5.8 (пять пунктов, включая календарный год) → Task 11 Steps 4–6 и 8; §5.9 (порог 1.989 и маршрут
TGKA) → Task 11 Step 4 и Task 12; §5.10 (правило прода) → Task 14 Step 1; §6 (артефакты) →
Tasks 1, 2, 9, 11–15; §7 (принятые риски) → Task 13 Step 2 и Task 15 Step 1; `pullparity` и
`mage ci` → Task 16.

**Плейсхолдеры.** Два, и оба неизбежны по построению процедуры: литерал точки в Task 12 (точка
известна только после Task 11) и путь к отчёту в скриптах Task 11 Steps 8–9 (имя файла содержит
метку времени прогона). Все прочие шаги несут конкретные оси, команды, пороги и числа сравнения.

**Согласованность имён.** `cnru.Ticker` и `cnru.DefaultParams()` заводятся в Task 2 и используются
в Tasks 12 и 13; одиннадцать файлов сеток из Task 1 читаются задачами 3–10 по тем же путям;
`plateau_stop_06.json`, `plateau_stop_07.json`, `plateau_stop_10.json`, `plateau_tp_03.json`
заводятся в Task 9 и упоминаются только там; `plateau_point.json` и файлы соседей — Task 11; тесты
`TestCNRUGridsStayWide` (Task 1), `TestParamsTrackTheBaselineUntilCalibrated`, `TestTickerIsCNRU` и
`TestRSIPullbackCNRUTracksBaseline` (Task 2) → `TestParamsAreTheAcceptedPoint`,
`TestPointDiffersFromTheCoreBaseline` и `TestRSIPullbackCNRUServesTheCalibratedPoint` (Task 12,
маршрут «литерал») либо `TestRSIPullbackCNRUServesTheCoreBaselineByDesign` плюс строка в
`baselineByDesignTickers` (Task 12, маршрут TGKA).
