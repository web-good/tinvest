# RTKMP под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести RTKMP (ПАО «Ростелеком», привилегированные акции) до вердикта по стратегии
`rsi_pullback`: каталог максимально широких сеток, канонический тематический walk-forward, принятая
точка, прошедшая два риск-гейта и пять пунктов стоп-условия, литерал в пакете и заведение в боевую
вселенную двадцать шестым тикером.

**Architecture:** Процедура **каноническая**: одиннадцать тем поверх дефолтов ядра (baseline
торгует прибыльно — 142 сделки, PF 1.313 — поэтому якорь не нужен), схема **36/12/6** (четыре
фолда, проверено до написания спеки), плюс обязательный контрольный прогон **24/12/3** — сессия
внутри окна расширялась дважды (19 → 29 → 35 получасовых баров в буднем дне). Два риск-гейта: гейт
выживаемости применяется к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, потолок 1.0; гейт
просадки — max DD точки ≤ 1.3 × 11.70% = 15.21%. **Особенность RTKMP:** шаг цены 0.05 ₽ на цене
42.75 ₽ даёт круг **0.234%** против 0.1% в модели, поэтому у стоп-условия пять пунктов, а не
четыре, и запасной маршрут «дефолты ядра» недоступен — дефолты проваливают контрольную схему
24/12/3 (pooled OOS 0.530).

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-05-rtkmp-rsi-pullback-prep-design.md`
**Замеры:** `reports/_analysis/rtkmp_pullback_prep_measurements.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех прогонах; `-interval Minutes30` обязателен в каждой команде
  (дефолт CLI — `Hour1`, забытый флаг даёт чужие числа, а не ошибку).
- **Схема:** `-months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor`;
  у темы `screen` — `-min-trades 1`.
- **Расчётное окно 2023-09-05 … 2026-09-05, 36 месяцев, без обрезки.**
- **Число фолдов сверяется на первой же теме:** «Фолдов: 3» в шапке отчёта темы `screen` —
  остановиться и доложить владельцу (ловушка ASTR).
- **`-refresh` НЕ запускать:** кэш дотянут 2026-09-05 (`RTKMP_Minutes30.json` — 33 797 баров,
  `RTKMP_Day1.json` — 1 134 свечи).
- **Сетки максимально широкие**; обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
  инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower`, ось тренда не порождает пар
  `EMAFast ≥ EMASlow`, `StopDailyATR` нигде не ноль, ось `TPDailyATR` содержит 0.15, ось `EMASlow`
  темы `trend_low` содержит 12.
- **Исключены как вырожденные:** `RSILower` 5 (7 сделок), `RSIPeriod` 12 и 14 (30 и 21),
  `EMASlow` 8 (0.945/317) и 10 (ноль сделок при равных периодах), `TPDailyATR` 0.1 (1.149/182).
- **Правило сборки точки:** ≥ 3 фолда из 4 за значение, иначе дефолт ядра; ничья 2/2 не считается;
  `trend_low` побеждает только при большинстве ≥ 3/4 И превосходстве pooled OOS над `trend`; голоса
  `RSIUpper` из темы `entry` в сборку не идут.
- **РИСК-ГЕЙТ A:** защита достижима ≥ 30% будних дней; применяется к
  `min(StopDailyATR, TrailDailyATR при UseTrail=1)`. Таблица (n = 764): 0.3 — 99.5%, 0.4 — 96.9%,
  0.5 — 90.8%, 0.6 — 83.2%, 0.7 — 73.8%, 0.8 — 63.7%, **1.0 — 47.1%**, 1.3 — 25.5%, 1.5 — 17.7%,
  2.0 — 5.6%. **Потолок — 1.0.**
- **РИСК-ГЕЙТ B:** max DD точки на полном окне ≤ **15.21%** (1.3 × 11.70%).
- **ПРАВИЛО НИЧЬИХ:** при разнице pooled OOS < 0.05 в точку идёт вариант с меньшей просадкой.
- **Третий контур:** опорные числа baseline — доля SL 26.8%, удержание 9 / 19 / 32 бара, ночёвок
  47.2%, выходов в выходную сессию 4.9%, max DD 11.70%.
- **Планка:** `entry` и `trend` (каноническая) обе дают pooled OOS PF ≥ 1.5 при ≥ 20 сделках;
  ведущая ось выбрана одинаково в ≥ 3 фолдах из 4.
- **Стоп-условие из ПЯТИ пунктов:** (1) pooled OOS PF < 1.0 на 36/12/6; (2) < 20 сделок в пуле;
  (3) pooled OOS PF < 1.0 на 24/12/3; (4) то же при `-commission 0.001`; (5) то же при
  **`-commission 0.0012`** (круг 0.24% — настоящие два шага 0.05 ₽ на цене 42.75 ₽). При
  срабатывании — числа владельцу, **задачи 12–16 не выполняются**. Маршрут «дефолты ядра» (TGKA)
  недоступен: дефолты дают 0.530 на 24/12/3 и 1.002 при круге 0.24%.
- **Дефолты ядра:** `RSIPeriod 4`, `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`,
  `DailyATRPeriod 14`, `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`,
  `TPDailyATR 0.6`, `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`,
  `UseRSIExit 1`, `UseTrail 0`, `TrailDailyATR 0`. Baseline на окне: **142 сделки, PF 1.313, net
  +22 085.26 ₽ (+22.09%)**, win rate 66.20%, max DD **16 170.49 ₽ (11.70%)**, exposure 4.34%,
  expectancy **155.53 ₽ = 0.156% капитала**, выходы RSI 86 / SL 38 (26.8%) / TP 18, удержание
  9 / 19 / 32, ночёвок 47.2%, выходов в выходную сессию 4.9%. Walk-forward дефолтов: 36/12/6 →
  **1.354/95** (фолды 1.846/36, 2.793/24, 1.105/15, **0.289/20**); 24/12/3 → **0.530/35**.
- **Полугодия baseline:** 0.850 (24 сделки, −1 744 ₽), 2.407 (23, +9 670), 1.846 (36, +14 849),
  2.789 (24, +12 981), 1.105 (15, +724), **0.289 (20, −14 395)**.
- **Издержки:** шаг 0.05 ₽; круг 0.165% при медиане окна, 0.172% при медиане 12 месяцев,
  **0.234% при текущей цене**. Чувствительность baseline: 0.1% → 1.313, 0.2% → 1.085,
  **0.24% → 1.002**, 0.3% → 0.885, 0.4% → 0.712.
- **Ликвидность:** медиана оборота будних дней 84.87 млн ₽ (36 мес), 70.70 (24), **56.94 (12)**;
  по годам 149.05 → 110.76 → 81.59 → **44.91 (2026)**. Выходная сессия: 254 дня, медиана 3.75 млн.
  Дневной ATR(14) медиана **2.73%**.
- **Априор скринера** (2026-09-05): PFmed 1.35, **holdout PFmed 1.53 на 10 сделках**, Plateau 46%,
  зажатых 0/24, молчащих 0/24, оборот 107 млн ₽, ATR 2.79%, лучшая конфигурация RSI 6/10,
  EMA 20/100, TP 1.5.
- **Режим:** buy&hold −40.7%; полугодия +10.5, −14.7, −6.7, +2.6, −5.7, **−29.7**; просадка
  инструмента **−61.0%**.
- **Каждый `_comment`** обязан содержать: что тема меряет и сколько прогонов; замеры осей с
  предупреждением о крае; полную команду запуска с путём `data/params/rsi_pullback/rtkmp/<файл>`;
  место под строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-05: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md).
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле истории.
- **Ветка:** `feat/rtkmp-pullback-prep` от `main` (`e94c486`); спека закоммичена (`291ac80`).

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:** `data/params/rsi_pullback/rtkmp/cal_*.json` (11 файлов),
`internal/service/backtest/rsi_pullback_rtkmp_grid_test.go`

- [ ] **Step 1:** написать падающий сторожевой тест осей по образцу
  `rsi_pullback_spbe_grid_test.go`: `TestRTKMPGridsStayWide` (наличие обязательных значений по
  таблице §5.1 спеки), `TestRTKMPEntryGridKeepsRSIUpperAboveRSILower`,
  `TestRTKMPTrendGridsKeepFastBelowSlow`. В доке теста — три отступления от канона с их замерами
  (`trend_low` до 12; цель до 0.15; стоп широкий до 2.0, режет гейт A).
- [ ] **Step 2:** убедиться, что тест падает.
  Run: `go test ./internal/service/backtest/ -run TestRTKMP` → FAIL
- [ ] **Step 3:** создать одиннадцать файлов сеток; замеры для `_comment` берутся из §4 спеки.
- [ ] **Step 4:** Run: `go test ./internal/service/backtest/ -run 'TestRTKMP|TestRSIPullback'` → PASS
- [ ] **Step 5:** коммит `feat(rsi_pullback): каталог сеток RTKMP`.

---

### Task 2: Пакет `strategy/rtkmp` в состоянии «калибровка не проводилась»

**Files:** `internal/service/trading_strategy/rsi_pullback/strategy/rtkmp/{rtkmp.go,rtkmp_test.go}`,
`internal/service/backtest/rsi_pullback_registry.go`, `rsi_pullback_registry_test.go`

- [ ] **Step 1:** падающие тесты `TestParamsTrackTheBaselineUntilCalibrated`, `TestTickerIsRTKMP`.
- [ ] **Step 2:** убедиться, что падают.
- [ ] **Step 3:** создать пакет; в шапке — окно и схема, baseline с анатомией, walk-forward
  дефолтов (1.354/95 и **0.530**), оба риск-гейта с числами, пять пунктов стоп-условия с прямой
  записью, что дефолты проваливают пункт 3, априор скринера, шаг цены и круг, ликвидность и её
  падение, режим и просадка инструмента, расширение сессии.
- [ ] **Step 4:** завести тикер в реестр бэктеста (алфавитный порядок) + тест
  `TestRSIPullbackRTKMPTracksBaseline`.
- [ ] **Step 5:** Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 6:** коммит `feat(rsi_pullback): пакет RTKMP до калибровки`.

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_screen.json -out ./reports/RTKMP_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** проверить «Фолдов: 4» — иначе остановиться и доложить владельцу.
- [ ] **Step 3:** выписать pooled OOS PF, сделки пула, пофолдовые `UseDayATRGate` / `UseVolume`,
  пофолдовые OOS PF / сделки / MaxDD%. Ожидание: `UseDayATRGate=1` почти наверняка (без гейта
  0.946/422); по `UseVolume` выбор свободный (лучшая форма даёт +0.10 PF ценой четверти сделок).
- [ ] **Step 4:** дописать `РЕЗУЛЬТАТ ПРОГОНА 2026-09-05: …` в `_comment`.
- [ ] **Step 5:** коммит `feat(rsi_pullback): RTKMP, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_entry.json -out ./reports/RTKMP_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать (504 прогона × 4 фолда).
- [ ] **Step 2:** планка по критериям A (pooled OOS PF ≥ 1.5 при ≥ 20 сделках) и B (`RSILower`
  одинаков в ≥ 3 фолдах). Записать «взят»/«провален» отдельно.
- [ ] **Step 3:** проверить края: победа `RSIPeriod` 2 или 10, победа `RSILower` 10 (1.861 на
  **30 сделках за три года** — десять на обучающее окно) пишется предупреждением прямым текстом.
- [ ] **Step 4:** отдельно записать, ушла ли пара от дефолта (ожидание §4.1: дефолт стоит в
  локальном провале обеих осей).
- [ ] **Step 5:** дописать результат, коммит `feat(rsi_pullback): RTKMP, тема entry`.

---

### Task 5: Темы `trend` и `trend_low`

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_trend.json -out ./reports/RTKMP_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_trend_low.json -out ./reports/RTKMP_trend_low \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** планка — только по канонической `cal_trend`.
- [ ] **Step 3:** сравнить `trend_low` с `trend`; решение прямым текстом.
- [ ] **Step 4:** проверить нижний край: победа `EMASlow` 12 означает неисчерпанную ось — снять
  зонд за краем одноточечной сеткой (узел 10 при `EMAFast` 10 вырожден, зонд ставится при
  `EMAFast` 3).
- [ ] **Step 5:** дописать результаты, коммит `feat(rsi_pullback): RTKMP, темы trend и trend_low`.

---

### Task 6: Темы `day` и `day_spent`

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_day.json -out ./reports/RTKMP_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_day_spent.json -out ./reports/RTKMP_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** сверить темы между собой; расхождение пофолдовых `SpentDayATR` — взаимодействие
  веток гейта, пишется прямым текстом.
- [ ] **Step 3:** проверить края: победа `SpentDayATR` 1.25 (39 сделок) или 1.5 (17) — уход в
  вырожденную выборку, записывается предупреждением. По `FreshDayATR` ось монотонно валится от
  дефолта, победа любого ненулевого узла — сигнал переоптимизации фолда.
- [ ] **Step 4:** дописать результаты, коммит `feat(rsi_pullback): RTKMP, темы day и day_spent`.

---

### Task 7: Тема `volume`

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_volume.json -out ./reports/RTKMP_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** выписать пофолдовых победителей `VolMult` / `VolBaseDays`; проверить победу краёв
  (3 или 20).
- [ ] **Step 3:** записать место RTKMP в каталожной гипотезе «объём говорит на тонких бумагах»:
  оборот 56.94 млн ₽ по медиане 12 месяцев — между NKHP (14 млн, гейт помогал) и SPBE.
- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): RTKMP, тема volume`.

---

### Task 8: Тема `vol_window`

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_vol_window.json -out ./reports/RTKMP_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** сверить `VolMult` с темой `volume`; расхождение = форма гейта не откалибрована,
  `VolMult` идёт дефолтом ядра.
- [ ] **Step 3:** проверить нижний край: точечно максимум оси стоит на `VolLookbackBars` 1 —
  **это край, ниже которого значений не существует**; победа края записывается прямым текстом.
  Верхний край: 12 и 16 совпадают побайтово — насыщение оси.
- [ ] **Step 4:** дописать результат, коммит `feat(rsi_pullback): RTKMP, тема vol_window`.

---

### Task 9: Тема `risk` и обязательная проверка риск-гейта A

**Files:** также `data/params/rsi_pullback/rtkmp/plateau_stop_10.json`, `plateau_stop_20.json`

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_risk.json -out ./reports/RTKMP_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** применить риск-гейт A: победитель `StopDailyATR` > 1.0 отвергается, в точку идёт
  1.0 (47.1% дней), цена решения в PF — прямым текстом. Ожидание высокое: точечно максимум оси
  стоит на 2.0 (1.886/138).
- [ ] **Step 3:** снять два зонда капкана (`plateau_stop_10.json`, `plateau_stop_20.json`) на
  полном окне; выписать PF, сделки, max DD, выходы с долей SL, удержание, ночёвки, выходы в
  выходную сессию.
- [ ] **Step 4:** записать форму капкана прямым текстом: пул стоит с 0.6 (140 → 138), PF растёт в
  полтора раза, **просадка падает** — гейт B бессилен, режет только гейт A.
- [ ] **Step 5:** дописать результат, коммит
  `feat(rsi_pullback): RTKMP, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit` и `trail`

```bash
go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_exit.json -out ./reports/RTKMP_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker RTKMP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkmp/cal_trail.json -out ./reports/RTKMP_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы. `exit` на RTKMP — сильнейшая ось бумаги (размах 0.45 PF).
- [ ] **Step 2:** проверить, не обходит ли трейл риск-гейт A (урок AFKS).
- [ ] **Step 3:** правило упрощения: дистанция ≥ 1.0 → `UseTrail 0`, `TrailDailyATR 0` (точечно
  трейл 1.0 и 1.5 побайтово равны его отсутствию).
- [ ] **Step 4:** если тема выбрала `UseRSIExit 0` — записать отдельной строкой: точечно
  1.018/131 против 1.313/142.
- [ ] **Step 5:** записать просадку по узлам трейла (0.3 → 8 646 ₽, 0.5 → 15 260, 0.7 → 15 899
  против 16 170 у дефолта) — материал для правила ничьих.
- [ ] **Step 6:** дописать результаты, коммит `feat(rsi_pullback): RTKMP, темы exit и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и пять пунктов стоп-условия

**Files:** `data/params/rsi_pullback/rtkmp/plateau_point.json`, соседи плато,
`docs/superpowers/plans/task-11-report-rtkmp.md`

- [ ] **Step 1:** собрать точку правилом большинства (≥ 3 из 4; иначе дефолт ядра); для каждого из
  восемнадцати полей записать голоса фолдов и решение.
- [ ] **Step 2:** применить риск-гейт A (эффективная защита, потолок 1.0).
- [ ] **Step 3:** прогнать точку схемой 36/12/6 (`-out ./reports/RTKMP_point_oos`); выписать pooled
  OOS PF, сделки, пофолдовые in-sample → OOS PF / сделки / NetPnL% / MaxDD%. Baseline — 1.354/95.
- [ ] **Step 4:** проверить пункты 1 и 2 стоп-условия. Провал → **остановиться**.
- [ ] **Step 5:** контрольная схема 24/12/3 (`-months 24 -train-months 12 -test-months 3`,
  `-out ./reports/RTKMP_point_24`) — пункт 3. Baseline на том же куске даёт **0.530**: точка обязана
  быть заметно лучше дефолтов, а не «не хуже».
- [ ] **Step 6:** пункт 4 — тот же прогон 36/12/6 при `-commission 0.001`
  (`-out ./reports/RTKMP_point_cost2`); пункт 5 — при `-commission 0.0012`
  (`-out ./reports/RTKMP_point_cost24`). Дополнительно снять справочную строку `-commission 0.0015`.
- [ ] **Step 7:** риск-гейт B на полной истории (`-out ./reports/RTKMP_point_full`), потолок max DD
  **15.21%**.
- [ ] **Step 8:** анатомия и полугодия точки против пяти чисел третьего контура; проверить подпись
  капкана (падение доли SL при росте удержания и ночёвок → более узкий стоп).
- [ ] **Step 9:** соседи плато по каждому полю, ушедшему от дефолта (±один узел оси), плюс правило
  разрешения ничьих.
- [ ] **Step 10:** написать `docs/superpowers/plans/task-11-report-rtkmp.md`.
- [ ] **Step 11:** коммит `feat(rsi_pullback): RTKMP, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Задача выполняется только если ни один пункт стоп-условия не сработал.**

- [ ] **Step 1:** заменить `TestParamsTrackTheBaselineUntilCalibrated` на
  `TestParamsAreTheAcceptedPoint` (снимок восемнадцати полей) плюс
  `TestPointDiffersFromTheCoreBaseline`.
- [ ] **Step 2:** убедиться, что тесты падают.
- [ ] **Step 3:** поставить литерал принятой точки в `DefaultParams()`.
- [ ] **Step 4:** заменить тест реестра на `TestRSIPullbackRTKMPServesTheCalibratedPoint`.
- [ ] **Step 5:** Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... && ./bin/golangci-lint run ./internal/...` → PASS
- [ ] **Step 6:** коммит `feat(rsi_pullback): RTKMP откалиброван — литерал вместо отслеживания baseline`.

---

### Task 13: Реестр живого раннера

- [ ] **Step 1:** добавить импорт и строку `rtkmp.Ticker: rtkmp.DefaultParams(),` в `paramsByTicker`.
- [ ] **Step 2:** дописать абзац комментария карты в англоязычном стиле соседей: вердикт по планке,
  оба риск-гейта и их цена, пятый пункт стоп-условия по издержкам и принятые риски (круг 0.234% на
  текущей цене против 0.1% в модели, падающая ликвидность 44.91 млн ₽ по медиане 2026 года,
  выходная сессия 3.75 млн ₽, режим −40.7% с худшим последним полугодием, расширившаяся сессия).
- [ ] **Step 3:** Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/...` → PASS
- [ ] **Step 4:** коммит `feat(rsi_pullback): RTKMP в реестре живого раннера`.

---

### Task 14: Боевая вселенная

- [ ] **Step 1:** перечитать числа Task 11 — ни один из пяти пунктов не сработал, оба гейта пройдены.
- [ ] **Step 2:** добавить `"RTKMP"` в `want` теста конфига двадцать шестым.
- [ ] **Step 3:** Run: `go test ./internal/config/ -run RSIPullback` → FAIL
- [ ] **Step 4:** дописать `"RTKMP"` в `Tickers` + комментарий-абзац; дописать `,RTKMP` в
  `env/prod.env`, `env/prod.env.example`, `env/local.env.example`.
- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице `docs/rsi_pullback/live.md`.
- [ ] **Step 6:** Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 7:** коммит `feat(rsi_pullback): завести RTKMP в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

- [ ] **Step 1:** переписать шапку пакета в итог калибровки: вердикт по планке по каждому критерию
  обеих ключевых тем, точка поле за полем с голосами, оба walk-forward со сравнением с baseline
  (1.354/95 и 0.530), строки издержек (0.1% / 0.2% / 0.24% / 0.3%), соседи плато, шесть полугодий,
  анатомия против baseline, результат обоих риск-гейтов и их цена. Отдельными абзацами — принятые
  риски и условия пересмотра из §7 спеки (издержки и пороги цены 60 ₽ / 35 ₽; падение ликвидности
  с порогом 30 млн ₽ за 6 месяцев; худшее последнее полугодие; капкан широкого стопа, который гейт
  B не ловит; расширение сессии; выходная сессия; дивидендные гэпы).
- [ ] **Step 2:** Run: `go test ./internal/service/trading_strategy/rsi_pullback/...` → PASS
- [ ] **Step 3:** механический вывод в `docs/rsi_pullback/strategy.md` — **только если** RTKMP дал
  вывод о механике. Кандидат назван заранее: **шаг цены как доля цены — фильтр не хуже оборота**.
  Пер-тикерных чисел, дат и вердиктов в `docs/rsi_pullback/` не писать.
- [ ] **Step 4:** коммит `docs(rsi_pullback): разбор калибровки RTKMP и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** Run: `go run ./cmd/pullparity -tickers RTKMP -months 24` → ноль расхождений
  (на 24 месяцах, не на 36 — урок VSMO).
- [ ] **Step 2:** Run: `./bin/mage ci` → зелёный
- [ ] **Step 3:** Run: `git status --porcelain | grep -c '^.. reports/'` → `0`
- [ ] **Step 4:** финальный коммит, если остались правки.

---

## Self-Review

**Покрытие спеки:** §1 → Global Constraints; §2 → Global Constraints и Architecture; §3 → Global
Constraints и Task 11 Step 8; §4 → `_comment` в Task 1 Step 3 и ожидания Tasks 3–10; §5.1 →
Tasks 3–10 и Task 1 (сторожевые тесты); §5.2 → Task 11 Step 1; §5.3 → Task 11 Step 9; §5.4 →
Task 9 Step 2, Task 10 Step 2, Task 11 Step 2; §5.5 → Task 11 Steps 7 и 9; §5.6 → Task 11 Step 8;
§5.7 → Tasks 4 и 5; §5.8 (пять пунктов) → Task 11 Steps 4–6; §5.9 → Task 14 Step 1; §6 →
Tasks 1, 2, 11–15; §7 → Task 15 Step 1; `pullparity` и `mage ci` → Task 16; ловушка ASTR →
Task 3 Step 2.

**Плейсхолдеры:** один — литерал в Task 12 (точка известна только после Task 11).

**Согласованность имён:** `rtkmp.Ticker` и `rtkmp.DefaultParams()` заводятся в Task 2 и
используются в Tasks 12–14; `TestParamsTrackTheBaselineUntilCalibrated` (Task 2) заменяется на
`TestParamsAreTheAcceptedPoint` (Task 12); `TestRSIPullbackRTKMPTracksBaseline` (Task 2) — на
`TestRSIPullbackRTKMPServesTheCalibratedPoint` (Task 12); файлы сеток из Task 1 используются в
Tasks 3–10 под теми же путями.
