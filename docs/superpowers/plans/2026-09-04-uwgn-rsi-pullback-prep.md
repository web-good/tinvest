# UWGN под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести UWGN (ПАО «Объединённая вагонная компания») до вердикта по стратегии
`rsi_pullback`: каталог максимально широких сеток, тематический walk-forward по схеме 30/10/5,
принятая точка, прошедшая два риск-гейта и четыре пункта стоп-условия, литерал в пакете и
заведение в боевую вселенную двадцать пятым тикером — если стоп-условие не сработало.

**Architecture:** Процедура каноническая по составу тем (одиннадцать тем поверх дефолтов ядра,
якоря нет), но окно и схема адаптированы: внутри канонических 36 месяцев стоит допэмиссия ОВК
(2023-11-13 … 11-24 цена 63.0 → 21.3 ₽, −66% за девять сессий) и обратный вынос 2023-12 … 2024-01
(24.9 → 76.1 ₽, +206%), поэтому расчётное окно начинается 2024-03-05 и схема — **30/10/5** (четыре
фолда, проверено контрольным прогоном до написания спеки), плюс обязательный контроль **24/12/3**.
Два риск-гейта: гейт выживаемости применяется к эффективной защите
`min(StopDailyATR, TrailDailyATR при UseTrail=1)` с потолком 1.0; гейт просадки ужесточён решением
владельца — max DD точки **не выше max DD baseline** (16.35%), а не 1.3×. **Особенность UWGN:**
шаг цены 0.02 ₽ при цене 20.6 ₽ даёт круг 0.187% против моделируемых 0.1%, поэтому стоп-условие
получает четвёртый пункт по издержкам, и **дефолты ядра проваливают пункты 3 и 4 уже сейчас**.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-04-uwgn-rsi-pullback-prep-design.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех без исключения прогонах. Флаг `-interval Minutes30` обязателен в
  каждой команде: дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Схема прогонов:** `-months 30 -train-months 10 -test-months 5 -min-trades 20 -metric
  profit_factor`, четыре фолда встык. У темы `screen` — `-min-trades 1`.
- **Расчётное окно 2024-03-05 … 2026-09-04, 30.0 месяца** (31 593 бара m30 внутри окна). Начало
  окна — решение владельца от 2026-09-04: допэмиссия ноября 2023 и январский вынос 2024 выброшены
  целиком, а не взвешены. Числа UWGN сравнимы построчно только с DIAS (30/10/5).
- **Число фолдов сверяется на первой же теме.** Контрольные прогоны 2026-09-04 уже дали «Фолдов: 4»
  и на 30/10/5 (pooled OOS 1.047), и на 24/12/3 (0.718), но сверка повторяется: если в шапке отчёта
  темы `screen` окажется «Фолдов: 3» — **остановиться и доложить владельцу**, остальные темы не
  запускать (ловушка ASTR).
- **`-refresh` НЕ запускать ни на одном шаге плана.** Кэш уже починен 2026-09-04:
  `UWGN_Minutes30.json` — 34 580 баров (2023-08-04 … 2026-09-04), `UWGN_Day1.json` — 1 133 свечи.
  До починки в получасовом ряду отсутствовали **29 будних дней** подряд (2025-03-03 … 2025-04-10)
  при полном дневном ряде за те же дни; после починки в окне нет ни одного отсутствующего дня и ни
  одного буднего дня с числом баров меньше десяти. **Все числа этого плана сняты на починенном
  кэше.**
- **ПРОЦЕДУРА КАНОНИЧЕСКАЯ по составу:** все одиннадцать тем поверх дефолтов ядра, якоря нет.
- **Сетки держатся максимально широкими.** Обрезок рабочих осей не делается; замеры, которые в узком
  каталоге были бы основанием вырезать край, идут в `_comment` как **предупреждения**. Жёсткие
  инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower` на всех парах темы `entry`,
  ось тренда не порождает пар `EMAFast ≥ EMASlow` ни в одном из двух файлов, `StopDailyATR` нигде не
  равен нулю, ось `TPDailyATR` содержит 0.2.
- **Два отличия сеток от каталожных, каждое по замеру бумаги:** ось цели расширена **вниз до 0.2**
  (максимум оси стоит на нижнем крае 0.3 → 1.111), и это одновременно зонд за краем; ось стопа
  остаётся широкой **до 2.0**, хотя замер показывает капкан — обрезает её не сетка, а риск-гейт A,
  чтобы решение было видно в отчёте.
- **Правило сборки точки:** поле берётся из темы, которая его меряет, и принимается только если за
  значение высказались **не менее трёх фолдов из четырёх**; иначе поле остаётся на **дефолте ядра**.
  Ничья 2/2 большинством не считается. Поле тренда идёт из `trend_low` только при большинстве ≥ 3/4
  **и** превосходстве pooled OOS над канонической `trend`. Голоса `RSIUpper` из темы `entry` в
  сборку не идут.
- **РИСК-ГЕЙТ A — выживаемость эффективной защиты.** Принимается только защита, достижимая **не
  менее чем в 30% будних дней окна**, гейт применяется к `min(StopDailyATR, TrailDailyATR при
  UseTrail=1)` (урок AFKS). Таблица выживаемости UWGN (n = 635): 0.3 — 99.7%, 0.4 — 97.5%,
  0.5 — 92.0%, 0.6 — 80.0%, 0.7 — 70.2%, 0.8 — 59.1%, **1.0 — 41.4%**, 1.3 — 24.7%, 1.5 — 18.1%,
  2.0 — 8.7%. **Потолок — 1.0.** Если тема голосует выше, в точку идёт ближайшее значение оси
  внутри гейта, а цена решения в PF пишется прямым текстом. Порог 30% после начала прогонов не
  двигается.
- **РИСК-ГЕЙТ B — просадка точки (ужесточён).** Max DD принятой точки на полном окне (одиночный
  прогон) **не превышает max DD baseline — 16.35%**. Каталожная норма 1.3× здесь не применяется:
  решение владельца 2026-09-04, основание — просадка baseline UWGN втрое выше каталожной нормы, а
  единственная растящая PF ось (широкий стоп) просадку и увеличивает.
- **Третий контур — риск-профиль точки против baseline.** Опорные числа baseline: доля SL-выходов
  **30.1%**, удержание медиана **11** / p90 **20** / максимум **46** баров, ночёвок **42.3%**,
  выходов в выходную сессию **5.7%**, max DD **16.35%**. Точка меряется по всем пяти. Выходная
  сессия — отдельный риск: её медианный оборот 8.12 млн ₽, проскальзывание там не моделируется.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` (каноническая, не
  `trend_low`) обе дают pooled OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для
  `entry`, `EMASlow` для `trend`) выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд в пользу
  тикера не засчитывается.
- **Стоп-условие из ЧЕТЫРЁХ пунктов.** Работа останавливается, если принятая точка даёт: (1) pooled
  OOS PF < 1.0 на 30/10/5, **либо** (2) меньше 20 сделок в пуле OOS, **либо** (3) pooled OOS PF < 1.0
  на контрольном прогоне 24/12/3, **либо** (4) pooled OOS PF < 1.0 на 30/10/5 при `-commission
  0.001`. При срабатывании — числа владельцу, **задачи 12–16 не выполняются**, маршрут (второй круг
  по прецеденту SOFL или закрытие по прецедентам HEAD, AFKS, KMAZ) выбирает владелец. **Маршрут
  «дефолты ядра» по прецеденту TGKA на UWGN недоступен: дефолты сами проваливают пункты 3 и 4.**
- **Издержки — ПУНКТ СТОП-УСЛОВИЯ.** Шаг цены **0.02 ₽** (1 208 наблюдений минимальной разности
  соседних уровней за последний год, все прочие разности ей кратны) при цене 20.64 ₽ даёт круг
  **0.187%** против моделируемых 0.1%: модель **оптимистична**. Чувствительность baseline:
  0.1% → **1.055** (+5 353 ₽), 0.2% → **0.929** (−6 839 ₽), 0.3% → 0.815, 0.4% → 0.713.
- **Правило прода:** литерал ставится и UWGN заводится в `RSI_PULLBACK_TICKERS` двадцать пятым,
  **если не сработал ни один пункт стоп-условия** — независимо от того, взята планка или нет
  (прецеденты WUSH, LENT, NKHP и ещё девять тикеров подряд).
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают все темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`, `UseDayATRGate 1`,
  `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`, `UseVolume 0`,
  `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`, `UseTrail 0`,
  `TrailDailyATR 0`. Контрольный прогон дефолтов на расчётном окне: **123 сделки, PF 1.055, net
  +5 352.54 ₽ (+5.35%)**, win rate 63.41%, max DD 20 593.27 ₽ (16.35%), выходы RSI 71 (57.7%) /
  SL 37 (30.1%) / TP 15 (12.2%), удержание 11/20/46 баров, ночёвок 42.3%, выходов в выходную сессию
  5.7%, **expectancy 43.52 ₽ = 0.043% капитала на сделку**. Walk-forward дефолтов: 30/10/5 →
  **1.047/85** (фолды 2.686/23, 0.430/19, 0.623/27, 0.891/16 — прибылен один из четырёх);
  24/12/3 → **0.718**, compounded −11.05%.
- **Полугодия baseline** (нужны в Task 11): 2024-03..2024-09 — 24 сделки, PF 0.984, −345 ₽;
  2024-09..2025-03 — 31, **1.896, +17 148**; 2025-03..2025-09 — 19, 0.973, −395;
  2025-09..2026-03 — 33, **0.603, −8 853**; 2026-03..2026-09 — 16, 0.891, −2 203. **Прибыльно одно
  полугодие из пяти, и весь baseline сделан окном сентябрь 2024 — март 2025.**
- **Ликвидность.** Медиана оборота будних дней внутри окна — **138.5 млн ₽**; по скользящим
  интервалам от даты запуска 157.4 (36 мес), 104.2 (24), **63.0 (12)**; по годам 680 (2023) →
  402 (2024) → 118 (2025) → **52.8 (2026)** при гейте вселенной 50 млн. Выходная сессия: 260 дней,
  медиана 8.12 млн ₽. Дневной ATR(14) медиана **3.80%** внутри окна (6.24% в отчёте скринера за 36
  месяцев — максимум прошедшей вселенной). Априор скринера (отчёт 2026-08-04): **25-е место**,
  PFmed 1.38, holdout PFmed 5.84 на 6 сделках, Plateau 50%, зажатых 0/24, молчащих 0/24, лучшая
  конфигурация RSI 6/15, EMA 20/100, TP 1.0.
- **Режим:** buy&hold за окно **−65.1%** (59.20 → 20.64 ₽), по полугодиям −30.9%, +36.5%, −31.5%,
  −25.5%, −28.7% — одно растущее полугодие из пяти.
- **Сессия расширилась внутри окна**: медиана баров в буднем дне 29 → 34 → 35 → 35 → 35. Отсюда
  контрольный прогон 24/12/3 и пункт 3 стоп-условия.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; полную команду запуска с путём
  `data/params/rsi_pullback/uwgn/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); место под
  строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md). Пер-тикерный
  разбор живёт в доке пакета `strategy/uwgn` и в `_comment` файлов сеток.
- **Каталог `reports/` — в `.gitignore`.**
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/uwgn-pullback-prep` от `feat/vsmo-pullback-prep` (`dba4d9e`), спека уже
  закоммичена (`3f2f3c7`).

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/uwgn/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_uwgn_grid_test.go`

**Interfaces:**
- Consumes: helper `rsiPullbackTickerGrid(t, "uwgn", "<file>.json")` из
  `internal/service/backtest/rsi_pullback_grid_test.go` (возвращает `map[string][]float64` для
  первой фазы файла); общий валидатор `TestRSIPullbackCalFilesValid` требует непустой `_comment` с
  путём файла внутри.
- Produces: одиннадцать файлов сеток, на которые ссылаются Tasks 3–10.

- [ ] **Step 1: Написать падающий сторожевой тест осей** — файл
  `internal/service/backtest/rsi_pullback_uwgn_grid_test.go`, три теста по образцу
  `rsi_pullback_vsmo_grid_test.go`:
  - `TestUWGNGridsStayWide` — проверяет НАЛИЧИЕ обязательных значений (лишние не запрещает), таблица
    `cases` ровно по таблице §5.1 спеки: `cal_screen.json` `UseDayATRGate` [0,1] и `UseVolume` [0,1];
    `cal_entry.json` `RSILower` [10,15,20,25,30,35,40,45,50], `RSIPeriod` [2,3,4,5,6,7,8,10],
    `RSIUpper` [55,60,65,70,75,80,85]; `cal_trend.json` `EMAFast` [3,5,8,10,15,20,30,40],
    `EMASlow` [50,75,100,150,200,250]; `cal_trend_low.json` `EMAFast` [3,5,8,10],
    `EMASlow` [20,30,40]; `cal_day.json` `FreshDayATR` [0,0.1,0.2,0.3,0.4,0.5],
    `SpentDayATR` [0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5]; `cal_day_spent.json`
    `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0]; `cal_volume.json`
    `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0], `VolBaseDays` [3,5,10,14,20]; `cal_vol_window.json`
    `VolLookbackBars` [1,2,3,5,8,12,16,24,32], `VolMult` [1.0,1.2,2.0]; `cal_risk.json`
    `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.3,1.5,2.0],
    `TPDailyATR` [0.2,0.3,0.4,0.5,0.6,0.8,1.0,1.5]; `cal_exit.json`
    `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90]; `cal_trail.json`
    `TrailDailyATR` [0.3,0.5,0.7,1.0,1.5], `UseRSIExit` [0,1].
  - `TestUWGNEntryGridKeepsRSIUpperAboveRSILower` — по `cal_entry.json`: для каждой пары
    (`RSILower`, `RSIUpper`) требует `RSIUpper > RSILower`.
  - `TestUWGNTrendGridsKeepFastBelowSlow` — по обоим файлам тренда: ни одна пара
    (`EMAFast`, `EMASlow`) не должна давать `EMAFast >= EMASlow`.

  В доке теста записать три отступления от канона с их замерами: **ось цели расширена вниз до 0.2**
  (максимум оси — нижний край 0.3 → 1.111/134 против дефолта 0.6 → 1.055/123, значит нужен зонд за
  краем); **ось стопа держится до 2.0 несмотря на капкан** (0.6 → 1.154/121, 1.0 → 1.232/117,
  2.0 → 1.527/115 при пуле, стоящем на месте — режет её риск-гейт A, а не сетка); **исключены
  вырожденные точки** `RSILower` 5 (0.801/5), `RSIPeriod` 12 (1.034/16) и 14 (0.705/12) — это не
  обрезка рабочей оси.

- [ ] **Step 2: Убедиться, что тест падает.**

Run: `go test ./internal/service/backtest/ -run TestUWGN`
Expected: FAIL — файлов сеток ещё нет.

- [ ] **Step 3: Создать одиннадцать файлов сеток** по таблице §5.1 спеки. Формат каждого файла:

```json
{
  "_comment": "data/params/rsi_pullback/uwgn/cal_<тема>.json — <что тема мерит>, N прогонов, поверх ДЕФОЛТОВ ЯДРА. ЗАМЕР 2026-09-04 (30 мес, окно 2024-03-05..2026-09-04, in-sample): <ось: значение -> PF/сделок ...>. ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: <какой край чем опасен>. ЗАПУСК: go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/uwgn/cal_<тема>.json -out ./reports/UWGN_<тема> -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: <заполняется после прогона>",
  "phases": [
    { "name": "<тема>", "grid": { "<Поле>": [ ... ] }, "keepTop": 5 }
  ]
}
```

  Замеры для `_comment` берутся из §4 спеки целиком, по теме:
  - `screen`: `UseDayATRGate` 0 → 0.945/383, 1 → 1.055/123 (гейт — носитель всего, что есть: без
    него дефолты убыточны); `UseVolume` 0 → 1.055/123, 1 → 0.980/108 в дефолтной форме гейта.
    `keepTop` 4, запуск с `-min-trades 1`.
  - `entry`: полный ряд `RSILower` (5 → 0.801/5, 10 → 1.444/15, 15 → 0.585/38, 20 → 0.839/67,
    25 → 1.465/98, 30 → 1.055/123, 35 → 1.166/154, 40 → 1.301/179, 45 → 1.264/200, 50 → 1.129/220)
    и `RSIPeriod` (2 → 1.242/329, 3 → 1.258/188, 4 → 1.055/123, 5 → 1.293/89, 6 → 0.903/72,
    7 → 0.761/55, 8 → 1.328/42, 10 → 1.351/26, 12 → 1.034/16, 14 → 0.705/12) с прямой записью, что
    **обе оси рваные и плато не имеют** — соседние узлы различаются в два с половиной раза при
    сопоставимом числе сделок, то есть тема будет мерить шум.
  - `trend` и `trend_low`: ряды `EMASlow` (20 → 0.923/108, 30 → 0.999/113, 50 → 0.983/121,
    75 → 1.009/125, 100 → 1.055/123, 150 → 0.993/118, 200 → 0.869/120, 250 → 0.846/121) и `EMAFast`
    (3 → 1.080/108, 5 → 1.033/114, 8 → 1.068/120, 10 → 1.055/123, 15 → 1.050/122, 20 → 1.054/126,
    30 → 1.007/127, 40 → 0.906/124) с пометкой, что весь разброс осей — 0.21 и 0.17 PF при почти
    неизменном числе сделок: тренд на UWGN не фильтрует.
  - `day` и `day_spent`: `FreshDayATR` (0 → 1.055/123, 0.1 → 1.123/143, 0.2 → 1.057/151,
    0.3 → 0.954/181, 0.4 → 0.966/236, 0.5 → 0.965/290, 0.6 → 0.959/330 — ось уходит под единицу с
    0.3, поэтому в `cal_day.json` край 0.6 не берётся) и `SpentDayATR` (0.4 → 0.947/294,
    0.5 → 0.973/239, 0.6 → 1.002/195, 0.7 → 0.984/159, 0.8 → 1.055/123, 0.9 → 1.093/100,
    1.0 → 1.186/82, 1.25 → 1.100/50, 1.5 → 1.176/33, 2.0 → 1.197/18) с предупреждением, что за 1.25
    объём выборки падает ниже `-min-trades 20` на обучающем окне.
  - `volume` и `vol_window`: полный ряд из §4.4 с прямой записью, что **ни одна из восемнадцати
    форм гейта не достаёт дефолта 1.055** (лучшая 1.042/113 при `VolMult` 1.0 / `VolBaseDays` 14) и
    PF падает монотонно с ростом множителя — тема прогоняется по правилу широких сеток, но
    отрицательный результат ожидается заранее.
  - `risk`: ряды из §4.5 с прямой записью подписи капкана — с 0.6 и выше пул сделок стоит на месте
    (121 → 115), и рост PF до 1.527 при стопе 2.0 куплен отодвинутым убытком, а не отбором входов;
    там же — потолок риск-гейта A (1.0, выживаемость 41.4%) и объяснение, почему ось цели расширена
    вниз до 0.2.
  - `exit` и `trail`: ряды из §4.6 с пометкой, что ось выхода шумная (провал 65 → 0.998 между
    60 → 1.149 и 70 → 1.055) и что трейл с 1.0 побайтово равен его отсутствию.

- [ ] **Step 4: Прогнать тесты.**

Run: `go test ./internal/service/backtest/ -run 'TestUWGN|TestRSIPullback'`
Expected: PASS

- [ ] **Step 5: Коммит.**

```bash
git add data/params/rsi_pullback/uwgn internal/service/backtest/rsi_pullback_uwgn_grid_test.go
git commit -m "feat(rsi_pullback): каталог сеток UWGN"
```

---

### Task 2: Пакет `strategy/uwgn` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/uwgn/uwgn.go`, `uwgn_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params` и `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`; хелпер
  `rsiPullbackBindingFor(ticker string, params func() core.Params)` из
  `internal/service/backtest/rsi_pullback_registry.go`.
- Produces: `uwgn.Ticker` (константа `"UWGN"`) и `uwgn.DefaultParams() core.Params` — их используют
  Tasks 12, 13 и реестр бэктеста.

- [ ] **Step 1: Написать падающие тесты** в `uwgn_test.go`:

```go
package uwgn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ПРОМЕЖУТОЧНОЕ состояние: пакет заведён под
// калибровку 2026-09-04, литерала ещё нет, и до вердикта он обязан отдавать ровно дефолты ядра.
// Тест снимается вместе с постановкой литерала (Task 12).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки UWGN обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsUWGN(t *testing.T) {
	if Ticker != "UWGN" {
		t.Fatalf("Ticker = %q, want UWGN", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/uwgn/`
Expected: FAIL — пакета ещё нет.

- [ ] **Step 3: Создать пакет** `uwgn.go` с честной шапкой в стиле соседних пакетов:

```go
// Package uwgn supplies the ticker and rsi_pullback Params for UWGN (ПАО «Объединённая вагонная
// компания», обыкновенные акции, лот 1).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. DefaultParams() отдаёт baseline ядра до вердикта.
// ...
package uwgn

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "UWGN"

// DefaultParams returns the rsi_pullback parameters for UWGN.
func DefaultParams() core.Params { return core.DefaultParams() }
```

  В шапке записать: расчётное окно 2024-03-05 … 2026-09-04 и почему оно начинается там (допэмиссия
  ноября 2023 и вынос января 2024 выброшены), схему 30/10/5 с контролем 24/12/3; оба риск-гейта с
  числами (гейт A — потолок 1.0 при выживаемости 41.4%, гейт B — 16.35%, ужесточён до baseline);
  четыре пункта стоп-условия с прямым указанием, что **дефолты ядра проваливают пункты 3 и 4**
  (0.718 и 0.929) и потому запасной маршрут TGKA недоступен; априор скринера (25-е место, PFmed
  1.38); особенности бумаги — шаг цены 0.02 ₽ при цене 20.6 ₽ (круг 0.187% против моделируемых
  0.1%, модель ОПТИМИСТИЧНА, а средняя сделка baseline даёт 0.043% капитала), ликвидность 63 млн ₽
  по медиане 12 месяцев при 52.8 млн за 2026 год, режим −65.1% за окно с одним растущим полугодием
  из пяти, дневной ATR 3.80%, просадка baseline 16.35%.

- [ ] **Step 4: Завести тикер в реестр бэктеста.** В `rsi_pullback_registry.go` добавить импорт
  `rsipullbackuwgn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/uwgn"` и строку
  `rsipullbackuwgn.Ticker: rsiPullbackBindingFor(rsipullbackuwgn.Ticker, rsipullbackuwgn.DefaultParams),`
  — **обе в алфавитном порядке среди соседей** (`uwgn` идёт после `ugld`, перед `vsmo`). В
  `rsi_pullback_registry_test.go` добавить `TestRSIPullbackUWGNTracksBaseline` по образцу соседнего
  теста VSMO: достать биндинг из `rsiPullbackRegistry[rsipullbackuwgn.Ticker]`, проверить `ok`,
  проверить, что параметры равны `rsipullbackuwgn.DefaultParams()`, и что алиас `rsipullbackuwgn`
  резолвится.

- [ ] **Step 5: Прогнать тесты.**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 6: Коммит.**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/uwgn internal/service/backtest
git commit -m "feat(rsi_pullback): пакет UWGN до калибровки"
```

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_screen.json -out ./reports/UWGN_screen \
  -months 30 -train-months 10 -test-months 5 -min-trades 1 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** проверить «Фолдов: 4» в шапке `*_walkforward.md` — иначе остановиться и доложить
  владельцу, остальные темы не запускать.
- [ ] **Step 3:** выписать pooled OOS PF, сделки пула, пофолдовые `UseDayATRGate` / `UseVolume`,
  пофолдовые OOS PF / сделки / MaxDD%. Ожидание из точечного замера: `UseDayATRGate=1` почти
  наверняка (без гейта 0.945/383, то есть убыточно), `UseVolume=0` (все восемнадцать форм гейта
  ниже дефолта).
- [ ] **Step 4:** дописать строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: …` в `_comment` файла сетки.
- [ ] **Step 5: Коммит** `feat(rsi_pullback): UWGN, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_entry.json -out ./reports/UWGN_entry \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему (504 прогона × 4 фолда).
- [ ] **Step 2:** выписать pooled OOS PF и сделки пула — это **критерий A планки** (порог 1.5 при
  ≥ 20 сделках).
- [ ] **Step 3:** выписать пофолдовые значения `RSIPeriod` / `RSILower` / `RSIUpper` — ведущая ось
  `RSILower` даёт **критерий B планки** (одинаковый выбор в ≥ 3 фолдах из 4).
- [ ] **Step 4:** отдельно записать, попал ли выбор какого-либо фолда на вырожденный по объёму узел
  (`RSILower` 10 даёт 15 сделок за 30 месяцев, `RSIPeriod` 10 — 26): такой фолд в пользу тикера не
  засчитывается.
- [ ] **Step 5:** дописать результат в `_comment`, **Коммит**
  `feat(rsi_pullback): UWGN, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая ключевая и нижний угол

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_trend.json -out ./reports/UWGN_trend \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_trend_low.json -out ./reports/UWGN_trend_low \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** по канонической `trend` выписать pooled OOS PF и сделки (**критерий A планки**) и
  пофолдовые `EMASlow` (**критерий B**).
- [ ] **Step 3:** по `trend_low` выписать то же самое; поле тренда идёт из `trend_low` только при
  большинстве ≥ 3/4 **и** превосходстве pooled OOS над канонической темой.
- [ ] **Step 4:** записать прямым текстом, разошлись ли фолды по `EMASlow`: точечный замер даёт
  разброс всей оси 0.21 PF при почти неизменном числе сделок, то есть обучающее окно будет выбирать
  узел по шуму (прецедент VSMO, где ось дала четыре разных значения в четырёх фолдах).
- [ ] **Step 5:** дописать результаты, **Коммит** `feat(rsi_pullback): UWGN, темы тренда`.

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_day.json -out ./reports/UWGN_day \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_day_spent.json -out ./reports/UWGN_day_spent \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** выписать пофолдовые `FreshDayATR` и `SpentDayATR` из обеих тем; при расхождении
  тем по `SpentDayATR` записать оба ряда голосов и принять поле по правилу большинства внутри той
  темы, которая его меряет одну (`day_spent`).
- [ ] **Step 3:** если большинство выбрало `SpentDayATR` ≥ 1.25, записать цену решения по объёму:
  точечно 1.25 → 50 сделок за 30 месяцев, 1.5 → 33, 2.0 → 18, то есть 6–17 сделок на обучающее
  окно при пороге `-min-trades 20`.
- [ ] **Step 4:** дописать результаты, **Коммит** `feat(rsi_pullback): UWGN, темы дневного гейта`.

---

### Task 7: Тема `volume`

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_volume.json -out ./reports/UWGN_volume \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему (30 прогонов × 4 фолда).
- [ ] **Step 2:** выписать пофолдовые `VolMult` / `VolBaseDays` и pooled OOS PF.
- [ ] **Step 3:** сверить с точечным замером: все восемнадцать форм гейта ниже дефолта 1.055,
  лучшая — 1.042/113. Если тема всё же голосует за гейт большинством 3/4, проверить точечным
  прогоном, не эквивалентен ли выбор простому `UseVolume 0`, и записать результат прямым текстом.
- [ ] **Step 4:** записать, какой из четырёх исходов каталожной объёмной гипотезы дал UWGN
  (NKHP — гейт нужен 4/4; TGKA — не выбран; VSMO — ничья по периодам; UWGN — ожидается монотонный
  вред).
- [ ] **Step 5:** дописать результат, **Коммит** `feat(rsi_pullback): UWGN, тема volume`.

---

### Task 8: Тема `vol_window`

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_vol_window.json -out ./reports/UWGN_vol_window \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему (27 прогонов × 4 фолда).
- [ ] **Step 2:** выписать пофолдовые `VolLookbackBars` и pooled OOS PF.
- [ ] **Step 3:** записать прямым текстом: если точка держит `UseVolume 0` (ожидаемый исход по
  Task 7), поле `VolLookbackBars` в ней **инертно**, и его голос принимается только для протокола.
- [ ] **Step 4:** дописать результат, **Коммит** `feat(rsi_pullback): UWGN, тема vol_window`.

---

### Task 9: Тема `risk` — стоп, цель и обязательная проверка риск-гейта A

**Files:** также `data/params/rsi_pullback/uwgn/plateau_stop_10.json`, `plateau_stop_20.json`

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_risk.json -out ./reports/UWGN_risk \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему (80 прогонов × 4 фолда).
- [ ] **Step 2:** применить риск-гейт A: если большинство фолдов выбрало `StopDailyATR` > 1.0,
  победитель темы **отвергается гейтом**, в точку идёт 1.0 (41.4% будних дней), цена решения в PF
  пишется прямым текстом. На UWGN это ожидаемый исход: максимум оси стоит на 2.0 (выживаемость
  8.7%).
- [ ] **Step 3:** снять два зонда капкана — файлы `plateau_stop_10.json` и `plateau_stop_20.json`
  (одноточечные сетки: дефолты ядра плюс поля, принятые темами, и `StopDailyATR` 1.0 и 2.0
  соответственно) на полном окне:

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/plateau_stop_10.json -out ./reports/UWGN_nb_stop_10 \
  -months 30 -min-trades 1
```

  Выписать PF, сделки, **max DD**, выходы RSI/SL/TP с долей SL, удержание (медиана, p90, максимум),
  долю ночёвок, долю выходов в выходную сессию.
- [ ] **Step 4:** записать, воспроизводится ли на UWGN подпись капкана «PF вверх при неизменном
  пуле»: точечно 0.6 → 1.154/121, 0.7 → 1.230/119, 1.0 → 1.232/117, 1.3 → 1.379/117,
  2.0 → 1.527/115 против 0.5 → 1.055/123. Проверить, растёт ли вместе с PF просадка — если да, это
  прямой материал риск-гейта B; если нет, единственным работающим фильтром остаётся гейт A, и это
  пишется прямым текстом.
- [ ] **Step 5:** по цели записать голоса и проверить край: точка на 0.2 или 0.3 означает, что
  выбран нижний край оси; сосед 0.2 в сетке уже стоит зондом за краем — сравнить 0.2 / 0.3 / 0.4 и
  сказать прямо, плато это или обрезанная ось.
- [ ] **Step 6:** дописать результат, **Коммит**
  `feat(rsi_pullback): UWGN, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit` и `trail`

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_exit.json -out ./reports/UWGN_exit \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/cal_trail.json -out ./reports/UWGN_trail \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** проверить, не обходит ли трейл риск-гейт A (урок AFKS): дистанция трейла ниже
  принятого стопа делает её эффективной защитой, и гейт применяется к ней.
- [ ] **Step 3:** применить правило упрощения: дистанция ≥ 1.0 → `UseTrail 0`, `TrailDailyATR 0`
  (на UWGN точечно трейл 1.0 и 1.5 побайтово равны его отсутствию).
- [ ] **Step 4:** если тема выбрала `UseRSIExit 0` — записать отдельной строкой: точечно это
  0.989/112 против 1.055/123, то есть выход по RSI обязателен.
- [ ] **Step 5:** дописать результаты, **Коммит** `feat(rsi_pullback): UWGN, темы exit и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и четыре пункта стоп-условия

**Files:** `data/params/rsi_pullback/uwgn/plateau_point.json`, соседи плато
`plateau_<поле>_<значение>.json`, `docs/superpowers/plans/task-11-report-uwgn.md`

- [ ] **Step 1:** собрать точку по правилу большинства (≥ 3 из 4; иначе дефолт ядра). Для каждого
  из восемнадцати полей записать голоса фолдов, решение и случайность совпадения с дефолтом.
- [ ] **Step 2:** применить риск-гейт A к собранной точке (эффективная защита
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, потолок 1.0).
- [ ] **Step 3:** прогнать точку схемой 30/10/5:

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/plateau_point.json -out ./reports/UWGN_point_oos \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor
```

  Выписать pooled OOS PF, сделки пула, пофолдовые in-sample → OOS PF / сделки / NetPnL% / MaxDD%.
  Сравнение с дефолтами: 1.047 на 85 сделках при одном прибыльном фолде из четырёх.
- [ ] **Step 4:** проверить пункты 1 и 2 стоп-условия (PF ≥ 1.0, ≥ 20 сделок в пуле). Провал →
  **остановиться**, задачи 12–16 не выполнять, числа владельцу.
- [ ] **Step 5:** прогнать контрольную схему 24/12/3 (пункт 3):

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/plateau_point.json -out ./reports/UWGN_point_24 \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

  Дефолты на том же куске дают **0.718** — то есть пункт 3 они проваливают, и точка обязана его
  взять сама. Провал → **остановиться**.
- [ ] **Step 6:** проверить пункт 4 стоп-условия — тот же прогон схемой 30/10/5 с удвоенным кругом:

```bash
go run ./cmd/backtest -ticker UWGN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/uwgn/plateau_point.json -out ./reports/UWGN_point_cost2 \
  -months 30 -train-months 10 -test-months 5 -min-trades 20 -metric profit_factor -commission 0.001
```

  pooled OOS PF < 1.0 → **остановиться**. Дефолты здесь дают 0.929 на полном окне in-sample.
  Дополнительно снять строку `-commission 0.0015` как справочную (тройной круг, не пункт
  стоп-условия) — она показывает, сколько запаса осталось при дальнейшем падении цены.
- [ ] **Step 7:** применить риск-гейт B на полном окне (одиночный прогон точки на 30 месяцах,
  `-out ./reports/UWGN_point_full`), потолок max DD **16.35%**. Нарушение → сдвинуть стоп (или
  трейл) к ближайшему значению оси, возвращающему точку под потолок, и записать цену решения в PF
  прямым текстом.
- [ ] **Step 8:** снять анатомию и полугодия точки; сравнить с baseline по пяти числам третьего
  контура (доля SL 30.1%, удержание 11/20/46, ночёвки 42.3%, выходная сессия 5.7%, DD 16.35%).
  Отдельно записать, держится ли точка на одном полугодии, как baseline (1.896 в сентябре
  2024 — марте 2025 против убытков в трёх других).
- [ ] **Step 9:** снять соседей плато по каждому полю, принятому темой большинством (±один узел
  оси) — по одному файлу `plateau_<поле>_<значение>.json` на соседа; совпадение результата
  побайтово = инертная ось, и это пишется прямым текстом. Плато шириной меньше 0.05 PF читается как
  отсутствие сигнала.
- [ ] **Step 10:** написать рабочую записку `docs/superpowers/plans/task-11-report-uwgn.md`: голоса
  по каждому полю, оба walk-forward, четыре пункта стоп-условия с числами, оба риск-гейта, соседи
  плато, полугодия, анатомия против baseline, вердикт по планке.
- [ ] **Step 11: Коммит** `feat(rsi_pullback): UWGN, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Задача выполняется только если ни один пункт стоп-условия не сработал.**

**Files:** `internal/service/trading_strategy/rsi_pullback/strategy/uwgn/uwgn.go`, `uwgn_test.go`,
`internal/service/backtest/rsi_pullback_registry_test.go`

- [ ] **Step 1:** заменить `TestParamsTrackTheBaselineUntilCalibrated` на
  `TestParamsAreTheAcceptedPoint` (снимок всех восемнадцати полей принятой точки, по образцу
  `vsmo_test.go`) плюс `TestPointDiffersFromTheCoreBaseline` (сторожит именно те поля, которыми
  точка отличается от дефолтов ядра, с их голосами в доке теста).
- [ ] **Step 2:** убедиться, что тесты падают.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/uwgn/`
Expected: FAIL — литерала ещё нет.

- [ ] **Step 3:** поставить литерал принятой точки в `DefaultParams()` (полный `core.Params{...}`,
  все восемнадцать полей явно). Значения берутся из Task 11 Step 1 после применения обоих
  риск-гейтов.
- [ ] **Step 4:** заменить тест реестра `TestRSIPullbackUWGNTracksBaseline` на
  `TestRSIPullbackUWGNServesTheCalibratedPoint`.
- [ ] **Step 5:** тесты и линт.

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... && ./bin/golangci-lint run ./internal/...`
Expected: PASS

- [ ] **Step 6: Коммит** `feat(rsi_pullback): UWGN откалиброван — литерал вместо отслеживания baseline`.

---

### Task 13: Реестр живого раннера

**Files:** `internal/service/trading_strategy/rsi_pullback/live/registry.go`, `registry_test.go`

- [ ] **Step 1:** добавить импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/uwgn"`
  и строку `uwgn.Ticker: uwgn.DefaultParams(),` в карту `paramsByTicker` (алфавитный порядок).
- [ ] **Step 2:** дописать абзац комментария карты в том же англоязычном стиле, что у соседей:
  вердикт по планке, оба риск-гейта и их цена, четвёртый пункт стоп-условия по издержкам и принятые
  риски (круг 0.187% против моделируемых 0.1% при средней сделке baseline 0.043% капитала;
  ликвидность 63 млн ₽ по медиане 12 месяцев при 52.8 млн за 2026 год и гейте вселенной 50 млн;
  выходная сессия 8.12 млн ₽; режим −65.1% за окно; окно 30 месяцев после допэмиссии и
  несравнимость чисел с 36-месячным каталогом).
- [ ] **Step 3:** прогнать тесты.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/...`
Expected: PASS

- [ ] **Step 4: Коммит** `feat(rsi_pullback): UWGN в реестре живого раннера`.

---

### Task 14: Боевая вселенная

**Files:** `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`,
`env/prod.env`, `env/prod.env.example`, `env/local.env.example`, `docs/rsi_pullback/live.md`

- [ ] **Step 1:** перечитать числа Task 11 Steps 4–7 — ни один из четырёх пунктов стоп-условия не
  сработал, оба риск-гейта пройдены.
- [ ] **Step 2:** добавить `"UWGN"` в `want` теста конфига двадцать пятым.
- [ ] **Step 3:** убедиться, что тест падает.

Run: `go test ./internal/config/ -run RSIPullback`
Expected: FAIL

- [ ] **Step 4:** дописать `"UWGN"` в `Tickers` + комментарий-абзац в стиле соседних; дописать
  `,UWGN` в три env-файла (`env/prod.env`, `env/prod.env.example`, `env/local.env.example`).
- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  `docs/rsi_pullback/live.md` (только значение — пер-тикерного ничего).
- [ ] **Step 6:** прогнать тесты.

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 7: Коммит** `feat(rsi_pullback): завести UWGN в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:** `internal/service/trading_strategy/rsi_pullback/strategy/uwgn/uwgn.go`

- [ ] **Step 1:** переписать шапку пакета в итог калибровки: вердикт по планке по каждому критерию
  обеих ключевых тем, точка поле за полем с голосами фолдов, оба walk-forward, строки издержек
  (0.1% / 0.2% / 0.15%), соседи плато с пометкой инертных осей, пять полугодий, анатомия против
  baseline, результат обоих риск-гейтов и их цена. Отдельными абзацами — принятые риски и условия
  пересмотра:
  1) **издержки**: круг 0.187% против моделируемых 0.1% (шаг 0.02 ₽ при цене 20.6 ₽) при средней
     сделке baseline 0.043% капитала, как это закрыто четвёртым пунктом стоп-условия; **условие
     пересмотра: падение цены ниже 15 ₽ поднимает круг выше 0.27% — перекалибровать вне планового
     цикла или вывести из вселенной**;
  2) **ликвидность**: медиана 63.0 млн ₽ за 12 месяцев при гейте вселенной 50 млн и 52.8 млн за
     2026 год, падение 680 → 402 → 118 → 52.8 млн по годам; **условие пересмотра: медиана ниже
     30 млн ₽ — вывести тикер из боевой вселенной, не дожидаясь планового цикла**;
  3) **просадка**: baseline 16.35% — втрое выше каталожной нормы; как её удержал ужесточённый
     гейт B;
  4) **режим −65.1% за окно** при одном растущем полугодии из пяти и baseline, сделанном одним
     полугодием; **при первом устойчиво растущем полугодии калибровку повторить вне планового
     цикла**;
  5) **короткая история нынешнего инструмента**: окно начинается через четыре месяца после
     допэмиссии, числа сравнимы построчно только с DIAS (30/10/5);
  6) **капкан широкого стопа** с пулом, стоящим на месте с 0.6 ATR, и роль обоих риск-гейтов;
  7) **выходная сессия** с медианным оборотом 8.12 млн ₽ и долей выходов точки.
- [ ] **Step 2:** прогнать тесты.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 3:** механический вывод в `docs/rsi_pullback/strategy.md` — **только если** UWGN дал
  вывод о механике. Кандидаты названы заранее: **объёмный гейт, вредящий монотонно** (четвёртый
  исход каталожной объёмной гипотезы) и **модель издержек как отдельный гейт отбора тикеров** —
  шаг цены против средней сделки, обобщение прецедентов KMAZ и VSMO. Пер-тикерных чисел, дат и
  вердиктов в `docs/rsi_pullback/` не писать.
- [ ] **Step 4: Коммит** `docs(rsi_pullback): разбор калибровки UWGN и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** сверка живой сборки с бэктестом.

Run: `go run ./cmd/pullparity -tickers UWGN -months 24`
Expected: ноль расхождений. (24 месяца, а не 30: при большем окне сам инструмент предупреждает,
что расхождения `Daily*` на ранних барах ожидаемы — окно `dailyFetchDays` живого сборщика короче
истории эталона. Урок VSMO.)

- [ ] **Step 2:** полный гейт.

Run: `./bin/mage ci`
Expected: зелёный

- [ ] **Step 3:** проверить, что отчёты не попали в индекс.

Run: `git status --porcelain | grep -c '^.. reports/'`
Expected: `0`

- [ ] **Step 4:** финальный коммит, если остались правки.

---

## Self-Review

**Покрытие спеки:** окно и схема 30/10/5 (§2) → Global Constraints, Task 3 Step 2 (сверка фолдов);
baseline и его числа (§3) → Global Constraints, используются в Tasks 9 и 11; свойства бумаги (§4)
→ `_comment` файлов в Task 1 Step 3; одиннадцать тем (§5.1) → Tasks 3–10; ширина сеток и два
отступления (§5.1) → Task 1 со сторожевыми тестами; правило сборки точки (§5.2) → Task 11 Step 1;
соседи плато (§5.3) → Task 9 Step 5 и Task 11 Step 9; риск-гейт A (§5.4) → Task 9 Step 2, Task 10
Step 2, Task 11 Step 2; риск-гейт B (§5.5) → Task 11 Step 7; третий контур (§5.6) → Task 11 Step 8;
планка (§5.7) → Tasks 4 и 5; четыре пункта стоп-условия (§5.8) → Task 11 Steps 4–6; правило прода
(§5.9) → Task 14 Step 1; артефакты (§6) → Tasks 1, 2, 11–15; риски и условия пересмотра (§7)
→ Task 15 Step 1; починенный кэш и запрет `-refresh` → Global Constraints; `pullparity` и `mage ci`
→ Task 16.

**Плейсхолдеры:** один — литерал в Task 12 Step 3 (точка известна только после Task 11). Указано,
откуда берётся значение и по какому правилу.

**Согласованность имён:** `uwgn.Ticker` и `uwgn.DefaultParams()` заводятся в Task 2 и используются
в Tasks 12–14 под теми же именами; тест `TestParamsTrackTheBaselineUntilCalibrated` (Task 2)
заменяется на `TestParamsAreTheAcceptedPoint` в Task 12; `TestRSIPullbackUWGNTracksBaseline`
(Task 2) — на `TestRSIPullbackUWGNServesTheCalibratedPoint` (Task 12); сторожевые тесты сеток
`TestUWGNGridsStayWide`, `TestUWGNEntryGridKeepsRSIUpperAboveRSILower`,
`TestUWGNTrendGridsKeepFastBelowSlow` заводятся в Task 1 и больше не переименовываются.
