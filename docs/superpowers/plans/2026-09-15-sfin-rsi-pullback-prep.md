# SFIN под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести SFIN (ПАО «ЭсЭфАй») до вердикта по стратегии `rsi_pullback`: каталог максимально
широких сеток, тематический walk-forward на схеме сборки, принятая точка, прошедшая **четыре**
риск-гейта и **семь** пунктов стоп-условия на **трёх** схемах проверки, при провале — второй круг по
правилам владельца, и либо литерал в пакете с заведением в боевую вселенную тридцать вторым тикером,
либо протокол отказа.

**Architecture:** Процедура каноническая, но у бумаги три свойства, которых нет ни у одного тикера
каталога. Первое: **пять дивидендных отсечек внутри окна, крупнейшая −48.3% за ночь** (2025-12-25,
1828.2 → 945.0 ₽); движок дивиденды не моделирует, а позиция равна всему капиталу — отсюда новый
риск-гейт D, требующий нуля сделок, удержанных через отсечку. Второе: **печать на пустом стакане
живёт в выходной сессии, а не в ночных часах** — выходные дают 29.1% ряда при медиане объёма 140
лотов против 3900 в будни, и все четыре бара класса «−4% и возврат внутри получаса на объёме меньше
тысячи лотов» стоят в субботах; отсюда риск-гейт C переопределён на **входы в выходные**. Третье:
**самый выраженный капкан широкого стопа в каталоге** — от стопа 0.5 к 2.0 PF растёт с 1.116 до
1.919 при падении числа SL-выходов с 41 до одного и изменении пула всего на двенадцать сделок из
141; барьеры против него — потолок гейта A 0.7 и **рублёвая** форма гейта B (процентная форма капкан
пропускает). Главным критерием прода владелец объявил **прибыльность каждого календарного года по
сырым числам**; дефолты ядра его проваливают 2024-м (−1 646 ₽, PF 0.968). Год 2023 из критерия
исключён как непредставительный (девять сделок, окно открывается 15 сентября).

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков), `python3` для
разбора журналов сделок.

**Spec:** `docs/superpowers/specs/2026-09-15-sfin-rsi-pullback-prep-design.md`
**Замеры:** `reports/_analysis/sfin_pullback_prep_measurements.md`

## Global Constraints

- **Ветка:** `feat/sfin-pullback-prep` от `main` (HEAD `16f8168`, вселенная из 31 тикера). Ветка уже
  создана, спека уже закоммичена (`5952262`).
- **Таймфрейм `Minutes30`** во всех прогонах; **`-interval Minutes30` обязателен в каждой команде**
  (дефолт CLI — `Hour1`, забытый флаг даёт чужие числа, а не ошибку).
- **Расчётное окно тем и точки: `-months 36`** (2023-09-15 … 2026-09-15). Контрольная схема 30/12/6
  идёт с `-months 30`, контрольная схема 24/12/3 — с `-months 24`; **больше никакие прогоны окно не
  меняют**.
- **Схема тематических прогонов первого круга:** `-months 36 -train-months 12 -test-months 6
  -min-trades 20 -metric profit_factor`; у темы `screen` — `-min-trades 1`. Все одиннадцать тем идут
  по этой схеме и только по ней: иначе темы несопоставимы между собой.
- **Три схемы проверки ТОЧКИ (не тем):**
  - **36/12/6** (`-months 36 -train-months 12 -test-months 6`) — **4 фолда**, схема сборки и
    вердикта;
  - **30/12/6** (`-months 30 -train-months 12 -test-months 6`) — **3 фолда**, контроль без разгона
    2024H1, пункт 3 стоп-условия; на дефолтах даёт **0.978** на пуле 69 (дефолты этот пункт валят);
  - **24/12/3** (`-months 24 -train-months 12 -test-months 3`) — **4 фолда**, контроль однородной
    части сессии; его фолд 3 вырожден (**OOS PF 158.145 на семи сделках** при DD 0.03%), и **в
    пользу тикера схема не засчитывается**.
- **`-refresh` НЕ запускать:** ряды дотянуты 2026-09-15 (`SFIN_Minutes30.json` — **33 440 баров**,
  2023-09-15 15:30 … 2026-09-15 15:00, единственный разрыв длиннее четырёх дней — новогодняя пауза
  2023-12-29 → 2024-01-03; `SFIN_Day1.json` — 1364 бара, 2021-09-15 … 2026-09-14). Повторный
  `-refresh` перезапишет кэш и сделает числа плана невоспроизводимыми.
- **Получасовой ряд ровно 36 месяцев.** Схему длиннее 36/12/6 (жёсткий контроль 42/12/6, который
  был на SMLT) построить **нельзя** — данных нет. Не пытаться.
- **Число фолдов сверяется на первой же теме:** «Фолдов: 4» в шапке отчёта темы `screen` — иначе
  остановиться и доложить владельцу (ловушка ASTR). Дефолты дают 4 / 3 / 4 фолда на трёх схемах,
  проверено фактическим прогоном.
- **Остановок торгов, редомициляций и смены инструмента внутри окна нет.**
- **ПЯТЬ ДИВИДЕНДНЫХ ОТСЕЧЕК ВНУТРИ ОКНА** (дата первого бара после разрыва, размер разрыва
  открытия к предыдущему закрытию): **2024-06-13 (−8.2%)**, **2024-12-23 (−10.6%)**,
  **2025-06-09 (−5.0%)**, **2025-12-25 (−48.3%, 1828.2 → 945.0 ₽)**, **2026-05-15 (−15.8%)**. Ни
  один разрыв не восстанавливается за десять торговых дней; `cmd/divscreen -probe SFIN` даёт
  yield ttm **180.50%** при payout 452.95%. **Движок дивиденды не моделирует.** Эти пять дат уже
  выписаны в константе `EX_DATES` скрипта `reports/_analysis/sfin_journal.py` — не менять.
- **Сессия НЕ однородна:** медиана баров в буднем дне по полугодиям окна **19 / 19 / 34 / 35 / 35 /
  35 / 35**, расширение в 2024H2. Отдельного «однородного окна» не заводить: однородная часть окна —
  ровно последние 24 месяца, и её несёт контрольная схема 24/12/3.
- **Разгон 2024H1** (buy&hold **+141.6%**, переоценка после IPO Европлана) целиком сидит в train
  первого фолда схемы сборки. Контроль 30/12/6 открывает окно 2024-03-15, на пике: подъёма
  542 → 2074 ₽ в нём нет ни в train, ни в OOS. Контроль 24/12/3 разгона не содержит вовсе.
- **Лот 1**, шаг цены **0.2 ₽**, реальный круг издержек **0.067%** при последней цене 595.8 ₽ —
  модель (`-commission 0.0005`, круг 0.1%) **ПЕССИМИСТИЧНА в 1.5 раза**. Круг по полугодиям окна:
  0.074 → 0.027 → 0.030 → 0.029 → 0.033 → 0.046 → 0.070%. **Пункт стоп-условия про круг 0.3% НЕ
  вводится** — требовать запас в 4.5 реальных круга значило бы гейтить точку чужим числом.
- **Профиль суток окна:** медиана объёма получасового бара по часу метки — 02 → 26 лотов, 03 → 15,
  04 → 17, 05 → 21, 06 → 35, 07 → 930, 08 → 695, 09 → 940, **10 → 7 700**, 11–18 → 2.7–5.0 тысячи,
  19–23 → 1.1–1.6 тысячи; медиана по окну **2 009 лотов**.
- **Профиль недели окна (главное деление бумаги):** будни — 23 709 баров, медиана объёма **3 900**;
  **выходные — 9 735 баров (29.1%), медиана объёма 140** (в 28 раз тоньше); будни в часах метки
  02–06 — 432 бара (1.8%), медиана 45.
- **Четыре бара класса «печать на пустом стакане»**, все в субботы: 2024-09-21 07:00 (242 лота),
  2025-04-26 19:00 (470), 2025-12-13 19:00 (609), 2026-06-06 19:00 (**56**). Каждый уходит вниз на
  4–6% и возвращается внутри того же получаса.
- **Сетки максимально широкие**; обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
  инварианты: `RSILower ≤ 50`; `RSIPeriod ≥ 2`; `StopDailyATR` нигде не ноль; ни `cal_trend.json`,
  ни `cal_trend_hump.json`, ни `cal_trend_volume.json` не порождают пар `EMAFast ≥ EMASlow`; ось
  `RSILower` темы `entry` содержит оба края 5 и 50; ось `EMASlow` темы `trend_hump` содержит 12 и
  45; ось `EMASlow` темы `trend` содержит 250; ось `RSIUpper` темы `exit` содержит 35 и 95; ось
  `StopDailyATR` темы `risk` содержит 0.35, 0.45, 0.55, 0.65 и верхний край 2.0; ось `TPDailyATR`
  темы `risk` содержит верхний край 2.5; ось `FreshDayATR` темы `day` содержит 0.05, 0.15 и оба края
  0 и 0.5; ось `SpentDayATR` темы `day` содержит верхний край 2.0; ось `VolLookbackBars` темы
  `vol_window` содержит верхний край 32; файлов `cal_entry_deep.json` и `cal_day_spent.json` в
  каталоге **нет** (обе темы намеренно не заводятся — оси гладкие).
- **Риск-гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **0.7**. Порог объявлен до
  прогонов и не двигается. Выживаемость: 0.5 → 79.4%, 0.7 → **57.5%**, 1.0 → 33.4%, 1.5 → 14.3%.
- **Риск-гейт B:** max DD точки на полном 36-месячном окне ≤ **22 297.84 ₽ (20.40%)**. Проверяются
  **обе формы**, рублёвая и процентная; пробита любая — гейт не пройден. **На оси стопа формы
  расходятся:** при стопе 2.0 рубли растут до 27 291 (пробой), проценты падают до 13.04% (планка
  цела) — работающий барьер здесь рублёвый.
- **Риск-гейт C (форма SFIN):** **входов в бары выходных дней у точки должно быть НОЛЬ И входов в
  часы метки 02–06 должно быть НОЛЬ** (оба числа дефолтов равны нулю). Гейтятся только эти два
  числа. Выходы в выходные (у дефолтов **8**, PnL **−1 874.83 ₽**, из них **2 SL**) и выходы в
  часы 02–06 (у дефолтов **9**, PnL **+5 550.33 ₽**) записываются всегда, но вердикта не меняют.
- **Риск-гейт D (новый, только для SFIN):** **сделок, удержанных через любую из пяти дивидендных
  отсечек, у точки должно быть НОЛЬ.** У дефолтов и у всех шестнадцати проб протокола замеров это
  число равно нулю. Сделка считается удержанной через отсечку, если `дата входа < дата отсечки ≤
  дата выхода`; это и считает блок «ГЕЙТ D» скрипта `sfin_journal.py`.
- **Семь пунктов стоп-условия первого круга** (§5.12 спеки). Круг останавливается, если точка даёт:
  1. pooled OOS PF < 1.0 на схеме сборки 36/12/6;
  2. меньше 20 сделок в пуле OOS схемы 36/12/6;
  3. pooled OOS PF < 1.0 на контроле без разгона 30/12/6;
  4. pooled OOS PF < 1.0 на схеме 36/12/6 при `-commission 0.001` (круг 0.2%);
  5. хотя бы один из календарных годов **2024, 2025, 2026** убыточен **по сырым числам** (год 2023
     не гейтится — девять сделок у дефолтов, непредставителен);
  6. провал риск-гейта C — хотя бы один вход в выходные или в часы метки 02–06;
  7. провал риск-гейта D — хотя бы одна сделка, удержанная через дивидендную отсечку.
- **Числа дефолтов, против которых меряется всё** (полное 36-месячное окно): сделок **141**,
  PF **1.116**, net **+13 886.71 ₽ (+13.89%)**, max DD **22 297.84 ₽ (20.40%)**, win rate
  **60.99%**, expectancy **+98.49 ₽**, лучшая / худшая сделка **+7 655.40 / −6 842.27 ₽**, exposure
  **4.29%**. Анатомия выходов: RSI 79 (56.0%), **SL 41 (29.1%)**, TP 21 (14.9%). Удержание
  медиана/p90/максимум **8/21/39** баров, ночёвок **69 (48.9%)**, переносов через два и более дня
  **9**, входов в выходные **0**, выходов в выходные **8**. Календарные годы: 2023 — **9 сделок**,
  +6 900.22 ₽, PF 5.013; **2024 — 45 сделок, −1 645.72 ₽, PF 0.968**; 2025 — 61 сделка,
  +2 493.07 ₽, PF 1.058; 2026 — 26 сделок, +6 139.17 ₽, PF 1.261. Walk-forward: **36/12/6 → 1.144
  на пуле 95** (пофолдово 1.912/26, **0.914/24**, **0.671/25**, 1.918/20), 30/12/6 → **0.978** на 69,
  24/12/3 → 1.006 на 45. Под `-commission 0.001`: 36/12/6 → **0.998**.
- **Готовые скрипты протокола замеров переиспользуются, а не пишутся заново.** Они лежат на диске в
  `reports/_analysis/`, но **в репозиторий не попадают** (`reports/` в `.gitignore`) — не удалять и
  не перезаписывать:
  - `sfin_recon.py` — свойства инструмента, ликвидность, волатильность, профиль суток, крупные бары;
  - `sfin_journal.py` — разбор журнала: календарные годы, полугодия, анатомия выходов, удержание,
    ночёвки, входы и выходы в выходные, **блок «ГЕЙТ C» (часы 02–06)** и **блок «ГЕЙТ D» (пять
    дивидендных отсечек)**, пять худших сделок;
  - `sfin_sweep.py` — печать одномерного свипа из отчёта калибровки: `python3
    reports/_analysis/sfin_sweep.py <ось[,ось2]> <отчёт>_calibration.md`.
- **Задачи прода (14–17) не выполняются**, пока вердикт не вынесен по §5.14 спеки.

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/sfin/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_hump.json`, `cal_trend_volume.json`, `cal_day.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_sfin_grid_test.go`

**Interfaces:**
- Produces: одиннадцать путей `data/params/rsi_pullback/sfin/cal_*.json`, которые читают задачи
  3–10; тест `TestSFINGridsStayWide`.

- [ ] **Step 1: Написать падающий сторожевой тест осей.** Образец —
  `internal/service/backtest/rsi_pullback_x5_grid_test.go` (перечисляет файлы **поимённо**, а не
  обходом каталога). Тест `TestSFINGridsStayWide` читает файлы каталога и проверяет **все**
  инварианты из Global Constraints:
  - `RSILower ≤ 50` во всех файлах, и ось `RSILower` в `cal_entry.json` содержит оба края `5` и `50`;
  - `RSIPeriod ≥ 2` везде;
  - `StopDailyATR != 0` везде, где ось присутствует;
  - ни `cal_trend.json`, ни `cal_trend_hump.json`, ни `cal_trend_volume.json` не порождают пар
    `EMAFast ≥ EMASlow` (перебрать декартово произведение осей внутри каждого файла);
  - ось `EMASlow` в `cal_trend_hump.json` содержит `12` и `45`;
  - ось `EMASlow` в `cal_trend.json` содержит `250`;
  - ось `RSIUpper` в `cal_exit.json` содержит `35` и `95`;
  - ось `StopDailyATR` в `cal_risk.json` содержит `0.35, 0.45, 0.55, 0.65` и верхний край `2.0`;
  - ось `TPDailyATR` в `cal_risk.json` содержит верхний край `2.5`;
  - ось `FreshDayATR` в `cal_day.json` содержит `0.05`, `0.15` и оба края `0` и `0.5`;
  - ось `SpentDayATR` в `cal_day.json` содержит верхний край `2.0`;
  - ось `VolLookbackBars` в `cal_vol_window.json` содержит верхний край `32`;
  - файлов `cal_entry_deep.json` и `cal_day_spent.json` в каталоге **нет** (обе темы намеренно не
    заводятся — проверить явно, чтобы исполнитель будущей правки не завёл их молча).

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/backtest/ -run SFINGridsStayWide -v`
  Expected: FAIL — файлов сеток ещё нет.

- [ ] **Step 3: Написать одиннадцать файлов сеток.** Формат — `{"phases": [{"name": "<тема>",
  "grid": {...}, "keepTop": 5}], "_comment": "..."}`; образец —
  `data/params/rsi_pullback/x5/cal_entry.json`. Оси:

| Файл | Тема | Оси | Прогонов |
|---|---|---|---|
| `cal_screen.json` | `screen` | `UseDayATRGate` [0,1] × `UseVolume` [0,1] | 4 |
| `cal_entry.json` | `entry` | `RSIPeriod` [2,3,4,5,6,7,8,10,12] × `RSILower` [5,10,15,20,25,30,35,40,45,50] | 90 |
| `cal_trend.json` | `trend` | `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] | 48 |
| `cal_trend_hump.json` | `trend_hump` | `EMAFast` [3,5,8,10] × `EMASlow` [12,15,18,20,25,30,35,40,45] | 36 |
| `cal_trend_volume.json` | `trend_volume` | `EMAFast` [5,8] × `EMASlow` [12,15,20,25] × `UseVolume` [0,1] | 16 |
| `cal_day.json` | `day` | `FreshDayATR` [0,0.05,0.1,0.15,0.2,0.3,0.4,0.5] × `SpentDayATR` [0.5,0.6,0.7,0.8,0.9,1.0,1.1,1.2,1.35,1.5,2.0] | 88 |
| `cal_volume.json` | `volume` | `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,1.8,2.0,2.2,2.5,3.0] × `VolBaseDays` [3,5,10,14,20,30,40,50] | 64 |
| `cal_vol_window.json` | `vol_window` | `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult` [1.0,2.5] | 18 |
| `cal_risk.json` | `risk` | `StopDailyATR` [0.3,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.8,1.0,1.2,1.5,2.0] × `TPDailyATR` [0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.0,2.5] | 126 |
| `cal_exit.json` | `exit` | `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90,95] | 13 |
| `cal_trail.json` | `trail` | `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.3,0.4,0.5,0.6,0.7,1.0,1.5] | 14 |

  В каждом `_comment` написать: что тема меряет и сколько в ней прогонов; **замер из §6 спеки, из
  которого получена каждая ось**, с предупреждением о вырожденных краях (`RSILower` 5 — три сделки;
  `RSIPeriod` 12 — двадцать сделок; узел `TPDailyATR` 2.5 кандидатом не считается, он введён только
  как контрольная строка асимметрии, которую требует `TestRSIPullbackCalFilesValid`); **полную
  команду запуска** (этого требует `TestRSIPullbackCalFilesValid`); место под результат прогона,
  голоса фолдов и сверку с ожиданием.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ -run 'SFINGridsStayWide|RSIPullbackCalFilesValid' -v`
  Expected: PASS — оба теста, включая репозиторный инвариант валидности файлов сеток.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): каталог сеток SFIN`.

---

### Task 2: Пакет `strategy/sfin` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`,
  `internal/service/backtest/rsi_pullback_registry_test.go`

**Внимание: пакет `internal/service/trading_strategy/reversion/strategy/sfin` уже существует** —
это другая стратегия и другой пакет. Создаётся новый пакет по пути `rsi_pullback/strategy/sfin`;
имена не конфликтуют, потому что импорты в реестрах алиасятся (`rsipullbacksfin`).

**Interfaces:**
- Produces: `sfin.Ticker` (константа `"SFIN"`), `sfin.DefaultParams() core.Params` — их читают
  задачи 14–16.

- [ ] **Step 1: Написать падающий тест пакета.** Образец —
  `internal/service/trading_strategy/rsi_pullback/strategy/x5/x5_test.go`, но в состоянии «до
  калибровки»: тест `TestParamsTrackTheBaselineUntilCalibrated` проверяет, что `DefaultParams()`
  равен `core.DefaultParams()` поле в поле, и тест `TestTickerIsSFIN` проверяет константу.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sfin/ -v`
  Expected: FAIL — пакета нет.

- [ ] **Step 3: Написать пакет.** `Ticker = "SFIN"`, `DefaultParams() core.Params { return
  core.DefaultParams() }`, doc-comment пакета — одна фраза «калибровка не проводилась, параметры
  следуют дефолтам ядра» со ссылкой на спеку.

- [ ] **Step 4: Зарегистрировать пакет в реестре бэктеста.** В
  `internal/service/backtest/rsi_pullback_registry.go` добавить импорт
  `rsipullbacksfin "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sfin"` и строку
  `rsipullbacksfin.Ticker: rsiPullbackBindingFor(rsipullbacksfin.Ticker, rsipullbacksfin.DefaultParams),`
  рядом со строкой X5. В `rsi_pullback_registry_test.go` добавить тест
  `TestRSIPullbackSFINIsRegisteredAndUncalibrated` по образцу соседних тестов
  **неоткалиброванных** тикеров.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'SFIN|RSIPullback' -v`
  Expected: PASS

- [ ] **Step 6: Проверить, что бэктест видит тикер.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -months 36 -metric profit_factor -out ./reports/SFIN_base
python3 reports/_analysis/sfin_journal.py reports/SFIN_base/SFIN_rsi_pullback_Minutes30_*.md
```

  Скрипт обязан выдать ровно числа дефолтов из Global Constraints: сделок **141**, PF **1.116**,
  max DD **22 297.84 (20.40%)**, календарные годы **+6 900.22 / −1 645.72 / +2 493.07 /
  +6 139.17 ₽**, анатомию выходов **RSI 79, SL 41, TP 21**, удержание **8/21/39**, ночёвок
  **69 (48.9%)**, **входов в выходные 0, выходов 8 (−1 874.83 ₽)**, ГЕЙТ C — **входов 02–06: 0,
  выходов: 9, PnL +5 550.33 ₽**, ГЕЙТ D — **ноль сделок через все пять отсечек**. Расхождение
  означает, что кэш свечей перезаписан `-refresh`; в этом случае остановиться и доложить владельцу.

- [ ] **Step 7: Коммит** `feat(rsi_pullback): пакет SFIN до калибровки`.

---

### Task 3: Тема `screen` — цена двух гейтов и сверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/sfin/cal_screen.json` (`_comment`); Create
`docs/superpowers/plans/task-3-report-sfin.md`

**Interfaces:**
- Consumes: `cal_screen.json` из Task 1.
- Produces: голоса фолдов по `UseDayATRGate` и `UseVolume` — их читает задача 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_screen.json -out ./reports/SFIN_screen \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Сверить число фолдов.** В шапке отчёта `*_walkforward.md` должно стоять
  «Фолдов: 4». Если стоит другое — **остановиться и доложить владельцу** (ловушка ASTR): все
  последующие темы станут несопоставимы.

- [ ] **Step 3: Записать pooled OOS PF, размер пула и голоса всех четырёх фолдов** по обеим осям.

- [ ] **Step 4: Сверить с ожиданием, записанным до прогона.** §6.3 спеки: **ожидается победа узла
  1×1**, то есть **включение объёмного гейта** — на полном окне 1×1 даёт 1.300/97 при DD 13 594 ₽
  против baseline 1×0 с 1.116/141 при DD 22 298 ₽. Это редкость каталога (прецеденты NKHP и X5).
  Если ожидание не подтвердилось — записать это прямым текстом, а не подгонять вывод.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-3-report-sfin.md` (образец —
  `docs/superpowers/plans/task-3-report-x5.md`) и перенести результат в `_comment` файла сетки.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): SFIN, тема screen`.

---

### Task 4: Тема `entry` — первая половина планки

**Files:** Modify `data/params/rsi_pullback/sfin/cal_entry.json` (`_comment`); Create
`docs/superpowers/plans/task-4-report-sfin.md`

**Interfaces:**
- Consumes: `cal_entry.json` из Task 1.
- Produces: голоса фолдов по `RSIPeriod` и `RSILower` — их читает задача 11; вердикт по первой
  половине планки — его читает задача 11 Step 14.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_entry.json -out ./reports/SFIN_entry \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать pooled OOS PF, размер пула и голоса всех четырёх фолдов** по обеим осям
  (`RSILower` — ведущая).

- [ ] **Step 3: Проверить первую половину планки** (§5.11 спеки): pooled OOS PF ≥ **1.5** при
  **≥ 20** сделках в пуле **и** `RSILower` одинаков в **≥ 3 фолдах из 4**. Вырожденный фолд в пользу
  тикера не засчитывается.

- [ ] **Step 4: Сверить с ожиданием.** §5.11 спеки: **ожидается, что тема планку ВОЗЬМЁТ**.
  Основание — §6.1: ось `RSILower` даёт 10 → 2.020/29, 15 → 1.868/48, 25 → 1.487/113 при
  дефолтных 30 → 1.116/141, ось `RSIPeriod` даёт 5 → 1.400/100; обе оси монотонны по числу сделок и
  провалов между высшими соседями не имеют.

- [ ] **Step 5: Отметить вырожденные узлы, если фолды их выбрали.** `RSILower` 5 даёт **три сделки**
  за всё окно, `RSIPeriod` 12 — **двадцать**. Выбор такого узла фолдом — подгонка под фолд, а не
  сигнал; записать прямым текстом.

- [ ] **Step 6: Написать отчёт** `docs/superpowers/plans/task-4-report-sfin.md` и перенести
  результат в `_comment` файла сетки.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): SFIN, тема entry`.

---

### Task 5: Темы `trend` и `trend_hump` — вторая половина планки

**Files:** Modify `data/params/rsi_pullback/sfin/cal_trend.json`, `cal_trend_hump.json`
(`_comment`); Create `docs/superpowers/plans/task-5-report-sfin.md`

**Interfaces:**
- Consumes: `cal_trend.json`, `cal_trend_hump.json` из Task 1.
- Produces: голоса фолдов по `EMAFast` и `EMASlow` из обеих тем и их pooled OOS — их читают задачи
  6 и 11.

- [ ] **Step 1: Прогнать каноническую тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_trend.json -out ./reports/SFIN_trend \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Прогнать тему горба.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_trend_hump.json -out ./reports/SFIN_trend_hump \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 3: Записать по каждой теме** pooled OOS PF, размер пула и голоса всех четырёх фолдов
  по обеим осям (`EMASlow` — ведущая).

- [ ] **Step 4: Проверить вторую половину планки** по **канонической** теме `trend`, **не** по
  `trend_hump`: pooled OOS PF ≥ 1.5 при ≥ 20 сделках и `EMASlow` одинаков в ≥ 3 фолдах из 4.

- [ ] **Step 5: Сверить с ожиданием.** §5.11 спеки: **ожидается, что тема `trend` планку НЕ
  возьмёт**. Основание — §6.2: вся каноническая часть оси `EMASlow` (50–250) лежит в полосе
  0.994–1.139 с размахом 0.15 PF, а весь трендовый сигнал бумаги сидит **ниже** её нижнего края
  (вершина на `EMASlow` 12 → 1.461/107 при DD 11 437 ₽).

- [ ] **Step 6: Записать, какая тема даёт больший pooled OOS.** Поле тренда идёт из `trend_hump`
  только при большинстве ≥ 3/4 **и** превосходстве её pooled OOS над канонической `trend` — это
  правило задачи 11.

- [ ] **Step 7: Написать отчёт** `docs/superpowers/plans/task-5-report-sfin.md` и перенести
  результаты в `_comment` обоих файлов сеток.

- [ ] **Step 8: Коммит** `docs(rsi_pullback): SFIN, темы trend и trend_hump`.

---

### Task 6: Тема `day`

**Files:** Modify `data/params/rsi_pullback/sfin/cal_day.json` (`_comment`); Create
`docs/superpowers/plans/task-6-report-sfin.md`

**Interfaces:**
- Consumes: `cal_day.json` из Task 1.
- Produces: голоса фолдов по `FreshDayATR` и `SpentDayATR` — их читает задача 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_day.json -out ./reports/SFIN_day \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать pooled OOS PF, размер пула и голоса всех четырёх фолдов** по обеим осям.

- [ ] **Step 3: Записать предупреждение по оси `FreshDayATR`.** Любое ненулевое значение заводит
  вход на **первый бар дня**, где дневной диапазон ещё нулевой. Если фолды голосуют за ненулевой
  узел — это кандидат на провал риск-гейта C в задаче 11, и там поле будет уведено на дефолт 0.
  Опорный замер §6.4: 0 → 1.116/141 (DD 22 298), 0.05 → 1.237/151 (DD 22 712), 0.1 → 1.214/154,
  0.15 → 1.229/158, 0.2 → 1.107/174, 0.5 → 0.966/323 — горб поднимает PF на 0.12 и просадку не
  улучшает.

- [ ] **Step 4: Записать опорный замер по оси `SpentDayATR`** для сверки: 0.8 → 1.116/141 (дефолт,
  DD 22 298 ₽), 0.9 → 1.166/110, 1.0 → 1.151/93, 1.1 → 1.434/76 (DD 11 150), **1.2 → 1.666/59
  (DD 10 654)**, 1.35 → 1.442/41, 1.5 → 1.425/32, 2.0 → 1.464/**15**. Максимум широкий — соседи 1.1
  и 1.35 дают 1.434 и 1.442; поэтому тема `day_spent` намеренно не заводилась.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-6-report-sfin.md` и перенести
  результат в `_comment` файла сетки.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): SFIN, тема day`.

---

### Task 7: Темы `volume` и `vol_window`

**Files:** Modify `data/params/rsi_pullback/sfin/cal_volume.json`, `cal_vol_window.json`
(`_comment`); Create `docs/superpowers/plans/task-7-report-sfin.md`

**Interfaces:**
- Consumes: `cal_volume.json`, `cal_vol_window.json` из Task 1.
- Produces: голоса фолдов по `UseVolume`, `VolMult`, `VolBaseDays`, `VolLookbackBars` и pooled OOS
  обеих тем — их читают задачи 8 и 11.

- [ ] **Step 1: Прогнать первичную тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_volume.json -out ./reports/SFIN_volume \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Прогнать вторичную тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_vol_window.json -out ./reports/SFIN_vol_window \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 3: Записать по каждой теме** pooled OOS PF, размер пула и голоса всех четырёх фолдов
  по всем осям. Поля объёмного гейта идут из вторичной `vol_window` только при большинстве ≥ 3/4
  **и** превосходстве её pooled OOS над первичной `volume`, которая структурно владеет формой гейта.

- [ ] **Step 4: Сверить с опорными замерами §6.3.** `VolMult`: 1.0 → 1.197/107,
  **1.2 → 1.300/97 (дефолт)**, 1.5 → 1.133/85, 2.5 → 0.936/66, 3.0 → 1.058/56. `VolBaseDays`:
  3 → 1.189/106, 5 → 1.280/107, **14 → 1.300/97 (дефолт)**, 40 → 1.303/77, 50 → 1.272/77.
  `VolLookbackBars`: 1 → 1.236/85, 2 → 1.266/92, **3 → 1.300/97 (дефолт)**, 8 → 1.143/119,
  16 → 1.036/132, 32 → 1.072/135. **Дефолтная форма гейта (1.2 / 14 / 3) — лучшая или на равных
  лучшей по всем трём осям**; уход от неё требует явного большинства фолдов, а не малого перевеса.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-7-report-sfin.md` и перенести
  результаты в `_comment` обоих файлов сеток.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): SFIN, темы volume и vol_window`.

---

### Task 8: Тема `trend_volume` — арбитр двух рычагов

**Files:** Modify `data/params/rsi_pullback/sfin/cal_trend_volume.json` (`_comment`); Create
`docs/superpowers/plans/task-8-report-sfin.md`

**Interfaces:**
- Consumes: `cal_trend_volume.json` из Task 1; pooled OOS тем `trend_hump` (Task 5) и `volume`
  (Task 7) для сравнения.
- Produces: вердикт «рычаги складываются / конкурируют» — его читает задача 11 Step 2.

**Эта тема — арбитр, а не источник полей** (§5.2 спеки). Она заведена потому, что на SFIN **оба**
рычага режут просадку впервые в каталоге: тренд с 22 298 до 9–12 тыс ₽, объёмный гейт с 22 298 до
13 594 ₽ при росте PF на 0.18.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_trend_volume.json -out ./reports/SFIN_trend_volume \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать pooled OOS PF, размер пула и голоса всех четырёх фолдов** по всем трём
  осям, в том числе по `UseVolume`.

- [ ] **Step 3: Сравнить pooled OOS темы с pooled OOS `trend_hump` (Task 5) и `volume` (Task 7).**
  Если тема выше обеих — рычаги **складываются**, и оба поля идут в точку по своим темам. Если ниже
  любой из них — рычаги **конкурируют**, и по правилу §5.2 в точку идёт **один** рычаг: тот, чья
  тема дала больший pooled OOS, а второе поле остаётся на дефолте ядра; цена решения пишется прямым
  текстом.

- [ ] **Step 4: Записать двумерный опорный замер из §6.2 спеки** для сверки: 5×12 → 1.574/57
  (DD 8 970 ₽), 5×20 → 1.415/102, 8×15 → 1.453/106 (DD 11 388 ₽), 8×12 → 1.320/95 — вся область
  `EMAFast` 5–8 × `EMASlow` 12–25 держит PF 1.27–1.57 при просадке 9–12 тыс ₽.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-8-report-sfin.md` и перенести
  результат в `_comment` файла сетки.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): SFIN, тема trend_volume`.

---

### Task 9: Тема `risk`, риск-гейт A и зонды капкана

**Files:** Modify `data/params/rsi_pullback/sfin/cal_risk.json` (`_comment`); Create
`data/params/rsi_pullback/sfin/probe_stop_10.json`, `probe_stop_15.json`, `probe_stop_20.json`;
Create `docs/superpowers/plans/task-9-report-sfin.md`

**Interfaces:**
- Consumes: `cal_risk.json` из Task 1.
- Produces: голоса фолдов по `StopDailyATR` и `TPDailyATR`, уже пропущенные через потолок гейта A —
  их читает задача 11.

**Это самая опасная тема плана.** На SFIN капкан широкого стопа выражен сильнее, чем у любого
тикера каталога, и он же закрывает главный критерий владельца — при стопе от 1.0 все четыре
календарных года прибыльны. Ни одно число in-sample не является основанием сдвинуть потолок гейта A.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_risk.json -out ./reports/SFIN_risk \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать pooled OOS PF, размер пула и голоса всех четырёх фолдов** по обеим осям.

- [ ] **Step 3: Применить потолок риск-гейта A.** Если фолды голосуют за `StopDailyATR` **выше
  0.7** — в точку идёт **0.7**, и цена решения в PF записывается прямым текстом. Порог объявлен до
  прогонов (§5.4 спеки) и после них **не двигается**. Обоснование, которое надо повторить в отчёте:
  выживаемость на 0.7 — 57.5% будних дней, на 1.0 — 33.4%; прямой замер срабатываний даёт SL 41 из
  141 на дефолтном 0.5, **11 из 133** на 1.0, **5 из 131** на 1.5, **1 из 129** на 2.0.

- [ ] **Step 4: Снять три зонда капкана на полном окне.** Файлы `probe_stop_10.json`,
  `probe_stop_15.json`, `probe_stop_20.json` — одна комбинация в каждом (`StopDailyATR` 1.0, 1.5,
  2.0 поверх дефолтов ядра), команда:

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/probe_stop_10.json -out ./reports/SFIN_probe_stop_10 \
  -months 36 -min-trades 1 -metric profit_factor
python3 reports/_analysis/sfin_journal.py reports/SFIN_probe_stop_10/*_best.md
```

  Повторить для 1.5 и 2.0. Записать по каждому зонду: PF, число сделок, **число SL-выходов**, обе
  формы max DD и разбивку по календарным годам. Опорные числа (§6.5 спеки): 1.0 → PF 1.553, 11 SL,
  DD 25 836 ₽ (13.67%); 1.5 → 1.736, 5 SL, 26 184 ₽ (13.90%); 2.0 → 1.919, 1 SL, 27 291 ₽ (13.04%).

- [ ] **Step 5: Записать прямым текстом вывод о расхождении форм просадки.** Рублёвая форма растёт
  (22 298 → 27 291), процентная падает (20.40% → 13.04%). **Гейт B проверяет обе формы, и капкан
  ловит рублёвая.** Это урок X5, и в отчёте он должен быть назван.

- [ ] **Step 6: Записать опорный замер по оси `TPDailyATR`** (§6.5): 0.3 → 1.199/153,
  0.4 → 1.086/147, 0.5 → 1.125/144, **0.6 → 1.116/141 (дефолт)**, 1.0 → 1.118/140, 1.5 → 1.055/139,
  2.0 → 1.000/137, 2.5 → 1.000/137. Узлы 2.0 и 2.5 совпадают и по PF, и по числу сделок — цель за
  этими порогами не достигается, кандидатами они не считаются.

- [ ] **Step 7: Написать отчёт** `docs/superpowers/plans/task-9-report-sfin.md` и перенести
  результат в `_comment` файла сетки.

- [ ] **Step 8: Коммит** `docs(rsi_pullback): SFIN, тема risk и зонды капкана`.

---

### Task 10: Темы `exit` и `trail`

**Files:** Modify `data/params/rsi_pullback/sfin/cal_exit.json`, `cal_trail.json` (`_comment`);
Create `docs/superpowers/plans/task-10-report-sfin.md`

**Interfaces:**
- Consumes: `cal_exit.json`, `cal_trail.json` из Task 1.
- Produces: голоса фолдов по `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` — их читает
  задача 11.

- [ ] **Step 1: Прогнать тему выхода.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_exit.json -out ./reports/SFIN_exit \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Прогнать тему трейла.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal_trail.json -out ./reports/SFIN_trail \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 3: Записать по каждой теме** pooled OOS PF, размер пула и голоса всех четырёх фолдов
  по всем осям.

- [ ] **Step 4: Сверить с опорными замерами §6.6.** `RSIUpper` (в скобках max DD, ₽):
  35 → 1.081/193 (18 010), 40 → 1.218/183, 45 → 1.247/170, **50 → 1.326/164 (22 637)**,
  55 → 1.284/160, **70 → 1.116/141 (дефолт, 22 298)**, 80 → 1.007/133 (30 852), 85 → 0.944/131,
  95 → 1.012/129. Нижняя половина оси лучше дефолта по PF; узлы 35–50 дополнительно лучше по
  просадке — это первый кандидат замены, если точка не пройдёт риск-гейт B.

- [ ] **Step 5: Сверить тему трейла с ожиданием.** §6.6: `TrailDailyATR` при `UseTrail` = 1 даёт
  0.3 → 1.223/158 (DD 18 574), 0.4 → 1.139/147, 0.5 → 1.138/144, 0.7 → 1.118/142, и выше 1.0 трейл
  неактивен. **Ожидание: трейл даёт не больше 0.11 PF над дефолтом**, и поле остаётся на дефолте,
  если тема не наберёт 3/4. Если тема голосует за `UseTrail` = 1, **риск-гейт A применяется к
  `min(StopDailyATR, TrailDailyATR)`** — урок AFKS.

- [ ] **Step 6: Написать отчёт** `docs/superpowers/plans/task-10-report-sfin.md` и перенести
  результаты в `_comment` обоих файлов сеток.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): SFIN, темы exit и trail`.

---

### Task 11: Сборка точки, четыре риск-гейта, три walk-forward и семь пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/sfin/plateau_point.json` и файлы соседей плато
(`plateau_<поле>_<значение>.json` по одному на каждого соседа); Create
`docs/superpowers/plans/task-11-report-sfin.md`

**Interfaces:**
- Consumes: результаты задач 3–10 (голоса фолдов по каждой теме); скрипты
  `reports/_analysis/sfin_journal.py` и `sfin_sweep.py`.
- Produces: принятую точку первого круга (восемнадцать полей) и вердикт — их читают задачи 12 и 14.

- [ ] **Step 1: Проверить скрипт разбора журнала на отчёте дефолтов.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -months 36 -metric profit_factor -out ./reports/SFIN_base
python3 reports/_analysis/sfin_journal.py reports/SFIN_base/SFIN_rsi_pullback_Minutes30_*.md
```

  Скрипт обязан выдать на дефолтах: сделок **141**, PF **1.116**, календарные годы
  **+6 900.22 / −1 645.72 / +2 493.07 / +6 139.17 ₽**, анатомию выходов **RSI 79, SL 41, TP 21**,
  удержание **8/21/39**, ночёвок **69 (48.9%)**, входов в выходные **0**, выходов в выходные **8
  (−1 874.83 ₽)**, ГЕЙТ C — **входов 02–06: 0, выходов: 9, PnL +5 550.33 ₽**, ГЕЙТ D — **ноль
  сделок через каждую из пяти отсечек**. Если выдал другое — неверен скрипт или перезаписан кэш;
  чинить это, а не числа.

- [ ] **Step 2: Собрать точку правилом большинства.** Поле берётся из темы, которая его меряет, и
  принимается только при **≥ 3 голосах из 4** фолдов схемы 36/12/6; иначе — дефолт ядра. Ничья 2/2
  большинством не считается. Приоритеты тем: `trend_hump` над `trend`, `vol_window` над `volume` —
  каждая вторичная тема берёт поле только при большинстве **и** превосходстве pooled OOS над своей
  первичной. **Тема `trend_volume` — арбитр:** при её вердикте «рычаги конкурируют» (Task 8 Step 3)
  в точку идёт только один из двух рычагов. Записать таблицу «поле → тема-источник → голоса →
  принятое значение → дефолт ядра» для всех восемнадцати полей.

- [ ] **Step 3: Применить риск-гейт A.** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤
  **0.7**. Если точка вышла выше — заменить на 0.7 и записать цену решения в PF.

- [ ] **Step 4: Снять одиночный прогон точки на полном окне.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/plateau_point.json -out ./reports/SFIN_point \
  -months 36 -min-trades 1 -metric profit_factor
python3 reports/_analysis/sfin_journal.py reports/SFIN_point/*_best.md
```

- [ ] **Step 5: Применить риск-гейт B.** Max DD точки на полном 36-месячном окне ≤ **22 297.84 ₽
  (20.40%)**, **обе формы** — и рублёвая, и процентная; пробита любая — гейт не пройден. Если
  пробита, точка не принимается, и в неё идёт ближайший вариант плато с меньшей просадкой (первый
  кандидат замены — узел `RSIUpper` из нижней половины оси, см. Task 10 Step 4); замена
  записывается прямым текстом с ценой решения в PF.

- [ ] **Step 6: Применить риск-гейт C.** Из вывода `sfin_journal.py` на отчёте из Step 4 взять
  строку про входы и выходы в выходные и строку «ГЕЙТ C». **Точка проходит, если входов в выходные
  ровно НОЛЬ И входов в часы метки 02–06 ровно НОЛЬ.** Выходы обоих срезов и их PnL записать, но
  вердикта по ним не выносить. **При провале поле `FreshDayATR` уводится на дефолт 0** — это
  единственная ось, которая механизмом управляет (`RSIUpper` на входы не влияет, проверено на X5);
  точка пересчитывается с шага 4, и цена решения в pooled OOS и в полнооконном PF пишется прямым
  текстом. Если провал сохранился и на дефолтном `FreshDayATR` — сработал пункт 6 стоп-условия.

- [ ] **Step 7: Применить риск-гейт D.** Из того же вывода взять блок «ГЕЙТ D». **Точка проходит,
  если через все пять отсечек удержано ровно НОЛЬ сделок.** Если удержана хотя бы одна — записать
  её номер, даты входа и выхода, размер PnL, и считать сработавшим пункт 7 стоп-условия.
  **Прохождение гейта D не закрывает риск прода** (§8 №1 спеки): ноль — свойство выборки, ни одно
  правило ядра про отсечки не знает.

- [ ] **Step 8: Снять соседей плато.** Для каждого поля, ушедшего от дефолта, — одиночный прогон
  самого поля и двух его соседей по оси на полном 36-месячном окне (файлы
  `plateau_<поле>_<значение>.json`, команда как в Step 4). Значение на краю сетки проверяется
  **зондом за краем**; соседа по оси стопа при включённом трейле проверяют **в сторону уменьшения**
  (§8 доки стратегии). Плато шириной меньше 0.05 PF записывается как отсутствие сигнала. При
  разнице pooled OOS < 0.05 в точку идёт вариант с меньшей просадкой. **Для каждого соседа снять
  разбивку по календарным годам и числа гейтов C и D** скриптом `sfin_journal.py` — на SFIN разница
  между хорошей и плохой точкой проходит по календарным годам, а не по полнооконному PF, и 2026 год
  ломается у четырёх проб из шестнадцати (§6.7 спеки).

- [ ] **Step 9: Снять третий контур.** Из вывода `sfin_journal.py` на отчёте точки взять: долю
  SL-выходов, удержание (медиана / p90 / максимум баров), долю ночёвок, число переносов через два и
  более дня, входы и выходы в выходные, max DD, expectancy. Сравнить с числами дефолтов из Global
  Constraints (SL **29.1%**, удержание **8/21/39**, ночёвок **48.9%**, переносов **9**, входов в
  выходные **0**, выходов **8**, DD **20.40%**, expectancy **+98.49 ₽**). **Падение доли SL-выходов
  вместе с ростом удержания и ночёвок — подпись капкана**; в этом случае точка берёт более узкий
  стоп.

- [ ] **Step 10: Прогнать ТРИ walk-forward точки.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/plateau_point.json -out ./reports/SFIN_point_wf36 \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/plateau_point.json -out ./reports/SFIN_point_wf30 \
  -months 30 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/plateau_point.json -out ./reports/SFIN_point_wf24 \
  -months 24 -min-trades 1 -train-months 12 -test-months 3 -metric profit_factor
```

  Ожидаемое число фолдов: **4, 3, 4**. Записать pooled OOS PF, пул и пофолдовые числа каждой схемы.
  **Схема 24/12/3 записывается вместе с напоминанием, что её фолд 3 вырожден (158.145 на семи
  сделках) и в пользу тикера она не засчитывается.** Опорные числа дефолтов: 1.144 / 0.978 / 1.006.

- [ ] **Step 11: Прогнать точку под утяжелёнными издержками.**

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/plateau_point.json -out ./reports/SFIN_point_c001 \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor -commission 0.001
```

  Это пункт 4 стоп-условия (порог 1.0, опорное число дефолтов **0.998** — дефолты этот пункт
  валят). **Прогон под `-commission 0.0015` НЕ делается:** реальный круг бумаги 0.067% против
  модельных 0.1%, модель пессимистична в 1.5 раза, и пункт про круг 0.3% спекой намеренно не введён.

- [ ] **Step 12: Посчитать календарные годы точки.** Из вывода `sfin_journal.py` на отчёте из
  Step 4 взять блок «календарные годы». Критерий владельца (§5.9 спеки): **годы 2024, 2025 и 2026
  прибыльны по сырым числам**, без изъятий; **год 2023 записывается, но не гейтится** (девять
  сделок у дефолтов). Опорные числа дефолтов: 2023 — 9 сделок, +6 900.22 ₽, PF 5.013; **2024 —
  45 сделок, −1 645.72 ₽, PF 0.968**; 2025 — 61 сделка, +2 493.07 ₽, PF 1.058; 2026 — 26 сделок,
  +6 139.17 ₽, PF 1.261.

- [ ] **Step 13: Применить семь пунктов стоп-условия** (Global Constraints). Записать по каждому
  пункту фактическое число и вердикт «взят / не взят». **Пункты 3 (контроль без разгона 30/12/6),
  5 (календарные годы), 6 (гейт C) и 7 (гейт D) — главные и нестандартные.**

- [ ] **Step 14: Проверить планку** — обе канонические темы (`entry` из Task 4, `trend` из Task 5)
  дают pooled OOS ≥ 1.5 при ≥ 20 сделках и одинаковую ведущую ось в ≥ 3 фолдах из 4. Сверить с
  ожиданием «`entry` возьмёт, `trend` не возьмёт».

- [ ] **Step 15: Написать отчёт** `docs/superpowers/plans/task-11-report-sfin.md`: таблица сборки
  точки по восемнадцати полям, четыре риск-гейта с ценой решений, три walk-forward, два уровня
  издержек, четыре календарных года (три гейтятся, 2023 — справочно), соседи плато с их годовой
  разбивкой и числами гейтов C и D, третий контур, вердикт по каждому из семи пунктов, вердикт по
  планке.

- [ ] **Step 16: Вынести вердикт.** Если **ни один** пункт стоп-условия не сработал и все четыре
  риск-гейта пройдены — переходить к задаче 14 (задачи 12 и 13 пропускаются). Иначе — переходить к
  задаче 12. **Ожидание, записанное до прогонов: второй круг НЕ понадобится** (§5.13 спеки);
  главный риск первого круга — расхождение голосов фолдов, а не отсутствие рабочей точки.

- [ ] **Step 17: Коммит** `feat(rsi_pullback): SFIN, точка первого круга и вердикт`.

---

### Task 12: Второй круг — узкие сетки по правилам владельца (только при провале первого)

**Files:** Create `data/params/rsi_pullback/sfin/cal2_*.json` (по одной узкой сетке на каждую тему,
чьё поле было принято в точку первого круга или соседствует с лучшей зоной); Create
`data/params/rsi_pullback/sfin/plateau_point2.json`; Create
`docs/superpowers/plans/task-12-report-sfin.md`

**Задача выполняется ТОЛЬКО при срабатывании хотя бы одного пункта стоп-условия в Task 11.**

**Interfaces:**
- Consumes: голоса фолдов первого круга (задачи 3–10) и вердикт Task 11.
- Produces: точку второго круга — её читают задачи 13 и 14.

- [ ] **Step 1: Выбрать зоны узких сеток по фактическим голосам фолдов первого круга**, а не заново
  по in-sample рельефу. Для каждой темы, чьё поле участвовало в точке или было близко к победе,
  построить сетку вокруг зоны голосов с шагом вдвое мельче каталожного. Записать в `_comment`
  каждого файла, из каких голосов какого фолда получена зона.

- [ ] **Step 2: Написать узкие сетки и расширить сторожевой тест.** Файлы
  `data/params/rsi_pullback/sfin/cal2_<тема>.json`. Тест `TestSFINGridsStayWide` расширяется
  проверкой, что **узкие сетки второго круга не нарушают жёстких инвариантов** Global Constraints
  (`RSILower ≤ 50`, `RSIPeriod ≥ 2`, `StopDailyATR != 0`, нет пар `EMAFast ≥ EMASlow`).

- [ ] **Step 3: Прогнать каждую узкую тему** по схеме сборки:

```bash
go run ./cmd/backtest -ticker SFIN -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sfin/cal2_<тема>.json -out ./reports/SFIN_r2_<тема> \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

  **`-min-trades 1`, а не 20:** во втором круге число сделок не является критерием выбора (прямое
  указание владельца).

- [ ] **Step 4: Собрать точку второго круга** тем же правилом большинства ≥ 3/4 (Task 11 Step 2).

- [ ] **Step 5: Применить все четыре риск-гейта без изменений** (Task 11 Steps 3, 5, 6, 7). Гейты
  вторым кругом не смягчаются.

- [ ] **Step 6: Прогнать три walk-forward и утяжелённые издержки** (Task 11 Steps 10, 11).

- [ ] **Step 7: Применить стоп-условие второго круга.** Пункты 1, 3, 4, 6, 7 — в силе полностью.
  **Пункт 5 (годы 2024, 2025, 2026 прибыльны) — главный критерий, не смягчается ничем.** **Пункт 2
  (≥ 20 сделок) снят**, но фактический размер пула OOS записывается обязательно, и **пул меньше 10
  сделок закрывает работу как непредставительный**.

- [ ] **Step 8: Снять соседей плато и третий контур** для точки второго круга (Task 11 Steps 8, 9).

- [ ] **Step 9: Написать отчёт** `docs/superpowers/plans/task-12-report-sfin.md` — той же структуры,
  что отчёт Task 11, плюс таблица «зона узкой сетки → из каких голосов первого круга получена».

- [ ] **Step 10: Вынести вердикт.** Если точка второго круга удовлетворяет условиям Step 7 —
  переходить к задаче 14. Иначе — переходить к задаче 13 (отказ).

- [ ] **Step 11: Коммит** `feat(rsi_pullback): SFIN, второй круг и вердикт`.

---

### Task 13: Протокол отказа (только при провале обоих кругов)

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin.go`
(doc-comment); Create `docs/superpowers/plans/task-13-report-sfin.md`

**Задача выполняется ТОЛЬКО при провале обоих кругов. Задачи 14–17 в этом случае НЕ выполняются.**

- [ ] **Step 1: Написать разбор отказа в doc-comment пакета.** Обязательное содержимое — всё из
  §5.15 спеки плюс: какой пункт стоп-условия сработал в каждом круге, с фактическими числами;
  почему сработал; что именно на бумаге этому причиной. Литерал **не ставится**, пакет остаётся на
  дефолтах ядра, тест `TestParamsTrackTheBaselineUntilCalibrated` сохраняется.

- [ ] **Step 2: Прогнать полный гейт.**
  Run: `./bin/mage ci`
  Expected: PASS

- [ ] **Step 3: Доложить владельцу** числа обоих кругов и причину отказа. Пакет тикера и сетки
  остаются в ветке как протокол отказа (прецеденты HEAD, AFKS, UWGN, RTKMP, TRNFP).

- [ ] **Step 4: Коммит** `docs(rsi_pullback): SFIN, протокол отказа`.

---

### Task 14: Литерал в пакете и снимок

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin.go`,
`internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin_test.go`,
`internal/service/backtest/rsi_pullback_registry_test.go`

**Задача выполняется ТОЛЬКО при вынесенном положительном вердикте (Task 11 Step 16 или Task 12
Step 10).**

**Interfaces:**
- Consumes: принятую точку из Task 11 или Task 12.
- Produces: `sfin.DefaultParams()`, возвращающий откалиброванный литерал, — его читают задачи 15
  и 16.

- [ ] **Step 1: Заменить тест пакета на снимок литерала.** Тест
  `TestParamsTrackTheBaselineUntilCalibrated` удаляется, вместо него пишется
  `TestParamsMatchTheCalibratedSnapshot`, сверяющий `DefaultParams()` с принятой точкой **поле за
  полем** (все восемнадцать полей выписаны литералами, а не вычислены из `core.DefaultParams()`).
  Образец — `internal/service/trading_strategy/rsi_pullback/strategy/x5/x5_test.go`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sfin/ -v`
  Expected: FAIL — литерал ещё не поставлен.

- [ ] **Step 3: Поставить литерал.** `DefaultParams()` возвращает `core.Params` со всеми
  восемнадцатью полями, выписанными **явно**, даже теми, что совпали с дефолтом ядра: пакет обязан
  быть читаемым без обращения к ядру.

- [ ] **Step 4: Обновить тест реестра бэктеста.** `TestRSIPullbackSFINIsRegisteredAndUncalibrated`
  заменяется на `TestRSIPullbackSFINIsRegisteredAndCalibrated` по образцу соседних тестов
  откалиброванных тикеров: реестр отдаёт именно литерал пакета, а не дефолты ядра.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'SFIN|RSIPullback' -v`
  Expected: PASS

- [ ] **Step 6: Коммит** `feat(rsi_pullback): литерал SFIN`.

---

### Task 15: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`,
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `sfin.Ticker`, `sfin.DefaultParams()` из Task 14.
- Produces: `ParamsFor("SFIN")`, отдающий литерал, — его проверяет Task 18 через `cmd/pullparity`.

- [ ] **Step 1: Написать падающий тест** по образцу теста X5 в
  `internal/service/trading_strategy/rsi_pullback/live/registry_test.go`:
  `ParamsFor(sfin.Ticker)` возвращает ровно `sfin.DefaultParams()`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run SFIN -v`
  Expected: FAIL — тикера нет в реестре.

- [ ] **Step 3: Добавить тикер в реестр.** Импорт
  `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sfin"` и строка
  `sfin.Ticker: sfin.DefaultParams(),` в карту реестра
  (`internal/service/trading_strategy/rsi_pullback/live/registry.go:710`, рядом со строкой
  `x5.Ticker: x5.DefaultParams(),`).

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
  Expected: PASS

- [ ] **Step 5: Коммит** `feat(rsi_pullback): SFIN в реестре живого раннера`.

---

### Task 16: Боевая вселенная

**Files:** Modify `internal/config/rsi_pullback.go:605`, `internal/config/rsi_pullback_test.go`,
`env/local.env.example:28`, `env/prod.env.example:30`

- [ ] **Step 1: Написать падающий тест** в `internal/config/rsi_pullback_test.go`, проверяющий, что
  `"SFIN"` присутствует в списке тикеров по умолчанию и что список содержит **32** элемента
  (сейчас 31, последний — `X5`).

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/config/ -run RSIPullback -v`
  Expected: FAIL

- [ ] **Step 3: Добавить `"SFIN"` последним элементом** в `Tickers` в
  `internal/config/rsi_pullback.go:605` и в обе строки `RSI_PULLBACK_TICKERS=` в
  `env/local.env.example:28` и `env/prod.env.example:30` (сейчас оба списка кончаются на `,X5`).
  **Все три списка правятся одновременно** — расхождение между ними молча теряет тикер в проде.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/config/ -v`
  Expected: PASS

- [ ] **Step 5: Коммит** `feat(rsi_pullback): завести SFIN в боевую вселенную`.

---

### Task 17: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin.go`
(doc-comment)

- [ ] **Step 1: Написать разбор в doc-comment пакета.** Обязательное содержимое §5.15 спеки:
  - **пять дивидендных отсечек** с их размерами (2024-06-13 −8.2%, 2024-12-23 −10.6%, 2025-06-09
    −5.0%, **2025-12-25 −48.3%**, 2026-05-15 −15.8%), факт, что **движок дивиденды не моделирует**,
    и условие **ручного контроля владельцем**: не держать позицию через известную дату отсечки.
    Записать прямым текстом, что **этот риск калибровкой не снимается**;
  - **числа риск-гейта D для принятой точки** и напоминание, что ноль у дефолтов — свойство выборки
    (ожидание около двух попаданий при 141 сделке), а не правило ядра;
  - **профиль выходной сессии** (29.1% ряда, медиана объёма **140 лотов против 3900** в будни) и
    четыре бара класса «печать на пустом стакане» — 2024-09-21 (242 лота), 2025-04-26 (470),
    2025-12-13 (609), 2026-06-06 (**56**), все в субботы;
  - **пять чисел риск-гейта C** для принятой точки (входы в выходные, выходы в выходные и их PnL,
    входы 02–06, выходы 02–06 и их PnL);
  - **разбивку принятой точки по всем четырём календарным годам** и фактический размер пула OOS, с
    пометкой, что **2023 год непредставителен** (девять сделок у дефолтов) и из критерия исключён;
  - **долю SL-выходов точки против дефолтных 29.1%** — на бумаге с дневным ATR 3.85% это главный
    признак капкана широкого стопа;
  - **расхождение форм просадки на оси стопа**: рублёвая растёт (22 298 → 27 291 ₽), процентная
    падает (20.40% → 13.04%), и работающий барьер — рублёвый;
  - **разгон 2024H1** (buy&hold +141.6% после IPO Европлана) и то, что контроль без него несёт схема
    30/12/6, а контроль однородности сессии — схема 24/12/3 с оговоркой про её вырожденный фолд
    (158.145 на семи сделках);
  - таблицу сборки точки, оба вердикта (стоп-условие по семи пунктам и планка), цену решений всех
    четырёх риск-гейтов.

  **Правило документации CLAUDE.md: пер-тикерный разбор живёт в doc-comment пакета и в `_comment`
  файлов сеток, а НЕ в `docs/rsi_pullback/`.** Файлы `docs/rsi_pullback/strategy.md`, `live.md`,
  `screener.md` этой задачей не трогаются.

- [ ] **Step 2: Прогнать линтер.**
  Run: `./bin/mage lint`
  Expected: PASS

- [ ] **Step 3: Коммит** `docs(rsi_pullback): разбор калибровки SFIN и принятый риск`.

---

### Task 18: Финальная проверка

**Files:** нет (только прогоны)

- [ ] **Step 1: Прогнать полный гейт.**
  Run: `./bin/mage ci`
  Expected: PASS (lint + `go test -race ./...` + проверка дрейфа моков)

- [ ] **Step 2: Сверить живую сборку с бэктестом.**

```bash
go run ./cmd/pullparity -tickers SFIN -months 36
```

  Флаг называется **`-tickers`** (множественное число, список через запятую) — `cmd/pullparity`
  флага `-ticker` не имеет и молча возьмёт дефолтный список `UGLD,T,GAZP`. Дефолт `-months` — 24,
  поэтому окно задаётся явно.
  Expected: ноль расхождений между параметрами живого раннера и литералом пакета.

- [ ] **Step 3: Сверить состав вселенной.** Убедиться, что `RSI_PULLBACK_TICKERS` в
  `internal/config/rsi_pullback.go` и в обоих `env/*.example` содержит **32** тикера и что SFIN в
  них последний. Три списка сверяются глазами построчно — тест из Task 16 проверяет только
  Go-константу.

- [ ] **Step 4: Доложить владельцу** итоговые числа: точку, все три walk-forward, утяжелённые
  издержки, календарные годы (три гейтятся, 2023 справочно), пять чисел гейта C, числа гейта D,
  вердикт по семи пунктам и по планке, а также **условие ручного контроля по дивидендным
  отсечкам** — главный риск прода, который калибровкой не снимается.

- [ ] **Step 5: Коммит** (если что-то поправлено финальной проверкой) и остановка — мерж ветки
  делает владелец.

---

## Self-Review

**Покрытие спеки задачами:**

| Раздел спеки | Задача |
|---|---|
| §1 инструмент и данные | Global Constraints (окно, лот, шаг цены, круг издержек, сессия) |
| §2 дивидендные отсечки | Global Constraints (пять дат), Task 11 Step 7 (гейт D), Task 17 Step 1 |
| §3 тонкая сессия и выходные | Global Constraints (профиль недели), Task 11 Step 6 (гейт C), Task 17 |
| §4 три схемы окон | Global Constraints, Task 3 Step 2 (сверка фолдов), Task 11 Step 10 |
| §5.0 baseline | Task 2 Step 6, Task 11 Step 1 |
| §5.1 одиннадцать тем | Task 1, задачи 3–10 |
| §5.2 правило сборки | Task 11 Step 2 (включая арбитраж `trend_volume` из Task 8 Step 3) |
| §5.3 соседи плато | Task 11 Step 8 |
| §5.4 риск-гейт A | Task 9 Step 3, Task 11 Step 3 |
| §5.5 риск-гейт B | Task 9 Step 5, Task 11 Step 5 |
| §5.6 риск-гейт C | Task 11 Step 6 |
| §5.7 риск-гейт D | Task 11 Step 7 |
| §5.8 третий контур | Task 11 Step 9 |
| §5.9 календарные годы | Task 11 Step 12 |
| §5.10 схемы проверки точки | Task 11 Step 10 |
| §5.11 планка | Task 4 Step 3, Task 5 Step 4, Task 11 Step 14 |
| §5.12 стоп-условие | Task 11 Step 13 |
| §5.13 второй круг | Task 12 |
| §5.14 правило прода | Task 11 Step 16, задачи 14–16 |
| §5.15 что записывается | Task 17 Step 1, Task 13 Step 1 (при отказе) |
| §6 свойства бумаги | опорные замеры в задачах 3–10, `_comment` файлов сеток |
| §7 артефакты | Task 1 (сетки), Task 2 (пакет, реестр), задачи 3–12 (отчёты) |
| §8 риски | Task 17 Step 1 (все семь записываются в доку пакета) |

**Проверка на плейсхолдеры:** пройдена — каждый шаг содержит либо точную команду, либо точные
числа, против которых сверяется результат.

**Согласованность имён:** `sfin.Ticker` и `sfin.DefaultParams()` определены в Task 2 Step 3 и
используются в задачах 14–16; тест `TestSFINGridsStayWide` создан в Task 1 Step 1 и расширен в
Task 12 Step 2; `TestRSIPullbackSFINIsRegisteredAndUncalibrated` создан в Task 2 Step 4 и заменён в
Task 14 Step 4; скрипты `sfin_journal.py` и `sfin_sweep.py` существуют до начала работы и только
читаются.

**Известные развилки плана:**
- Task 11 Step 16 — точка прошла: задачи 12 и 13 пропускаются.
- Task 11 Step 16 — точка не прошла: задача 12; при её провале задача 13, и задачи 14–18 не
  выполняются.
- Task 8 Step 3 — рычаги конкурируют: в точку идёт один из двух, а не оба.
