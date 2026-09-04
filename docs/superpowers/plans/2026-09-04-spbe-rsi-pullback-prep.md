# SPBE под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести SPBE (ПАО «СПБ Биржа») до вердикта по стратегии `rsi_pullback`: каталог
максимально широких сеток, канонический тематический walk-forward, принятая точка, прошедшая два
риск-гейта и четыре пункта стоп-условия, литерал в пакете и заведение в боевую вселенную двадцать
пятым тикером.

**Architecture:** Процедура **каноническая**: все одиннадцать тем идут поверх дефолтов ядра — на
SPBE ось `RSIPeriod` унимодальна с единственным максимумом ровно на дефолте (2 → 0.942/394,
3 → 1.128/223, **4 → 1.296/148**, 5 → 1.094/113, спад монотонен до края), поэтому якорь (BSPB, SOFL)
не нужен и числа тем сравнимы с каталогом построчно. Схема прогонов **36/12/6** (четыре фолда,
проверено контрольным прогоном до написания спеки), плюс **обязательный контрольный прогон принятой
точки по схеме 24/12/3** — внутри окна сессия расширилась 28 → 34 получасовых бара в буднем дне. Два
риск-гейта: гейт выживаемости применяется к эффективной защите `min(StopDailyATR, TrailDailyATR при
UseTrail=1)`, гейт просадки — max DD точки ≤ max DD baseline **без множителя 1.3**. **Особенность
SPBE:** просадка baseline худшая в каталоге (24.83%), а широкий стоп поднимает PF и одновременно
СНИЖАЕТ просадку, то есть гейт B его не ловит — единственный фильтр капкана здесь гейт A.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-04-spbe-rsi-pullback-prep-design.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех без исключения прогонах. Флаг `-interval Minutes30` обязателен в
  каждой команде: дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Схема прогонов каноническая:** `-months 36 -train-months 12 -test-months 6 -min-trades 20
  -metric profit_factor`, четыре фолда встык. У темы `screen` — `-min-trades 1`.
- **Расчётное окно 2023-09-04 … 2026-09-04, 36.0 месяца.** Обрезки нет: у SPBE нет структурного
  разрыва (число акций не менялось), обвал осени 2023 и мания весны 2025 — режимы одной бумаги.
- **Число фолдов сверяется на первой же теме.** Контрольные прогоны 2026-09-04 уже дали «Фолдов: 4»
  и на 36/12/6 (pooled OOS 1.176), и на 24/12/3 (1.156), но сверка повторяется: если в шапке отчёта
  темы `screen` окажется «Фолдов: 3» — **остановиться и доложить владельцу**, остальные темы не
  запускать (ловушка ASTR).
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш дотянут `-refresh` 2026-09-04 до написания
  спеки: `SPBE_Minutes30.json` — **29 105 баров** (2023-09-04 … 2026-09-04), `SPBE_Day1.json` —
  **1 184 свечи** (2022-09-05 … 2026-09-03).
- **Дыр в серии нет.** Разрыв длиннее четырёх дней ровно один — новогодние каникулы
  2023-12-29 → 2024-01-03 (4.4 дня), календарный.
- **ПРОЦЕДУРА КАНОНИЧЕСКАЯ:** все темы поверх дефолтов ядра, якоря нет.
- **Сетки держатся максимально широкими.** Обрезок осей не делается; замеры, которые в узком
  каталоге были бы основанием вырезать край, идут в `_comment` как **предупреждения**. Жёсткие
  инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower` на всех парах, ось тренда не
  порождает пар `EMAFast ≥ EMASlow`, `StopDailyATR` нигде не равен нулю, ось `TPDailyATR` содержит
  0.2, ось `EMASlow` темы `trend_low` содержит 15.
- **Из сеток исключены только вырожденные точки:** `RSILower` 5 (7 сделок за 36 месяцев при
  PF 3.253), `RSIPeriod` 12 (0.841/33) и 14 (0.434/18). Каждое исключение — с замером в `_comment`.
- **Правило сборки точки:** поле берётся из темы, которая его меряет, и принимается только если за
  значение высказались **не менее трёх фолдов из четырёх**; иначе поле остаётся на **дефолте ядра**.
  Ничья 2/2 большинством не считается. Поле тренда идёт из `trend_low` только при большинстве ≥ 3/4
  **и** превосходстве pooled OOS над канонической `trend`. Голоса `RSIUpper` из темы `entry` в
  сборку не идут — поле меряет `exit`.
- **РИСК-ГЕЙТ A — выживаемость эффективной защиты.** Принимается только защита, достижимая **не
  менее чем в 30% будних дней окна**, гейт применяется к `min(StopDailyATR, TrailDailyATR при
  UseTrail=1)` (урок AFKS). Таблица выживаемости SPBE (n = 772): 0.3 — 98.7%, 0.4 — 95.7%,
  0.5 — 90.5%, 0.6 — 82.9%, 0.7 — 73.2%, 0.8 — 64.0%, **1.0 — 44.7%**, 1.3 — 27.3%, 1.5 — 20.3%,
  2.0 — 9.6%. **Потолок — 1.0.** Если тема голосует выше, в точку идёт ближайшее значение оси внутри
  гейта, цена решения в PF пишется прямым текстом. Порог 30% после начала прогонов не двигается.
- **РИСК-ГЕЙТ B — просадка точки.** Max DD принятой точки на полном окне (одиночный прогон
  `-params`/одноточечная сетка) **не превышает max DD baseline: 24.83%**. Множитель 1.3 здесь **не
  применяется** (решение владельца 2026-09-04).
- **ПРАВИЛО РАЗРЕШЕНИЯ НИЧЬИХ В ПОЛЬЗУ РИСКА.** Среди соседей плато, различающихся по pooled OOS PF
  менее чем на 0.05, в точку идёт вариант с **меньшей просадкой** на полном окне. Правило объявлено
  до прогонов и применяется механически. Материал для него есть: трейл 0.3 → DD 24 034 ₽ и 0.5 →
  32 154 ₽ против 42 902 ₽ у дефолта при PF 1.262 и 1.314 против 1.296.
- **Третий контур — риск-профиль точки против baseline.** Опорные числа baseline: доля SL-выходов
  **27.0%**, удержание медиана **9** / p90 **22** / максимум **37** баров, ночёвок **41.2%**,
  выходов в выходную сессию **0.7%**, max DD **24.83%**. Точка меряется по всем пяти. **Падение доли
  SL-выходов вместе с ростом удержания и ночёвок — подпись капкана**, и тогда точка берёт более
  узкий стоп.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` (каноническая, не
  `trend_low`) обе дают pooled OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для
  `entry`, `EMASlow` для `trend`) выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд в пользу
  тикера не засчитывается.
- **Стоп-условие из ЧЕТЫРЁХ пунктов.** Работа останавливается, если принятая точка даёт: (1) pooled
  OOS PF < 1.0 на 36/12/6, **либо** (2) меньше 20 сделок в пуле OOS, **либо** (3) pooled OOS PF < 1.0
  на контрольном прогоне 24/12/3, **либо** (4) pooled OOS PF < 1.0 на 36/12/6 при
  `-commission 0.001`. При срабатывании — числа владельцу, **задачи 12–16 не выполняются**, маршрут
  выбирает владелец. На SPBE доступен запасной маршрут «дефолты ядра» (прецедент TGKA): они сами
  проходят все четыре пункта.
- **Издержки.** Шаг цены **0.1 ₽** во всех ценовых полосах; круг из двух шагов **0.088%** при
  медианной цене последних 12 месяцев (227.80 ₽) и **0.140%** при текущей (142.50 ₽) против 0.1% в
  модели. Чувствительность baseline: 0.1% → **1.296**, 0.2% → **1.177**, 0.3% → **1.064**,
  0.4% → **0.960**.
- **Правило прода:** литерал ставится и SPBE заводится в `RSI_PULLBACK_TICKERS` **двадцать пятым**,
  если не сработал ни один пункт стоп-условия — независимо от того, взята планка или нет.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают все темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`, `UseDayATRGate 1`,
  `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`, `UseVolume 0`,
  `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`, `UseTrail 0`,
  `TrailDailyATR 0`. Контрольный прогон дефолтов на окне: **148 сделок, PF 1.296, net +48 508.20 ₽
  (+48.51%)**, win rate 66.89%, max DD **42 901.52 ₽ (24.83%)**, exposure 5.27%, expectancy
  **317.69 ₽ = 0.318% капитала**, выходы RSI 87 (58.8%) / SL 40 (27.0%) / TP 21 (14.2%), удержание
  9 / 22 / 37 баров, ночёвок 41.2%, выходов в выходную сессию 0.7%. Walk-forward дефолтов:
  36/12/6 → **1.176/87** (фолды in-sample → OOS: 1.574 → 1.468/35 при DD 16.24%; 1.514 → 0.726/14
  при 14.95%; 1.164 → 1.097/20 при 6.45%; 1.045 → 1.228/18 при 4.24%); 24/12/3 → **1.156**,
  compounded +4.96%.
- **Полугодия baseline** (нужны в Task 11): 2023-09..2024-03 — 17 сделок, PF **2.065**, +10 249 ₽;
  2024-03..2024-09 — 44, 1.458, +18 700; 2024-09..2025-03 — 35, 1.468, +18 516;
  2025-03..2025-09 — 14, **0.726, −7 057**; 2025-09..2026-03 — 20, 1.097, +2 330;
  2026-03..2026-09 — 18, 1.228, +4 279. **Прибыльны пять из шести**, концентрации нет.
- **Ликвидность.** Медиана оборота будних дней: 148.4 млн ₽ (36 мес), 213.5 (24 мес), **164.9
  (12 мес)**; по годам 63.0 → 95.8 → **437.1 (2025)** → 124.2 (2026). Гейт вселенной скринера 50 млн
  пройден втрое. Выходная сессия: 159 дней, медиана **27.85 млн ₽** — самая живая в каталоге.
  Дневной ATR(14) медиана **4.78%** (p10 2.50%, p90 8.38%).
- **Априор скринера** (`pullback_screen` 2026-08-04): 26-е место, PFmed 1.37, **holdout PFmed 0.90
  на 6 сделках**, Plateau 42%, зажатых 0/24, молчащих 0/24, оборот 318 млн ₽, ATR 6.08%, лучшая
  конфигурация — RSI 6/10, EMA 20/150, TP 1.0. Априор **ниже прямого замера** (как на ELFV).
- **Режим:** buy&hold за окно **−24.0%**, по полугодиям −52.2%, +18.7%, **+181.5%**, −11.0%, −7.1%,
  −42.4%; максимальная просадка инструмента внутри окна **70.4%**.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; полную команду запуска с путём
  `data/params/rsi_pullback/spbe/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); место под
  строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md). Пер-тикерный
  разбор живёт в доке пакета `strategy/spbe` и в `_comment` файлов сеток.
- **Каталог `reports/` — в `.gitignore`.**
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/spbe-pullback-prep` от `feat/uwgn-pullback-prep` (`96fe6f9`), спека уже
  закоммичена (`4f1c8bf`).

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/spbe/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_spbe_grid_test.go`

**Interfaces:**
- Consumes: helper `rsiPullbackTickerGrid(t, "spbe", "<file>.json")` из
  `internal/service/backtest/rsi_pullback_grid_test.go` (возвращает `map[string][]float64` для
  первой фазы файла); общий валидатор `TestRSIPullbackCalFilesValid` требует непустой `_comment` с
  путём файла внутри.
- Produces: одиннадцать файлов сеток, на которые ссылаются Tasks 3–10.

- [ ] **Step 1: Написать падающий сторожевой тест осей** — файл
  `internal/service/backtest/rsi_pullback_spbe_grid_test.go`, три теста по образцу
  `rsi_pullback_uwgn_grid_test.go`:
  - `TestSPBEGridsStayWide` — проверяет НАЛИЧИЕ обязательных значений (лишние не запрещает), таблица
    `cases` ровно по таблице §5.1 спеки: `cal_screen.json` `UseDayATRGate` [0,1] и `UseVolume` [0,1];
    `cal_entry.json` `RSIPeriod` [2,3,4,5,6,7,8,10], `RSILower` [10,15,20,25,30,35,40,45,50],
    `RSIUpper` [55,60,65,70,75,80,85]; `cal_trend.json` `EMAFast` [3,5,8,10,15,20,30,40],
    `EMASlow` [50,75,100,150,200,250]; `cal_trend_low.json` `EMAFast` [3,5,8,10],
    `EMASlow` [15,20,30,40]; `cal_day.json` `FreshDayATR` [0,0.1,0.2,0.3,0.4,0.5],
    `SpentDayATR` [0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5]; `cal_day_spent.json`
    `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0]; `cal_volume.json`
    `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0], `VolBaseDays` [3,5,10,14,20]; `cal_vol_window.json`
    `VolLookbackBars` [1,2,3,5,8,12,16,24,32], `VolMult` [1.0,1.2,2.0]; `cal_risk.json`
    `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.3,1.5,2.0],
    `TPDailyATR` [0.2,0.3,0.4,0.5,0.6,0.8,1.0,1.5]; `cal_exit.json`
    `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90]; `cal_trail.json`
    `TrailDailyATR` [0.3,0.5,0.7,1.0,1.5], `UseRSIExit` [0,1].
  - `TestSPBEEntryGridKeepsRSIUpperAboveRSILower` — по `cal_entry.json`: для каждой пары
    (`RSILower`, `RSIUpper`) требует `RSIUpper > RSILower`.
  - `TestSPBETrendGridsKeepFastBelowSlow` — по обоим файлам тренда: ни одна пара
    (`EMAFast`, `EMASlow`) не должна давать `EMAFast >= EMASlow`.

  В доке теста записать три отступления от канона с их замерами: **`cal_trend_low.json` расширен
  вниз до `EMASlow` 15** (канон [20,30,40]) — точечно 20 → 1.461/122 и 30 → 1.496/128, то есть
  максимум оси `EMASlow` стоит в нижнем углу, и без узла 15 он совпал бы с краем сетки;
  **ось `TPDailyATR` расширена вниз до 0.2** (канон от 0.3) — точечно максимум 0.3 → 1.492/168 стоит
  рядом с нижним краем, узел 0.2 (1.311/176) служит зондом за краем; **ось `StopDailyATR` оставлена
  широкой до 2.0**, хотя 1.3 → 1.752/140, 1.5 → 1.969/140 и 2.0 → 1.930/139 — это капкан
  (пул сделок не меняется), и режет его риск-гейт A, а не граница сетки.

- [ ] **Step 2: Убедиться, что тест падает.**

Run: `go test ./internal/service/backtest/ -run TestSPBE`
Expected: FAIL — файлов сеток ещё нет.

- [ ] **Step 3: Создать одиннадцать файлов сеток** по таблице §5.1 спеки. Формат каждого файла:

```json
{
  "_comment": "data/params/rsi_pullback/spbe/cal_<тема>.json — <что тема мерит>, N прогонов, поверх ДЕФОЛТОВ ЯДРА. ЗАМЕР 2026-09-04 (36 мес, in-sample): <ось: значение -> PF/сделок ...>. ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: <какой край чем опасен>. ЗАПУСК: go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/spbe/cal_<тема>.json -out ./reports/SPBE_<тема> -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: <заполняется после прогона>",
  "phases": [
    { "name": "<тема>", "grid": { "<Поле>": [ ... ] }, "keepTop": 5 }
  ]
}
```

  Замеры для `_comment` берутся из §4 спеки целиком, по теме:
  - `screen`: `UseDayATRGate` 0 → **0.889/437**, 1 → 1.296/148 (гейт покупает 0.41 PF — сильнейший
    одиночный вклад каталога); `UseVolume` 0 → 1.296/148, 1 → 1.382/120 при дефолтной форме гейта.
    `keepTop` 4, запуск с `-min-trades 1`.
  - `entry`: полный ряд `RSIPeriod` (2 → 0.942/394, 3 → 1.128/223, 4 → 1.296/148 — МАКСИМУМ И
    ДЕФОЛТ, 5 → 1.094/113, 6 → 1.118/93, 7 → 1.072/83, 8 → 1.030/72, 10 → 0.872/49) с пометкой, что
    ось унимодальна и что 12 (0.841/33) и 14 (0.434/18) исключены как вырожденные и убыточные; ряд
    `RSILower` (5 → 3.253/**7** — исключено, 10 → 1.485/39, 15 → 1.156/74, 20 → 1.185/95,
    25 → 1.281/125, 30 → 1.296/148 — ДЕФОЛТ, 35 → 1.227/175, 40 → 1.134/212, 45 → 1.073/243,
    50 → 1.093/276) с предупреждением, что победа края 10 означала бы ~13 сделок на обучающее окно.
  - `trend` и `trend_low`: ряды `EMASlow` (20 → 1.461/122, 30 → 1.496/128 — МАКСИМУМ, 50 → 1.403/139,
    75 → 1.302/147, 100 → 1.296/148 — ДЕФОЛТ, 150 → 1.142/152, 200 → 1.234/154, 250 → 1.271/155) и
    `EMAFast` (3 → 1.358/130 — максимум, 5 → 1.307/141, 8 → 1.278/147, 10 → 1.296/148 — ДЕФОЛТ,
    15 → 1.259/150, 20 → 1.202/156, 30 → 1.248/159, 40 → 1.195/161) с прямой записью, что максимум
    медленной EMA стоит в нижнем углу и `trend_low` — претендент на победу, а не формальность.
  - `day` и `day_spent`: ряд `FreshDayATR` (0 → 1.296/148 — ДЕФОЛТ, 0.1 → 1.301/149 — максимум с
    отличием 0.005, 0.2 → 1.190/162, 0.3 → 1.056/208, 0.4 → 1.010/265, 0.5 → 0.961/315,
    0.6 → 0.892/372) и ряд `SpentDayATR` (0.4 → 0.967/349, 0.5 → 1.013/300, 0.6 → 1.119/242,
    0.7 → 1.225/191, 0.8 → 1.296/148 — ДЕФОЛТ, 0.9 → 1.333/118, 1.0 → 1.273/90, 1.25 → 1.326/54,
    1.5 → 1.341/34 — максимум на вырожденной выборке, 2.0 → 1.159/17) с предупреждением, что за 1.25
    объём выборки уходит ниже порога `-min-trades 20` на обучающем окне.
  - `volume` и `vol_window`: ряд из §4.4 (лучшая форма 1.0/10 → **1.395/124**, далее 1.2/14 →
    1.382/120 при DD 31 609 ₽ против 42 902 у дефолта, 1.0/14 → 1.365/126, 3.0/10 → 1.361/73,
    1.5/10 → 1.356/105, худшая 3.0/20 → 1.014/70) с прямой записью ожидания: **гейт может выиграть**
    — семь форм из восемнадцати обходят дефолт, и это второй такой случай в каталоге после NKHP, но
    здесь на самой ЛИКВИДНОЙ бумаге, а не на самой тонкой.
  - `risk`: ряды `StopDailyATR` (0.3 → 1.306/154, 0.4 → 1.238/152, 0.5 → 1.296/148 — ДЕФОЛТ,
    0.6 → 1.258/145, 0.7 → 1.173/144, 0.8 → 1.209/141, 1.0 → 1.292/140, 1.3 → 1.752/140,
    1.5 → 1.969/140, 2.0 → 1.930/139) и `TPDailyATR` (0.2 → 1.311/176, 0.3 → 1.492/168 — МАКСИМУМ,
    0.4 → 1.462/156, 0.5 → 1.352/150, 0.6 → 1.296/148 — ДЕФОЛТ, 0.8 → 1.290/147, 1.0 → 1.293/146,
    1.5 → 1.310/146; 2.0 и 2.5 дают одинаковые 1.239/145 — цель выше 2.0 ATR не достигается ни
    разу). В `_comment` прямо записать подпись капкана: с 1.0 и выше пул сделок стоит (140 → 139), а
    PF растёт в полтора раза; **просадка при этом ПАДАЕТ** (1.5 → 27 614 ₽ против 42 902 у дефолта),
    поэтому гейт B капкан не поймает и режет его только гейт A с потолком 1.0 (выживаемость 44.7%).
  - `exit` и `trail`: ряд `RSIUpper` (35 → 1.109/212, 40 → 1.088/196, 45 → 1.093/183, 50 → 1.168/176,
    55 → 1.178/170, 60 → 1.175/163, 65 → 1.314/158 — максимум, 70 → 1.296/148 — ДЕФОЛТ,
    75 → 1.222/143, 80 → 1.252/141, 85 → 1.186/140, 90 → 1.167/138), `UseRSIExit` (0 → 1.106/137,
    1 → 1.296/148) и `TrailDailyATR` (0.3 → 1.262/165 при DD **24 034 ₽**, 0.5 → 1.314/152 при
    32 154, 0.7 → 1.297/148 при 40 270, 1.0 и 1.5 → 1.296/148 — побайтово равны отсутствию трейла) с
    пометкой, что трейл 0.3–0.5 — единственная ось, снижающая просадку почти вдвое без потери PF.

- [ ] **Step 4: Прогнать тесты.**

Run: `go test ./internal/service/backtest/ -run 'TestSPBE|TestRSIPullback'`
Expected: PASS

- [ ] **Step 5: Коммит.**

```bash
git add data/params/rsi_pullback/spbe internal/service/backtest/rsi_pullback_spbe_grid_test.go
git commit -m "feat(rsi_pullback): каталог сеток SPBE"
```

---

### Task 2: Пакет `strategy/spbe` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/spbe/spbe.go`, `spbe_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params` и `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`; хелпер
  `rsiPullbackBindingFor(ticker string, params func() core.Params)` из
  `internal/service/backtest/rsi_pullback_registry.go`.
- Produces: `spbe.Ticker` (константа `"SPBE"`) и `spbe.DefaultParams() core.Params` — их используют
  Tasks 12, 13 и реестр бэктеста.

- [ ] **Step 1: Написать падающие тесты** в `spbe_test.go`:

```go
package spbe

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ПРОМЕЖУТОЧНОЕ состояние: пакет заведён под
// калибровку 2026-09-04, литерала ещё нет, и до вердикта он обязан отдавать ровно дефолты ядра.
// Тест снимается вместе с постановкой литерала (Task 12).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки SPBE обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSPBE(t *testing.T) {
	if Ticker != "SPBE" {
		t.Fatalf("Ticker = %q, want SPBE", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/spbe/`
Expected: FAIL — пакета ещё нет.

- [ ] **Step 3: Создать пакет** `spbe.go` с честной шапкой в стиле соседних пакетов:

```go
// Package spbe supplies the ticker and rsi_pullback Params for SPBE (ПАО «СПБ Биржа»,
// обыкновенные акции, лот 1).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. DefaultParams() отдаёт baseline ядра до вердикта.
// ...
package spbe

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "SPBE"

// DefaultParams returns the rsi_pullback parameters for SPBE.
func DefaultParams() core.Params { return core.DefaultParams() }
```

  В шапке записать: окно 2023-09-04 … 2026-09-04 и схему 36/12/6; baseline (148 сделок, PF 1.296,
  max DD 24.83%, expectancy 0.318% капитала) и walk-forward дефолтов (1.176/87 и 1.156); оба
  риск-гейта с числами (гейт A — потолок 1.0 при выживаемости 44.7%; гейт B — 24.83% **без**
  множителя 1.3, плюс правило разрешения ничьих в пользу меньшей просадки); четыре пункта
  стоп-условия с указанием, что дефолты ядра проходят все четыре ещё до калибровки; априор скринера
  (26-е место, PFmed 1.37, holdout 0.90/6) и его расхождение с прямым замером; особенности бумаги —
  шаг цены 0.1 ₽ (круг 0.088–0.140% против моделируемых 0.1%), ликвидность 164.9 млн ₽ по медиане
  12 месяцев, дневной ATR 4.78%, режим −24.0% за окно с полугодиями −52.2 / +18.7 / +181.5 / −11.0 /
  −7.1 / −42.4 и просадкой инструмента 70.4%, сессия расширилась 28 → 34 бара.

- [ ] **Step 4: Завести тикер в реестр бэктеста.** В `rsi_pullback_registry.go` добавить импорт
  `rsipullbackspbe "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/spbe"` и строку
  `rsipullbackspbe.Ticker: rsiPullbackBindingFor(rsipullbackspbe.Ticker, rsipullbackspbe.DefaultParams),`
  — **обе в алфавитном порядке среди соседей** (`spbe` идёт после `sofl`/`sngsp` по существующему
  порядку файла: сверить фактический порядок и вписать по нему). В
  `rsi_pullback_registry_test.go` добавить `TestRSIPullbackSPBETracksBaseline` по образцу соседнего
  теста UWGN: достать биндинг из `rsiPullbackRegistry[rsipullbackspbe.Ticker]`, проверить `ok`,
  проверить, что параметры равны `rsipullbackspbe.DefaultParams()`, и что алиас `rsipullbackspbe`
  резолвится.

- [ ] **Step 5: Прогнать тесты.**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 6: Коммит.**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/spbe internal/service/backtest
git commit -m "feat(rsi_pullback): пакет SPBE до калибровки"
```

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_screen.json -out ./reports/SPBE_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** проверить «Фолдов: 4» в шапке `*_walkforward.md` — иначе остановиться и доложить
  владельцу, остальные темы не запускать.
- [ ] **Step 3:** выписать pooled OOS PF, сделки пула, пофолдовые `UseDayATRGate` / `UseVolume`,
  пофолдовые OOS PF / сделки / MaxDD%. Ожидание из точечного замера: `UseDayATRGate=1` почти
  наверняка (без гейта 0.889/437 — дефолты убыточны); по `UseVolume` **ожидается свободный выбор в
  пользу единицы** (точечно 1.382/120 против 1.296/148), и это ключевой замер темы: гейт может
  войти в точку.
- [ ] **Step 4:** дописать строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-04: …` в `_comment` файла сетки.
- [ ] **Step 5: Коммит** `feat(rsi_pullback): SPBE, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_entry.json -out ./reports/SPBE_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему (504 прогона × 4 фолда).
- [ ] **Step 2:** проверить планку по критериям A (pooled OOS PF ≥ 1.5 при ≥ 20 сделках) и B
  (`RSILower` одинаков в ≥ 3 фолдах из 4). Записать «взят»/«провален» отдельно по каждому.
- [ ] **Step 3:** проверить края расширенной оси периода. На SPBE ось `RSIPeriod` унимодальна с
  максимумом на дефолте 4, поэтому **победа любого края (2 или 10) означает переоптимизацию
  обучающего окна** — точечно 2 → 0.942/394 (убыточно на полном окне) и 10 → 0.872/49. Победа края
  пишется прямым текстом.
- [ ] **Step 4:** отдельно записать, победил ли нижний край `RSILower` 10: точечно 1.485 на
  39 сделках за три года, то есть ~13 сделок на обучающее окно — победа края означала бы редкий
  вход, и это предупреждение обязано попасть в `_comment`.
- [ ] **Step 5:** дописать результат в `_comment`, **Коммит** `feat(rsi_pullback): SPBE, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая ключевая и нижний угол

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_trend.json -out ./reports/SPBE_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_trend_low.json -out ./reports/SPBE_trend_low \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** планка проверяется **только по канонической** `cal_trend` (критерии A и B).
- [ ] **Step 3:** сравнить `trend_low` с `trend`: поле тренда идёт из `trend_low` **только** при
  большинстве ≥ 3/4 **и** превосходстве pooled OOS над канонической темой. Иначе — из `trend` (или
  дефолт ядра, если большинства нет). Решение пишется прямым текстом. Ожидание: точечно нижний угол
  сильнее (30 → 1.496, 20 → 1.461 против 100 → 1.296), то есть `trend_low` — реальный претендент.
- [ ] **Step 4:** проверить нижний край расширенной оси: победа `EMASlow` 15 в большинстве фолдов
  означает, что ось не исчерпана и вопрос края остаётся открытым — записать прямым текстом и снять
  зонд за краем (`EMASlow` 10) одноточечной сеткой на полном окне.
- [ ] **Step 5:** дописать результаты в оба `_comment`, **Коммит**
  `feat(rsi_pullback): SPBE, темы trend и trend_low`.

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_day.json -out ./reports/SPBE_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_day_spent.json -out ./reports/SPBE_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** сверить темы между собой: расхождение пофолдовых `SpentDayATR` — взаимодействие
  веток гейта, пишется прямым текстом.
- [ ] **Step 3:** проверить края: точечно `SpentDayATR` 0.9 → 1.333/118 и 1.25 → 1.326/54 — победа
  1.5 (1.341/**34** за три года) означала бы уход в вырожденную выборку и записывается
  предупреждением. По `FreshDayATR` отличие 0 и 0.1 составляет 0.005 PF — если тема выберет 0.1,
  это плато шириной меньше 0.05 и читается как отсутствие сигнала (§5.3 спеки), поле идёт дефолтом.
- [ ] **Step 4:** дописать результаты, **Коммит** `feat(rsi_pullback): SPBE, темы day и day_spent`.

---

### Task 7: Тема `volume`

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_volume.json -out ./reports/SPBE_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** выписать пофолдовых победителей `VolMult` и `VolBaseDays`; проверить, победил ли
  край `VolBaseDays` 3 или 20 (точечно лучший узел — 10 при `VolMult` 1.0: 1.395/124).
- [ ] **Step 3:** записать место SPBE в каталожной гипотезе «объём говорит на тонких бумагах»:
  оборот **164.9 млн ₽** по медиане 12 месяцев — на порядок выше NKHP (14 млн, единственный
  подтверждённый случай) и UWGN (63 млн, где гейт вредил монотонно). Если гейт выигрывает и здесь,
  гипотеза о ликвидности как причине **опровергнута**, и это кандидат в механический вывод
  `docs/rsi_pullback/strategy.md` §8.1 (Task 15 Step 3).
- [ ] **Step 4:** дописать результат, **Коммит** `feat(rsi_pullback): SPBE, тема volume`.

---

### Task 8: Тема `vol_window`

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_vol_window.json -out ./reports/SPBE_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** сверить `VolMult` с темой `volume`; расхождение = форма гейта не откалибрована,
  `VolMult` идёт дефолтом ядра.
- [ ] **Step 3:** проверить верхний край `VolLookbackBars` 32 — насыщается ли ось (побайтовое
  равенство с 24 означает насыщение, как было на TGKA; четыре разных значения по фолдам означают
  отсутствие устойчивого оптимума, как было на NKHP).
- [ ] **Step 4:** дописать результат, **Коммит** `feat(rsi_pullback): SPBE, тема vol_window`.

---

### Task 9: Тема `risk` — стоп, цель и обязательная проверка риск-гейта A

**Files:** также `data/params/rsi_pullback/spbe/plateau_stop_10.json`, `plateau_stop_20.json`

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_risk.json -out ./reports/SPBE_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** применить риск-гейт A: если большинство фолдов выбрало `StopDailyATR` > 1.0,
  победитель темы **отвергается гейтом**, в точку идёт 1.0 (44.7% дней), цена решения в PF пишется
  прямым текстом. Ожидание высокое: точечно максимум оси стоит на 1.5 (1.969/140).
- [ ] **Step 3:** снять два зонда капкана — файлы `plateau_stop_10.json` и `plateau_stop_20.json`
  (одноточечные сетки: дефолты ядра плюс поля, принятые темами, и `StopDailyATR` 1.0 и 2.0
  соответственно) на полном окне:

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/plateau_stop_10.json -out ./reports/SPBE_nb_stop_10 -months 36 -min-trades 1
```

  Выписать PF, сделки, **max DD**, выходы RSI/SL/TP с долей SL, удержание (медиана, p90, максимум),
  долю ночёвок, долю выходов в выходную сессию.
- [ ] **Step 4:** записать форму капкана на SPBE прямым текстом: точечно 0.5 → 1.296/148,
  1.0 → 1.292/140, 1.3 → 1.752/140, 1.5 → 1.969/140, 2.0 → 1.930/139 — пул стоит, PF растёт в
  полтора раза, **а просадка при этом падает** (1.5 → 27 614 ₽ против 42 902 у дефолта). Это форма,
  в которой риск-гейт B бессилен: он смотрит на просадку, а она улучшается. Проверить по зондам,
  подтверждается ли это на принятых полях, и записать, что единственным фильтром остаётся гейт A.
- [ ] **Step 5:** дописать результат, **Коммит**
  `feat(rsi_pullback): SPBE, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit` и `trail`

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_exit.json -out ./reports/SPBE_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/cal_trail.json -out ./reports/SPBE_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** проверить, не обходит ли трейл риск-гейт A (урок AFKS): дистанция трейла ниже
  принятого стопа делает её эффективной защитой, и гейт применяется к ней.
- [ ] **Step 3:** применить правило упрощения: дистанция ≥ 1.0 → `UseTrail 0`, `TrailDailyATR 0`
  (на SPBE точечно трейл 1.0 и 1.5 побайтово равны его отсутствию).
- [ ] **Step 4:** если тема выбрала `UseRSIExit 0` — записать отдельной строкой: точечно это
  1.106/137 против 1.296/148, то есть −0.19 PF.
- [ ] **Step 5:** отдельно записать просадку по узлам трейла (0.3 → 24 034 ₽, 0.5 → 32 154,
  0.7 → 40 270 против 42 902 у дефолта) — это материал для правила разрешения ничьих в Task 11
  Step 9.
- [ ] **Step 6:** дописать результаты, **Коммит** `feat(rsi_pullback): SPBE, темы exit и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и четыре пункта стоп-условия

**Files:** `data/params/rsi_pullback/spbe/plateau_point.json`, соседи плато
`plateau_<поле>_<значение>.json`, `docs/superpowers/plans/task-11-report-spbe.md`

- [ ] **Step 1:** собрать точку по правилу большинства (≥ 3 из 4; иначе дефолт ядра). Для каждого
  из восемнадцати полей записать голоса фолдов, решение и случайность совпадения с дефолтом.
- [ ] **Step 2:** применить риск-гейт A к собранной точке (эффективная защита
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, потолок 1.0).
- [ ] **Step 3:** прогнать точку схемой 36/12/6:

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/plateau_point.json -out ./reports/SPBE_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

  Выписать pooled OOS PF, сделки пула, пофолдовые in-sample → OOS PF / сделки / NetPnL% / MaxDD%.
  Baseline на той же схеме — **1.176 на 87 сделках**; точка ниже baseline записывается прямым
  текстом, и тогда в вердикт (Task 15) идёт вопрос о маршруте «дефолты ядра» по прецеденту TGKA.
- [ ] **Step 4:** проверить пункты 1 и 2 стоп-условия (PF ≥ 1.0, ≥ 20 сделок в пуле). Провал →
  **остановиться**, задачи 12–16 не выполнять, числа владельцу.
- [ ] **Step 5:** прогнать контрольную схему 24/12/3 (пункт 3):

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/plateau_point.json -out ./reports/SPBE_point_24 \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

  Baseline на том же куске даёт **1.156** — если точка ниже baseline, это записывается прямым
  текстом.
- [ ] **Step 6:** проверить пункт 4 стоп-условия — тот же прогон схемой 36/12/6 с удвоенным кругом:

```bash
go run ./cmd/backtest -ticker SPBE -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/spbe/plateau_point.json -out ./reports/SPBE_point_cost2 \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.001
```

  pooled OOS PF < 1.0 → **остановиться**. Дополнительно снять строку `-commission 0.0015` как
  справочную (тройной круг, не пункт стоп-условия); у baseline на полном окне это 1.064.
- [ ] **Step 7:** применить риск-гейт B на полной истории (одиночный прогон точки на 36 месяцах,
  `-out ./reports/SPBE_point_full`), потолок max DD **24.83%** (без множителя 1.3). Нарушение →
  сдвинуть поле стопа или трейла к ближайшему значению оси, возвращающему точку под потолок, и
  записать цену решения в PF.
- [ ] **Step 8:** снять анатомию и полугодия точки; сравнить с baseline по пяти числам третьего
  контура (доля SL 27.0%, удержание 9/22/37, ночёвки 41.2%, выходная сессия 0.7%, DD 24.83%).
  Отдельно проверить подпись капкана: падение доли SL-выходов вместе с ростом удержания и ночёвок →
  точка берёт более узкий стоп.
- [ ] **Step 9:** снять соседей плато по каждому полю, принятому темой большинством (±один узел
  оси) — по одному файлу `plateau_<поле>_<значение>.json` на соседа; совпадение результата
  побайтово = инертная ось, и это пишется прямым текстом. **Применить правило разрешения ничьих:**
  если соседи различаются по pooled OOS PF менее чем на 0.05, в точку идёт вариант с меньшей
  просадкой на полном окне; решение и его цена записываются прямым текстом.
- [ ] **Step 10:** написать рабочую записку `docs/superpowers/plans/task-11-report-spbe.md`: голоса
  по каждому полю, оба walk-forward, четыре пункта стоп-условия с числами, оба риск-гейта, соседи
  плато и применения правила ничьих, полугодия, анатомия против baseline, вердикт по планке,
  сравнение точки с baseline на обеих схемах.
- [ ] **Step 11: Коммит** `feat(rsi_pullback): SPBE, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Задача выполняется только если ни один пункт стоп-условия не сработал.**

**Files:** `internal/service/trading_strategy/rsi_pullback/strategy/spbe/spbe.go`, `spbe_test.go`,
`internal/service/backtest/rsi_pullback_registry_test.go`

- [ ] **Step 1:** заменить `TestParamsTrackTheBaselineUntilCalibrated` на
  `TestParamsAreTheAcceptedPoint` (снимок всех восемнадцати полей принятой точки, по образцу
  `nkhp_test.go`) плюс `TestPointDiffersFromTheCoreBaseline` (сторожит именно те поля, которыми
  точка отличается от дефолтов ядра, с их голосами в доке теста). Значения полей берутся из Task 11
  Step 1 и файла `plateau_point.json`.
- [ ] **Step 2:** убедиться, что тесты падают.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/spbe/`
Expected: FAIL — литерала ещё нет.

- [ ] **Step 3:** поставить литерал принятой точки в `DefaultParams()` (полный `core.Params{...}`,
  все восемнадцать полей явно).
- [ ] **Step 4:** заменить тест реестра `TestRSIPullbackSPBETracksBaseline` на
  `TestRSIPullbackSPBEServesTheCalibratedPoint`.
- [ ] **Step 5:** тесты и линт.

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... && ./bin/golangci-lint run ./internal/...`
Expected: PASS

- [ ] **Step 6: Коммит** `feat(rsi_pullback): SPBE откалиброван — литерал вместо отслеживания baseline`.

---

### Task 13: Реестр живого раннера

**Files:** `internal/service/trading_strategy/rsi_pullback/live/registry.go`, `registry_test.go`

- [ ] **Step 1:** добавить импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/spbe"`
  и строку `spbe.Ticker: spbe.DefaultParams(),` в карту `paramsByTicker` (алфавитный порядок).
- [ ] **Step 2:** дописать абзац комментария карты в том же англоязычном стиле, что у соседей:
  вердикт по планке, оба риск-гейта и их цена, четвёртый пункт стоп-условия по издержкам и принятые
  риски (круг 0.088–0.140% против моделируемых 0.1%, ликвидность 164.9 млн ₽ по медиане 12 месяцев,
  выходная сессия 27.85 млн ₽, режим с полугодиями от −52.2% до +181.5% и просадкой инструмента
  70.4%, санкционный статус эмитента, расширившаяся сессия).
- [ ] **Step 3:** прогнать тесты.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/...`
Expected: PASS

- [ ] **Step 4: Коммит** `feat(rsi_pullback): SPBE в реестре живого раннера`.

---

### Task 14: Боевая вселенная

**Files:** `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`,
`env/prod.env`, `env/prod.env.example`, `env/local.env.example`, `docs/rsi_pullback/live.md`

- [ ] **Step 1:** перечитать числа Task 11 Steps 4–7 — ни один из четырёх пунктов стоп-условия не
  сработал, оба риск-гейта пройдены.
- [ ] **Step 2:** добавить `"SPBE"` в `want` теста конфига двадцать пятым.
- [ ] **Step 3:** убедиться, что тест падает.

Run: `go test ./internal/config/ -run RSIPullback`
Expected: FAIL

- [ ] **Step 4:** дописать `"SPBE"` в `Tickers` + комментарий-абзац в стиле соседних; дописать
  `,SPBE` в три env-файла (`env/prod.env`, `env/prod.env.example`, `env/local.env.example`).
- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  `docs/rsi_pullback/live.md` (только значение — пер-тикерного ничего).
- [ ] **Step 6:** прогнать тесты.

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 7: Коммит** `feat(rsi_pullback): завести SPBE в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:** `internal/service/trading_strategy/rsi_pullback/strategy/spbe/spbe.go`

- [ ] **Step 1:** переписать шапку пакета в итог калибровки: вердикт по планке по каждому критерию
  обеих ключевых тем, точка поле за полем с голосами фолдов, оба walk-forward со сравнением с
  baseline (1.176/87 и 1.156), строки издержек (0.1% / 0.2% / 0.15%), соседи плато с пометкой
  инертных осей и применения правила ничьих, шесть полугодий, анатомия против baseline, результат
  обоих риск-гейтов и их цена. Отдельными абзацами — принятые риски и условия пересмотра:
  1) **режим окна рваный до предела**: −52.2% и +181.5% в двух полугодиях из шести, просадка
     инструмента 70.4%; **условие пересмотра: два подряд полугодия с амплитудой инструмента меньше
     15% — перекалибровать вне планового цикла**;
  2) **просадка baseline худшая в каталоге** (24.83%) и как её закрыл гейт B без множителя 1.3;
  3) **капкан широкого стопа в форме, где просадка не предупреждает** (PF вверх, DD вниз при
     неизменном пуле) — гейт B бессилен, режет только гейт A с потолком 1.0;
  4) **издержки**: круг 0.088–0.140% против моделируемых 0.1% (шаг 0.1 ₽); **условие пересмотра:
     падение цены ниже 100 ₽ поднимает круг выше 0.2% — сверить с пунктом 4 стоп-условия вне цикла**;
  5) **априор скринера ниже замера** (PFmed 1.37, holdout 0.90 на шести сделках) — расхождение как
     на ELFV;
  6) **санкционный и регуляторный риск эмитента**: блокирующие санкции США с ноября 2023 года,
     новостное движение больше дневного ATR, единственная защита в ядре — гейт «потраченного дня»;
     **условие пересмотра: любое изменение санкционного или листингового статуса — перекалибровать
     или вывести из вселенной**;
  7) **выходная сессия** (медиана 27.85 млн ₽, доля выходов точки) и **расширившаяся сессия**
     (28 → 34 бара), откуда контрольный прогон 24/12/3.
- [ ] **Step 2:** прогнать тесты.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS

- [ ] **Step 3:** механический вывод в `docs/rsi_pullback/strategy.md` — **только если** SPBE дал
  вывод о механике. Кандидаты названы заранее: (а) **объёмный гейт на ликвидной бумаге** — если тема
  `volume` выиграла при обороте 165 млн ₽, каталожное объяснение «объём говорит на тонких бумагах»
  (§8.1, NKHP) опровергнуто, и текст §8.1 правится; (б) **капкан широкого стопа в форме, где
  просадка падает вместе с ростом PF** — дополнение к §9 о том, что гейт просадки эту форму не
  ловит. Пер-тикерных чисел, дат и вердиктов в `docs/rsi_pullback/` не писать.
- [ ] **Step 4: Коммит** `docs(rsi_pullback): разбор калибровки SPBE и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** сверка живой сборки с бэктестом. **На 24 месяцах, не на 36** (урок VSMO: при
  `-months 36` инструмент сам предупреждает об ожидаемых расхождениях `Daily*` на ранних барах —
  окно `dailyFetchDays` живого сборщика короче истории эталона).

Run: `go run ./cmd/pullparity -tickers SPBE -months 24`
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

**Покрытие спеки:** инструмент и данные (§1) → Global Constraints; окно и схема (§2) → Global
Constraints и Architecture; baseline с анатомией и полугодиями (§3) → Global Constraints, опорные
числа третьего контура → Task 11 Step 8; замеры осей (§4) → `_comment` файлов в Task 1 Step 3 и
ожидания в Tasks 3–10; одиннадцать тем (§5.1) → Tasks 3–10; ширина сеток, три отступления от канона
и исключённые вырожденные точки (§5.1) → Task 1 со сторожевыми тестами; правило сборки точки (§5.2)
→ Task 11 Step 1; соседи плато (§5.3) → Task 11 Step 9; риск-гейт A (§5.4) → Task 9 Step 2,
Task 10 Step 2, Task 11 Step 2; риск-гейт B и правило разрешения ничьих (§5.5) → Task 11 Steps 7 и 9;
третий контур (§5.6) → Task 11 Step 8; планка (§5.7) → Tasks 4 и 5; четыре пункта стоп-условия (§5.8)
→ Task 11 Steps 4–6; правило прода (§5.9) → Task 14 Step 1; артефакты (§6) → Tasks 1, 2, 11–15;
принятые риски (§7) → Task 15 Step 1; `pullparity` и `mage ci` → Task 16; проверка числа фолдов
(ловушка ASTR) → Task 3 Step 2.

**Плейсхолдеры:** один — литерал в Task 12 (точка известна только после Task 11). Указано, откуда
берётся значение и по какому правилу.

**Согласованность имён:** `spbe.Ticker` и `spbe.DefaultParams()` заводятся в Task 2 и используются
в Tasks 12–14 под теми же именами; тест `TestParamsTrackTheBaselineUntilCalibrated` (Task 2)
заменяется на `TestParamsAreTheAcceptedPoint` в Task 12; `TestRSIPullbackSPBETracksBaseline`
(Task 2) — на `TestRSIPullbackSPBEServesTheCalibratedPoint` (Task 12); файлы сеток из Task 1
используются в Tasks 3–10 под теми же путями `data/params/rsi_pullback/spbe/cal_<тема>.json`.
