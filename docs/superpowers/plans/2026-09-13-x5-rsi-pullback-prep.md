# X5 под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести X5 (ПАО «Корпоративный центр ИКС 5») до вердикта по стратегии `rsi_pullback`:
каталог максимально широких сеток, тематический walk-forward на схеме сборки, принятая точка,
прошедшая **три** риск-гейта и **шесть** пунктов стоп-условия на **трёх** схемах проверки, при
провале — второй круг по правилам владельца, и либо литерал в пакете с заведением в боевую
вселенную тридцать первым тикером, либо протокол отказа.

**Architecture:** Процедура каноническая, но исходная позиция отличается от всего каталога тем, что
**у бумаги нет ни одной привычной проблемы и есть одна своя**. Издержки ниже модели (реальный круг
0.055% против 0.1% модели), ликвидность в 26 раз выше гейта вселенной и не падает, сессия однородна
на всём окне (35 баров в буднем дне во всех четырёх полугодиях), дефолты ядра уже прибыльны
(PF 1.374, net +9 800 ₽). Взамен весь реализованный убыток сконцентрирован в одном механизме:
**девять выходов через часы метки 02–06, где стакан в 200–800 раз тоньше основной сессии, дают
−7 812 ₽**, и один из них — исполнение по печати на пустом стакане (2026-01-02 02:00, Open 2730 при
Close 3036 и объёме 1137 лотов), стоившее **−9.62% капитала одной сделкой**. Отсюда четыре отличия
от каталога: окно равно всей истории инструмента после редомициляции (20 месяцев, досуспензионные
расписки FIVE не используются); точка собирается на схеме **8/3** — единственной, которая даёт
четыре фолда и держит печать вне обучающего окна, тогда как схема 12/2 объявлена льстивой до
прогонов (её pooled OOS на дефолтах 3.408 против 1.246 у 8/3); вводится **третий риск-гейт C** —
экспозиция точки в тонкую сессию; главным критерием прода объявлена **прибыльность обоих
календарных лет по сырым числам**, которую дефолты проваливают.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков), `python3` для
разбора журналов сделок.

**Spec:** `docs/superpowers/specs/2026-09-13-x5-rsi-pullback-prep-design.md`
**Замеры:** `reports/_analysis/x5_pullback_prep_measurements.md`

## Global Constraints

- **Ветка:** `feat/x5-pullback-prep` от `main` (HEAD `c077695`, вселенная из 30 тикеров). Ветка уже
  создана, спека уже закоммичена (`c4bb52b`).
- **Таймфрейм `Minutes30`** во всех прогонах; **`-interval Minutes30` обязателен в каждой команде**
  (дефолт CLI — `Hour1`, забытый флаг даёт чужие числа, а не ошибку).
- **Расчётное окно одно для всех прогонов: `-months 20`** (2025-01-13 … 2026-09-13). Это вся
  история инструмента после редомициляции; расширять назад некуда, сокращать — нечего.
  Досуспензионные расписки FIVE (до 2024-04-03) не используются ни в одном прогоне.
- **Схема тематических прогонов первого круга:** `-months 20 -train-months 8 -test-months 3
  -min-trades 20 -metric profit_factor`; у темы `screen` — `-min-trades 1`. Все одиннадцать тем идут
  по этой схеме и только по ней: иначе темы несопоставимы между собой.
- **Три схемы проверки ТОЧКИ (не тем), окно у всех `-months 20`:**
  - **8/3** (`-train-months 8 -test-months 3`) — **4 фолда**, схема сборки и вердикта, держит печать
    2026-01-02 в OOS (фолд 2);
  - **9/3** (`-train-months 9 -test-months 3`) — **3 фолда**, контроль того же механизма;
  - **12/2** (`-train-months 12 -test-months 2`) — **4 фолда**, объявлена льстивой: её OOS
    начинается 2026-01-13, через одиннадцать дней после печати, и покрывает только 2026 год.
- **`-refresh` НЕ запускать:** ряды дотянуты 2026-09-13 (`X5_Minutes30.json` — 22 827 баров,
  2025-01-13 22:30 … 2026-09-13 22:00, **разрывов нет**; `X5_Day1.json` — 604 бара, 2024-01-15 …
  2026-09-11, с разрывом редомициляции 2024-04-03 → 2025-01-09, который лежит до начала окна).
- **Число фолдов сверяется на первой же теме:** «Фолдов: 4» в шапке отчёта темы `screen` — иначе
  остановиться и доложить владельцу (ловушка ASTR). Дефолты дают 4 / 3 / 4 фолда на трёх схемах,
  проверено фактическим прогоном.
- **Единственный разрыв торгов внутри окна** — новогодняя пауза 2025-12-30 → 2026-01-05, и это
  место события спеки §2.3. Дивидендные гэпы около −10%: 2025-07-09, 2026-01-06, 2026-07-07; сделок
  через отсечку у дефолтов **ноль**.
- **Лот 1**, шаг цены **0.5 ₽**, реальный круг издержек **0.055%** при последней цене 1817 ₽ —
  модель (`-commission 0.0005`, круг 0.1%) **пессимистична в 1.8 раза**.
- **Сетки максимально широкие**; обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
  инварианты: `RSILower ≤ 50`; `RSIPeriod ≥ 2`; `StopDailyATR` нигде не ноль; ни `cal_trend.json`,
  ни `cal_trend_hump.json` не порождают пар `EMAFast ≥ EMASlow`; ось `RSILower` темы `entry`
  содержит оба края 5 и 50; ось `EMASlow` темы `trend_hump` содержит 12 и 45; ось `EMASlow` темы
  `trend` содержит 250; ось `RSIUpper` темы `exit` содержит 35 и 95; ось `StopDailyATR` темы `risk`
  содержит 0.35, 0.45, 0.55, 0.65 и верхний край 2.0; ось `VolLookbackBars` темы `vol_window`
  содержит 12, 16, 24 и верхний край 32; ось `FreshDayATR` темы `day_fresh` содержит 0.05, 0.15,
  0.25; ось `FreshDayATR` темы `day` содержит 0 и 0.5; ось `TPDailyATR` темы `risk` содержит верхний
  край 2.0; файла `cal_trend_volume.json` в каталоге **нет**.
- **Риск-гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **0.7**. Порог объявлен до
  прогонов и не двигается.
- **Риск-гейт B:** max DD точки на полном 20-месячном окне ≤ **9.73% (10 645.76 ₽)**.
- **Риск-гейт C (только у X5):** экспозиция точки в бары с часом метки **02–06** не хуже дефолтов:
  входов **0**, выходов **9**, суммарный PnL сделок, закрытых в этих барах, **−7 812 ₽**.
- **Шесть пунктов стоп-условия первого круга** (§5.11 спеки). Круг останавливается, если точка даёт:
  1. pooled OOS PF < 1.0 на схеме 8/3;
  2. меньше 20 сделок в пуле OOS схемы 8/3;
  3. pooled OOS PF < 1.0 на схеме 9/3;
  4. pooled OOS PF < 1.0 на схеме 8/3 при `-commission 0.001`;
  5. хотя бы один календарный год окна убыточен **по сырым числам** (2025 с 13 января, 2026 до
     13 сентября);
  6. провал риск-гейта C.
- **Пункта про круг 0.3% нет** (модель издержек пессимистична); замер при `-commission 0.0015`
  всё равно снимается и идёт в отчёт.
- **Числа дефолтов, против которых меряется всё** (полное 20-месячное окно): сделок **76**,
  PF **1.374**, net **+9 799.98 ₽**, max DD **10 645.76 ₽ (9.73%)**, win rate **76.32%**,
  expectancy **+128.95 ₽**, худшая сделка **−10 453.45 ₽**. Анатомия выходов: RSI 56, **SL 12
  (15.8%)**, TP 8. Удержание медиана/p90/максимум **9/20/49** баров, ночёвок **44.7%**, переносов
  через два и более дня **2**, входов в выходные **0**, выходов в выходные **5**. Календарные годы:
  2025 — 42 сделки, **−1 202 ₽, PF 0.947**; 2026 — 34 сделки, +11 002 ₽, PF 3.951. Walk-forward:
  **8/3 → 1.246 на пуле 49**, 9/3 → 1.234 на 41, 12/2 → 3.408 на 33. Под `-commission 0.001`:
  8/3 → **0.945**, 9/3 → 0.967, 12/2 → 2.243.
- **Задачи прода (13–16) не выполняются**, пока вердикт не вынесен по §5.13 спеки.

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/x5/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_hump.json`, `cal_day.json`, `cal_day_fresh.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_x5_grid_test.go`

**Interfaces:**
- Produces: одиннадцать путей `data/params/rsi_pullback/x5/cal_*.json`, которые читают задачи 3–10;
  тест `TestX5GridsStayWide`.

- [ ] **Step 1: Написать падающий сторожевой тест осей.** Образец —
  `internal/service/backtest/rsi_pullback_irkt_grid_test.go` (перечисляет файлы **поимённо**, а не
  обходом каталога). Тест `TestX5GridsStayWide` читает файлы каталога и проверяет **все**
  инварианты из Global Constraints:
  - `RSILower ≤ 50` во всех файлах, и ось `RSILower` в `cal_entry.json` содержит оба края `5` и `50`;
  - `RSIPeriod ≥ 2` везде;
  - `StopDailyATR != 0` везде, где ось присутствует;
  - ни `cal_trend.json`, ни `cal_trend_hump.json` не порождают пар `EMAFast ≥ EMASlow` (перебрать
    декартово произведение осей внутри каждого файла);
  - ось `EMASlow` в `cal_trend_hump.json` содержит `12` и `45`;
  - ось `EMASlow` в `cal_trend.json` содержит `250`;
  - ось `RSIUpper` в `cal_exit.json` содержит `35` и `95`;
  - ось `StopDailyATR` в `cal_risk.json` содержит `0.35, 0.45, 0.55, 0.65` и верхний край `2.0`;
  - ось `TPDailyATR` в `cal_risk.json` содержит верхний край `2.0`;
  - ось `VolLookbackBars` в `cal_vol_window.json` содержит `12, 16, 24` и верхний край `32`;
  - ось `FreshDayATR` в `cal_day_fresh.json` содержит `0.05, 0.15, 0.25`;
  - ось `FreshDayATR` в `cal_day.json` содержит `0` и `0.5`;
  - файла `cal_trend_volume.json` в каталоге **нет** (тема не заводится — проверить явно, чтобы
    исполнитель будущей правки не завёл её молча).

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/backtest/ -run TestX5GridsStayWide -v`
  Expected: FAIL — файлов сеток нет.

- [ ] **Step 3: Создать одиннадцать файлов сеток.** Оси дословно (§5.1 спеки):
  - `cal_screen.json`: `UseDayATRGate` [0,1] × `UseVolume` [0,1] — **4** прогона.
  - `cal_entry.json`: `RSIPeriod` [2,3,4,5,6,7,8,10,12] × `RSILower`
    [5,10,15,20,25,30,35,40,45,50] — **90**.
  - `cal_trend.json`: `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] — **48**.
  - `cal_trend_hump.json`: `EMAFast` [3,5,8,10] × `EMASlow` [12,15,18,20,25,30,35,40,45] — **36**.
  - `cal_day.json`: `FreshDayATR` [0,0.1,0.2,0.3,0.4,0.5] × `SpentDayATR`
    [0.5,0.6,0.7,0.8,0.9,1.0,1.1,1.2,1.35,1.5,2.0] — **66**.
  - `cal_day_fresh.json`: `FreshDayATR` [0,0.05,0.1,0.15,0.2,0.25] × `SpentDayATR` [0.8] — **6**.
  - `cal_volume.json`: `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,1.8,2.0,2.2,2.5,3.0] ×
    `VolBaseDays` [3,5,10,14,20,30,40,50] — **64**.
  - `cal_vol_window.json`: `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult`
    [1.0,1.2] — **18**.
  - `cal_risk.json`: `StopDailyATR` [0.3,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.8,1.0,1.2,1.5,2.0] ×
    `TPDailyATR` [0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.0] — **112**.
  - `cal_exit.json`: `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90,95] — **13**.
  - `cal_trail.json`: `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR`
    [0.3,0.4,0.5,0.6,0.7,1.0,1.5] — **14**.

  Формат файла — как у `data/params/rsi_pullback/irkt/cal_screen.json`: объект с `_comment` и
  массивом `phases`, каждая фаза — `{"name": ..., "grid": {...}, "keepTop": N}`.

  **Каждый `_comment` содержит замер оси на 20-месячном окне поверх дефолтов ядра** (у X5 одна
  шкала, а не две, как у IRKT, — окно однородно по сессии, и второй шкалы не существует).
  Обязательные замеры и предупреждения:
  - `cal_screen.json`: 0×0 → 0.973/298 (DD 28 774 ₽), 0×1 → 1.090/168 (DD 17 062),
    **1×0 → 1.374/76 (BASELINE, дефолт ядра, DD 10 646)**, 1×1 → **1.494/51 (DD 7341)**. Дневной
    гейт даёт **+0.40 PF** и режет просадку втрое; объёмный гейт поверх него улучшает **и PF, и
    просадку** — редкость каталога (прецедент NKHP; у IRKT, TRNFP и большинства тикеров вредит).
    **Ожидание, записанное заранее: победит узел 1×1, а не дефолтный 1×0.**
  - `cal_entry.json`: `RSILower` — 5 → 3983.429/**3**, 10 → 6.027/**12**, 15 → 2.259/24 (DD 4221),
    **20 → 2.536/41 (DD 6071, net +15 061)**, 25 → 1.347/57, **30 → 1.374/76 (дефолт, DD 10 646)**,
    35 → 1.193/98, 40 → 1.141/108, 45 → 1.056/121, 50 → 1.041/136. `RSIPeriod` — 2 → 0.924/204,
    3 → 1.163/118, **4 → 1.374/76 (дефолт)**, 5 → 1.308/52, **6 → 2.490/43 (DD 6420, net +18 043)**,
    7 → 1.852/35, 8 → 1.843/26, 10 → 2.343/**17**, 12 → 2.223/**11**. Узлы `RSILower` 5 и 10
    вырождены (3 и 12 сделок), узлы `RSIPeriod` 10 и 12 тонкие (17 и 11). `RSIPeriod` 14 и выше
    исключён. **Падение просадки с 10.6 до 6.1–6.4 тыс ₽ на максимумах обеих осей — подпись того,
    что более редкий вход разминается с печатью 2026-01-02, а не того, что риск на сделку меньше.**
    **Тема `entry_deep` не заводится, ось `RSILower` не уплотняется.**
  - `cal_trend.json` / `cal_trend_hump.json`: `EMASlow` при `EMAFast` = 10 — 12 → 1.221/65,
    15 → 1.184/68, 18 → 1.190/68, 20 → 1.231/69, 25 → 1.258/73, 30 → 1.303/75, **40 → 1.520/75**,
    50 → 1.488/80, 75 → 1.375/78, **100 → 1.374/76 (дефолт)**, 150 → 1.354/75, 200 → 1.361/78,
    250 → 1.356/77. `EMAFast` при `EMASlow` = 100 — 3 → 1.249/66, 5 → 1.377/70, **8 → 1.493/75**,
    **10 → 1.374/76 (дефолт)**, 15 → 1.342/80, 20 → 1.366/83, 30 → 1.366/83, 40 → 1.420/84. Размах
    осей 0.34 и 0.24 PF — сигнал слабый. **Вершина `EMASlow` стоит на 40, ниже нижнего края
    канонической темы 50 — отсюда разделение тем** (прецеденты AQUA, CNRU, MAGN, IRKT).
    **ПРЕДУПРЕЖДЕНИЕ: просадка не меняется ни на одном узле обеих осей (10.3–10.9 тыс ₽) — трендовая
    ось с печатью 2026-01-02 не разминается нигде, в отличие от осей входа и выхода.** `EMASlow`
    ниже 12 исключён как вырожденный. **Тема `trend_volume` НЕ заводится** (складывающегося сигнала
    двух рычагов нет; прецедент MAGN, где она OOS не пережила).
  - `cal_day.json` / `cal_day_fresh.json`: `SpentDayATR` — 0.5 → 1.095/168 (DD 22 174),
    0.6 → 1.248/134, 0.7 → 1.236/101, **0.8 → 1.374/76 (дефолт, ЛОКАЛЬНЫЙ МАКСИМУМ)**,
    0.9 → 1.238/65, 1.0 → 1.057/54, 1.1 → 1.228/39, 1.2 → 1.279/25, 1.35 → 0.813/15, 1.5 → 0.508/10,
    2.0 → 0.508/6 — **жёсткие пороги ось ломают**, и опасности прецедента TRNFP (схлопывание пула)
    здесь нет, в отличие от IRKT. `FreshDayATR` — **0 → 1.374/76 (дефолт)**, **0.1 → 1.523/92
    (net +14 993)**, 0.2 → 1.402/99, 0.3 → 1.036/142, 0.4 → 1.021/180 (DD 20 297), 0.5 → 0.993/218
    (DD 21 647). **Максимум на 0.1 узкий, соседний узел 0.2 уже ниже — отсюда отдельная тема
    `day_fresh` с шагом 0.05.**
  - `cal_volume.json` / `cal_vol_window.json`: `VolMult` при `UseVolume` = 1 — **1.0 → 1.537/55**,
    1.2 → 1.494/51, 1.5 → 1.376/42, 1.8 → 1.131/37, 2.0 → 0.926/33, 2.2 → 1.038/30, 2.5 → 0.990/27,
    3.0 → 0.996/23: работают только мягкие пороги, отсюда второй узел оси `VolMult` темы
    `vol_window` взят **1.0**, а не 1.8. `VolBaseDays` — **3 → 1.851/55 (DD 6486)**, 5 → 1.849/56,
    10 → 1.601/50, **14 → 1.494/51 (дефолт гейта)**, 20 → 1.137/42, 30 → 1.192/41, 40 → 1.527/38
    (DD 3518), 50 → 1.764/35 (DD 3847): ось немонотонна, максимумы на обоих краях.
    `VolLookbackBars` — 1 → 1.484/49, 2 → 1.472/49, **3 → 1.494/51 (дефолт)**, 5 → 1.828/59,
    8 → 1.932/64, 12 → **2.073/69 (DD 6231)**, **16 → 2.110/70 (DD 6231)**, 24 → 2.104/71,
    32 → 2.104/71. **Сильнейшая ось объёмного гейта в каталоге: монотонный рост до плато 12–32 при
    РАСТУЩЕМ пуле (51 → 70) и просадке на 41% ниже дефолта.** Поле зафиксировано в доке стратегии
    как несвипуемое в фазовом гриде, поэтому вторичная тема обязательна.
  - `cal_risk.json`: `StopDailyATR` — 0.3 → 1.127/79, 0.35 → 1.095/78, 0.4 → 1.201/77,
    0.45 → 1.368/77, **0.5 → 1.374/76 (дефолт)**, 0.55 → 1.523/75, 0.6 → 1.703/75, 0.7 → 1.683/75,
    0.8 → 1.768/74, 1.0 → 1.872/74, 1.2 → 2.337/74, 1.5 → **2.457/74**, 2.0 → 2.457/74. Прямой замер
    срабатываний защиты: 0.5 → **12** SL из 76 сделок (15.8%, DD 10 646 ₽); 0.7 → **6** из 75 (8.0%,
    DD 10 942); 1.0 → **4** из 74 (5.4%, DD 11 238); 1.5 → **1** из 74 (1.4%, DD 11 829).
    **Капкан широкого стопа в предельной форме: число сделок меняется на две по всей оси, PF растёт
    вдвое ровно по мере того, как защита перестаёт существовать. Просадка при этом РАСТЁТ — гейт B
    здесь не бессилен (в отличие от SPBE, RTKMP, IRKT), но запас до планки всего 11%, решает
    гейт A.** Узлы 0.8–2.0 остаются в сетке (требование максимальной ширины) и отсекаются гейтом A
    при сборке точки, а не вырезанием оси. `TPDailyATR` — 0.3 → 1.529/80, **0.4 → 1.559/78**,
    0.5 → 1.476/78, **0.6 → 1.374/76 (дефолт)**, 0.8 → 1.269/75, 1.0 → 1.283/75, 1.5 → 1.161/75,
    2.0 → 1.161/75: узлы 1.5 и 2.0 совпадают по числу сделок — цель за этими порогами не достигается
    (прецедент RENI), выше 2.0 ось не продолжается.
  - `cal_exit.json`: `RSIUpper` (в скобках max DD, ₽) — 35 → 1.568/105 (**2583**), 40 → 1.318/95
    (3456), 45 → 1.747/91 (3171), 50 → 1.902/88 (4520), 55 → 2.173/86 (4222), **60 → 2.889/85
    (3465, net +23 708)**, 65 → 1.668/82 (**11 238**), **70 → 1.374/76 (10 646, дефолт)**,
    75 → 1.440/75, 80 → 1.442/71, 85 → 1.461/69 (12 099), 90 → 1.120/67 (14 924), 95 → 1.100/67
    (16 900). **Сильнейшая ось бумаги и единственный инструмент риск-гейта C: порог просадки
    ломается ровно между 60 и 65 (3465 → 11 238 ₽), потому что при `RSIUpper` ≤ 60 позиция
    закрывается ДО печати 2026-01-02.**
  - `cal_trail.json`: `TrailDailyATR` при `UseTrail` = 1 — 0.3 → 1.074/82, 0.4 → 1.300/80,
    0.5 → 1.370/77, **0.6 → 1.486/77**, 0.7 → 1.466/76, 1.0 → 1.374/76, 1.5 → 1.374/76 (выше 1.0
    трейл неактивен). `UseRSIExit × TrailDailyATR` — 0×0.5 → 1.384/70 (DD 11 794), 0×0.8 → 1.203/67
    (DD 15 255), 1×0.5 → 1.370/77 (DD 10 646), 1×0.8 → 1.398/76 (DD 10 646). **Выключение
    RSI-выхода ухудшает просадку на всех узлах: на X5 RSI-выход является защитным механизмом, а не
    только механизмом прибыли. Ожидание, записанное заранее: побеждает `UseRSIExit = 1`, трейл даёт
    не больше 0.11 PF над дефолтом и гейт C не улучшает.**

  Каждый `_comment` обязан содержать **полную команду запуска** темы (этого требует
  `TestRSIPullbackCalFilesValid`), например для `cal_screen.json`:

```
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/x5/cal_screen.json -out ./reports/X5_screen -months 20 -min-trades 1 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ -run 'TestX5GridsStayWide|TestRSIPullbackCalFilesValid' -v`
  Expected: PASS

- [ ] **Step 5: Коммит.**

```bash
git add data/params/rsi_pullback/x5 internal/service/backtest/rsi_pullback_x5_grid_test.go
git commit -m "feat(rsi_pullback): каталог сеток X5"
```

---

### Task 2: Пакет `strategy/x5` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/x5/x5.go`,
  `internal/service/trading_strategy/rsi_pullback/strategy/x5/x5_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Produces: `x5.Ticker` (константа `"X5"`) и `x5.DefaultParams() core.Params` — их используют задачи
  13 и 14.

- [ ] **Step 1: Написать падающие тесты** по образцу пакета `irkt`
  (`internal/service/trading_strategy/rsi_pullback/strategy/irkt/irkt_test.go`):
  `TestParamsTrackTheBaselineUntilCalibrated` (`DefaultParams()` равен `core.DefaultParams()` поле за
  полем) и `TestTickerIsX5` (константа равна `"X5"`).

- [ ] **Step 2: Убедиться, что тесты падают.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/x5/ -v`
  Expected: FAIL — пакета нет.

- [ ] **Step 3: Создать пакет.** `x5.go` содержит `const Ticker = "X5"` и
  `func DefaultParams() core.Params { return core.DefaultParams() }`. Doc-comment пакета на этом
  этапе фиксирует состояние «калибровка не проводилась, литерал появится по §5.13 спеки» и
  содержимое §5.14 спеки в части, уже известной до калибровки: печать 2026-01-02 02:00 с механизмом
  и числами, профиль ликвидности внутри суток, расхождение схем на дефолтах (1.246 против 3.408),
  факт остановки торгов на редомициляции, пессимистичность модели издержек в 1.8 раза, дивидендные
  гэпы и нулевое число сделок через отсечку.

- [ ] **Step 4: Зарегистрировать пакет в реестре бэктеста.** В
  `internal/service/backtest/rsi_pullback_registry.go` добавить импорт
  `rsipullbackx5 "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/x5"` и строку
  `rsipullbackx5.Ticker: rsiPullbackBindingFor(rsipullbackx5.Ticker, rsipullbackx5.DefaultParams),`
  в карту привязок. В `rsi_pullback_registry_test.go` добавить тест
  `TestRSIPullbackX5IsRegisteredAndUncalibrated` по образцу соседних
  `Test...IsRegisteredAndCalibrated`, проверяющий, что тикер в реестре и его параметры **равны
  дефолтам ядра** (на этом этапе литерала нет).

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/x5/ ./internal/service/backtest/ -run 'TestTickerIsX5|TestParamsTrackTheBaselineUntilCalibrated|TestRSIPullbackX5IsRegisteredAndUncalibrated|TestRSIPullbackRegistryEntriesMatchTheirTicker|TestRSIPullbackRegistryKeepsTheStopArmed|TestRSIPullbackTickersKeepTheRSIExitArmed' -v`
  Expected: PASS

- [ ] **Step 6: Коммит.**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/x5 internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "feat(rsi_pullback): пакет X5 до калибровки"
```

---

### Task 3: Тема `screen` — цена двух гейтов и сверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/x5/cal_screen.json` (дописать результат в `_comment`);
Create `docs/superpowers/plans/task-3-report-x5.md`

**Interfaces:**
- Consumes: `data/params/rsi_pullback/x5/cal_screen.json` из Task 1.
- Produces: голоса фолдов по осям `UseDayATRGate` и `UseVolume` — их читает Task 10.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_screen.json -out ./reports/X5_screen \
  -months 20 -min-trades 1 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 2: СВЕРИТЬ ЧИСЛО ФОЛДОВ.** В шапке отчёта должно стоять **«Фолдов: 4»**. Если стоит
  другое число — **остановиться и доложить владельцу**: это ловушка ASTR, и все последующие темы
  будут мерить не то окно.

- [ ] **Step 3: Записать результат:** pooled OOS PF и размер пула, пофолдовые OOS PF/сделок,
  победителя каждого фолда по обеим осям, голоса (сколько фолдов из четырёх за какое значение).

- [ ] **Step 4: Сверить с ожиданием**, записанным до прогона: **победит узел 1×1**. Записать
  «подтвердилось / не подтвердилось» прямым текстом с числами.

- [ ] **Step 5: Дописать результат в `_comment` файла** и написать отчёт
  `docs/superpowers/plans/task-3-report-x5.md`.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): X5, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая, первая половина планки

**Files:** Modify `data/params/rsi_pullback/x5/cal_entry.json`; Create
`docs/superpowers/plans/task-4-report-x5.md`

**Interfaces:**
- Produces: голоса фолдов по `RSIPeriod` и `RSILower`, pooled OOS PF темы и размер пула — их читают
  Task 10 (сборка точки) и Task 10 Step 12 (планка).

- [ ] **Step 1: Прогнать тему** (90 прогонов на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_entry.json -out ./reports/X5_entry \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 2: Записать результат:** pooled OOS PF, размер пула, пофолдовые OOS PF/сделок,
  победителя каждого фолда по обеим осям, голоса по ведущей оси `RSILower`.

- [ ] **Step 3: Проверить первую половину планки** (§5.9 спеки): pooled OOS PF ≥ 1.5 при ≥ 20
  сделках в пуле **и** `RSILower` выбран одинаково в ≥ 3 фолдах из 4. Ожидание, записанное до
  прогона: **эту половину планка возьмёт**. Записать факт.

- [ ] **Step 4: Проверить, не ушёл ли победитель на вырожденный узел.** Если фолд выбрал
  `RSILower` 5 или 10 (3 и 12 сделок на полном окне) либо `RSIPeriod` 10 или 12 (17 и 11 сделок),
  записать это прямым текстом как подозрение на подгонку под фолд, а не как сигнал.

- [ ] **Step 5: Дописать результат в `_comment`** и написать отчёт
  `docs/superpowers/plans/task-4-report-x5.md`.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): X5, тема entry`.

---

### Task 5: Темы `trend` и `trend_hump` — вторая половина планки

**Files:** Modify `data/params/rsi_pullback/x5/cal_trend.json`, `cal_trend_hump.json`; Create
`docs/superpowers/plans/task-5-report-x5.md`

**Interfaces:**
- Produces: голоса фолдов по `EMAFast`/`EMASlow` из обеих тем и их pooled OOS PF — их читает Task 10
  (правило приоритета `trend_hump` над `trend`).

- [ ] **Step 1: Прогнать каноническую тему.**

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_trend.json -out ./reports/X5_trend \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 2: Прогнать тему горба.**

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_trend_hump.json -out ./reports/X5_trend_hump \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 3: Записать результаты обеих тем** (pooled OOS PF, пул, пофолдовые числа, победители,
  голоса) и **сравнить их pooled OOS между собой** — это вход правила приоритета §5.2 спеки.

- [ ] **Step 4: Проверить вторую половину планки** по **канонической** теме `trend` (не
  `trend_hump`): pooled OOS ≥ 1.5 при ≥ 20 сделках и `EMASlow` одинаков в ≥ 3 фолдах из 4. Ожидание,
  записанное до прогона: **эту половину планка НЕ возьмёт** (вся ось лежит в полосе 1.18–1.52 с
  размахом 0.34 PF). Записать факт.

- [ ] **Step 5: Дописать результаты в оба `_comment`** и написать отчёт
  `docs/superpowers/plans/task-5-report-x5.md`.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): X5, темы trend и trend_hump`.

---

### Task 6: Темы `day` и `day_fresh`

**Files:** Modify `data/params/rsi_pullback/x5/cal_day.json`, `cal_day_fresh.json`; Create
`docs/superpowers/plans/task-6-report-x5.md`

**Interfaces:**
- Produces: голоса фолдов по `FreshDayATR` и `SpentDayATR` из обеих тем и их pooled OOS PF — их
  читает Task 10 (правило приоритета `day_fresh` над `day`).

- [ ] **Step 1: Прогнать тему `day`** (66 прогонов на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_day.json -out ./reports/X5_day \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 2: Прогнать тему `day_fresh`** (6 прогонов на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_day_fresh.json -out ./reports/X5_day_fresh \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 3: Записать результаты обеих тем** и сравнить их pooled OOS между собой.

- [ ] **Step 4: Проверить, не ушёл ли `SpentDayATR` в зону тонкого пула.** Узлы 1.35, 1.5 и 2.0
  дают на полном окне 15, 10 и 6 сделок; победа любого из них в фолде записывается прямым текстом
  как угроза пункту 2 стоп-условия (≥ 20 сделок пула OOS). Прецедент TRNFP, где тема `trend_low`
  схлопнула пул со 155 до 35 сделок.

- [ ] **Step 5: Дописать результаты в оба `_comment`** и написать отчёт
  `docs/superpowers/plans/task-6-report-x5.md`.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): X5, темы day и day_fresh`.

---

### Task 7: Темы `volume` и `vol_window`

**Files:** Modify `data/params/rsi_pullback/x5/cal_volume.json`, `cal_vol_window.json`; Create
`docs/superpowers/plans/task-7-report-x5.md`

**Interfaces:**
- Produces: голоса фолдов по `UseVolume`, `VolMult`, `VolBaseDays`, `VolLookbackBars` и pooled OOS
  обеих тем — их читает Task 10 (правило приоритета `vol_window` над `volume`).

- [ ] **Step 1: Прогнать тему `volume`** (64 прогона на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_volume.json -out ./reports/X5_volume \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 2: Прогнать тему `vol_window`** (18 прогонов на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_vol_window.json -out ./reports/X5_vol_window \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 3: Записать результаты обеих тем** и сравнить их pooled OOS между собой.

- [ ] **Step 4: Сверить с априором бумаги.** На X5 объёмный гейт — один из двух реально работающих
  рычагов (1×1 → 1.494/51 против baseline 1.374/76; `VolLookbackBars` 16 → 2.110/70 при просадке на
  41% ниже дефолта). Если темы голосуют за выключение гейта или за инертные узлы, записать
  расхождение с априором прямым текстом: это тот же урок, что дала тема `screen` на IRKT — цена
  гейта, измеренная на полном окне, не обязана переноситься в train-окна фолдов.

- [ ] **Step 5: Дописать результаты в оба `_comment`** и написать отчёт
  `docs/superpowers/plans/task-7-report-x5.md`.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): X5, темы volume и vol_window`.

---

### Task 8: Тема `risk`, риск-гейт A и зонды капкана

**Files:** Modify `data/params/rsi_pullback/x5/cal_risk.json`; Create
`data/params/rsi_pullback/x5/probe_stop_10.json`, `probe_stop_15.json`, `probe_stop_20.json`;
Create `docs/superpowers/plans/task-8-report-x5.md`

**Interfaces:**
- Produces: голоса фолдов по `StopDailyATR` и `TPDailyATR`, значение стопа после применения гейта A
  и числа зондов капкана — их читает Task 10.

- [ ] **Step 1: Прогнать тему** (112 прогонов на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_risk.json -out ./reports/X5_risk \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 2: Записать результат** (pooled OOS, пул, пофолдовые числа, победители, голоса по обеим
  осям).

- [ ] **Step 3: Применить риск-гейт A к голосу темы.** Потолок **0.7**. Если тема голосует выше —
  записать это как **сработавший капкан**, поставить в точку 0.7 и посчитать цену решения в PF
  (разница pooled OOS темы на голосе фолдов против значения 0.7). Порог не двигается.

- [ ] **Step 4: Снять зонды капкана.** Создать три файла точки с дефолтами ядра и единственным
  изменённым полем `StopDailyATR` = 1.0, 1.5, 2.0 (формат — как `plateau_*.json` у IRKT: одна фаза,
  все поля пинуются одним значением) и прогнать каждый одиночным прогоном:

```bash
for p in probe_stop_10 probe_stop_15 probe_stop_20; do
  go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
    -calibrate data/params/rsi_pullback/x5/$p.json -out ./reports/X5_probe \
    -months 20 -min-trades 1 -metric profit_factor
done
```

  Для каждого зонда разобрать журнал сделок скриптом на `python3` и записать: число сделок, число
  **SL-выходов**, PF, max DD. Опорные числа из замеров: 0.5 → 12 SL / DD 10 646 ₽; 0.7 → 6 /
  10 942; 1.0 → 4 / 11 238; 1.5 → 1 / 11 829. **Зонды документируют, что за потолком гейта A защита
  перестаёт существовать, и попадают в отчёт независимо от того, куда проголосовала тема.**

- [ ] **Step 5: Дописать результат в `_comment`** и написать отчёт
  `docs/superpowers/plans/task-8-report-x5.md`.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): X5, тема risk и зонды капкана`.

---

### Task 9: Темы `exit` и `trail`

**Files:** Modify `data/params/rsi_pullback/x5/cal_exit.json`, `cal_trail.json`; Create
`docs/superpowers/plans/task-9-report-x5.md`

**Interfaces:**
- Produces: голоса фолдов по `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` — их читает
  Task 10, причём `RSIUpper` является инструментом риск-гейта C.

- [ ] **Step 1: Прогнать тему `exit`** (13 прогонов на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_exit.json -out ./reports/X5_exit \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 2: Прогнать тему `trail`** (14 прогонов на фолд).

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal_trail.json -out ./reports/X5_trail \
  -months 20 -min-trades 20 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 3: Записать результаты обеих тем** и сверить с ожиданиями, записанными до прогонов:
  тема `trail` должна выбрать `UseRSIExit = 1`, а трейл не должен давать больше 0.11 PF над
  дефолтом.

- [ ] **Step 4: Отдельно записать порог просадки оси `RSIUpper`.** Для узлов 55, 60, 65 и 70
  снять одиночные прогоны на полном окне и записать max DD каждого. Опорные числа: 55 → 4222 ₽,
  60 → 3465, 65 → 11 238, 70 → 10 646. **Это вход риск-гейта C: при `RSIUpper` ≤ 60 позиция
  закрывается до печати 2026-01-02, и именно этой осью гейт C чинится, если точка его провалит.**

- [ ] **Step 5: Дописать результаты в оба `_comment`** и написать отчёт
  `docs/superpowers/plans/task-9-report-x5.md`.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): X5, темы exit и trail`.

---

### Task 10: Сборка точки, три риск-гейта, три walk-forward и шесть пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/x5/plateau_point.json` и файлы соседей плато
(`plateau_<поле>_<значение>.json` по одному на каждого соседа); Create
`reports/_analysis/x5_exposure.py`; Create `docs/superpowers/plans/task-10-report-x5.md`

**Interfaces:**
- Consumes: результаты задач 3–9 (голоса фолдов по каждой теме).
- Produces: принятую точку первого круга (восемнадцать полей) и вердикт — их читают задачи 11 и 13;
  скрипт `x5_exposure.py`, который читает отчёт бэктеста и печатает три числа гейта C.

- [ ] **Step 1: Написать скрипт замера экспозиции** `reports/_analysis/x5_exposure.py`. Он
  принимает путь к markdown-отчёту прогона, парсит журнал сделок (строки вида `| N | вход | цена |
  выход | цена | причина | баров | PnL | ...`) и печатает: (а) число входов в бары с часом метки
  02–06, (б) число выходов в такие бары, (в) суммарный PnL сделок, закрытых в таких барах, плюс
  общий список этих сделок. Проверить скрипт на отчёте дефолтов: он обязан выдать **0 / 9 /
  −7 812 ₽**; если выдал другое — скрипт неверен, чинить его, а не числа.

- [ ] **Step 2: Собрать точку правилом большинства.** Поле берётся из темы, которая его меряет, и
  принимается только при **≥ 3 голосах из 4** фолдов схемы 8/3; иначе — дефолт ядра. Ничья 2/2
  большинством не считается. Приоритеты тем: `trend_hump` над `trend` только при большинстве **и**
  превосходстве pooled OOS; `vol_window` над `volume` по тому же правилу; `day_fresh` над `day` по
  тому же правилу. Записать таблицу «поле → тема-источник → голоса → принятое значение → дефолт
  ядра» для всех восемнадцати полей.

- [ ] **Step 3: Применить риск-гейт A.** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤
  **0.7**. Если точка вышла выше — заменить на 0.7 и записать цену решения в PF.

- [ ] **Step 4: Снять одиночный прогон точки на полном окне.**

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/plateau_point.json -out ./reports/X5_point \
  -months 20 -min-trades 1 -metric profit_factor
```

- [ ] **Step 5: Применить риск-гейт B.** Max DD точки на полном 20-месячном окне ≤ **9.73%
  (10 645.76 ₽)**. Если выше — точка не принимается, и в неё идёт ближайший вариант плато с меньшей
  просадкой (первый кандидат замены — узел `RSIUpper` из нижней половины оси, см. Task 9 Step 4);
  замена записывается прямым текстом.

- [ ] **Step 6: Применить риск-гейт C.** Прогнать `reports/_analysis/x5_exposure.py` на отчёте из
  Step 4 и сравнить три числа с дефолтными **0 / 9 / −7 812 ₽**. Точка проходит, если (а) не
  выросло, (б) не выросло и (в) не ухудшилось. **При провале точка берёт более раннее значение
  `RSIUpper`** (узлы 60 и ниже, см. Task 9 Step 4), и замена записывается прямым текстом с ценой
  решения в PF.

- [ ] **Step 7: Снять соседей плато.** Для каждого поля, ушедшего от дефолта, — одиночный прогон
  самого поля и двух его соседей по оси на полном 20-месячном окне. Значение на краю сетки
  проверяется **зондом за краем**; соседа по оси стопа при включённом трейле проверяют **в сторону
  уменьшения** (§8 доки стратегии). Плато шириной меньше 0.05 PF записывается как отсутствие
  сигнала. При разнице pooled OOS < 0.05 в точку идёт вариант с меньшей просадкой. **Для каждого
  соседа снять и три числа гейта C** скриптом из Step 1 — на X5 разница между хорошей и плохой
  точкой чаще проходит не по PF, а по тому, ночует ли позиция в час без ликвидности.

- [ ] **Step 8: Снять третий контур.** Разобрать журнал сделок точки скриптом на `python3` и
  получить: долю SL-выходов, удержание (медиана / p90 / максимум баров), долю ночёвок, число
  переносов через два и более дня, входы и выходы в выходные, max DD, expectancy. Сравнить с
  числами дефолтов из Global Constraints. **Падение доли SL-выходов вместе с ростом удержания и
  ночёвок — подпись капкана**; в этом случае точка берёт более узкий стоп.

- [ ] **Step 9: Прогнать ТРИ walk-forward точки.**

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/plateau_point.json -out ./reports/X5_point \
  -months 20 -min-trades 1 -train-months 8 -test-months 3 -metric profit_factor
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/plateau_point.json -out ./reports/X5_point \
  -months 20 -min-trades 1 -train-months 9 -test-months 3 -metric profit_factor
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/plateau_point.json -out ./reports/X5_point \
  -months 20 -min-trades 1 -train-months 12 -test-months 2 -metric profit_factor
```

  Ожидаемое число фолдов: **4, 3, 4**. Записать pooled OOS PF, пул и пофолдовые числа каждой схемы.
  **Схема 12/2 записывается вместе с напоминанием, что печать 2026-01-02 у неё в train, и её
  похвала вердикта не меняет.**

- [ ] **Step 10: Прогнать точку под утяжелёнными издержками.**

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/plateau_point.json -out ./reports/X5_point \
  -months 20 -min-trades 1 -train-months 8 -test-months 3 -metric profit_factor -commission 0.001
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/plateau_point.json -out ./reports/X5_point \
  -months 20 -min-trades 1 -train-months 8 -test-months 3 -metric profit_factor -commission 0.0015
```

  Первая команда — пункт 4 стоп-условия (порог 1.0). Вторая — справочный замер: пункта про круг
  0.3% на X5 нет, потому что реальный круг бумаги 0.055%, но число идёт в отчёт и в доку пакета.

- [ ] **Step 11: Посчитать календарные годы точки.** Разобрать журнал сделок прогона из Step 4
  скриптом на `python3`, сгруппировав по году входа: число сделок, net PnL, PF. Критерий владельца
  (§5.8 спеки): **оба года прибыльны по сырым числам**, без изъятий — сделка 2025-12-31 → 2026-01-02
  входит в расчёт 2025 года. Опорные числа дефолтов: 2025 — 42 сделки, −1 202 ₽, PF 0.947; 2026 —
  34 сделки, +11 002 ₽, PF 3.951.

- [ ] **Step 12: Применить шесть пунктов стоп-условия** (Global Constraints). Записать по каждому
  пункту фактическое число и вердикт «взят / не взят». **Пункты 5 (календарные годы) и 6 (гейт C) —
  главные и нестандартные.**

- [ ] **Step 13: Проверить планку** — обе канонические темы (`entry` из Task 4, `trend` из Task 5)
  дают pooled OOS ≥ 1.5 при ≥ 20 сделках и одинаковую ведущую ось в ≥ 3 фолдах из 4. Сверить с
  ожиданием «`entry` возьмёт, `trend` не возьмёт».

- [ ] **Step 14: Написать отчёт** `docs/superpowers/plans/task-10-report-x5.md` (образец —
  `docs/superpowers/plans/task-10-report-irkt.md`): таблица сборки точки по полям, три риск-гейта с
  ценой решений, три walk-forward, два уровня издержек, календарные годы, соседи плато с их числами
  гейта C, третий контур, вердикт по каждому из шести пунктов, вердикт по планке.

- [ ] **Step 15: Вынести вердикт.** Если **ни один** пункт стоп-условия не сработал и все три
  риск-гейта пройдены — переходить к задаче 13 (задачи 11 и 12 пропускаются). Иначе — переходить к
  задаче 11.

- [ ] **Step 16: Коммит** `feat(rsi_pullback): X5, точка первого круга и вердикт`.

---

### Task 11: Второй круг — узкие сетки по правилам владельца (только при провале первого)

**Files:** Create `data/params/rsi_pullback/x5/cal2_*.json` (по одной узкой сетке на каждую тему,
чьё поле было принято в точку первого круга или соседствует с лучшей зоной); Create
`data/params/rsi_pullback/x5/plateau_point2.json`; Create
`docs/superpowers/plans/task-11-report-x5.md`

**Задача выполняется ТОЛЬКО при срабатывании хотя бы одного пункта стоп-условия в Task 10.**

**Interfaces:**
- Consumes: голоса фолдов первого круга (задачи 3–9) и вердикт Task 10.
- Produces: точку второго круга — её читают задачи 12 и 13.

- [ ] **Step 1: Выбрать зоны узких сеток по фактическим голосам фолдов первого круга**, а не заново
  по in-sample рельефу. Для каждой темы, чьё поле участвовало в точке или было близко к победе,
  построить сетку из 3–5 узлов вокруг зоны голосования с шагом вдвое мельче исходного. Записать
  прямым текстом, какой голос какого фолда породил каждую зону.

- [ ] **Step 2: Прогнать узкие темы** по той же схеме, что первый круг, **но со снятым порогом
  числа сделок** (прямое указание владельца: во втором круге число сделок не критерий выбора):

```bash
go run ./cmd/backtest -ticker X5 -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/x5/cal2_<тема>.json -out ./reports/X5_r2_<тема> \
  -months 20 -min-trades 1 -train-months 8 -test-months 3 -metric profit_factor
```

- [ ] **Step 3: Собрать точку второго круга** тем же правилом большинства ≥ 3/4 и записать таблицу
  «поле → тема → голоса → значение».

- [ ] **Step 4: Применить все три риск-гейта без изменений** (A — потолок 0.7; B — max DD ≤ 9.73%;
  C — экспозиция не хуже 0 / 9 / −7 812 ₽ скриптом `reports/_analysis/x5_exposure.py`).

- [ ] **Step 5: Прогнать три walk-forward и издержки** — команды те же, что в Task 10 Step 9 и
  Step 10, с файлом `plateau_point2.json`.

- [ ] **Step 6: Посчитать календарные годы** точки второго круга — критерий владельца остаётся
  главным и не смягчается: **оба года прибыльны по сырым числам**.

- [ ] **Step 7: Применить стоп-условие второго круга.** Пункты 1, 3, 4, 5, 6 в силе полностью;
  **пункт 2 (≥ 20 сделок пула OOS) снят**, но фактический размер пула записывается обязательно, и
  **пул меньше 10 сделок закрывает работу как непредставительный**.

- [ ] **Step 8: Написать отчёт** `docs/superpowers/plans/task-11-report-x5.md` с полным набором
  чисел и вердиктом.

- [ ] **Step 9: Вынести вердикт.** Точка второго круга прошла — переходить к задаче 13. Не прошла —
  переходить к задаче 12.

- [ ] **Step 10: Коммит** `feat(rsi_pullback): X5, второй круг`.

---

### Task 12: Протокол отказа (только при провале обоих кругов)

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/x5/x5.go` (doc-comment);
Create `docs/superpowers/plans/task-12-report-x5.md`

**Задача выполняется ТОЛЬКО если и первый, и второй круг не дали проходящей точки.**

- [ ] **Step 1: Записать в doc-comment пакета `x5` протокол отказа:** какие пункты стоп-условия
  сработали с фактическими числами, какие риск-гейты не прошли, вердикт «в прод не заводится» и
  условия возможного возврата к тикеру. Обязательное содержимое §5.14 спеки идёт в доку **в полном
  объёме** — отказ не отменяет записи.

- [ ] **Step 2: Убедиться, что литерал НЕ поставлен** — `DefaultParams()` по-прежнему возвращает
  `core.DefaultParams()`, тикера нет ни в реестре живого раннера, ни в `RSI_PULLBACK_TICKERS`.

- [ ] **Step 3: Прогнать полный набор тестов.**
  Run: `./bin/mage ci`
  Expected: PASS

- [ ] **Step 4: Коммит** `docs(rsi_pullback): X5, протокол отказа` и **остановиться**: задачи 13–16
  не выполняются, владельцу докладываются числа.

---

### Task 13: Литерал в пакете и снимок

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/x5/x5.go`,
`internal/service/trading_strategy/rsi_pullback/strategy/x5/x5_test.go`,
`internal/service/backtest/rsi_pullback_registry_test.go`

**Задача выполняется ТОЛЬКО при вынесенном положительном вердикте (Task 10 Step 15 или Task 11
Step 9).**

**Interfaces:**
- Consumes: принятую точку из Task 10 или Task 11.
- Produces: `x5.DefaultParams()`, возвращающий откалиброванный литерал, — его читают задачи 14 и 15.

- [ ] **Step 1: Заменить тест пакета на снимок литерала.** Тест
  `TestParamsTrackTheBaselineUntilCalibrated` удаляется, вместо него пишется
  `TestParamsMatchTheCalibratedSnapshot`, сверяющий `DefaultParams()` с принятой точкой **поле за
  полем** (все восемнадцать полей выписаны литералами, а не вычислены из `core.DefaultParams()`).
  Образец — `internal/service/trading_strategy/rsi_pullback/strategy/irkt/irkt_test.go`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/x5/ -v`
  Expected: FAIL — литерал ещё не поставлен.

- [ ] **Step 3: Поставить литерал.** `DefaultParams()` возвращает `core.Params` со всеми
  восемнадцатью полями, выписанными **явно**, даже теми, что совпали с дефолтом ядра: пакет обязан
  быть читаемым без обращения к ядру.

- [ ] **Step 4: Обновить тест реестра бэктеста.** `TestRSIPullbackX5IsRegisteredAndUncalibrated`
  заменяется на `TestRSIPullbackX5IsRegisteredAndCalibrated` по образцу соседних тестов
  откалиброванных тикеров: реестр отдаёт именно литерал пакета, а не дефолты ядра.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'X5|RSIPullback' -v`
  Expected: PASS

- [ ] **Step 6: Коммит** `feat(rsi_pullback): литерал X5`.

---

### Task 14: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`,
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `x5.Ticker`, `x5.DefaultParams()` из Task 13.
- Produces: `ParamsFor("X5")`, отдающий литерал, — его проверяет Task 17 через `cmd/pullparity`.

- [ ] **Step 1: Написать падающий тест** по образцу теста IRKT в
  `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (строки 184–191):
  `ParamsFor(x5.Ticker)` возвращает ровно `x5.DefaultParams()`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run X5 -v`
  Expected: FAIL — тикера нет в реестре.

- [ ] **Step 3: Добавить тикер в реестр.** Импорт
  `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/x5"` и строка
  `x5.Ticker: x5.DefaultParams(),` в карту реестра.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
  Expected: PASS

- [ ] **Step 5: Коммит** `feat(rsi_pullback): X5 в реестре живого раннера`.

---

### Task 15: Боевая вселенная

**Files:** Modify `internal/config/rsi_pullback.go:605`, `env/local.env.example:28`,
`env/prod.env.example:30`

- [ ] **Step 1: Написать падающий тест** (или расширить существующий тест конфига), проверяющий,
  что `"X5"` присутствует в списке тикеров по умолчанию и что список содержит **31** элемент.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/config/ -run RSIPullback -v`
  Expected: FAIL

- [ ] **Step 3: Добавить `"X5"` последним элементом** в `Tickers` в `internal/config/rsi_pullback.go`
  и в обе строки `RSI_PULLBACK_TICKERS=` в `env/local.env.example` и `env/prod.env.example`.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/config/ -v`
  Expected: PASS

- [ ] **Step 5: Коммит** `feat(rsi_pullback): завести X5 в боевую вселенную`.

---

### Task 16: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/x5/x5.go` (doc-comment)

- [ ] **Step 1: Написать разбор в doc-comment пакета.** Обязательное содержимое §5.14 спеки:
  - печать **2026-01-02 02:00** (Open 2730.00, Close 3036.00, объём 1137 лотов) с механизмом
    биржевой стоп-заявки и числом **−10 453.45 ₽ (−9.62% капитала)**;
  - профиль ликвидности внутри суток (часы метки 02–06 → 24–91 лот против 15–21 тысячи в часы
    10–17) и **три числа риск-гейта C для принятой точки**;
  - расхождение схем на дефолтах (**1.246** на 8/3 против **3.408** на 12/2) как свойство границы
    train/test, а не бумаги;
  - факт остановки торгов на редомициляции (2024-04-03 → 2025-01-09) и то, что расчётное окно равно
    всей истории инструмента;
  - пессимистичность модели издержек в **1.8 раза** (реальный круг 0.055% против 0.1% модели) и
    вытекающее отсутствие пункта про круг 0.3%;
  - дивидендные гэпы около −10% (2025-07-09, 2026-01-06, 2026-07-07) и измеренный факт, что ни одна
    сделка дефолтов через отсечку не удерживалась;
  - таблицу сборки точки, оба вердикта (стоп-условие и планка), цену решений всех трёх риск-гейтов.

  **Правило документации CLAUDE.md: пер-тикерный разбор живёт в doc-comment пакета и в `_comment`
  файлов сеток, а НЕ в `docs/rsi_pullback/`.** Файлы `docs/rsi_pullback/strategy.md`, `live.md`,
  `screener.md` этой задачей не трогаются.

- [ ] **Step 2: Прогнать линтер.**
  Run: `./bin/mage lint`
  Expected: PASS

- [ ] **Step 3: Коммит** `docs(rsi_pullback): разбор калибровки X5 и принятый риск`.

---

### Task 17: Финальная проверка

**Files:** нет (только прогоны)

- [ ] **Step 1: Прогнать полный гейт.**
  Run: `./bin/mage ci`
  Expected: PASS (lint + `go test -race ./...` + проверка дрейфа моков)

- [ ] **Step 2: Сверить живую сборку с бэктестом.**

```bash
go run ./cmd/pullparity -ticker X5
```

  Expected: ноль расхождений между параметрами живого раннера и литералом пакета.

- [ ] **Step 3: Сверить состав вселенной.** Убедиться, что `RSI_PULLBACK_TICKERS` в
  `internal/config/rsi_pullback.go` и в обоих `env/*.example` содержит **31** тикер и что X5 в них
  последний.

- [ ] **Step 4: Доложить владельцу** итоговые числа: точку, все три walk-forward, оба календарных
  года, три числа гейта C, вердикт по шести пунктам и по планке.

- [ ] **Step 5: Коммит** (если что-то поправлено финальной проверкой) и остановка — мерж ветки
  делает владелец.

---

## Self-Review

**Покрытие спеки задачами:**

| Раздел спеки | Задача |
|---|---|
| §1 Инструмент и данные | Global Constraints, Task 2 Step 3 (дока), Task 16 |
| §2 Тонкая сессия | Task 10 Step 1 (скрипт), Step 6 (гейт C), Task 16 |
| §3 Расчётные окна | Global Constraints, Task 3 Step 2 (сверка фолдов), Task 10 Step 9 |
| §4 Baseline | Global Constraints (числа), Task 10 Steps 5, 8, 11 (сравнение) |
| §5.1 Одиннадцать тем | Task 1, Tasks 3–9 |
| §5.2 Правило сборки | Task 10 Step 2 |
| §5.3 Соседи плато | Task 10 Step 7 |
| §5.4 Риск-гейт A | Task 8 Step 3, Task 10 Step 3 |
| §5.5 Риск-гейт B | Task 10 Step 5 |
| §5.6 Риск-гейт C | Task 10 Steps 1, 6, 7; Task 11 Step 4 |
| §5.7 Третий контур | Task 10 Step 8 |
| §5.8 Календарные годы | Task 10 Step 11, Task 11 Step 6 |
| §5.9 Планка | Task 4 Step 3, Task 5 Step 4, Task 10 Step 13 |
| §5.10 Схемы проверки точки | Task 10 Step 9 |
| §5.11 Стоп-условие | Task 10 Step 12 |
| §5.12 Второй круг | Task 11 |
| §5.13 Правило прода | Task 10 Step 15, Tasks 13–15 |
| §5.14 Что записывается | Task 2 Step 3, Task 12 Step 1, Task 16 Step 1 |
| §6 Свойства бумаги | Task 1 Step 3 (замеры в `_comment`) |
| §7 Артефакты | Tasks 1, 2, 10, 16 |
| §8 Риски | Task 16 Step 1 |

Гапов нет.

**Плейсхолдеры:** проверено — каждый шаг содержит либо точную команду, либо точный список чисел,
либо точный список инвариантов. «Обработать ошибки» и «написать тесты для вышеперечисленного» не
встречаются.

**Согласованность имён:** `x5.Ticker` и `x5.DefaultParams()` определены в Task 2 Step 3 и
используются в Task 13 Step 3, Task 14 Step 3 — одни и те же имена. Тест
`TestRSIPullbackX5IsRegisteredAndUncalibrated` создаётся в Task 2 Step 4 и заменяется в Task 13
Step 4 на `TestRSIPullbackX5IsRegisteredAndCalibrated` — переход описан явно. Тест
`TestParamsTrackTheBaselineUntilCalibrated` создаётся в Task 2 Step 1 и удаляется в Task 13 Step 1 —
переход описан явно. Скрипт `reports/_analysis/x5_exposure.py` создаётся в Task 10 Step 1 и
используется в Task 10 Steps 6, 7 и Task 11 Step 4 — один и тот же путь.
